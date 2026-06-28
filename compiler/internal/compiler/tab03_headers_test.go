package compiler

import (
	"strings"
	"testing"
)

// tab03_headers_test.go — тесты вкладки 3: HTTP-заголовки
// Покрывает: ReferrerPolicy, CSP, PermissionsPolicy, CORS, CookieFlags,
// KeepUpstreamHeaders, HSTS (полный набор опций).

// --- ReferrerPolicy ---

func TestHeaders_ReferrerPolicy_Set(t *testing.T) {
	conf := mustRenderSiteConf(t, "ref-pol", EasyProfileInput{
		SiteID:         "ref-pol",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		ReferrerPolicy: "strict-origin-when-cross-origin",
	})
	if !strings.Contains(conf, `add_header Referrer-Policy "strict-origin-when-cross-origin" always;`) {
		t.Fatalf("expected Referrer-Policy header, got:\n%s", conf)
	}
}

func TestHeaders_ReferrerPolicy_Empty_NoHeader(t *testing.T) {
	conf := mustRenderSiteConf(t, "ref-pol-off", EasyProfileInput{
		SiteID:         "ref-pol-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		ReferrerPolicy: "",
	})
	if strings.Contains(conf, "Referrer-Policy") {
		t.Fatalf("did not expect Referrer-Policy when empty, got:\n%s", conf)
	}
}

// --- ContentSecurityPolicy ---

func TestHeaders_CSP_Set(t *testing.T) {
	conf := mustRenderSiteConf(t, "csp-on", EasyProfileInput{
		SiteID:                "csp-on",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ContentSecurityPolicy: "default-src 'self'; script-src 'nonce-abc123'",
	})
	if !strings.Contains(conf, `add_header Content-Security-Policy "default-src 'self'; script-src 'nonce-abc123'" always;`) {
		t.Fatalf("expected Content-Security-Policy header, got:\n%s", conf)
	}
}

func TestHeaders_CSP_Empty_NoHeader(t *testing.T) {
	conf := mustRenderSiteConf(t, "csp-off", EasyProfileInput{
		SiteID:                "csp-off",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ContentSecurityPolicy: "",
	})
	if strings.Contains(conf, "Content-Security-Policy") {
		t.Fatalf("did not expect CSP header when empty, got:\n%s", conf)
	}
}

// --- PermissionsPolicy ---

func TestHeaders_PermissionsPolicy_Set(t *testing.T) {
	conf := mustRenderSiteConf(t, "pp-on", EasyProfileInput{
		SiteID:            "pp-on",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET"},
		MaxClientSize:     "10m",
		PermissionsPolicy: []string{"camera=()", "microphone=()", "geolocation=*"},
	})
	if !strings.Contains(conf, "add_header Permissions-Policy") {
		t.Fatalf("expected Permissions-Policy header, got:\n%s", conf)
	}
	if !strings.Contains(conf, "camera=()") {
		t.Fatalf("expected camera=() in Permissions-Policy, got:\n%s", conf)
	}
}

func TestHeaders_PermissionsPolicy_Empty_NoHeader(t *testing.T) {
	conf := mustRenderSiteConf(t, "pp-off", EasyProfileInput{
		SiteID:            "pp-off",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET"},
		MaxClientSize:     "10m",
		PermissionsPolicy: nil,
	})
	if strings.Contains(conf, "Permissions-Policy") {
		t.Fatalf("did not expect Permissions-Policy when nil, got:\n%s", conf)
	}
}

// --- CORS ---

func TestHeaders_CORS_Enabled_AllowOrigin(t *testing.T) {
	conf := mustRenderSiteConf(t, "cors-on", EasyProfileInput{
		SiteID:             "cors-on",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		UseCORS:            true,
		CORSAllowedOrigins: []string{"https://example.com"},
	})
	if !strings.Contains(conf, "Access-Control-Allow-Origin") {
		t.Fatalf("expected Access-Control-Allow-Origin when CORS enabled, got:\n%s", conf)
	}
	if !strings.Contains(conf, "Access-Control-Allow-Methods") {
		t.Fatalf("expected Access-Control-Allow-Methods when CORS enabled, got:\n%s", conf)
	}
	if !strings.Contains(conf, "Access-Control-Allow-Headers") {
		t.Fatalf("expected Access-Control-Allow-Headers when CORS enabled, got:\n%s", conf)
	}
}

func TestHeaders_CORS_Disabled_NoAllowOrigin(t *testing.T) {
	conf := mustRenderSiteConf(t, "cors-off", EasyProfileInput{
		SiteID:         "cors-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UseCORS:        false,
	})
	if strings.Contains(conf, "Access-Control-Allow-Origin") {
		t.Fatalf("did not expect CORS headers when UseCORS=false, got:\n%s", conf)
	}
}

func TestHeaders_CORS_MultipleOrigins(t *testing.T) {
	conf := mustRenderSiteConf(t, "cors-multi", EasyProfileInput{
		SiteID:             "cors-multi",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		UseCORS:            true,
		CORSAllowedOrigins: []string{"https://a.com", "https://b.com"},
	})
	if !strings.Contains(conf, "Access-Control-Allow-Origin") {
		t.Fatalf("expected CORS headers with multiple origins, got:\n%s", conf)
	}
}

// --- CookieFlags (proxy_cookie_flags) ---

