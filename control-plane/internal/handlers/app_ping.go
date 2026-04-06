package handlers

import (
	"net/http"
	"time"
)

type AppPingHandler struct{}

func NewAppPingHandler() *AppPingHandler {
	return &AppPingHandler{}
}

func (h *AppPingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/app/ping" || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"now_utc": time.Now().UTC(),
	})
}
