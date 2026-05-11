package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/myrepo/myserver/database"
)

func (h *Handler) registerAuthRoutes() {
	h.Add(http.MethodPost, h.routePath("/auth/login"), h.LoginOrCreateAccount)
}

func (h *Handler) LoginOrCreateAccount(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Email string `json:"email"`
	}
	var req loginRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	otp, err := randomOTP()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute).Unix()
	if err := h.db.Querier().UpsertAccountLoginRequest(r.Context(), email, otp, expiresAt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type loginResponse struct {
		SentOTP bool `json:"sent_otp"`
	}

	h.logger.Debug("generated login otp", "email", email, "otp", fmt.Sprintf("%06d", otp), "expires_at", expiresAt)
	writeJSON(w, http.StatusOK, loginResponse{SentOTP: true})
}

type AccountIdentity struct {
	Sub int64
}

func (i AccountIdentity) Subject() string {
	return strconv.FormatInt(i.Sub, 10)
}

func (h *Handler) VerifyAuthenticationCode(w http.ResponseWriter, r *http.Request) {
	type verifyRequest struct {
		Email string `json:"email"`
		OTP   int64  `json:"otp"`
	}

	var req verifyRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	saved, err := h.db.Querier().SelectAccountLoginRequest(r.Context(), email)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if saved.Otp != req.OTP || saved.ExpiresAt < time.Now().UTC().Unix() {
		if saved.ExpiresAt < time.Now().UTC().Unix() {
			_ = h.db.Querier().DeleteAccountLoginRequest(r.Context(), email)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var subject int64
	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		var err error

		subject, err = q.OncesertAccountByEmail(r.Context(), email, time.Now().UTC().Unix())
		if err != nil {
			return err
		}

		return q.DeleteAccountLoginRequest(r.Context(), email)
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokens, err := h.authenticator.Issue(r.Context(), AccountIdentity{Sub: subject})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type sessionResponse struct {
		AccessToken           string `json:"accessToken"`
		RefreshToken          string `json:"refreshToken"`
		ExpiresIn             int64  `json:"expiresIn"`
		RefreshTokenExpiresIn int64  `json:"refreshTokenExpiresIn"`
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		AccessToken:           tokens.Access.Value,
		RefreshToken:          tokens.Refresh.Value,
		ExpiresIn:             expiresIn(tokens.Access.ExpiresAt),
		RefreshTokenExpiresIn: expiresIn(tokens.Refresh.ExpiresAt),
	})
}

func expiresIn(expiresAt int64) int64 {
	if expiresAt == 0 {
		return 0
	}

	now := time.Now().Unix()

	return expiresAt - now
}
