package handlers

import (
	"app/routes"
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

	prefix, _ := h.URL(routes.Static)
	server := http.StripPrefix(prefix, http.FileServerFS(files))
	h.AddRoute(routes.Static, server)
	h.AddRoute(routes.APIDocs, server)
}
