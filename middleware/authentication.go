// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	auth "github.com/tavocg/go-auth"
)

type authenticatedIdentityKey[T any] struct{}

func AuthenticateBearer[T auth.Identifier](authenticator auth.Authenticator[T], next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity, err := authenticator.Verify(r.Context(), parts[1])
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), authenticatedIdentityKey[T]{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthenticatedIdentity[T any](ctx context.Context) (identity T, ok bool) {
	identity, ok = ctx.Value(authenticatedIdentityKey[T]{}).(T)
	return identity, ok
}
