package handler

import "net/http"

func init() {
	Add(http.MethodGet, "/", Root)
}

func Root(w http.ResponseWriter, r *http.Request) {
	handler.logger.Info("hit root")
	if handler.dev {
		_, _ = w.Write([]byte("ok (dev)\n"))
		return
	}

	_, _ = w.Write([]byte("ok\n"))
}
