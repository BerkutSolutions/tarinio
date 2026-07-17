package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnboardingRedirectUsesOnlyPublicBootstrapState(t *testing.T) {
	for _, name := range []string{"guard.js", "login.js"} {
		content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(content), "has_active_revision") {
			t.Fatalf("%s must not infer onboarding from private active-revision state", name)
		}
	}
}
