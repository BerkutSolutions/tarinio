package references

import "testing"

func TestResolutionErrorContainsStableCode(t *testing.T) {
	err := NewError(CodeCertificateHostMismatch, "certificate_id", "cert-a")
	if err.Error() != "certificate_host_mismatch: certificate_id cert-a" {
		t.Fatalf("unexpected error: %v", err)
	}
}
