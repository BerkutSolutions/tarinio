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

func TestAdminScriptCatalogIncludesHardeningCollector(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	catalog := service.Catalog()

	var target AdminScriptDefinition
	found := false
	for _, item := range catalog.Scripts {
		if item.ID == "collect-waf-hardening" {
			target = item
			found = true
			break
		}
	}
	if !found {
		t.Fatal("collect-waf-hardening definition not found")
	}
	fieldNames := map[string]bool{}
	for _, field := range target.Fields {
		fieldNames[field.Name] = true
	}
	for _, required := range []string{"RUNTIME_CONTAINER", "DEPLOY_DIR", "EXPECTED_TCP_TIMESTAMPS"} {
		if !fieldNames[required] {
			t.Fatalf("expected %s field in hardening collector", required)
		}
	}
}

func TestAdminScriptCatalogIncludesIndexHealthCollector(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	catalog := service.Catalog()

	var target AdminScriptDefinition
	found := false
	for _, item := range catalog.Scripts {
		if item.ID == "collect-waf-index-health" {
			target = item
			found = true
			break
		}
	}
	if !found {
		t.Fatal("collect-waf-index-health definition not found")
	}
	fieldNames := map[string]bool{}
	for _, field := range target.Fields {
		fieldNames[field.Name] = true
	}
	for _, required := range []string{"SINCE", "RUNTIME_CONTAINER", "CONTROL_PLANE_CONTAINER", "OPENSEARCH_CONTAINER", "CLICKHOUSE_CONTAINER"} {
		if !fieldNames[required] {
			t.Fatalf("expected %s field in index health collector", required)
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

func TestAdminScriptBuildEnvironmentAcceptsIndexHealthContainers(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	definition := service.catalog["collect-waf-index-health"]

	env, err := service.buildEnvironment(definition, map[string]string{
		"OPENSEARCH_CONTAINER": "tarinio-opensearch",
		"CLICKHOUSE_CONTAINER": "tarinio-clickhouse",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("buildEnvironment failed: %v", err)
	}

	joined := strings.Join(env, "\n")
	for _, expected := range []string{"SINCE=24h", "OPENSEARCH_CONTAINER=tarinio-opensearch", "CLICKHOUSE_CONTAINER=tarinio-clickhouse"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %s in env, got %q", expected, joined)
		}
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

func TestAdminScriptBuildEnvironmentRejectsInvalidExpectedTCPValue(t *testing.T) {
	service := NewAdminScriptService(t.TempDir(), "")
	definition := service.catalog["collect-waf-hardening"]
	_, err := service.buildEnvironment(definition, map[string]string{
		"EXPECTED_TCP_TIMESTAMPS": "2",
	}, t.TempDir())
	if err == nil {
		t.Fatalf("expected validation error for invalid EXPECTED_TCP_TIMESTAMPS")
	}
}
