package handler

import (
	"log"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	silentDrop bool
}

func (rec *responseRecorder) Flush() {
	if flusher, ok := rec.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rec *responseRecorder) Unwrap() http.ResponseWriter {
	return rec.ResponseWriter
}

func MarkSilentDrop(w http.ResponseWriter) {
	if rec, ok := w.(*responseRecorder); ok {
		rec.silentDrop = true
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.silentDrop {
			return
		}
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
