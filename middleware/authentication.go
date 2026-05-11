// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	auth "github.com/tavocg/go-auth"
)

func AuthenticateBearer[T auth.Identifier](r *http.Request, authenticator auth.Authenticator[T]) (identity T, status int, err error) {
	parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return identity, http.StatusUnauthorized, auth.ErrInvalidToken
	}

	identity, err = authenticator.Verify(r.Context(), parts[1])
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
			return identity, http.StatusUnauthorized, err
		}

		return identity, http.StatusInternalServerError, err
	}

	return identity, http.StatusOK, nil
}
