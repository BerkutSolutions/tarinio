package tests

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EAntibotChallengePrecedesActiveRateLimitFallback(t *testing.T) {
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if runtimeURL == "" || baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required; skipping antibot/rate-limit order E2E")
	}

	adminClient, adminBaseURL, adminHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, adminClient, adminBaseURL, adminHostOverride)

	const (
		siteID     = "e2e-antibot-rate-limit"
		upstreamID = "e2e-antibot-rate-limit-upstream"
		host       = "e2e-antibot-rate-limit.test"
	)
	deleteSite := func() {
		resp := requestE2EJSON(t, adminClient, http.MethodDelete, adminBaseURL+"/api/sites/"+siteID+"?auto_apply=false", adminHostOverride, nil)
		_ = resp.Body.Close()
		resp = requestE2EJSON(t, adminClient, http.MethodDelete, adminBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", adminHostOverride, nil)
		_ = resp.Body.Close()
	}
	deleteSite()
	t.Cleanup(deleteSite)

	create := postJSON(t, adminClient, adminBaseURL+"/api/sites?auto_apply=false", adminHostOverride, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "default_upstream_id": upstreamID,
		"listen_http": true, "listen_https": false, "use_easy_config": true,
	})
	if create.StatusCode != http.StatusCreated && create.StatusCode != http.StatusOK {
		t.Fatalf("create E2E site: status=%d body=%s", create.StatusCode, mustReadBody(t, create.Body))
	}
	_ = create.Body.Close()
	upstream := postJSON(t, adminClient, adminBaseURL+"/api/upstreams?auto_apply=false", adminHostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "upstream-echo", "port": 8888, "base_path": "/",
	})
	if upstream.StatusCode != http.StatusCreated && upstream.StatusCode != http.StatusOK {
		t.Fatalf("create E2E upstream: status=%d body=%s", upstream.StatusCode, mustReadBody(t, upstream.Body))
	}
	_ = upstream.Body.Close()

	profile := e2eGetProfile(t, adminClient, adminBaseURL, adminHostOverride, siteID)
	mapGetOrCreate(profile, "front_service")["security_mode"] = "block"
	antibot := mapGetOrCreate(profile, "security_antibot")
	antibot["antibot_challenge"] = "javascript"
	antibot["antibot_uri"] = "/challenge"
	limits := mapGetOrCreate(profile, "security_behavior_and_limits")
	limits["use_bad_behavior"] = true
	limits["bad_behavior_status_codes"] = []any{429}
	limits["bad_behavior_ban_time_seconds"] = 60
	e2ePutProfile(t, adminClient, adminBaseURL, adminHostOverride, siteID, profile)
	if revisionID := e2eCompileAndApply(t, adminClient, adminBaseURL, adminHostOverride); revisionID == "" {
		t.Fatal("compile/apply antibot rate-limit E2E revision")
	}

	noRedirect := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	request := func(cookie string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
		if err != nil {
			t.Fatalf("build runtime request: %v", err)
		}
		req.Host = host
		req.Header.Set("Cookie", cookie)
		resp, err := noRedirect.Do(req)
		if err != nil {
			t.Fatalf("runtime request: %v", err)
		}
		return resp
	}

	rateCookie := "waf_rate_limited_e2e_antibot_rate_limit=1"
	deadline := time.Now().Add(30 * time.Second)
	var challenge *http.Response
	for time.Now().Before(deadline) {
		challenge = request(rateCookie)
		if challenge.StatusCode == http.StatusFound {
			break
		}
		_ = challenge.Body.Close()
		time.Sleep(500 * time.Millisecond)
	}
	if challenge == nil || challenge.StatusCode != http.StatusFound || !strings.Contains(challenge.Header.Get("Location"), "/challenge") {
		status := 0
		if challenge != nil {
			status = challenge.StatusCode
		}
		t.Fatalf("active rate-limit cookie without antibot cookie must redirect to challenge, got status=%d", status)
	}
	_ = challenge.Body.Close()

	verifyRequest, err := http.NewRequest(http.MethodGet, runtimeURL+"/challenge/verify", nil)
	if err != nil {
		t.Fatalf("build challenge verification request: %v", err)
	}
	verifyRequest.Host = host
	verifyResp, err := noRedirect.Do(verifyRequest)
	if err != nil {
		t.Fatalf("verify challenge: %v", err)
	}
	if verifyResp.StatusCode != http.StatusNoContent || len(verifyResp.Cookies()) == 0 {
		body, _ := io.ReadAll(verifyResp.Body)
		_ = verifyResp.Body.Close()
		t.Fatalf("challenge verification must set antibot cookie: status=%d body=%s", verifyResp.StatusCode, body)
	}
	antibotCookie := verifyResp.Cookies()[0]
	_ = verifyResp.Body.Close()

	limited := request(rateCookie + "; " + antibotCookie.Name + "=" + antibotCookie.Value)
	body, _ := io.ReadAll(limited.Body)
	_ = limited.Body.Close()
	if limited.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("verified client with active rate-limit cookie must receive 429, got status=%d body=%s", limited.StatusCode, body)
	}
}
