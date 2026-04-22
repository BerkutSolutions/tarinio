package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/enterprise"
	"waf/control-plane/internal/services"
)

type enterpriseSettingsService interface {
	GetSettingsView() (enterprise.SettingsView, error)
	UpdateSettings(ctx context.Context, input services.EnterpriseSettingsInput) (enterprise.SettingsView, error)
	CreateSCIMToken(ctx context.Context, displayName string) (enterprise.CreateSCIMTokenResult, error)
	DeleteSCIMToken(ctx context.Context, id string) error
	BuildSupportBundle() ([]byte, string, error)
}

type EnterpriseHandler struct {
	service enterpriseSettingsService
}

func NewEnterpriseHandler(service enterpriseSettingsService) *EnterpriseHandler {
	return &EnterpriseHandler{service: service}
}

func (h *EnterpriseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/administration/enterprise" && r.Method == http.MethodGet:
		h.getSettings(w, r)
	case r.URL.Path == "/api/administration/enterprise" && r.Method == http.MethodPut:
		h.updateSettings(w, r)
	case r.URL.Path == "/api/administration/enterprise/scim-tokens" && r.Method == http.MethodPost:
		h.createSCIMToken(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/administration/enterprise/scim-tokens/") && r.Method == http.MethodDelete:
		h.deleteSCIMToken(w, r)
	case r.URL.Path == "/api/administration/support-bundle" && r.Method == http.MethodGet:
		h.downloadSupportBundle(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *EnterpriseHandler) getSettings(w http.ResponseWriter, _ *http.Request) {
	view, err := h.service.GetSettingsView()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *EnterpriseHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input services.EnterpriseSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	view, err := h.service.UpdateSettings(withActorIP(r), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *EnterpriseHandler) createSCIMToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DisplayName string `json:"display_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	result, err := h.service.CreateSCIMToken(withActorIP(r), input.DisplayName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *EnterpriseHandler) deleteSCIMToken(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/administration/enterprise/scim-tokens/"), "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scim token id is required"})
		return
	}
	if err := h.service.DeleteSCIMToken(withActorIP(r), id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

func (h *EnterpriseHandler) downloadSupportBundle(w http.ResponseWriter, _ *http.Request) {
	content, filename, err := h.service.BuildSupportBundle()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
