package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/wafpolicies"
)

type fakeEasySiteProfileStore struct {
	items map[string]easysiteprofiles.EasySiteProfile
}

func (f *fakeEasySiteProfileStore) Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error) {
	item, ok := f.items[siteID]
	return item, ok, nil
}

func (f *fakeEasySiteProfileStore) Create(profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	f.items[profile.SiteID] = profile
	return profile, nil
}

func (f *fakeEasySiteProfileStore) Update(profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	f.items[profile.SiteID] = profile
	return profile, nil
}

type fakeEasyWAFStore struct {
	items []wafpolicies.WAFPolicy
}

func (f *fakeEasyWAFStore) List() ([]wafpolicies.WAFPolicy, error) {
	return append([]wafpolicies.WAFPolicy(nil), f.items...), nil
}
func (f *fakeEasyWAFStore) Create(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeEasyWAFStore) Update(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	f.items = append(f.items, item)
	return item, nil
}

type fakeEasyAccessStore struct {
	items []accesspolicies.AccessPolicy
}

func (f *fakeEasyAccessStore) List() ([]accesspolicies.AccessPolicy, error) {
	return append([]accesspolicies.AccessPolicy(nil), f.items...), nil
}
func (f *fakeEasyAccessStore) Create(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeEasyAccessStore) Update(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	f.items = append(f.items, item)
	return item, nil
}

type fakeEasyRateStore struct {
	items []ratelimitpolicies.RateLimitPolicy
}

func (f *fakeEasyRateStore) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return append([]ratelimitpolicies.RateLimitPolicy(nil), f.items...), nil
}
func (f *fakeEasyRateStore) Create(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeEasyRateStore) Update(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	f.items = append(f.items, item)
	return item, nil
}

type fakeEasyCompileService struct {
	calls int
}

func (f *fakeEasyCompileService) Create(ctx context.Context) (CompileRequestResult, error) {
	f.calls++
	return CompileRequestResult{Revision: revisions.Revision{ID: "rev-000001"}}, nil
}

type fakeEasyApplyService struct {
	calls int
}

func (f *fakeEasyApplyService) Apply(ctx context.Context, revisionID string) (jobs.Job, error) {
	f.calls++
	return jobs.Job{ID: "apply-" + revisionID, Status: jobs.StatusSucceeded, Result: "revision applied"}, nil
}

func TestEasySiteProfileService_GetReturnsDefaultWhenMissing(t *testing.T) {
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		nil,
		nil,
		nil,
	)

	item, err := service.Get("site-a")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if item.SiteID != "site-a" {
		t.Fatalf("unexpected site_id: %s", item.SiteID)
	}
	if item.FrontService.ServerName != "www.example.com" {
		t.Fatalf("expected server_name from site host, got %s", item.FrontService.ServerName)
	}
}

func TestEasySiteProfileService_UpsertCreatesThenUpdates(t *testing.T) {
	store := &fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}}
	service := NewEasySiteProfileService(
		store,
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		nil,
		nil,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("site-a")
	item.FrontService.ServerName = "www.example.com"
	created, err := service.Upsert(context.Background(), item)
	if err != nil {
		t.Fatalf("upsert create failed: %v", err)
	}
	if created.SiteID != "site-a" {
		t.Fatalf("unexpected created profile: %+v", created)
	}

	item.SecurityAntibot.AntibotChallenge = easysiteprofiles.AntibotChallengeCookie
	updated, err := service.Upsert(context.Background(), item)
	if err != nil {
		t.Fatalf("upsert update failed: %v", err)
	}
	if updated.SecurityAntibot.AntibotChallenge != easysiteprofiles.AntibotChallengeCookie {
		t.Fatalf("unexpected antibot mode: %s", updated.SecurityAntibot.AntibotChallenge)
	}
}

func TestEasySiteProfileService_UpsertRejectsMissingSite(t *testing.T) {
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		nil,
		nil,
		nil,
	)
	_, err := service.Upsert(context.Background(), easysiteprofiles.DefaultProfile("site-a"))
	if err == nil {
		t.Fatal("expected missing site error")
	}
}

