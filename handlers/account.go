package handlers

import (
	"net/http"

	"github.com/myrepo/myserver/middleware"
)

func (h *Handler) registerAccountRoutes() {
	h.AddHandler(http.MethodGet, h.routePath("/account"), middleware.AuthenticateBearer(h.authenticator, http.HandlerFunc(h.GetAccount)))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.AuthenticatedIdentity[AccountIdentity](r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	account, err := h.db.Querier().SelectAccountBySub(r.Context(), identity.Sub)
	if err != nil {
		if h.db.IsErrNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type accountResponse struct {
		Email     string `json:"email"`
		CreatedAt int64  `json:"created_at"`
	}

	writeJSON(w, http.StatusOK, accountResponse{
		Email:     account.Email,
		CreatedAt: account.CreatedAt,
	})
}
