package handlers

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func (h *Handler) registerStaticRoutes() {
	files, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to mount embedded static files")
	}

	h.AddHandler(http.MethodGet, "/s/", http.StripPrefix("/s/", http.FileServerFS(files)))
}
