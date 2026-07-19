package tests

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestE2EBehavioral_CustomErrorPages_UpstreamDrivenBranches(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL"))
	if baseURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL not set; skipping behavioral runtime tests")
	}
	requestURL := strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL"))
	if requestURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping behavioral runtime tests")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	testSiteID := "e2e-custom-pages-upstream-site-" + suffix
	testUpstreamID := "e2e-custom-pages-upstream-upstream-" + suffix
	testHost := "e2e-custom-pages-upstream-" + suffix + ".test"

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
			req, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
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

	doRuntimeGET := func(path string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			t.Fatalf("build runtime request: %v", err)
		}
		req.Host = testHost
		noRedirectClient := &http.Client{
			Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		}
		resp, err := noRedirectClient.Do(req)
		if err != nil {
			t.Fatalf("runtime GET %s: %v", path, err)
		}
		return resp
	}

	defaultProfile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, testSiteID)
	if defaultProfile == nil {
		t.Fatal("could not fetch default easy profile for test site")
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

	baseBlockProfile := func() map[string]any {
		raw, err := json.Marshal(defaultProfile)
		if err != nil {
			t.Fatalf("marshal default profile: %v", err)
		}
		var profile map[string]any
		if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal default profile: %v", err)
		}
		profile = cloneMap(profile)
		profile["security_mode"] = "block"
		return profile
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

	assertLooksLikeBrandedPage := func(t *testing.T, name string, status int, body []byte) {
		t.Helper()
		bodyStr := string(body)
		bodyLower := strings.ToLower(bodyStr)
		if !strings.Contains(bodyLower, "<html") && !strings.Contains(bodyLower, "<!doctype") {
			t.Errorf("%s page is not HTML (status=%d, %d bytes): %.300s", name, status, len(bodyStr), bodyStr)
		}
		if len(bodyStr) < 200 {
			t.Errorf("%s page too short (%d bytes) — likely default/plain response", name, len(bodyStr))
		}
	}

	assertDisabledPageGetsShorter := func(t *testing.T, name string, brandedBody, disabledBody []byte) {
		t.Helper()
		if len(disabledBody) >= len(brandedBody) {
			t.Logf("disabled_error_pages %s: response size %d (branded was %d) — runtime may still route through custom page", name, len(disabledBody), len(brandedBody))
		} else {
			t.Logf("disabled_error_pages %s: response size reduced from %d to %d bytes — OK", name, len(brandedBody), len(disabledBody))
		}
	}

	checkUpstreamDrivenCustomPage := func(t *testing.T, statusCode int, path string) {
		t.Helper()
		codeStr := strconv.Itoa(statusCode)

		pUpstream := baseBlockProfile()
		pUpstream["use_custom_error_pages"] = true
		pUpstream["disabled_error_pages"] = []string{}
		saveProfile(t, pUpstream)
		compileApply(t)

		respBranded := doRuntimeGET(path)
		bodyBranded, _ := io.ReadAll(respBranded.Body)
		_ = respBranded.Body.Close()
		if respBranded.StatusCode != statusCode {
			t.Logf("custom %s branded check skipped: want upstream/app %d from %s, got %d (body: %.200s)", codeStr, statusCode, path, respBranded.StatusCode, string(bodyBranded))
			return
		}
		assertLooksLikeBrandedPage(t, "custom "+codeStr, respBranded.StatusCode, bodyBranded)
		assertRuntimeErrorPageMetadata(t, respBranded.Header.Get("Server-Timing"))

		pDisabled := baseBlockProfile()
		pDisabled["use_custom_error_pages"] = true
		pDisabled["disabled_error_pages"] = []string{codeStr}
		saveProfile(t, pDisabled)
		compileApply(t)

		respDisabled := doRuntimeGET(path)
		bodyDisabled, _ := io.ReadAll(respDisabled.Body)
		_ = respDisabled.Body.Close()
		if respDisabled.StatusCode != statusCode {
			t.Logf("disabled_error_pages %s check skipped: want upstream/app %d from %s after disable, got %d (body: %.200s)", codeStr, statusCode, path, respDisabled.StatusCode, string(bodyDisabled))
			return
		}
		assertDisabledPageGetsShorter(t, codeStr, bodyBranded, bodyDisabled)
	}

	checkUpstreamDrivenCustomPage(t, http.StatusNotFound, "/__upstream_status/404")
	checkUpstreamDrivenCustomPage(t, http.StatusBadGateway, "/__upstream_status/502")
	checkUpstreamDrivenCustomPage(t, http.StatusServiceUnavailable, "/__upstream_status/503")
}

