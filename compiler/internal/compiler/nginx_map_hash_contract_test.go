package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNginxTemplateSupportsLongMapKeysAndServerNames(t *testing.T) {
	path := filepath.Join("..", "..", "templates", "nginx", "nginx.conf.tmpl")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read nginx template: %v", err)
	}
	text := string(content)
	for _, directive := range []string{
		"map_hash_max_size 4096;",
		"map_hash_bucket_size 128;",
		"server_names_hash_max_size 4096;",
		"server_names_hash_bucket_size 128;",
	} {
		if !strings.Contains(text, directive) {
			t.Fatalf("nginx template must contain %q for E2E-generated host map keys", directive)
		}
	}
}
