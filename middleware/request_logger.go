// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"net/http"
	"time"
)

type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(p)
}

func RequestLogger(logger Logger, pattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(sw, r)

		args := []any{
			"pattern", pattern,
			"status", sw.status,
			"took", time.Since(start),
		}

		if sw.status >= http.StatusInternalServerError {
			logger.Error("hit", args...)
			return
		}

		logger.Debug("hit", args...)
	})
}
