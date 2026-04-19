package handlers

import (
	"context"
	"net/http"

	"waf/control-plane/internal/services"
)

type revisionCatalogService interface {
	List(ctx context.Context) (services.RevisionCatalogResponse, error)
}

type RevisionCatalogHandler struct {
	service revisionCatalogService
}

func NewRevisionCatalogHandler(service revisionCatalogService) *RevisionCatalogHandler {
	return &RevisionCatalogHandler{service: service}
}

func (h *RevisionCatalogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/revisions" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	payload, err := h.service.List(withActorIP(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
