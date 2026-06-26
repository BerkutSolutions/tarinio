package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSitesRuntimeBridge_PassesDetailAuthDeps(t *testing.T) {
	files := []string{
		filepath.Join("..", "app", "static", "js", "pages", "sites.page-main-actions-runtime.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.page-main-helpers.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.detail-bind-runtime.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.runtime-load-list.js"),
	}

	contents := map[string]string{}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		contents[file] = string(raw)
	}

	requiredBridgeMarkers := []string{
		"renderAuthExclusionRulesEditor",
		"renderAuthServiceTokensEditor",
		"renderAuthHelpModal",
		"renderAntibotHelpModal",
		"formatBanDurationSeconds",
		"normalizeAuthMode",
	}
	bridgeFile := files[0]
	for _, marker := range requiredBridgeMarkers {
		if !strings.Contains(contents[bridgeFile], marker) {
			t.Fatalf("missing runtime bridge marker %q in %s", marker, bridgeFile)
		}
	}

	helperFile := files[1]
	for _, marker := range []string{
		"renderAuthExclusionRulesEditor",
		"renderAuthServiceTokensEditor",
		"renderAuthHelpModal",
		"renderAntibotHelpModal",
		"formatBanDurationSeconds",
		"renderAuthHelpModal,",
		"renderAntibotHelpModal,",
		"normalizeAuthMode",
	} {
		if !strings.Contains(contents[helperFile], marker) {
			t.Fatalf("missing helper marker %q in %s", marker, helperFile)
		}
	}

	bindFile := files[2]
	expectedCall := "syncDerivedFieldsFromIDModule(idInput, certificateInput, upstreamInput, computeUpstreamID)"
	if !strings.Contains(contents[bindFile], expectedCall) {
		t.Fatalf("expected syncDerivedFieldsFromID call %q in %s", expectedCall, bindFile)
	}

	runtimeFile := files[3]
	for _, marker := range []string{
		`console.error("[sites-runtime]"`,
		"runtime-load-failed",
		`<pre class="waf-code"`,
	} {
		if !strings.Contains(contents[runtimeFile], marker) {
			t.Fatalf("missing runtime diagnostics marker %q in %s", marker, runtimeFile)
		}
	}

	draftProfilePart2File := filepath.Join("..", "app", "static", "js", "pages", "sites.draft-profile-part2.js")
	draftProfilePart2Raw, err := os.ReadFile(draftProfilePart2File)
	if err != nil {
		t.Fatalf("read %s: %v", draftProfilePart2File, err)
	}
	draftProfilePart2 := string(draftProfilePart2Raw)
	if !strings.Contains(draftProfilePart2, "deps.siteDraftFromData(") {
		t.Fatalf("expected deps.siteDraftFromData bridge in %s", draftProfilePart2File)
	}
}
