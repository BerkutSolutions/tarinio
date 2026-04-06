package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/http"
	"sort"
	"strings"

	"waf/control-plane/internal/certificatematerials"
)

type certificateMaterialReader interface {
	Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error)
}

type CertificateMaterialExportHandler struct {
	reader certificateMaterialReader
}

func NewCertificateMaterialExportHandler(reader certificateMaterialReader) *CertificateMaterialExportHandler {
	return &CertificateMaterialExportHandler{reader: reader}
}

func (h *CertificateMaterialExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.reader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "certificate material store unavailable"})
		return
	}

	switch {
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

func (h *CertificateMaterialExportHandler) buildArchive(certificateIDs []string) ([]byte, error) {
	buf := &bytes.Buffer{}
	zipWriter := zip.NewWriter(buf)

	for _, certificateID := range certificateIDs {
		record, certificatePEM, privateKeyPEM, err := h.reader.Read(certificateID)
		if err != nil {
			_ = zipWriter.Close()
			return nil, err
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
