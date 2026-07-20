package tests

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type antibotEndpoint struct {
	originalBaseURL string
	requestBaseURL  string
	hostOverride    string
	originalParsed  *url.URL
	requestParsed   *url.URL
}

func TestE2EAntiBot_ProfileCutoffAndCookiePersistence(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(firstNonEmpty(os.Getenv("WAF_E2E_ANTIBOT_BASE_URL"), os.Getenv("WAF_E2E_RUNTIME_HTTPS_URL"), os.Getenv("WAF_E2E_BASE_URL"))), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_ANTIBOT_BASE_URL or WAF_E2E_BASE_URL is not set; skipping antibot e2e test")
	}

	challengeURI := normalizeChallengeURI(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_CHALLENGE_URI")))
	verifyURI := antibotVerifyURI(challengeURI)
	hackerPath := normalizePathWithDefault(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_HACKER_PATH")), "/wp-admin/")
	normalPath := normalizePathWithDefault(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_USER_PATH")), "/")

	endpoint, err := resolveAntibotEndpoint(baseURL)
	if err != nil {
		t.Fatalf("resolve antibot endpoint: %v", err)
	}
	var telemetryClient *http.Client
	var telemetryBaseURL, telemetryHostOverride string
	if runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/"); runtimeURL != "" {
		adminClient, adminBaseURL, adminHostOverride := newE2EClientAndBase(t, os.Getenv("WAF_E2E_BASE_URL"))
		telemetryClient, telemetryBaseURL, telemetryHostOverride = adminClient, adminBaseURL, adminHostOverride
		loginE2EUser(t, adminClient, adminBaseURL, adminHostOverride)
		for _, path := range []string{"/api/tls-configs/e2e-antibot-site?auto_apply=false", "/api/sites/e2e-antibot-site?auto_apply=false", "/api/upstreams/e2e-antibot-upstream?auto_apply=false"} {
			resp := requestE2EJSON(t, adminClient, http.MethodDelete, adminBaseURL+path, adminHostOverride, nil)
			_ = resp.Body.Close()
		}
		createE2EModSecuritySite(t, adminClient, adminBaseURL, adminHostOverride, "e2e-antibot-site", "e2e-antibot-upstream", endpoint.hostOverride)
		certificateID := "e2e-antibot-tls"
		issue := postJSON(t, adminClient, adminBaseURL+"/api/certificates/self-signed/issue", adminHostOverride, map[string]any{
			"certificate_id": certificateID, "common_name": endpoint.hostOverride,
		})
		if issue.StatusCode != http.StatusCreated && issue.StatusCode != http.StatusOK {
			t.Fatalf("issue antibot TLS certificate: status=%d body=%s", issue.StatusCode, mustReadBody(t, issue.Body))
		}
		_ = issue.Body.Close()
		existingTLS := requestE2EJSON(t, adminClient, http.MethodDelete, adminBaseURL+"/api/tls-configs/e2e-antibot-site?auto_apply=false", adminHostOverride, nil)
		_ = existingTLS.Body.Close()
		bind := postJSON(t, adminClient, adminBaseURL+"/api/tls-configs?auto_apply=false", adminHostOverride, map[string]any{
			"site_id": "e2e-antibot-site", "certificate_id": certificateID,
		})
		if bind.StatusCode != http.StatusCreated && bind.StatusCode != http.StatusOK {
			t.Fatalf("bind antibot TLS certificate: status=%d body=%s", bind.StatusCode, mustReadBody(t, bind.Body))
		}
		_ = bind.Body.Close()
		// Keep this uniquely named site for the immediately following template
		// scenario. The E2E stack is disposable and the runner removes its
		// volumes afterwards; retaining it lets that scenario exercise the
		// compiled challenge flow rather than skipping it.
		profile := e2eGetProfile(t, adminClient, adminBaseURL, adminHostOverride, "e2e-antibot-site")
		mapGetOrCreate(profile, "front_service")["security_mode"] = "block"
		antibot := mapGetOrCreate(profile, "security_antibot")
		antibot["antibot_challenge"] = "javascript"
		antibot["antibot_uri"] = challengeURI
		e2ePutProfile(t, adminClient, adminBaseURL, adminHostOverride, "e2e-antibot-site", profile)
		e2eCompileAndApply(t, adminClient, adminBaseURL, adminHostOverride)
		time.Sleep(2 * time.Second)
	}

	probeClient := newAntibotHTTPClient(endpoint, false)
	probeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if err := waitForHTTP(probeClient, endpoint.requestBaseURL+normalPath, endpoint.hostOverride, 90*time.Second); err != nil {
		t.Fatalf("target is not ready: %v", err)
	}

	actors := []struct {
		name                 string
		userAgent            string
		targets              []string
		executesChallenge    bool
		persistCookies       bool
		expectBypassAfterSet bool
	}{
		{
			name:                 "human-regular",
			userAgent:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			targets:              []string{normalPath, "/dashboard"},
			executesChallenge:    true,
			persistCookies:       true,
			expectBypassAfterSet: true,
		},
		{
			name:                 "human-protected-site-login",
			userAgent:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			targets:              []string{"/auth/login?next=%2Faccount"},
			executesChallenge:    true,
			persistCookies:       true,
			expectBypassAfterSet: true,
		},
		{
			name:                 "human-hacker",
			userAgent:            "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			targets:              []string{hackerPath, normalPath},
			executesChallenge:    true,
			persistCookies:       true,
			expectBypassAfterSet: true,
		},
		{
			name:              "bot-curl",
			userAgent:         "curl/8.7.1",
			targets:           []string{normalPath},
			executesChallenge: false,
			persistCookies:    false,
		},
		{
			name:              "bot-python-requests",
			userAgent:         "python-requests/2.32.3",
			targets:           []string{normalPath},
			executesChallenge: false,
			persistCookies:    false,
		},
		{
			name:                 "bot-scanner-direct-verify-no-jar",
			userAgent:            "sqlmap/1.8.4#stable (https://sqlmap.org)",
			targets:              []string{hackerPath},
			executesChallenge:    true,
			persistCookies:       false,
			expectBypassAfterSet: false,
		},
	}

	for _, actor := range actors {
		actor := actor
		t.Run(actor.name, func(t *testing.T) {
			client := newAntibotHTTPClient(endpoint, actor.persistCookies)
			for targetIndex, targetPath := range actor.targets {
				targetPath = normalizePathWithDefault(strings.TrimSpace(targetPath), "/")
				targetURL := endpoint.requestBaseURL + targetPath

				firstResp := antibotDoRequest(t, client, endpoint, http.MethodGet, targetURL, actor.userAgent, false)
				challengeLocation, challenged := extractChallengeLocation(firstResp, challengeURI)
				if !challenged {
					if actor.persistCookies && targetIndex > 0 {
						continue
					}
					t.Fatalf("expected challenge redirect for actor=%s path=%s status=%d location=%q", actor.name, targetPath, firstResp.StatusCode, firstResp.Header.Get("Location"))
				}
				challengeLocation = localAntibotLocation(challengeLocation)
				if strings.HasPrefix(targetPath, "/auth/login") {
					challengeURL, err := url.Parse(challengeLocation)
					if err != nil || challengeURL.Query().Get("return_uri") != "/auth/login" || challengeURL.Query().Get("return_args") != "next=/account" {
						t.Fatalf("challenge must retain protected login target, got %q", challengeLocation)
					}
				}

				if !actor.executesChallenge {
					secondResp := antibotDoRequest(t, client, endpoint, http.MethodGet, targetURL, actor.userAgent, false)
					if _, stillChallenged := extractChallengeLocation(secondResp, challengeURI); !stillChallenged {
						t.Fatalf("expected actor=%s to stay challenged without verification on %s", actor.name, targetPath)
					}
					continue
				}

				challengePageURL := absolutizeLocation(endpoint.requestBaseURL, challengeLocation)
				challengeResp := antibotDoRequest(t, client, endpoint, http.MethodGet, challengePageURL, actor.userAgent, false)
				if challengeResp.StatusCode != http.StatusOK {
					t.Fatalf("expected challenge page status=200 for actor=%s path=%s, got=%d", actor.name, targetPath, challengeResp.StatusCode)
				}
				challengeBody := readAndClose(t, challengeResp.Body)
				if !strings.Contains(strings.ToLower(challengeBody), "verification") && !strings.Contains(strings.ToLower(challengeBody), "challenge") {
					t.Fatalf("challenge page contract mismatch for actor=%s path=%s", actor.name, targetPath)
				}

				verifyURL, err := buildVerifyURL(endpoint.requestBaseURL, challengeLocation, verifyURI)
				if err != nil {
					t.Fatalf("build verify url for actor=%s path=%s: %v", actor.name, targetPath, err)
				}
				verifyResp := antibotDoRequest(t, client, endpoint, http.MethodGet, verifyURL, actor.userAgent, false)
				if verifyResp.StatusCode != http.StatusNoContent {
					t.Fatalf("expected interstitial verify to set the cookie without redirecting, got=%d", verifyResp.StatusCode)
				}
				_ = verifyResp.Body.Close()

				postVerifyResp := antibotDoRequest(t, client, endpoint, http.MethodGet, targetURL, actor.userAgent, false)
				_, challengedAfterVerify := extractChallengeLocation(postVerifyResp, challengeURI)
				if actor.expectBypassAfterSet && challengedAfterVerify {
					t.Fatalf("expected actor=%s to pass antibot after verify on %s", actor.name, targetPath)
				}
				if !actor.expectBypassAfterSet && !challengedAfterVerify {
					t.Fatalf("expected actor=%s to stay challenged without cookie persistence on %s", actor.name, targetPath)
				}
			}

			if actor.persistCookies {
				cookies := client.Jar.Cookies(endpoint.originalParsed)
				if len(cookies) == 0 {
					cookies = client.Jar.Cookies(endpoint.requestParsed)
				}
				if !hasCookiePrefix(cookies, "waf_antibot_") {
					t.Fatalf("expected persisted antibot cookie for actor=%s", actor.name)
				}
			}
		})
	}
	if telemetryClient != nil {
		blocked := antibotDoRequest(t, probeClient, endpoint, http.MethodPost, endpoint.requestBaseURL+normalPath, "curl/8.7.1", false)
		if blocked.StatusCode != http.StatusForbidden {
			_ = blocked.Body.Close()
			t.Fatalf("unverified unsafe antibot request must be blocked with 403, got status=%d", blocked.StatusCode)
		}
		_ = blocked.Body.Close()
		e2eWaitForRequestTelemetry(t, telemetryClient, telemetryBaseURL, telemetryHostOverride, normalPath, http.StatusForbidden, "antibot", "security")
	}
}

func resolveAntibotEndpoint(baseURL string) (antibotEndpoint, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return antibotEndpoint{}, err
	}
	out := antibotEndpoint{
		originalBaseURL: baseURL,
		requestBaseURL:  baseURL,
		originalParsed:  parsed,
		requestParsed:   parsed,
	}
	if strings.EqualFold(parsed.Hostname(), "localhost") || net.ParseIP(parsed.Hostname()).IsLoopback() {
		hostOverride := strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_HOST"))
		if hostOverride == "" {
			hostOverride = parsed.Hostname()
		}
		requestParsed := &url.URL{}
		*requestParsed = *parsed
		requestParsed.Host = net.JoinHostPort(hostOverride, effectivePort(parsed))
		out.hostOverride = hostOverride
		out.requestParsed = requestParsed
		out.requestBaseURL = strings.TrimRight(requestParsed.String(), "/")
	}
	return out, nil
}

