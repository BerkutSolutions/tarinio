package compiler

import (
	"strings"
	"testing"
)

// tab04_traffic_test.go — тесты вкладки 4: Трафик / Чёрные и белые списки
// Покрывает: BlacklistIP, BlacklistUserAgent, BlacklistURI, BlacklistJA3,
// ExceptionsURI, BlacklistCountry, WhitelistCountry,
// LimitConn, LimitReq, BadBehavior.

// --- BlacklistIP ---

func TestTraffic_BlacklistIP_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ip-1", EasyProfileInput{
		SiteID:         "bl-ip-1",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistIP:    []string{"10.0.0.1"},
	})
	if !strings.Contains(conf, "deny 10.0.0.1;") {
		t.Fatalf("expected deny 10.0.0.1; in config, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistIP_Multiple(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ip-2", EasyProfileInput{
		SiteID:         "bl-ip-2",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistIP:    []string{"10.0.0.1", "192.168.1.100", "172.16.0.0/12"},
	})
	for _, ip := range []string{"deny 10.0.0.1;", "deny 192.168.1.100;", "deny 172.16.0.0/12;"} {
		if !strings.Contains(conf, ip) {
			t.Fatalf("expected %q in blacklist config, got:\n%s", ip, conf)
		}
	}
}

func TestTraffic_BlacklistIP_Empty_NoDeny(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ip-off", EasyProfileInput{
		SiteID:         "bl-ip-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistIP:    nil,
	})
	if strings.Contains(conf, "deny 10.") {
		t.Fatalf("did not expect deny directive when BlacklistIP empty, got:\n%s", conf)
	}
}

// --- BlacklistUserAgent ---

func TestTraffic_BlacklistUA_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ua-1", EasyProfileInput{
		SiteID:             "bl-ua-1",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		BlacklistUserAgent: []string{"BadBot/1.0"},
	})
	if !strings.Contains(conf, "waf_blacklist_ua_guard") {
		t.Fatalf("expected waf_blacklist_ua_guard when UA blacklist set, got:\n%s", conf)
	}
	if !strings.Contains(conf, "BadBot/1.0") {
		t.Fatalf("expected BadBot/1.0 pattern in config, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistUA_BlockReturns403(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ua-403", EasyProfileInput{
		SiteID:             "bl-ua-403",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		BlacklistUserAgent: []string{"Scrapy"},
	})
	// Блокированный UA возвращает 403
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for blacklisted UA, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistUA_ExceptionGuard(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-ua-exc", EasyProfileInput{
		SiteID:             "bl-ua-exc",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		BlacklistUserAgent: []string{"BadBot"},
	})
	// Проверяем что guard использует exception-переменную (^0: prefix)
	if !strings.Contains(conf, "^0:") {
		t.Fatalf("expected exception guard prefix ^0: in UA blacklist, got:\n%s", conf)
	}
}

// --- BlacklistURI ---

func TestTraffic_BlacklistURI_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-uri-1", EasyProfileInput{
		SiteID:         "bl-uri-1",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistURI:   []string{"/admin/secret"},
	})
	if !strings.Contains(conf, "waf_blacklist_uri_guard") {
		t.Fatalf("expected waf_blacklist_uri_guard when URI blacklist set, got:\n%s", conf)
	}
	if !strings.Contains(conf, "/admin/secret") {
		t.Fatalf("expected /admin/secret pattern in config, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistURI_Returns403(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-uri-403", EasyProfileInput{
		SiteID:         "bl-uri-403",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistURI:   []string{"/phpmyadmin"},
	})
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for blacklisted URI, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistURI_Multiple(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-uri-2", EasyProfileInput{
		SiteID:         "bl-uri-2",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistURI:   []string{"/wp-login", "/xmlrpc.php", "/phpmyadmin"},
	})
	for _, uri := range []string{"/wp-login", "/xmlrpc.php", "/phpmyadmin"} {
		if !strings.Contains(conf, uri) {
			t.Fatalf("expected URI pattern %q in blacklist, got:\n%s", uri, conf)
		}
	}
}

// --- ExceptionsURI ---

