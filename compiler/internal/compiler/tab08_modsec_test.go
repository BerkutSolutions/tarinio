package compiler

import (
	"strings"
	"testing"
)

// tab08_modsec_test.go — тесты вкладки 8: ModSecurity
// Покрывает: UseModSecurity включён/выключен, артефакт modsec/easy/<id>.conf,
// modsecurity_rules_file в site.conf, CRS версия, плагины, custom path/content.

// --- UseModSecurity: артефакт создаётся ---

func TestModsec_Enabled_ArtifactCreated(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-on", Enabled: true, PrimaryHost: "modsec-on.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "modsec-on",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if _, ok := byPath["modsecurity/easy/modsec-on.conf"]; !ok {
		t.Fatalf("expected modsecurity/easy/modsec-on.conf artifact, got: %v", func() []string {
			var keys []string
			for k := range byPath {
				keys = append(keys, k)
			}
			return keys
		}())
	}
}

// --- UseModSecurity: артефакт НЕ создаётся при выключенном ModSec ---

func TestModsec_Disabled_NoArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-off", Enabled: true, PrimaryHost: "modsec-off.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "modsec-off",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: false,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if _, ok := byPath["modsecurity/easy/modsec-off.conf"]; ok {
		t.Fatalf("did not expect modsecurity artifact when UseModSecurity=false")
	}
}

// --- modsecurity_rules_file в site.conf ---

func TestModsec_RulesFileDirective_InSiteConf(t *testing.T) {
	conf := mustRenderSiteConf(t, "modsec-rules", EasyProfileInput{
		SiteID:         "modsec-rules",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UseModSecurity: true,
	})
	if !strings.Contains(conf, "modsecurity_rules_file /etc/waf/modsecurity/easy/modsec-rules.conf;") {
		t.Fatalf("expected modsecurity_rules_file directive in site.conf, got:\n%s", conf)
	}
}

func TestModsec_Disabled_NoRulesFileDirective(t *testing.T) {
	conf := mustRenderSiteConf(t, "modsec-norules", EasyProfileInput{
		SiteID:         "modsec-norules",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		UseModSecurity: false,
	})
	if strings.Contains(conf, "modsecurity_rules_file /etc/waf/modsecurity/easy/") {
		t.Fatalf("did not expect easy modsecurity_rules_file when UseModSecurity=false, got:\n%s", conf)
	}
}

// --- CRS версия в артефакте ---

func TestModsec_CRSVersion_InArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-crs", Enabled: true, PrimaryHost: "modsec-crs.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:                "modsec-crs",
			SecurityMode:          "block",
			AllowedMethods:        []string{"GET"},
			MaxClientSize:         "10m",
			UseModSecurity:        true,
			ModSecurityCRSVersion: "3",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	content := string(byPath["modsecurity/easy/modsec-crs.conf"].Content)
	if !strings.Contains(content, "crs") && !strings.Contains(content, "CRS") && !strings.Contains(content, "coreruleset") {
		t.Fatalf("expected CRS reference in modsec artifact, got:\n%s", content)
	}
}

// --- CRS плагины ---

func TestModsec_Plugins_InArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-plg", Enabled: true, PrimaryHost: "modsec-plg.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:                   "modsec-plg",
			SecurityMode:             "block",
			AllowedMethods:           []string{"GET"},
			MaxClientSize:            "10m",
			UseModSecurity:           true,
			UseModSecurityCRSPlugins: true,
			ModSecurityCRSPlugins:    []string{"test-plugin"},
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	content := string(byPath["modsecurity/easy/modsec-plg.conf"].Content)
	if !strings.Contains(content, "test-plugin") {
		t.Fatalf("expected plugin test-plugin in modsec artifact, got:\n%s", content)
	}
}

// --- Custom content ---

func TestModsec_CustomContent_InArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-custom", Enabled: true, PrimaryHost: "modsec-custom.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:                            "modsec-custom",
			SecurityMode:                      "block",
			AllowedMethods:                    []string{"GET"},
			MaxClientSize:                     "10m",
			UseModSecurity:                    true,
			UseModSecurityCustomConfiguration: true,
			ModSecurityCustomContent:          "SecRule ARGS \"@rx evil\" \"id:9001,phase:2,deny\"",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	content := string(byPath["modsecurity/easy/modsec-custom.conf"].Content)
	if !strings.Contains(content, "SecRule ARGS") {
		t.Fatalf("expected custom SecRule in modsec artifact, got:\n%s", content)
	}
}

