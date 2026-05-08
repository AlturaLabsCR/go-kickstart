package handler

import (
	"log/slog"
	"net/http"
	"os"
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

	logger      Logger

	mux         *http.ServeMux
	paths       []string
}

type Options struct {
	Logger Logger
	Dev    bool
}

func NewHandler(opts Options) *Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	next := &Handler{
		initialized: true,
		dev:         opts.Dev,
		logger:      logger,
		mux:         http.NewServeMux(),
	}

	next.registerRoutes()

	return next
}

func (h *Handler) Add(method, path string, fn http.HandlerFunc) {
	pattern := path
	if method != "" {
		pattern = method + " " + path
	}

	h.paths = append(h.paths, path)
	h.mux.HandleFunc(pattern, fn)
}

func (h *Handler) Mux() *http.ServeMux {
	if h == nil || !h.initialized {
		panic("handler not initialized")
	}

	return h.mux
}

func (h *Handler) registerRoutes() {
	h.registerRootRoutes()
}
