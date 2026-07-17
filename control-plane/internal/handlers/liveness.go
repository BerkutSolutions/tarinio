package handlers

import "net/http"

// LivenessHandler is safe for public container probes. Operational details are
// exposed only through the authenticated administration health endpoint.
type LivenessHandler struct{}

func NewLivenessHandler() *LivenessHandler { return &LivenessHandler{} }

func (h *LivenessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
