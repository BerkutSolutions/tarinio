package easysiteprofiles

import "testing"

func TestDefaultProfile_ManagementSiteDisablesBlockingSecurityControls(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "localhost")

	profile := DefaultProfile("localhost")

	if profile.FrontService.SecurityMode != SecurityModeTransparent {
		t.Fatalf("expected management default security mode %q, got %q", SecurityModeTransparent, profile.FrontService.SecurityMode)
	}
	if profile.SecurityModSecurity.UseModSecurity {
		t.Fatal("expected management default profile to disable modsecurity")
	}
	if profile.SecurityModSecurity.UseModSecurityCRSPlugins {
		t.Fatal("expected management default profile to disable CRS plugins")
	}
	if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		t.Fatalf("expected management default antibot challenge %q, got %q", AntibotChallengeNo, profile.SecurityAntibot.AntibotChallenge)
	}
	if !profile.FrontService.AdaptiveModelEnabled {
		t.Fatal("expected management default profile to keep adaptive model enabled")
	}
}

func TestDefaultProfile_RegularSiteKeepsBlockingDefaults(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")

	profile := DefaultProfile("localhost")

	if profile.FrontService.SecurityMode != SecurityModeBlock {
		t.Fatalf("expected regular default security mode %q, got %q", SecurityModeBlock, profile.FrontService.SecurityMode)
	}
	if !profile.SecurityModSecurity.UseModSecurity {
		t.Fatal("expected regular default profile to keep modsecurity enabled")
	}
	if !profile.SecurityModSecurity.UseModSecurityCRSPlugins {
		t.Fatal("expected regular default profile to keep CRS plugins enabled")
	}
	if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		t.Fatalf("expected regular default antibot challenge %q, got %q", AntibotChallengeNo, profile.SecurityAntibot.AntibotChallenge)
	}
}

func TestDefaultProfile_EmptyManagementEnvDoesNotPromoteLocalhostToManagementSite(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "")

	profile := DefaultProfile("localhost")

	if profile.FrontService.SecurityMode != SecurityModeBlock {
		t.Fatalf("expected localhost to stay regular when management env is empty, got %q", profile.FrontService.SecurityMode)
	}
	if !profile.SecurityModSecurity.UseModSecurity {
		t.Fatal("expected localhost to keep modsecurity enabled when management env is empty")
	}
}

func TestDefaultProfile_BuiltInManagementSiteIDsDisableBlockingSecurityControls(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "")

	for _, siteID := range []string{"control-plane-access", "control-plane", "ui"} {
		profile := DefaultProfile(siteID)
		if profile.FrontService.SecurityMode != SecurityModeTransparent {
			t.Fatalf("expected %s security mode %q, got %q", siteID, SecurityModeTransparent, profile.FrontService.SecurityMode)
		}
		if profile.SecurityModSecurity.UseModSecurity {
			t.Fatalf("expected %s to disable modsecurity", siteID)
		}
		if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
			t.Fatalf("expected %s antibot challenge %q, got %q", siteID, AntibotChallengeNo, profile.SecurityAntibot.AntibotChallenge)
		}
	}
}