func TestModsec_ExclusionRules_PrecedeCustomContentInArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-order", Enabled: true, PrimaryHost: "modsec-order.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "modsec-order",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
			ModSecurityExclusionRules: []ModSecurityExclusionRuleInput{{
				Path:    "/admin",
				Methods: []string{"GET"},
				RuleIDs: []int{942100},
				Comment: "keep-login-safe",
			}},
			UseModSecurityCustomConfiguration: true,
			ModSecurityCustomContent:          "SecRule ARGS \"@rx evil\" \"id:9001,phase:2,deny\"",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	content := string(byPath["modsecurity/easy/modsec-order.conf"].Content)
	exclusionIndex := strings.Index(content, "# structured_exclusion_rules")
	customIndex := strings.Index(content, "SecRule ARGS \"@rx evil\"")
	if exclusionIndex == -1 {
		t.Fatalf("expected structured exclusion rules block in modsec artifact, got:\n%s", content)
	}
	if customIndex == -1 {
		t.Fatalf("expected custom SecRule in modsec artifact, got:\n%s", content)
	}
	if exclusionIndex > customIndex {
		t.Fatalf("expected structured exclusions before raw custom content, got:\n%s", content)
	}
}

func TestModsec_ManagementSite_SafeguardPrecedesArtifactStructuredExclusionsAndRawCustomRules(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "control-plane-access", Enabled: true, PrimaryHost: "control-plane-access.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "control-plane-access",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
			ModSecurityExclusionRules: []ModSecurityExclusionRuleInput{{
				Path:    "/services",
				Mode:    "prefix",
				Methods: []string{"GET", "POST"},
				RuleIDs: []int{942100},
				Comment: "structured-services-exclusion",
			}},
			UseModSecurityCustomConfiguration: true,
			ModSecurityCustomContent:          "SecRule ARGS \"@rx evil\" \"id:9001,phase:2,deny\"",
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	conf := string(byPath["nginx/easy/control-plane-access.conf"].Content)
	if strings.Contains(conf, "modsecurity on;") || strings.Contains(conf, "modsecurity_rules_file") {
		t.Fatalf("management easy snippet must not enable ModSecurity, got:\n%s", conf)
	}
	for _, expected := range []string{"api/administration(?:/.*)?", "api/management-hosts(?:/.*)?", "static/.*", "dashboard(?:/.*)?", "services(?:/.*)?", "login(?:/.*)?", "login/2fa(?:/.*)?", "auth(?:/.*)?", "auth/verify(?:/.*)?"} {
		if !strings.Contains(conf, expected) {
			t.Fatalf("expected management safeguard to include route fragment %q, got:\n%s", expected, conf)
		}
	}
	artifact := string(byPath["modsecurity/easy/control-plane-access.conf"].Content)
	structuredIndex := strings.Index(artifact, `# exclusion_comment: structured-services-exclusion`)
	rawIndex := strings.Index(artifact, `SecRule ARGS "@rx evil"`)
	if structuredIndex == -1 {
		t.Fatalf("expected structured exclusion in modsecurity artifact, got:\n%s", artifact)
	}
	if rawIndex == -1 {
		t.Fatalf("expected raw custom modsecurity rule in modsecurity artifact, got:\n%s", artifact)
	}
	if structuredIndex > rawIndex {
		t.Fatalf("expected structured exclusions before raw custom rule in modsecurity artifact, got:\n%s", artifact)
	}
}

