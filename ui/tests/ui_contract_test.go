package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIContract_OnboardingAndSidebarMarkers(t *testing.T) {
	contracts := []struct {
		name    string
		files   []string
		markers []string
	}{
		{
			name:  "onboarding html",
			files: []string{filepath.Join("..", "app", "onboarding.html")},
			markers: []string{
				`id="admin-password-toggle"`,
				`id="admin-password-confirm-toggle"`,
				`data-i18n="onboarding.step3"`,
				`id="onboarding-apply"`,
			},
		},
		{
			name:  "onboarding js",
			files: []string{filepath.Join("..", "app", "static", "js", "onboarding.js")},
			markers: []string{
				`"X-WAF-Auto-Apply-Disabled": "1"`,
				`/api/revisions/compile`,
				`/apply`,
				`has_active_revision`,
			},
		},
		{
			name:  "guard",
			files: []string{filepath.Join("..", "app", "static", "js", "guard.js")},
			markers: []string{
				`const initializationIncomplete = Boolean(setup && !setup.has_active_revision);`,
				`replace(httpUrl("/onboarding/user-creation"));`,
			},
		},
		{
			name:  "index",
			files: []string{filepath.Join("..", "app", "index.html")},
			markers: []string{
				`class="sidebar-logo-collapsed`,
				`id="menu" class="sidebar-nav"`,
				`id="logout-btn" class="icon-btn"`,
			},
		},
		{
			name:  "anti-ddos menu",
			files: []string{filepath.Join("..", "app", "static", "js", "app.js")},
			markers: []string{
				`pathBase: "/anti-ddos"`,
				`labelKey: "app.antiddos"`,
				`render: renderAntiDDoS`,
			},
		},
		{
			name: "dashboard modular markers",
			files: []string{
				filepath.Join("..", "app", "static", "js", "pages", "dashboard.js"),
				filepath.Join("..", "app", "static", "js", "pages", "dashboard.data-fetch.js"),
				filepath.Join("..", "app", "static", "js", "pages", "dashboard.contract-markers.js"),
				filepath.Join("..", "app", "static", "js", "pages", "dashboard.detail-builder.js"),
				filepath.Join("..", "app", "static", "js", "pages", "dashboard.frame.js"),
			},
			markers: []string{
				`/api/dashboard/stats`,
				`/api/dashboard/containers/overview`,
				`dashboard.widget.services`,
				`dashboard.widget.trafficSummary`,
				`dashboard.widget.requestsDay`,
				`dashboard.widget.attacksDay`,
				`dashboard.widget.blockedAttacks`,
				`services`,
				`traffic-summary`,
				`requests-day`,
				`attacks-day`,
				`blocked-attacks`,
				`frame-resize-handle`,
			},
		},
		{
			name: "sites stable renderer",
			files: []string{
				filepath.Join("..", "app", "static", "js", "app.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.page-main-runtime.js"),
			},
			markers: []string{
				`const module = await loadPageModule("sites.js");`,
				`ServicesStableFacadeLoadError`,
				`facadeTarget: "sites.stable-page.js"`,
				`Legacy-broken compatibility path.`,
			},
		},
		{
			name: "sites modular markers",
			files: []string{
				filepath.Join("..", "app", "static", "js", "pages", "sites.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.stable-page.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.stable-resources.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.stable-renderers.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.stable-detail-bind.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.access-upsert.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.service-policy-helpers.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.import-export.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.save-apply.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.view-io.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.draft-profile.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-bind-runtime.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-events-actions.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-render-view.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-render-view-part2.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-submit-delete.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.auth-extended-editors.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.page-main-actions-runtime.js"),
			},
			markers: []string{
				`function computeUpstreamID(siteID)`,
				`<div class="waf-upstream-target-row">`,
				`function draftToEnvText(draft)`,
				`function renderRawEditor(state, ctx, isNew)`,
				`data-mode-tab="raw"`,
				`async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null)`,
				`"X-WAF-Auto-Apply-Disabled": "1"`,
				`const requestOptions = options?.requestOptions || {};`,
				`ctx.api.post("/api/access-policies/upsert", payload, requestOptions);`,
				`/api/revisions/compile`,
				`/apply`,
				`id="services-select-all"`,
				`id="service-auth-help-btn"`,
				`id="service-antibot-help-btn"`,
				`bindDetailRuleEvents({`,
				`id="service-auth-mode"`,
				`data-auth-token-service-name=`,
				`const siteIDMetaHTML = showSiteIDMeta`,
				`waf-services-site-id`,
				`from "./sites.geo-lists.js";`,
				`buildGeoCatalogFallback`,
				`normalizeGeoCatalogPayload`,
				`pendingImportedDraftRef`,
			},
		},
		{
			name:  "requests",
			files: []string{
				filepath.Join("..", "app", "static", "js", "pages", "requests.js"),
				filepath.Join("..", "app", "static", "js", "pages", "requests.security.js"),
			},
			markers: []string{
				`export async function renderRequests`,
				`data-sort-col=`,
				`/api/requests`,
				`legacy_row_type_support`,
				`inferLegacyRequestRowType`,
				`requests.securityReason.modsecurity`,
			},
		},
		{
			name:  "revisions modal actions",
			files: []string{filepath.Join("..", "app", "static", "js", "pages", "revisions.js")},
			markers: []string{
				`id="revisions-delete-others"`,
				`ctx.t("revisions.action.deleteOthers")`,
				`revisions.toast.deleteOthersSucceeded`,
			},
		},
		{
			name: "settings modular markers",
			files: []string{
				filepath.Join("..", "app", "static", "js", "pages", "settings.js"),
				filepath.Join("..", "app", "static", "js", "pages", "settings.storage-logging.js"),
			},
			markers: []string{
				`data-settings-panel="secrets"`,
				`data-storage-index-stream="events"`,
				`data-storage-index-stream="activity"`,
				`settings-storage-hot-index-days`,
				`settings-storage-cold-index-days`,
			},
		},
		{
			name:  "requests menu",
			files: []string{filepath.Join("..", "app", "static", "js", "app.js")},
			markers: []string{
				`id: "requests"`,
				`labelKey: "app.requests"`,
				`render: renderRequests`,
			},
		},
	}

	for _, contract := range contracts {
		if len(contract.files) == 0 {
			t.Fatalf("contract %s has no files", contract.name)
		}
		contents := make([]string, 0, len(contract.files))
		for _, file := range contract.files {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s (%s): %v", contract.name, file, err)
			}
			contents = append(contents, string(raw))
		}
		for _, marker := range contract.markers {
			found := false
			for _, content := range contents {
				if strings.Contains(content, marker) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("contract broken for %s: missing %s in %v", contract.name, marker, contract.files)
			}
		}
	}
}
