package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/myrepo/myserver/templates/base"
	rootpage "github.com/myrepo/myserver/templates/root"
)

func (h *Handler) registerRootRoutes() {
	h.Add(http.MethodGet, "/", h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	L := h.localizer.LocalizerFunc(h.localizer.PickLanguageFromRequest(r))

	if acceptsHTML(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page := base.Page(L, base.PageParams{
			Head: base.HeadParams{
				Title:                 "MyServer",
				Subtitle:              L("root.greeting"),
				Description:           L("root.greeting"),
				RobotsIndex:           true,
				RobotsGoogleTranslate: true,
			},
			Body: base.BodyParams{
				Content: rootpage.RootMain(L),
				Active:  "/",
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
