package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/services"
)

type administrationScriptsService interface {
	Catalog() services.AdminScriptCatalog
	Run(ctx context.Context, scriptID string, input map[string]string) (services.AdminScriptRunResult, error)
	Download(runID string) (string, []byte, error)
}

type AdministrationScriptsHandler struct {
	service administrationScriptsService
}

func NewAdministrationScriptsHandler(service administrationScriptsService) *AdministrationScriptsHandler {
	return &AdministrationScriptsHandler{service: service}
}

func (h *AdministrationScriptsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "administration scripts service unavailable"})
		return
	}
	switch {
	case r.URL.Path == "/api/administration/scripts" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, h.service.Catalog())
	case strings.HasPrefix(r.URL.Path, "/api/administration/scripts/") && strings.HasSuffix(r.URL.Path, "/run") && r.Method == http.MethodPost:
		h.handleRun(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/scripts/runs/") && strings.HasSuffix(r.URL.Path, "/download") && r.Method == http.MethodGet:
		h.handleDownload(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *AdministrationScriptsHandler) handleRun(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/administration/scripts/")
	scriptID := strings.TrimSuffix(trimmed, "/run")
	scriptID = strings.Trim(scriptID, "/")
	if scriptID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "script id is required"})
		return
	}
	var payload struct {
		Input map[string]string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json payload"})
		return
	}
	result, err := h.service.Run(r.Context(), scriptID, payload.Input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": strings.TrimSpace(err.Error())})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdministrationScriptsHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/administration/scripts/runs/")
	runID := strings.TrimSuffix(trimmed, "/download")
	runID = strings.Trim(runID, "/")
	fileName, content, err := h.service.Download(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": strings.TrimSpace(err.Error())})
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(fileName, `"`, "")+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