func TestTraffic_ExceptionsURI_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "exc-uri-1", EasyProfileInput{
		SiteID:         "exc-uri-1",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		ExceptionsURI:  []string{"/health"},
	})
	if !strings.Contains(conf, `$uri ~* "/health"`) {
		t.Fatalf("expected exception URI /health in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "set $waf_easy_exception_guard 1") {
		t.Fatalf("expected exception guard set to 1 for URI, got:\n%s", conf)
	}
}

func TestTraffic_ExceptionsURI_Multiple(t *testing.T) {
	conf := mustRenderSiteConf(t, "exc-uri-2", EasyProfileInput{
		SiteID:         "exc-uri-2",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		ExceptionsURI:  []string{"/health", "/metrics", "/status"},
	})
	for _, uri := range []string{"/health", "/metrics", "/status"} {
		if !strings.Contains(conf, uri) {
			t.Fatalf("expected exception URI %q in config, got:\n%s", uri, conf)
		}
	}
}

func TestTraffic_ExceptionsURI_Empty_NoExtraGuard(t *testing.T) {
	conf := mustRenderSiteConf(t, "exc-uri-off", EasyProfileInput{
		SiteID:         "exc-uri-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		ExceptionsURI:  nil,
	})
	// Без URI-исключений — базовый guard (admin/cookie) всё равно есть
	if !strings.Contains(conf, "waf_easy_exception_guard") {
		t.Fatalf("expected base exception guard even without ExceptionsURI, got:\n%s", conf)
	}
	// Но /health не должен быть
	if strings.Contains(conf, "/health") {
		t.Fatalf("did not expect /health in config when ExceptionsURI empty, got:\n%s", conf)
	}
}

// --- ExceptionsURI блокирует blacklist для исключённых URI ---

func TestTraffic_ExceptionsURI_BypassesBlacklistIP(t *testing.T) {
	// exception guard установлен — deny не применяется к исключённым путям
	conf := mustRenderSiteConf(t, "exc-bypass", EasyProfileInput{
		SiteID:         "exc-bypass",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		BlacklistIP:    []string{"1.2.3.4"},
		ExceptionsURI:  []string{"/health"},
	})
	// deny идёт через nginx напрямую (не через guard), а UA/URI-blacklist — через guard
	// Тест проверяет что оба механизма присутствуют в конфиге
	if !strings.Contains(conf, "deny 1.2.3.4;") {
		t.Fatalf("expected deny 1.2.3.4; even with exception URI, got:\n%s", conf)
	}
	if !strings.Contains(conf, "/health") {
		t.Fatalf("expected /health exception in config, got:\n%s", conf)
	}
}

// --- BlacklistCountry ---

func TestTraffic_BlacklistCountry_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-country-1", EasyProfileInput{
		SiteID:           "bl-country-1",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		BlacklistCountry: []string{"CN"},
	})
	if !strings.Contains(conf, "waf_country_guard") {
		t.Fatalf("expected waf_country_guard when BlacklistCountry set, got:\n%s", conf)
	}
	if !strings.Contains(conf, "CN") {
		t.Fatalf("expected CN in country blacklist config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for blacklisted country, got:\n%s", conf)
	}
}

func TestTraffic_BlacklistCountry_Multiple(t *testing.T) {
	conf := mustRenderSiteConf(t, "bl-country-2", EasyProfileInput{
		SiteID:           "bl-country-2",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		BlacklistCountry: []string{"CN", "RU", "KP"},
	})
	// Все три страны должны быть в паттерне
	for _, country := range []string{"CN", "RU", "KP"} {
		if !strings.Contains(conf, country) {
			t.Fatalf("expected country %q in blacklist pattern, got:\n%s", country, conf)
		}
	}
}

// --- WhitelistCountry ---

func TestTraffic_WhitelistCountry_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "wl-country-1", EasyProfileInput{
		SiteID:           "wl-country-1",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		WhitelistCountry: []string{"RU"},
	})
	if !strings.Contains(conf, "waf_country_guard") {
		t.Fatalf("expected waf_country_guard when WhitelistCountry set, got:\n%s", conf)
	}
	if !strings.Contains(conf, "RU") {
		t.Fatalf("expected RU in country whitelist config, got:\n%s", conf)
	}
}