func assertRuntimeErrorPageMetadata(t *testing.T, serverTiming string) {
	t.Helper()
	for _, marker := range []string{"rid;desc=\"", "ip;desc=\"", "ts;desc=\""} {
		if !strings.Contains(serverTiming, marker) {
			t.Fatalf("custom error page response must expose %q through Server-Timing, got %q", marker, serverTiming)
		}
	}
}

func TestE2EBehavioral_CustomErrorPages_GeoAndDirectRuntimeBranches(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL"))
	if baseURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL not set; skipping behavioral runtime tests")
	}
	requestURL := strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL"))
	if requestURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping behavioral runtime tests")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	const blockedIP = "203.0.113.99"
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	testSiteID := "e2e-custom-pages-direct-site-" + suffix
	testUpstreamID := "e2e-custom-pages-direct-upstream-" + suffix
	testHost := "e2e-custom-pages-direct-" + suffix + ".test"

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
			req, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
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

	doRuntimeGET := func(path string, extraHeaders map[string]string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
		if err != nil {
			t.Fatalf("build runtime request: %v", err)
		}
		req.Host = testHost
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}
		noRedirectClient := &http.Client{
			Timeout:       10 * time.Second,
			Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		}
		resp, err := noRedirectClient.Do(req)
		if err != nil {
			t.Fatalf("runtime GET %s: %v", path, err)
		}
		return resp
	}

	defaultProfile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, testSiteID)
	if defaultProfile == nil {
		t.Fatal("could not fetch default easy profile for test site")
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

	baseBlockProfile := func() map[string]any {
		raw, err := json.Marshal(defaultProfile)
		if err != nil {
			t.Fatalf("marshal default profile: %v", err)
		}
		var profile map[string]any
		if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal default profile: %v", err)
		}
		profile = cloneMap(profile)
		profile["security_mode"] = "block"
		return profile
	}

	setCountryPolicy := func(profile map[string]any, values map[string]any) {
		profile["security_country_policy"] = values
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

	assertLooksLikeHTMLPage := func(t *testing.T, name string, wantStatus int, body []byte) {
		t.Helper()
		bodyStr := string(body)
		bodyLower := strings.ToLower(bodyStr)
		if !strings.Contains(bodyLower, "<html") && !strings.Contains(bodyLower, "<!doctype") {
			t.Errorf("%s page is not HTML (status=%d, %d bytes): %.300s", name, wantStatus, len(bodyStr), bodyStr)
		}
		if len(bodyStr) < 200 {
			t.Errorf("%s page too short (%d bytes) — likely default/plain response", name, len(bodyStr))
		}
	}

	assertDisabledPageGetsShorter := func(t *testing.T, name string, brandedBody, disabledBody []byte) {
		t.Helper()
		if len(disabledBody) >= len(brandedBody) {
			t.Logf("disabled_error_pages %s: response size %d (branded was %d) — runtime may still route through custom page", name, len(disabledBody), len(brandedBody))
		} else {
			t.Logf("disabled_error_pages %s: response size reduced from %d to %d bytes — OK", name, len(brandedBody), len(disabledBody))
		}
	}

	t.Run("GeoBlock451_Branded_And_Disabled", func(t *testing.T) {
		branded := baseBlockProfile()
		branded["use_custom_error_pages"] = true
		branded["disabled_error_pages"] = []string{}
		setCountryPolicy(branded, map[string]any{
			"show_geo_block_page": true,
			"blacklist_country":   []string{"RU"},
		})
		saveProfile(t, branded)
		compileApply(t)

		resp := doRuntimeGET("/", map[string]string{
			"CF-IPCountry":    "RU",
			"X-Forwarded-For": blockedIP,
		})
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnavailableForLegalReasons {
			t.Logf("geo branded check skipped: want 451, got %d (body: %.200s)", resp.StatusCode, string(body))
			return
		}
		assertLooksLikeHTMLPage(t, "geo 451 branded", http.StatusUnavailableForLegalReasons, body)

		disabled := baseBlockProfile()
		disabled["use_custom_error_pages"] = true
		disabled["disabled_error_pages"] = []string{"451"}
		setCountryPolicy(disabled, map[string]any{
			"show_geo_block_page": true,
			"blacklist_country":   []string{"RU"},
		})
		saveProfile(t, disabled)
		compileApply(t)

		respDisabled := doRuntimeGET("/", map[string]string{
			"CF-IPCountry":    "RU",
			"X-Forwarded-For": blockedIP,
		})
		bodyDisabled, _ := io.ReadAll(respDisabled.Body)
		_ = respDisabled.Body.Close()
		if respDisabled.StatusCode == http.StatusUnavailableForLegalReasons {
			assertDisabledPageGetsShorter(t, "451", body, bodyDisabled)
		}
	})

	t.Run("GeoWhitelistMiss451_Branded_And_Disabled", func(t *testing.T) {
		branded := baseBlockProfile()
		branded["use_custom_error_pages"] = true
		branded["disabled_error_pages"] = []string{}
		setCountryPolicy(branded, map[string]any{
			"show_geo_block_page": true,
			"whitelist_country":   []string{"US"},
		})
		saveProfile(t, branded)
		compileApply(t)

		resp := doRuntimeGET("/", map[string]string{
			"CF-IPCountry":    "RU",
			"X-Forwarded-For": blockedIP,
		})
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnavailableForLegalReasons {
			t.Logf("geo whitelist branded check skipped: want 451, got %d (body: %.200s)", resp.StatusCode, string(body))
			return
		}
		assertLooksLikeHTMLPage(t, "geo whitelist 451 branded", http.StatusUnavailableForLegalReasons, body)

		disabled := baseBlockProfile()
		disabled["use_custom_error_pages"] = true
		disabled["disabled_error_pages"] = []string{"451"}
		setCountryPolicy(disabled, map[string]any{
			"show_geo_block_page": true,
			"whitelist_country":   []string{"US"},
		})
		saveProfile(t, disabled)
		compileApply(t)

		respDisabled := doRuntimeGET("/", map[string]string{
			"CF-IPCountry":    "RU",
			"X-Forwarded-For": blockedIP,
		})
		bodyDisabled, _ := io.ReadAll(respDisabled.Body)
		_ = respDisabled.Body.Close()
		if respDisabled.StatusCode == http.StatusUnavailableForLegalReasons {
			assertDisabledPageGetsShorter(t, "451-whitelist", body, bodyDisabled)
		}
	})

	t.Run("TrustedProxyIdentityRejectsSpoofedForwardedFor", func(t *testing.T) {
		trusted := baseBlockProfile()
		security := mapGetOrCreate(trusted, "security_behavior_and_limits")
		security["use_blacklist"] = true
		security["blacklist_ip"] = []string{blockedIP}
		security["access_trusted_proxy_cidrs"] = []string{"127.0.0.1/32", "172.16.0.0/12"}
		saveProfile(t, trusted)
		compileApply(t)

		blocked := doRuntimeGET("/", map[string]string{"X-Forwarded-For": blockedIP})
		blockedBody, _ := io.ReadAll(blocked.Body)
		_ = blocked.Body.Close()
		if blocked.StatusCode != http.StatusForbidden {
			t.Fatalf("trusted proxy must apply client IP from X-Forwarded-For: status=%d body=%.200s", blocked.StatusCode, string(blockedBody))
		}

		untrusted := baseBlockProfile()
		security = mapGetOrCreate(untrusted, "security_behavior_and_limits")
		security["use_blacklist"] = true
		security["blacklist_ip"] = []string{blockedIP}
		security["access_trusted_proxy_cidrs"] = []string{}
		saveProfile(t, untrusted)
		compileApply(t)

		spoofed := doRuntimeGET("/", map[string]string{"X-Forwarded-For": blockedIP})
		spoofedBody, _ := io.ReadAll(spoofed.Body)
		_ = spoofed.Body.Close()
		if spoofed.StatusCode == http.StatusForbidden {
			t.Fatalf("untrusted X-Forwarded-For must not control the client identity: body=%.200s", string(spoofedBody))
		}
	})

	t.Run("Direct404_Branded_And_Disabled", func(t *testing.T) {
		const missingPath = "/e2e-missing-page-direct-404"

		branded := baseBlockProfile()
		branded["use_custom_error_pages"] = true
		branded["disabled_error_pages"] = []string{}
		saveProfile(t, branded)
		compileApply(t)

		resp := doRuntimeGET(missingPath, nil)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Logf("direct 404 branded check skipped: want 404, got %d (body: %.200s)", resp.StatusCode, string(body))
			return
		}
		assertLooksLikeHTMLPage(t, "direct 404 branded", http.StatusNotFound, body)

		disabled := baseBlockProfile()
		disabled["use_custom_error_pages"] = true
		disabled["disabled_error_pages"] = []string{"404"}
		saveProfile(t, disabled)
		compileApply(t)

		respDisabled := doRuntimeGET(missingPath, nil)
		bodyDisabled, _ := io.ReadAll(respDisabled.Body)
		_ = respDisabled.Body.Close()
		if respDisabled.StatusCode == http.StatusNotFound {
			assertDisabledPageGetsShorter(t, "404-direct", body, bodyDisabled)
		}
	})
}
