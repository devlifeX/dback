package v1

import (
	"net/http"
)

func (h *Handler) systemServerInfo(w http.ResponseWriter, r *http.Request) {
	info := h.App.ServerInfo(r.Context(), h.Cfg.DataDir)
	writeJSON(w, http.StatusOK, info)
}
