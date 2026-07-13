package services

import (
	"context"
	"strings"
	"testing"

	"waf/control-plane/internal/audits"
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
	for index := range f.items {
		if f.items[index].SiteID == item.SiteID {
			f.items[index] = item
			return item, nil
		}
	}
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

type tlsAuditStore struct{ items []audits.AuditEvent }

func (s *tlsAuditStore) Append(item audits.AuditEvent) error {
	s.items = append(s.items, item)
	return nil
}
func (s *tlsAuditStore) List(audits.Query) (audits.ListResult, error) {
	return audits.ListResult{}, nil
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

func TestTLSConfigService_CreateRejectsCertificateHostMismatch(t *testing.T) {
	service := NewTLSConfigService(&fakeTLSConfigStore{}, &fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "app.example"}}}, &fakeCertificateReader{items: []certificates.Certificate{{ID: "cert-a", CommonName: "other.example"}}}, nil)
	if _, err := service.Create(context.Background(), tlsconfigs.TLSConfig{SiteID: "site-a", CertificateID: "cert-a"}); err == nil {
		t.Fatal("expected certificate-host mismatch")
	} else if !strings.Contains(err.Error(), "certificate_host_mismatch") {
		t.Fatalf("expected stable mismatch code, got %v", err)
	}
}

func TestTLSConfigService_CreateAcceptsWildcardCertificate(t *testing.T) {
	service := NewTLSConfigService(&fakeTLSConfigStore{}, &fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "app.example"}}}, &fakeCertificateReader{items: []certificates.Certificate{{ID: "cert-a", CommonName: "*.example"}}}, nil)
	if _, err := service.Create(context.Background(), tlsconfigs.TLSConfig{SiteID: "site-a", CertificateID: "cert-a"}); err != nil {
		t.Fatalf("expected wildcard certificate to match: %v", err)
	}
}

func TestTLSConfigService_UpdateEmitsRebindAudit(t *testing.T) {
	auditStore := &tlsAuditStore{}
	service := NewTLSConfigService(
		&fakeTLSConfigStore{items: []tlsconfigs.TLSConfig{{SiteID: "site-a", CertificateID: "cert-old"}}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "app.example"}}},
		&fakeCertificateReader{items: []certificates.Certificate{{ID: "cert-old", CommonName: "app.example"}, {ID: "cert-new", CommonName: "app.example"}}},
		NewAuditService(auditStore),
	)
	if _, err := service.Update(context.Background(), tlsconfigs.TLSConfig{SiteID: "site-a", CertificateID: "cert-new"}); err != nil {
		t.Fatal(err)
	}
	if len(auditStore.items) != 1 || auditStore.items[0].Action != "tlsconfig.rebind" {
		t.Fatalf("unexpected audit events: %+v", auditStore.items)
	}
	if auditStore.items[0].Details["previous_certificate_id"] != "cert-old" {
		t.Fatalf("missing previous binding: %+v", auditStore.items[0])
	}
}
