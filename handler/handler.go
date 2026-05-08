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
	logger Logger
	paths  []string
}

func NewHandler() *Handler {
	return &Handler{
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

var handler = NewHandler()
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
	return rootMux
}
