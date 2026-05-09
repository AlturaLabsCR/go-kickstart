package handlers

import "net/http"

func (h *Handler) registerRootRoutes() {
	h.Add(http.MethodGet, "/", h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	L := h.localizer.LocalizerFunc(h.localizer.PickLanguageFromRequest(r))
	_, _ = w.Write([]byte(L("root.greeting") + "\n"))
}