func TestEasySiteProfileService_GetLoadsLegacyPolicies(t *testing.T) {
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{items: []wafpolicies.WAFPolicy{{ID: "waf-a", SiteID: "site-a", Enabled: true, CRSEnabled: true, CustomRuleIncludes: []string{"plugin-a"}}}},
		&fakeEasyAccessStore{items: []accesspolicies.AccessPolicy{{ID: "access-a", SiteID: "site-a", DenyList: []string{"203.0.113.10"}}}},
		&fakeEasyRateStore{items: []ratelimitpolicies.RateLimitPolicy{{ID: "rate-a", SiteID: "site-a", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 25, Burst: 5}}}},
		nil,
		nil,
		nil,
	)

	item, err := service.Get("site-a")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !item.SecurityModSecurity.UseModSecurity || !item.SecurityModSecurity.UseModSecurityCRSPlugins {
		t.Fatalf("expected modsecurity bridge from legacy policy: %+v", item.SecurityModSecurity)
	}
	if item.SecurityBehaviorAndLimits.LimitReqRate != "25r/s" {
		t.Fatalf("expected bridged rate, got %s", item.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if len(item.SecurityBehaviorAndLimits.BlacklistIP) != 1 || item.SecurityBehaviorAndLimits.BlacklistIP[0] != "203.0.113.10" {
		t.Fatalf("expected bridged blacklist ip, got %+v", item.SecurityBehaviorAndLimits.BlacklistIP)
	}
}

func TestEasySiteProfileService_UpsertSyncsLegacyPolicies(t *testing.T) {
	wafStore := &fakeEasyWAFStore{}
	accessStore := &fakeEasyAccessStore{}
	rateStore := &fakeEasyRateStore{}
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		wafStore,
		accessStore,
		rateStore,
		nil,
		nil,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("site-a")
	item.FrontService.SecurityMode = easysiteprofiles.SecurityModeBlock
	item.SecurityBehaviorAndLimits.UseBlacklist = true
	item.SecurityBehaviorAndLimits.BlacklistIP = []string{"203.0.113.20"}
	item.SecurityBehaviorAndLimits.UseLimitReq = true
	item.SecurityBehaviorAndLimits.LimitReqRate = "15r/s"
	item.SecurityModSecurity.UseModSecurity = true
	item.SecurityModSecurity.UseModSecurityCRSPlugins = true
	item.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-x"}

	if _, err := service.Upsert(context.Background(), item); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	if len(wafStore.items) != 1 || wafStore.items[0].SiteID != "site-a" || wafStore.items[0].Mode != wafpolicies.ModePrevention {
		t.Fatalf("unexpected synced waf policy: %+v", wafStore.items)
	}
	if len(accessStore.items) != 1 || len(accessStore.items[0].DenyList) != 1 || accessStore.items[0].DenyList[0] != "203.0.113.20" {
		t.Fatalf("unexpected synced access policy: %+v", accessStore.items)
	}
	if len(rateStore.items) != 1 || rateStore.items[0].Limits.RequestsPerSecond != 15 {
		t.Fatalf("unexpected synced rate policy: %+v", rateStore.items)
	}
}

func TestEasySiteProfileService_UpsertUpdatesExistingEasyLegacyPolicies(t *testing.T) {
	wafStore := &fakeEasyWAFStore{
		items: []wafpolicies.WAFPolicy{{
			ID:      "easy-site-a-waf",
			SiteID:  "site-a",
			Enabled: true,
			Mode:    wafpolicies.ModeDetection,
		}},
	}
	accessStore := &fakeEasyAccessStore{
		items: []accesspolicies.AccessPolicy{{
			ID:      "easy-site-a-access",
			SiteID:  "site-a",
			Enabled: true,
		}},
	}
	rateStore := &fakeEasyRateStore{
		items: []ratelimitpolicies.RateLimitPolicy{{
			ID:      "easy-site-a-rate",
			SiteID:  "site-a",
			Enabled: true,
			Limits: ratelimitpolicies.Limits{
				RequestsPerSecond: 25,
				Burst:             25,
			},
		}},
	}
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		wafStore,
		accessStore,
		rateStore,
		nil,
		nil,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("site-a")
	item.SecurityBehaviorAndLimits.UseLimitReq = false
	item.SecurityBehaviorAndLimits.UseBlacklist = false
	item.SecurityModSecurity.UseModSecurity = false

	if _, err := service.Upsert(context.Background(), item); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if len(wafStore.items) != 1 {
		t.Fatalf("expected existing waf policy to be updated in place, got %+v", wafStore.items)
	}
	if len(accessStore.items) != 1 {
		t.Fatalf("expected existing access policy to be updated in place, got %+v", accessStore.items)
	}
	if len(rateStore.items) != 1 {
		t.Fatalf("expected existing rate policy to be updated in place, got %+v", rateStore.items)
	}
}

func TestEasySiteProfileService_UpsertDisablesManagementRatePolicies(t *testing.T) {
	rateStore := &fakeEasyRateStore{
		items: []ratelimitpolicies.RateLimitPolicy{
			{
				ID:      "easy-control-plane-access-rate",
				SiteID:  "control-plane-access",
				Enabled: true,
				Limits: ratelimitpolicies.Limits{
					RequestsPerSecond: 25,
					Burst:             25,
				},
			},
			{
				ID:      "manual-control-plane-access-rate",
				SiteID:  "control-plane-access",
				Enabled: true,
				Limits: ratelimitpolicies.Limits{
					RequestsPerSecond: 50,
					Burst:             50,
				},
			},
		},
	}
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "control-plane-access", PrimaryHost: "localhost"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		rateStore,
		nil,
		nil,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("control-plane-access")
	if !item.SecurityBehaviorAndLimits.UseLimitReq {
		t.Fatal("expected default control-plane-access profile to keep request limit enabled")
	}
	if _, err := service.Upsert(context.Background(), item); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	for _, policy := range rateStore.items {
		if policy.SiteID == "control-plane-access" && !policy.Enabled {
			t.Fatalf("expected management site rate policy to stay enabled, got %+v", policy)
		}
	}
}

func TestEasySiteProfileService_GetMasksSecrets(t *testing.T) {
	store := &fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}}
	item := easysiteprofiles.DefaultProfile("site-a")
	item.SecurityAntibot.AntibotRecaptchaSecret = "recaptcha-secret"
	item.SecurityAntibot.AntibotHcaptchaSecret = "hcaptcha-secret"
	item.SecurityAntibot.AntibotTurnstileSecret = "turnstile-secret"
	item.SecurityAuthBasic.AuthBasicPassword = "basic-password"
	store.items["site-a"] = item

	service := NewEasySiteProfileService(
		store,
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		nil,
		nil,
		nil,
	)

	got, err := service.Get("site-a")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.SecurityAntibot.AntibotRecaptchaSecret != "********" {
		t.Fatalf("expected masked recaptcha secret, got %q", got.SecurityAntibot.AntibotRecaptchaSecret)
	}
	if got.SecurityAntibot.AntibotHcaptchaSecret != "********" {
		t.Fatalf("expected masked hcaptcha secret, got %q", got.SecurityAntibot.AntibotHcaptchaSecret)
	}
	if got.SecurityAntibot.AntibotTurnstileSecret != "********" {
		t.Fatalf("expected masked turnstile secret, got %q", got.SecurityAntibot.AntibotTurnstileSecret)
	}
	if got.SecurityAuthBasic.AuthBasicPassword != "********" {
		t.Fatalf("expected masked basic auth password, got %q", got.SecurityAuthBasic.AuthBasicPassword)
	}
}

