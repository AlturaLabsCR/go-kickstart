// Package middleware provides HTTP middleware helpers.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	appauth "app/auth"
	goauth "github.com/tavocg/go-auth"
)

type identityContextKey struct{}

// Identity holds verified claims and their validated account subject.
type Identity struct {
	Claims *appauth.Claims
	Sub    int64
}

func AuthenticateBearer(logger Logger, localize LocalizeFunc, authenticator goauth.Authenticator[*appauth.Claims], next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			logger.Debug("missing bearer token", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path)
			writeJSONError(w, r, localize, http.StatusUnauthorized, "err.missing_bearer")
			return
		}

		claims, err := authenticator.Verify(r.Context(), parts[1])
		if err != nil {
			if errors.Is(err, goauth.ErrInvalidToken) || errors.Is(err, goauth.ErrExpiredToken) {
				logger.Debug("failed to verify bearer token", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path, "error", err)
				writeJSONError(w, r, localize, http.StatusUnauthorized, "err.verify_bearer")
				return
			}

			logger.Error("failed to verify bearer token", "status", http.StatusInternalServerError, "method", r.Method, "path", r.URL.Path, "error", err)
			writeJSONError(w, r, localize, http.StatusInternalServerError, "err.verify_bearer")
			return
		}

		if claims == nil || claims.Subject() == "" {
			logger.Debug("missing authenticated identity", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path)
			writeJSONError(w, r, localize, http.StatusUnauthorized, "err.missing_identity")
			return
		}

		sub, err := strconv.ParseInt(claims.Subject(), 10, 64)
		if err != nil || sub <= 0 {
			logger.Debug("invalid authenticated subject", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path, "sub", claims.Subject())
			writeJSONError(w, r, localize, http.StatusUnauthorized, "err.invalid_subject")
			return
		}

		identity := Identity{Claims: claims, Sub: sub}
		ctx := context.WithValue(r.Context(), identityContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthenticatedIdentity(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

type PermissionChecker interface {
	RoleHasPermission(ctx context.Context, roleKey string, permissionKey string) (bool, error)
}

func RequirePermission(logger Logger, localize LocalizeFunc, permissions PermissionChecker, permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := AuthenticatedIdentity(r.Context())
		if !ok {
			logger.Debug("missing authenticated identity", "status", http.StatusUnauthorized, "method", r.Method, "path", r.URL.Path)
			writeJSONError(w, r, localize, http.StatusUnauthorized, "err.missing_identity")
			return
		}

		sub := identity.Sub

		for _, role := range identity.Claims.Roles {
			role = strings.TrimSpace(role)
			if role == "" {
				continue
			}

			allowed, err := permissions.RoleHasPermission(r.Context(), role, permission)
			if err != nil {
				logger.Error("failed to check role permission", "status", http.StatusInternalServerError, "method", r.Method, "path", r.URL.Path, "sub", sub, "role", role, "permission", permission, "error", err)
				writeJSONError(w, r, localize, http.StatusInternalServerError, "err.check_role_permission")
				return
			}
			if allowed {
				next.ServeHTTP(w, r)
				return
			}
		}

		logger.Debug("permission denied", "status", http.StatusForbidden, "method", r.Method, "path", r.URL.Path, "sub", sub, "permission", permission, "roles", identity.Claims.Roles)
		writeJSONError(w, r, localize, http.StatusForbidden, "err.permission_denied")
	})
}
