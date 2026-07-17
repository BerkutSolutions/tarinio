package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIContainerNormalizesStaticAssetPermissions(t *testing.T) {
	path := filepath.Join("..", "Dockerfile")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), "RUN chmod -R a+rX /usr/share/nginx/html") {
		t.Fatal("ui image must normalize static asset permissions after COPY")
	}
}
