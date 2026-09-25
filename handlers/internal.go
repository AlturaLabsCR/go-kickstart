package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"app/middleware"
	email "github.com/tavocg/go-email"
)

func decodeJSON(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain a single JSON value")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) writeStatus(w http.ResponseWriter, r *http.Request, status int, key string, msg string, args ...any) {
	logArgs := []any{
		"status", status,
		"method", r.Method,
		"path", r.URL.Path,
	}
	logArgs = append(logArgs, args...)

	if status >= http.StatusInternalServerError {
		h.logger.Error(msg, logArgs...)
	} else if status >= http.StatusBadRequest {
		h.logger.Debug(msg, logArgs...)
	}

	writeJSON(w, status, map[string]string{"error": h.localize(r, key)})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, status int, err error, key string, msg string, args ...any) {
	if err != nil {
		args = append([]any{"error", err}, args...)
	}
	h.writeStatus(w, r, status, key, msg, args...)
}

func (h *Handler) localize(r *http.Request, key string, args ...any) string {
	if h.localizer == nil {
		return key
	}

	L := h.localizer.LocalizerFunc(h.localizer.PickLanguageFromRequest(r))
	return L(key, args...)
}

// normalizeEmail returns normalized email address if it meets strict rules.
// Returns an empty string for invalid input.
func normalizeEmail(original string) string {
	valid, err := email.StrictParser(original)
	if err != nil {
		return ""
	}

	valid.Normalize(email.StripPlusTag())
	return valid.Address()
}

func (h *Handler) requireIdentity(w http.ResponseWriter, r *http.Request) (middleware.Identity, bool) {
	identity, ok := middleware.AuthenticatedIdentity(r.Context())
	if !ok {
		h.writeStatus(
			w,
			r,
			http.StatusInternalServerError,
			"err.missing_account_subject",
			"missing authenticated account subject",
		)
	}
	return identity, ok
}
