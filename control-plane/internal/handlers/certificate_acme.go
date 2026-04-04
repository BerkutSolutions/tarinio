package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/services"
)

type certificateACMEService interface {
	Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *services.ACMEIssueOptions) (jobs.Job, error)
	Renew(ctx context.Context, certificateID string, options *services.ACMEIssueOptions) (jobs.Job, error)
}

type certificateSelfSignedService interface {
	Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *services.ACMEIssueOptions) (jobs.Job, error)
}

type certificateIssueRequest struct {
	CertificateID              string            `json:"certificate_id"`
	CommonName                 string            `json:"common_name"`
	SANList                    []string          `json:"san_list"`
	CertificateAuthorityServer string            `json:"certificate_authority_server"`
	CustomDirectoryURL         string            `json:"custom_directory_url"`
	UseLetsEncryptStaging      bool              `json:"use_lets_encrypt_staging"`
	AccountEmail               string            `json:"account_email"`
	ChallengeType              string            `json:"challenge_type"`
	DNSProvider                string            `json:"dns_provider"`
	DNSProviderEnv             map[string]string `json:"dns_provider_env"`
	DNSResolvers               []string          `json:"dns_resolvers"`
	DNSPropagationSeconds      int               `json:"dns_propagation_seconds"`
	ZeroSSLEABKID              string            `json:"zerossl_eab_kid"`
	ZeroSSLEABHMACKey          string            `json:"zerossl_eab_hmac_key"`
}

type CertificateACMEHandler struct {
	acme       certificateACMEService
	selfSigned certificateSelfSignedService
}

func NewCertificateACMEHandler(acme certificateACMEService, selfSigned certificateSelfSignedService) *CertificateACMEHandler {
	return &CertificateACMEHandler{acme: acme, selfSigned: selfSigned}
}

func (h *CertificateACMEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/certificates/acme/issue" && r.Method == http.MethodPost:
		h.issue(w, r)
	case r.URL.Path == "/api/certificates/self-signed/issue" && r.Method == http.MethodPost:
		h.issueSelfSigned(w, r)
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
	options := &services.ACMEIssueOptions{
		CertificateAuthorityServer: req.CertificateAuthorityServer,
		CustomDirectoryURL:         req.CustomDirectoryURL,
		UseLetsEncryptStaging:      req.UseLetsEncryptStaging,
		AccountEmail:               req.AccountEmail,
		ChallengeType:              req.ChallengeType,
		DNSProvider:                req.DNSProvider,
		DNSProviderEnv:             req.DNSProviderEnv,
		DNSResolvers:               req.DNSResolvers,
		DNSPropagationSeconds:      req.DNSPropagationSeconds,
		ZeroSSLEABKID:              req.ZeroSSLEABKID,
		ZeroSSLEABHMACKey:          req.ZeroSSLEABHMACKey,
	}
	job, err := h.acme.Issue(withActorIP(r), req.CertificateID, req.CommonName, req.SANList, options)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if job.Status == jobs.StatusFailed {
		message := strings.TrimSpace(job.Result)
		if message == "" {
			message = "acme issue failed"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": message,
			"job":   job,
		})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (h *CertificateACMEHandler) issueSelfSigned(w http.ResponseWriter, r *http.Request) {
	var req certificateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	job, err := h.selfSigned.Issue(withActorIP(r), req.CertificateID, req.CommonName, req.SANList, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if job.Status == jobs.StatusFailed {
		message := strings.TrimSpace(job.Result)
		if message == "" {
			message = "self-signed issue failed"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": message,
			"job":   job,
		})
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
	job, err := h.acme.Renew(withActorIP(r), certificateID, nil)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	if job.Status == jobs.StatusFailed {
		message := strings.TrimSpace(job.Result)
		if message == "" {
			message = "acme renew failed"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": message,
			"job":   job,
		})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}
