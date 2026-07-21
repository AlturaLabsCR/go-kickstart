package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	appauth "app/auth"
	"app/database"
	"app/perms"
	auth "github.com/tavocg/go-auth"
	secrets "github.com/tavocg/go-secrets"
)

func (h *Handler) registerAuthRoutes() {
	h.Add(http.MethodPost, h.routePath("/auth/login"), h.LoginOrCreateAccount)
	h.Add(http.MethodPost, h.routePath("/auth/verify"), h.VerifyAuthenticationCode)
	h.Add(http.MethodPost, h.routePath("/auth/refresh"), h.RefreshSession)
}

func (h *Handler) LoginOrCreateAccount(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Email string `json:"email"`
	}
	var req loginRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err, "invalid login request body")
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		h.writeStatus(w, r, http.StatusBadRequest, "invalid login email", "email", req.Email)
		return
	}

	otp, err := secrets.RandOTP()
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to generate login otp")
		return
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute).Unix()
	if err := h.db.Querier().UpsertAccountLoginRequest(r.Context(), email, otp, expiresAt); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to upsert login request", "email", email)
		return
	}

	h.logger.Debug("generated login otp", "email", email, "otp", otp, "expires_at", expiresAt)
	w.WriteHeader(http.StatusNoContent)
}

type sessionResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
}

func (h *Handler) VerifyAuthenticationCode(w http.ResponseWriter, r *http.Request) {
	type verifyRequest struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	var req verifyRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err, "invalid verify request body")
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		h.writeStatus(w, r, http.StatusBadRequest, "invalid verify email", "email", req.Email)
		return
	}

	saved, err := h.db.Querier().SelectAccountLoginRequest(r.Context(), email)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(w, r, http.StatusUnauthorized, err, "missing login request", "email", email)
			return
		}

		h.writeError(w, r, http.StatusInternalServerError, err, "failed to select login request", "email", email)
		return
	}

	if saved.Otp != req.OTP || saved.ExpiresAt < time.Now().UTC().Unix() {
		if saved.ExpiresAt < time.Now().UTC().Unix() {
			_ = h.db.Querier().DeleteAccountLoginRequest(r.Context(), email)
		}
		h.writeStatus(w, r, http.StatusUnauthorized, "invalid login verification code", "email", email)
		return
	}

	var subject int64
	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		var err error

		subject, err = q.OncesertAccountByEmail(r.Context(), email, time.Now().UTC().Unix())
		if err != nil {
			return err
		}

		if err := q.AssignRoleToAccount(r.Context(), subject, perms.RoleDefault); err != nil {
			return err
		}

		return q.DeleteAccountLoginRequest(r.Context(), email)
	}); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to finalize login verification", "email", email)
		return
	}

	accessToken, refreshToken, err := h.authenticator.Issue(r.Context(), &appauth.Claims{
		Sub: fmt.Sprintf("%d", subject),
	})
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to issue session tokens", "sub", subject)
		return
	}

	writeJSON(w, http.StatusOK, newSessionResponse(accessToken, refreshToken))
}

func (h *Handler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	type refreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req refreshRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err, "invalid refresh request body")
		return
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		h.writeStatus(w, r, http.StatusBadRequest, "missing refresh token")
		return
	}

	accessToken, refreshToken, err := h.authenticator.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
			h.writeError(w, r, http.StatusUnauthorized, err, "failed to refresh session")
			return
		}

		h.writeError(w, r, http.StatusInternalServerError, err, "failed to refresh session")
		return
	}

	writeJSON(w, http.StatusOK, newSessionResponse(accessToken, refreshToken))
}

func newSessionResponse(accessToken *auth.Token, refreshToken *auth.Token) sessionResponse {
	return sessionResponse{
		AccessToken:           accessToken.Value,
		RefreshToken:          refreshToken.Value,
		ExpiresIn:             expiresIn(accessToken.ExpiresAt),
		RefreshTokenExpiresIn: expiresIn(refreshToken.ExpiresAt),
	}
}

func expiresIn(expiresAt int64) int64 {
	if expiresAt == 0 {
		return 0
	}

	now := time.Now().Unix()
	return expiresAt - now
}
