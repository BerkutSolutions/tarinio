package tests

import (
	"path/filepath"
	"testing"
)

func TestOWASPCRSErrorTranslations(t *testing.T) {
	codes := []string{
		"crs_release_unavailable",
		"crs_release_invalid",
		"crs_release_digest_invalid",
		"crs_archive_download_failed",
		"crs_archive_digest_mismatch",
		"crs_archive_invalid",
		"crs_update_failed",
	}
	for _, locale := range []string{"ru", "en", "zh", "de", "sr"} {
		lang := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", locale+".json"))
		for _, code := range codes {
			key := "owaspCrs.errors." + code
			if lang[key] == "" {
				t.Fatalf("%s is missing %s", locale, key)
			}
		}
	}
}
