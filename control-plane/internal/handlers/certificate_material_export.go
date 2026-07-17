package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/certificateexportapprovals"
	"waf/control-plane/internal/certificatematerials"
)

const maxCertificateExportItems = 100
const maxCertificateExportBytes = 32 << 20

type certificateMaterialReader interface {
	Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error)
}

type certificateExportApprovalStore interface {
	Request(id, requesterID string, certificateIDs []string, ttl time.Duration) (certificateexportapprovals.Approval, error)
	Approve(id, actorID string) (certificateexportapprovals.Approval, error)
	Consume(id, requesterID string, certificateIDs []string) error
}

type certificateExportStepUpVerifier interface {
	RequireStepUp(sessionID string) error
}

type CertificateMaterialExportHandler struct {
	reader           certificateMaterialReader
	approvals        certificateExportApprovalStore
	stepUp           certificateExportStepUpVerifier
	approvalRequired func() bool
}

func (h *CertificateMaterialExportHandler) SetApprovalRequiredProvider(provider func() bool) {
	if h != nil {
		h.approvalRequired = provider
	}
}

func (h *CertificateMaterialExportHandler) SetStepUpVerifier(verifier certificateExportStepUpVerifier) {
	if h != nil {
		h.stepUp = verifier
	}
}

func NewCertificateMaterialExportHandler(reader certificateMaterialReader, approvals ...certificateExportApprovalStore) *CertificateMaterialExportHandler {
	h := &CertificateMaterialExportHandler{reader: reader}
	if len(approvals) > 0 {
		h.approvals = approvals[0]
	}
	return h
}

func (h *CertificateMaterialExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.reader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "certificate material store unavailable"})
		return
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/certificate-materials/export-approvals":
		h.requestApproval(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/certificate-materials/export-approvals/") && strings.HasSuffix(r.URL.Path, "/approve"):
		h.approve(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/certificate-materials/export/"):
		h.exportSingle(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/certificate-materials/export":
		h.exportArchive(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *CertificateMaterialExportHandler) exportSingle(w http.ResponseWriter, r *http.Request) {
	certificateID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/certificate-materials/export/"))
	if certificateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate id is required"})
		return
	}
	if !h.consumeApproval(w, r, r.URL.Query().Get("approval_id"), []string{certificateID}) {
		return
	}

	record, certificatePEM, privateKeyPEM, err := h.reader.Read(certificateID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"certificate_id":  record.CertificateID,
		"certificate_ref": record.CertificateRef,
		"private_key_ref": record.PrivateKeyRef,
		"updated_at":      record.UpdatedAt,
		"certificate_pem": string(certificatePEM),
		"private_key_pem": string(privateKeyPEM),
	})
}

func (h *CertificateMaterialExportHandler) exportArchive(w http.ResponseWriter, r *http.Request) {
	body, ok := readJSONBody(w, r)
	if !ok {
		return
	}
	rawIDs, exists := body["certificate_ids"]
	if !exists {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate_ids is required"})
		return
	}
	ids, valid := asStringList(rawIDs)
	if !valid || len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate_ids must be a non-empty array"})
		return
	}

	normalized := normalizeCertificateIDs(ids)
	if len(normalized) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate_ids must contain valid values"})
		return
	}
	if len(normalized) > maxCertificateExportItems {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "too many certificate ids"})
		return
	}
	approvalID, _ := body["approval_id"].(string)
	if !h.consumeApproval(w, r, approvalID, normalized) {
		return
	}

	archiveBytes, err := h.buildArchive(normalized)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="certificate-materials.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archiveBytes)
}

func (h *CertificateMaterialExportHandler) requestApproval(w http.ResponseWriter, r *http.Request) {
	if h.approvals == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "certificate export approval service unavailable"})
		return
	}
	actorID, ok := certificateExportActorID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authenticated session is required"})
		return
	}
	body, ok := readJSONBody(w, r)
	if !ok {
		return
	}
	ids, valid := asStringList(body["certificate_ids"])
	if !valid {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "certificate_ids must be a non-empty array"})
		return
	}
	ids = normalizeCertificateIDs(ids)
	if len(ids) == 0 || len(ids) > maxCertificateExportItems {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid certificate_ids"})
		return
	}
	id, err := newCertificateExportApprovalID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "create approval id"})
		return
	}
	approval, err := h.approvals.Request(id, actorID, ids, 10*time.Minute)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, approval)
}

func (h *CertificateMaterialExportHandler) approve(w http.ResponseWriter, r *http.Request) {
	if h.approvals == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "certificate export approval service unavailable"})
		return
	}
	actorID, ok := certificateExportActorID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authenticated session is required"})
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/certificate-materials/export-approvals/"), "/approve")
	approval, err := h.approvals.Approve(id, actorID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

func (h *CertificateMaterialExportHandler) consumeApproval(w http.ResponseWriter, r *http.Request, approvalID string, certificateIDs []string) bool {
	if h.stepUp != nil {
		session, ok := auth.SessionFromContext(r.Context())
		if !ok || strings.TrimSpace(session.SessionID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authenticated session is required"})
			return false
		}
		if err := h.stepUp.RequireStepUp(session.SessionID); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": err.Error()})
			return false
		}
	}
	if h.approvals == nil || (h.approvalRequired != nil && !h.approvalRequired()) {
		return true
	}
	actorID, ok := certificateExportActorID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authenticated session is required"})
		return false
	}
	if err := h.approvals.Consume(strings.TrimSpace(approvalID), actorID, certificateIDs); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": err.Error()})
		return false
	}
	return true
}

func certificateExportActorID(r *http.Request) (string, bool) {
	session, ok := auth.SessionFromContext(r.Context())
	return strings.TrimSpace(session.UserID), ok && strings.TrimSpace(session.UserID) != ""
}
func newCertificateExportApprovalID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "cea-" + hex.EncodeToString(raw), nil
}

func (h *CertificateMaterialExportHandler) buildArchive(certificateIDs []string) ([]byte, error) {
	buf := &bytes.Buffer{}
	zipWriter := zip.NewWriter(buf)

	for _, certificateID := range certificateIDs {
		record, certificatePEM, privateKeyPEM, err := h.reader.Read(certificateID)
		if err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
		if len(certificatePEM)+len(privateKeyPEM) > maxCertificateExportBytes || buf.Len()+len(certificatePEM)+len(privateKeyPEM) > maxCertificateExportBytes {
			_ = zipWriter.Close()
			return nil, fmt.Errorf("certificate archive exceeds size limit")
		}
		normalizedID := strings.ToLower(strings.TrimSpace(record.CertificateID))
		if normalizedID == "" {
			normalizedID = strings.ToLower(strings.TrimSpace(certificateID))
		}
		if err := addZipFile(zipWriter, normalizedID+"/certificate.pem", certificatePEM); err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
		if err := addZipFile(zipWriter, normalizedID+"/private.key", privateKeyPEM); err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addZipFile(writer *zip.Writer, name string, content []byte) error {
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = entry.Write(content)
	return err
}

func normalizeCertificateIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func asStringList(value any) ([]string, bool) {
	array, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(array))
	for _, item := range array {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}
