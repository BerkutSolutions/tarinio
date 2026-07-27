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
	source := string(raw)
	for _, marker := range []string{
		"COPY --chmod=0644 ui/nginx.conf /etc/nginx/conf.d/default.conf",
		"COPY --chmod=0644 ui/nginx.rootless.conf /etc/nginx/nginx.conf",
		"RUN chmod -R a+rX /usr/share/nginx/html",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("ui image must normalize rootless-readable permissions with %q", marker)
		}
	}
}

func TestUIImageTestStageIncludesRepositoryContracts(t *testing.T) {
	path := filepath.Join("..", "Dockerfile")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(raw)
	for _, marker := range []string{
		"RUN apk add --no-cache nodejs npm python3",
		"COPY .gitlab-ci.yml ./",
		"COPY scripts ./scripts",
		"COPY deploy/compose ./deploy/compose",
		"COPY control-plane/Dockerfile ./control-plane/Dockerfile",
		"COPY runtime/image/Dockerfile ./runtime/image/Dockerfile",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("ui test stage must include repository contract dependency %q", marker)
		}
	}
}
