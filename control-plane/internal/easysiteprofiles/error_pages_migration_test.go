package easysiteprofiles

import (
	"slices"
	"testing"
)

func TestNormalizeProfileMigratesLegacy451GeoToggleIdempotently(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.DisabledErrorPages = []string{"451", "451"}

	first := normalizeProfile(profile)
	second := normalizeProfile(first)
	for _, migrated := range []EasySiteProfile{first, second} {
		if !slices.Contains(migrated.DisabledErrorPages, "451") || !slices.Contains(migrated.DisabledErrorPages, "geo_block") {
			t.Fatalf("legacy 451 toggle must retain legal state and disable geo block: %+v", migrated.DisabledErrorPages)
		}
		count := 0
		for _, slug := range migrated.DisabledErrorPages {
			if slug == "geo_block" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("migration must not create duplicate geo_block values: %+v", migrated.DisabledErrorPages)
		}
	}
}
