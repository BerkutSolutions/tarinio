package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/certificates"
)

type certificateService interface {
	Create(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error)
	List() ([]certificates.Certificate, error)
	Update(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error)
	Delete(ctx context.Context, id string) error
}

type CertificatesHandler struct {
	certificates certificateService
}

func NewCertificatesHandler(certificates certificateService) *CertificatesHandler {
	return &CertificatesHandler{certificates: certificates}
}

func (h *CertificatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/certificates" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/certificates" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/certificates/") && r.Method == http.MethodPut:
		h.update(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/certificates/") && r.Method == http.MethodDelete:
		h.delete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *CertificatesHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.certificates.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "certificate store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *CertificatesHandler) create(w http.ResponseWriter, r *http.Request) {
	var item certificates.Certificate
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	created, err := h.certificates.Create(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CertificatesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/certificates/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate id is required"})
		return
	}

	var item certificates.Certificate
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item.ID = id
	updated, err := h.certificates.Update(withActorIP(r), item)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *CertificatesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/certificates/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate id is required"})
		return
	}
	if err := h.certificates.Delete(withActorIP(r), id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
