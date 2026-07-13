package compiler

import (
	"strings"
	"testing"
)

func TestGeneratedErrorPagesHaveDedicatedCodeAndTarinioLink(t *testing.T) {
	seenGuides := map[string]string{}
	for code, expected := range map[string]string{
		"451": "Unavailable For Legal Reasons",
		"494": "Request Header Too Large", "495": "SSL Certificate Error", "496": "SSL Certificate Required",
		"497": "HTTP Request Sent to HTTPS Port", "499": "Client Closed Request", "506": "Variant Also Negotiates",
		"520": "Unknown Error", "521": "Web Server Is Down", "522": "Connection Timed Out",
		"523": "Origin Is Unreachable", "524": "A Timeout Occurred", "525": "SSL Handshake Failed", "526": "Invalid SSL Certificate",
	} {
		content, ok := GeneratedErrorPage(code)
		if !ok || !strings.Contains(string(content), "HTTP "+code) || !strings.Contains(string(content), expected) {
			t.Fatalf("generated page %s is not dedicated: %q", code, content)
		}
		if !strings.Contains(string(content), "https://github.com/BerkutSolutions/tarinio") {
			t.Fatalf("generated page %s lacks the Tarinio link", code)
		}
		for _, marker := range []string{"class=\"icon-wrap\"", "class=\"flow-row\"", "Request path", "What you can try", "Client IP"} {
			if !strings.Contains(string(content), marker) {
				t.Fatalf("generated page %s lacks the full WAF page layout marker %q", code, marker)
			}
		}
		guide := errorPageGuides[code]
		if guide.Target == "" || guide.FirstState == "" || guide.SecondState == "" || guide.Steps[0][0] == "" || guide.Steps[2][1] == "" {
			t.Fatalf("generated page %s lacks dedicated request-path or recovery guidance", code)
		}
		if other, duplicate := seenGuides[guide.Steps[0][0]]; duplicate {
			t.Fatalf("generated pages %s and %s share the same first recovery action", code, other)
		}
		seenGuides[guide.Steps[0][0]] = code
	}
}

func TestGeneratedErrorPagesLocalizeGuidanceAndKeepHostEndpoint(t *testing.T) {
	content, ok := GeneratedErrorPage("496")
	if !ok {
		t.Fatal("494 error page was not generated")
	}
	page := string(content)
	for _, marker := range []string{`"ru":{"Client":"TLS-клиент"`, "ТРЕБУЕТСЯ СЕРТИФИКАТ", "Выберите клиентский сертификат", "Сервис принимает только запросы с действительным клиентским сертификатом."} {
		if !strings.Contains(page, marker) {
			t.Fatalf("generated 496 page lacks localized guidance marker %q", marker)
		}
	}
}

func TestGeneratedErrorPagesUseStatusSpecificDescriptions(t *testing.T) {
	content, ok := GeneratedErrorPage("496")
	if !ok {
		t.Fatal("496 page was not generated")
	}
	page := string(content)
	for _, marker := range []string{
		"This address requires a valid client TLS certificate, but the client did not provide one.",
		"Для этого адреса нужен действительный клиентский TLS-сертификат, но клиент его не предоставил.",
	} {
		if !strings.Contains(page, marker) {
			t.Fatalf("generated 496 page lacks specific description %q", marker)
		}
	}
}

func TestGeneratedErrorPagesKeepDedicatedGuidesAfterLocalization(t *testing.T) {
	for code, expected := range map[string][]string{
		"451": {"Origin", "Read the restriction notice"},
		"494": {"HEADER TOO LARGE", "Reduce request headers"},
		"495": {"CERTIFICATE ERROR", "Choose a valid certificate"},
		"496": {"CERTIFICATE REQUIRED", "Select a client certificate"},
		"497": {"HTTPS port", "Use an HTTPS address"},
		"499": {"CONNECTION CLOSED", "Avoid duplicate operations"},
		"506": {"NEGOTIATION FAILED", "Review content negotiation"},
		"520": {"UNKNOWN RESPONSE", "Check origin health"},
		"521": {"SERVER DOWN", "Start or restore the service"},
		"522": {"CONNECT TIMEOUT", "Check network reachability"},
		"523": {"ORIGIN UNREACHABLE", "Verify the origin address"},
		"524": {"RESPONSE TIMEOUT", "Inspect slow operations"},
		"525": {"HANDSHAKE FAILED", "Check TLS compatibility"},
		"526": {"CERTIFICATE INVALID", "Renew or replace the certificate"},
	} {
		content, ok := GeneratedErrorPage(code)
		if !ok {
			t.Fatalf("%s page was not generated", code)
		}
		page := string(content)
		for _, marker := range expected {
			if !strings.Contains(page, marker) {
				t.Fatalf("generated %s page lacks dedicated guide marker %q", code, marker)
			}
		}
		if !strings.Contains(page, `"route":{"Nodes"`) {
			t.Fatalf("generated %s page does not include the post-localization guide override", code)
		}
	}
}

