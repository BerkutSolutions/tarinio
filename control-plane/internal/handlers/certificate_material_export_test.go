package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/certificatematerials"
)

type fakeCertificateMaterialReader struct {
	record         certificatematerials.MaterialRecord
	certificatePEM []byte
	privateKeyPEM  []byte
	err            error
}

func (f *fakeCertificateMaterialReader) Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error) {
	if f.err != nil {
		return certificatematerials.MaterialRecord{}, nil, nil, f.err
	}
	record := f.record
	if strings.TrimSpace(record.CertificateID) == "" {
		record.CertificateID = certificateID
	}
	return record, f.certificatePEM, f.privateKeyPEM, nil
}

func TestCertificateMaterialExportHandler_Export(t *testing.T) {
	h := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{
		record: certificatematerials.MaterialRecord{
			CertificateID: "cert-a",
		},
		certificatePEM: []byte("CERT"),
		privateKeyPEM:  []byte("KEY"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/certificate-materials/export/cert-a", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestCertificateMaterialExportHandler_NotFound(t *testing.T) {
	h := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{err: errors.New("certificate material cert-a not found")})

	req := httptest.NewRequest(http.MethodGet, "/api/certificate-materials/export/cert-a", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func TestCertificateMaterialExportHandler_Archive(t *testing.T) {
	h := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{
		record:         certificatematerials.MaterialRecord{CertificateID: "cert-a"},
		certificatePEM: []byte("CERT"),
		privateKeyPEM:  []byte("KEY"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/export", bytes.NewBufferString(`{"certificate_ids":["cert-a"]}`))
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if ct := resp.Header().Get("Content-Type"); !strings.Contains(ct, "application/zip") {
		t.Fatalf("expected zip content type, got %q", ct)
	}
	zr, err := zip.NewReader(bytes.NewReader(resp.Body.Bytes()), int64(resp.Body.Len()))
	if err != nil {
		t.Fatalf("invalid zip response: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("expected 2 files in archive, got %d", len(zr.File))
	}
}

func TestCertificateMaterialExportHandler_ArchiveEmptyList(t *testing.T) {
	h := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{})

	req := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/export", bytes.NewBufferString(`{"certificate_ids":[]}`))
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
