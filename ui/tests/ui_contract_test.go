package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIContract_OnboardingAndSidebarMarkers(t *testing.T) {
	contracts := []struct {
		file    string
		markers []string
	}{
		{
			file: filepath.Join("..", "app", "onboarding.html"),
			markers: []string{
				`id="admin-password-toggle"`,
				`id="admin-password-confirm-toggle"`,
				`data-i18n="onboarding.step3"`,
				`id="onboarding-apply"`,
			},
		},
		{
			file: filepath.Join("..", "app", "index.html"),
			markers: []string{
				`class="sidebar-logo-collapsed`,
				`id="menu" class="sidebar-nav"`,
				`id="logout-btn" class="icon-btn"`,
			},
		},
		{
			file: filepath.Join("..", "app", "static", "js", "app.js"),
			markers: []string{
				`pathBase: "/anti-ddos"`,
				`labelKey: "app.antiddos"`,
				`render: renderAntiDDoS`,
			},
		},
		{
			file: filepath.Join("..", "app", "static", "js", "pages", "dashboard.js"),
			markers: []string{
				`/api/dashboard/stats`,
				`/api/dashboard/containers/overview`,
				`dashboard.widget.servicesUp`,
				`dashboard.widget.servicesDown`,
				`dashboard.widget.requestsDay`,
				`dashboard.widget.attacksDay`,
				`dashboard.widget.blockedAttacks`,
				`services-up`,
				`services-down`,
				`requests-day`,
				`attacks-day`,
				`blocked-attacks`,
				`frame-resize-handle`,
			},
		},
		{
			file: filepath.Join("..", "app", "static", "js", "pages", "sites.js"),
			markers: []string{
				`function computeUpstreamID(siteID)`,
				`if (String(existingSite?._origin || "") === "secondary")`,
				`<div class="waf-upstream-target-row">`,
				`function draftToEnvText(draft)`,
				`function envToDraft(text)`,
				`id="services-select-all"`,
			},
		},
		{
			file: filepath.Join("..", "app", "static", "js", "pages", "requests.js"),
			markers: []string{
				`export async function renderRequests`,
				`data-sort-col=`,
				`/api/requests`,
			},
		},
		{
			file: filepath.Join("..", "app", "static", "js", "app.js"),
			markers: []string{
				`id: "requests"`,
				`labelKey: "app.requests"`,
				`render: renderRequests`,
			},
		},
	}

	for _, contract := range contracts {
		raw, err := os.ReadFile(contract.file)
		if err != nil {
			t.Fatalf("read %s: %v", contract.file, err)
		}
		content := string(raw)
		for _, marker := range contract.markers {
			if !strings.Contains(content, marker) {
				t.Fatalf("contract broken for %s: missing %s", contract.file, marker)
			}
		}
	}
}
