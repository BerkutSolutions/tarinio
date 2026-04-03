package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
)

type fakeTLSConfigStore struct {
	items []tlsconfigs.TLSConfig
}

func (f *fakeTLSConfigStore) Create(item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeTLSConfigStore) List() ([]tlsconfigs.TLSConfig, error) {
	return append([]tlsconfigs.TLSConfig(nil), f.items...), nil
}

func (f *fakeTLSConfigStore) Update(item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	return item, nil
}

func (f *fakeTLSConfigStore) Delete(siteID string) error {
	return nil
}

type fakeSiteReader struct {
	items []sites.Site
}

func (f *fakeSiteReader) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), f.items...), nil
}

type fakeCertificateReader struct {
	items []certificates.Certificate
}

func (f *fakeCertificateReader) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

func TestTLSConfigService_CreateValidatesSiteAndCertificate(t *testing.T) {
	service := NewTLSConfigService(
		&fakeTLSConfigStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		&fakeCertificateReader{items: []certificates.Certificate{{ID: "cert-a"}}},
		nil,
	)

	created, err := service.Create(context.Background(), tlsconfigs.TLSConfig{
		SiteID:        "site-a",
		CertificateID: "cert-a",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.SiteID != "site-a" || created.CertificateID != "cert-a" {
		t.Fatalf("unexpected tls config: %+v", created)
	}
}

func TestTLSConfigService_CreateRejectsMissingSite(t *testing.T) {
	service := NewTLSConfigService(
		&fakeTLSConfigStore{},
		&fakeSiteReader{},
		&fakeCertificateReader{items: []certificates.Certificate{{ID: "cert-a"}}},
		nil,
	)

	if _, err := service.Create(context.Background(), tlsconfigs.TLSConfig{SiteID: "site-a", CertificateID: "cert-a"}); err == nil {
		t.Fatal("expected missing site error")
	}
}

func TestTLSConfigService_CreateRejectsMissingCertificate(t *testing.T) {
	service := NewTLSConfigService(
		&fakeTLSConfigStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		&fakeCertificateReader{},
		nil,
	)

	if _, err := service.Create(context.Background(), tlsconfigs.TLSConfig{SiteID: "site-a", CertificateID: "cert-a"}); err == nil {
		t.Fatal("expected missing certificate error")
	}
}
