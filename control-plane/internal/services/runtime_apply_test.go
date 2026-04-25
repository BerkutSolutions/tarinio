package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
)

type fakeRevisionStoreForApply struct {
	revision revisions.Revision
	active   string
	failed   string
	apply    struct {
		revisionID string
		jobID      string
		status     string
		result     string
		appliedAt  string
	}
}

func (f *fakeRevisionStoreForApply) Get(revisionID string) (revisions.Revision, bool, error) {
	if f.revision.ID == revisionID {
		return f.revision, true, nil
	}
	return revisions.Revision{}, false, nil
}

func (f *fakeRevisionStoreForApply) MarkActive(revisionID string) error {
	f.active = revisionID
	return nil
}

func (f *fakeRevisionStoreForApply) MarkFailed(revisionID string) error {
	f.failed = revisionID
	return nil
}

func (f *fakeRevisionStoreForApply) RecordApplyResult(revisionID string, jobID string, status string, result string, appliedAt string) error {
	f.apply.revisionID = revisionID
	f.apply.jobID = jobID
	f.apply.status = status
	f.apply.result = result
	f.apply.appliedAt = appliedAt
	return nil
}

type fakeSnapshotReader struct {
	snapshot revisionsnapshots.Snapshot
	files    map[string][]byte
}

func (f *fakeSnapshotReader) Load(snapshotPath string) (revisionsnapshots.Snapshot, error) {
	return f.snapshot, nil
}

func (f *fakeSnapshotReader) ReadMaterial(ref string) ([]byte, error) {
	content, ok := f.files[ref]
	if !ok {
		return nil, errors.New("not found")
	}
	return content, nil
}

type fakeApplyExecutor struct {
	syntaxCalls int
	reloadCalls int
	reloadErr   error
}

func (f *fakeApplyExecutor) Run(name string, args []string, workdir string) error {
	if len(args) > 0 && args[0] == "-t" {
		f.syntaxCalls++
		return nil
	}
	if len(args) > 0 && args[len(args)-1] == "reload" {
		f.reloadCalls++
		return f.reloadErr
	}
	return nil
}

type fakeApplyHealthChecker struct {
	err   error
	calls int
}

func (f *fakeApplyHealthChecker) Check(active *pipeline.ActivePointer) error {
	f.calls++
	return f.err
}

func TestApplyService_ApplyUsesRevisionSnapshotAndMarksActive(t *testing.T) {
	root := t.TempDir()
	revisionStore := &fakeRevisionStoreForApply{
		revision: revisions.Revision{
			ID:         "rev-000001",
			Version:    1,
			CreatedAt:  "2026-04-01T00:00:00Z",
			Checksum:   "abc",
			BundlePath: "snapshots/rev-000001.json",
			Status:     revisions.StatusPending,
		},
	}
	jobStore, err := jobs.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatalf("new job store: %v", err)
	}
	snapshotReader := &fakeSnapshotReader{
		snapshot: revisionsnapshots.Snapshot{
			Sites:             []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
			Upstreams:         []upstreams.Upstream{{ID: "upstream-a", SiteID: "site-a", Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
			TLSConfigs:        []tlsconfigs.TLSConfig{{SiteID: "site-a", CertificateID: "cert-a"}},
			AccessPolicies:    []accesspolicies.AccessPolicy{{ID: "access-a", SiteID: "site-a"}},
			RateLimitPolicies: []ratelimitpolicies.RateLimitPolicy{{ID: "rate-a", SiteID: "site-a", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 10, Burst: 5}}},
			CertificateMaterials: []revisionsnapshots.CertificateMaterialSnapshot{{
				CertificateID:  "cert-a",
				CertificateRef: "revision-snapshots/files/rev-000001/cert-a/certificate.pem",
				PrivateKeyRef:  "revision-snapshots/files/rev-000001/cert-a/private.key",
			}},
		},
		files: map[string][]byte{
			"revision-snapshots/files/rev-000001/cert-a/certificate.pem": []byte("CERT"),
			"revision-snapshots/files/rev-000001/cert-a/private.key":     []byte("KEY"),
		},
	}
	exec := &fakeApplyExecutor{}
	health := &fakeApplyHealthChecker{}
	eventStore := &fakeEventStore{}
	eventService := NewEventService(eventStore)

	service := NewApplyService(root, revisionStore, snapshotReader, jobStore, eventService, exec, exec, health, nil)

	job, err := service.Apply(context.Background(), "rev-000001")
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if job.Status != jobs.StatusSucceeded || job.Type != jobs.TypeApply {
		t.Fatalf("unexpected job: %+v", job)
	}
	if revisionStore.active != "rev-000001" {
		t.Fatalf("expected active revision to be updated, got %s", revisionStore.active)
	}
	if revisionStore.apply.revisionID != "rev-000001" || revisionStore.apply.status != string(jobs.StatusSucceeded) || revisionStore.apply.result != "revision applied" || revisionStore.apply.appliedAt == "" {
		t.Fatalf("expected apply metadata to be recorded, got %+v", revisionStore.apply)
	}
	if exec.syntaxCalls != 1 || exec.reloadCalls != 1 || health.calls != 1 {
		t.Fatalf("unexpected pipeline call counts: syntax=%d reload=%d health=%d", exec.syntaxCalls, exec.reloadCalls, health.calls)
	}
	content, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		t.Fatalf("read active pointer: %v", err)
	}
	if !strings.Contains(string(content), "rev-000001") {
		t.Fatalf("expected active pointer to contain revision id, got %s", string(content))
	}
	if len(eventStore.items) != 2 || eventStore.items[0].Type != events.TypeApplyStarted || eventStore.items[1].Type != events.TypeApplySucceeded {
		t.Fatalf("unexpected success events: %+v", eventStore.items)
	}
}

