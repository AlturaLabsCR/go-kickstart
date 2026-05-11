package handlers

import (
	"net/http"
	"time"

	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/middleware"
)

func (h *Handler) registerAccountRoutes() {
	authenticated := func(fn http.HandlerFunc) http.Handler {
		return middleware.AuthenticateBearer(h.authenticator, http.HandlerFunc(fn))
	}

	h.AddHandler(http.MethodGet, h.routePath("/account"), authenticated(h.GetAccount))
	h.AddHandler(http.MethodDelete, h.routePath("/account"), authenticated(h.DeleteAccount))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change"), authenticated(h.RequestEmailChange))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change/confirm"), authenticated(h.ConfirmEmailChange))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.AuthenticatedIdentity[AccountIdentity](r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), identity.Sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
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
	identity, ok := middleware.AuthenticatedIdentity[AccountIdentity](r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.authenticator.RevokeAll(r.Context(), identity); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.DeleteAccountEmailChangeRequest(r.Context(), identity.Sub); err != nil {
			return err
		}

		return q.DeleteAccount(r.Context(), identity.Sub)
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.AuthenticatedIdentity[AccountIdentity](r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type changeEmailRequest struct {
		NewEmail string `json:"new_email"`
	}

	var req changeEmailRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newEmail := normalizeEmail(req.NewEmail)
	if newEmail == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), identity.Sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if account.Email == newEmail {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	otp, err := randomOTP()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute).Unix()
	if err := h.db.Querier().UpsertAccountEmailChangeRequest(r.Context(), identity.Sub, newEmail, otp, expiresAt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("generated account email change otp", "sub", identity.Sub, "email", newEmail, "otp", otp, "expires_at", expiresAt)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.AuthenticatedIdentity[AccountIdentity](r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type changeEmailConfirmRequest struct {
		OTP int64 `json:"otp"`
	}

	var req changeEmailConfirmRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	saved, err := h.db.Querier().SelectAccountEmailChangeRequestBySub(r.Context(), identity.Sub)
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
			_ = h.db.Querier().DeleteAccountEmailChangeRequest(r.Context(), identity.Sub)
		}

		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.UpdateAccountEmail(r.Context(), identity.Sub, saved.Email); err != nil {
			return err
		}

		return q.DeleteAccountEmailChangeRequest(r.Context(), identity.Sub)
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