func newAntibotHTTPClient(endpoint antibotEndpoint, withJar bool) *http.Client {
	client := newE2EHTTPClient(endpoint.requestBaseURL, withJar)
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	} else {
		transport = transport.Clone()
	}
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err == nil && strings.EqualFold(host, endpoint.hostOverride) {
			return dialer.DialContext(ctx, network, net.JoinHostPort("127.0.0.1", port))
		}
		return dialer.DialContext(ctx, network, addr)
	}
	client.Transport = transport
	return client
}

func newE2EHTTPClient(baseURL string, withJar bool) *http.Client {
	client := &http.Client{Timeout: 20 * time.Second}
	if withJar {
		jar, err := cookiejar.New(nil)
		if err == nil {
			client.Jar = jar
		}
	}

	if strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		transport := &http.Transport{
			Proxy:                 nil,
			ForceAttemptHTTP2:     false,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         "localhost",
			},
		}
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return dialer.DialContext(ctx, network, addr)
			}
			if strings.EqualFold(host, "localhost") {
				host = "127.0.0.1"
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		}
		client.Transport = transport
	}
	return client
}

func antibotDoRequest(t *testing.T, client *http.Client, endpoint antibotEndpoint, method, targetURL, userAgent string, follow bool) *http.Response {
	t.Helper()
	requestClient := client
	if !follow {
		tmp := *client
		tmp.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		requestClient = &tmp
	}
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		t.Fatalf("create request %s %s: %v", method, targetURL, err)
	}
	req.Header.Set("Accept", "text/html,application/json")
	if strings.TrimSpace(userAgent) != "" {
		req.Header.Set("User-Agent", strings.TrimSpace(userAgent))
	}
	if endpoint.hostOverride != "" {
		req.Host = endpoint.hostOverride
	}
	resp, err := requestClient.Do(req)
	if err != nil {
		t.Fatalf("request failed %s %s: %v", method, targetURL, err)
	}
	return resp
}

