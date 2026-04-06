package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.URL.Path {
	case "/api/certificate-materials/upload":
		h.uploadSingle(w, r)
	case "/api/certificate-materials/import-archive":
		h.importArchive(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *CertificateUploadHandler) uploadSingle(w http.ResponseWriter, r *http.Request) {
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

func (h *CertificateUploadHandler) importArchive(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxCertificateUploadMemory); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid multipart form"})
		return
	}

	archive, err := readUploadedFile(r, "archive_file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	entries, err := parseCertificateArchive(archive)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(entries) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "archive does not contain valid certificate folders"})
		return
	}

	ids := make([]string, 0, len(entries))
	for certificateID := range entries {
		ids = append(ids, certificateID)
	}
	sort.Strings(ids)

	imported := make([]services.CertificateUploadResult, 0, len(ids))
	for _, certificateID := range ids {
		pair := entries[certificateID]
		result, uploadErr := h.upload.Upload(withActorIP(r), services.CertificateUploadRequest{
			CertificateID: certificateID,
			CommonName:    certificateID,
			Status:        "inactive",
		}, pair.certificatePEM, pair.privateKeyPEM)
		if uploadErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": uploadErr.Error()})
			return
		}
		imported = append(imported, result)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"imported_count": len(imported),
		"items":          imported,
	})
}

type archiveCertificatePair struct {
	certificatePEM []byte
	privateKeyPEM  []byte
}

func parseCertificateArchive(content []byte) (map[string]archiveCertificatePair, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive")
	}

	pairs := map[string]archiveCertificatePair{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		clean := strings.TrimSpace(strings.ReplaceAll(file.Name, "\\", "/"))
		clean = path.Clean("/" + clean)
		clean = strings.TrimPrefix(clean, "/")
		parts := strings.Split(clean, "/")
		if len(parts) != 2 {
			continue
		}
		folder := strings.ToLower(strings.TrimSpace(parts[0]))
		name := strings.ToLower(strings.TrimSpace(parts[1]))
		if folder == "" {
			continue
		}
		if name != "certificate.pem" && name != "private.key" {
			continue
		}
		entryReader, openErr := file.Open()
		if openErr != nil {
			return nil, openErr
		}
		data, readErr := io.ReadAll(entryReader)
		_ = entryReader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(data) == 0 {
			continue
		}
		pair := pairs[folder]
		if name == "certificate.pem" {
			pair.certificatePEM = data
		} else {
			pair.privateKeyPEM = data
		}
		pairs[folder] = pair
	}

	valid := map[string]archiveCertificatePair{}
	for certificateID, pair := range pairs {
		if len(pair.certificatePEM) == 0 || len(pair.privateKeyPEM) == 0 {
			continue
		}
		valid[certificateID] = pair
	}
	return valid, nil
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
