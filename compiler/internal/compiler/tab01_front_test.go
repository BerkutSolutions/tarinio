package compiler

import (
	"strings"
	"testing"
)

// tab01_front_test.go — тесты вкладки 1: Фронтовый сервис
// Покрывает: HSTS, AllowedMethods, HttpStrictParsing, mTLS, TLS-сниппеты в конфиге.

// --- HSTS ---

func TestFront_HSTS_FullDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "hsts-site", EasyProfileInput{
		SiteID:                "hsts-site",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET", "POST"},
		MaxClientSize:         "10m",
		HSTSEnabled:           true,
		HSTSMaxAgeSeconds:     31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	})
	want := `add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;`
	if !strings.Contains(conf, want) {
		t.Fatalf("HSTS full directive not found in site conf.\nwant substring: %s\ngot:\n%s", want, conf)
	}
}

func TestFront_HSTS_MaxAgeOnly(t *testing.T) {
	conf := mustRenderSiteConf(t, "hsts-maxage", EasyProfileInput{
		SiteID:                "hsts-maxage",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		HSTSEnabled:           true,
		HSTSMaxAgeSeconds:     7776000,
		HSTSIncludeSubdomains: false,
		HSTSPreload:           false,
	})
	if !strings.Contains(conf, `max-age=7776000"`) {
		t.Fatalf("expected max-age=7776000 in HSTS header, got:\n%s", conf)
	}
	if strings.Contains(conf, "includeSubDomains") {
		t.Fatalf("did not expect includeSubDomains when disabled, got:\n%s", conf)
	}
	if strings.Contains(conf, "preload") {
		t.Fatalf("did not expect preload when disabled, got:\n%s", conf)
	}
}

func TestFront_HSTS_Disabled_NoHeader(t *testing.T) {
	conf := mustRenderSiteConf(t, "hsts-off", EasyProfileInput{
		SiteID:         "hsts-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		HSTSEnabled:    false,
	})
	if strings.Contains(conf, "Strict-Transport-Security") {
		t.Fatalf("expected no HSTS header when disabled, got:\n%s", conf)
	}
}

func TestFront_HSTS_DefaultMaxAge_WhenZero(t *testing.T) {
	// When HSTSMaxAgeSeconds=0, compiler should apply default (15552000).
	conf := mustRenderSiteConf(t, "hsts-defage", EasyProfileInput{
		SiteID:            "hsts-defage",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET"},
		MaxClientSize:     "10m",
		HSTSEnabled:       true,
		HSTSMaxAgeSeconds: 0,
	})
	if !strings.Contains(conf, "max-age=15552000") {
		t.Fatalf("expected default max-age=15552000 when HSTSMaxAgeSeconds=0, got:\n%s", conf)
	}
}

// --- AllowedMethods ---

func TestFront_AllowedMethods_LimitedSet(t *testing.T) {
	conf := mustRenderSiteConf(t, "methods-site", EasyProfileInput{
		SiteID:         "methods-site",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET", "POST"},
		MaxClientSize:  "10m",
	})
	if !strings.Contains(conf, "GET|POST") && !strings.Contains(conf, "GET POST") {
		t.Fatalf("expected GET|POST in allowed methods pattern, got:\n%s", conf)
	}
	if strings.Contains(conf, "DELETE") || strings.Contains(conf, "PUT") {
		t.Fatalf("unexpected method DELETE or PUT in conf with limited set, got:\n%s", conf)
	}
}

func TestFront_AllowedMethods_DefaultWhenEmpty(t *testing.T) {
	// Empty AllowedMethods should fall back to the full default set.
	conf := mustRenderSiteConf(t, "methods-default", EasyProfileInput{
		SiteID:         "methods-default",
		SecurityMode:   "block",
		AllowedMethods: []string{},
		MaxClientSize:  "10m",
	})
	for _, m := range []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"} {
		if !strings.Contains(conf, m) {
			t.Fatalf("expected default method %s in conf when AllowedMethods is empty, got:\n%s", m, conf)
		}
	}
}

func TestFront_AllowedMethods_BlocksOtherMethods(t *testing.T) {
	conf := mustRenderSiteConf(t, "methods-block", EasyProfileInput{
		SiteID:         "methods-block",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
	})
	// Should include a method-check block that returns 405 for non-allowed.
	if !strings.Contains(conf, "405") {
		t.Fatalf("expected 405 for disallowed methods, got:\n%s", conf)
	}
}

// --- MaxClientSize ---

