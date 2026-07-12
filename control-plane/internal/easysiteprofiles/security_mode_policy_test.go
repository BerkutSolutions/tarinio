package easysiteprofiles

import "testing"

func TestNormalizeProfilePreservesSecuritySettingsOutsideBlockMode(t *testing.T) {
	base := DefaultProfile("site-a")
	base.SecurityBehaviorAndLimits.UseBlacklist = true
	base.SecurityBehaviorAndLimits.BlacklistIP = []string{"198.51.100.10"}
	base.SecurityAntibot.AntibotChallenge = AntibotChallengeJavascript
	base.SecurityAuthBasic.UseAuthBasic = true
	base.SecurityAuthBasic.AuthBasicUser = "admin"
	base.SecurityAuthBasic.AuthBasicPassword = "secret"
	base.SecurityAuthBasic.Users = []SecurityAuthUser{{Username: "admin", Password: "secret", Enabled: true}}
	base.SecurityAPIPositive.UseAPIPositiveSecurity = true
	base.SecurityAPIPositive.EndpointPolicies = []APIPositiveEndpointPolicy{{Path: "/api/private", Methods: []string{"GET"}}}

	transparent := base
	transparent.FrontService.SecurityMode = SecurityModeTransparent
	monitor := base
	monitor.FrontService.SecurityMode = SecurityModeMonitor
	for _, profile := range []EasySiteProfile{normalizeProfile(transparent), normalizeProfile(monitor)} {
		if !profile.SecurityModSecurity.UseModSecurity || !profile.SecurityBehaviorAndLimits.UseLimitReq || !profile.SecurityBehaviorAndLimits.UseLimitConn {
			t.Fatal("security mode must not clear configured protection modules")
		}
		if !profile.SecurityBehaviorAndLimits.UseBadBehavior || !profile.SecurityBehaviorAndLimits.UseBlacklist {
			t.Fatal("security mode must not clear configured behavior modules")
		}
		if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeJavascript || !profile.SecurityAuthBasic.UseAuthBasic || !profile.SecurityAPIPositive.UseAPIPositiveSecurity {
			t.Fatal("security mode must preserve configured antibot, auth, and API protection")
		}
	}
}

func TestValidateProfileAllowsAntibotRulesWhenAntibotIsDisabled(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.SecurityAntibot.AntibotChallenge = AntibotChallengeNo
	profile.SecurityAntibot.ExclusionRules = []AntibotExclusionRule{{Path: "/health", Methods: []string{"GET"}}}
	profile.SecurityAntibot.ChallengeRules = []AntibotChallengeRule{{Path: "/admin", Challenge: AntibotChallengeCaptcha}}

	if err := validateProfile(normalizeProfile(profile)); err != nil {
		t.Fatalf("antibot rules should remain configurable while antibot is disabled: %v", err)
	}
}
