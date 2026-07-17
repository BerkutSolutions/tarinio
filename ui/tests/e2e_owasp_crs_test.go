package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EOWASPCRSCheckUsesOfficialReleaseDigest(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping CRS update check e2e")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)

	before := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/owasp-crs/status", hostOverride, nil)
	if before.StatusCode != http.StatusOK {
		t.Fatalf("read CRS status: status=%d body=%s", before.StatusCode, mustReadBody(t, before.Body))
	}
	var initial struct {
		ActiveVersion string `json:"active_version"`
	}
	if err := json.NewDecoder(before.Body).Decode(&initial); err != nil {
		t.Fatalf("decode initial CRS status: %v", err)
	}
	_ = before.Body.Close()

	check := requestE2EJSON(t, client, http.MethodPost, requestBaseURL+"/api/owasp-crs/check-updates", hostOverride, map[string]any{"dry_run": true})
	if check.StatusCode != http.StatusOK {
		t.Fatalf("official CRS check failed: status=%d body=%s", check.StatusCode, mustReadBody(t, check.Body))
	}
	var status struct {
		LatestVersion string `json:"latest_version"`
	}
	if err := json.NewDecoder(check.Body).Decode(&status); err != nil {
		t.Fatalf("decode CRS check: %v", err)
	}
	_ = check.Body.Close()
	if status.LatestVersion == "" {
		t.Fatal("official CRS check returned no latest version")
	}

	after := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/owasp-crs/status", hostOverride, nil)
	if after.StatusCode != http.StatusOK {
		t.Fatalf("read CRS status after dry run: status=%d body=%s", after.StatusCode, mustReadBody(t, after.Body))
	}
	var final struct {
		ActiveVersion string `json:"active_version"`
	}
	if err := json.NewDecoder(after.Body).Decode(&final); err != nil {
		t.Fatalf("decode final CRS status: %v", err)
	}
	if final.ActiveVersion != initial.ActiveVersion {
		t.Fatalf("dry-run changed active CRS from %q to %q", initial.ActiveVersion, final.ActiveVersion)
	}
}