func TestFront_MaxClientSize_IsSet(t *testing.T) {
	conf := mustRenderSiteConf(t, "maxsize-site", EasyProfileInput{
		SiteID:         "maxsize-site",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET", "POST"},
		MaxClientSize:  "25m",
	})
	if !strings.Contains(conf, "client_max_body_size 25m;") {
		t.Fatalf("expected client_max_body_size 25m; in conf, got:\n%s", conf)
	}
}

// --- HttpStrictParsing ---
func TestFront_HttpStrictParsing_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "strict-on", EasyProfileInput{
		SiteID:            "strict-on",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET", "POST"},
		MaxClientSize:     "10m",
		HttpStrictParsing: true,
	})
	for _, directive := range []string{
		"ignore_invalid_headers on;",
		"underscores_in_headers off;",
		`proxy_set_header Transfer-Encoding "";`,
	} {
		if !strings.Contains(conf, directive) {
			t.Fatalf("expected directive %q when HttpStrictParsing=true, got:\n%s", directive, conf)
		}
	}
}

func TestFront_HttpStrictParsing_Disabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "strict-off", EasyProfileInput{
		SiteID:            "strict-off",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET", "POST"},
		MaxClientSize:     "10m",
		HttpStrictParsing: false,
	})
	for _, directive := range []string{
		"underscores_in_headers off;",
	} {
		if strings.Contains(conf, directive) {
			t.Fatalf("directive %q must NOT be present when HttpStrictParsing=false, got:\n%s", directive, conf)
		}
	}
}

// --- mTLS (клиентские сертификаты) ---

func TestFront_MTLS_Required_DirectivesPresent(t *testing.T) {
	conf := mustRenderSiteConf(t, "mtls-req", EasyProfileInput{
		SiteID:         "mtls-req",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET", "POST"},
		MaxClientSize:  "10m",
		MTLS: MTLSInput{
			MTLSEnabled:     true,
			MTLSOptional:    false,
			MTLSClientCARef: "/etc/ssl/ca.crt",
			MTLSVerifyDepth: 2,
		},
	})
	if !strings.Contains(conf, "ssl_client_certificate /etc/ssl/ca.crt;") {
		t.Fatalf("expected ssl_client_certificate directive, got:\n%s", conf)
	}
	if !strings.Contains(conf, "ssl_verify_client on;") {
		t.Fatalf("expected ssl_verify_client on; for required mTLS, got:\n%s", conf)
	}
	if !strings.Contains(conf, "ssl_verify_depth 2;") {
		t.Fatalf("expected ssl_verify_depth 2; got:\n%s", conf)
	}
}

func TestFront_MTLS_Optional_DirectivesPresent(t *testing.T) {
	conf := mustRenderSiteConf(t, "mtls-opt", EasyProfileInput{
		SiteID:         "mtls-opt",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		MTLS: MTLSInput{
			MTLSEnabled:     true,
			MTLSOptional:    true,
			MTLSClientCARef: "/etc/ssl/ca.crt",
		},
	})
	if !strings.Contains(conf, "ssl_verify_client optional;") {
		t.Fatalf("expected ssl_verify_client optional; for optional mTLS, got:\n%s", conf)
	}
}

func TestFront_MTLS_Disabled_NoDirectives(t *testing.T) {
	conf := mustRenderSiteConf(t, "mtls-off", EasyProfileInput{
		SiteID:         "mtls-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		MTLS:           MTLSInput{MTLSEnabled: false},
	})
	if strings.Contains(conf, "ssl_verify_client") {
		t.Fatalf("expected no ssl_verify_client when mTLS disabled, got:\n%s", conf)
	}
	if strings.Contains(conf, "ssl_client_certificate") {
		t.Fatalf("expected no ssl_client_certificate when mTLS disabled, got:\n%s", conf)
	}
}

func TestFront_MTLS_PassHeaders_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "mtls-headers", EasyProfileInput{
		SiteID:         "mtls-headers",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		MTLS: MTLSInput{
			MTLSEnabled:     true,
			MTLSClientCARef: "/etc/ssl/ca.crt",
			MTLSPassHeaders: true,
		},
	})
	if !strings.Contains(conf, "proxy_set_header X-Client-Verify $ssl_client_verify;") {
		t.Fatalf("expected proxy_set_header X-Client-Verify when MTLSPassHeaders=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_set_header X-Client-DN $ssl_client_s_dn;") {
		t.Fatalf("expected proxy_set_header X-Client-DN when MTLSPassHeaders=true, got:\n%s", conf)
	}
}

func TestFront_MTLS_Validation_NoCA(t *testing.T) {
	err := ValidateMTLS(MTLSInput{MTLSEnabled: true, MTLSClientCARef: ""})
	if err == nil {
		t.Fatal("expected error when MTLSEnabled=true and no CA ref set")
	}
}