func TestTab08ExplicitManagementSnapshotGeneratesOnlyManagementSafeguard(t *testing.T) {
	sites := []SiteInput{
		{ID: "panel", Enabled: true, PrimaryHost: "panel.example", ListenHTTP: true, Management: true, ManagementConfigured: true},
		{ID: "ordinary", Enabled: true, PrimaryHost: "ordinary.example", ListenHTTP: true, ManagementConfigured: true},
	}
	profiles := []EasyProfileInput{defaultEasyProfileForSite("panel"), defaultEasyProfileForSite("ordinary")}
	artifacts, err := RenderEasyArtifacts(sites, profiles)
	if err != nil {
		t.Fatalf("render tab08 artifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if strings.Contains(string(byPath["nginx/easy/panel.conf"].Content), "modsecurity on;") {
		t.Fatal("management tab08 artifact enables ModSecurity")
	}
	if strings.Contains(string(byPath["nginx/easy/ordinary.conf"].Content), "ctl:ruleEngine=Off") {
		t.Fatal("ordinary tab08 artifact inherited safeguard")
	}
}

func TestModsec_ManagementSite_ModSecurityToggleControlsEasyArtifact(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "management-site")

	artifactsOn, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "management-site", Enabled: true, PrimaryHost: "ui.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:                   "management-site",
			SecurityMode:             "block",
			AllowedMethods:           []string{"GET", "POST"},
			MaxClientSize:            "10m",
			UseModSecurity:           true,
			UseModSecurityCRSPlugins: true,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts(enabled): %v", err)
	}
	byPathOn := artifactsByPath(artifactsOn)
	artifactOn, ok := byPathOn["modsecurity/easy/management-site.conf"]
	if !ok {
		t.Fatalf("expected management-site modsecurity artifact when UseModSecurity=true")
	}
	contentOn := string(artifactOn.Content)
	if !strings.Contains(contentOn, "SecRuleEngine On") {
		t.Fatalf("expected management-site modsecurity artifact to enable modsecurity, got:\n%s", contentOn)
	}
	confOn := string(byPathOn["nginx/easy/management-site.conf"].Content)
	for _, marker := range []string{"login(?:/.*)?", "login/2fa(?:/.*)?", "api/administration(?:/.*)?", "static/.*"} {
		if !strings.Contains(confOn, marker) {
			t.Fatalf("expected management-site nginx config to include safeguard marker %q, got:\n%s", marker, confOn)
		}
	}

	artifactsOff, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "management-site", Enabled: true, PrimaryHost: "ui.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "management-site",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET", "POST"},
			MaxClientSize:  "10m",
			UseModSecurity: false,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts(disabled): %v", err)
	}
	byPathOff := artifactsByPath(artifactsOff)
	if _, ok := byPathOff["modsecurity/easy/management-site.conf"]; ok {
		t.Fatalf("did not expect management-site modsecurity artifact when UseModSecurity=false")
	}
	confOff := string(byPathOff["nginx/easy/management-site.conf"].Content)
	if strings.Contains(confOff, "ctl:ruleEngine=Off") {
		t.Fatalf("did not expect management-site safeguard when UseModSecurity=false, got:\n%s", confOff)
	}
}

func TestModsec_OrdinarySite_DoesNotInheritManagementSafeguardMarkers(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "management-site")
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "app.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "site-a",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET", "POST"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	ordinary := string(byPath["nginx/easy/site-a.conf"].Content)
	if strings.Contains(ordinary, "ctl:ruleEngine=Off") {
		t.Fatalf("did not expect ordinary site nginx config to inherit management safeguard, got:\n%s", ordinary)
	}
	for _, marker := range []string{"/login", "/login/2fa", "/api/", "/static/", "dashboard(?:/.*)?", "services(?:/.*)?"} {
		if strings.Contains(ordinary, marker) {
			t.Fatalf("did not expect ordinary site nginx config to inherit management safeguard marker %q, got:\n%s", marker, ordinary)
		}
	}
}

func TestModsec_ManagementSite_CRSVariantsKeepSafeguardWhenModSecurityEnabled(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "management-site")

	cases := []struct {
		name       string
		profile    EasyProfileInput
		wantPlugin bool
	}{
		{
			name: "crs-v4-with-plugins",
			profile: EasyProfileInput{
				SiteID:                   "management-site",
				SecurityMode:             "block",
				AllowedMethods:           []string{"GET", "POST"},
				MaxClientSize:            "10m",
				UseModSecurity:           true,
				UseModSecurityCRSPlugins: true,
				ModSecurityCRSVersion:    "4",
				ModSecurityCRSPlugins:    []string{"test-plugin"},
			},
			wantPlugin: true,
		},
		{
			name: "crs-v3-without-plugins",
			profile: EasyProfileInput{
				SiteID:                   "management-site",
				SecurityMode:             "block",
				AllowedMethods:           []string{"GET", "POST"},
				MaxClientSize:            "10m",
				UseModSecurity:           true,
				UseModSecurityCRSPlugins: false,
				ModSecurityCRSVersion:    "3",
				ModSecurityCRSPlugins:    []string{"test-plugin"},
			},
			wantPlugin: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			artifacts, err := RenderEasyArtifacts(
				[]SiteInput{{ID: "management-site", Enabled: true, PrimaryHost: "ui.example.com", ListenHTTP: true}},
				[]EasyProfileInput{tc.profile},
			)
			if err != nil {
				t.Fatalf("RenderEasyArtifacts: %v", err)
			}
			byPath := artifactsByPath(artifacts)
			conf := string(byPath["nginx/easy/management-site.conf"].Content)
			if strings.Contains(conf, "modsecurity on;") || strings.Contains(conf, "modsecurity_rules_file") {
				t.Fatalf("management-site must not enable ModSecurity for %s, got:\n%s", tc.name, conf)
			}
			for _, marker := range []string{"login(?:/.*)?", "login/2fa(?:/.*)?", "api/administration(?:/.*)?", "static/.*"} {
				if !strings.Contains(conf, marker) {
					t.Fatalf("expected management-site safeguard marker %q for %s, got:\n%s", marker, tc.name, conf)
				}
			}

			artifact := string(byPath["modsecurity/easy/management-site.conf"].Content)
			if !strings.Contains(artifact, "# crs_version: "+tc.profile.ModSecurityCRSVersion) {
				t.Fatalf("expected management-site modsecurity artifact to retain CRS version %q, got:\n%s", tc.profile.ModSecurityCRSVersion, artifact)
			}
			hasPlugin := strings.Contains(artifact, "test-plugin")
			if hasPlugin != tc.wantPlugin {
				t.Fatalf("expected plugin presence=%t for %s, got artifact:\n%s", tc.wantPlugin, tc.name, artifact)
			}
		})
	}
}

