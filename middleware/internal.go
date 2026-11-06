package middleware

import (
	"net/http"

	"app/internal/httpx"
)

type LocalizeFunc func(r *http.Request, key string, args ...any) string

func writeJSONError(w http.ResponseWriter, r *http.Request, localize LocalizeFunc, status int, key string) {
	msg := key
	if localize != nil {
		msg = localize(r, key)
	}

	httpx.WriteJSONError(w, status, msg)
}
