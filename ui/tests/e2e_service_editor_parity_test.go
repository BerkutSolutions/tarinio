package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestE2EServiceEditorParity proves that the simple editor's underlying
// resources, raw export values, compiled revision and active WAF all describe
// the same user-created service.
func TestE2EServiceEditorParity(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if baseURL == "" || runtimeURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)
	attackerIP := e2eAttackerIP()

	const siteID = "e2e-editor-parity"
	const upstreamID = siteID + "-upstream"
	const host = siteID + ".test"
	for _, path := range []string{
		"/api/sites/" + siteID + "?auto_apply=false",
		"/api/upstreams/" + upstreamID + "?auto_apply=false",
		"/api/access-policies/" + siteID + "-access?auto_apply=false",
	} {
		response := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+path, hostOverride, nil)
		_ = response.Body.Close()
	}
	t.Cleanup(func() {
		for _, path := range []string{
			"/api/access-policies/" + siteID + "-access?auto_apply=false",
			"/api/sites/" + siteID + "?auto_apply=false",
			"/api/upstreams/" + upstreamID + "?auto_apply=false",
		} {
			response := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+path, hostOverride, nil)
			_ = response.Body.Close()
		}
		e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	})

	assertE2EStatus(t, postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", hostOverride, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "listen_http": true, "listen_https": false, "use_easy_config": true, "default_upstream_id": upstreamID,
	}), "create protected service", http.StatusCreated, http.StatusOK)
	assertE2EStatus(t, postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", hostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "127.0.0.1", "port": 8080, "base_path": "/",
	}), "create initial upstream", http.StatusCreated, http.StatusOK)

	profile := e2eGetProfile(t, client, requestBaseURL, hostOverride, siteID)
	front := mapGetOrCreate(profile, "front_service")
	front["server_name"] = host
	updatedProfile := e2ePutProfileWithoutAutoApply(t, client, requestBaseURL, hostOverride, siteID, profile)
	if got := mapGetOrCreate(updatedProfile, "front_service")["server_name"]; got != host {
		t.Fatalf("easy profile did not persist service host: got=%v want=%s", got, host)
	}

	// This is the resource write performed by the simple editor after the user
	// changes the upstream fields. The raw editor is derived from these fields.
	assertE2EStatus(t, requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", hostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "privatebin", "port": 8080, "base_path": "/",
	}), "save changed upstream from simple editor", http.StatusOK)
	assertE2EStatus(t, postJSON(t, client, requestBaseURL+"/api/access-policies/upsert?auto_apply=false", hostOverride, map[string]any{
		"id": siteID + "-access", "site_id": siteID, "enabled": true, "allowlist": []string{attackerIP},
	}), "save service allowlist", http.StatusCreated, http.StatusOK)

	// A subsequent Easy Profile save must not erase the independently persisted
	// allowlist. This is the ordering used by the simple editor.
	e2ePutProfileWithoutAutoApply(t, client, requestBaseURL, hostOverride, siteID, updatedProfile)

	upstreamsResponse := getWithAuth(t, client, requestBaseURL+"/api/upstreams", hostOverride)
	defer upstreamsResponse.Body.Close()
	var upstreams []map[string]any
	if err := json.NewDecoder(upstreamsResponse.Body).Decode(&upstreams); err != nil {
		t.Fatalf("decode raw upstream resources: %v", err)
	}
	foundUpstream := false
	for _, item := range upstreams {
		if item["id"] == upstreamID {
			port, ok := item["port"].(float64)
			if !ok || item["host"] != "privatebin" || item["scheme"] != "http" || int(port) != 8080 {
				t.Fatalf("raw upstream differs from simple editor draft: %#v", item)
			}
			foundUpstream = true
			break
		}
	}
	if !foundUpstream {
		t.Fatalf("raw upstream resource %q was not found: %#v", upstreamID, upstreams)
	}

	accessResponse := getWithAuth(t, client, requestBaseURL+"/api/access-policies", hostOverride)
	defer accessResponse.Body.Close()
	var policies []map[string]any
	if err := json.NewDecoder(accessResponse.Body).Decode(&policies); err != nil {
		t.Fatalf("decode access policies: %v", err)
	}
	foundAllowlist := false
	for _, policy := range policies {
		if policy["site_id"] != siteID {
			continue
		}
		entries, ok := policy["allowlist"].([]any)
		if !ok {
			t.Fatalf("raw allowlist has unexpected type: %#v", policy)
		}
		for _, entry := range entries {
			if entry == attackerIP {
				foundAllowlist = true
			}
		}
	}
	if !foundAllowlist {
		t.Fatal("allowlist was not preserved in the raw access policy")
	}

	revisionID := e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	if revisionID == "" {
		t.Fatal("compile/apply returned an empty revision ID")
	}
	controlPlaneContainer := strings.TrimSpace(os.Getenv("WAF_E2E_CONTROL_PLANE_CONTAINER"))
	if controlPlaneContainer == "" {
		controlPlaneContainer = "waf-e2e-control-plane"
	}
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		runtimeContainer = "waf-e2e-runtime"
	}
	artifactPath := "nginx/sites/" + siteID + ".conf"
	revisionArtifact, err := exec.Command("docker", "exec", controlPlaneContainer, "cat", "/var/lib/waf/candidates/"+revisionID+"/"+artifactPath).CombinedOutput()
	if err != nil {
		t.Fatalf("read revision artifact: %v: %s", err, revisionArtifact)
	}
	if !strings.Contains(string(revisionArtifact), "server privatebin:8080;") {
		t.Fatalf("compiled revision retained stale upstream: %s", revisionArtifact)
	}
	accessArtifact, err := exec.Command("docker", "exec", controlPlaneContainer, "cat", "/var/lib/waf/candidates/"+revisionID+"/nginx/access/"+siteID+".conf").CombinedOutput()
	if err != nil || !strings.Contains(string(accessArtifact), "allow "+attackerIP+";") || !strings.Contains(string(accessArtifact), "deny all;") {
		t.Fatalf("compiled allowlist is incomplete: err=%v artifact=%s", err, accessArtifact)
	}
	activeArtifact, err := exec.Command("docker", "exec", runtimeContainer, "cat", "/etc/waf/current/"+artifactPath).CombinedOutput()
	if err != nil || string(activeArtifact) != string(revisionArtifact) {
		t.Fatalf("active runtime artifact differs from revision: err=%v", err)
	}

	allowed := e2eContainerHTTPStatus(t, e2eContainerName(t, "WAF_E2E_ATTACKER_CONTAINER", "waf-e2e-attacker"), host)
	if allowed != http.StatusOK {
		t.Fatalf("allowlisted E2E client: status=%d, want 200 from protected upstream", allowed)
	}
	if denied := e2eContainerHTTPStatus(t, e2eContainerName(t, "WAF_E2E_L4_ATTACKER_CONTAINER", "waf-e2e-l4-attacker"), host); denied != http.StatusForbidden {
		t.Fatalf("non-allowlisted E2E client: status=%d, want 403", denied)
	}
	t.Logf("editor/raw/runtime parity revision=%s upstream=privatebin:8080 allowlist=%s allowed_status=%d", revisionID, attackerIP, allowed)
}

