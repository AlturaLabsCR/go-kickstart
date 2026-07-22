package middleware

import (
	"encoding/json"
	"net/http"
)

type LocalizeFunc func(r *http.Request, key string, args ...any) string

func writeJSONError(w http.ResponseWriter, r *http.Request, localize LocalizeFunc, status int, key string) {
	msg := key
	if localize != nil {
		msg = localize(r, key)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
