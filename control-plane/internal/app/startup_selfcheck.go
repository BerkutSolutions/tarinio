package app

import (
	"context"
	"fmt"
	"time"

	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/sites"
)

func (a *App) RunStartupSelfTest(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("startup self-test: app is nil")
	}
	if !a.Config.StartupSelfTest {
		return nil
	}
	if a.SiteService == nil || a.EasySiteProfileService == nil || a.EasySiteProfileStore == nil {
		return fmt.Errorf("startup self-test: dependencies are not wired")
	}

	testCtx := services.WithAutoApplyDisabled(ctx)
	siteID := fmt.Sprintf("startup-self-test-%d", time.Now().UTC().UnixNano())
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
			_ = a.SiteService.Delete(testCtx, cleanupSite)
		}
	}()

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
