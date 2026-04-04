package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/jobs"
)

type fakeLetsEncryptClient struct{}

func (f *fakeLetsEncryptClient) Issue(commonName string, sanList []string, options *ACMEIssueOptions) (IssuedMaterial, error) {
	return IssuedMaterial{
		CertificatePEM: []byte("CERT"),
		PrivateKeyPEM:  []byte("KEY"),
		NotBefore:      "2026-04-01T00:00:00Z",
		NotAfter:       "2026-10-01T00:00:00Z",
	}, nil
}

func (f *fakeLetsEncryptClient) Renew(commonName string, sanList []string, options *ACMEIssueOptions) (IssuedMaterial, error) {
	return IssuedMaterial{
		CertificatePEM: []byte("CERT-RENEW"),
		PrivateKeyPEM:  []byte("KEY-RENEW"),
		NotBefore:      "2026-05-01T00:00:00Z",
		NotAfter:       "2026-11-01T00:00:00Z",
	}, nil
}

type fakeLEMaterialStore struct {
	lastID string
}

func (f *fakeLEMaterialStore) Put(certificateID string, certificatePEM []byte, privateKeyPEM []byte) (certificatematerials.MaterialRecord, error) {
	f.lastID = certificateID
	return certificatematerials.MaterialRecord{CertificateID: certificateID}, nil
}

func (f *fakeLEMaterialStore) Get(certificateID string) (certificatematerials.MaterialRecord, error) {
	return certificatematerials.MaterialRecord{CertificateID: certificateID}, nil
}

type fakeCertificateRepo struct {
	items []certificates.Certificate
}

func (f *fakeCertificateRepo) Create(item certificates.Certificate) (certificates.Certificate, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeCertificateRepo) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

func (f *fakeCertificateRepo) Update(item certificates.Certificate) (certificates.Certificate, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	return item, nil
}

func (f *fakeCertificateRepo) Delete(id string) error {
	return nil
}

func TestLetsEncryptService_IssueCreatesJobAndCertificate(t *testing.T) {
	jobStore, err := jobs.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create job store failed: %v", err)
	}
	certStore := &fakeCertificateRepo{}
	materialStore := &fakeLEMaterialStore{}
	service := NewLetsEncryptService(&fakeLetsEncryptClient{}, jobStore, certStore, materialStore, nil)

	job, err := service.Issue(context.Background(), "cert-a", "example.com", []string{"www.example.com"}, nil)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if job.Status != jobs.StatusSucceeded {
		t.Fatalf("expected succeeded job, got %+v", job)
	}
	if len(certStore.items) != 1 || certStore.items[0].Status != "active" {
		t.Fatalf("expected active certificate, got %+v", certStore.items)
	}
	if materialStore.lastID != "cert-a" {
		t.Fatalf("expected material write for cert-a, got %s", materialStore.lastID)
	}
}

func TestLetsEncryptService_RenewUpdatesExistingCertificate(t *testing.T) {
	jobStore, err := jobs.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create job store failed: %v", err)
	}
	certStore := &fakeCertificateRepo{
		items: []certificates.Certificate{{
			ID:         "cert-a",
			CommonName: "example.com",
			SANList:    []string{"www.example.com"},
			Status:     "active",
		}},
	}
	materialStore := &fakeLEMaterialStore{}
	service := NewLetsEncryptService(&fakeLetsEncryptClient{}, jobStore, certStore, materialStore, nil)

	job, err := service.Renew(context.Background(), "cert-a", nil)
	if err != nil {
		t.Fatalf("renew failed: %v", err)
	}
	if job.Status != jobs.StatusSucceeded {
		t.Fatalf("expected succeeded job, got %+v", job)
	}
	if certStore.items[0].NotAfter != "2026-11-01T00:00:00Z" {
		t.Fatalf("expected renewed metadata, got %+v", certStore.items[0])
	}
}