func extractChallengeLocation(resp *http.Response, challengeURI string) (string, bool) {
	defer func() { _ = resp.Body.Close() }()
	location := strings.TrimSpace(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusFound || location == "" {
		return "", false
	}
	if strings.Contains(location, challengeURI+"?") || strings.HasSuffix(location, challengeURI) || strings.Contains(location, challengeURI+"/") {
		return location, true
	}
	return "", false
}

func localAntibotLocation(location string) string {
	parsed, err := url.Parse(strings.TrimSpace(location))
	if err != nil || !parsed.IsAbs() {
		return location
	}
	return parsed.RequestURI()
}

func buildVerifyURL(baseURL, challengeLocation, verifyURI string) (string, error) {
	challengeURL, err := url.Parse(absolutizeLocation(baseURL, challengeLocation))
	if err != nil {
		return "", err
	}
	verifyURL, err := url.Parse(absolutizeLocation(baseURL, verifyURI))
	if err != nil {
		return "", err
	}
	query := verifyURL.Query()
	if value := strings.TrimSpace(challengeURL.Query().Get("return_uri")); value != "" {
		query.Set("return_uri", value)
	}
	if value := strings.TrimSpace(challengeURL.Query().Get("return_args")); value != "" {
		query.Set("return_args", value)
	}
	verifyURL.RawQuery = query.Encode()
	return verifyURL.String(), nil
}

func absolutizeLocation(baseURL, location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return strings.TrimRight(baseURL, "/")
	}
	parsed, err := url.Parse(location)
	if err == nil && parsed.IsAbs() {
		return parsed.String()
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(location, "/")
}

func hasCookiePrefix(cookies []*http.Cookie, prefix string) bool {
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(cookie.Name), prefix) && strings.TrimSpace(cookie.Value) != "" {
			return true
		}
	}
	return false
}

func normalizeChallengeURI(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/challenge"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func normalizePathWithDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func antibotVerifyURI(challengeURI string) string {
	trimmed := strings.TrimSpace(challengeURI)
	if trimmed == "" || trimmed == "/" {
		return "/challenge/verify"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed + "/verify"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func readAndClose(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(raw)
}
