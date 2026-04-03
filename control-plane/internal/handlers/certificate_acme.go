package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/jobs"
)

type certificateACMEService interface {
	Issue(ctx context.Context, certificateID string, commonName string, sanList []string) (jobs.Job, error)
	Renew(ctx context.Context, certificateID string) (jobs.Job, error)
}

type certificateIssueRequest struct {
	CertificateID string   `json:"certificate_id"`
	CommonName    string   `json:"common_name"`
	SANList       []string `json:"san_list"`
}

type CertificateACMEHandler struct {
	acme certificateACMEService
}

func NewCertificateACMEHandler(acme certificateACMEService) *CertificateACMEHandler {
	return &CertificateACMEHandler{acme: acme}
}

func (h *CertificateACMEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/certificates/acme/issue" && r.Method == http.MethodPost:
		h.issue(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/certificates/acme/renew/") && r.Method == http.MethodPost:
		h.renew(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *CertificateACMEHandler) issue(w http.ResponseWriter, r *http.Request) {
	var req certificateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	job, err := h.acme.Issue(withActorIP(r), req.CertificateID, req.CommonName, req.SANList)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (h *CertificateACMEHandler) renew(w http.ResponseWriter, r *http.Request) {
	certificateID := strings.TrimPrefix(r.URL.Path, "/api/certificates/acme/renew/")
	if strings.TrimSpace(certificateID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate id is required"})
		return
	}
	job, err := h.acme.Renew(withActorIP(r), certificateID)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}
