package compiler

import (
	"strings"
	"testing"
)

// --- ValidateMTLS ---

func TestValidateMTLS_Disabled(t *testing.T) {
	t.Parallel()
	if err := ValidateMTLS(MTLSInput{MTLSEnabled: false}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateMTLS_EnabledNoCA(t *testing.T) {
	t.Parallel()
	err := ValidateMTLS(MTLSInput{MTLSEnabled: true, MTLSClientCARef: ""})
	if err == nil {
		t.Fatal("expected error for missing CA ref")
	}
}

func TestValidateMTLS_EnabledWithCA(t *testing.T) {
	t.Parallel()
	err := ValidateMTLS(MTLSInput{MTLSEnabled: true, MTLSClientCARef: "/etc/ssl/ca.crt", MTLSVerifyDepth: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMTLS_NegativeDepth(t *testing.T) {
	t.Parallel()
	err := ValidateMTLS(MTLSInput{MTLSEnabled: true, MTLSClientCARef: "/etc/ssl/ca.crt", MTLSVerifyDepth: -1})
	if err == nil {
		t.Fatal("expected error for negative depth")
	}
}

// --- ValidateUpstreamMTLS ---

func TestValidateUpstreamMTLS_Disabled(t *testing.T) {
	t.Parallel()
	if err := ValidateUpstreamMTLS(UpstreamMTLSInput{UpstreamMTLSEnabled: false}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateUpstreamMTLS_EnabledNoCert(t *testing.T) {
	t.Parallel()
	err := ValidateUpstreamMTLS(UpstreamMTLSInput{UpstreamMTLSEnabled: true, UpstreamMTLSCertRef: "", UpstreamMTLSKeyRef: "/k.key"})
	if err == nil {
		t.Fatal("expected error for missing cert ref")
	}
}

func TestValidateUpstreamMTLS_EnabledNoKey(t *testing.T) {
	t.Parallel()
	err := ValidateUpstreamMTLS(UpstreamMTLSInput{UpstreamMTLSEnabled: true, UpstreamMTLSCertRef: "/c.crt", UpstreamMTLSKeyRef: ""})
	if err == nil {
		t.Fatal("expected error for missing key ref")
	}
}

// --- buildMTLSServerSnippet ---

func TestBuildMTLSServerSnippet_Disabled(t *testing.T) {
	t.Parallel()
	out := buildMTLSServerSnippet(MTLSInput{MTLSEnabled: false})
	if out != "" {
		t.Fatalf("expected empty snippet, got %q", out)
	}
}

func TestBuildMTLSServerSnippet_Required(t *testing.T) {
	t.Parallel()
	out := buildMTLSServerSnippet(MTLSInput{
		MTLSEnabled:     true,
		MTLSOptional:    false,
		MTLSVerifyDepth: 2,
		MTLSClientCARef: "/etc/ssl/ca.crt",
	})
	if !strings.Contains(out, "ssl_verify_client on") {
		t.Errorf("expected 'ssl_verify_client on', got: %s", out)
	}
	if !strings.Contains(out, "ssl_client_certificate /etc/ssl/ca.crt") {
		t.Errorf("expected ssl_client_certificate directive, got: %s", out)
	}
	if !strings.Contains(out, "ssl_verify_depth 2") {
		t.Errorf("expected ssl_verify_depth 2, got: %s", out)
	}
	if strings.Contains(out, "ssl_verify_client optional") {
		t.Errorf("should not contain optional, got: %s", out)
	}
}

func TestBuildMTLSServerSnippet_Optional(t *testing.T) {
	t.Parallel()
	out := buildMTLSServerSnippet(MTLSInput{
		MTLSEnabled:     true,
		MTLSOptional:    true,
		MTLSVerifyDepth: 3,
		MTLSClientCARef: "/etc/ssl/ca.crt",
	})
	if !strings.Contains(out, "ssl_verify_client optional") {
		t.Errorf("expected 'ssl_verify_client optional', got: %s", out)
	}
	if !strings.Contains(out, "ssl_verify_depth 3") {
		t.Errorf("expected ssl_verify_depth 3, got: %s", out)
	}
}

func TestBuildMTLSServerSnippet_PassHeaders(t *testing.T) {
	t.Parallel()
	out := buildMTLSServerSnippet(MTLSInput{
		MTLSEnabled:     true,
		MTLSClientCARef: "/etc/ssl/ca.crt",
		MTLSPassHeaders: true,
	})
	if !strings.Contains(out, "X-Client-Verify") {
		t.Errorf("expected X-Client-Verify header, got: %s", out)
	}
	if !strings.Contains(out, "X-Client-DN") {
		t.Errorf("expected X-Client-DN header, got: %s", out)
	}
}

func TestBuildMTLSServerSnippet_DefaultDepth(t *testing.T) {
	t.Parallel()
	// depth=0 should fall back to 2
	out := buildMTLSServerSnippet(MTLSInput{
		MTLSEnabled:     true,
		MTLSClientCARef: "/etc/ssl/ca.crt",
		MTLSVerifyDepth: 0,
	})
	if !strings.Contains(out, "ssl_verify_depth 2") {
		t.Errorf("expected default depth 2, got: %s", out)
	}
}

// --- buildUpstreamMTLSSnippet ---

func TestBuildUpstreamMTLSSnippet_Disabled(t *testing.T) {
	t.Parallel()
	out := buildUpstreamMTLSSnippet(UpstreamMTLSInput{UpstreamMTLSEnabled: false})
	if out != "" {
		t.Fatalf("expected empty snippet, got %q", out)
	}
}

func TestBuildUpstreamMTLSSnippet_Enabled(t *testing.T) {
	t.Parallel()
	out := buildUpstreamMTLSSnippet(UpstreamMTLSInput{
		UpstreamMTLSEnabled: true,
		UpstreamMTLSCertRef: "/etc/ssl/waf.crt",
		UpstreamMTLSKeyRef:  "/etc/ssl/waf.key",
		UpstreamMTLSCARef:   "/etc/ssl/upstream-ca.crt",
	})
	if !strings.Contains(out, "proxy_ssl_certificate /etc/ssl/waf.crt") {
		t.Errorf("expected proxy_ssl_certificate, got: %s", out)
	}
	if !strings.Contains(out, "proxy_ssl_certificate_key /etc/ssl/waf.key") {
		t.Errorf("expected proxy_ssl_certificate_key, got: %s", out)
	}
	if !strings.Contains(out, "proxy_ssl_trusted_certificate /etc/ssl/upstream-ca.crt") {
		t.Errorf("expected proxy_ssl_trusted_certificate, got: %s", out)
	}
	if !strings.Contains(out, "proxy_ssl_verify on") {
		t.Errorf("expected proxy_ssl_verify on, got: %s", out)
	}
}

func TestBuildUpstreamMTLSSnippet_NoCA(t *testing.T) {
	t.Parallel()
	out := buildUpstreamMTLSSnippet(UpstreamMTLSInput{
		UpstreamMTLSEnabled: true,
		UpstreamMTLSCertRef: "/etc/ssl/waf.crt",
		UpstreamMTLSKeyRef:  "/etc/ssl/waf.key",
		UpstreamMTLSCARef:   "",
	})
	if strings.Contains(out, "proxy_ssl_verify") {
		t.Errorf("should not include proxy_ssl_verify without CA, got: %s", out)
	}
}
