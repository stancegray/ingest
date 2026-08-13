package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/main/ingest/internal/store"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /v1/ingest", h.ingest)
	mux.HandleFunc("POST /v1/batches", h.createBatch)
	mux.HandleFunc("POST /v1/batches/{id}/close", h.closeBatch)
	mux.HandleFunc("POST /v1/sources", h.createSource)
	mux.HandleFunc("POST /v1/webhooks", h.createWebhook)

	mux.HandleFunc("GET /api/webhooks/{id}/{token}", h.discordGetWebhook)
	mux.HandleFunc("POST /api/webhooks/{id}/{token}", h.discordExecuteWebhook)
	mux.HandleFunc("PATCH /api/webhooks/{id}/{token}/messages/{messageID}", h.discordEditMessage)
	mux.HandleFunc("DELETE /api/webhooks/{id}/{token}/messages/{messageID}", h.discordDeleteMessage)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Health(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	Source     string          `json:"source"`
	EventType  string          `json:"event_type"`
	ExternalID *string         `json:"external_id"`
	Payload    json.RawMessage `json:"payload"`
	Metadata   json.RawMessage `json:"metadata"`
	BatchID    *string         `json:"batch_id"`
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.store.Ingest(r.Context(), store.IngestInput{
		Source:     req.Source,
		EventType:  req.EventType,
		ExternalID: req.ExternalID,
		Payload:    req.Payload,
		Metadata:   req.Metadata,
		BatchID:    req.BatchID,
	})
	if errors.Is(err, store.ErrSourceNotFound) {
		writeError(w, http.StatusNotFound, "source not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

type createBatchRequest struct {
	Source string `json:"source"`
}

func (h *Handler) createBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	batch, err := h.store.CreateBatch(r.Context(), store.CreateBatchInput{Source: req.Source})
	if errors.Is(err, store.ErrSourceNotFound) {
		writeError(w, http.StatusNotFound, "source not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, batch)
}

func (h *Handler) closeBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.CloseBatch(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed", "id": id})
}

type createSourceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) createSource(w http.ResponseWriter, r *http.Request) {
	var req createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.store.CreateSource(r.Context(), req.Name, req.Description); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
