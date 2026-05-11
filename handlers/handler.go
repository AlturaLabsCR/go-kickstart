// Package handlers registers the application's HTTP handlers.
package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	appauth "github.com/myrepo/myserver/auth"
	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/middleware"
	"github.com/tavocg/go-auth"
	"github.com/tavocg/go-i18n"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Handler struct {
	initialized bool
	dev         bool

	logger        Logger
	db            database.Database
	authenticator auth.Authenticator[*appauth.Claims]
	localizer     *i18n.Localizer
	rootPrefix    string

	mux   *http.ServeMux
	paths []string
}

type Options struct {
	Logger        Logger
	Dev           bool
	DB            database.Database
	Authenticator auth.Authenticator[*appauth.Claims]
	Localizer     *i18n.Localizer
	RootPrefix    string
}

func NewHandler(opts Options) *Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	localizer := opts.Localizer
	if localizer == nil {
		panic("handler localizer is required")
	}

	authenticator := opts.Authenticator
	if authenticator == nil {
		panic("handler authenticator is required")
	}

	rootPrefix := normalizeRootPrefix(opts.RootPrefix)

	next := &Handler{
		initialized:   true,
		dev:           opts.Dev,
		logger:        logger,
		db:            opts.DB,
		authenticator: authenticator,
		localizer:     localizer,
		rootPrefix:    rootPrefix,
		mux:           http.NewServeMux(),
	}

	next.registerRoutes()

	return next
}

func (h *Handler) Add(method, path string, fn http.HandlerFunc) {
	h.AddHandler(method, path, fn)
}

func (h *Handler) AddHandler(method, path string, handler http.Handler) {
	pattern := path
	if method != "" {
		pattern = method + " " + path
	}

	h.paths = append(h.paths, path)
	h.mux.Handle(pattern, middleware.RequestLogger(h.logger, pattern, handler))
}

func (h *Handler) Mux() *http.ServeMux {
	if h == nil || !h.initialized {
		panic("handler not initialized")
	}

	return h.mux
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
