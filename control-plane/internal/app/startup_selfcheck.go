package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
)

func (a *App) RunStartupSelfTest(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("startup self-test: app is nil")
	}
	if !a.Config.StartupSelfTest {
		return nil
	}
	if a.SiteService == nil ||
		a.UpstreamService == nil ||
		a.TLSConfigService == nil ||
		a.EasySiteProfileService == nil ||
		a.EasySiteProfileStore == nil ||
		a.SiteStore == nil ||
		a.UpstreamStore == nil ||
		a.CertificateStore == nil ||
		a.TLSConfigStore == nil ||
		a.WAFPolicyStore == nil ||
		a.AccessPolicyStore == nil ||
		a.RateLimitPolicyStore == nil ||
		a.RevisionCompileService == nil {
		return fmt.Errorf("startup self-test: dependencies are not wired")
	}

	testCtx := services.WithAutoApplyDisabled(services.WithAuditDisabled(ctx))
	if err := a.cleanupStartupSelfTestArtifacts(); err != nil {
		return fmt.Errorf("startup self-test: cleanup stale artifacts failed: %w", err)
	}

	siteID := fmt.Sprintf("startup-self-test-%d", time.Now().UTC().UnixNano())
	upstreamID := siteID + "-upstream"
	certInitialID := siteID + "-cert-initial"
	certUpdatedID := siteID + "-cert-updated"
	site := sites.Site{
		ID:          siteID,
		PrimaryHost: fmt.Sprintf("%s.example.test", siteID),
		Enabled:     true,
	}

	createdSite, err := a.SiteService.Create(testCtx, site)
	if err != nil {
		return fmt.Errorf("startup self-test: create site failed: %w", err)
	}

	cleanupSite := createdSite.ID
	defer func() {
		if cleanupSite != "" {
			_ = a.EasySiteProfileStore.Delete(cleanupSite)
			_ = a.deletePoliciesBySiteID(cleanupSite)
			_ = a.deleteTLSConfigIfExists(testCtx, cleanupSite)
			_ = a.deleteUpstreamsBySiteID(testCtx, cleanupSite)
			_ = a.SiteService.Delete(testCtx, cleanupSite)
		}
		_ = a.CertificateStore.Delete(certInitialID)
		_ = a.CertificateStore.Delete(certUpdatedID)
	}()

	createdUpstream, err := a.UpstreamService.Create(testCtx, upstreams.Upstream{
		ID:     upstreamID,
		SiteID: siteID,
		Host:   "upstream-a.internal",
		Port:   8080,
		Scheme: "http",
	})
	if err != nil {
		return fmt.Errorf("startup self-test: create upstream failed: %w", err)
	}
	createdUpstream.Host = "upstream-b.internal"
	createdUpstream.Port = 8443
	createdUpstream.Scheme = "https"
	updatedUpstream, err := a.UpstreamService.Update(testCtx, createdUpstream)
	if err != nil {
		return fmt.Errorf("startup self-test: update upstream failed: %w", err)
	}
	if updatedUpstream.Scheme != "https" || updatedUpstream.Host != "upstream-b.internal" || updatedUpstream.Port != 8443 {
		return fmt.Errorf("startup self-test: upstream values mismatch after update")
	}

	now := time.Now().UTC()
	if _, err := a.CertificateStore.Create(certificates.Certificate{
		ID:         certInitialID,
		CommonName: createdSite.PrimaryHost,
		SANList:    []string{createdSite.PrimaryHost},
		Status:     "active",
		NotBefore:  now.Add(-time.Hour).Format(time.RFC3339),
		NotAfter:   now.Add(24 * time.Hour).Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("startup self-test: create initial certificate failed: %w", err)
	}
	if _, err := a.CertificateStore.Create(certificates.Certificate{
		ID:         certUpdatedID,
		CommonName: "updated-" + createdSite.PrimaryHost,
		SANList:    []string{"updated-" + createdSite.PrimaryHost},
		Status:     "active",
		NotBefore:  now.Add(-time.Hour).Format(time.RFC3339),
		NotAfter:   now.Add(48 * time.Hour).Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("startup self-test: create updated certificate failed: %w", err)
	}

	if _, err := a.TLSConfigService.Create(testCtx, tlsconfigs.TLSConfig{
		SiteID:        siteID,
		CertificateID: certInitialID,
	}); err != nil {
		return fmt.Errorf("startup self-test: create tls config failed: %w", err)
	}
	updatedTLS, err := a.TLSConfigService.Update(testCtx, tlsconfigs.TLSConfig{
		SiteID:        siteID,
		CertificateID: certUpdatedID,
	})
	if err != nil {
		return fmt.Errorf("startup self-test: update tls config failed: %w", err)
	}
	if updatedTLS.CertificateID != certUpdatedID {
		return fmt.Errorf("startup self-test: tls certificate mismatch after update")
	}

	createdSite.PrimaryHost = fmt.Sprintf("updated-%s.example.test", siteID)
	updatedSite, err := a.SiteService.Update(testCtx, createdSite)
	if err != nil {
		return fmt.Errorf("startup self-test: update site failed: %w", err)
	}
	if updatedSite.PrimaryHost != createdSite.PrimaryHost {
		return fmt.Errorf("startup self-test: site primary_host mismatch after update")
	}

	createProfile := startupSelfTestCreateProfile(siteID, updatedSite.PrimaryHost)
	if _, err := a.EasySiteProfileService.Upsert(testCtx, createProfile); err != nil {
		return fmt.Errorf("startup self-test: create easy profile failed: %w", err)
	}

	storedCreate, ok, err := a.EasySiteProfileStore.Get(siteID)
	if err != nil {
		return fmt.Errorf("startup self-test: read created easy profile failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("startup self-test: created easy profile not found")
	}
	if err := assertStartupProfileCreate(storedCreate); err != nil {
		return fmt.Errorf("startup self-test: create profile assertion failed: %w", err)
	}

	updateProfile := startupSelfTestUpdateProfile(siteID, updatedSite.PrimaryHost)
	if _, err := a.EasySiteProfileService.Upsert(testCtx, updateProfile); err != nil {
		return fmt.Errorf("startup self-test: update easy profile failed: %w", err)
	}

	storedUpdate, ok, err := a.EasySiteProfileStore.Get(siteID)
	if err != nil {
		return fmt.Errorf("startup self-test: read updated easy profile failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("startup self-test: updated easy profile not found")
	}
	if err := assertStartupProfileUpdate(storedUpdate); err != nil {
		return fmt.Errorf("startup self-test: update profile assertion failed: %w", err)
	}

	snapshotAfterUpdate, err := a.startupSelfTestPreviewSnapshot()
	if err != nil {
		return fmt.Errorf("startup self-test: preview snapshot after update failed: %w", err)
	}
	if err := assertStartupSnapshotContainsServiceArtifacts(snapshotAfterUpdate, siteID, upstreamID, certUpdatedID); err != nil {
		return fmt.Errorf("startup self-test: snapshot after update assertion failed: %w", err)
	}

	if err := a.EasySiteProfileStore.Delete(siteID); err != nil && !isNotFoundError(err) {
		return fmt.Errorf("startup self-test: delete easy profile failed: %w", err)
	}
	if err := a.deletePoliciesBySiteID(siteID); err != nil {
		return fmt.Errorf("startup self-test: delete policies failed: %w", err)
	}
	if err := a.deleteTLSConfigIfExists(testCtx, siteID); err != nil {
		return fmt.Errorf("startup self-test: delete tls config failed: %w", err)
	}
	if err := a.deleteUpstreamsBySiteID(testCtx, siteID); err != nil {
		return fmt.Errorf("startup self-test: delete upstreams failed: %w", err)
	}
	if err := a.SiteService.Delete(testCtx, siteID); err != nil {
		return fmt.Errorf("startup self-test: delete site failed: %w", err)
	}
	if err := a.CertificateStore.Delete(certInitialID); err != nil && !isNotFoundError(err) {
		return fmt.Errorf("startup self-test: delete initial certificate failed: %w", err)
	}
	if err := a.CertificateStore.Delete(certUpdatedID); err != nil && !isNotFoundError(err) {
		return fmt.Errorf("startup self-test: delete updated certificate failed: %w", err)
	}
	cleanupSite = ""

	snapshotAfterDelete, err := a.startupSelfTestPreviewSnapshot()
	if err != nil {
		return fmt.Errorf("startup self-test: preview snapshot after delete failed: %w", err)
	}
	if err := assertStartupSnapshotCleaned(snapshotAfterDelete, siteID); err != nil {
		return fmt.Errorf("startup self-test: snapshot cleanup assertion failed: %w", err)
	}
	if err := a.assertNoStartupSelfTestArtifacts(); err != nil {
		return fmt.Errorf("startup self-test: residual artifacts detected: %w", err)
	}

	return nil
}

func (a *App) startupSelfTestPreviewSnapshot() (revisionsnapshots.Snapshot, error) {
	snapshot, err := a.RevisionCompileService.Preview()
	if err != nil {
		return revisionsnapshots.Snapshot{}, err
	}
	return snapshot, nil
}

func (a *App) deletePoliciesBySiteID(siteID string) error {
	wafPolicies, err := a.WAFPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range wafPolicies {
		if policy.SiteID != siteID {
			continue
		}
		if err := a.WAFPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	accessPolicies, err := a.AccessPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range accessPolicies {
		if policy.SiteID != siteID {
			continue
		}
		if err := a.AccessPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	ratePolicies, err := a.RateLimitPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range ratePolicies {
		if policy.SiteID != siteID {
			continue
		}
		if err := a.RateLimitPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	return nil
}

func (a *App) deleteUpstreamsBySiteID(ctx context.Context, siteID string) error {
	items, err := a.UpstreamStore.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.SiteID != siteID {
			continue
		}
		if err := a.UpstreamService.Delete(ctx, item.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}
	return nil
}

func (a *App) deleteTLSConfigIfExists(ctx context.Context, siteID string) error {
	if err := a.TLSConfigService.Delete(ctx, siteID); err != nil && !isNotFoundError(err) {
		return err
	}
	return nil
}

func (a *App) cleanupStartupSelfTestArtifacts() error {
	const prefix = "startup-self-test-"

	easyProfiles, err := a.EasySiteProfileStore.List()
	if err != nil {
		return err
	}
	for _, item := range easyProfiles {
		if !strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			continue
		}
		if err := a.EasySiteProfileStore.Delete(item.SiteID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	tlsItems, err := a.TLSConfigStore.List()
	if err != nil {
		return err
	}
	for _, item := range tlsItems {
		if !strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			continue
		}
		if err := a.TLSConfigStore.Delete(item.SiteID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	upstreamItems, err := a.UpstreamStore.List()
	if err != nil {
		return err
	}
	for _, item := range upstreamItems {
		if !strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			continue
		}
		if err := a.UpstreamStore.Delete(item.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	if err := a.deletePoliciesBySitePrefix(prefix); err != nil {
		return err
	}

	siteItems, err := a.SiteStore.List()
	if err != nil {
		return err
	}
	for _, item := range siteItems {
		id := strings.TrimSpace(item.ID)
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		if err := a.SiteStore.Delete(id); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	certificatesItems, err := a.CertificateStore.List()
	if err != nil {
		return err
	}
	for _, item := range certificatesItems {
		if !strings.HasPrefix(strings.TrimSpace(item.ID), prefix) {
			continue
		}
		if err := a.CertificateStore.Delete(item.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}
	return nil
}

func (a *App) deletePoliciesBySitePrefix(prefix string) error {
	wafPolicies, err := a.WAFPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range wafPolicies {
		if !strings.HasPrefix(strings.TrimSpace(policy.SiteID), prefix) {
			continue
		}
		if err := a.WAFPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	accessPolicies, err := a.AccessPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range accessPolicies {
		if !strings.HasPrefix(strings.TrimSpace(policy.SiteID), prefix) {
			continue
		}
		if err := a.AccessPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	ratePolicies, err := a.RateLimitPolicyStore.List()
	if err != nil {
		return err
	}
	for _, policy := range ratePolicies {
		if !strings.HasPrefix(strings.TrimSpace(policy.SiteID), prefix) {
			continue
		}
		if err := a.RateLimitPolicyStore.Delete(policy.ID); err != nil && !isNotFoundError(err) {
			return err
		}
	}
	return nil
}

func (a *App) assertNoStartupSelfTestArtifacts() error {
	const prefix = "startup-self-test-"

	siteItems, err := a.SiteStore.List()
	if err != nil {
		return err
	}
	for _, item := range siteItems {
		if strings.HasPrefix(strings.TrimSpace(item.ID), prefix) {
			return fmt.Errorf("site %s still exists", item.ID)
		}
	}

	upstreamItems, err := a.UpstreamStore.List()
	if err != nil {
		return err
	}
	for _, item := range upstreamItems {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("upstream %s for startup self-test site still exists", item.ID)
		}
	}

	tlsItems, err := a.TLSConfigStore.List()
	if err != nil {
		return err
	}
	for _, item := range tlsItems {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("tls config for startup self-test site %s still exists", item.SiteID)
		}
	}

	easyItems, err := a.EasySiteProfileStore.List()
	if err != nil {
		return err
	}
	for _, item := range easyItems {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("easy profile for startup self-test site %s still exists", item.SiteID)
		}
	}

	wafPolicies, err := a.WAFPolicyStore.List()
	if err != nil {
		return err
	}
	for _, item := range wafPolicies {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("waf policy %s for startup self-test site still exists", item.ID)
		}
	}

	accessPolicies, err := a.AccessPolicyStore.List()
	if err != nil {
		return err
	}
	for _, item := range accessPolicies {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("access policy %s for startup self-test site still exists", item.ID)
		}
	}

	ratePolicies, err := a.RateLimitPolicyStore.List()
	if err != nil {
		return err
	}
	for _, item := range ratePolicies {
		if strings.HasPrefix(strings.TrimSpace(item.SiteID), prefix) {
			return fmt.Errorf("rate limit policy %s for startup self-test site still exists", item.ID)
		}
	}

	certificateItems, err := a.CertificateStore.List()
	if err != nil {
		return err
	}
	for _, item := range certificateItems {
		if strings.HasPrefix(strings.TrimSpace(item.ID), prefix) {
			return fmt.Errorf("certificate %s from startup self-test still exists", item.ID)
		}
	}

	return nil
}

func startupSelfTestCreateProfile(siteID, host string) easysiteprofiles.EasySiteProfile {
	profile := easysiteprofiles.DefaultProfile(siteID)

	// 1) Front service.
	profile.FrontService.ServerName = host
	profile.FrontService.SecurityMode = easysiteprofiles.SecurityModeBlock
	profile.FrontService.AutoLetsEncrypt = true
	profile.FrontService.CertificateAuthorityServer = "letsencrypt"
	profile.FrontService.ACMEAccountEmail = "ops@example.test"

	// 2) Upstream.
	profile.UpstreamRouting.UseReverseProxy = true
	profile.UpstreamRouting.ReverseProxyHost = "http://upstream-a.internal:8080"
	profile.UpstreamRouting.ReverseProxyURL = "/app"
	profile.UpstreamRouting.ReverseProxySSLSNI = true
	profile.UpstreamRouting.ReverseProxySSLSNIName = "upstream-a.internal"
	profile.UpstreamRouting.ReverseProxyWebsocket = true
	profile.UpstreamRouting.ReverseProxyKeepalive = true

	// 3) Common HTTP.
	profile.HTTPBehavior.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	profile.HTTPBehavior.MaxClientSize = "32m"
	profile.HTTPBehavior.HTTP2 = true
	profile.HTTPBehavior.HTTP3 = true
	profile.HTTPBehavior.SSLProtocols = []string{"TLSv1.2", "TLSv1.3"}

	// 4) Headers and CORS.
	profile.HTTPHeaders.CookieFlags = "* SameSite=Strict"
	profile.HTTPHeaders.ContentSecurityPolicy = "default-src 'self'; frame-ancestors 'none'"
	profile.HTTPHeaders.PermissionsPolicy = []string{"camera=()", "microphone=()"}
	profile.HTTPHeaders.KeepUpstreamHeaders = []string{"X-Request-Id", "X-Trace-Id"}
	profile.HTTPHeaders.ReferrerPolicy = "strict-origin"
	profile.HTTPHeaders.UseCORS = true
	profile.HTTPHeaders.CORSAllowedOrigins = []string{"https://app.example.test"}

	// 5) Traffic control.
	profile.SecurityBehaviorAndLimits.UseBadBehavior = true
	profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{401, 403, 429}
	profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = 90
	profile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 10
	profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds = 30
	profile.SecurityBehaviorAndLimits.UseBlacklist = true
	profile.SecurityBehaviorAndLimits.UseDNSBL = true
	profile.SecurityBehaviorAndLimits.BlacklistIP = []string{"203.0.113.10"}
	profile.SecurityBehaviorAndLimits.BlacklistUserAgent = []string{"curl/*"}
	profile.SecurityBehaviorAndLimits.BlacklistURI = []string{"/private"}
	profile.SecurityBehaviorAndLimits.UseLimitConn = true
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = 60
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = 120
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = 120
	profile.SecurityBehaviorAndLimits.UseLimitReq = true
	profile.SecurityBehaviorAndLimits.LimitReqURL = "/app"
	profile.SecurityBehaviorAndLimits.LimitReqRate = "25r/s"
	profile.SecurityBehaviorAndLimits.CustomLimitRules = []easysiteprofiles.CustomLimitRule{
		{Path: "/login", Rate: "5r/s"},
	}

	// 6) Ban stages.
	profile.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	profile.SecurityBehaviorAndLimits.BanEscalationScope = "all_sites"
	profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 3600, 0}

	// 7) AntiBot and basic auth.
	profile.SecurityAntibot.AntibotChallenge = easysiteprofiles.AntibotChallengeTurnstile
	profile.SecurityAntibot.AntibotURI = "/challenge"
	profile.SecurityAntibot.AntibotTurnstileSitekey = "turnstile-sitekey-a"
	profile.SecurityAntibot.AntibotTurnstileSecret = "turnstile-secret-a"
	profile.SecurityAuthBasic.UseAuthBasic = true
	profile.SecurityAuthBasic.AuthBasicLocation = easysiteprofiles.AuthBasicLocationSitewide
	profile.SecurityAuthBasic.AuthBasicUser = "admin"
	profile.SecurityAuthBasic.AuthBasicPassword = "password-a"
	profile.SecurityAuthBasic.AuthBasicText = "Restricted area"

	// 8) Country policy.
	profile.SecurityCountryPolicy.BlacklistCountry = []string{"RU", "CN"}
	profile.SecurityCountryPolicy.WhitelistCountry = []string{"US"}

	// 9) ModSecurity.
	profile.SecurityModSecurity.UseModSecurity = true
	profile.SecurityModSecurity.UseModSecurityCRSPlugins = true
	profile.SecurityModSecurity.ModSecurityCRSVersion = "4.1"
	profile.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-a"}
	profile.SecurityModSecurity.UseCustomConfiguration = true
	profile.SecurityModSecurity.CustomConfiguration.Path = "modsec/custom-a.conf"
	profile.SecurityModSecurity.CustomConfiguration.Content = `SecAction "id:110001,phase:1,pass"`

	return profile
}

func assertStartupSnapshotContainsServiceArtifacts(snapshot revisionsnapshots.Snapshot, siteID, upstreamID, certificateID string) error {
	if !snapshotHasSite(snapshot, siteID) {
		return fmt.Errorf("site %s not found in snapshot", siteID)
	}
	if !snapshotHasUpstream(snapshot, upstreamID, siteID) {
		return fmt.Errorf("upstream %s not found in snapshot", upstreamID)
	}
	if !snapshotHasTLSConfig(snapshot, siteID, certificateID) {
		return fmt.Errorf("tls config for site %s not found in snapshot", siteID)
	}
	if !snapshotHasEasyProfile(snapshot, siteID) {
		return fmt.Errorf("easy profile for site %s not found in snapshot", siteID)
	}
	if !snapshotHasWAFPolicy(snapshot, siteID) {
		return fmt.Errorf("waf policy for site %s not found in snapshot", siteID)
	}
	if !snapshotHasAccessPolicy(snapshot, siteID) {
		return fmt.Errorf("access policy for site %s not found in snapshot", siteID)
	}
	if !snapshotHasRateLimitPolicy(snapshot, siteID) {
		return fmt.Errorf("rate limit policy for site %s not found in snapshot", siteID)
	}
	return nil
}

func assertStartupSnapshotCleaned(snapshot revisionsnapshots.Snapshot, siteID string) error {
	if snapshotHasSite(snapshot, siteID) {
		return fmt.Errorf("site %s still present in snapshot", siteID)
	}
	for _, item := range snapshot.Upstreams {
		if item.SiteID == siteID {
			return fmt.Errorf("upstream %s for site %s still present in snapshot", item.ID, siteID)
		}
	}
	for _, item := range snapshot.TLSConfigs {
		if item.SiteID == siteID {
			return fmt.Errorf("tls config for site %s still present in snapshot", siteID)
		}
	}
	for _, item := range snapshot.EasySiteProfiles {
		if item.SiteID == siteID {
			return fmt.Errorf("easy profile for site %s still present in snapshot", siteID)
		}
	}
	for _, item := range snapshot.WAFPolicies {
		if item.SiteID == siteID {
			return fmt.Errorf("waf policy %s for site %s still present in snapshot", item.ID, siteID)
		}
	}
	for _, item := range snapshot.AccessPolicies {
		if item.SiteID == siteID {
			return fmt.Errorf("access policy %s for site %s still present in snapshot", item.ID, siteID)
		}
	}
	for _, item := range snapshot.RateLimitPolicies {
		if item.SiteID == siteID {
			return fmt.Errorf("rate limit policy %s for site %s still present in snapshot", item.ID, siteID)
		}
	}
	return nil
}

func snapshotHasSite(snapshot revisionsnapshots.Snapshot, siteID string) bool {
	for _, item := range snapshot.Sites {
		if item.ID == siteID {
			return true
		}
	}
	return false
}

func snapshotHasUpstream(snapshot revisionsnapshots.Snapshot, upstreamID, siteID string) bool {
	for _, item := range snapshot.Upstreams {
		if item.ID == upstreamID && item.SiteID == siteID {
			return true
		}
	}
	return false
}

func snapshotHasTLSConfig(snapshot revisionsnapshots.Snapshot, siteID, certificateID string) bool {
	for _, item := range snapshot.TLSConfigs {
		if item.SiteID == siteID && item.CertificateID == certificateID {
			return true
		}
	}
	return false
}

func snapshotHasEasyProfile(snapshot revisionsnapshots.Snapshot, siteID string) bool {
	for _, item := range snapshot.EasySiteProfiles {
		if item.SiteID == siteID {
			return true
		}
	}
	return false
}

func snapshotHasWAFPolicy(snapshot revisionsnapshots.Snapshot, siteID string) bool {
	for _, item := range snapshot.WAFPolicies {
		if item.SiteID == siteID {
			return true
		}
	}
	return false
}

func snapshotHasAccessPolicy(snapshot revisionsnapshots.Snapshot, siteID string) bool {
	for _, item := range snapshot.AccessPolicies {
		if item.SiteID == siteID {
			return true
		}
	}
	return false
}

func snapshotHasRateLimitPolicy(snapshot revisionsnapshots.Snapshot, siteID string) bool {
	for _, item := range snapshot.RateLimitPolicies {
		if item.SiteID == siteID {
			return true
		}
	}
	return false
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func startupSelfTestUpdateProfile(siteID, host string) easysiteprofiles.EasySiteProfile {
	profile := startupSelfTestCreateProfile(siteID, host)

	// 1) Front service.
	profile.FrontService.SecurityMode = easysiteprofiles.SecurityModeMonitor
	profile.FrontService.ACMEAccountEmail = "security@example.test"

	// 2) Upstream.
	profile.UpstreamRouting.ReverseProxyHost = "https://upstream-b.internal:8443"
	profile.UpstreamRouting.ReverseProxyURL = "/api"
	profile.UpstreamRouting.ReverseProxyKeepalive = false
	profile.UpstreamRouting.ReverseProxyWebsocket = false

	// 3) Common HTTP.
	profile.HTTPBehavior.AllowedMethods = []string{"GET", "POST"}
	profile.HTTPBehavior.MaxClientSize = "16m"
	profile.HTTPBehavior.HTTP3 = false

	// 4) Headers and CORS.
	profile.HTTPHeaders.ContentSecurityPolicy = "default-src 'self' https:"
	profile.HTTPHeaders.CORSAllowedOrigins = []string{"https://admin.example.test"}

	// 5) Traffic control.
	profile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 7
	profile.SecurityBehaviorAndLimits.BlacklistIP = []string{"198.51.100.25"}
	profile.SecurityBehaviorAndLimits.LimitReqRate = "15r/s"
	profile.SecurityBehaviorAndLimits.CustomLimitRules = []easysiteprofiles.CustomLimitRule{
		{Path: "/api", Rate: "8r/s"},
	}

	// 6) Ban stages.
	profile.SecurityBehaviorAndLimits.BanEscalationScope = "current_site"
	profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{120, 0}

	// 7) AntiBot and basic auth.
	profile.SecurityAntibot.AntibotTurnstileSitekey = "turnstile-sitekey-b"
	profile.SecurityAntibot.AntibotTurnstileSecret = "turnstile-secret-b"
	profile.SecurityAuthBasic.AuthBasicPassword = "password-b"

	// 8) Country policy.
	profile.SecurityCountryPolicy.BlacklistCountry = []string{"IR"}
	profile.SecurityCountryPolicy.WhitelistCountry = []string{"DE", "US"}

	// 9) ModSecurity.
	profile.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-b", "plugin-c"}
	profile.SecurityModSecurity.CustomConfiguration.Path = "modsec/custom-b.conf"
	profile.SecurityModSecurity.CustomConfiguration.Content = `SecAction "id:110002,phase:2,pass"`

	return profile
}

func assertStartupProfileCreate(profile easysiteprofiles.EasySiteProfile) error {
	if profile.FrontService.SecurityMode != easysiteprofiles.SecurityModeBlock {
		return fmt.Errorf("front_service.security_mode expected %s, got %s", easysiteprofiles.SecurityModeBlock, profile.FrontService.SecurityMode)
	}
	if profile.UpstreamRouting.ReverseProxyURL != "/app" {
		return fmt.Errorf("upstream_routing.reverse_proxy_url expected /app, got %s", profile.UpstreamRouting.ReverseProxyURL)
	}
	if profile.HTTPBehavior.MaxClientSize != "32m" {
		return fmt.Errorf("http_behavior.max_client_size expected 32m, got %s", profile.HTTPBehavior.MaxClientSize)
	}
	if !profile.HTTPHeaders.UseCORS {
		return fmt.Errorf("http_headers.use_cors expected true")
	}
	if profile.SecurityBehaviorAndLimits.LimitReqRate != "25r/s" {
		return fmt.Errorf("security_behavior_and_limits.limit_req_rate expected 25r/s, got %s", profile.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) != 3 {
		return fmt.Errorf("security_behavior_and_limits.ban_escalation_stages_seconds expected 3 items")
	}
	if profile.SecurityAntibot.AntibotChallenge != easysiteprofiles.AntibotChallengeTurnstile {
		return fmt.Errorf("security_antibot.antibot_challenge expected %s, got %s", easysiteprofiles.AntibotChallengeTurnstile, profile.SecurityAntibot.AntibotChallenge)
	}
	if profile.SecurityAuthBasic.AuthBasicPassword != "password-a" {
		return fmt.Errorf("security_auth_basic.auth_basic_password expected initial value")
	}
	if len(profile.SecurityCountryPolicy.WhitelistCountry) != 1 || profile.SecurityCountryPolicy.WhitelistCountry[0] != "US" {
		return fmt.Errorf("security_country_policy.whitelist_country expected [US]")
	}
	if profile.SecurityModSecurity.CustomConfiguration.Path != "modsec/custom-a.conf" {
		return fmt.Errorf("security_modsecurity.custom_configuration.path expected modsec/custom-a.conf, got %s", profile.SecurityModSecurity.CustomConfiguration.Path)
	}
	return nil
}

func assertStartupProfileUpdate(profile easysiteprofiles.EasySiteProfile) error {
	if profile.FrontService.SecurityMode != easysiteprofiles.SecurityModeMonitor {
		return fmt.Errorf("front_service.security_mode expected %s, got %s", easysiteprofiles.SecurityModeMonitor, profile.FrontService.SecurityMode)
	}
	if profile.UpstreamRouting.ReverseProxyURL != "/api" {
		return fmt.Errorf("upstream_routing.reverse_proxy_url expected /api, got %s", profile.UpstreamRouting.ReverseProxyURL)
	}
	if profile.HTTPBehavior.MaxClientSize != "16m" {
		return fmt.Errorf("http_behavior.max_client_size expected 16m, got %s", profile.HTTPBehavior.MaxClientSize)
	}
	if profile.HTTPHeaders.ContentSecurityPolicy != "default-src 'self' https:" {
		return fmt.Errorf("http_headers.content_security_policy was not updated")
	}
	if profile.SecurityBehaviorAndLimits.LimitReqRate != "15r/s" {
		return fmt.Errorf("security_behavior_and_limits.limit_req_rate expected 15r/s, got %s", profile.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if profile.SecurityBehaviorAndLimits.BanEscalationScope != "current_site" {
		return fmt.Errorf("security_behavior_and_limits.ban_escalation_scope expected current_site, got %s", profile.SecurityBehaviorAndLimits.BanEscalationScope)
	}
	if profile.SecurityAntibot.AntibotTurnstileSecret != "turnstile-secret-b" {
		return fmt.Errorf("security_antibot.antibot_turnstile_secret expected updated value")
	}
	if profile.SecurityAuthBasic.AuthBasicPassword != "password-b" {
		return fmt.Errorf("security_auth_basic.auth_basic_password expected updated value")
	}
	if len(profile.SecurityCountryPolicy.BlacklistCountry) != 1 || profile.SecurityCountryPolicy.BlacklistCountry[0] != "IR" {
		return fmt.Errorf("security_country_policy.blacklist_country expected [IR]")
	}
	if profile.SecurityModSecurity.CustomConfiguration.Path != "modsec/custom-b.conf" {
		return fmt.Errorf("security_modsecurity.custom_configuration.path expected modsec/custom-b.conf, got %s", profile.SecurityModSecurity.CustomConfiguration.Path)
	}
	return nil
}
