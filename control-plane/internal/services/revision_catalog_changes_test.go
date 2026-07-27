package services

import (
	"testing"

	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
)

func TestBuildRevisionChangesReportsExactServiceFieldPaths(t *testing.T) {
	first := revisionsnapshots.Snapshot{
		Sites: []sites.Site{{ID: "shop", PrimaryHost: "shop.example", Enabled: true}},
		EasySiteProfiles: []easysiteprofiles.EasySiteProfile{{
			SiteID:          "shop",
			SecurityAntibot: easysiteprofiles.SecurityAntibotSettings{AntibotChallenge: "no"},
		}},
	}
	second := first
	second.EasySiteProfiles = append([]easysiteprofiles.EasySiteProfile(nil), first.EasySiteProfiles...)
	second.EasySiteProfiles[0].SecurityAntibot.AntibotChallenge = "javascript"
	items := []revisions.Revision{{ID: "rev-1"}, {ID: "rev-2", TargetSiteIDs: []string{"shop"}}}
	changes := buildRevisionChanges(items, map[string]revisionsnapshots.Snapshot{"rev-1": first, "rev-2": second})
	got := filterRevisionChanges(changes["rev-2"], items[1].TargetSiteIDs)
	if len(got) != 1 || got[0].SiteID != "shop" || got[0].Action != "updated" {
		t.Fatalf("unexpected revision changes: %+v", got)
	}
	want := "easy_profile.security_antibot.antibot_challenge"
	found := false
	for _, field := range got[0].Fields {
		if field == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected changed field %q, got %+v", want, got[0].Fields)
	}
}
