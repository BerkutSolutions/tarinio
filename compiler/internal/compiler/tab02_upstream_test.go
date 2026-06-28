package compiler

import (
	"strings"
	"testing"
)

// tab02_upstream_test.go — тесты вкладки 2: Апстрим
// Покрывает: proxy_pass, PassHostHeader, CustomHost, ReverseProxySSLSNI,
// ReverseProxyWebsocket, ReverseProxyKeepalive, SendXForwardedFor/Proto/RealIP,
// UpstreamMTLS, HealthCheck.

// --- PassHostHeader ---

func TestUpstream_PassHostHeader_WithCustomHost(t *testing.T) {
	// PassHostHeader=true + CustomHost → proxy_set_header Host <custom>
	conf := mustRenderSiteConf(t, "passhost-custom", EasyProfileInput{
		SiteID:                 "passhost-custom",
		SecurityMode:           "block",
		AllowedMethods:         []string{"GET"},
		MaxClientSize:          "10m",
		PassHostHeader:         true,
		ReverseProxyCustomHost: "api.internal.example.com",
	})
	if !strings.Contains(conf, "proxy_set_header Host api.internal.example.com;") {
		t.Fatalf("expected proxy_set_header Host api.internal.example.com; got:\n%s", conf)
	}
}

func TestUpstream_PassHostHeader_NoCustomHost_NoOverride(t *testing.T) {
	// PassHostHeader=true без CustomHost — не добавляет proxy_set_header Host
	conf := mustRenderSiteConf(t, "passhost-nohost", EasyProfileInput{
		SiteID:                 "passhost-nohost",
		SecurityMode:           "block",
		AllowedMethods:         []string{"GET"},
		MaxClientSize:          "10m",
		PassHostHeader:         true,
		ReverseProxyCustomHost: "",
	})
	if strings.Contains(conf, "proxy_set_header Host api") {
		t.Fatalf("did not expect custom Host header when ReverseProxyCustomHost is empty, got:\n%s", conf)
	}
}

// --- ReverseProxyCustomHost ---

func TestUpstream_CustomHost_IsSet(t *testing.T) {
	// CustomHost требует PassHostHeader=true для вставки в конфиг
	conf := mustRenderSiteConf(t, "customhost", EasyProfileInput{
		SiteID:                 "customhost",
		SecurityMode:           "block",
		AllowedMethods:         []string{"GET"},
		MaxClientSize:          "10m",
		PassHostHeader:         true,
		ReverseProxyCustomHost: "api.internal.example.com",
	})
	if !strings.Contains(conf, "proxy_set_header Host api.internal.example.com;") {
		t.Fatalf("expected proxy_set_header Host api.internal.example.com; when CustomHost set, got:\n%s", conf)
	}
}

func TestUpstream_CustomHost_Empty_NoOverride(t *testing.T) {
	conf := mustRenderSiteConf(t, "customhost-empty", EasyProfileInput{
		SiteID:                 "customhost-empty",
		SecurityMode:           "block",
		AllowedMethods:         []string{"GET"},
		MaxClientSize:          "10m",
		ReverseProxyCustomHost: "",
		PassHostHeader:         false,
	})
	if strings.Contains(conf, "api.internal.example.com") {
		t.Fatalf("did not expect custom host when ReverseProxyCustomHost is empty, got:\n%s", conf)
	}
}

// --- ReverseProxySSLSNI ---

func TestUpstream_SSLSNI_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "sni-on", EasyProfileInput{
		SiteID:               "sni-on",
		SecurityMode:         "block",
		AllowedMethods:       []string{"GET"},
		MaxClientSize:        "10m",
		ReverseProxySSLSNI:   true,
		ReverseProxySSLSNIName: "backend.example.com",
	})
	if !strings.Contains(conf, "proxy_ssl_server_name on;") {
		t.Fatalf("expected proxy_ssl_server_name on; when ReverseProxySSLSNI=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_ssl_name backend.example.com;") {
		t.Fatalf("expected proxy_ssl_name backend.example.com; when SNIName set, got:\n%s", conf)
	}
}

func TestUpstream_SSLSNI_Disabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "sni-off", EasyProfileInput{
		SiteID:             "sni-off",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		ReverseProxySSLSNI: false,
	})
	if strings.Contains(conf, "proxy_ssl_server_name on;") {
		t.Fatalf("did not expect proxy_ssl_server_name on; when ReverseProxySSLSNI=false, got:\n%s", conf)
	}
}

