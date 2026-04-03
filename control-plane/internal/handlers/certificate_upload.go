package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"

	"waf/control-plane/internal/services"
)

const maxCertificateUploadMemory = 10 << 20

type certificateUploadService interface {
	Upload(ctx context.Context, req services.CertificateUploadRequest, certificatePEM []byte, privateKeyPEM []byte) (services.CertificateUploadResult, error)
}

type CertificateUploadHandler struct {
	upload certificateUploadService
}

func NewCertificateUploadHandler(upload certificateUploadService) *CertificateUploadHandler {
	return &CertificateUploadHandler{upload: upload}
}

func (h *CertificateUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/certificate-materials/upload" || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(maxCertificateUploadMemory); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid multipart form"})
		return
	}

	certificatePEM, err := readUploadedFile(r, "certificate_file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	privateKeyPEM, err := readUploadedFile(r, "private_key_file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	result, err := h.upload.Upload(withActorIP(r), services.CertificateUploadRequest{
		CertificateID: r.FormValue("certificate_id"),
		CommonName:    r.FormValue("common_name"),
		SANList:       readSANList(r),
		NotBefore:     r.FormValue("not_before"),
		NotAfter:      r.FormValue("not_after"),
		Status:        r.FormValue("status"),
	}, certificatePEM, privateKeyPEM)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func readUploadedFile(r *http.Request, field string) ([]byte, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, http.ErrMissingFile
	}
	return content, nil
}

func readSANList(r *http.Request) []string {
	values := r.MultipartForm.Value["san_list"]
	if len(values) == 0 {
		return nil
	}

	sans := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			sans = append(sans, part)
		}
	}
	return sans
}
