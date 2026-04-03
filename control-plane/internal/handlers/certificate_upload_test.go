package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/services"
)

type fakeCertificateUploadService struct{}

func (f *fakeCertificateUploadService) Upload(ctx context.Context, req services.CertificateUploadRequest, certificatePEM []byte, privateKeyPEM []byte) (services.CertificateUploadResult, error) {
	return services.CertificateUploadResult{
		Certificate: certificates.Certificate{
			ID:         req.CertificateID,
			CommonName: req.CommonName,
			Status:     "active",
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

	NewCertificateUploadHandler(&fakeCertificateUploadService{}).ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
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