// --- ReverseProxyWebsocket ---

func TestUpstream_Websocket_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "ws-up-on", EasyProfileInput{
		SiteID:                "ws-up-on",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyWebsocket: true,
	})
	if !strings.Contains(conf, `proxy_set_header Upgrade $http_upgrade;`) {
		t.Fatalf("expected Upgrade header when websocket enabled, got:\n%s", conf)
	}
	if !strings.Contains(conf, `proxy_set_header Connection "upgrade";`) &&
		!strings.Contains(conf, `proxy_set_header Connection $connection_upgrade;`) {
		t.Fatalf("expected Connection upgrade header when websocket enabled, got:\n%s", conf)
	}
}

func TestUpstream_Websocket_Disabled_NoUpgrade(t *testing.T) {
	conf := mustRenderSiteConf(t, "ws-up-off", EasyProfileInput{
		SiteID:                "ws-up-off",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyWebsocket: false,
	})
	if strings.Contains(conf, "proxy_set_header Upgrade") {
		t.Fatalf("did not expect Upgrade header when websocket disabled, got:\n%s", conf)
	}
}

// --- ReverseProxyKeepalive ---

func TestUpstream_Keepalive_Enabled_ConnectionEmpty(t *testing.T) {
	// В easy-режиме ReverseProxyKeepalive=true → proxy_set_header Connection "";
	conf := mustRenderSiteConf(t, "ka-on", EasyProfileInput{
		SiteID:                "ka-on",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyKeepalive: true,
	})
	if !strings.Contains(conf, `proxy_set_header Connection "";`) {
		t.Fatalf("expected proxy_set_header Connection empty when keepalive on, got:\n%s", conf)
	}
}

func TestUpstream_Keepalive_Disabled_NoConnectionEmpty(t *testing.T) {
	conf := mustRenderSiteConf(t, "ka-off", EasyProfileInput{
		SiteID:                "ka-off",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyKeepalive: false,
	})
	// Без keepalive — Connection "" не должен присутствовать (он только для keepalive)
	if strings.Contains(conf, `proxy_set_header Connection "";`) {
		t.Fatalf("did not expect proxy_set_header Connection empty when keepalive off, got:\n%s", conf)
	}
}

// --- HealthCheck (nginxSiteData / sites шаблон) ---

func TestUpstream_HealthCheck_Enabled(t *testing.T) {
	conf := renderSiteConfForTest(t, nginxSiteData{
		SiteID:             "hc-site",
		SiteIDSlug:         "hc_site",
		ServerNames:        []string{"hc.example.com"},
		ListenHTTP:         true,
		UpstreamName:       "site_hc_site_upstream",
		UpstreamAddress:    "backend:8080",
		ProxyPassTarget:    "http://site_hc_site_upstream",
		PassHostHeader:     true,
		HealthCheckEnabled: true,
	})
	if !strings.Contains(conf, "proxy_next_upstream error timeout http_502 http_503") {
		t.Fatalf("expected proxy_next_upstream when HealthCheckEnabled=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_next_upstream_tries 2") {
		t.Fatalf("expected proxy_next_upstream_tries 2, got:\n%s", conf)
	}
	if !strings.Contains(conf, "keepalive 32") {
		t.Fatalf("expected keepalive 32 in upstream block, got:\n%s", conf)
	}
}

func TestUpstream_HealthCheck_Disabled_NoNextUpstream(t *testing.T) {
	conf := renderSiteConfForTest(t, nginxSiteData{
		SiteID:             "hc-off",
		SiteIDSlug:         "hc_off",
		ServerNames:        []string{"hc-off.example.com"},
		ListenHTTP:         true,
		UpstreamName:       "site_hc_off_upstream",
		UpstreamAddress:    "backend:8080",
		ProxyPassTarget:    "http://site_hc_off_upstream",
		PassHostHeader:     true,
		HealthCheckEnabled: false,
	})
	if strings.Contains(conf, "proxy_next_upstream ") {
		t.Fatalf("did not expect proxy_next_upstream when HealthCheckEnabled=false, got:\n%s", conf)
	}
}

// --- X-Forwarded-* заголовки ---

func TestUpstream_XForwardedFor_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "xff-on", EasyProfileInput{
		SiteID:            "xff-on",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET"},
		MaxClientSize:     "10m",
		SendXForwardedFor: true,
	})
	if !strings.Contains(conf, "proxy_set_header X-Forwarded-For") {
		t.Fatalf("expected proxy_set_header X-Forwarded-For when SendXForwardedFor=true, got:\n%s", conf)
	}
}

