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
			files: []string{filepath.Join("..", "app", "static", "js", "onboarding.js"), filepath.Join("..", "app", "static", "js", "onboarding-management-hosts.js")},
			markers: []string{
				`"X-WAF-Auto-Apply-Disabled": "1"`,
				`/api/revisions/compile`,
				`/apply`,
				`has_active_revision`,
				`/api/settings/management-hosts`,
			},
		},
		{
			name:  "guard",
			files: []string{filepath.Join("..", "app", "static", "js", "guard.js")},
			markers: []string{
				`const onboardingRequired = Boolean(setup.needs_bootstrap);`,
				`replace(httpUrl("/onboarding/user-creation"));`,
			},
		},
		{
			name:  "index",
			files: []string{filepath.Join("..", "app", "index.html")},
			markers: []string{
				`class="sidebar-brand no-select"`,
				`class="sidebar-logo"`,
				`id="menu" class="sidebar-nav"`,
				`id="notifications-btn" class="icon-btn"`,
				`id="sidebar-toggle" class="sidebar-collapse-btn"`,
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
			name: "grouped sidebar menu",
			files: []string{
				filepath.Join("..", "app", "static", "js", "app.sidebar-menu.js"),
				filepath.Join("..", "app", "static", "js", "app.sidebar-status.js"),
			},
			markers: []string{
				`app.incidents`,
				`app.sidebarCertificates`,
				`app.sidebarGroup.system`,
				`sections: ["administration", "events", "activity", "settings"]`,
				`events: "app.sidebarJournal"`,
				`activity: "app.sidebarAudit"`,
				`/healthz`,
				`/api/reports/revisions`,
				`revision_apply?.active_revision_id`,
				`/api/anti-ddos/settings`,
				`/api/owasp-crs/status`,
				`sidebar-mode-model`,
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
				filepath.Join("..", "app", "static", "js", "pages", "sites.detail-draft-builder.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.modsec-exclusion-editors.js"),
				filepath.Join("..", "app", "static", "js", "pages", "sites.draft-core.js"),
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
				`normalizeModSecurityExclusionRules`,
				`modsecurity_exclusion_rules`,
				`use_modsecurity_custom_configuration`,
				`security_modsecurity: {`,
				`custom_configuration: {`,
				`state.rawEnvText = draftToEnvText`,
			},
		},
		{
			name:  "basic auth password visibility binding",
			files: []string{filepath.Join("..", "app", "static", "js", "pages", "sites.detail-events-rules.js")},
			markers: []string{
				`container.querySelectorAll("[data-auth-user-toggle]").forEach`,
				`/auth-password/reveal?username=`,
				`input.type = nextVisible ? "text" : "password"`,
				`syncAuthPasswordToggle(button, nextVisible, ctx)`,
			},
		},
		{
			name:  "basic auth password mask preserves character count",
			files: []string{filepath.Join("..", "app", "static", "js", "pages", "sites.auth-rules-editors.js")},
			markers: []string{
				`password_length`,
				`"•".repeat(Math.max(1, user.password_length))`,
				`data-auth-user-password-stored="true"`,
			},
		},
		{
			name: "requests",
			files: []string{
				filepath.Join("..", "app", "static", "js", "pages", "requests.js"),
				filepath.Join("..", "app", "static", "js", "pages", "requests.security.js"),
			},
			markers: []string{
				`export async function renderRequests`,
				`data-sort-col=`,
				`/api/requests`,
				`requests-filter-security-reason`,
				`selectedSecurityReason`,
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

func TestUIContract_SessionExpiryReturnsDirectlyToLogin(t *testing.T) {
	files := []string{
		filepath.Join("..", "app", "static", "js", "guard.js"),
		filepath.Join("..", "app", "static", "js", "api.js"),
		filepath.Join("..", "app", "static", "js", "login.js"),
		filepath.Join("..", "app", "static", "js", "login-2fa.js"),
		filepath.Join("..", "app", "login.html"),
		filepath.Join("..", "app", "login-2fa.html"),
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(raw), "/challenge?") || strings.Contains(string(raw), "buildLoginChallengeUrl") {
			t.Fatalf("session recovery must not redirect management login through challenge: %s", file)
		}
	}
	guard, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read guard: %v", err)
	}
	if !strings.Contains(string(guard), "export function buildLoginURL") || !strings.Contains(string(guard), "secureAppUrl(`/login?${params.toString()}`)") {
		t.Fatal("session recovery must construct a direct login URL")
	}
}

func TestUIContract_LoginAppearanceUsesSharedThemeStyles(t *testing.T) {
	files := map[string]string{
		"login":        filepath.Join("..", "app", "login.html"),
		"login2fa":     filepath.Join("..", "app", "login-2fa.html"),
		"theme styles": filepath.Join("..", "app", "static", "login-appearance.css"),
		"theme script": filepath.Join("..", "app", "static", "js", "login-appearance.js"),
		"settings":     filepath.Join("..", "app", "static", "js", "pages", "settings.js"),
	}
	contents := make(map[string]string, len(files))
	for name, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		contents[name] = string(raw)
	}
	for _, theme := range []string{"command-center", "incident-console", "command-center-classic", "security-card", "incident-console-classic"} {
		if !strings.Contains(contents["theme styles"], `data-login-appearance="`+theme+`"`) {
			t.Fatalf("theme CSS is missing %q", theme)
		}
		if !strings.Contains(contents["theme script"], `"`+theme+`"`) || !strings.Contains(contents["settings"], `value="`+theme+`"`) {
			t.Fatalf("theme is missing from login runtime or settings: %q", theme)
		}
	}
	for _, page := range []string{"login", "login2fa"} {
		if !strings.Contains(contents[page], `/static/login-appearance.css?v=20260714-4`) || !strings.Contains(contents[page], `class="login-theme-shell"`) {
			t.Fatalf("%s must use the shared login appearance shell", page)
		}
	}
	if strings.Contains(contents["theme styles"], "sidebar-rail") || strings.Contains(contents["theme script"], "sidebar-rail") || strings.Contains(contents["settings"], "sidebar-rail") {
		t.Fatal("removed Sidebar Rail appearance is still exposed")
	}
}

func TestUIContract_LocalhostServiceIDIsNotRewrittenToLegacyManagementID(t *testing.T) {
	files := []string{
		filepath.Join("..", "app", "static", "js", "pages", "sites.routing-merge.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.service-policy-helpers.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.page-utilities.js"),
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(raw)
		if strings.Contains(content, `"localhost" ||`) && strings.Contains(content, `"control-plane-access"`) {
			t.Fatalf("localhost must retain its persisted site ID instead of becoming control-plane-access: %s", file)
		}
	}
}

func TestUIContract_ManagementShellNeverReplacesItselfAfterBackgroundRateLimit(t *testing.T) {
	pages := []string{
		filepath.Join("..", "app", "index.html"),
		filepath.Join("..", "app", "onboarding.html"),
	}
	for _, page := range pages {
		raw, err := os.ReadFile(page)
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		content := string(raw)
		if !strings.Contains(content, `type="text/plain" data-retired-legacy-rate-limit-fallback="true"`) {
			t.Fatalf("%s must keep the retired legacy fallback script non-executable", page)
		}
		if strings.Contains(content, `window.__wafRenderBlocked({ code: 429 })`) {
			t.Fatalf("%s must not turn a failed asset into a synthetic 429 page", page)
		}
	}
}
