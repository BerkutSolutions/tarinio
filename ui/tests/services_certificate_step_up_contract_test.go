package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesCertificateExportUsesApprovalStepUp(t *testing.T) {
	for _, name := range []string{"sites.detail-certs-bulk.js", "sites.detail-events-actions.js"} {
		body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		content := string(body)
		for _, marker := range []string{"createTLSExportStepUp", "exportCertificates([certificateID])"} {
			if !strings.Contains(content, marker) {
				t.Fatalf("%s missing certificate step-up marker %q", name, marker)
			}
		}
	}
}
