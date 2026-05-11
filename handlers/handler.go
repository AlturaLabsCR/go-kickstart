// Package handlers registers the application's HTTP handlers.
package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

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
	authenticator auth.Authenticator[AccountIdentity]
	localizer     *i18n.Localizer
	rootPath      string

	mux   *http.ServeMux
	paths []string
}

type Options struct {
	Logger    Logger
	Dev       bool
	DB        database.Database
	Localizer *i18n.Localizer
	RootPath  string
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

	rootPath := normalizeRootPath(opts.RootPath)

	next := &Handler{
		initialized: true,
		dev:         opts.Dev,
		logger:      logger,
		db:          opts.DB,
		localizer:   localizer,
		rootPath:    rootPath,
		mux:         http.NewServeMux(),
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
	h.registerRootRoutes()
	h.registerStaticRoutes()
}

func normalizeRootPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	path = strings.TrimRight(path, "/")
	if path == "" {
		return "/"
	}

	return path
}
