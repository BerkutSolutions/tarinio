package compiler

import (
	"fmt"
	"strings"
)

// MTLSInput holds incoming mTLS settings (client → WAF).
type MTLSInput struct {
	MTLSEnabled     bool
	MTLSOptional    bool
	MTLSVerifyDepth int
	MTLSClientCARef string
	MTLSPassHeaders bool
}

// UpstreamMTLSInput holds outgoing mTLS settings (WAF → upstream).
type UpstreamMTLSInput struct {
	UpstreamMTLSEnabled bool
	UpstreamMTLSCertRef string
	UpstreamMTLSKeyRef  string
	UpstreamMTLSCARef   string
}

// ValidateMTLS returns an error if MTLSEnabled=true but no CA ref is set.
func ValidateMTLS(input MTLSInput) error {
	if !input.MTLSEnabled {
		return nil
	}
	if strings.TrimSpace(input.MTLSClientCARef) == "" {
		return fmt.Errorf("mtls_client_ca_ref must be set when mtls_enabled=true")
	}
	if err := validateNginxCertificatePath(input.MTLSClientCARef, "mtls_client_ca_ref"); err != nil {
		return err
	}
	if input.MTLSVerifyDepth < 0 {
		return fmt.Errorf("mtls_verify_depth must be >= 0, got %d", input.MTLSVerifyDepth)
	}
	return nil
}

// ValidateUpstreamMTLS returns an error if UpstreamMTLSEnabled=true but cert or key refs are missing.
func ValidateUpstreamMTLS(input UpstreamMTLSInput) error {
	if !input.UpstreamMTLSEnabled {
		return nil
	}
	if strings.TrimSpace(input.UpstreamMTLSCertRef) == "" {
		return fmt.Errorf("upstream_mtls_cert_ref must be set when upstream_mtls_enabled=true")
	}
	if strings.TrimSpace(input.UpstreamMTLSKeyRef) == "" {
		return fmt.Errorf("upstream_mtls_key_ref must be set when upstream_mtls_enabled=true")
	}
	for field, value := range map[string]string{"upstream_mtls_cert_ref": input.UpstreamMTLSCertRef, "upstream_mtls_key_ref": input.UpstreamMTLSKeyRef, "upstream_mtls_ca_ref": input.UpstreamMTLSCARef} {
		if strings.TrimSpace(value) != "" {
			if err := validateNginxCertificatePath(value, field); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildMTLSServerSnippet generates the nginx server-block directives for
// incoming mTLS client certificate verification. Returns "" if disabled.
func buildMTLSServerSnippet(input MTLSInput) string {
	if !input.MTLSEnabled {
		return ""
	}

	depth := input.MTLSVerifyDepth
	if depth <= 0 {
		depth = 2
	}

	var b strings.Builder

	// ssl_verify_client on | optional
	if input.MTLSOptional {
		b.WriteString("    ssl_verify_client optional;\n")
	} else {
		b.WriteString("    ssl_verify_client on;\n")
	}

	caRef := strings.TrimSpace(input.MTLSClientCARef)
	fmt.Fprintf(&b, "    ssl_client_certificate %s;\n", caRef)
	fmt.Fprintf(&b, "    ssl_verify_depth %d;\n", depth)

	if input.MTLSPassHeaders {
		b.WriteString("    proxy_set_header X-Client-Verify $ssl_client_verify;\n")
		b.WriteString("    proxy_set_header X-Client-DN $ssl_client_s_dn;\n")
	}

	return b.String()
}

// buildUpstreamMTLSSnippet generates the nginx proxy_ssl_* directives for
// outgoing mTLS connections to upstream. Returns "" if disabled.
func buildUpstreamMTLSSnippet(input UpstreamMTLSInput) string {
	if !input.UpstreamMTLSEnabled {
		return ""
	}

	var b strings.Builder

	certRef := strings.TrimSpace(input.UpstreamMTLSCertRef)
	keyRef := strings.TrimSpace(input.UpstreamMTLSKeyRef)

	fmt.Fprintf(&b, "    proxy_ssl_certificate %s;\n", certRef)
	fmt.Fprintf(&b, "    proxy_ssl_certificate_key %s;\n", keyRef)

	if caRef := strings.TrimSpace(input.UpstreamMTLSCARef); caRef != "" {
		fmt.Fprintf(&b, "    proxy_ssl_trusted_certificate %s;\n", caRef)
		b.WriteString("    proxy_ssl_verify on;\n")
		b.WriteString("    proxy_ssl_verify_depth 2;\n")
	}

	return b.String()
}
