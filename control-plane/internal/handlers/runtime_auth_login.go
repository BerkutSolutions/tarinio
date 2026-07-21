package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const runtimeAuthHeader = "X-WAF-Runtime-Token"

type basicAuthLoginRecorder interface {
	MarkBasicAuthLogin(siteID, username string, when time.Time) error
}

type RuntimeBasicAuthLoginHandler struct {
	token    string
	recorder basicAuthLoginRecorder
}

func NewRuntimeBasicAuthLoginHandler(token string, recorder basicAuthLoginRecorder) *RuntimeBasicAuthLoginHandler {
	return &RuntimeBasicAuthLoginHandler{token: strings.TrimSpace(token), recorder: recorder}
}

func (h *RuntimeBasicAuthLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.token == "" || r.Header.Get(runtimeAuthHeader) != h.token {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var body struct {
		SiteID     string `json:"site_id"`
		Username   string `json:"username"`
		OccurredAt string `json:"occurred_at"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	when, err := time.Parse(time.RFC3339, strings.TrimSpace(body.OccurredAt))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid occurrence time"})
		return
	}
	if err := h.recorder.MarkBasicAuthLogin(body.SiteID, body.Username, when); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
