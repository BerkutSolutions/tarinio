package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
)

type fakeCertificateMaterialStore struct {
	record certificatematerials.MaterialRecord
}

func (f *fakeCertificateMaterialStore) Put(certificateID string, certificatePEM []byte, privateKeyPEM []byte) (certificatematerials.MaterialRecord, error) {
	f.record = certificatematerials.MaterialRecord{
		CertificateID:  certificateID,
		CertificateRef: "certificate-materials/files/" + certificateID + "/certificate.pem",
		PrivateKeyRef:  "certificate-materials/files/" + certificateID + "/private.key",
	}
	return f.record, nil
}

func (f *fakeCertificateMaterialStore) Get(certificateID string) (certificatematerials.MaterialRecord, error) {
	return f.record, nil
}

type fakeCertificateStore struct {
	items []certificates.Certificate
}

func (f *fakeCertificateStore) Create(item certificates.Certificate) (certificates.Certificate, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeCertificateStore) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

func (f *fakeCertificateStore) Update(item certificates.Certificate) (certificates.Certificate, error) {
	return item, nil
}

func (f *fakeCertificateStore) Delete(id string) error {
	return nil
}

func TestCertificateUploadService_UploadToExistingCertificate(t *testing.T) {
	service := NewCertificateUploadService(
		&fakeCertificateStore{items: []certificates.Certificate{{ID: "cert-a", CommonName: "example.com", Status: "active"}}},
		&fakeCertificateMaterialStore{},
		nil,
	)

	result, err := service.Upload(context.Background(), CertificateUploadRequest{
		CertificateID: "cert-a",
	}, []byte("CERT"), []byte("KEY"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if result.Certificate.ID != "cert-a" || result.Material.CertificateID != "cert-a" {
		t.Fatalf("unexpected upload result: %+v", result)
	}
}

func TestCertificateUploadService_CreatesCertificateWhenMetadataProvided(t *testing.T) {
	store := &fakeCertificateStore{}
	service := NewCertificateUploadService(store, &fakeCertificateMaterialStore{}, nil)

	result, err := service.Upload(context.Background(), CertificateUploadRequest{
		CertificateID: "cert-a",
		CommonName:    "example.com",
		Status:        "active",
	}, []byte("CERT"), []byte("KEY"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if len(store.items) != 1 || result.Certificate.CommonName != "example.com" {
		t.Fatalf("expected controlled certificate creation, got %+v", result)
	}
}

func TestCertificateUploadService_RejectsUnknownCertificateWithoutMetadata(t *testing.T) {
	service := NewCertificateUploadService(&fakeCertificateStore{}, &fakeCertificateMaterialStore{}, nil)

	if _, err := service.Upload(context.Background(), CertificateUploadRequest{CertificateID: "cert-a"}, []byte("CERT"), []byte("KEY")); err == nil {
		t.Fatal("expected missing certificate error")
	}
}
