package handlers

import (
	"net/http"
	"time"

	"app/database"
	"app/middleware"
)

func (h *Handler) registerAccountRoutes() {
	authenticated := func(fn http.HandlerFunc) http.Handler {
		return middleware.AuthenticateBearer(h.logger, h.authenticator, http.HandlerFunc(fn))
	}

	h.AddHandler(http.MethodGet, h.routePath("/account"), authenticated(h.GetAccount))
	h.AddHandler(http.MethodDelete, h.routePath("/account"), authenticated(h.DeleteAccount))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change"), authenticated(h.RequestEmailChange))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change/confirm"), authenticated(h.ConfirmEmailChange))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(w, r, status, "missing authenticated account subject")
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(w, r, http.StatusNotFound, err, "account not found", "sub", sub)
			return
		}

		h.writeError(w, r, http.StatusInternalServerError, err, "failed to select account", "sub", sub)
		return
	}

	type accountResponse struct {
		Email     string `json:"email"`
		CreatedAt int64  `json:"created_at"`
	}

	writeJSON(w, http.StatusOK, accountResponse{
		Email:     account.Email,
		CreatedAt: account.CreatedAt,
	})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	identity, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(w, r, status, "missing authenticated account subject")
		return
	}

	if err := h.authenticator.RevokeAll(r.Context(), identity); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to revoke account sessions", "sub", sub)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.DeleteAccountEmailChangeRequest(r.Context(), sub); err != nil {
			return err
		}

		return q.DeleteAccount(r.Context(), sub)
	}); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to delete account", "sub", sub)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(w, r, status, "missing authenticated account subject")
		return
	}

	type changeEmailRequest struct {
		NewEmail string `json:"new_email"`
	}

	var req changeEmailRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err, "invalid account email change request body")
		return
	}

	newEmail := normalizeEmail(req.NewEmail)
	if newEmail == "" {
		h.writeStatus(w, r, http.StatusBadRequest, "invalid account email change target", "email", req.NewEmail, "sub", sub)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(w, r, http.StatusNotFound, err, "account not found", "sub", sub)
			return
		}

		h.writeError(w, r, http.StatusInternalServerError, err, "failed to select account", "sub", sub)
		return
	}

	if account.Email == newEmail {
		h.writeStatus(w, r, http.StatusBadRequest, "account email change target matches current email", "sub", sub, "email", newEmail)
		return
	}

	otp, err := randomOTP()
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to generate account email change otp", "sub", sub)
		return
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute).Unix()
	if err := h.db.Querier().UpsertAccountEmailChangeRequest(r.Context(), sub, newEmail, otp, expiresAt); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to upsert account email change request", "sub", sub, "email", newEmail)
		return
	}

	h.logger.Debug("generated account email change otp", "sub", sub, "email", newEmail, "otp", otp, "expires_at", expiresAt)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(w, r, status, "missing authenticated account subject")
		return
	}

	type changeEmailConfirmRequest struct {
		OTP int64 `json:"otp"`
	}

	var req changeEmailConfirmRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, err, "invalid account email confirm request body")
		return
	}

	saved, err := h.db.Querier().SelectAccountEmailChangeRequestBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(w, r, http.StatusUnauthorized, err, "missing account email change request", "sub", sub)
			return
		}

		h.writeError(w, r, http.StatusInternalServerError, err, "failed to select account email change request", "sub", sub)
		return
	}

	if saved.Otp != req.OTP || saved.ExpiresAt < time.Now().UTC().Unix() {
		if saved.ExpiresAt < time.Now().UTC().Unix() {
			_ = h.db.Querier().DeleteAccountEmailChangeRequest(r.Context(), sub)
		}

		h.writeStatus(w, r, http.StatusUnauthorized, "invalid account email change verification code", "sub", sub)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.DeleteAccountEmailChangeRequest(r.Context(), sub); err != nil {
			return err
		}

		return q.UpdateAccountEmail(r.Context(), sub, saved.Email)
	}); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, err, "failed to confirm account email change", "sub", sub, "email", saved.Email)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
