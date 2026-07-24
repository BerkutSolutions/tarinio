//go:build e2e

package tests

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestE2EVirtualPatchAPICompileApplyRuntime(t *testing.T) {
	requestURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if requestURL == "" || runtimeURL == "" {
		t.Fatal("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	siteID := e2eUniqueID(t, "e2e-virtual-patch")
	upstreamID := siteID + "-upstream"
	host := siteID + ".test"
	patchPath := "/e2e-virtual-patch-block"
	patchID := "vp-e2e-runtime"

	t.Cleanup(func() {
		for _, path := range []string{
			"/api/virtual-patches/" + siteID + "/" + patchID,
			"/api/sites/" + siteID + "?auto_apply=false",
			"/api/upstreams/" + upstreamID + "?auto_apply=false",
		} {
			resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+path, requestHostOverride, nil)
			_ = resp.Body.Close()
		}
	})

	createE2EModSecuritySite(t, client, requestBaseURL, requestHostOverride, siteID, upstreamID, host)
	profile := e2eGetProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	front := mapGetOrCreate(profile, "front_service")
	front["security_mode"] = "block"
	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = false
	e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)

	created := postJSON(t, client, requestBaseURL+"/api/virtual-patches/"+siteID+"?auto_apply=false", requestHostOverride, map[string]any{
		"id": patchID, "pattern": "^" + patchPath + "$", "target": "uri", "action": "block", "description": "disposable runtime E2E patch",
	})
	createdBody, _ := io.ReadAll(created.Body)
	_ = created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create virtual patch: status=%d body=%s", created.StatusCode, createdBody)
	}

	listed := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/virtual-patches/"+siteID, requestHostOverride, nil)
	listedBody, _ := io.ReadAll(listed.Body)
	_ = listed.Body.Close()
	if listed.StatusCode != http.StatusOK || !strings.Contains(string(listedBody), patchID) || !strings.Contains(string(listedBody), patchPath) {
		t.Fatalf("virtual patch API readback: status=%d body=%s", listed.StatusCode, listedBody)
	}

	revisionID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	assertE2EArtifactActive(t, revisionID, "modsecurity/easy/"+siteID+".conf", "Virtual patch "+patchID, patchPath)
	assertE2EVirtualPatchRuntimeStatus(t, runtimeURL, host, patchPath, http.StatusForbidden, "active virtual patch")

	deleted := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/virtual-patches/"+siteID+"/"+patchID+"?auto_apply=false", requestHostOverride, nil)
	deletedBody, _ := io.ReadAll(deleted.Body)
	_ = deleted.Body.Close()
	if deleted.StatusCode != http.StatusNoContent {
		t.Fatalf("delete virtual patch: status=%d body=%s", deleted.StatusCode, deletedBody)
	}

	revisionID = e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	assertE2EArtifactActive(t, revisionID, "modsecurity/easy/"+siteID+".conf", "SecRuleEngine", "On")
	assertE2EVirtualPatchArtifactAbsent(t, revisionID, "modsecurity/easy/"+siteID+".conf", patchID)
	assertE2EVirtualPatchRuntimeStatus(t, runtimeURL, host, patchPath, http.StatusOK, "deleted virtual patch")
}

func assertE2EVirtualPatchArtifactAbsent(t *testing.T, revisionID, path, forbidden string) {
	t.Helper()
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		runtimeContainer = "waf-e2e-runtime"
	}
	contents, err := exec.Command("docker", "exec", runtimeContainer, "cat", "/etc/waf/current/"+path).CombinedOutput()
	if err != nil {
		t.Fatalf("read active artifact for revision %s: %v: %s", revisionID, err, contents)
	}
	if strings.Contains(string(contents), forbidden) {
		t.Fatalf("active artifact %s still contains deleted virtual patch %q", path, forbidden)
	}
}

func assertE2EVirtualPatchRuntimeStatus(t *testing.T, runtimeURL, host, path string, want int, stage string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, runtimeURL+path, nil)
	if err != nil {
		t.Fatalf("build %s request: %v", stage, err)
	}
	req.Host = host
	resp, err := (&http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}).Do(req)
	if err != nil {
		t.Fatalf("runtime %s request: %v", stage, err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("runtime %s: status=%d want=%d body=%s", stage, resp.StatusCode, want, body)
	}
}
