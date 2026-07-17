package virtualpatches

import "testing"

func TestValidateRejectsVirtualPatchControlCharactersAndUnsafeID(t *testing.T) {
	valid := VirtualPatch{ID: "vp-1", SiteID: "site-a", Pattern: "^/safe$", Target: TargetURI, Action: ActionBlock}
	if err := Validate(valid); err != nil {
		t.Fatalf("expected valid patch: %v", err)
	}
	for _, patch := range []VirtualPatch{
		{ID: "vp'1", SiteID: "site-a", Pattern: "^/safe$", Target: TargetURI, Action: ActionBlock},
		{ID: "vp-2", SiteID: "site-a", Pattern: "safe\nSecRuleEngine Off", Target: TargetURI, Action: ActionBlock},
	} {
		if err := Validate(patch); err == nil {
			t.Fatalf("expected unsafe patch %#v to be rejected", patch)
		}
	}
}