func TestApplyService_RollsBackToKnownGoodOnHealthFailure(t *testing.T) {
	root := t.TempDir()
	candidatePath := filepath.Join(root, "candidates", "rev-good")
	if err := os.MkdirAll(candidatePath, 0o755); err != nil {
		t.Fatalf("mkdir candidate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(candidatePath, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := (pipeline.AtomicActivator{Root: root}).Activate("rev-good"); err != nil {
		t.Fatalf("activate known-good: %v", err)
	}

	revisionStore := &fakeRevisionStoreForApply{
		revision: revisions.Revision{
			ID:         "rev-bad",
			Version:    2,
			CreatedAt:  "2026-04-01T00:00:00Z",
			Checksum:   "abc",
			BundlePath: "snapshots/rev-bad.json",
			Status:     revisions.StatusPending,
		},
	}
	jobStore, err := jobs.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatalf("new job store: %v", err)
	}
	snapshotReader := &fakeSnapshotReader{
		snapshot: revisionsnapshots.Snapshot{
			Sites:     []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
			Upstreams: []upstreams.Upstream{{ID: "upstream-a", SiteID: "site-a", Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
		},
	}
	exec := &fakeApplyExecutor{}
	health := &fakeApplyHealthChecker{err: errors.New("unhealthy")}
	eventStore := &fakeEventStore{}
	eventService := NewEventService(eventStore)
	service := NewApplyService(root, revisionStore, snapshotReader, jobStore, eventService, exec, exec, health, nil)

	job, err := service.Apply(context.Background(), "rev-bad")
	if err != nil {
		t.Fatalf("apply returned store error: %v", err)
	}
	if job.Status != jobs.StatusFailed {
		t.Fatalf("expected failed job, got %+v", job)
	}
	if revisionStore.failed != "rev-bad" {
		t.Fatalf("expected failed revision mark, got %s", revisionStore.failed)
	}
	if revisionStore.apply.revisionID != "rev-bad" || revisionStore.apply.status != string(jobs.StatusFailed) || revisionStore.apply.appliedAt == "" {
		t.Fatalf("expected failed apply metadata to be recorded, got %+v", revisionStore.apply)
	}
	content, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		t.Fatalf("read active pointer: %v", err)
	}
	if !strings.Contains(string(content), "rev-good") {
		t.Fatalf("expected rollback to known-good, got %s", string(content))
	}
	if len(eventStore.items) != 4 {
		t.Fatalf("expected 4 events, got %+v", eventStore.items)
	}
	if eventStore.items[1].Type != events.TypeHealthCheckFailed || eventStore.items[2].Type != events.TypeRollbackPerformed || eventStore.items[3].Type != events.TypeApplyFailed {
		t.Fatalf("unexpected failure events: %+v", eventStore.items)
	}
}

func TestApplyService_CompilesEasyProfileArtifacts(t *testing.T) {
	root := t.TempDir()
	revisionStore := &fakeRevisionStoreForApply{
		revision: revisions.Revision{
			ID:         "rev-easy",
			Version:    3,
			CreatedAt:  "2026-04-01T00:00:00Z",
			Checksum:   "abc",
			BundlePath: "snapshots/rev-easy.json",
			Status:     revisions.StatusPending,
		},
	}
	jobStore, err := jobs.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatalf("new job store: %v", err)
	}
	easy := easysiteprofiles.DefaultProfile("site-a")
	easy.HTTPBehavior.AllowedMethods = []string{"GET", "POST", "DELETE"}
	easy.HTTPBehavior.MaxClientSize = "64m"
	easy.HTTPHeaders.ReferrerPolicy = "same-origin"
	easy.HTTPHeaders.ContentSecurityPolicy = "default-src 'self';"
	easy.HTTPHeaders.PermissionsPolicy = []string{"geolocation=()", "camera=()"}
	easy.HTTPHeaders.UseCORS = true
	easy.HTTPHeaders.CORSAllowedOrigins = []string{"https://app.example.com"}
	easy.UpstreamRouting.ReverseProxyCustomHost = "backend.internal"
	easy.UpstreamRouting.ReverseProxySSLSNI = true
	easy.UpstreamRouting.ReverseProxySSLSNIName = "backend.internal"
	easy.UpstreamRouting.ReverseProxyWebsocket = true
	easy.UpstreamRouting.ReverseProxyKeepalive = true
	easy.SecurityBehaviorAndLimits.BlacklistIP = []string{"203.0.113.7"}
	easy.SecurityBehaviorAndLimits.BlacklistUserAgent = []string{"curl"}
	easy.SecurityBehaviorAndLimits.BlacklistURI = []string{"/admin"}
	easy.SecurityCountryPolicy.WhitelistCountry = []string{"US"}
	easy.SecurityCountryPolicy.BlacklistCountry = []string{"RU"}
	easy.SecurityAuthBasic.UseAuthBasic = true
	easy.SecurityAuthBasic.AuthBasicUser = "admin"
	easy.SecurityAuthBasic.AuthBasicPassword = "secret"
	easy.SecurityAntibot.AntibotChallenge = easysiteprofiles.AntibotChallengeRecaptcha
	easy.SecurityAntibot.AntibotURI = "/challenge"
	easy.SecurityAntibot.AntibotRecaptchaSitekey = "site-key"
	easy.SecurityAntibot.AntibotRecaptchaSecret = "secret-key"
	easy.SecurityModSecurity.UseModSecurity = true
	easy.SecurityModSecurity.UseModSecurityCRSPlugins = true
	easy.SecurityModSecurity.UseCustomConfiguration = true
	easy.SecurityModSecurity.ModSecurityCRSVersion = "4"
	easy.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-a"}
	easy.SecurityModSecurity.CustomConfiguration.Path = "modsec/anomaly_score.conf"
	easy.SecurityModSecurity.CustomConfiguration.Content = "SecRuleEngine On"

	snapshotReader := &fakeSnapshotReader{
		snapshot: revisionsnapshots.Snapshot{
			Sites:             []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
			Upstreams:         []upstreams.Upstream{{ID: "upstream-a", SiteID: "site-a", Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
			RateLimitPolicies: []ratelimitpolicies.RateLimitPolicy{{ID: "rate-a", SiteID: "site-a", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 10, Burst: 5}}},
			EasySiteProfiles:  []easysiteprofiles.EasySiteProfile{easy},
			AntiDDoSSettings: antiddos.Settings{
				UseL4Guard:    true,
				ChainMode:     antiddos.ChainModeInput,
				ConnLimit:     333,
				RatePerSecond: 111,
				RateBurst:     222,
				Ports:         []int{443},
				Target:        antiddos.TargetReject,
				EnforceL7Rate: true,
				L7RequestsPS:  40,
				L7Burst:       60,
				L7StatusCode:  429,
			},
		},
	}
	exec := &fakeApplyExecutor{}
	health := &fakeApplyHealthChecker{}
	eventStore := &fakeEventStore{}
	eventService := NewEventService(eventStore)

	service := NewApplyService(root, revisionStore, snapshotReader, jobStore, eventService, exec, exec, health, nil)
	if _, err := service.Apply(context.Background(), "rev-easy"); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	easyConfPath := filepath.Join(root, "candidates", "rev-easy", "nginx", "easy", "site-a.conf")
	easyConfContent, err := os.ReadFile(easyConfPath)
	if err != nil {
		t.Fatalf("read easy conf: %v", err)
	}
	easyConf := string(easyConfContent)
	if !strings.Contains(easyConf, "client_max_body_size 64m;") {
		t.Fatalf("expected max body size directive in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "if ($request_method !~ ^(DELETE|GET|POST)$)") {
		t.Fatalf("expected allowed methods guard in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "add_header Referrer-Policy \"same-origin\" always;") {
		t.Fatalf("expected referrer policy header in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "add_header Content-Security-Policy \"default-src 'self';\" always;") {
		t.Fatalf("expected csp header in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "add_header Permissions-Policy \"camera=(), geolocation=()\" always;") {
		t.Fatalf("expected permissions policy header in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "add_header Access-Control-Allow-Origin \"https://app.example.com\" always;") {
		t.Fatalf("expected cors header in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "proxy_set_header Host backend.internal;") {
		t.Fatalf("expected reverse proxy custom host in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;") {
		t.Fatalf("expected default x-forwarded-for forwarding in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "proxy_set_header X-Forwarded-Proto $scheme;") {
		t.Fatalf("expected default x-forwarded-proto forwarding in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "proxy_set_header X-Real-IP \"\";") {
		t.Fatalf("expected x-real-ip disabled by default in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "proxy_ssl_server_name on;") || !strings.Contains(easyConf, "proxy_ssl_name backend.internal;") {
		t.Fatalf("expected reverse proxy sni directives in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "deny 203.0.113.7;") {
		t.Fatalf("expected blacklist ip in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "if ($waf_blacklist_ua_guard ~* \"^0:.*curl\") { return 403; }") {
		t.Fatalf("expected blacklist user-agent in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "if ($waf_blacklist_uri_guard ~* \"^0:.*/admin\") { return 403; }") {
		t.Fatalf("expected blacklist uri in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "if ($waf_country_guard !~ \"^(?:1:.*|0:(?:|US))$\") { return 403; }") {
		t.Fatalf("expected whitelist country in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "if ($waf_country_guard ~ \"^(?:0:(?:RU))$\") { return 403; }") {
		t.Fatalf("expected blacklist country in easy conf, got: %s", easyConf)
	}
	countryPos := strings.Index(easyConf, "if ($waf_country_guard !~")
	antibotPos := strings.Index(easyConf, "set $waf_antibot_exception_guard 0;")
	if countryPos == -1 || antibotPos == -1 || countryPos > antibotPos {
		t.Fatalf("expected country hard-block guard to run before antibot challenge, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "set $waf_antibot_scanner_guard \"$waf_easy_exception_guard:$request_uri:$http_user_agent\";") {
		t.Fatalf("expected antibot scanner auto-ban guard in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "add_header X-WAF-Antibot-Mode \"$waf_antibot_effective_challenge\" always;") {
		t.Fatalf("expected antibot directive in easy conf, got: %s", easyConf)
	}
	if !strings.Contains(easyConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/site-a.conf;") {
		t.Fatalf("expected modsecurity include in easy conf, got: %s", easyConf)
	}

	modsecConfPath := filepath.Join(root, "candidates", "rev-easy", "modsecurity", "easy", "site-a.conf")
	modsecConfContent, err := os.ReadFile(modsecConfPath)
	if err != nil {
		t.Fatalf("read modsecurity easy conf: %v", err)
	}
	modsecConf := string(modsecConfContent)
	if !strings.Contains(modsecConf, "SecRuleEngine On") {
		t.Fatalf("expected custom modsecurity content, got: %s", modsecConf)
	}
	if !strings.Contains(modsecConf, "Include /etc/waf/modsecurity/crs-overrides/plugin-a.conf") {
		t.Fatalf("expected crs plugin include in modsecurity conf, got: %s", modsecConf)
	}

	l4GuardPath := filepath.Join(root, "candidates", "rev-easy", "l4guard", "config.json")
	l4GuardContent, err := os.ReadFile(l4GuardPath)
	if err != nil {
		t.Fatalf("read l4guard config: %v", err)
	}
	if !strings.Contains(string(l4GuardContent), "\"conn_limit\": 333") || !strings.Contains(string(l4GuardContent), "\"rate_per_second\": 111") {
		t.Fatalf("unexpected l4guard config: %s", string(l4GuardContent))
	}

	rateHTTPPath := filepath.Join(root, "candidates", "rev-easy", "nginx", "conf.d", "ratelimits.conf")
	rateHTTPContent, err := os.ReadFile(rateHTTPPath)
	if err != nil {
		t.Fatalf("read ratelimits http config: %v", err)
	}
	if !strings.Contains(string(rateHTTPContent), "rate=40r/s") {
		t.Fatalf("expected global anti-ddos l7 rate override in ratelimits config, got: %s", string(rateHTTPContent))
	}
}

func TestMapSiteUpstreamInputs_RespectsHostHeaderToggle(t *testing.T) {
	siteInputs, upstreamInputs := mapSiteUpstreamInputs(
		[]sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
		[]upstreams.Upstream{{ID: "upstream-a", SiteID: "site-a", Host: "127.0.0.1", Port: 8080, Scheme: "http"}},
		nil,
		[]easysiteprofiles.EasySiteProfile{{
			SiteID: "site-a",
			UpstreamRouting: easysiteprofiles.UpstreamRoutingSettings{
				DisableHostHeader: true,
			},
		}},
	)
	if len(siteInputs) != 1 || len(upstreamInputs) != 1 {
		t.Fatalf("unexpected mapped inputs: sites=%d upstreams=%d", len(siteInputs), len(upstreamInputs))
	}
	if upstreamInputs[0].PassHostHeader {
		t.Fatalf("expected host header forwarding disabled, got %+v", upstreamInputs[0])
	}
}

func TestMapRateLimitInputs_KeepsManagementSiteEnabled(t *testing.T) {
	input := []ratelimitpolicies.RateLimitPolicy{
		{ID: "rate-mgmt", SiteID: "control-plane-access", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 25, Burst: 25}},
		{ID: "rate-site-a", SiteID: "site-a", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 10, Burst: 5}},
	}

	got := mapRateLimitInputs(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 mapped items, got %d", len(got))
	}
	for _, item := range got {
		if item.SiteID == "control-plane-access" && !item.Enabled {
			t.Fatalf("expected management site rate limit to stay enabled, got %+v", item)
		}
		if item.SiteID == "site-a" && !item.Enabled {
			t.Fatalf("expected non-management rate limit to stay enabled, got %+v", item)
		}
	}
}

func TestMapAccessInputs_AllowlistDefaultsToDenyForAllSites(t *testing.T) {
	got := mapAccessInputs([]accesspolicies.AccessPolicy{
		{ID: "site-a-access", SiteID: "site-a", AllowList: []string{"10.0.0.0/24"}},
		{ID: "site-b-access", SiteID: "site-b", AllowList: nil},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 mapped access policies, got %d", len(got))
	}
	bySite := map[string]pipeline.AccessPolicyInput{}
	for _, item := range got {
		bySite[item.SiteID] = item
	}
	if bySite["site-a"].DefaultAction != "deny" {
		t.Fatalf("expected allowlist-backed site to default deny, got %+v", bySite["site-a"])
	}
	if bySite["site-b"].DefaultAction != "allow" {
		t.Fatalf("expected site without allowlist to default allow, got %+v", bySite["site-b"])
	}
}

func TestApplyAntiDDoSRateOverrides_SkipsManagementSite(t *testing.T) {
	siteInputs := []pipeline.SiteInput{
		{ID: "control-plane-access", Enabled: true},
		{ID: "site-a", Enabled: true},
	}
	items := []pipeline.RateLimitPolicyInput{
		{ID: "legacy-management", SiteID: "control-plane-access", Enabled: true, Requests: 25, WindowSeconds: 1, Burst: 25, StatusCode: 429},
	}
	settings := antiddos.Settings{
		EnforceL7Rate: true,
		L7RequestsPS:  40,
		L7Burst:       60,
		L7StatusCode:  429,
	}

	got := applyAntiDDoSRateOverrides(siteInputs, items, settings)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (legacy management + site-a override), got %d", len(got))
	}
	for _, item := range got {
		if item.SiteID == "control-plane-access" && item.ID != "legacy-management" {
			t.Fatalf("expected management policy to stay unchanged, got %+v", item)
		}
		if item.SiteID == "site-a" && item.ID != "antiddos-global-site-a" {
			t.Fatalf("expected anti-ddos override for site-a, got %+v", item)
		}
	}
}

func TestAdaptiveModelScopeUsesPerSiteFlag(t *testing.T) {
	profiles := []easysiteprofiles.EasySiteProfile{
		{
			SiteID: "control-plane-access",
			FrontService: easysiteprofiles.FrontServiceSettings{
				ServerName:           "waf.example.test",
				AdaptiveModelEnabled: true,
			},
		},
		{
			SiteID: "imported-site",
			FrontService: easysiteprofiles.FrontServiceSettings{
				ServerName:           "app.example.test",
				AdaptiveModelEnabled: false,
			},
		},
	}

	enabled, sites := adaptiveModelScope(true, profiles)
	if !enabled {
		t.Fatal("expected adaptive model to stay enabled when at least one site opted in")
	}
	want := []string{"control-plane-access", "waf.example.test"}
	if !reflect.DeepEqual(sites, want) {
		t.Fatalf("expected enabled sites %+v, got %+v", want, sites)
	}

	enabled, sites = adaptiveModelScope(true, profiles[1:])
	if enabled || len(sites) != 0 {
		t.Fatalf("expected adaptive model disabled without opted-in sites, got enabled=%t sites=%+v", enabled, sites)
	}
}
