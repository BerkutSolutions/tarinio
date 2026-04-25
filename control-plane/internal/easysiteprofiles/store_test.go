package easysiteprofiles

import (
	"slices"
	"testing"
)

func TestDefaultProfile_HasExpectedDefaults(t *testing.T) {
	profile := DefaultProfile("site-a")
	if profile.SiteID != "site-a" {
		t.Fatalf("unexpected site id: %s", profile.SiteID)
	}
	if profile.FrontService.SecurityMode != SecurityModeBlock {
		t.Fatalf("unexpected security mode: %s", profile.FrontService.SecurityMode)
	}
	if profile.FrontService.Profile != ServiceProfileBalanced {
		t.Fatalf("unexpected service profile default: %s", profile.FrontService.Profile)
	}
	if profile.FrontService.AdaptiveModelEnabled {
		t.Fatal("expected adaptive model disabled for regular site default profile")
	}
	if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		t.Fatalf("unexpected antibot default: %s", profile.SecurityAntibot.AntibotChallenge)
	}
	if !profile.SecurityAntibot.ScannerAutoBanEnabled {
		t.Fatal("expected scanner auto-ban enabled by default")
	}
	if !profile.SecurityModSecurity.UseModSecurityCRSPlugins {
		t.Fatal("expected CRS to be enabled by default for easy profiles")
	}
	if got := len(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes); got != 4 {
		t.Fatalf("expected 4 default bad behavior codes, got %d", got)
	}
	if profile.SecurityAPIPositive.UseAPIPositiveSecurity {
		t.Fatal("expected API positive security disabled by default")
	}
}

func TestDefaultProfile_ControlPlaneAccessIncludesAPIMethods(t *testing.T) {
	profile := DefaultProfile("control-plane-access")
	if !profile.FrontService.AdaptiveModelEnabled {
		t.Fatal("expected adaptive model enabled for control-plane-access default profile")
	}
	required := []string{"PUT", "PATCH", "DELETE"}
	for _, method := range required {
		found := false
		for _, item := range profile.HTTPBehavior.AllowedMethods {
			if item == method {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s in default allowed methods for control-plane-access, got %+v", method, profile.HTTPBehavior.AllowedMethods)
		}
	}
}

func TestDefaultProfile_ControlPlaneAccessUsesHigherRateLimitsWithout429Escalation(t *testing.T) {
	profile := DefaultProfile("control-plane-access")

	if profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 < 300 {
		t.Fatalf("expected higher http/1 conn limit for control-plane-access, got %d", profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1)
	}
	if profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 < 500 {
		t.Fatalf("expected higher http/2 conn limit for control-plane-access, got %d", profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2)
	}
	if profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 < 500 {
		t.Fatalf("expected higher http/3 conn limit for control-plane-access, got %d", profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3)
	}
	if profile.SecurityBehaviorAndLimits.LimitReqRate != "300r/s" {
		t.Fatalf("expected higher request rate for control-plane-access, got %s", profile.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if profile.SecurityBehaviorAndLimits.LimitReqURL != "/api/" {
		t.Fatalf("expected management site request limiting on /api/, got %s", profile.SecurityBehaviorAndLimits.LimitReqURL)
	}
	if profile.SecurityBehaviorAndLimits.BadBehaviorThreshold < 100 {
		t.Fatalf("expected higher bad behavior threshold for control-plane-access, got %d", profile.SecurityBehaviorAndLimits.BadBehaviorThreshold)
	}
	if profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds < 60 {
		t.Fatalf("expected higher bad behavior period for control-plane-access, got %d", profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds)
	}
	if slices.Contains(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes, 403) {
		t.Fatalf("expected control-plane-access bad behavior codes to exclude 403, got %+v", profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes)
	}
	if slices.Contains(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes, 404) {
		t.Fatalf("expected control-plane-access bad behavior codes to exclude 404, got %+v", profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes)
	}
	if slices.Contains(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes, 429) {
		t.Fatalf("expected control-plane-access bad behavior codes to exclude 429, got %+v", profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes)
	}
}

func TestStore_CreateUpdateGetDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	profile := DefaultProfile("site-a")
	profile.FrontService.ServerName = "WWW.Example.COM"
	profile.SecurityBehaviorAndLimits.LimitReqRate = "100 r/s"
	profile.SecurityBehaviorAndLimits.CustomLimitRules = []CustomLimitRule{{Path: "/login", Rate: " 6 r/s "}, {Path: "/login", Rate: "6r/s"}}

	created, err := store.Create(profile)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.FrontService.ServerName != "www.example.com" {
		t.Fatalf("expected normalized host, got %s", created.FrontService.ServerName)
	}
	if created.SecurityBehaviorAndLimits.LimitReqRate != "100r/s" {
		t.Fatalf("expected normalized rate, got %s", created.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if len(created.SecurityBehaviorAndLimits.CustomLimitRules) != 1 {
		t.Fatalf("expected duplicate custom limits to collapse, got %+v", created.SecurityBehaviorAndLimits.CustomLimitRules)
	}
	if created.SecurityBehaviorAndLimits.CustomLimitRules[0].Rate != "6r/s" {
		t.Fatalf("expected normalized custom limit rate, got %+v", created.SecurityBehaviorAndLimits.CustomLimitRules)
	}

	got, ok, err := store.Get("site-a")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected profile to exist")
	}
	if got.SiteID != "site-a" {
		t.Fatalf("unexpected profile site id: %s", got.SiteID)
	}

	got.SecurityAntibot.AntibotChallenge = AntibotChallengeCookie
	updated, err := store.Update(got)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.SecurityAntibot.AntibotChallenge != AntibotChallengeCookie {
		t.Fatalf("unexpected antibot mode after update: %s", updated.SecurityAntibot.AntibotChallenge)
	}

	if err := store.Delete("site-a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, ok, err = store.Get("site-a")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if ok {
		t.Fatal("expected profile to be deleted")
	}
}

func TestStore_CreateRejectsUnsupportedBadBehaviorCode(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{400, 999}
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for unsupported bad behavior code")
	}
}

func TestStore_CreateRejectsRecaptchaWithoutSecrets(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityAntibot.AntibotChallenge = AntibotChallengeRecaptcha
	profile.SecurityAntibot.AntibotRecaptchaSitekey = "site-key"
	profile.SecurityAntibot.AntibotRecaptchaSecret = ""
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for missing recaptcha secret")
	}
}

func TestStore_CreateRejectsAuthBasicWithoutPassword(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityAuthBasic.UseAuthBasic = true
	profile.SecurityAuthBasic.AuthBasicUser = "admin"
	profile.SecurityAuthBasic.AuthBasicPassword = ""
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for missing auth basic password")
	}
}

