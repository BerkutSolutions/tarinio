package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestE2ESiteSettingsRoundtrip проверяет что каждая настройка сайта реально сохраняется
// и попадает в скомпилированный nginx конфиг. Паттерн:
//
//  1. Создать тестовый сайт с минимальными настройками
//  2. Применить "всё включено" профиль → compile+apply → проверить что все поля есть в конфиге
//  3. Применить "всё выключено" профиль → compile+apply → проверить что полей нет
//  4. Снова применить "всё включено" → compile+apply → проверить снова
//  5. Удалить тестовый сайт
//
// Если хоть одно поле не сохранилось — тест падает.
// Требует: WAF_E2E_BASE_URL
func TestE2ESiteSettingsRoundtrip(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping site settings roundtrip e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	const testSiteID = "e2e-roundtrip-site"
	const testUpstreamID = "e2e-roundtrip-upstream"

	// ── cleanup on exit ───────────────────────────────────────────────────────
	t.Cleanup(func() {
		deleteResp := requestE2EJSON(t, client, http.MethodDelete,
			requestBaseURL+"/api/sites/"+testSiteID+"?auto_apply=false", requestHostOverride, nil)
		_ = deleteResp.Body.Close()
		requestE2EJSON(t, client, http.MethodDelete,
			requestBaseURL+"/api/upstreams/"+testUpstreamID+"?auto_apply=false", requestHostOverride, nil)
	})

	// ── step 1: create site + upstream ───────────────────────────────────────
	t.Run("Setup", func(t *testing.T) {
		siteResp := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
			"id":                  testSiteID,
			"primary_host":        "e2e-roundtrip.test",
			"enabled":             true,
			"default_upstream_id": testUpstreamID,
			"listen_http":         true,
			"listen_https":        false,
			"use_easy_config":     true,
		})
		body, _ := io.ReadAll(siteResp.Body)
		_ = siteResp.Body.Close()
		if siteResp.StatusCode != http.StatusCreated && siteResp.StatusCode != http.StatusOK {
			t.Fatalf("create site: status=%d body=%s", siteResp.StatusCode, string(body))
		}

		upstreamResp := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
			"id":               testUpstreamID,
			"site_id":          testSiteID,
			"name":             testUpstreamID,
			"scheme":           "http",
			"host":             "127.0.0.1",
			"port":             9999,
			"base_path":        "/",
			"pass_host_header": true,
		})
		body, _ = io.ReadAll(upstreamResp.Body)
		_ = upstreamResp.Body.Close()
		if upstreamResp.StatusCode != http.StatusCreated && upstreamResp.StatusCode != http.StatusOK {
			t.Fatalf("create upstream: status=%d body=%s", upstreamResp.StatusCode, string(body))
		}
	})

	// ── helpers ───────────────────────────────────────────────────────────────

	defaultResp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+testSiteID, requestHostOverride)
	defer defaultResp.Body.Close()
	if defaultResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(defaultResp.Body)
		t.Fatalf("get default easy profile: status=%d body=%s", defaultResp.StatusCode, string(body))
	}
	var defaultProfile map[string]any
	if err := json.NewDecoder(defaultResp.Body).Decode(&defaultProfile); err != nil {
		t.Fatalf("decode default easy profile: %v", err)
	}

	saveProfile := func(t *testing.T, profile map[string]any) {
		t.Helper()
		resp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+testSiteID, requestHostOverride, mergeE2EProfile(defaultProfile, profile))
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Fatalf("save easy profile: status=%d body=%s", resp.StatusCode, string(body))
		}
	}

	getArtifact := func(t *testing.T, revID, artifact string) string {
		t.Helper()
		url := fmt.Sprintf("%s/api/revisions/%s/artifacts/%s", requestBaseURL, revID, artifact)
		resp := getWithAuth(t, client, url, requestHostOverride)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
			if runtimeContainer == "" {
				runtimeContainer = "waf-e2e-runtime"
			}
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				active, activeErr := exec.Command("docker", "exec", runtimeContainer, "cat", "/var/lib/waf/active/current.json").CombinedOutput()
				if activeErr == nil && strings.Contains(string(active), revID) {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			output, err := exec.Command("docker", "exec", runtimeContainer, "cat", "/etc/waf/current/"+artifact).CombinedOutput()
			if err != nil {
				t.Fatalf("get compiled artifact %s for revision %s: api status=%d body=%s; runtime artifact: %v output=%s", artifact, revID, resp.StatusCode, string(body), err, string(output))
			}
			body = output
		}
		if len(body) == 0 {
			t.Fatalf("compiled artifact %s for revision %s is empty", artifact, revID)
		}
		return string(body)
	}

	assertInConf := func(t *testing.T, conf, needle, feature string) {
		t.Helper()
		if !strings.Contains(conf, needle) {
			t.Errorf("FAIL %s: expected %q in nginx config, not found\nconfig snippet (first 2000 chars):\n%.2000s", feature, needle, conf)
		}
	}

	assertNotInConf := func(t *testing.T, conf, needle, feature string) {
		t.Helper()
		if strings.Contains(conf, needle) {
			t.Errorf("FAIL %s: expected %q to be absent in nginx config, but found\nconfig snippet (first 2000 chars):\n%.2000s", feature, needle, conf)
		}
	}

	compileApplyAndGetConf := func(t *testing.T) (easyConf, siteConf string) {
		t.Helper()
		revID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
		if revID == "" {
			t.Fatal("compile+apply returned empty revision ID")
		}
		easyConf = getArtifact(t, revID, "nginx/easy/"+testSiteID+".conf")
		easyConf += getArtifact(t, revID, "nginx/easy-locations/"+testSiteID+".conf")
		easyConf += getArtifact(t, revID, "nginx/ratelimits/"+testSiteID+".conf")
		siteConf = getArtifact(t, revID, "nginx/sites/"+testSiteID+".conf")
		return
	}

	// ── all-on profile ────────────────────────────────────────────────────────
	allOnProfile := map[string]any{
		"site_id":          testSiteID,
		"front_service":    map[string]any{"server_name": "e2e-roundtrip.test", "certificate_authority_server": "letsencrypt"},
		"upstream_routing": map[string]any{"reverse_proxy_url": "/"},
		"http_behavior":    map[string]any{"allowed_methods": []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, "max_client_size": "50m"},
		"http_headers":     map[string]any{"hsts_enabled": true, "hsts_max_age_seconds": 31536000, "hsts_include_subdomains": true, "hsts_preload": true},
		"security_behavior_and_limits": map[string]any{
			"use_limit_conn": true, "limit_conn_max_http1": 100, "limit_conn_max_http2": 200, "limit_conn_max_http3": 200,
			"use_limit_req": true, "limit_req_rate": "60r/s", "limit_req_url": "/",
			"use_bad_behavior": true, "bad_behavior_status_codes": []int{400, 403, 404, 429}, "bad_behavior_ban_time_seconds": 300, "bad_behavior_threshold": 10, "bad_behavior_count_time_seconds": 60,
			"ban_escalation_scope": "current_site",
		},
		"security_mode":                 "block",
		"allowed_methods":               []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		"max_client_size":               "50m",
		"http2":                         true,
		"hsts_enabled":                  true,
		"hsts_max_age_seconds":          31536000,
		"hsts_include_subdomains":       true,
		"hsts_preload":                  true,
		"send_x_forwarded_for":          true,
		"send_x_forwarded_proto":        true,
		"send_x_real_ip":                true,
		"pass_host_header":              true,
		"use_limit_conn":                true,
		"limit_conn_max_http1":          100,
		"limit_conn_max_http2":          200,
		"use_limit_req":                 true,
		"limit_req_rate":                "60r/s",
		"use_bad_behavior":              true,
		"bad_behavior_status_codes":     []int{400, 403, 404, 429},
		"bad_behavior_ban_time_seconds": 300,
		"bad_behavior_threshold":        10,
		"use_blacklist":                 true,
		"blacklist_ip":                  []string{"203.0.113.1"},
		"blacklist_user_agent":          []string{"badbot/1.0"},
		"blacklist_uri":                 []string{"/wp-admin"},
		"blacklist_country":             []string{"XX"},
		"use_custom_error_pages":        true,
		"disabled_error_pages":          []string{},
		"security_modsecurity": map[string]any{
			"use_modsecurity":             true,
			"use_modsecurity_crs_plugins": false,
		},
		"security_antibot": map[string]any{
			"antibot_challenge":          "javascript",
			"antibot_uri":                "/challenge",
			"scanner_auto_ban_enabled":   true,
			"antibot_challenge_template": "v1",
		},
		"use_hsts":              true,
		"referrer_policy":       "no-referrer-when-downgrade",
		"proxy_cookie_flags":    "* SameSite=Lax",
		"keep_upstream_headers": []string{"*"},
	}

	// ── all-off profile ───────────────────────────────────────────────────────
	allOffProfile := map[string]any{
		"site_id":          testSiteID,
		"front_service":    map[string]any{"server_name": "e2e-roundtrip.test", "certificate_authority_server": "letsencrypt"},
		"upstream_routing": map[string]any{"reverse_proxy_url": "/"},
		"http_behavior":    map[string]any{"allowed_methods": []string{"GET"}, "max_client_size": "10m"},
		"http_headers":     map[string]any{"hsts_enabled": false, "hsts_max_age_seconds": 0, "hsts_include_subdomains": false, "hsts_preload": false},
		"security_behavior_and_limits": map[string]any{
			"limit_conn_max_http1": 100, "limit_conn_max_http2": 100, "limit_conn_max_http3": 100,
			"limit_req_rate": "10r/s", "limit_req_url": "/", "bad_behavior_status_codes": []int{400},
			"bad_behavior_threshold": 1, "bad_behavior_count_time_seconds": 60, "ban_escalation_scope": "current_site",
		},
		"security_mode":          "transparent",
		"allowed_methods":        []string{"GET"},
		"max_client_size":        "10m",
		"hsts_enabled":           false,
		"send_x_forwarded_for":   false,
		"send_x_forwarded_proto": false,
		"send_x_real_ip":         false,
		"use_limit_conn":         false,
		"use_limit_req":          false,
		"use_bad_behavior":       false,
		"use_blacklist":          false,
		"blacklist_ip":           []string{},
		"blacklist_user_agent":   []string{},
		"blacklist_uri":          []string{},
		"blacklist_country":      []string{},
		"use_custom_error_pages": false,
		"security_modsecurity": map[string]any{
			"use_modsecurity": false,
		},
		"security_antibot": map[string]any{
			"antibot_challenge":        "no",
			"scanner_auto_ban_enabled": false,
		},
		"referrer_policy":       "no-referrer-when-downgrade",
		"proxy_cookie_flags":    "* SameSite=Lax",
		"keep_upstream_headers": []string{"*"},
	}

	// ── round 1: all ON ───────────────────────────────────────────────────────
	t.Run("Round1_AllOn", func(t *testing.T) {
		saveProfile(t, allOnProfile)
		easyConf, siteConf := compileApplyAndGetConf(t)

		conf := easyConf + siteConf

		assertInConf(t, conf, "203.0.113.1", "blacklist_ip")
		assertInConf(t, conf, "badbot/1.0", "blacklist_user_agent")
		assertInConf(t, conf, "/wp-admin", "blacklist_uri")
		assertInConf(t, conf, "waf_antibot_guard", "antibot_enabled")
		assertInConf(t, conf, "waf_antibot_scanner_guard", "antibot_scanner_auto_ban")
		assertInConf(t, conf, "Strict-Transport-Security", "hsts")
		assertInConf(t, conf, "max-age=31536000", "hsts_max_age")
		assertInConf(t, conf, "includeSubDomains", "hsts_include_subdomains")
		assertInConf(t, conf, "preload", "hsts_preload")
		assertInConf(t, conf, "proxy_set_header X-Forwarded-For", "x_forwarded_for")
		assertInConf(t, conf, "proxy_set_header X-Forwarded-Proto", "x_forwarded_proto")
		assertInConf(t, conf, "client_max_body_size 50m", "max_client_size")
	})

	// ── round 2: all OFF ──────────────────────────────────────────────────────
	t.Run("Round2_AllOff", func(t *testing.T) {
		saveProfile(t, allOffProfile)
		easyConf, siteConf := compileApplyAndGetConf(t)

		conf := easyConf + siteConf

		assertNotInConf(t, conf, "203.0.113.1", "blacklist_ip_absent")
		assertNotInConf(t, conf, "badbot/1.0", "blacklist_user_agent_absent")
		assertNotInConf(t, conf, "waf_antibot_guard", "antibot_disabled")
		assertNotInConf(t, conf, "Strict-Transport-Security", "hsts_absent")
		assertNotInConf(t, conf, "modsecurity on;", "modsecurity_duplicate_absent")
		// transparent mode: proxy_intercept_errors must be off
		assertInConf(t, siteConf, "proxy_intercept_errors off", "proxy_intercept_errors_off")
	})

	// ── round 3: all ON again ─────────────────────────────────────────────────
	t.Run("Round3_AllOnAgain", func(t *testing.T) {
		saveProfile(t, allOnProfile)
		easyConf, siteConf := compileApplyAndGetConf(t)

		conf := easyConf + siteConf

		// Same checks as round 1 — settings must be restorable
		assertInConf(t, conf, "203.0.113.1", "blacklist_ip_restored")
		assertInConf(t, conf, "waf_antibot_guard", "antibot_restored")
		assertInConf(t, conf, "Strict-Transport-Security", "hsts_restored")
		assertInConf(t, conf, "waf_antibot_scanner_guard", "antibot_scanner_restored")
		// Guard: modsecurity on; must NOT appear more than once per location block
		count := strings.Count(conf, "modsecurity on;")
		if count > strings.Count(conf, "location") {
			t.Errorf("modsecurity on; count (%d) exceeds location block count — possible duplicate", count)
		}
	})

	// ── round 4: security mode transitions ───────────────────────────────────
	// block  → все модули активны и блокируют
	// monitor → всё выключено, только логи (SecRuleEngine DetectionOnly)
	// transparent → чистое проксирование без WAF вообще
	t.Run("Round4_SecurityModes", func(t *testing.T) {
		// Базовый профиль со всеми модулями включёнными — только mode меняем
		baseProfile := map[string]any{
			"site_id":          testSiteID,
			"front_service":    map[string]any{"server_name": "e2e-roundtrip.test", "certificate_authority_server": "letsencrypt"},
			"upstream_routing": map[string]any{"reverse_proxy_url": "/"},
			"http_behavior":    map[string]any{"allowed_methods": []string{"GET", "POST"}, "max_client_size": "10m"},
			"security_behavior_and_limits": map[string]any{
				"use_limit_conn": true, "limit_conn_max_http1": 80, "limit_conn_max_http2": 80, "limit_conn_max_http3": 80,
				"use_limit_req": true, "limit_req_rate": "30r/s", "limit_req_url": "/",
				"use_bad_behavior": true, "bad_behavior_status_codes": []int{400, 403, 429}, "bad_behavior_ban_time_seconds": 300, "bad_behavior_threshold": 10, "bad_behavior_count_time_seconds": 60,
				"ban_escalation_scope": "current_site",
			},
			"allowed_methods":               []string{"GET", "POST"},
			"max_client_size":               "10m",
			"use_limit_conn":                true,
			"limit_conn_max_http1":          80,
			"use_limit_req":                 true,
			"limit_req_rate":                "30r/s",
			"use_bad_behavior":              true,
			"bad_behavior_status_codes":     []int{400, 403, 429},
			"bad_behavior_ban_time_seconds": 300,
			"bad_behavior_threshold":        10,
			"use_blacklist":                 true,
			"blacklist_ip":                  []string{"203.0.113.99"},
			"blacklist_user_agent":          []string{"evilbot/2.0"},
			"blacklist_uri":                 []string{"/evil"},
			"blacklist_country":             []string{"XX"},
			"security_modsecurity": map[string]any{
				"use_modsecurity": true,
			},
			"security_antibot": map[string]any{
				"antibot_challenge":        "javascript",
				"antibot_uri":              "/challenge",
				"scanner_auto_ban_enabled": true,
			},
			"use_custom_error_pages": true,
		}

		withMode := func(mode string) map[string]any {
			p := make(map[string]any, len(baseProfile)+1)
			for k, v := range baseProfile {
				p[k] = v
			}
			p["security_mode"] = mode
			return p
		}

		// ── block: все модули должны активно блокировать ──────────────────────
		t.Run("block", func(t *testing.T) {
			saveProfile(t, withMode("block"))
			easyConf, siteConf := compileApplyAndGetConf(t)
			conf := easyConf + siteConf

			// Rate limits активны
			assertInConf(t, conf, "limit_conn", "block_limit_conn")
			assertInConf(t, conf, "limit_req", "block_limit_req")
			// Bad behavior ban gate активен (ненулевой ban time)
			assertInConf(t, conf, "set $waf_rate_limited_max_age 300", "block_bad_behavior_ban_time")
			// Blacklist блокирует
			assertInConf(t, conf, "203.0.113.99", "block_blacklist_ip")
			assertInConf(t, conf, "evilbot/2.0", "block_blacklist_ua")
			assertInConf(t, conf, "/evil", "block_blacklist_uri")
			// Antibot активен
			assertInConf(t, conf, "waf_antibot_guard", "block_antibot")
			assertInConf(t, conf, "waf_antibot_scanner_guard", "block_scanner_ban")
			// ModSecurity подключён
			assertInConf(t, easyConf, "modsecurity_rules_file", "block_modsec_rules_file")
			// Кастомные страницы ошибок
			assertInConf(t, siteConf, "proxy_intercept_errors on", "block_error_pages_on")
		})

		// ── monitor: WAF наблюдает, но не блокирует ───────────────────────────
		// ModSecurity в DetectionOnly, blacklist/antibot/rate-limit guards убраны,
		// bad-behavior ban time = 0 (не банит)
		t.Run("monitor", func(t *testing.T) {
			saveProfile(t, withMode("monitor"))
			easyConf, siteConf := compileApplyAndGetConf(t)
			conf := easyConf + siteConf

			// Blacklist guards убраны — трафик не блокируется
			assertNotInConf(t, conf, "deny 203.0.113.99", "monitor_no_ip_deny")
			assertNotInConf(t, conf, "if ($waf_blacklist_ua_guard", "monitor_no_ua_guard")
			assertNotInConf(t, conf, "if ($waf_blacklist_uri_guard", "monitor_no_uri_guard")
			// Antibot не блокирует
			assertNotInConf(t, conf, "if ($waf_antibot_guard", "monitor_no_antibot_block")
			// Bad behavior ban gate отключён (max_age=0)
			assertInConf(t, conf, "set $waf_rate_limited_max_age 0", "monitor_bad_behavior_disabled")
			// ModSecurity подключён но в detection-only (логирует, не блокирует)
			// modsecurity_rules_file должен присутствовать для monitor
			assertInConf(t, easyConf, "modsecurity_rules_file", "monitor_modsec_attached")
			// Нет active enforce на antibot
			assertNotInConf(t, conf, "return 302 $scheme://$host/challenge", "monitor_no_antibot_redirect")
		})

		// ── transparent: чистое проксирование, WAF полностью выключен ─────────
		// Нет modsecurity, нет guards, нет error_page override
		t.Run("transparent", func(t *testing.T) {
			saveProfile(t, withMode("transparent"))
			easyConf, siteConf := compileApplyAndGetConf(t)
			conf := easyConf + siteConf

			// Никаких WAF guard переменных
			assertNotInConf(t, conf, "waf_antibot_guard", "transparent_no_antibot")
			assertNotInConf(t, conf, "deny 203.0.113.99", "transparent_no_ip_deny")
			assertNotInConf(t, conf, "if ($waf_blacklist_ua_guard", "transparent_no_ua_guard")
			assertNotInConf(t, conf, "if ($waf_blacklist_uri_guard", "transparent_no_uri_guard")
			// ModSecurity полностью убран
			assertNotInConf(t, conf, "modsecurity on;", "transparent_no_modsec")
			assertNotInConf(t, conf, "modsecurity_rules_file", "transparent_no_modsec_rules")
			// Bad behavior ban gate отключён
			assertInConf(t, conf, "set $waf_rate_limited_max_age 0", "transparent_bad_behavior_disabled")
			// proxy_intercept_errors off — upstream ответы идут без подмены
			assertInConf(t, siteConf, "proxy_intercept_errors off", "transparent_no_error_intercept")
		})
	})
}

func mergeE2EProfile(defaults, overrides map[string]any) map[string]any {
	merged := make(map[string]any, len(defaults)+len(overrides))
	for key, value := range defaults {
		if nested, ok := value.(map[string]any); ok {
			merged[key] = mergeE2EProfile(nested, nil)
			continue
		}
		merged[key] = value
	}
	for key, value := range overrides {
		overrideNested, isNested := value.(map[string]any)
		defaultNested, hasDefaultNested := merged[key].(map[string]any)
		if isNested && hasDefaultNested {
			merged[key] = mergeE2EProfile(defaultNested, overrideNested)
			continue
		}
		merged[key] = value
	}
	return merged
}
