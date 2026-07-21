package tests

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2ERecovery_InvalidCandidateDoesNotReplaceActiveProtection(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if baseURL == "" || runtimeURL == "" {
		t.Skip("E2E control-plane and runtime URLs are required")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)
	siteID := e2eUniqueID(t, "e2e-active-guard")
	upstreamID, host := siteID+"-upstream", siteID+".test"
	createE2EModSecuritySite(t, client, requestBaseURL, hostOverride, siteID, upstreamID, host)
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
			response := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
			_ = response.Body.Close()
		}
	})
	profile := e2eGetProfile(t, client, requestBaseURL, hostOverride, siteID)
	mapGetOrCreate(profile, "front_service")["security_mode"] = "block"
	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = false
	modsec["use_modsecurity_custom_configuration"] = true
	modsec["custom_configuration"] = map[string]any{"path": "modsec/active-guard.conf", "content": `SecRule REQUEST_URI "@streq /protected" "id:190001,phase:2,deny,status:403,log"`}
	e2ePutProfileWithoutAutoApply(t, client, requestBaseURL, hostOverride, siteID, profile)
	activeRevision := e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	if activeRevision == "" { t.Fatal("apply protected baseline") }
	assertE2EArtifactActive(t, activeRevision, "modsecurity/easy/"+siteID+".conf", "190001")
	request := func() int {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/protected", nil); if err != nil { t.Fatal(err) }; req.Host = host
		resp, err := newE2EHTTPClient(runtimeURL, false).Do(req); if err != nil { t.Fatalf("protected request: %v", err) }; _, _ = io.ReadAll(resp.Body); _ = resp.Body.Close(); return resp.StatusCode
	}
	if status := request(); status != http.StatusForbidden { t.Fatalf("baseline protection must block, got %d", status) }
	modsec["custom_configuration"] = map[string]any{"path": "modsec/broken.conf", "content": `SecRule REQUEST_URI "@streq /protected" "id:broken,phase:2,deny"`}
	e2ePutProfileWithoutAutoApply(t, client, requestBaseURL, hostOverride, siteID, profile)
	if badRevision := e2eCompileAndApply(t, client, requestBaseURL, hostOverride); badRevision != "" {
		t.Fatalf("invalid candidate must fail compile or apply, unexpectedly activated %s", badRevision)
	}
	assertE2EArtifactActive(t, activeRevision, "modsecurity/easy/"+siteID+".conf", "190001")
	if status := request(); status != http.StatusForbidden { t.Fatalf("failed candidate must not weaken active protection, got %d", status) }
}
