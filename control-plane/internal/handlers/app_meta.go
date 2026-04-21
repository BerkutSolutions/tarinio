package handlers

import (
	"net/http"
	"strings"

	"waf/control-plane/internal/appmeta"
)

type AppMetaHandler struct {
	haEnabled bool
	haNodeID  string
}

func NewAppMetaHandler(haEnabled bool, haNodeID string) *AppMetaHandler {
	return &AppMetaHandler{
		haEnabled: haEnabled,
		haNodeID:  strings.TrimSpace(haNodeID),
	}
}

func (h *AppMetaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	version := strings.TrimSpace(appmeta.AppVersion)
	if version == "" {
		version = "-"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app_version":     version,
		"product_name":    appmeta.ProductName,
		"repository_url":  appmeta.RepositoryURL,
		"github_releases": appmeta.GitHubAPIReleases,
		"ha_enabled":      h.haEnabled,
		"ha_node_id":      h.haNodeID,
	})
}