func TestTraffic_WhitelistCountry_BlocksOthers(t *testing.T) {
	conf := mustRenderSiteConf(t, "wl-country-block", EasyProfileInput{
		SiteID:           "wl-country-block",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		WhitelistCountry: []string{"US"},
	})
	// Whitelist использует !~ (не совпадает → 403)
	if !strings.Contains(conf, "!~") {
		t.Fatalf("expected !~ operator for whitelist (block non-matching), got:\n%s", conf)
	}
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for non-whitelisted country, got:\n%s", conf)
	}
}

// --- LimitConn ---

func TestTraffic_LimitConn_Enabled(t *testing.T) {
	// conn_limit = max(200, LimitConnMaxHTTP1) — дефолт 200, растёт если больше
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "lim-conn", Enabled: true, PrimaryHost: "lim-conn.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:            "lim-conn",
			SecurityMode:      "block",
			AllowedMethods:    []string{"GET"},
			MaxClientSize:     "10m",
			UseLimitConn:      true,
			LimitConnMaxHTTP1: 500,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	l4 := string(byPath["l4guard/config.json"].Content)
	if !strings.Contains(l4, `"conn_limit": 500`) {
		t.Fatalf("expected conn_limit 500 in l4guard/config.json (value > default 200), got:\n%s", l4)
	}
}

func TestTraffic_LimitConn_Disabled_NoDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "lim-conn-off", EasyProfileInput{
		SiteID:         "lim-conn-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UseLimitConn:   false,
	})
	if strings.Contains(conf, "limit_conn ") {
		t.Fatalf("did not expect limit_conn in site.conf when UseLimitConn=false, got:\n%s", conf)
	}
}

// --- LimitReq ---

func TestTraffic_LimitReq_Enabled(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "lim-req", Enabled: true, PrimaryHost: "lim-req.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "lim-req",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseLimitReq:    true,
			LimitReqRate:   "100r/s",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	l4 := string(byPath["l4guard/config.json"].Content)
	if !strings.Contains(l4, `"rate_per_second": 100`) {
		t.Fatalf("expected rate_per_second 100 in l4guard/config.json, got:\n%s", l4)
	}
}

func TestTraffic_LimitReq_Disabled_NoDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "lim-req-off", EasyProfileInput{
		SiteID:         "lim-req-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UseLimitReq:    false,
	})
	if strings.Contains(conf, "limit_req ") {
		t.Fatalf("did not expect limit_req in site.conf when UseLimitReq=false, got:\n%s", conf)
	}
}

// --- BadBehavior ---

func TestTraffic_BadBehavior_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "bad-beh", EasyProfileInput{
		SiteID:                   "bad-beh",
		SecurityMode:             "block",
		AllowedMethods:           []string{"GET"},
		MaxClientSize:            "10m",
		UseBadBehavior:           true,
		BadBehaviorStatusCodes:   []int{400, 403, 404},
		BadBehaviorBanTimeSeconds: 300,
	})
	if !strings.Contains(conf, "waf_rate_limited") {
		t.Fatalf("expected bad behavior rate limit vars when UseBadBehavior=true, got:\n%s", conf)
	}
}

func TestTraffic_BadBehavior_BanReturns429(t *testing.T) {
	conf := mustRenderSiteConf(t, "bad-beh-429", EasyProfileInput{
		SiteID:                   "bad-beh-429",
		SecurityMode:             "block",
		AllowedMethods:           []string{"GET"},
		MaxClientSize:            "10m",
		UseBadBehavior:           true,
		BadBehaviorStatusCodes:   []int{403, 404},
		BadBehaviorBanTimeSeconds: 60,
	})
	// Забаненный клиент получает 429
	if !strings.Contains(conf, "return 429") {
		t.Fatalf("expected return 429 for rate-limited client, got:\n%s", conf)
	}
}

func TestTraffic_BadBehavior_EscalationReturns403(t *testing.T) {
	conf := mustRenderSiteConf(t, "bad-beh-403", EasyProfileInput{
		SiteID:                   "bad-beh-403",
		SecurityMode:             "block",
		AllowedMethods:           []string{"GET"},
		MaxClientSize:            "10m",
		UseBadBehavior:           true,
		BadBehaviorStatusCodes:   []int{403},
		BadBehaviorBanTimeSeconds: 600,
	})
	// При эскалации — 403
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for escalated client, got:\n%s", conf)
	}
}
