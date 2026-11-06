package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"app/database"
	"app/internal/httpx"
	"app/middleware"
	secrets "github.com/tavocg/go-secrets"
)

type accountResponse struct {
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
}

type changeEmailRequest struct {
	NewEmail string `json:"new_email"`
}

type changeEmailConfirmRequest struct {
	OTP string `json:"otp"`
}

func (h *Handler) registerAccountRoutes() {
	authenticated := func(next http.Handler) http.Handler {
		return middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, next)
	}
	canChangeEmail := func(fn http.HandlerFunc) http.Handler {
		return authenticated(middleware.RequirePermission(
			h.logger,
			h.localize,
			h.db.Querier(),
			database.PermissionChangeEmail,
			fn,
		))
	}

	h.AddHandler(http.MethodGet, "/account", authenticated(http.HandlerFunc(h.GetAccount)))
	h.AddHandler(http.MethodDelete, "/account", authenticated(http.HandlerFunc(h.DeleteAccount)))
	h.AddHandler(http.MethodPatch, "/account/email/change", canChangeEmail(h.RequestEmailChange))
	h.AddHandler(http.MethodPatch, "/account/email/change/confirm", canChangeEmail(h.ConfirmEmailChange))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.requireIdentity(w, r)
	if !ok {
		return
	}
	sub := identity.Sub

	account, ok := h.loadAccount(w, r, sub)
	if !ok {
		return
	}

	httpx.WriteJSON(w, http.StatusOK, accountResponse{
		Email:     account.Email,
		CreatedAt: account.CreatedAt,
	})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.requireIdentity(w, r)
	if !ok {
		return
	}
	sub := identity.Sub

	if err := h.authenticator.RevokeAll(r.Context(), identity.Claims); err != nil {
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

	if err := h.db.Querier().DeleteAccount(r.Context(), sub); err != nil {
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
	identity, ok := h.requireIdentity(w, r)
	if !ok {
		return
	}
	sub := identity.Sub
	subject := strconv.FormatInt(sub, 10)

	req, ok := decodeBody[changeEmailRequest](h, w, r,
		"err.invalid_email_change_body",
		"invalid account email change request body",
	)
	if !ok {
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

	account, ok := h.loadAccount(w, r, sub)
	if !ok {
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

	if !h.allowOTPIssuance(w, r, otpPurposeEmailChange, subject) {
		return
	}

	code, err := secrets.RandOTP()
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
	codeHash := h.signOTP(otpPurposeEmailChange, subject, code)
	expiresAt := time.Now().UTC().Add(otpLifetime).Unix()
	if err := h.db.Querier().UpsertAccountEmailChangeRequest(r.Context(), sub, newEmail, codeHash, expiresAt); err != nil {
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

	if !h.sendOTP(w, r, newEmail, code, otpPurposeEmailChange) {
		return
	}

	h.logger.Debug("generated account email change otp", "sub", sub, "email", newEmail, "otp", maskedOTP(h.dev, code), "expires_at", expiresAt)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	identity, ok := h.requireIdentity(w, r)
	if !ok {
		return
	}
	sub := identity.Sub
	subject := strconv.FormatInt(sub, 10)

	req, ok := decodeBody[changeEmailConfirmRequest](h, w, r,
		"err.invalid_email_confirm_body",
		"invalid account email confirm request body",
	)
	if !ok {
		return
	}

	if !h.allowOTPVerification(w, r, otpPurposeEmailChange, subject) {
		return
	}

	codeHash := h.signOTP(otpPurposeEmailChange, subject, req.OTP)
	if err := h.db.WithTx(r.Context(), func(q database.Querier) error {
		saved, err := q.ConsumeAccountEmailChangeRequest(r.Context(), sub, codeHash, time.Now().Unix())
		if err != nil {
			return err
		}

		return q.UpdateAccountEmail(r.Context(), sub, saved.Email)
	}); err != nil {
		if errors.Is(err, database.ErrEmailUnavailable) {
			h.writeStatus(
				w,
				r,
				http.StatusConflict,
				"err.email_unavailable",
				"account email change target is unavailable",
			)
			return
		}
		if h.db.IsErrNotFound(err) {
			h.writeStatus(
				w,
				r,
				http.StatusUnauthorized,
				"err.invalid_email_change_otp",
				"invalid or expired email change code",
			)
			return
		}
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.confirm_email_change",
			"failed to confirm account email change",
			"sub",
			sub,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
