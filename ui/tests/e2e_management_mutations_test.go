package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EManagementHostMutationWorkflow(t *testing.T) {
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if runtimeURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL is not set; skipping management mutation e2e")
	}
	activeRuntimeURL := runtimeURL
	if httpsRuntimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_HTTPS_URL")), "/"); httpsRuntimeURL != "" {
		activeRuntimeURL = httpsRuntimeURL
	}
	// Requests deliberately go through nginx/ModSecurity. Calling control-plane
	// directly cannot detect a CRS 403 before the API handler.
	client, requestBaseURL, _ := newE2EClientAndBase(t, runtimeURL)
	host := strings.TrimSpace(os.Getenv("WAF_E2E_MANAGEMENT_HOST"))
	if host == "" {
		host = "e2e-management.test"
	}
	loginE2EUser(t, client, requestBaseURL, host)
	managementSiteID := e2eManagementSiteID(t, client, requestBaseURL, host, "http://"+host)
	originalProfile := e2eGetProfile(t, client, requestBaseURL, host, managementSiteID)
	protectedProfile := cloneMap(originalProfile)
	front := mapGetOrCreate(protectedProfile, "front_service")
	front["security_mode"] = "block"
	modsec := mapGetOrCreate(protectedProfile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = true
	e2ePutProfile(t, client, requestBaseURL, host, managementSiteID, protectedProfile)
	if rev := e2eCompileAndApply(t, client, requestBaseURL, host); rev == "" {
		t.Fatal("enable ModSecurity and apply management revision through runtime failed")
	}
	if activeRuntimeURL != runtimeURL {
		// The bootstrap management site has a TLS binding. Its first apply
		// replaces the bootstrap HTTP listener, so remaining mutations must
		// traverse the active HTTPS listener.
		var requestHostOverride string
		client, requestBaseURL, requestHostOverride = newE2EClientAndBase(t, activeRuntimeURL)
		requestHostOverride = host
		loginE2EUser(t, client, requestBaseURL, requestHostOverride)
	}
	id := "e2e-management-" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	userID, policyID, upstreamID := id+"-user", id+"-policy", id+"-upstream"
	t.Cleanup(func() {
		e2ePutProfile(t, client, requestBaseURL, host, managementSiteID, originalProfile)
		_ = e2eCompileAndApply(t, client, requestBaseURL, host)
		_ = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/waf-policies/"+policyID+"?auto_apply=false", host, nil).Body.Close()
		_ = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/administration/users/"+userID, host, nil).Body.Close()
		_ = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+id+"?auto_apply=false", host, nil).Body.Close()
		_ = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", host, nil).Body.Close()
	})
	created := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", host, map[string]any{"id": id, "primary_host": id + ".example.test", "enabled": true, "default_upstream_id": upstreamID})
	assertE2EStatus(t, created, "create site through management host", http.StatusCreated, http.StatusOK)
	upstream := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", host, map[string]any{"id": upstreamID, "site_id": id, "name": upstreamID, "scheme": "http", "host": "upstream-echo", "port": 8888, "base_path": "/"})
	assertE2EStatus(t, upstream, "create upstream through management host", http.StatusCreated, http.StatusOK)
	updated := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/sites/"+id+"?auto_apply=false", host, map[string]any{"id": id, "primary_host": id + ".example.test", "enabled": false, "default_upstream_id": upstreamID})
	assertE2EStatus(t, updated, "disable site through management host", http.StatusOK)
	user := postJSON(t, client, requestBaseURL+"/api/administration/users", host, map[string]any{"id": userID, "username": userID, "email": userID + "@example.test", "password": "password-123", "role_ids": []string{"admin"}})
	assertE2EStatus(t, user, "create user through management host", http.StatusCreated, http.StatusOK)
	userUpdate := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/administration/users/"+userID, host, map[string]any{"email": "updated-" + userID + "@example.test", "role_ids": []string{"admin"}, "is_active": true})
	assertE2EStatus(t, userUpdate, "update user through management host", http.StatusOK)
	policy := postJSON(t, client, requestBaseURL+"/api/waf-policies?auto_apply=false", host, map[string]any{"id": policyID, "site_id": id, "enabled": true, "mode": "detection", "crs_enabled": true})
	assertE2EStatus(t, policy, "create policy through management host", http.StatusCreated, http.StatusOK)
	if rev := e2eCompileAndApply(t, client, requestBaseURL, host); rev == "" {
		t.Fatal("compile/apply through management host failed")
	}
	deleted := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+id+"?auto_apply=false", host, nil)
	assertE2EStatus(t, deleted, "delete site through management host", http.StatusNoContent, http.StatusOK)
}
