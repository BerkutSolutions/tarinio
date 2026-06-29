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
			SecurityMode:         "block",
			AllowedMethods:       []string{"GET"},
			MaxClientSize:        "10m",
			UseModSecurity:       true,
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
			SiteID:                  "modsec-plg",
			SecurityMode:            "block",
			AllowedMethods:          []string{"GET"},
			MaxClientSize:           "10m",
			UseModSecurity:          true,
			UseModSecurityCRSPlugins: true,
			ModSecurityCRSPlugins:   []string{"test-plugin"},
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
			SiteID:                          "modsec-custom",
			SecurityMode:                    "block",
			AllowedMethods:                  []string{"GET"},
			MaxClientSize:                   "10m",
			UseModSecurity:                  true,
			UseModSecurityCustomConfiguration: true,
			ModSecurityCustomContent:         "SecRule ARGS \"@rx evil\" \"id:9001,phase:2,deny\"",
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
