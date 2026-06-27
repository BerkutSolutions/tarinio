package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/virtualpatches"
)

type virtualPatchService interface {
	Create(ctx context.Context, patch virtualpatches.VirtualPatch) (virtualpatches.VirtualPatch, error)
	List(ctx context.Context, siteID string) ([]virtualpatches.VirtualPatch, error)
	ListActive(ctx context.Context, siteID string) ([]virtualpatches.VirtualPatch, error)
	Delete(ctx context.Context, id string) error
}

// VirtualPatchesHandler handles CRUD for virtual patches scoped to a site.
// Routes:
//
//	POST   /api/sites/:siteID/virtual-patches
//	GET    /api/sites/:siteID/virtual-patches
//	DELETE /api/sites/:siteID/virtual-patches/:patchID
type VirtualPatchesHandler struct {
	svc virtualPatchService
}

func NewVirtualPatchesHandler(svc virtualPatchService) *VirtualPatchesHandler {
	return &VirtualPatchesHandler{svc: svc}
}

func (h *VirtualPatchesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Routes:
	//   GET    /api/virtual-patches/{siteID}
	//   POST   /api/virtual-patches/{siteID}
	//   DELETE /api/virtual-patches/{siteID}/{patchID}
	path := strings.TrimSuffix(r.URL.Path, "/")
	const prefix = "/api/virtual-patches/"
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(rest, "/", 2)

	siteID := parts[0]
	patchID := ""
	if len(parts) == 2 {
		patchID = parts[1]
	}

	if strings.TrimSpace(siteID) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch {
	case r.Method == http.MethodGet && patchID == "":
		h.list(w, r, siteID)
	case r.Method == http.MethodPost && patchID == "":
		h.create(w, r, siteID)
	case r.Method == http.MethodDelete && patchID != "":
		h.delete(w, r, patchID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *VirtualPatchesHandler) list(w http.ResponseWriter, r *http.Request, siteID string) {
	items, err := h.svc.List(r.Context(), siteID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *VirtualPatchesHandler) create(w http.ResponseWriter, r *http.Request, siteID string) {
	var patch virtualpatches.VirtualPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	patch.SiteID = siteID
	created, err := h.svc.Create(r.Context(), patch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *VirtualPatchesHandler) delete(w http.ResponseWriter, r *http.Request, patchID string) {
	if err := h.svc.Delete(r.Context(), patchID); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