func TestModsec_ExclusionRules_GenerateExpectedMatchersAndActions(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"site-exclusions",
		"block",
		false,
		"",
		nil,
		[]ModSecurityExclusionRuleInput{
			{
				Path:    "/login",
				Methods: []string{"GET", "HEAD"},
				RuleIDs: []int{942100, 949110},
				Comment: "exact-login",
			},
			{
				Path:    "/services",
				Mode:    "prefix",
				Methods: []string{"POST"},
				RuleIDs: []int{942100},
				Comment: "prefix-services",
			},
			{
				PathPattern: `^/api/(?:sites|access-policies)/[^/]+$`,
				Mode:        "regex",
				Methods:     []string{"PUT", "DELETE"},
				RuleIDs:     []int{949110},
				Targets:     []string{"ARGS:payload", "REQUEST_HEADERS:Content-Type"},
				Comment:     "regex-targeted",
			},
		},
		false,
		"",
		"",
		false,
		"",
		"",
		"",
		nil,
		nil,
	)
	for _, expected := range []string{
		`# exclusion_comment: exact-login`,
		`SecRule REQUEST_METHOD "@rx ^(?:GET|HEAD)$" "id:191000,phase:1,t:none,chain,pass,nolog"`,
		`SecRule REQUEST_URI "@rx ^/login$" "t:none,ctl:ruleRemoveById=942100,ctl:ruleRemoveById=949110"`,
		`# exclusion_comment: prefix-services`,
		`SecRule REQUEST_METHOD "@rx ^(?:POST)$" "id:191001,phase:1,t:none,chain,pass,nolog"`,
		`SecRule REQUEST_URI "@rx ^/services(?:$|/)" "t:none,ctl:ruleRemoveById=942100"`,
		`# exclusion_comment: regex-targeted`,
		`SecRule REQUEST_METHOD "@rx ^(?:PUT|DELETE)$" "id:191002,phase:1,t:none,chain,pass,nolog"`,
		`SecRule REQUEST_URI "@rx ^/api/(?:sites|access-policies)/[^/]+$" "t:none,ctl:ruleRemoveTargetById=949110;ARGS:payload,ctl:ruleRemoveTargetById=949110;REQUEST_HEADERS:Content-Type"`,
	} {
		if !strings.Contains(rules, expected) {
			t.Fatalf("expected generated exclusion directive %q, got:\n%s", expected, rules)
		}
	}
}

// --- SecurityMode=disabled → modsec артефакт всё равно создаётся (UseModSecurity=true) ---

func TestModsec_SecurityMode_Disabled_ArtifactStillCreated(t *testing.T) {
	// UseModSecurity=true создаёт артефакт независимо от SecurityMode
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-dis", Enabled: true, PrimaryHost: "modsec-dis.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "modsec-dis",
			SecurityMode:   "disabled",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if _, ok := byPath["modsecurity/easy/modsec-dis.conf"]; !ok {
		t.Fatalf("expected modsec artifact even with SecurityMode=disabled when UseModSecurity=true")
	}
}

// --- SecurityMode=disabled + UseModSecurity=false → нет артефакта ---

func TestModsec_SecurityMode_Disabled_UseModSecFalse_NoArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "modsec-dis2", Enabled: true, PrimaryHost: "modsec-dis2.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "modsec-dis2",
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
	if _, ok := byPath["modsecurity/easy/modsec-dis2.conf"]; ok {
		t.Fatalf("did not expect modsec artifact when UseModSecurity=false")
	}
}
