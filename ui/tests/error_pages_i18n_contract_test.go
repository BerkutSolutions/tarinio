package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestErrorPageListUsesLocalizedStatusLabels(t *testing.T) {
	codes := []string{
		"400", "401", "402", "403", "404", "405", "406", "407", "408", "409",
		"410", "411", "412", "413", "414", "415", "416", "417", "418", "421",
		"422", "423", "424", "425", "426", "428", "429", "431", "444", "500",
		"501", "502", "503", "504", "505", "507", "508", "510", "511",
	}
	keys := []string{"sites.easy.errorpages.geoBlock"}
	for _, code := range codes {
		keys = append(keys, "sites.easy.errorpages.status."+code)
	}

	for _, locale := range []string{"ru", "en", "zh", "de", "sr"} {
		lang := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", locale+".json"))
		for _, key := range keys {
			if strings.TrimSpace(lang[key]) == "" {
				t.Fatalf("%s misses %s", locale, key)
			}
		}
	}

	source, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "sites.detail-render-errorpages.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "sites.easy.errorpages.status.${slug}") ||
		!strings.Contains(string(source), "sites.easy.errorpages.geoBlock") {
		t.Fatal("error-page list must resolve standard statuses and Geo Block through i18n keys")
	}
}