func TestHeaders_CookieFlags_Set(t *testing.T) {
	conf := mustRenderSiteConf(t, "cookie-flags", EasyProfileInput{
		SiteID:         "cookie-flags",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		CookieFlags:    "* HttpOnly SameSite=Strict",
	})
	if !strings.Contains(conf, "proxy_cookie_flags * HttpOnly SameSite=Strict;") {
		t.Fatalf("expected proxy_cookie_flags directive, got:\n%s", conf)
	}
}

func TestHeaders_CookieFlags_Empty_NoDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "cookie-flags-off", EasyProfileInput{
		SiteID:         "cookie-flags-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		CookieFlags:    "",
	})
	if strings.Contains(conf, "proxy_cookie_flags") {
		t.Fatalf("did not expect proxy_cookie_flags when CookieFlags empty, got:\n%s", conf)
	}
}

func TestHeaders_CookieFlags_Secure(t *testing.T) {
	conf := mustRenderSiteConf(t, "cookie-secure", EasyProfileInput{
		SiteID:         "cookie-secure",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		CookieFlags:    "* Secure",
	})
	if !strings.Contains(conf, "proxy_cookie_flags * Secure;") {
		t.Fatalf("expected proxy_cookie_flags * Secure;, got:\n%s", conf)
	}
}

// --- KeepUpstreamHeaders (proxy_pass_header) ---

func TestHeaders_KeepUpstreamHeaders_Single(t *testing.T) {
	conf := mustRenderSiteConf(t, "keep-hdrs-1", EasyProfileInput{
		SiteID:              "keep-hdrs-1",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET"},
		MaxClientSize:       "10m",
		KeepUpstreamHeaders: []string{"X-Custom-Header"},
	})
	if !strings.Contains(conf, "proxy_pass_header X-Custom-Header;") {
		t.Fatalf("expected proxy_pass_header X-Custom-Header;, got:\n%s", conf)
	}
}

func TestHeaders_KeepUpstreamHeaders_Multiple(t *testing.T) {
	conf := mustRenderSiteConf(t, "keep-hdrs-2", EasyProfileInput{
		SiteID:              "keep-hdrs-2",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET"},
		MaxClientSize:       "10m",
		KeepUpstreamHeaders: []string{"X-Foo", "X-Bar", "Server"},
	})
	for _, hdr := range []string{"X-Foo", "X-Bar", "Server"} {
		if !strings.Contains(conf, "proxy_pass_header "+hdr+";") {
			t.Fatalf("expected proxy_pass_header %s;, got:\n%s", hdr, conf)
		}
	}
}

func TestHeaders_KeepUpstreamHeaders_Empty_NoDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "keep-hdrs-off", EasyProfileInput{
		SiteID:              "keep-hdrs-off",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET"},
		MaxClientSize:       "10m",
		KeepUpstreamHeaders: nil,
	})
	if strings.Contains(conf, "proxy_pass_header") {
		t.Fatalf("did not expect proxy_pass_header when KeepUpstreamHeaders nil, got:\n%s", conf)
	}
}

// --- HSTS в связке с заголовками ---

func TestHeaders_HSTS_FullDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "hsts-full-hdr", EasyProfileInput{
		SiteID:                "hsts-full-hdr",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		HSTSEnabled:           true,
		HSTSMaxAgeSeconds:     31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	})
	if !strings.Contains(conf, "Strict-Transport-Security") {
		t.Fatalf("expected Strict-Transport-Security header, got:\n%s", conf)
	}
	if !strings.Contains(conf, "max-age=31536000") {
		t.Fatalf("expected max-age=31536000 in HSTS, got:\n%s", conf)
	}
	if !strings.Contains(conf, "includeSubDomains") {
		t.Fatalf("expected includeSubDomains in HSTS, got:\n%s", conf)
	}
	if !strings.Contains(conf, "preload") {
		t.Fatalf("expected preload in HSTS, got:\n%s", conf)
	}
}

func TestHeaders_HSTS_Disabled_NoSTS(t *testing.T) {
	conf := mustRenderSiteConf(t, "hsts-off-hdr", EasyProfileInput{
		SiteID:         "hsts-off-hdr",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		HSTSEnabled:    false,
	})
	if strings.Contains(conf, "Strict-Transport-Security") {
		t.Fatalf("did not expect HSTS header when disabled, got:\n%s", conf)
	}
}

// --- Комбинация заголовков ---

func TestHeaders_AllSecurityHeaders_Together(t *testing.T) {
	conf := mustRenderSiteConf(t, "all-hdrs", EasyProfileInput{
		SiteID:                "all-hdrs",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'none'",
		PermissionsPolicy:     []string{"camera=()"},
		HSTSEnabled:           true,
		HSTSMaxAgeSeconds:     3600,
		UseCORS:               true,
		CORSAllowedOrigins:    []string{"*"},
		CookieFlags:           "* HttpOnly Secure",
		KeepUpstreamHeaders:   []string{"X-Trace-ID"},
	})
	checks := []string{
		"Referrer-Policy",
		"Content-Security-Policy",
		"Permissions-Policy",
		"Strict-Transport-Security",
		"Access-Control-Allow-Origin",
		"proxy_cookie_flags",
		"proxy_pass_header X-Trace-ID;",
	}
	for _, check := range checks {
		if !strings.Contains(conf, check) {
			t.Fatalf("expected %q in combined headers config, got:\n%s", check, conf)
		}
	}
}
