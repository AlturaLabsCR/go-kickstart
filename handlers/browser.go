package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"app/internal/browser"
	"app/internal/httpx"
	"app/middleware"
	"app/routes"
	"app/templates/base"
	pages "app/templates/browser"
	"app/templates/meta"
	"github.com/a-h/templ"
	"github.com/mssola/useragent"
	goauth "github.com/tavocg/go-auth"
)

// Remove this file and its registration call, plus routes.LoginPage, to remove
// the browser surface. API handlers, JSON contracts and Bearer authentication are
// not modified.
func (h *Handler) registerBrowserRoutes() {
	const loginPath, homePath = "/login", "/home"
	cookies := browser.Cookies{Path: h.links.Mount("/"), Secure: !h.dev}
	if !strings.HasSuffix(cookies.Path, "/") {
		cookies.Path += "/"
	}
	protected := middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(h.ListSessions))
	login := h.links.Mount(loginPath)
	home := h.links.Mount(homePath)
	page := func(w http.ResponseWriter, r *http.Request, status int, title string, content templ.Component) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Add("Vary", "Accept-Language")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.WriteHeader(status)
		if err := base.Page(base.PageParams{Links: h.links, Head: base.HeadParams{Title: meta.AppTitle, Subtitle: title}, Body: base.BodyParams{Content: pages.Surface(h.links, content, h.localize(r, "web.update_error"))}}).Render(r.Context(), w); err != nil {
			h.logger.Error("failed to render browser page", "error", err)
		}
	}
	showLogin := func(w http.ResponseWriter, r *http.Request, status int, email, message string, verify bool) {
		tr := func(key string) string { return h.localize(r, key) }
		page(w, r, status, tr("web.login"), pages.Login(login, email, message, verify, status == http.StatusOK && r.Method == http.MethodGet && cookies.Read(r, "refresh") != "" && r.URL.Query().Get("resume") != "0" && !prefetch(r), tr))
	}

	// GET never consumes a rotating token; the browser coordinates an automatic
	// POST between tabs. A valid access cookie can redirect immediately.
	h.AddRoute(routes.LoginPage, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated, err := h.browserAuthenticated(r, cookies)
		if err != nil {
			h.logger.Error("failed to verify browser session", "error", err)
			status, _ := httpx.ErrorStatus(r.Context(), http.StatusServiceUnavailable, "err.verify_bearer")
			showLogin(w, r, status, "", h.localize(r, "err.verify_bearer"), false)
			return
		}
		if authenticated {
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, home, http.StatusSeeOther)
			return
		}
		showLogin(w, r, http.StatusOK, r.URL.Query().Get("email"), "", r.URL.Query().Get("step") == "verify")
	}))

	// Cross-origin form posts are rejected. The debug log records why and the
	// headers the check compares, because tunnels and proxies can rewrite Host.
	csrf := http.NewCrossOriginProtection()
	for _, origin := range h.csrfOrigins {
		if err := csrf.AddTrustedOrigin(origin); err != nil {
			h.logger.Error("failed to trust browser form origin", "origin", origin, "error", err)
		}
	}
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.writeStatus(
			w,
			r,
			http.StatusForbidden,
			"err.permission_denied",
			"cross-origin browser form rejected",
			"reason",
			csrf.Check(r),
			"origin",
			r.Header.Get("Origin"),
			"host",
			r.Host,
			"sec_fetch_site",
			r.Header.Get("Sec-Fetch-Site"),
		)
	}))
	h.AddHandler("POST", loginPath, csrf.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			showLogin(w, r, 400, "", h.localize(r, "err.invalid_login_body"), false)
			return
		}
		email := r.PostForm.Get("email")
		action := r.PostForm.Get("action")

		if action == "resume" {
			authenticated, status := h.resumeSession(w, r, cookies)
			if authenticated {
				http.Redirect(w, r, home, http.StatusSeeOther)
				return
			}
			message := ""
			if status >= 500 {
				message = h.localize(r, "err.refresh_session")
			}
			showLogin(w, r, status, "", message, false)
			return
		}
		if action == "revoke" {
			request := r.Clone(r.Context())
			id := r.PostForm.Get("session_id")
			request.SetPathValue("id", id)
			current := false
			revoke := middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				identity, _ := middleware.AuthenticatedIdentity(r.Context())
				current = identity.Claims.SessionID == id
				h.RevokeSession(w, r)
			}))
			reply := browser.Call(request, revoke, nil, cookies.Read(r, "access"))
			if reply.Status == 401 {
				http.Redirect(w, r, login, http.StatusSeeOther)
				return
			}
			if reply.Status >= 400 {
				showLogin(w, r, reply.Status, "", h.localize(r, "err.revoke_session"), false)
				return
			}
			if current {
				cookies.Clear(w)
				http.Redirect(w, r, login, http.StatusSeeOther)
			} else {
				http.Redirect(w, r, home, http.StatusSeeOther)
			}
			return
		}
		if action == "logout" {
			// A refresh cookie can expire before a still-valid access token.
			if access := cookies.Read(r, "access"); access != "" {
				if claims, err := h.authenticator.Verify(r.Context(), access); err == nil {
					request := r.Clone(r.Context())
					request.SetPathValue("id", claims.SessionID)
					revoke := middleware.AuthenticateBearer(h.logger, h.localize, h.authenticator, http.HandlerFunc(h.RevokeSession))
					reply := browser.Call(request, revoke, nil, access)
					if reply.Status >= 400 {
						showLogin(w, r, reply.Status, "", h.localize(r, "err.revoke_session"), false)
						return
					}
				}
			}
			if token := cookies.Read(r, "refresh"); token != "" {
				if err := h.authenticator.Revoke(r.Context(), token); err != nil {
					h.writeError(
						w,
						r,
						http.StatusInternalServerError,
						err,
						"err.revoke_session",
						"failed to revoke browser session",
					)
					return
				}
			}
			cookies.Clear(w)
			http.Redirect(w, r, login, http.StatusSeeOther)
			return
		}
		var reply *browser.Reply
		switch action {
		case "login":
			reply = browser.Call(r, http.HandlerFunc(h.LoginOrCreateAccount), loginRequest{Email: email}, "")
		case "verify":
			reply = browser.Call(r, http.HandlerFunc(h.VerifyAuthenticationCode), verifyRequest{Email: email, OTP: r.PostForm.Get("otp")}, "")
		default:
			showLogin(w, r, 400, email, h.localize(r, "err.invalid_login_body"), false)
			return
		}
		if reply.Status >= 400 {
			var failure struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(reply.Body.Bytes(), &failure) != nil {
				failure.Error = h.localize(r, "err.finalize_login")
			}
			showLogin(w, r, reply.Status, email, failure.Error, action == "verify")
			return
		}
		if action == "login" {
			http.Redirect(w, r, login+"?step=verify&email="+url.QueryEscape(email), http.StatusSeeOther)
			return
		}
		var tokens browser.Tokens
		if err := json.Unmarshal(reply.Body.Bytes(), &tokens); err != nil || tokens.Access == "" || tokens.Refresh == "" {
			showLogin(w, r, 500, email, h.localize(r, "err.issue_tokens"), false)
			return
		}
		cookies.Save(w, tokens)
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, home, http.StatusSeeOther)
	})))
	h.Add("GET", homePath, func(w http.ResponseWriter, r *http.Request) {
		reply := browser.Call(r, protected, nil, cookies.Read(r, "access"))
		if reply.Status == 401 {
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, login, http.StatusSeeOther)
			return
		}
		if reply.Status != 200 {
			for key, values := range reply.HeaderMap {
				w.Header()[key] = values
			}
			w.WriteHeader(reply.Status)
			_, _ = w.Write(reply.Body.Bytes())
			return
		}
		var sessions accountSessionsResponse
		if err := json.Unmarshal(reply.Body.Bytes(), &sessions); err != nil {
			showLogin(w, r, 500, "", h.localize(r, "err.select_sessions"), false)
			return
		}
		rows := make([]pages.Session, 0, len(sessions.Sessions))
		for _, session := range sessions.Sessions {
			browserName, osName := deviceNames(session.UserAgent)
			rows = append(rows, pages.Session{
				ID:      session.ID,
				Device:  session.UserAgent,
				Browser: browserName,
				OS:      osName,
				Created: session.CreatedAt,
				Used:    session.LastUsedAt,
				Expires: session.ExpiresAt,
				Current: session.Current,
			})
		}
		tr := func(key string) string { return h.localize(r, key) }
		next := ""
		if sessions.NextCursor != "" {
			next = home + "?cursor=" + url.QueryEscape(sessions.NextCursor)
		}
		page(w, r, 200, tr("web.home"), pages.Home(login, home, next, rows, h.language(r), tr))
	})
}

