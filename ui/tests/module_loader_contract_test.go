package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIContract_DynamicModulesUseBoundedBackoff(t *testing.T) {
	app, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.js"))
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	source := string(app)
	for _, marker := range []string{
		"const PAGE_MODULE_RETRY_DELAYS_MS = [250, 750];",
		"window.setTimeout(resolve, PAGE_MODULE_RETRY_DELAYS_MS[index])",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("app.js missing module-load resilience marker %q", marker)
		}
	}

	index, err := os.ReadFile(filepath.Join("..", "app", "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	if !strings.Contains(string(index), "app.js?v=20260726-critical-styles-1") {
		t.Fatal("index.html must invalidate the cached module loader")
	}
}

func TestUIContract_CriticalStylesUseBoundedReload(t *testing.T) {
	loader, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.critical-styles.js"))
	if err != nil {
		t.Fatalf("read app.critical-styles.js: %v", err)
	}
	source := string(loader)
	for _, marker := range []string{
		"const CRITICAL_STYLE_RETRY_DELAYS_MS = [250, 750];",
		"link?.sheet",
		"retryURL.searchParams.set(\"style_retry\"",
		"link[rel=\"stylesheet\"][data-critical-style]",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("critical stylesheet loader missing marker %q", marker)
		}
	}

	app, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.js"))
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	if !strings.Contains(string(app), "await ensureCriticalStyles();") {
		t.Fatal("app bootstrap must await critical styles before rendering")
	}
	index, err := os.ReadFile(filepath.Join("..", "app", "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	if strings.Count(string(index), "data-critical-style") != 2 {
		t.Fatal("both application stylesheets must be marked critical")
	}
}

func TestUIContract_EntryTransportFailureDoesNotRevokeSession(t *testing.T) {
	guard, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "guard.js"))
	if err != nil {
		t.Fatalf("read guard.js: %v", err)
	}
	source := string(guard)
	start := strings.Index(source, "export async function checkEntryAccess(mode)")
	end := strings.Index(source, "const onboardingRequired")
	if start < 0 || end <= start {
		t.Fatal("guard.js entry-access contract is missing")
	}
	setupCheck := source[start:end]
	if !strings.Contains(setupCheck, "const setup = await getSetupStatus();") {
		t.Fatal("entry access must fail closed when the setup request rejects")
	}
	if strings.Contains(setupCheck, "forceRelogin") {
		t.Fatal("a setup transport failure must not revoke an authenticated session")
	}
	if !strings.Contains(source, "await forceRelogin(\"session_missing\")") {
		t.Fatal("a real missing auth session must still force reauthentication")
	}

	app, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.js"))
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	if !strings.Contains(string(app), "./guard.js?v=20260726-entry-race-1") {
		t.Fatal("app.js must invalidate the cached entry guard")
	}
}
