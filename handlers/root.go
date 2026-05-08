package handlers

import "net/http"

func (h *Handler) registerRootRoutes() {
	h.Add(http.MethodGet, "/", h.Root)
}

func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("hit root")
	if h.dev {
		_, _ = w.Write([]byte("ok (dev)\n"))
		return
	}

	_, _ = w.Write([]byte("ok\n"))
}
