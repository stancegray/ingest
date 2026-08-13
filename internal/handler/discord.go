package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/main/ingest/internal/discord"
	"github.com/main/ingest/internal/store"
)

type createWebhookRequest struct {
	Name   string  `json:"name"`
	Source string  `json:"source"`
	Avatar *string `json:"avatar"`
}

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.store.CreateWebhook(r.Context(), store.CreateWebhookInput{
		Name:   req.Name,
		Source: req.Source,
		Avatar: req.Avatar,
	})
	if errors.Is(err, store.ErrSourceNotFound) {
		writeError(w, http.StatusNotFound, "source not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) discordExecuteWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token := r.PathValue("token")
	wait := r.URL.Query().Get("wait") == "true"

	wh, err := h.store.GetWebhook(r.Context(), id, token)
	if errors.Is(err, store.ErrWebhookNotFound) {
		writeDiscordError(w, http.StatusNotFound, "Unknown Webhook", 10015)
		return
	}
	if err != nil {
		writeDiscordError(w, http.StatusInternalServerError, "Internal server error", 0)
		return
	}

	payload, files, err := parseDiscordExecuteBody(r)
	if err != nil {
		writeDiscordError(w, http.StatusBadRequest, "Invalid Form Body", 50035)
		return
	}

	if !payload.HasDeliverableContent(len(files) > 0) {
		writeDiscordError(w, http.StatusBadRequest, "Cannot send an empty message", 50006)
		return
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil || !isJSONObject(rawPayload) {
		writeDiscordError(w, http.StatusBadRequest, "Invalid Form Body", 50035)
		return
	}

	requestInfo := captureRequestInfo(r)
	if len(files) > 0 {
		var info map[string]any
		_ = json.Unmarshal(requestInfo, &info)
		info["files"] = files
		requestInfo, _ = json.Marshal(info)
	}

	result, err := h.store.IngestDiscord(r.Context(), store.DiscordIngestInput{
		Webhook:     wh,
		Payload:     rawPayload,
		RequestInfo: requestInfo,
	})
	if err != nil {
		writeDiscordError(w, http.StatusInternalServerError, "Internal server error", 0)
		return
	}

	if !wait {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	username := wh.Name
	var avatar *string
	if payload.Username != nil && *payload.Username != "" {
		username = *payload.Username
	}
	if payload.AvatarURL != nil && *payload.AvatarURL != "" {
		avatar = payload.AvatarURL
	} else {
		avatar = wh.Avatar
	}

	msg := discord.BuildMessage(result.MessageID, discord.Webhook{
		ID:        wh.ID,
		Name:      wh.Name,
		Avatar:    wh.Avatar,
		ChannelID: wh.ChannelID,
	}, payload, username, avatar)

	writeJSON(w, http.StatusOK, msg)
}

func parseDiscordExecuteBody(r *http.Request) (discord.ExecutePayload, []map[string]any, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		return parseMultipartExecute(r)
	}

	var payload discord.ExecutePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return discord.ExecutePayload{}, nil, err
	}
	return payload, nil, nil
}

func parseMultipartExecute(r *http.Request) (discord.ExecutePayload, []map[string]any, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return discord.ExecutePayload{}, nil, err
	}

	reader, err := r.MultipartReader()
	if err != nil {
		return discord.ExecutePayload{}, nil, err
	}

	var payload discord.ExecutePayload
	var files []map[string]any

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return discord.ExecutePayload{}, nil, err
		}

		name := part.FormName()
		if name == "payload_json" {
			if err := json.NewDecoder(part).Decode(&payload); err != nil {
				return discord.ExecutePayload{}, nil, err
			}
			continue
		}

		if name == "file" || strings.HasPrefix(name, "files[") {
			data, err := io.ReadAll(part)
			if err != nil {
				return discord.ExecutePayload{}, nil, err
			}
			files = append(files, map[string]any{
				"field":        name,
				"filename":     part.FileName(),
				"content_type": part.Header.Get("Content-Type"),
				"size":         len(data),
			})
		}
	}

	return payload, files, nil
}

func writeDiscordError(w http.ResponseWriter, status int, message string, code int) {
	writeJSON(w, status, discord.APIError{Message: message, Code: code})
}