func TestUpstream_XForwardedFor_Disabled_Cleared(t *testing.T) {
	conf := mustRenderSiteConf(t, "xff-off", EasyProfileInput{
		SiteID:            "xff-off",
		SecurityMode:      "block",
		AllowedMethods:    []string{"GET"},
		MaxClientSize:     "10m",
		SendXForwardedFor: false,
	})
	// Когда отключён — заголовок должен быть очищен пустой строкой
	if !strings.Contains(conf, `proxy_set_header X-Forwarded-For "";`) {
		t.Fatalf("expected X-Forwarded-For cleared when SendXForwardedFor=false, got:\n%s", conf)
	}
}

func TestUpstream_XForwardedProto_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "xfp-on", EasyProfileInput{
		SiteID:              "xfp-on",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET"},
		MaxClientSize:       "10m",
		SendXForwardedProto: true,
	})
	if !strings.Contains(conf, "proxy_set_header X-Forwarded-Proto") {
		t.Fatalf("expected proxy_set_header X-Forwarded-Proto when SendXForwardedProto=true, got:\n%s", conf)
	}
}

func TestUpstream_XRealIP_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "xri-on", EasyProfileInput{
		SiteID:         "xri-on",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		SendXRealIP:    true,
	})
	if !strings.Contains(conf, "proxy_set_header X-Real-IP") {
		t.Fatalf("expected proxy_set_header X-Real-IP when SendXRealIP=true, got:\n%s", conf)
	}
}

// --- Upstream mTLS ---

func TestUpstream_MTLS_DirectivesPresent(t *testing.T) {
	conf := mustRenderSiteConf(t, "up-mtls", EasyProfileInput{
		SiteID:         "up-mtls",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UpstreamMTLS: UpstreamMTLSInput{
			UpstreamMTLSEnabled: true,
			UpstreamMTLSCertRef: "/etc/ssl/waf.crt",
			UpstreamMTLSKeyRef:  "/etc/ssl/waf.key",
			UpstreamMTLSCARef:   "/etc/ssl/upstream-ca.crt",
		},
	})
	if !strings.Contains(conf, "proxy_ssl_certificate /etc/ssl/waf.crt;") {
		t.Fatalf("expected proxy_ssl_certificate when UpstreamMTLS enabled, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_ssl_certificate_key /etc/ssl/waf.key;") {
		t.Fatalf("expected proxy_ssl_certificate_key when UpstreamMTLS enabled, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_ssl_trusted_certificate /etc/ssl/upstream-ca.crt;") {
		t.Fatalf("expected proxy_ssl_trusted_certificate when UpstreamMTLSCARef set, got:\n%s", conf)
	}
}

func TestUpstream_MTLS_Disabled_NoProxySSLCert(t *testing.T) {
	conf := mustRenderSiteConf(t, "up-mtls-off", EasyProfileInput{
		SiteID:         "up-mtls-off",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UpstreamMTLS:   UpstreamMTLSInput{UpstreamMTLSEnabled: false},
	})
	if strings.Contains(conf, "proxy_ssl_certificate ") {
		t.Fatalf("did not expect proxy_ssl_certificate when UpstreamMTLS disabled, got:\n%s", conf)
	}
}

func TestUpstream_MTLS_Validation_NoCert(t *testing.T) {
	err := ValidateUpstreamMTLS(UpstreamMTLSInput{
		UpstreamMTLSEnabled: true,
		UpstreamMTLSCertRef: "",
		UpstreamMTLSKeyRef:  "/etc/ssl/waf.key",
	})
	if err == nil {
		t.Fatal("expected error when UpstreamMTLSEnabled=true and no cert ref")
	}
}

func TestUpstream_MTLS_Validation_NoKey(t *testing.T) {
	err := ValidateUpstreamMTLS(UpstreamMTLSInput{
		UpstreamMTLSEnabled: true,
		UpstreamMTLSCertRef: "/etc/ssl/waf.crt",
		UpstreamMTLSKeyRef:  "",
	})
	if err == nil {
		t.Fatal("expected error when UpstreamMTLSEnabled=true and no key ref")
	}
}
