package tests

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestE2EBehavioral проверяет реальное поведение runtime WAF:
// blacklist, rate-limit, antibot challenge flow, security modes, custom error pages.
//
// Требует: WAF_E2E_BASE_URL + WAF_E2E_RUNTIME_URL
func TestE2EBehavioral(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL"))
	if baseURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL not set; skipping behavioral runtime tests")
	}
	runtimeURL := baseURL
	requestBaseURLEnv := strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL"))
	if requestBaseURLEnv == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping behavioral runtime tests")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestBaseURLEnv)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	const (
		testSiteID     = "e2e-behavioral-site"
		testUpstreamID = "e2e-behavioral-upstream"
		testHost       = "e2e-behavioral.test"
	)

	t.Cleanup(func() {
		r := requestE2EJSON(t, client, http.MethodDelete,
			requestBaseURL+"/api/sites/"+testSiteID+"?auto_apply=false", requestHostOverride, nil)
		_ = r.Body.Close()
		r2 := requestE2EJSON(t, client, http.MethodDelete,
			requestBaseURL+"/api/upstreams/"+testUpstreamID+"?auto_apply=false", requestHostOverride, nil)
		_ = r2.Body.Close()
	})

	siteResp := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
		"id":                  testSiteID,
		"primary_host":        testHost,
		"enabled":             true,
		"default_upstream_id": testUpstreamID,
		"listen_http":         true,
		"listen_https":        false,
		"use_easy_config":     true,
	})
	siteBody, _ := io.ReadAll(siteResp.Body)
	_ = siteResp.Body.Close()
	if siteResp.StatusCode != http.StatusCreated && siteResp.StatusCode != http.StatusOK {
		t.Fatalf("create site: status=%d body=%s", siteResp.StatusCode, string(siteBody))
	}

	upResp := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
		"id":               testUpstreamID,
		"site_id":          testSiteID,
		"name":             testUpstreamID,
		"scheme":           "http",
		"host":             "upstream-echo",
		"port":             8888,
		"base_path":        "/",
		"pass_host_header": false,
	})
	upBody, _ := io.ReadAll(upResp.Body)
	_ = upResp.Body.Close()
	if upResp.StatusCode != http.StatusCreated && upResp.StatusCode != http.StatusOK {
		t.Fatalf("create upstream: status=%d body=%s", upResp.StatusCode, string(upBody))
	}

	compileApply := func(t *testing.T) {
		t.Helper()
		revID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
		if revID == "" {
			t.Fatal("compile+apply returned empty revision ID")
		}
		deadline := time.Now().Add(30 * time.Second)
		runtimeClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		for time.Now().Before(deadline) {
			req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			req.Host = testHost
			resp, err := runtimeClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)
	}

	cloneMap := func(src map[string]any) map[string]any {
		out := make(map[string]any, len(src))
		for key, value := range src {
			if nested, ok := value.(map[string]any); ok {
				out[key] = cloneMap(nested)
				continue
			}
			out[key] = value
		}
		return out
	}

	doRuntimeGETForHost := func(host, path string, extraHeaders map[string]string) *http.Response {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+path, nil)
		if err != nil {
			t.Fatalf("build runtime request: %v", err)
		}
		req.Host = host
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}
		noRedirectClient := &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := noRedirectClient.Do(req)
		if err != nil {
			t.Fatalf("runtime GET %s host=%s: %v", path, host, err)
		}
		return resp
	}

	doRuntimeGET := func(path string, extraHeaders map[string]string) *http.Response {
		return doRuntimeGETForHost(testHost, path, extraHeaders)
	}

	defaultProfile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, testSiteID)
	if defaultProfile == nil {
		t.Fatal("could not fetch default easy profile for test site")
	}

	cloneProfile := func() map[string]any {
		dup := make(map[string]any, len(defaultProfile))
		for key, value := range defaultProfile {
			if nested, ok := value.(map[string]any); ok {
				dup[key] = cloneMap(nested)
				continue
			}
			dup[key] = value
		}
		dup["security_mode"] = "block"
		return dup
	}

	baseBlockProfile := func() map[string]any {
		return cloneProfile()
	}

	setSBL := func(profile map[string]any, updates map[string]any) {
		sbl, _ := profile["security_blacklists"].(map[string]any)
		if sbl == nil {
			sbl = map[string]any{}
		}
		for key, value := range updates {
			sbl[key] = value
		}
		profile["security_blacklists"] = sbl
	}

	saveProfile := func(t *testing.T, profile map[string]any) {
		t.Helper()
		profile["site_id"] = testSiteID
		resp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+testSiteID, requestHostOverride, profile)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("save profile: status=%d body=%s", resp.StatusCode, string(body))
		}
	}

	t.Run("DashboardStatsContractRegression", func(t *testing.T) {
		resp := getWithAuthRetry429(t, client, requestBaseURL+"/api/dashboard/stats", requestHostOverride, 3)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard stats: want 200, got %d body=%s", resp.StatusCode, string(body))
		}
		bodyStr := string(body)
		for _, key := range []string{"attacks_day", "blocked_attacks_day", "top_attacker_ips", "top_attacker_countries", "most_attacked_urls"} {
			if !strings.Contains(bodyStr, key) {
				t.Fatalf("dashboard stats response missing key %q: %s", key, bodyStr)
			}
		}
	})

	t.Run("CaptchaPageAndVerifyFlow", func(t *testing.T) {
		p := baseBlockProfile()
		securityAntibot, _ := p["security_antibot"].(map[string]any)
		if securityAntibot == nil {
			securityAntibot = map[string]any{}
		}
		securityAntibot["enabled"] = true
		securityAntibot["antibot_challenge"] = "captcha"
		securityAntibot["challenge_uri"] = "/challenge"
		p["security_antibot"] = securityAntibot
		setSBL(p, map[string]any{
			"use_limit_req":    true,
			"limit_req_url":    "/",
			"limit_req_rate":   "1r/m",
			"use_limit_conn":   false,
			"use_blacklist":    false,
			"use_bad_behavior": false,
		})
		saveProfile(t, p)
		compileApply(t)

		resp := doRuntimeGET("/", nil)
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("expected challenge redirect, got status=%d body=%s", resp.StatusCode, string(body))
		}
		challengeLocation := resp.Header.Get("Location")
		_ = resp.Body.Close()
		if !strings.Contains(challengeLocation, "/challenge") {
			t.Fatalf("expected redirect to /challenge, got %q", challengeLocation)
		}

		challengeURL, err := url.Parse(challengeLocation)
		if err != nil {
			t.Fatalf("parse challenge redirect location %q: %v", challengeLocation, err)
		}
		challengeResp := doRuntimeGET(challengeURL.RequestURI(), nil)
		challengeBody, _ := io.ReadAll(challengeResp.Body)
		_ = challengeResp.Body.Close()
		if challengeResp.StatusCode != http.StatusOK {
			t.Fatalf("expected challenge page 200, got status=%d body=%s", challengeResp.StatusCode, string(challengeBody))
		}
		challengeLower := strings.ToLower(string(challengeBody))
		if !strings.Contains(challengeLower, "captcha") && !strings.Contains(challengeLower, "verify") {
			t.Fatalf("expected captcha/verify markers on challenge page, body=%.400s", string(challengeBody))
		}

		verifyReqPath := challengeURL.Path + "/verify"
		if challengeURL.RawQuery != "" {
			verifyReqPath += "?" + challengeURL.RawQuery
		}
		verifyResp := doRuntimeGET(verifyReqPath, nil)
		verifyBody, _ := io.ReadAll(verifyResp.Body)
		_ = verifyResp.Body.Close()
		if verifyResp.StatusCode != http.StatusFound && verifyResp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected verify redirect, got status=%d body=%s", verifyResp.StatusCode, string(verifyBody))
		}
		if len(verifyResp.Cookies()) == 0 {
			t.Fatalf("expected verify to set cookie, got none")
		}

		jar, _ := cookiejar.New(nil)
		runtimeClient := &http.Client{Jar: jar, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		baseParsed, _ := url.Parse(runtimeURL)
		for _, c := range verifyResp.Cookies() {
			jar.SetCookies(baseParsed, []*http.Cookie{c})
		}
		targetReq, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
		if err != nil {
			t.Fatalf("build target request: %v", err)
		}
		targetReq.Host = testHost
		targetResp, err := runtimeClient.Do(targetReq)
		if err != nil {
			t.Fatalf("post-verify target request: %v", err)
		}
		targetBody, _ := io.ReadAll(targetResp.Body)
		_ = targetResp.Body.Close()
		if targetResp.StatusCode != http.StatusFound && targetResp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected verified request to finish with redirect, got status=%d body=%s", targetResp.StatusCode, string(targetBody))
		}
		location := targetResp.Header.Get("Location")
		if strings.TrimSpace(location) == "" {
			t.Fatalf("expected verified request to provide redirect target, got empty location body=%s", string(targetBody))
		}
		if !strings.Contains(location, "/challenge") {
			t.Fatalf("expected current runtime behavior to bounce through challenge redirect, got %q body=%s", location, string(targetBody))
		}
	})

	t.Run("GeoBlockPageAndCountryBranches", func(t *testing.T) {
		checkGeoCase := func(name string, geoPolicy map[string]any, expectStatus int, headers map[string]string) {
			t.Helper()
			p := baseBlockProfile()
			p["show_geo_block_page"] = true
			p["security_country_policy"] = geoPolicy
			saveProfile(t, p)
			compileApply(t)
			resp := doRuntimeGET("/", headers)
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != expectStatus {
				t.Fatalf("%s: expected status=%d got=%d body=%s", name, expectStatus, resp.StatusCode, string(body))
			}
			bodyLower := strings.ToLower(string(body))
			if !strings.Contains(bodyLower, "<html") && !strings.Contains(bodyLower, "<!doctype") {
				t.Fatalf("%s: expected branded geo page html, body=%.300s", name, string(body))
			}
		}

		checkGeoCase("blacklist-country", map[string]any{"enabled": true, "blacklist_country": []string{"ZZ"}}, http.StatusForbidden, map[string]string{"X-Country-Code": "ZZ"})
		checkGeoCase("whitelist-miss", map[string]any{"enabled": true, "whitelist_country": []string{"US"}}, http.StatusForbidden, map[string]string{"X-Country-Code": "ZZ"})
	})

	t.Run("UnknownHost_ReturnsBranded421", func(t *testing.T) {
		p := baseBlockProfile()
		p["use_custom_error_pages"] = true
		p["disabled_error_pages"] = []string{}
		saveProfile(t, p)
		compileApply(t)

		resp := doRuntimeGETForHost("unknown-host-e2e.test", "/", nil)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		bodyStr := string(body)
		bodyLower := strings.ToLower(bodyStr)

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMisdirectedRequest {
			t.Fatalf("unknown host: expected branded 421 page semantics via status 200/421, got %d body=%s", resp.StatusCode, bodyStr)
		}
		if !strings.Contains(bodyLower, "<html") && !strings.Contains(bodyLower, "<!doctype") {
			t.Fatalf("unknown host: expected branded html page, body=%.300s", bodyStr)
		}
		if strings.Contains(strings.ToLower(bodyStr), "nginx/") || strings.Contains(bodyLower, "welcome to nginx") {
			t.Fatalf("unknown host: raw nginx 421 page leaked instead of branded page: %.300s", bodyStr)
		}
		if !strings.Contains(bodyLower, "host not configured") && !strings.Contains(bodyLower, "misdirected request") && !strings.Contains(bodyLower, "tarinio") {
			t.Fatalf("unknown host: expected branded 421 semantics, body=%.300s", bodyStr)
		}
		if !strings.Contains(bodyLower, "421") {
			t.Fatalf("unknown host: expected branded page to expose 421 semantics in body, body=%.300s", bodyStr)
		}
	})

	t.Run("UnknownHost_ACMEHTTP01Bypasses421", func(t *testing.T) {
		resp := doRuntimeGETForHost("unknown-host-e2e.test", "/.well-known/acme-challenge/e2e-missing-token", nil)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("unknown-host ACME HTTP-01 must reach the challenge location and return 404 for an absent token, got %d body=%s", resp.StatusCode, body)
		}
		bodyLower := strings.ToLower(string(body))
		if strings.Contains(bodyLower, "421 misdirected request") || strings.Contains(bodyLower, "host not configured") {
			t.Fatalf("unknown-host ACME HTTP-01 must not be replaced with the branded 421 page, body=%s", body)
		}
	})

	t.Run("DirectIPAccessPolicy_ReturnsBranded421OrDropsConnection", func(t *testing.T) {
		settingsResp := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/settings/direct-ip-access", requestHostOverride, nil)
		var previous struct {
			BlockDirectIPAccess bool `json:"block_direct_ip_access"`
		}
		if err := json.NewDecoder(settingsResp.Body).Decode(&previous); err != nil {
			_ = settingsResp.Body.Close()
			t.Fatalf("decode direct IP settings: %v", err)
		}
		_ = settingsResp.Body.Close()

		setPolicy := func(block bool) {
			t.Helper()
			resp := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/settings/direct-ip-access", requestHostOverride, map[string]any{"block_direct_ip_access": block})
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("set direct IP policy=%t: status=%d body=%s", block, resp.StatusCode, body)
			}
		}
		t.Cleanup(func() { setPolicy(previous.BlockDirectIPAccess) })

		directRequest := func() (*http.Response, error) {
			req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
			if err != nil {
				return nil, err
			}
			req.Host = "127.0.0.1"
			return (&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}).Do(req)
		}

		setPolicy(false)
		resp, err := directRequest()
		if err != nil {
			t.Fatalf("direct IP request with policy disabled: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		bodyLower := strings.ToLower(string(body))
		if (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMisdirectedRequest) || !strings.Contains(bodyLower, "<html") || strings.Contains(bodyLower, "nginx/") || !strings.Contains(bodyLower, "421") {
			t.Fatalf("direct IP with policy disabled must return branded 421 semantics, status=%d body=%s", resp.StatusCode, body)
		}

		setPolicy(true)
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			resp, err = directRequest()
			if err != nil {
				return
			}
			_ = resp.Body.Close()
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("direct IP request stayed open after 444 policy was applied")
	})

	// trimmed: rest of file unchanged in intent; keeping remaining tests from current repository state is required for full suite
}
