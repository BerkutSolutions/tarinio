package easysiteprofiles

import "testing"

func TestNormalizeProfileAppliesSecurityModePolicy(t *testing.T) {
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
	transparent = normalizeProfile(transparent)
	if transparent.SecurityModSecurity.UseModSecurity {
		t.Fatal("transparent mode must disable modsecurity")
	}
	if transparent.SecurityBehaviorAndLimits.UseLimitReq || transparent.SecurityBehaviorAndLimits.UseLimitConn {
		t.Fatal("transparent mode must disable rate and connection limits")
	}
	if transparent.SecurityBehaviorAndLimits.UseBadBehavior || transparent.SecurityBehaviorAndLimits.UseBlacklist {
		t.Fatal("transparent mode must disable blocking behavior modules")
	}
	if transparent.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		t.Fatal("transparent mode must disable antibot challenge")
	}
	if transparent.SecurityAuthBasic.UseAuthBasic {
		t.Fatal("transparent mode must disable auth basic")
	}
	if transparent.SecurityAPIPositive.UseAPIPositiveSecurity {
		t.Fatal("transparent mode must disable api positive security")
	}

	monitor := base
	monitor.FrontService.SecurityMode = SecurityModeMonitor
	monitor = normalizeProfile(monitor)
	if monitor.SecurityModSecurity.UseModSecurity {
		t.Fatal("monitor mode must not apply modsecurity")
	}
	if monitor.SecurityBehaviorAndLimits.UseLimitReq || monitor.SecurityBehaviorAndLimits.UseLimitConn {
		t.Fatal("monitor mode must disable rate and connection limits")
	}
	if monitor.SecurityBehaviorAndLimits.UseBadBehavior || monitor.SecurityBehaviorAndLimits.UseBlacklist {
		t.Fatal("monitor mode must disable blocking behavior modules")
	}
	if monitor.SecurityModSecurity.UseModSecurityCRSPlugins || monitor.SecurityModSecurity.UseCustomConfiguration {
		t.Fatal("monitor mode must disable modsecurity plugins and custom configuration")
	}
	if monitor.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		t.Fatal("monitor mode must disable antibot challenge")
	}
	if monitor.SecurityAuthBasic.UseAuthBasic {
		t.Fatal("monitor mode must disable auth basic")
	}
	if monitor.SecurityAPIPositive.UseAPIPositiveSecurity {
		t.Fatal("monitor mode must disable api positive security")
	}
}