func TestExtendedErrorPageIconsRemainStatusSpecific(t *testing.T) {
	for code, expectedPath := range map[string]string{
		"495": "M40 57l8-8m0 8-8-8",
		"522": "M44 31a13 13",
		"524": "M50 34a7 7",
		"525": "M28 37l10 10",
		"526": "M47 46a6 6",
	} {
		if generatedErrorPages[code].Symbol == "" || !strings.Contains(generatedErrorPages[code].Symbol, expectedPath) {
			t.Fatalf("%s does not have its intended status icon", code)
		}
	}
}

func TestExtendedErrorPagesVisualizeTheirActualFailedHop(t *testing.T) {
	for code, expected := range map[string]errorRouteVisual{
		"494": {Nodes: [3]string{"error", "ok", "muted"}, Arrows: [3]string{"error", "muted"}},
		"496": {Nodes: [3]string{"tls", "ok", "muted"}, Arrows: [3]string{"tls", "muted"}},
		"499": {Nodes: [3]string{"closed", "ok", "muted"}, Arrows: [3]string{"closed", "muted"}},
		"521": {Nodes: [3]string{"ok", "ok", "error"}, Arrows: [3]string{"ok", "error"}},
		"525": {Nodes: [3]string{"ok", "ok", "tls"}, Arrows: [3]string{"ok", "tls"}},
		"526": {Nodes: [3]string{"ok", "ok", "cert"}, Arrows: [3]string{"ok", "cert"}},
	} {
		actual, ok := errorRouteVisuals[code]
		if !ok || actual != expected {
			t.Fatalf("%s route visual = %#v, want %#v", code, actual, expected)
		}
	}
	content, ok := GeneratedErrorPage("494")
	if !ok || !strings.Contains(string(content), `"Nodes":["error","ok","muted"]`) || !strings.Contains(string(content), "function hostIcon") {
		t.Fatal("generated page does not carry its route visual or the centered host cross adjustment")
	}
}

func TestExtendedErrorPagesUseConsistentTerminalRouteLabels(t *testing.T) {
	content, ok := GeneratedErrorPage("524")
	if !ok {
		t.Fatal("524 page was not generated")
	}
	page := string(content)
	for _, marker := range []string{
		`l==="ru"?"\u0421\u0435\u0440\u0432\u0435\u0440":"Origin"`,
		`v.style.setProperty("stroke",q.c,"important")`,
	} {
		if !strings.Contains(page, marker) {
			t.Fatalf("generated route page lacks marker %q", marker)
		}
	}
}

func TestNormalizeErrorPageRouteLabelsUpdatesLegacyPagesOnly(t *testing.T) {
	legacy := []byte(`<span id="n-host">Host</span></body>`)
	updated := string(NormalizeErrorPageRouteLabels(legacy))
	if !strings.Contains(updated, `l==="ru"?"\u0421\u0435\u0440\u0432\u0435\u0440":"Origin"`) {
		t.Fatalf("legacy route label script was not added: %s", updated)
	}
	plain := []byte(`<main>no route</main></body>`)
	if string(NormalizeErrorPageRouteLabels(plain)) != string(plain) {
		t.Fatal("content without a route endpoint must remain unchanged")
	}
}

func TestHandshakeErrorIconMarksCentralBreakInRed(t *testing.T) {
	content, ok := GeneratedErrorPage("525")
	if !ok {
		t.Fatal("525 page was not generated")
	}
	for _, marker := range []string{"M40 40l8 8m0-8-8 8", "#ff6b6b"} {
		if !strings.Contains(string(content), marker) {
			t.Fatalf("525 icon lacks central red break marker %q", marker)
		}
	}
}
