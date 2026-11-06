package handlers

import (
	"net/http"
	"strings"

	"app/templates/base"
	"app/templates/meta"
	"app/templates/root"
)

func (h *Handler) registerRootRoutes() {
	route := "/"
	if h.rootPrefix == "" {
		route = "/{$}"
	}
	h.Add(http.MethodGet, route, h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	L := func(key string, args ...any) string {
		return h.localize(r, key, args...)
	}

	if acceptsHTML(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := base.Page(base.PageParams{
			Head: base.HeadParams{
				Title:       meta.AppTitle,
				Subtitle:    L("root.greeting"),
				RobotsIndex: true,
			},
			Body: base.BodyParams{
				Content: root.RootMain(L),
			},
		})

		if err := page.Render(r.Context(), w); err != nil {
			h.writeError(
				w,
				r,
				http.StatusInternalServerError,
				err,
				"err.render_root",
				"failed to render root page",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": L("root.greeting"),
	})
}

func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "application/xhtml+xml")
}
