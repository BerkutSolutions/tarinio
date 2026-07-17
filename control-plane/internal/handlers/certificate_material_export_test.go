package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/certificateexportapprovals"
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

func TestCertificateMaterialExportHandler_RequiresOneTimeDistinctApproval(t *testing.T) {
	approvals, err := certificateexportapprovals.NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{record: certificatematerials.MaterialRecord{CertificateID: "cert-a"}, certificatePEM: []byte("CERT"), privateKeyPEM: []byte("KEY")}, approvals)

	request := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/export-approvals", bytes.NewBufferString(`{"certificate_ids":["cert-a"]}`))
	request = request.WithContext(auth.ContextWithSession(request.Context(), auth.SessionView{UserID: "requester"}))
	requestResponse := httptest.NewRecorder()
	handler.ServeHTTP(requestResponse, request)
	if requestResponse.Code != http.StatusCreated {
		t.Fatalf("request approval: %d %s", requestResponse.Code, requestResponse.Body.String())
	}
	var approval certificateexportapprovals.Approval
	if err := json.Unmarshal(requestResponse.Body.Bytes(), &approval); err != nil {
		t.Fatal(err)
	}

	approve := httptest.NewRequest(http.MethodPost, "/api/certificate-materials/export-approvals/"+approval.ID+"/approve", nil)
	approve = approve.WithContext(auth.ContextWithSession(approve.Context(), auth.SessionView{UserID: "reviewer"}))
	approveResponse := httptest.NewRecorder()
	handler.ServeHTTP(approveResponse, approve)
	if approveResponse.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", approveResponse.Code, approveResponse.Body.String())
	}

	export := httptest.NewRequest(http.MethodGet, "/api/certificate-materials/export/cert-a?approval_id="+approval.ID, nil)
	export = export.WithContext(auth.ContextWithSession(export.Context(), auth.SessionView{UserID: "requester"}))
	exportResponse := httptest.NewRecorder()
	handler.ServeHTTP(exportResponse, export)
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("export: %d %s", exportResponse.Code, exportResponse.Body.String())
	}

	replay := httptest.NewRequest(http.MethodGet, "/api/certificate-materials/export/cert-a?approval_id="+approval.ID, nil)
	replay = replay.WithContext(auth.ContextWithSession(replay.Context(), auth.SessionView{UserID: "requester"}))
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusForbidden {
		t.Fatalf("expected replay denial, got %d %s", replayResponse.Code, replayResponse.Body.String())
	}
}

func TestCertificateMaterialExportHandler_AllowsExportWhenApprovalPolicyDisabled(t *testing.T) {
	approvals, err := certificateexportapprovals.NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewCertificateMaterialExportHandler(&fakeCertificateMaterialReader{record: certificatematerials.MaterialRecord{CertificateID: "cert-a"}, certificatePEM: []byte("CERT"), privateKeyPEM: []byte("KEY")}, approvals)
	handler.SetApprovalRequiredProvider(func() bool { return false })

	request := httptest.NewRequest(http.MethodGet, "/api/certificate-materials/export/cert-a", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected export without approval when policy is disabled, got %d %s", response.Code, response.Body.String())
	}
}
