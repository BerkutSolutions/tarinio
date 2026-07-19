package tests

import (
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestE2EMultisiteBasicAuthIsolation(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if panelURL == "" || runtimeURL == "" {
		t.Skip("panel or runtime URL is not configured")
	}
	adminClient, requestBaseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	loginE2EUser(t, adminClient, requestBaseURL, hostOverride)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	siteA, siteB := "e2e-auth-isolation-a-"+suffix, "e2e-auth-isolation-b-"+suffix
	hostA, hostB := siteA+".test", siteB+".test"
	upstreamA, upstreamB := siteA+"-upstream", siteB+"-upstream"
	t.Cleanup(func() {
		for _, endpoint := range []string{
			"/api/sites/" + siteA + "?auto_apply=false",
			"/api/sites/" + siteB + "?auto_apply=false",
			"/api/upstreams/" + upstreamA + "?auto_apply=false",
			"/api/upstreams/" + upstreamB + "?auto_apply=false",
		} {
			resp := requestE2EJSON(t, adminClient, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
			_ = resp.Body.Close()
		}
	})

	for _, item := range []struct{ siteID, host, upstreamID string }{{siteA, hostA, upstreamA}, {siteB, hostB, upstreamB}} {
		resp := postJSON(t, adminClient, requestBaseURL+"/api/sites?auto_apply=false", hostOverride, map[string]any{
			"id": item.siteID, "primary_host": item.host, "enabled": true, "listen_http": true, "listen_https": false, "use_easy_config": true, "default_upstream_id": item.upstreamID,
		})
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("create site %s: status=%d body=%s", item.siteID, resp.StatusCode, mustReadBody(t, resp.Body))
		}
		_ = resp.Body.Close()
		resp = postJSON(t, adminClient, requestBaseURL+"/api/upstreams?auto_apply=false", hostOverride, map[string]any{
			"id": item.upstreamID, "site_id": item.siteID, "name": item.upstreamID, "scheme": "http", "host": "upstream-echo", "port": 8888, "base_path": "/",
		})
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("create upstream %s: status=%d body=%s", item.upstreamID, resp.StatusCode, mustReadBody(t, resp.Body))
		}
		_ = resp.Body.Close()
	}

	const userA, passA = "isolation-a", "isolation-a-password"
	const userB, passB = "isolation-b", "isolation-b-password"
	for _, item := range []struct{ siteID, username, password string }{{siteA, userA, passA}, {siteB, userB, passB}} {
		profile := e2eGetEasyProfile(t, adminClient, requestBaseURL, hostOverride, item.siteID)
		httpBehavior, _ := profile["http_behavior"].(map[string]any)
		if httpBehavior == nil {
			httpBehavior = map[string]any{}
			profile["http_behavior"] = httpBehavior
		}
		httpBehavior["allowed_methods"] = []string{"GET", "POST", "HEAD", "OPTIONS"}
		profile["allowed_methods"] = []string{"GET", "POST", "HEAD", "OPTIONS"}
		auth, _ := profile["security_auth_basic"].(map[string]any)
		if auth == nil {
			auth = map[string]any{}
			profile["security_auth_basic"] = auth
		}
		auth["use_auth_basic"] = true
		auth["auth_mode"] = "basic"
		auth["auth_order"] = "auth_first"
		auth["auth_basic_location"] = "sitewide"
		auth["auth_basic_user"] = item.username
		auth["auth_basic_password"] = item.password
		auth["users"] = []map[string]any{{"username": item.username, "password": item.password, "enabled": true}}
		resp := postJSON(t, adminClient, requestBaseURL+"/api/easy-site-profiles/"+item.siteID, hostOverride, profile)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("save auth profile %s: status=%d body=%s", item.siteID, resp.StatusCode, mustReadBody(t, resp.Body))
		}
		_ = resp.Body.Close()
	}
	if revisionID := e2eCompileAndApply(t, adminClient, requestBaseURL, hostOverride); revisionID == "" {
		t.Fatal("compile/apply multisite auth profiles returned an empty revision")
	}
	e2eWaitForMultisiteHost(t, runtimeURL, hostA)
	e2eWaitForMultisiteHost(t, runtimeURL, hostB)

	verified := e2eMultisiteBasicVerify(t, runtimeURL, hostA, userA, passA)
	if verified.StatusCode != http.StatusNoContent {
		body := readAndClose(t, verified.Body)
		t.Fatalf("site A credentials must be accepted by site A: status=%d body=%s", verified.StatusCode, body)
	}
	cookieA := verified.Header.Get("Set-Cookie")
	_ = verified.Body.Close()
	if !strings.Contains(cookieA, "waf_auth_") {
		t.Fatalf("site A did not issue its scoped auth cookie: %q", cookieA)
	}

	wrongSite := e2eMultisiteBasicVerify(t, runtimeURL, hostB, userA, passA)
	if wrongSite.StatusCode != http.StatusUnauthorized {
		body := readAndClose(t, wrongSite.Body)
		t.Fatalf("site A credentials must not be accepted by site B: status=%d body=%s", wrongSite.StatusCode, body)
	}
	_ = wrongSite.Body.Close()
	staleOnB := e2eMultisiteRequest(t, runtimeURL+"/", hostB, cookieA)
	if staleOnB.StatusCode != http.StatusFound {
		body := readAndClose(t, staleOnB.Body)
		t.Fatalf("site A cookie must not bypass site B: status=%d body=%s", staleOnB.StatusCode, body)
	}
	_ = staleOnB.Body.Close()
	if verifiedB := e2eMultisiteBasicVerify(t, runtimeURL, hostB, userB, passB); verifiedB.StatusCode != http.StatusNoContent {
		body := readAndClose(t, verifiedB.Body)
		t.Fatalf("site B credentials must be accepted by site B: status=%d body=%s", verifiedB.StatusCode, body)
	} else {
		_ = verifiedB.Body.Close()
	}
}

func e2eWaitForMultisiteHost(t *testing.T, runtimeURL, host string) {
	t.Helper()
	client := newE2EHTTPClient(runtimeURL, false)
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
		if err != nil {
			t.Fatalf("create multisite readiness request: %v", err)
		}
		req.Host = host
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusMisdirectedRequest {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("runtime did not activate host %s after profile apply", host)
}

func e2eMultisiteBasicVerify(t *testing.T, runtimeURL, host, username, password string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, runtimeURL+"/auth/verify/basic", nil)
	if err != nil {
		t.Fatalf("create multisite Basic Auth request: %v", err)
	}
	req.Host = host
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	resp, err := newE2EHTTPClient(runtimeURL, false).Do(req)
	if err != nil {
		t.Fatalf("send multisite Basic Auth request: %v", err)
	}
	return resp
}

func e2eMultisiteRequest(t *testing.T, endpoint, host, setCookie string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create multisite cookie request: %v", err)
	}
	req.Host = host
	req.Header.Set("Cookie", strings.SplitN(setCookie, ";", 2)[0])
	client := newE2EHTTPClient(endpoint, false)
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("send multisite cookie request: %v", err)
	}
	return resp
}
