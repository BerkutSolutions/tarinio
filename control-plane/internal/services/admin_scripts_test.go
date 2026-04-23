package services

import (
	"strings"
	"testing"
)

func TestAdminScriptCatalogCollectWAFEventsHasNoCredentialFields(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	catalog := service.Catalog()

	var target AdminScriptDefinition
	found := false
	for _, item := range catalog.Scripts {
		if item.ID == "collect-waf-events" {
			target = item
			found = true
			break
		}
	}
	if !found {
		t.Fatal("collect-waf-events definition not found")
	}

	fieldNames := map[string]bool{}
	for _, field := range target.Fields {
		fieldNames[field.Name] = true
	}
	for _, forbidden := range []string{"WAF_USER", "WAF_PASS"} {
		if fieldNames[forbidden] {
			t.Fatalf("expected %s field to be removed from collect-waf-events", forbidden)
		}
	}
	for _, field := range []string{"SINCE", "FILTER_URI"} {
		if !fieldNames[field] {
			t.Fatalf("expected %s field to be present", field)
		}
	}
}

func TestAdminScriptBuildEnvironmentCollectWAFEventsUsesNoAuthByDefault(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	definition := service.catalog["collect-waf-events"]

	env, err := service.buildEnvironment(definition, map[string]string{
		"FILTER_SITE": "sentry.hantico.ru",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("buildEnvironment failed: %v", err)
	}

	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "FILTER_SITE=sentry.hantico.ru") {
		t.Fatalf("expected filter site in env, got %q", joined)
	}
	if !strings.Contains(joined, "SINCE=24h") {
		t.Fatalf("expected default SINCE value in env, got %q", joined)
	}
}

func TestAdminScriptBuildEnvironmentRejectsShellMetacharacters(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	definition := service.catalog["collect-waf-events"]

	_, err := service.buildEnvironment(definition, map[string]string{
		"FILTER_SITE": "safe.example ' && rm -rf /",
	}, t.TempDir())
	if err == nil {
		t.Fatalf("expected validation error for unsafe shell characters")
	}
}
