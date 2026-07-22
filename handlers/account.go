package handlers

import (
	"net/http"
	"time"

	"app/database"
	"app/middleware"
	"app/perms"
	secrets "github.com/tavocg/go-secrets"
)

func (h *Handler) registerAccountRoutes() {
	authenticated := func(fn http.HandlerFunc) http.Handler {
		return middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(fn))
	}
	defaultRole := func(fn http.HandlerFunc) http.Handler {
		return middleware.AuthenticateBearer(
			h.logger,
			h.localize,
			h.authenticator,
			middleware.RequirePermission(
				h.logger,
				h.localize,
				h.db,
				perms.PermissionChangeEmail,
				http.HandlerFunc(fn),
			),
		)
	}

	h.AddHandler(http.MethodGet, h.routePath("/account"), authenticated(h.GetAccount))
	h.AddHandler(http.MethodDelete, h.routePath("/account"), authenticated(h.DeleteAccount))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change"), defaultRole(h.RequestEmailChange))
	h.AddHandler(http.MethodPatch, h.routePath("/account/email/change/confirm"), defaultRole(h.ConfirmEmailChange))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(
			w,
			r,
			status,
			"err.missing_account_subject",
			"missing authenticated account subject",
		)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(
				w,
				r,
				http.StatusNotFound,
				err,
				"err.account_not_found",
				"account not found",
				"sub",
				sub,
			)
			return
		}

		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.select_account",
			"failed to select account",
			"sub",
			sub,
		)
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
		h.writeStatus(
			w,
			r,
			status,
			"err.missing_account_subject",
			"missing authenticated account subject",
		)
		return
	}

	if err := h.authenticator.RevokeAll(r.Context(), identity); err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.revoke_sessions",
			"failed to revoke account sessions",
			"sub",
			sub,
		)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.DeleteAccountEmailChangeRequest(r.Context(), sub); err != nil {
			return err
		}

		return q.DeleteAccount(r.Context(), sub)
	}); err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.delete_account",
			"failed to delete account",
			"sub",
			sub,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(
			w,
			r,
			status,
			"err.missing_account_subject",
			"missing authenticated account subject",
		)
		return
	}

	type changeEmailRequest struct {
		NewEmail string `json:"new_email"`
	}

	var req changeEmailRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(
			w,
			r,
			http.StatusBadRequest,
			err,
			"err.invalid_email_change_body",
			"invalid account email change request body",
		)
		return
	}

	newEmail := normalizeEmail(req.NewEmail)
	if newEmail == "" {
		h.writeStatus(
			w,
			r,
			http.StatusBadRequest,
			"err.invalid_email_change",
			"invalid account email change target",
			"email",
			req.NewEmail,
			"sub",
			sub,
		)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(
				w,
				r,
				http.StatusNotFound,
				err,
				"err.account_not_found",
				"account not found",
				"sub",
				sub,
			)
			return
		}

		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.select_account",
			"failed to select account",
			"sub",
			sub,
		)
		return
	}

	if account.Email == newEmail {
		h.writeStatus(
			w,
			r,
			http.StatusBadRequest,
			"err.email_change_same",
			"account email change target matches current email",
			"sub",
			sub,
			"email",
			newEmail,
		)
		return
	}

	otp, err := secrets.RandOTP()
	if err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.generate_email_change_otp",
			"failed to generate account email change otp",
			"sub",
			sub,
		)
		return
	}
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Unix()
	if err := h.db.Querier().UpsertAccountEmailChangeRequest(r.Context(), sub, newEmail, otp, expiresAt); err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.save_email_change",
			"failed to upsert account email change request",
			"sub",
			sub,
			"email",
			newEmail,
		)
		return
	}

	h.logger.Debug("generated account email change otp", "sub", sub, "email", newEmail, "otp", maskedOTP(h.dev, otp), "expires_at", expiresAt)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	_, sub, status, ok := authenticatedAccountClaimsAndSubject(r)
	if !ok {
		h.writeStatus(
			w,
			r,
			status,
			"err.missing_account_subject",
			"missing authenticated account subject",
		)
		return
	}

	type changeEmailConfirmRequest struct {
		OTP string `json:"otp"`
	}

	var req changeEmailConfirmRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(
			w,
			r,
			http.StatusBadRequest,
			err,
			"err.invalid_email_confirm_body",
			"invalid account email confirm request body",
		)
		return
	}

	saved, err := h.db.Querier().SelectAccountEmailChangeRequestBySub(r.Context(), sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			h.writeError(
				w,
				r,
				http.StatusUnauthorized,
				err,
				"err.missing_email_change",
				"missing account email change request",
				"sub",
				sub,
			)
			return
		}

		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.select_email_change",
			"failed to select account email change request",
			"sub",
			sub,
		)
		return
	}

	if saved.Otp != req.OTP || saved.ExpiresAt < time.Now().UTC().Unix() {
		if saved.ExpiresAt < time.Now().UTC().Unix() {
			_ = h.db.Querier().DeleteAccountEmailChangeRequest(r.Context(), sub)
		}

		h.writeStatus(
			w,
			r,
			http.StatusUnauthorized,
			"err.invalid_email_change_otp",
			"invalid account email change verification code",
			"sub",
			sub,
		)
		return
	}

	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		if err := q.DeleteAccountEmailChangeRequest(r.Context(), sub); err != nil {
			return err
		}

		return q.UpdateAccountEmail(r.Context(), sub, saved.Email)
	}); err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.confirm_email_change",
			"failed to confirm account email change",
			"sub",
			sub,
			"email",
			saved.Email,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
