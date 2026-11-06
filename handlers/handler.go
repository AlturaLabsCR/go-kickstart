// Package handlers registers the application's HTTP handlers.
package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"

	appauth "app/auth"
	"app/database"
	"app/internal/logging"
	"app/middleware"
	"github.com/tavocg/go-auth"
	"github.com/tavocg/go-email/mailer"
	"github.com/tavocg/go-i18n"
	"github.com/tavocg/go-secrets/otp"
)

type Handler struct {
	dev bool

	logger        logging.Logger
	db            database.Database
	authenticator auth.Authenticator[*appauth.Claims]
	localizer     *i18n.Localizer
	rootPrefix    string
	mailer        mailer.Mailer
	mailFrom      string
	otpSigner     *otp.Signer

	mux   *http.ServeMux
	entry http.Handler
}

type Options struct {
	Logger        logging.Logger
	Dev           bool
	DB            database.Database
	Authenticator auth.Authenticator[*appauth.Claims]
	Localizer     *i18n.Localizer
	RootPrefix    string
	Mailer        mailer.Mailer
	MailFrom      string
	OTPSigner     *otp.Signer
}

func NewHandler(opts Options) (*Handler, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	if opts.DB == nil {
		return nil, errors.New("handler database is required")
	}
	if opts.Localizer == nil {
		return nil, errors.New("handler localizer is required")
	}

	if opts.Authenticator == nil {
		return nil, errors.New("handler authenticator is required")
	}

	if opts.Mailer == nil {
		return nil, errors.New("handler mailer is required")
	}
	sender, err := mail.ParseAddress(opts.MailFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid handler mail sender: %w", err)
	}

	if opts.OTPSigner == nil {
		return nil, errors.New("handler OTP signer is required")
	}

	rootPrefix := normalizeRootPrefix(opts.RootPrefix)

	next := &Handler{
		dev:           opts.Dev,
		logger:        logger,
		db:            opts.DB,
		authenticator: opts.Authenticator,
		localizer:     opts.Localizer,
		rootPrefix:    rootPrefix,
		mailer:        opts.Mailer,
		mailFrom:      sender.Address,
		otpSigner:     opts.OTPSigner,
		mux:           http.NewServeMux(),
	}

	next.registerRoutes()
	next.entry = middleware.RequestLogger(next.logger, http.HandlerFunc(next.route))

	return next, nil
}

// Add registers a handler function relative to the configured root prefix.
func (h *Handler) Add(method, path string, fn http.HandlerFunc) {
	h.AddHandler(method, path, fn)
}

// AddHandler registers an http.Handler, including middleware-wrapped routes.
func (h *Handler) AddHandler(method, path string, handler http.Handler) {
	pattern := h.routePath(path)
	if method != "" {
		pattern = method + " " + pattern
	}

	h.mux.Handle(pattern, handler)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.entry == nil {
		panic("handler not initialized")
	}

	h.entry.ServeHTTP(w, r)
}

func (h *Handler) registerRoutes() {
	h.registerAuthRoutes()
	h.registerAccountRoutes()
	h.registerRootRoutes()
	h.registerStaticRoutes()
}

func (h *Handler) routePath(route string) string {
	route = strings.TrimSpace(route)
	if route == "" || route == "/" {
		if h.rootPrefix == "" {
			return "/"
		}

		return h.rootPrefix
	}

	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}

	return h.rootPrefix + route
}

func normalizeRootPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || prefix == "/" {
		return ""
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	return strings.TrimRight(prefix, "/")
}
