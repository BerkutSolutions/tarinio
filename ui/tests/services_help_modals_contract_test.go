package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesDNSBLHelpHasTriggerModalAndBinding(t *testing.T) {
	read := func(name string) string {
		body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(body)
	}
	renderer := read("sites.detail-render-view-part2.js")
	events := read("sites.detail-events-rules.js")
	for _, marker := range []string{
		`id="service-traffic-dnsbl-help-btn"`,
		`renderTrafficDnsblHelpModal(ctx)`,
	} {
		if !strings.Contains(renderer, marker) {
			t.Fatalf("DNSBL help renderer missing %q", marker)
		}
	}
	if !strings.Contains(events, `toggleHelpModal("service-traffic-dnsbl-help-modal", true)`) {
		t.Fatal("DNSBL help trigger must open its modal")
	}
}

func TestServicesHelpModalsCloseOnEscapeAndRestoreFocus(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "sites.detail-events-rules.js"))
	if err != nil {
		t.Fatalf("read help events: %v", err)
	}
	content := string(body)
	for _, marker := range []string{
		`event.key !== "Escape"`,
		`.waf-modal[role='dialog']:not(.waf-hidden)`,
		`helpReturnFocus.focus()`,
	} {
		if !strings.Contains(content, marker) {
			t.Fatalf("help keyboard contract missing %q", marker)
		}
	}
}

func TestServicesAllowlistHelpModalIsRenderedOnce(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "sites.detail-render-view-part2.js"))
	if err != nil {
		t.Fatalf("read detail renderer: %v", err)
	}
	if count := strings.Count(string(body), `renderTrafficAllowlistHelpModal(ctx, escapeHtml)`); count != 1 {
		t.Fatalf("allowlist help modal must be rendered once, got %d", count)
	}
}