func TestStore_CreateRejectsCountryConflict(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityCountryPolicy.BlacklistCountry = []string{"US", "EU"}
	profile.SecurityCountryPolicy.WhitelistCountry = []string{"US"}
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for conflicting country selectors")
	}
}

func TestStore_CreateRejectsInvalidCountrySelector(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityCountryPolicy.BlacklistCountry = []string{"INVALID_SELECTOR"}
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for invalid country selector")
	}
}

func TestStore_CreateRejectsInvalidModSecurityCustomPath(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityModSecurity.UseCustomConfiguration = true
	profile.SecurityModSecurity.CustomConfiguration.Path = "../modsec/bad.conf"
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for invalid modsecurity custom path")
	}
}

func TestStore_CreateRejectsInvalidModSecurityCRSVersion(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("site-a")
	profile.SecurityModSecurity.ModSecurityCRSVersion = "v4"
	if _, err := store.Create(profile); err == nil {
		t.Fatal("expected validation error for invalid modsecurity crs version")
	}
}

func TestStore_UpdateAutoRepairsControlPlaneAccessAPIMethods(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	profile := DefaultProfile("control-plane-access")
	profile.FrontService.ServerName = "localhost"
	created, err := store.Create(profile)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	created.HTTPBehavior.AllowedMethods = []string{"GET", "POST", "HEAD", "OPTIONS"}
	updated, err := store.Update(created)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	required := []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
	for _, method := range required {
		if !slices.Contains(updated.HTTPBehavior.AllowedMethods, method) {
			t.Fatalf("expected method %s after auto repair, got %+v", method, updated.HTTPBehavior.AllowedMethods)
		}
	}
}

func TestDefaultProfile_ControlPlaneAccessAddsCustomRouteLimits(t *testing.T) {
	profile := DefaultProfile("control-plane-access")
	if len(profile.SecurityBehaviorAndLimits.CustomLimitRules) == 0 {
		t.Fatal("expected management profile to include custom route limits")
	}
	if profile.SecurityBehaviorAndLimits.CustomLimitRules[0].Path == "" || profile.SecurityBehaviorAndLimits.CustomLimitRules[0].Rate == "" {
		t.Fatalf("expected non-empty custom route limit entries, got %+v", profile.SecurityBehaviorAndLimits.CustomLimitRules)
	}
}
