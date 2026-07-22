package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"app/templates/base"
	"app/templates/meta"
	"app/templates/root"
)

func (h *Handler) registerRootRoutes() {
	h.Add(http.MethodGet, h.routePath("/"), h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	L := func(key string, args ...any) string {
		return h.localize(r, key, args...)
	}

	if acceptsHTML(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := base.Page(L, base.PageParams{
			Head: base.HeadParams{
				Title:       meta.AppTitle,
				Subtitle:    L("root.greeting"),
				RobotsIndex: true,
			},
			Body: base.BodyParams{
				Content: root.RootMain(L),
				Active:  h.rootPrefix,
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": L("root.greeting"),
	})
}

func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "application/xhtml+xml")
}