func e2eContainerName(t *testing.T, environment, fallback string) string {
	t.Helper()
	if name := strings.TrimSpace(os.Getenv(environment)); name != "" {
		return name
	}
	return fallback
}

func e2eContainerHTTPStatus(t *testing.T, container, host string) int {
	t.Helper()
	output, err := exec.Command("docker", "exec", container, "wget", "-S", "-O", "/dev/null", "--header=Host: "+host, "http://runtime/").CombinedOutput()
	if err != nil && !strings.Contains(string(output), "HTTP/1.1") {
		t.Fatalf("request from %s: %v: %s", container, err, output)
	}
	for _, status := range []int{http.StatusOK, http.StatusMovedPermanently, http.StatusFound, http.StatusForbidden, http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable} {
		if strings.Contains(string(output), "HTTP/1.1 "+strconv.Itoa(status)) {
			return status
		}
	}
	t.Fatalf("request from %s has no supported HTTP status: %s", container, output)
	return 0
}

func e2ePutProfileWithoutAutoApply(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID string, profile map[string]any) map[string]any {
	t.Helper()
	response := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/easy-site-profiles/"+siteID+"?auto_apply=false", requestHostOverride, profile)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("put profile without auto-apply: status=%d body=%s", response.StatusCode, mustReadBody(t, response.Body))
	}
	var updated map[string]any
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	return updated
}
