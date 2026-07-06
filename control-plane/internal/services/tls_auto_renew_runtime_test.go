package services

import (
	"context"
	"testing"
	"time"

	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/tlsconfigs"
)

type fakeTLSAutoRenewCertificates struct {
	items []certificates.Certificate
}

func (f *fakeTLSAutoRenewCertificates) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

type fakeTLSAutoRenewConfigs struct {
	items []tlsconfigs.TLSConfig
}

func (f *fakeTLSAutoRenewConfigs) List() ([]tlsconfigs.TLSConfig, error) {
	return append([]tlsconfigs.TLSConfig(nil), f.items...), nil
}

type fakeTLSAutoRenewRenewer struct {
	certificateID string
	options       *ACMEIssueOptions
	calls         int
}

func (f *fakeTLSAutoRenewRenewer) Renew(_ context.Context, certificateID string, options *ACMEIssueOptions) (jobs.Job, error) {
	f.certificateID = certificateID
	f.options = options
	f.calls++
	return jobs.Job{ID: "renew-job", Status: jobs.StatusSucceeded}, nil
}

type fakeTLSAutoRenewProfiles struct {
	items map[string]easysiteprofiles.EasySiteProfile
}

func (f *fakeTLSAutoRenewProfiles) Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error) {
	item, ok := f.items[siteID]
	return item, ok, nil
}

func TestTLSAutoRenewService_UsesEasySiteProfileOptions(t *testing.T) {
	renewer := &fakeTLSAutoRenewRenewer{}
	service, err := NewTLSAutoRenewService(t.TempDir(), &fakeTLSAutoRenewCertificates{
		items: []certificates.Certificate{{
			ID:         "cert-a",
			CommonName: "ui.example.test",
			Status:     "active",
			NotAfter:   time.Now().UTC().Add(20 * 24 * time.Hour).Format(time.RFC3339),
		}},
	}, &fakeTLSAutoRenewConfigs{
		items: []tlsconfigs.TLSConfig{{
			SiteID:        "ui.example.test",
			CertificateID: "cert-a",
		}},
	}, renewer)
	if err != nil {
		t.Fatalf("service init failed: %v", err)
	}
	service.SetEasySiteProfileReader(&fakeTLSAutoRenewProfiles{
		items: map[string]easysiteprofiles.EasySiteProfile{
			"ui.example.test": {
				SiteID: "ui.example.test",
				FrontService: easysiteprofiles.FrontServiceSettings{
					AutoLetsEncrypt:            true,
					CertificateAuthorityServer: "letsencrypt",
					ACMEAccountEmail:           "renewals@example.test",
					UseLetsEncryptStaging:      true,
				},
			},
		},
	})
	service.runOnceUnlocked(context.Background())
	if renewer.calls != 1 {
		t.Fatalf("expected renewer to be called once, got %d", renewer.calls)
	}
	if renewer.certificateID != "cert-a" {
		t.Fatalf("expected renewer to receive cert-a, got %q", renewer.certificateID)
	}
	if renewer.options == nil {
		t.Fatal("expected renew options from easy profile")
	}
	if renewer.options.AccountEmail != "renewals@example.test" {
		t.Fatalf("expected profile email to be passed, got %#v", renewer.options)
	}
	if renewer.options.CertificateAuthorityServer != "letsencrypt" || !renewer.options.UseLetsEncryptStaging {
		t.Fatalf("expected CA options to be passed, got %#v", renewer.options)
	}
}
