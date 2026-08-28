package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/main/ingest/internal/discord"
	"github.com/main/ingest/internal/store"
)

var botUserAgentHints = []string{
	"curl/",
	"wget/",
	"python-",
	"python/",
	"go-http-client",
	"java/",
	"okhttp",
	"postman",
	"insomnia",
	"httpie",
	"scrapy",
	"libwww",
	"axios/",
	"node-fetch",
	"undici",
	"headless",
	"phantomjs",
	"selenium",
	"apache-httpclient",
	"powershell",
}

var browserUserAgentSignals = []string{
	"AppleWebKit/",
	"Chrome/",
	"Safari/",
	"Gecko",
	"Firefox/",
	"Edg/",
	"OPR/",
}

func isPlausibleUserAgent(ua string) bool {
	ua = strings.TrimSpace(ua)
	if len(ua) < 40 {
		return false
	}

	lower := strings.ToLower(ua)
	for _, hint := range botUserAgentHints {
		if strings.Contains(lower, hint) {
			return false
		}
	}

	if !strings.HasPrefix(ua, "Mozilla/5.0") {
		return false
	}

	signals := 0
	for _, signal := range browserUserAgentSignals {
		if strings.Contains(ua, signal) {
			signals++
		}
	}

	return signals >= 2
}

func writeDiscordSilentAccept(w http.ResponseWriter, wait bool, wh store.WebhookRecord, payload discord.ExecutePayload) {
	MarkSilentDrop(w)
	if !wait {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	username := wh.Name
	var avatar *string
	if payload.Username != nil && *payload.Username != "" {
		username = *payload.Username
	} else {
		avatar = wh.Avatar
	}

	fakeID := discord.MessageSnowflake(time.Now().UnixNano())
	msg := discord.BuildMessage(fakeID, discord.Webhook{
		ID:        wh.ID,
		Name:      wh.Name,
		Avatar:    wh.Avatar,
		ChannelID: wh.ChannelID,
	}, payload, username, avatar)

	writeJSON(w, http.StatusOK, msg)
}
