// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	appauth "github.com/myrepo/myserver/auth"
	goauth "github.com/tavocg/go-auth"
)

type authenticatedClaimsContextKey struct{}

var AuthenticatedClaimsContextKey = authenticatedClaimsContextKey{}

func AuthenticateBearer(authenticator goauth.Authenticator[*appauth.Claims], next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity, err := authenticator.Verify(r.Context(), parts[1])
		if err != nil {
			if errors.Is(err, goauth.ErrInvalidToken) || errors.Is(err, goauth.ErrExpiredToken) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), AuthenticatedClaimsContextKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthenticatedClaims(ctx context.Context) (*appauth.Claims, bool) {
	identity, ok := ctx.Value(AuthenticatedClaimsContextKey).(*appauth.Claims)
	return identity, ok
}
