package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTLSExportUsesApprovalAndTOTPModal(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "tls.export-step-up.js"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, marker := range []string{"/api/certificate-materials/export-approvals", "/api/auth/step-up/totp", "approval_id", "tls-export-step-up-modal", "Escape"} {
		if !strings.Contains(source, marker) {
			t.Fatalf("missing TLS export step-up marker %s", marker)
		}
	}
}
