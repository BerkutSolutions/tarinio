package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicNginxConfigsDoNotExposePersistedControlPlaneData(t *testing.T) {
	for _, name := range []string{"nginx.conf", "nginx.testpage.conf"} {
		content, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(content)
		for _, forbidden := range []string{
			"alias /var/lib/waf",
			"alias /etc/waf",
			"alias /var/run/waf",
			"alias /var/lib/waf/control-plane",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s exposes persisted control-plane data through %q", name, forbidden)
			}
		}
		for _, retired := range []string{
			"location = /api-app/sites {\n        return 404;",
			"location = /api-app/easy-site-profiles {\n        return 404;",
		} {
			if name == "nginx.testpage.conf" && !strings.Contains(text, retired) {
				t.Fatalf("%s must retain an explicit 404 for retired endpoint %q", name, retired)
			}
		}
	}
}
