package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/myrepo/myserver/templates/base"
	"github.com/myrepo/myserver/templates/meta"
	"github.com/myrepo/myserver/templates/root"
)

func (h *Handler) registerRootRoutes() {
	h.Add(http.MethodGet, h.rootPath, h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	L := h.localizer.LocalizerFunc(h.localizer.PickLanguageFromRequest(r))

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
				Active:  h.rootPath,
			},
		})

		if err := page.Render(r.Context(), w); err != nil {
			http.Error(w, "failed to render root page", http.StatusInternalServerError)
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
