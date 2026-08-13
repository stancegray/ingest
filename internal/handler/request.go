package handler

import (
	"encoding/json"
	"net/http"
	"strings"
)

var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"x-api-key":           {},
	"x-auth-token":        {},
	"proxy-authorization": {},
}

func captureRequestInfo(r *http.Request) json.RawMessage {
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		lower := strings.ToLower(key)
		if _, redact := sensitiveHeaders[lower]; redact {
			headers[lower] = "[REDACTED]"
			continue
		}
		headers[lower] = strings.Join(values, ", ")
	}

	query := make(map[string]string, len(r.URL.Query()))
	for key, values := range r.URL.Query() {
		query[key] = strings.Join(values, ", ")
	}

	info := map[string]any{
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       query,
		"headers":     headers,
		"remote_addr": r.RemoteAddr,
		"host":        r.Host,
		"proto":       r.Proto,
	}

	if ua := r.UserAgent(); ua != "" {
		info["user_agent"] = ua
	}

	raw, err := json.Marshal(info)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func isJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 || !json.Valid(raw) {
		return false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	_, ok := v.(map[string]any)
	return ok
}
