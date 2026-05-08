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
	logger      Logger
	dev         bool
	initialized bool
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
		logger:      logger,
		dev:         opts.Dev,
		initialized: true,
	}

	if handler != nil {
		next.paths = append(next.paths, handler.paths...)
	}

	handler = next

	return next
}

var handler = &Handler{}
var rootMux = http.NewServeMux()

func Add(method, path string, fn http.HandlerFunc) {
	pattern := path
	if method != "" {
		pattern = method + " " + path
	}

	handler.paths = append(handler.paths, path)
	rootMux.HandleFunc(pattern, fn)
}

func Mux() *http.ServeMux {
	if handler == nil || !handler.initialized {
		panic("handler not initialized")
	}

	return rootMux
}
