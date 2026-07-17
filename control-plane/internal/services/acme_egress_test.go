package services

import "testing"

func TestACMECustomDirectoryRequiresDeploymentAllowlist(t *testing.T) {
	client := &ACMELetsEncryptClient{cfg: ACMEClientConfig{CustomDirectoryURLs: normalizeACMEDirectoryURLs([]string{"https://acme.example.test/directory"})}}
	if _, err := client.resolveIssueOptions(&ACMEIssueOptions{CertificateAuthorityServer: "custom", CustomDirectoryURL: "https://127.0.0.1/directory", AccountEmail: "ops@example.test"}); err == nil {
		t.Fatal("expected unapproved custom directory to be rejected")
	}
	options, err := client.resolveIssueOptions(&ACMEIssueOptions{CertificateAuthorityServer: "custom", CustomDirectoryURL: "https://acme.example.test/directory", AccountEmail: "ops@example.test"})
	if err != nil || options.DirectoryURL != "https://acme.example.test/directory" {
		t.Fatalf("expected approved custom directory to remain usable, options=%+v err=%v", options, err)
	}
}

func TestACMEDirectoryURLRejectsUnsafeDestinations(t *testing.T) {
	for _, value := range []string{"http://acme.example.test/directory", "https://acme.example.test:8443/directory", "https://user:pass@acme.example.test/directory"} {
		if _, err := validateACMEDirectoryURL(value); err == nil {
			t.Fatalf("expected unsafe URL %q to be rejected", value)
		}
	}
}
