package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"

	"app/internal/browser"
	"app/middleware"
	"app/templates/base"
	homepage "app/templates/home"
	"app/templates/meta"
	"github.com/a-h/templ"
)

func (h *Handler) browserPage(w http.ResponseWriter, r *http.Request, status int, title string, content templ.Component) {
	documentHeaders(w)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "Accept-Language")
	w.WriteHeader(status)
	if err := base.Page(base.PageParams{Links: h.links, Head: base.HeadParams{Title: meta.AppTitle, Subtitle: title}, Body: base.BodyParams{Content: base.BrowserSurface(content, h.localize(r, "web.update_error"))}}).Render(r.Context(), w); err != nil {
		h.logger.Error("failed to render browser page", "error", err)
	}
}

func (h *Handler) browserHome(w http.ResponseWriter, r *http.Request, cookies browser.Cookies, status int, notice string, verifyEmail bool) {
	login, home := h.links.Mount("/login"), h.links.Mount("/home")
	view := homepage.View{VerifyEmail: verifyEmail, Notice: notice, Failed: status >= 400}
	// Verify once, then reuse the request identity for profile and session reads.
	protected := middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, _ := middleware.AuthenticatedIdentity(r.Context())
		account, ok := h.loadAccount(w, r, identity.Sub)
		if !ok {
			return
		}
		view.Email, view.CreatedAt = account.Email, account.CreatedAt
		h.ListSessions(w, r)
	}))
	reply := browser.Call(r, protected, nil, cookies.Read(r, "access"))
	if reply.Status == http.StatusUnauthorized {
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, login, http.StatusSeeOther)
		return
	}
	if reply.Status != http.StatusOK {
		for key, values := range reply.HeaderMap {
			w.Header()[key] = values
		}
		w.WriteHeader(reply.Status)
		_, _ = w.Write(reply.Body.Bytes())
		return
	}
	var sessions accountSessionsResponse
	if err := json.Unmarshal(reply.Body.Bytes(), &sessions); err != nil {
		h.writeError(
			w,
			r,
			http.StatusInternalServerError,
			err,
			"err.select_sessions",
			"failed to decode browser sessions",
		)
		return
	}
	rows := make([]homepage.Session, 0, len(sessions.Sessions))
	for _, session := range sessions.Sessions {
		browserName, osName := deviceNames(session.UserAgent)
		rows = append(rows, homepage.Session{ID: session.ID, Device: session.UserAgent, Browser: browserName, OS: osName, Created: session.CreatedAt, Used: session.LastUsedAt, Expires: session.ExpiresAt, Current: session.Current})
	}
	next := ""
	if sessions.NextCursor != "" {
		next = home + "?cursor=" + url.QueryEscape(sessions.NextCursor)
	}
	tr := func(key string) string { return h.localize(r, key) }
	h.browserPage(w, r, status, tr("web.home"), homepage.Home(login, home, next, rows, h.language(r), view, tr))
}

// Browser forms adapt existing JSON handlers, retaining live permission checks.
func (h *Handler) browserEmailChange(w http.ResponseWriter, r *http.Request, cookies browser.Cookies) {
	confirm := r.PostForm.Get("action") == "email_confirm"
	var payload any = changeEmailRequest{NewEmail: r.PostForm.Get("new_email")}
	next := http.HandlerFunc(h.RequestEmailChange)
	fallback := "err.save_email_change"
	if confirm {
		payload = changeEmailConfirmRequest{OTP: r.PostForm.Get("otp")}
		next, fallback = h.ConfirmEmailChange, "err.confirm_email_change"
	}
	authenticated := false
	protected := middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated = true
		h.requireEmailChangePermission(next).ServeHTTP(w, r)
	}))
	reply := browser.Call(r, protected, payload, cookies.Read(r, "access"))
	if !authenticated && reply.Status == http.StatusUnauthorized {
		http.Redirect(w, r, h.links.Mount("/login"), http.StatusSeeOther)
		return
	}
	if reply.Status >= 400 {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(reply.Body.Bytes(), &failure) != nil || failure.Error == "" {
			failure.Error = h.localize(r, fallback)
		}
		h.browserHome(w, r, cookies, reply.Status, failure.Error, confirm)
		return
	}
	step := "verify"
	if confirm {
		step = "updated"
	}
	http.Redirect(w, r, h.links.Mount("/home")+"?email="+step, http.StatusSeeOther)
}
