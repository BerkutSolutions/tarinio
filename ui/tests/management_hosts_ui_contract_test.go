package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagementHostsUIContractAcrossLocales(t *testing.T) {
	keys := []string{
		"settings.tabs.managementHosts", "settings.managementHosts.title", "settings.managementHosts.drift",
		"sites.tls.confirmRebind", "sites.tls.rebindCancelled",
	}
	for _, locale := range []string{"ru", "en", "zh", "de", "sr"} {
		lang := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", locale+".json"))
		for _, key := range keys {
			if strings.TrimSpace(lang[key]) == "" {
				t.Fatalf("%s misses %s", locale, key)
			}
		}
	}
	for _, file := range []string{"settings.management-hosts.js", "sites.resource-upsert.js"} {
		content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "management-hosts") && file == "settings.management-hosts.js" {
			t.Fatalf("%s lacks management-host API contract", file)
		}
		if !strings.Contains(string(content), "confirmRebind") && file == "sites.resource-upsert.js" {
			t.Fatalf("%s lacks rebind confirmation", file)
		}
	}
}