func TestEasySiteProfileService_UpsertKeepsStoredSecretWhenMaskedSentBack(t *testing.T) {
	store := &fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}}
	existing := easysiteprofiles.DefaultProfile("site-a")
	existing.SecurityAntibot.AntibotRecaptchaSecret = "recaptcha-secret"
	existing.SecurityAuthBasic.AuthBasicPassword = "basic-password"
	store.items["site-a"] = existing

	service := NewEasySiteProfileService(
		store,
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		nil,
		nil,
		nil,
	)

	incoming := existing
	incoming.SecurityAntibot.AntibotRecaptchaSecret = "********"
	incoming.SecurityAuthBasic.AuthBasicPassword = "********"

	updated, err := service.Upsert(context.Background(), incoming)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if updated.SecurityAntibot.AntibotRecaptchaSecret != "********" {
		t.Fatalf("expected masked output recaptcha secret, got %q", updated.SecurityAntibot.AntibotRecaptchaSecret)
	}
	if updated.SecurityAuthBasic.AuthBasicPassword != "********" {
		t.Fatalf("expected masked output basic password, got %q", updated.SecurityAuthBasic.AuthBasicPassword)
	}

	stored := store.items["site-a"]
	if stored.SecurityAntibot.AntibotRecaptchaSecret != "recaptcha-secret" {
		t.Fatalf("expected stored recaptcha secret to be preserved, got %q", stored.SecurityAntibot.AntibotRecaptchaSecret)
	}
	if stored.SecurityAuthBasic.AuthBasicPassword != "basic-password" {
		t.Fatalf("expected stored basic password to be preserved, got %q", stored.SecurityAuthBasic.AuthBasicPassword)
	}
}

func TestEasySiteProfileService_UpsertTriggersCompileAndApply(t *testing.T) {
	compile := &fakeEasyCompileService{}
	apply := &fakeEasyApplyService{}
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		compile,
		apply,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("site-a")
	if _, err := service.Upsert(context.Background(), item); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if compile.calls != 1 {
		t.Fatalf("expected compile call, got %d", compile.calls)
	}
	if apply.calls != 1 {
		t.Fatalf("expected apply call, got %d", apply.calls)
	}
}

func TestEasySiteProfileService_UpsertSkipsCompileAndApplyWhenDisabledInContext(t *testing.T) {
	compile := &fakeEasyCompileService{}
	apply := &fakeEasyApplyService{}
	service := NewEasySiteProfileService(
		&fakeEasySiteProfileStore{items: map[string]easysiteprofiles.EasySiteProfile{}},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a", PrimaryHost: "www.example.com"}}},
		&fakeEasyWAFStore{},
		&fakeEasyAccessStore{},
		&fakeEasyRateStore{},
		compile,
		apply,
		nil,
	)

	item := easysiteprofiles.DefaultProfile("site-a")
	if _, err := service.Upsert(withAutoApplyDisabled(context.Background()), item); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if compile.calls != 0 {
		t.Fatalf("expected compile call to be skipped, got %d", compile.calls)
	}
	if apply.calls != 0 {
		t.Fatalf("expected apply call to be skipped, got %d", apply.calls)
	}
}
