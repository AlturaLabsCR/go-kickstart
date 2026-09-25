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
	if w.wroteHeader {
		return
	}

	w.ResponseWriter.WriteHeader(status)
	if status < 200 {
		return
	}

	w.status = status
	w.wroteHeader = true
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(p)
}

func RequestLogger(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(sw, r)

		args := []any{
			"pattern", r.Pattern,
			"method", r.Method,
			"path", r.URL.Path,
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
