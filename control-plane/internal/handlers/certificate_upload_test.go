package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/services"
)

type fakeCertificateUploadService struct {
	requests []services.CertificateUploadRequest
}

func (f *fakeCertificateUploadService) Upload(ctx context.Context, req services.CertificateUploadRequest, certificatePEM []byte, privateKeyPEM []byte) (services.CertificateUploadResult, error) {
	f.requests = append(f.requests, req)
	return services.CertificateUploadResult{
		Certificate: certificates.Certificate{
			ID:         req.CertificateID,
			CommonName: req.CommonName,
			Status:     req.Status,
		},
		Material: certificatematerials.MaterialRecord{
			CertificateID:  req.CertificateID,
			CertificateRef: "certificate-materials/files/" + req.CertificateID + "/certificate.pem",
			PrivateKeyRef:  "certificate-materials/files/" + req.CertificateID + "/private.key",
		},
	}, nil
}

func TestCertificateUploadHandler_Upload(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("certificate_id", "cert-a"); err != nil {
		t.Fatalf("write certificate_id failed: %v", err)
	}
	if err := writer.WriteField("common_name", "example.com"); err != nil {
		t.Fatalf("write common_name failed: %v", err)
	}
	if err := writer.WriteField("san_list", "www.example.com,api.example.com"); err != nil {
		t.Fatalf("write san_list failed: %v", err)
	}
	writeMultipartFile(t, writer, "certificate_file", "certificate.pem", "CERT")
	writeMultipartFile(t, writer, "private_key_file", "private.key", "KEY")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	service := &fakeCertificateUploadService{}
	NewCertificateUploadHandler(service).ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if len(service.requests) != 1 {
		t.Fatalf("expected one upload request, got %d", len(service.requests))
	}
}

func TestCertificateUploadHandler_ImportArchive(t *testing.T) {
	archive := &bytes.Buffer{}
	zipWriter := zip.NewWriter(archive)
	writeZipFile(t, zipWriter, "cert-a/certificate.pem", "CERT-A")
	writeZipFile(t, zipWriter, "cert-a/private.key", "KEY-A")
	writeZipFile(t, zipWriter, "cert-b/certificate.pem", "CERT-B")
	writeZipFile(t, zipWriter, "cert-b/private.key", "KEY-B")
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writeMultipartFile(t, writer, "archive_file", "certificate-materials.zip", archive.String())
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/import-archive", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	service := &fakeCertificateUploadService{}
	NewCertificateUploadHandler(service).ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if len(service.requests) != 2 {
		t.Fatalf("expected 2 imported certificates, got %d", len(service.requests))
	}
	gotIDs := []string{service.requests[0].CertificateID, service.requests[1].CertificateID}
	sort.Strings(gotIDs)
	if gotIDs[0] != "cert-a" || gotIDs[1] != "cert-b" {
		t.Fatalf("unexpected imported ids: %+v", gotIDs)
	}
	for _, item := range service.requests {
		if item.Status != "inactive" {
			t.Fatalf("expected inactive status, got %q", item.Status)
		}
	}
}

func writeMultipartFile(t *testing.T, writer *multipart.Writer, field string, name string, content string) {
	t.Helper()

	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(content)); err != nil {
		t.Fatalf("write form file failed: %v", err)
	}
}

func writeZipFile(t *testing.T, writer *zip.Writer, name string, content string) {
	t.Helper()

	part, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip file failed: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(content)); err != nil {
		t.Fatalf("write zip file failed: %v", err)
	}
}