func TestFront_MTLS_Validation_NegativeDepth(t *testing.T) {
	err := ValidateMTLS(MTLSInput{
		MTLSEnabled:     true,
		MTLSClientCARef: "/etc/ssl/ca.crt",
		MTLSVerifyDepth: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative MTLSVerifyDepth")
	}
}

// --- SecurityMode в конфиге ---

func TestFront_SecurityMode_Block_ModSecOn(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "sec-block", Enabled: true, PrimaryHost: "sec-block.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:                   "sec-block",
			SecurityMode:             "block",
			AllowedMethods:           []string{"GET"},
			MaxClientSize:            "10m",
			UseModSecurity:           true,
			UseModSecurityCRSPlugins: true,
			ModSecurityCRSVersion:    "4",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	// block+UseModSecurity → modsec артефакт должен существовать
	if _, ok := byPath["modsecurity/easy/sec-block.conf"]; !ok {
		t.Fatal("expected modsecurity/easy/sec-block.conf artifact for block mode")
	}
	// site.conf должен ссылаться на modsec файл
	siteConf := string(byPath["nginx/easy/sec-block.conf"].Content)
	if !strings.Contains(siteConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/sec-block.conf;") {
		t.Fatalf("expected modsecurity_rules_file directive in site conf, got:\n%s", siteConf)
	}
}

func TestFront_SecurityMode_Disabled_NoModSec(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "sec-disabled", Enabled: true, PrimaryHost: "sec-disabled.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "sec-disabled",
			SecurityMode:   "disabled",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: false,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if _, ok := byPath["modsecurity/easy/sec-disabled.conf"]; ok {
		t.Fatal("unexpected modsecurity artifact for disabled security mode")
	}
}

func TestFront_SecurityMode_Monitor_DetectionOnlyModSecArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "sec-monitor", Enabled: true, PrimaryHost: "sec-monitor.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "sec-monitor",
			SecurityMode:   "monitor",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	modsec, ok := byPath["modsecurity/easy/sec-monitor.conf"]
	if !ok {
		t.Fatal("expected modsecurity/easy artifact for monitor mode")
	}
	if !strings.Contains(string(modsec.Content), "SecRuleEngine DetectionOnly") {
		t.Fatalf("expected detection-only modsecurity engine for monitor mode, got:\n%s", string(modsec.Content))
	}
	siteConf := string(byPath["nginx/easy/sec-monitor.conf"].Content)
	if !strings.Contains(siteConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/sec-monitor.conf;") {
		t.Fatalf("expected monitor mode site conf to reference easy modsecurity rules, got:\n%s", siteConf)
	}
}

// artifactsByPath индексирует артефакты по пути для удобного поиска в тестах.
func artifactsByPath(artifacts []ArtifactOutput) map[string]ArtifactOutput {
	m := make(map[string]ArtifactOutput, len(artifacts))
	for _, a := range artifacts {
		m[a.Path] = a
	}
	return m
}

// --- helpers ---

// mustRenderSiteConf рендерит артефакты для одного сайта и возвращает nginx site.conf.
// Если артефакт не найден или рендер падает — тест провален.
func mustRenderSiteConf(t *testing.T, siteID string, profile EasyProfileInput) string {
	t.Helper()
	siteInput := SiteInput{
		ID:                  siteID,
		Enabled:             true,
		PrimaryHost:         siteID + ".example.com",
		ListenHTTP:          true,
		ListenHTTPS:         profile.MTLS.MTLSEnabled,
		MTLS:                profile.MTLS,
		UseEasyConfig:       true,
		UseCustomErrorPages: profile.UseCustomErrorPages,
		DefaultUpstreamID:   "up",
	}
	easyArtifacts, err := RenderEasyArtifacts([]SiteInput{siteInput}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("RenderEasyArtifacts(%s): %v", siteID, err)
	}
	upstreamArtifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{siteInput},
		[]UpstreamInput{{ID: "up", SiteID: siteID, Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
	)
	if err != nil {
		t.Fatalf("RenderSiteUpstreamArtifacts(%s): %v", siteID, err)
	}
	easyPath := "nginx/easy/" + siteID + ".conf"
	sitesPath := "nginx/sites/" + siteID + ".conf"
	var combined strings.Builder
	for _, a := range append(easyArtifacts, upstreamArtifacts...) {
		if a.Path == easyPath || a.Path == sitesPath {
			combined.WriteString(string(a.Content))
			combined.WriteString("\n")
		}
	}
	if combined.Len() == 0 {
		t.Fatalf("neither %q nor %q found in rendered artifacts", easyPath, sitesPath)
	}
	return combined.String()
}