// browserAuthenticated distinguishes invalid credentials from dependency failures.
func (h *Handler) browserAuthenticated(r *http.Request, cookies browser.Cookies) (bool, error) {
	token := cookies.Read(r, "access")
	if token == "" {
		return false, nil
	}
	claims, err := h.authenticator.Verify(r.Context(), token)
	if errors.Is(err, goauth.ErrInvalidToken) || errors.Is(err, goauth.ErrExpiredToken) {
		return false, nil
	}
	return err == nil && claims != nil, err
}

// resumeSession runs only after CSRF checking on POST. Recheck access after the
// browser acquires its origin-wide lock: another tab may already have renewed.
// Failures never clear cookies, because their response may arrive after a newer
// successful login/renewal. Only explicit sign out removes browser credentials.
func (h *Handler) resumeSession(w http.ResponseWriter, r *http.Request, cookies browser.Cookies) (bool, int) {
	if prefetch(r) {
		return false, http.StatusOK
	}
	authenticated, err := h.browserAuthenticated(r, cookies)
	if err != nil {
		h.logger.Error("failed to verify browser session before renewal", "error", err)
		status, _ := httpx.ErrorStatus(r.Context(), http.StatusServiceUnavailable, "err.verify_bearer")
		return false, status
	}
	if authenticated {
		return true, http.StatusOK
	}
	refresh := cookies.Read(r, "refresh")
	if refresh == "" {
		return false, http.StatusOK
	}
	reply := browser.Call(r, http.HandlerFunc(h.RefreshSession), refreshRequest{RefreshToken: refresh}, "")
	if reply.Status == http.StatusUnauthorized {
		return false, http.StatusOK
	}
	if reply.Status != http.StatusOK {
		return false, reply.Status
	}
	var tokens browser.Tokens
	if err := json.Unmarshal(reply.Body.Bytes(), &tokens); err != nil || tokens.Access == "" || tokens.Refresh == "" || tokens.AccessTTL <= 0 || tokens.RefreshTTL <= 0 {
		return false, http.StatusInternalServerError
	}
	cookies.Save(w, tokens)
	return true, http.StatusOK
}

// prefetch reports speculative requests whose response, including cookies, may
// be discarded. Renewing there would consume the rotating refresh token.
func prefetch(r *http.Request) bool {
	purpose := strings.ToLower(r.Header.Get("Sec-Purpose") + " " + r.Header.Get("Purpose"))
	return strings.Contains(purpose, "prefetch") || strings.Contains(purpose, "prerender")
}

// deviceNames extracts the browser and operating system shown as device icons.
func deviceNames(userAgent string) (browserName, osName string) {
	ua := useragent.New(userAgent)
	browserName, _ = ua.Browser()
	osName = ua.OSInfo().Name
	switch ua.Platform() {
	case "iPad", "iPhone", "iPod touch":
		osName = "iOS"
	}
	return browserName, osName
}
