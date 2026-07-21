// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	appauth "app/auth"
	"app/database"
	goauth "github.com/tavocg/go-auth"
)

type authenticatedClaimsContextKey struct{}

var AuthenticatedClaimsContextKey = authenticatedClaimsContextKey{}

func AuthenticateBearer(logger Logger, authenticator goauth.Authenticator[*appauth.Claims], next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			logger.Debug("missing bearer token", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity, err := authenticator.Verify(r.Context(), parts[1])
		if err != nil {
			if errors.Is(err, goauth.ErrInvalidToken) || errors.Is(err, goauth.ErrExpiredToken) {
				logger.Debug("failed to verify bearer token", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path, "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			logger.Error("failed to verify bearer token", "status", http.StatusInternalServerError, "method", r.Method, "path", r.URL.Path, "error", err)
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

func RequirePermission(logger Logger, db database.Database, permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := AuthenticatedClaims(r.Context())
		if !ok || identity == nil || identity.Subject() == "" {
			logger.Debug("missing authenticated identity", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sub, err := strconv.ParseInt(identity.Subject(), 10, 64)
		if err != nil || sub <= 0 {
			logger.Debug("invalid authenticated subject", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path, "sub", identity.Subject())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		for _, role := range identity.Roles {
			role = strings.TrimSpace(role)
			if role == "" {
				continue
			}

			allowed, err := db.Querier().RoleHasPermission(r.Context(), role, permission)
			if err != nil {
				logger.Error("failed to check role permission", "status", http.StatusInternalServerError, "method", r.Method, "path", r.URL.Path, "sub", sub, "role", role, "permission", permission, "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if allowed {
				next.ServeHTTP(w, r)
				return
			}
		}

		logger.Debug("permission denied", "status", http.StatusForbidden, "method", r.Method, "path", r.URL.Path, "sub", sub, "permission", permission, "roles", identity.Roles)
		w.WriteHeader(http.StatusForbidden)
	})
}
