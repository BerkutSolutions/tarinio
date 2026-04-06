package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"waf/control-plane/internal/appcompat"
)

type AppCompatHandler struct {
	runtimeRoot      string
	revisionStoreDir string
}

func NewAppCompatHandler(runtimeRoot, revisionStoreDir string) *AppCompatHandler {
	return &AppCompatHandler{
		runtimeRoot:      runtimeRoot,
		revisionStoreDir: revisionStoreDir,
	}
}

func (h *AppCompatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/app/compat" && r.Method == http.MethodGet:
		h.report(w)
	case r.URL.Path == "/api/app/compat/fix" && r.Method == http.MethodPost:
		h.fix(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *AppCompatHandler) report(w http.ResponseWriter) {
	report := appcompat.BuildReport(time.Now().UTC(), appcompat.PendingLegacyByModule(h.runtimeRoot, h.revisionStoreDir))
	writeJSON(w, http.StatusOK, report)
}

type appCompatFixRequest struct {
	ModuleID string `json:"module_id"`
}

func (h *AppCompatHandler) fix(w http.ResponseWriter, r *http.Request) {
	var req appCompatFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	moduleID := strings.TrimSpace(req.ModuleID)
	if moduleID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}
	if err := appcompat.EnsureLegacyModuleTransferred(h.runtimeRoot, h.revisionStoreDir, moduleID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":     err.Error(),
			"module_id": moduleID,
		})
		return
	}
	report := appcompat.BuildReport(time.Now().UTC(), appcompat.PendingLegacyByModule(h.runtimeRoot, h.revisionStoreDir))
	var item any
	for _, candidate := range report.Items {
		if candidate.ModuleID == moduleID {
			item = candidate
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"module_id": moduleID,
		"item":      item,
		"report":    report,
	})
}
