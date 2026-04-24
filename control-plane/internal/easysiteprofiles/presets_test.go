package easysiteprofiles

import "testing"

func TestApplyServiceProfilePreset_Strict(t *testing.T) {
	profile := ApplyServiceProfilePreset(DefaultProfile("site-a"), ServiceProfileStrict)
	if profile.FrontService.Profile != ServiceProfileStrict {
		t.Fatalf("expected strict profile, got %s", profile.FrontService.Profile)
	}
	if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeJavascript {
		t.Fatalf("expected javascript antibot for strict, got %s", profile.SecurityAntibot.AntibotChallenge)
	}
	if profile.SecurityBehaviorAndLimits.LimitReqRate != "80r/s" {
		t.Fatalf("expected strict limit req rate 80r/s, got %s", profile.SecurityBehaviorAndLimits.LimitReqRate)
	}
}

func TestApplyServiceProfilePreset_API(t *testing.T) {
	profile := ApplyServiceProfilePreset(DefaultProfile("site-a"), ServiceProfileAPI)
	if profile.FrontService.Profile != ServiceProfileAPI {
		t.Fatalf("expected api profile, got %s", profile.FrontService.Profile)
	}
	if !profile.SecurityAPIPositive.UseAPIPositiveSecurity {
		t.Fatal("expected API positive security enabled for API preset")
	}
	if profile.SecurityBehaviorAndLimits.LimitReqURL != "/api/" {
		t.Fatalf("expected /api/ limit scope, got %s", profile.SecurityBehaviorAndLimits.LimitReqURL)
	}
}
