package easysiteprofiles

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var rateRegexp = regexp.MustCompile(`^\d+r/s$`)
var crsVersionRegexp = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
var safeHeaderValueRegexp = regexp.MustCompile(`^[^\r\n"]+$`)
var safeHostHeaderRegexp = regexp.MustCompile(`^(?i:[a-z0-9][a-z0-9.-]*[a-z0-9]|[a-z0-9])(:[0-9]{1,5})?$`)
var safeCORSOriginRegexp = regexp.MustCompile(`^(?:\*|null|https?://[a-z0-9][a-z0-9.-]*(:[0-9]{1,5})?)$`)

var allowedCountrySelectors = map[string]struct{}{
	"AF": {}, "AN": {}, "AS": {}, "EU": {}, "NA": {}, "OC": {}, "SA": {},
	"APAC": {}, "EMEA": {}, "LATAM": {}, "DACH": {}, "CIS": {}, "GCC": {}, "NORAM": {},
}

func normalizeProfile(profile EasySiteProfile) EasySiteProfile {
	profile.SiteID = normalizeID(profile.SiteID)

	profile.FrontService.ServerName = strings.ToLower(strings.TrimSpace(profile.FrontService.ServerName))
	profile.FrontService.SecurityMode = strings.ToLower(strings.TrimSpace(profile.FrontService.SecurityMode))
	profile.FrontService.Profile = strings.ToLower(strings.TrimSpace(profile.FrontService.Profile))
	if profile.FrontService.Profile == "" {
		profile.FrontService.Profile = ServiceProfileBalanced
	}
	profile.FrontService.CertificateAuthorityServer = strings.ToLower(strings.TrimSpace(profile.FrontService.CertificateAuthorityServer))
	profile.FrontService.ACMEAccountEmail = strings.ToLower(strings.TrimSpace(profile.FrontService.ACMEAccountEmail))

	profile.UpstreamRouting.ReverseProxyHost = strings.TrimSpace(profile.UpstreamRouting.ReverseProxyHost)
	profile.UpstreamRouting.ReverseProxyURL = strings.TrimSpace(profile.UpstreamRouting.ReverseProxyURL)
	profile.UpstreamRouting.ReverseProxyCustomHost = strings.TrimSpace(profile.UpstreamRouting.ReverseProxyCustomHost)
	profile.UpstreamRouting.ReverseProxySSLSNIName = strings.TrimSpace(profile.UpstreamRouting.ReverseProxySSLSNIName)

	profile.HTTPBehavior.AllowedMethods = normalizeUpperList(profile.HTTPBehavior.AllowedMethods)
	profile.HTTPBehavior.AllowedMethods = ensureControlPlaneAccessMethods(profile.SiteID, profile.HTTPBehavior.AllowedMethods)
	profile.HTTPBehavior.MaxClientSize = strings.ToLower(strings.TrimSpace(profile.HTTPBehavior.MaxClientSize))
	profile.HTTPBehavior.SSLProtocols = normalizeTrimmedList(profile.HTTPBehavior.SSLProtocols)

	profile.HTTPHeaders.CookieFlags = strings.TrimSpace(profile.HTTPHeaders.CookieFlags)
	profile.HTTPHeaders.ContentSecurityPolicy = strings.TrimSpace(profile.HTTPHeaders.ContentSecurityPolicy)
	profile.HTTPHeaders.PermissionsPolicy = normalizeTrimmedList(profile.HTTPHeaders.PermissionsPolicy)
	profile.HTTPHeaders.KeepUpstreamHeaders = normalizeTrimmedList(profile.HTTPHeaders.KeepUpstreamHeaders)
	profile.HTTPHeaders.ReferrerPolicy = strings.TrimSpace(profile.HTTPHeaders.ReferrerPolicy)
	profile.HTTPHeaders.CORSAllowedOrigins = normalizeTrimmedList(profile.HTTPHeaders.CORSAllowedOrigins)

	profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = normalizeStatusCodes(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes)
	profile.SecurityBehaviorAndLimits.BanEscalationScope = strings.ToLower(strings.TrimSpace(profile.SecurityBehaviorAndLimits.BanEscalationScope))
	if profile.SecurityBehaviorAndLimits.BanEscalationScope == "" {
		profile.SecurityBehaviorAndLimits.BanEscalationScope = "all_sites"
	}
	profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = normalizeBanEscalationStages(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds)
	if len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) == 0 {
		profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 86400, 0}
	}
	profile.SecurityBehaviorAndLimits.ExceptionsIP = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.ExceptionsIP)
	profile.SecurityBehaviorAndLimits.BlacklistIP = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistIP)
	profile.SecurityBehaviorAndLimits.BlacklistRDNS = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistRDNS)
	profile.SecurityBehaviorAndLimits.BlacklistASN = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistASN)
	profile.SecurityBehaviorAndLimits.BlacklistUserAgent = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistUserAgent)
	profile.SecurityBehaviorAndLimits.BlacklistURI = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistURI)
	profile.SecurityBehaviorAndLimits.BlacklistIPURLs = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistIPURLs)
	profile.SecurityBehaviorAndLimits.BlacklistRDNSURLs = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistRDNSURLs)
	profile.SecurityBehaviorAndLimits.BlacklistASNURLs = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistASNURLs)
	profile.SecurityBehaviorAndLimits.BlacklistUserAgentURLs = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistUserAgentURLs)
	profile.SecurityBehaviorAndLimits.BlacklistURIURLs = normalizeTrimmedList(profile.SecurityBehaviorAndLimits.BlacklistURIURLs)
	profile.SecurityBehaviorAndLimits.LimitReqURL = strings.TrimSpace(profile.SecurityBehaviorAndLimits.LimitReqURL)
	profile.SecurityBehaviorAndLimits.LimitReqRate = normalizeLimitReqRate(profile.SecurityBehaviorAndLimits.LimitReqRate)
	profile.SecurityBehaviorAndLimits.CustomLimitRules = normalizeCustomLimitRules(profile.SecurityBehaviorAndLimits.CustomLimitRules)

	profile.SecurityAntibot.AntibotChallenge = strings.ToLower(strings.TrimSpace(profile.SecurityAntibot.AntibotChallenge))
	profile.SecurityAntibot.AntibotURI = strings.TrimSpace(profile.SecurityAntibot.AntibotURI)
	profile.SecurityAntibot.AntibotRecaptchaSitekey = strings.TrimSpace(profile.SecurityAntibot.AntibotRecaptchaSitekey)
	profile.SecurityAntibot.AntibotRecaptchaSecret = strings.TrimSpace(profile.SecurityAntibot.AntibotRecaptchaSecret)
	profile.SecurityAntibot.AntibotHcaptchaSitekey = strings.TrimSpace(profile.SecurityAntibot.AntibotHcaptchaSitekey)
	profile.SecurityAntibot.AntibotHcaptchaSecret = strings.TrimSpace(profile.SecurityAntibot.AntibotHcaptchaSecret)
	profile.SecurityAntibot.AntibotTurnstileSitekey = strings.TrimSpace(profile.SecurityAntibot.AntibotTurnstileSitekey)
	profile.SecurityAntibot.AntibotTurnstileSecret = strings.TrimSpace(profile.SecurityAntibot.AntibotTurnstileSecret)
	profile.SecurityAntibot.ChallengeEscalationMode = strings.ToLower(strings.TrimSpace(profile.SecurityAntibot.ChallengeEscalationMode))
	if profile.SecurityAntibot.ChallengeEscalationMode == "" {
		profile.SecurityAntibot.ChallengeEscalationMode = AntibotChallengeJavascript
	}
	profile.SecurityAntibot.ChallengeRules = normalizeAntibotChallengeRules(profile.SecurityAntibot.ChallengeRules)

	profile.SecurityAuthBasic.AuthBasicLocation = strings.ToLower(strings.TrimSpace(profile.SecurityAuthBasic.AuthBasicLocation))
	profile.SecurityAuthBasic.AuthBasicUser = strings.TrimSpace(profile.SecurityAuthBasic.AuthBasicUser)
	profile.SecurityAuthBasic.AuthBasicPassword = strings.TrimSpace(profile.SecurityAuthBasic.AuthBasicPassword)
	profile.SecurityAuthBasic.AuthBasicText = strings.TrimSpace(profile.SecurityAuthBasic.AuthBasicText)
	profile.SecurityAuthBasic.Users = normalizeAuthUsers(profile.SecurityAuthBasic.Users)
	if len(profile.SecurityAuthBasic.Users) == 0 && profile.SecurityAuthBasic.AuthBasicUser != "" {
		profile.SecurityAuthBasic.Users = []SecurityAuthUser{
			{
				Username: profile.SecurityAuthBasic.AuthBasicUser,
				Password: profile.SecurityAuthBasic.AuthBasicPassword,
				Enabled:  true,
			},
		}
	}
	if profile.SecurityAuthBasic.UseAuthBasic &&
		profile.SecurityAuthBasic.AuthBasicUser != "" &&
		profile.SecurityAuthBasic.AuthBasicPassword != "" &&
		!hasEnabledUserWithPassword(profile.SecurityAuthBasic.Users) {
		profile.SecurityAuthBasic.Users = []SecurityAuthUser{
			{
				Username: profile.SecurityAuthBasic.AuthBasicUser,
				Password: profile.SecurityAuthBasic.AuthBasicPassword,
				Enabled:  true,
			},
		}
	}
	if profile.SecurityAuthBasic.AuthBasicUser == "" && len(profile.SecurityAuthBasic.Users) > 0 {
		profile.SecurityAuthBasic.AuthBasicUser = profile.SecurityAuthBasic.Users[0].Username
	}
	if profile.SecurityAuthBasic.AuthBasicPassword == "" && len(profile.SecurityAuthBasic.Users) > 0 {
		profile.SecurityAuthBasic.AuthBasicPassword = profile.SecurityAuthBasic.Users[0].Password
	}
	if profile.SecurityAuthBasic.SessionInactivityMinutes == 0 {
		profile.SecurityAuthBasic.SessionInactivityMinutes = 60
	}

	profile.SecurityCountryPolicy.BlacklistCountry = normalizeCountryList(profile.SecurityCountryPolicy.BlacklistCountry)
	profile.SecurityCountryPolicy.WhitelistCountry = normalizeCountryList(profile.SecurityCountryPolicy.WhitelistCountry)

	profile.SecurityAPIPositive.OpenAPISchemaRef = strings.TrimSpace(profile.SecurityAPIPositive.OpenAPISchemaRef)
	profile.SecurityAPIPositive.EnforcementMode = strings.ToLower(strings.TrimSpace(profile.SecurityAPIPositive.EnforcementMode))
	if profile.SecurityAPIPositive.EnforcementMode == "" {
		profile.SecurityAPIPositive.EnforcementMode = APIPositiveEnforcementMonitor
	}
	profile.SecurityAPIPositive.DefaultAction = strings.ToLower(strings.TrimSpace(profile.SecurityAPIPositive.DefaultAction))
	if profile.SecurityAPIPositive.DefaultAction == "" {
		profile.SecurityAPIPositive.DefaultAction = APIPositiveDefaultActionAllow
	}
	profile.SecurityAPIPositive.EndpointPolicies = normalizeAPIPositiveEndpointPolicies(profile.SecurityAPIPositive.EndpointPolicies)

	profile.SecurityModSecurity.ModSecurityCRSVersion = strings.TrimSpace(profile.SecurityModSecurity.ModSecurityCRSVersion)
	profile.SecurityModSecurity.ModSecurityCRSPlugins = normalizeTrimmedList(profile.SecurityModSecurity.ModSecurityCRSPlugins)
	profile.SecurityModSecurity.CustomConfiguration.Path = strings.TrimSpace(profile.SecurityModSecurity.CustomConfiguration.Path)
	profile.SecurityModSecurity.CustomConfiguration.Content = strings.TrimSpace(profile.SecurityModSecurity.CustomConfiguration.Content)

	return profile
}

func ensureControlPlaneAccessMethods(siteID string, methods []string) []string {
	if normalizeID(siteID) != "control-plane-access" {
		return methods
	}
	required := []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
	merged := append([]string(nil), methods...)
	for _, method := range required {
		if !slices.Contains(merged, method) {
			merged = append(merged, method)
		}
	}
	sort.Strings(merged)
	return slices.Compact(merged)
}

func validateProfile(profile EasySiteProfile) error {
	if profile.SiteID == "" {
		return errors.New("easy site profile site_id is required")
	}
	if profile.FrontService.ServerName == "" {
		return errors.New("easy site profile front_service.server_name is required")
	}
	if !slices.Contains([]string{SecurityModeBlock, SecurityModeMonitor, SecurityModeTransparent}, profile.FrontService.SecurityMode) {
		return errors.New("easy site profile front_service.security_mode must be block, monitor, or transparent")
	}
	allowedProfiles := []string{
		ServiceProfileStrict,
		ServiceProfileBalanced,
		ServiceProfileCompat,
		ServiceProfileAPI,
		ServiceProfilePublicEdge,
	}
	if !slices.Contains(allowedProfiles, profile.FrontService.Profile) {
		return errors.New("easy site profile front_service.profile has unsupported value")
	}
	if profile.FrontService.CertificateAuthorityServer == "" {
		return errors.New("easy site profile front_service.certificate_authority_server is required")
	}

	if profile.UpstreamRouting.UseReverseProxy && profile.UpstreamRouting.ReverseProxyHost == "" {
		return errors.New("easy site profile upstream_routing.reverse_proxy_host is required when reverse proxy is enabled")
	}
	if profile.UpstreamRouting.ReverseProxyURL == "" || !strings.HasPrefix(profile.UpstreamRouting.ReverseProxyURL, "/") {
		return errors.New("easy site profile upstream_routing.reverse_proxy_url must start with /")
	}

	allowedMethods := []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE", "PATCH"}
	if len(profile.HTTPBehavior.AllowedMethods) == 0 {
		return errors.New("easy site profile http_behavior.allowed_methods must not be empty")
	}
	for _, method := range profile.HTTPBehavior.AllowedMethods {
		if !slices.Contains(allowedMethods, method) {
			return fmt.Errorf("easy site profile http_behavior.allowed_methods contains unsupported method %s", method)
		}
	}
	if profile.SiteID == "control-plane-access" {
		requiredMethods := []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
		for _, method := range requiredMethods {
			if !slices.Contains(profile.HTTPBehavior.AllowedMethods, method) {
				return fmt.Errorf("easy site profile http_behavior.allowed_methods must include %s for control-plane-access", method)
			}
		}
	}
	if profile.HTTPBehavior.MaxClientSize == "" {
		return errors.New("easy site profile http_behavior.max_client_size is required")
	}

	if len(profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes) == 0 {
		return errors.New("easy site profile security_behavior_and_limits.bad_behavior_status_codes must not be empty")
	}
	for _, code := range profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes {
		if !isAllowedBadBehaviorStatusCode(code) {
			return fmt.Errorf("easy site profile security_behavior_and_limits.bad_behavior_status_codes contains unsupported code %d", code)
		}
	}
	if profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds < 0 {
		return errors.New("easy site profile security_behavior_and_limits.bad_behavior_ban_time_seconds must be zero or positive")
	}
	if profile.SecurityBehaviorAndLimits.BadBehaviorThreshold <= 0 {
		return errors.New("easy site profile security_behavior_and_limits.bad_behavior_threshold must be positive")
	}
	if profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds <= 0 {
		return errors.New("easy site profile security_behavior_and_limits.bad_behavior_count_time_seconds must be positive")
	}
	if profile.SecurityBehaviorAndLimits.BanEscalationScope != "current_site" && profile.SecurityBehaviorAndLimits.BanEscalationScope != "all_sites" {
		return errors.New("easy site profile security_behavior_and_limits.ban_escalation_scope must be current_site or all_sites")
	}
	if profile.SecurityBehaviorAndLimits.BanEscalationEnabled {
		if len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) == 0 {
			return errors.New("easy site profile security_behavior_and_limits.ban_escalation_stages_seconds must not be empty when escalation is enabled")
		}
		if len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) > 12 {
			return errors.New("easy site profile security_behavior_and_limits.ban_escalation_stages_seconds must not exceed 12 stages")
		}
		seenPermanent := false
		for idx, seconds := range profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds {
			if seconds < 0 {
				return errors.New("easy site profile security_behavior_and_limits.ban_escalation_stages_seconds must contain zero or positive values")
			}
			if seconds == 0 {
				seenPermanent = true
				if idx != len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds)-1 {
					return errors.New("easy site profile security_behavior_and_limits.ban_escalation_stages_seconds permanent stage must be last")
				}
				break
			}
		}
		if !seenPermanent && len(profile.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) == 1 {
			// A single finite stage is allowed.
		}
	}
	for _, value := range profile.SecurityBehaviorAndLimits.ExceptionsIP {
		if !isValidIPOrCIDR(value) {
			return fmt.Errorf("easy site profile security_behavior_and_limits.exceptions_ip contains invalid value %s", value)
		}
	}
	if profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 <= 0 ||
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 <= 0 ||
		profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 <= 0 {
		return errors.New("easy site profile security_behavior_and_limits limit_conn values must be positive")
	}
	if profile.SecurityBehaviorAndLimits.LimitReqURL == "" || !strings.HasPrefix(profile.SecurityBehaviorAndLimits.LimitReqURL, "/") {
		return errors.New("easy site profile security_behavior_and_limits.limit_req_url must start with /")
	}
	if !rateRegexp.MatchString(profile.SecurityBehaviorAndLimits.LimitReqRate) {
		return errors.New("easy site profile security_behavior_and_limits.limit_req_rate must match Nr/s")
	}
	if len(profile.SecurityBehaviorAndLimits.CustomLimitRules) > 32 {
		return errors.New("easy site profile security_behavior_and_limits.custom_limit_rules must not exceed 32 entries")
	}
	for _, rule := range profile.SecurityBehaviorAndLimits.CustomLimitRules {
		if rule.Path == "" || !strings.HasPrefix(rule.Path, "/") {
			return errors.New("easy site profile security_behavior_and_limits.custom_limit_rules.path must start with /")
		}
		if !rateRegexp.MatchString(rule.Rate) {
			return errors.New("easy site profile security_behavior_and_limits.custom_limit_rules.rate must match Nr/s")
		}
	}

	antibotModes := []string{
		AntibotChallengeNo, AntibotChallengeCookie, AntibotChallengeJavascript, AntibotChallengeCaptcha,
		AntibotChallengeRecaptcha, AntibotChallengeHcaptcha, AntibotChallengeTurnstile, AntibotChallengeMcaptcha,
	}
	if !slices.Contains(antibotModes, profile.SecurityAntibot.AntibotChallenge) {
		return errors.New("easy site profile security_antibot.antibot_challenge has unsupported mode")
	}
	if profile.SecurityAntibot.AntibotChallenge != AntibotChallengeNo {
		if profile.SecurityAntibot.AntibotURI == "" || !strings.HasPrefix(profile.SecurityAntibot.AntibotURI, "/") {
			return errors.New("easy site profile security_antibot.antibot_uri must start with / when antibot is enabled")
		}
	} else if len(profile.SecurityAntibot.ChallengeRules) > 0 {
		return errors.New("easy site profile security_antibot.challenge_rules requires antibot enabled")
	}
	if profile.SecurityAntibot.AntibotRecaptchaScore < 0 || profile.SecurityAntibot.AntibotRecaptchaScore > 1 {
		return errors.New("easy site profile security_antibot.antibot_recaptcha_score must be between 0 and 1")
	}
	if profile.SecurityAntibot.AntibotChallenge == AntibotChallengeRecaptcha {
		if profile.SecurityAntibot.AntibotRecaptchaSitekey == "" || profile.SecurityAntibot.AntibotRecaptchaSecret == "" {
			return errors.New("easy site profile security_antibot recaptcha mode requires sitekey and secret")
		}
	}
	if profile.SecurityAntibot.AntibotChallenge == AntibotChallengeHcaptcha {
		if profile.SecurityAntibot.AntibotHcaptchaSitekey == "" || profile.SecurityAntibot.AntibotHcaptchaSecret == "" {
			return errors.New("easy site profile security_antibot hcaptcha mode requires sitekey and secret")
		}
	}
	if profile.SecurityAntibot.AntibotChallenge == AntibotChallengeTurnstile {
		if profile.SecurityAntibot.AntibotTurnstileSitekey == "" || profile.SecurityAntibot.AntibotTurnstileSecret == "" {
			return errors.New("easy site profile security_antibot turnstile mode requires sitekey and secret")
		}
	}
	if profile.SecurityAntibot.ChallengeEscalationEnabled {
		if profile.SecurityAntibot.AntibotChallenge == AntibotChallengeNo {
			return errors.New("easy site profile security_antibot.challenge_escalation_enabled requires antibot enabled")
		}
		if profile.SecurityAntibot.ChallengeEscalationMode == AntibotChallengeNo || !slices.Contains(antibotModes, profile.SecurityAntibot.ChallengeEscalationMode) {
			return errors.New("easy site profile security_antibot.challenge_escalation_mode has unsupported mode")
		}
	}
	if len(profile.SecurityAntibot.ChallengeRules) > 32 {
		return errors.New("easy site profile security_antibot.challenge_rules must not exceed 32 entries")
	}
	for _, rule := range profile.SecurityAntibot.ChallengeRules {
		if rule.Path == "" || !strings.HasPrefix(rule.Path, "/") {
			return errors.New("easy site profile security_antibot.challenge_rules.path must start with /")
		}
		if rule.Challenge == AntibotChallengeNo || !slices.Contains(antibotModes, rule.Challenge) {
			return errors.New("easy site profile security_antibot.challenge_rules.challenge has unsupported mode")
		}
	}

	if profile.SecurityAuthBasic.AuthBasicLocation != AuthBasicLocationSitewide {
		return errors.New("easy site profile security_auth_basic.auth_basic_location must be sitewide")
	}
	if profile.SecurityAuthBasic.UseAuthBasic && profile.SecurityAuthBasic.AuthBasicUser == "" {
		return errors.New("easy site profile security_auth_basic.auth_basic_user is required when auth basic is enabled")
	}
	if profile.SecurityAuthBasic.UseAuthBasic && profile.SecurityAuthBasic.AuthBasicPassword == "" {
		return errors.New("easy site profile security_auth_basic.auth_basic_password is required when auth basic is enabled")
	}
	if profile.SecurityAuthBasic.AuthBasicText == "" {
		return errors.New("easy site profile security_auth_basic.auth_basic_text is required")
	}
	if profile.SecurityAuthBasic.SessionInactivityMinutes < -1 || profile.SecurityAuthBasic.SessionInactivityMinutes > 24*60 {
		return errors.New("easy site profile security_auth_basic.session_inactivity_minutes must be between -1 and 1440")
	}
	if len(profile.SecurityAuthBasic.Users) > 128 {
		return errors.New("easy site profile security_auth_basic.users must not exceed 128 entries")
	}
	if profile.SecurityAuthBasic.UseAuthBasic {
		enabledUsers := 0
		for _, user := range profile.SecurityAuthBasic.Users {
			if user.Enabled {
				enabledUsers++
				if strings.TrimSpace(user.Password) == "" {
					return errors.New("easy site profile security_auth_basic.users.password is required for enabled users")
				}
			}
		}
		if enabledUsers == 0 {
			return errors.New("easy site profile security_auth_basic.users must include at least one enabled user")
		}
	}
	if !isSafeHeaderValue(profile.SecurityAuthBasic.AuthBasicText) {
		return errors.New("easy site profile security_auth_basic.auth_basic_text contains unsupported characters")
	}

	if profile.HTTPHeaders.ReferrerPolicy != "" {
		if !isAllowedReferrerPolicy(profile.HTTPHeaders.ReferrerPolicy) {
			return errors.New("easy site profile http_headers.referrer_policy contains unsupported value")
		}
		if !isSafeHeaderValue(profile.HTTPHeaders.ReferrerPolicy) {
			return errors.New("easy site profile http_headers.referrer_policy contains unsupported characters")
		}
	}
	if profile.HTTPHeaders.ContentSecurityPolicy != "" && !isSafeHeaderValue(profile.HTTPHeaders.ContentSecurityPolicy) {
		return errors.New("easy site profile http_headers.content_security_policy contains unsupported characters")
	}
	for _, item := range profile.HTTPHeaders.CORSAllowedOrigins {
		if !safeCORSOriginRegexp.MatchString(strings.ToLower(strings.TrimSpace(item))) {
			return fmt.Errorf("easy site profile http_headers.cors_allowed_origins contains invalid origin %s", item)
		}
	}
	for _, item := range profile.HTTPHeaders.PermissionsPolicy {
		if !isSafeHeaderValue(item) {
			return errors.New("easy site profile http_headers.permissions_policy contains unsupported characters")
		}
	}
	if profile.UpstreamRouting.ReverseProxyCustomHost != "" && !safeHostHeaderRegexp.MatchString(strings.TrimSpace(profile.UpstreamRouting.ReverseProxyCustomHost)) {
		return errors.New("easy site profile upstream_routing.reverse_proxy_custom_host must be host or host:port")
	}
	if profile.UpstreamRouting.ReverseProxySSLSNIName != "" && !safeHostHeaderRegexp.MatchString(strings.TrimSpace(profile.UpstreamRouting.ReverseProxySSLSNIName)) {
		return errors.New("easy site profile upstream_routing.reverse_proxy_ssl_sni_name must be host or host:port")
	}

	for _, value := range profile.SecurityCountryPolicy.BlacklistCountry {
		if !isValidCountrySelector(value) {
			return fmt.Errorf("easy site profile security_country_policy.blacklist_country contains unsupported value %s", value)
		}
	}
	for _, value := range profile.SecurityCountryPolicy.WhitelistCountry {
		if !isValidCountrySelector(value) {
			return fmt.Errorf("easy site profile security_country_policy.whitelist_country contains unsupported value %s", value)
		}
	}
	blacklist := make(map[string]struct{}, len(profile.SecurityCountryPolicy.BlacklistCountry))
	for _, value := range profile.SecurityCountryPolicy.BlacklistCountry {
		blacklist[value] = struct{}{}
	}
	for _, value := range profile.SecurityCountryPolicy.WhitelistCountry {
		if _, ok := blacklist[value]; ok {
			return fmt.Errorf("easy site profile security_country_policy contains conflicting value %s in blacklist and whitelist", value)
		}
	}

	if profile.SecurityAPIPositive.EnforcementMode != APIPositiveEnforcementMonitor &&
		profile.SecurityAPIPositive.EnforcementMode != APIPositiveEnforcementBlock {
		return errors.New("easy site profile security_api_positive.enforcement_mode must be monitor or block")
	}
	if profile.SecurityAPIPositive.DefaultAction != APIPositiveDefaultActionAllow &&
		profile.SecurityAPIPositive.DefaultAction != APIPositiveDefaultActionDeny {
		return errors.New("easy site profile security_api_positive.default_action must be allow or deny")
	}
	if profile.SecurityAPIPositive.UseAPIPositiveSecurity {
		if profile.SecurityAPIPositive.OpenAPISchemaRef == "" && len(profile.SecurityAPIPositive.EndpointPolicies) == 0 {
			return errors.New("easy site profile security_api_positive requires openapi_schema_ref or endpoint_policies when enabled")
		}
		for _, policy := range profile.SecurityAPIPositive.EndpointPolicies {
			if policy.Path == "" || !strings.HasPrefix(policy.Path, "/") {
				return errors.New("easy site profile security_api_positive.endpoint_policies.path must start with /")
			}
			if policy.Mode != "" &&
				policy.Mode != APIPositiveEnforcementMonitor &&
				policy.Mode != APIPositiveEnforcementBlock {
				return errors.New("easy site profile security_api_positive.endpoint_policies.mode must be monitor or block")
			}
		}
	}

	if profile.SecurityModSecurity.ModSecurityCRSVersion == "" {
		return errors.New("easy site profile security_modsecurity.modsecurity_crs_version is required")
	}
	if profile.SecurityModSecurity.UseCustomConfiguration {
		if profile.SecurityModSecurity.CustomConfiguration.Path == "" {
			return errors.New("easy site profile security_modsecurity.custom_configuration.path is required when custom configuration is enabled")
		}
		if strings.Contains(profile.SecurityModSecurity.CustomConfiguration.Path, "..") ||
			strings.Contains(profile.SecurityModSecurity.CustomConfiguration.Path, `\`) ||
			!strings.HasPrefix(profile.SecurityModSecurity.CustomConfiguration.Path, "modsec/") {
			return errors.New("easy site profile security_modsecurity.custom_configuration.path must stay under modsec/")
		}
	}
	if !crsVersionRegexp.MatchString(profile.SecurityModSecurity.ModSecurityCRSVersion) {
		return errors.New("easy site profile security_modsecurity.modsecurity_crs_version must be numeric")
	}

	return nil
}

func isValidIPOrCIDR(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if ip := net.ParseIP(trimmed); ip != nil {
		return true
	}
	_, _, err := net.ParseCIDR(trimmed)
	return err == nil
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTrimmedList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return slices.Compact(items)
}

func normalizeUpperList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return slices.Compact(items)
}

func normalizeCountryList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return slices.Compact(items)
}

func normalizeAPIPositiveEndpointPolicies(values []APIPositiveEndpointPolicy) []APIPositiveEndpointPolicy {
	if len(values) == 0 {
		return nil
	}
	out := make([]APIPositiveEndpointPolicy, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		if path == "" {
			continue
		}
		methods := normalizeUpperList(value.Methods)
		tokenIDs := normalizeTrimmedList(value.TokenIDs)
		contentTypes := normalizeTrimmedList(value.ContentTypes)
		for i := range contentTypes {
			contentTypes[i] = strings.ToLower(strings.TrimSpace(contentTypes[i]))
		}
		mode := strings.ToLower(strings.TrimSpace(value.Mode))
		key := strings.ToLower(path) + "\x00" + strings.Join(methods, ",") + "\x00" + strings.Join(tokenIDs, ",") + "\x00" + strings.Join(contentTypes, ",") + "\x00" + mode
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, APIPositiveEndpointPolicy{
			Path:         path,
			Methods:      methods,
			TokenIDs:     tokenIDs,
			ContentTypes: contentTypes,
			Mode:         mode,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return strings.Join(out[i].Methods, ",") < strings.Join(out[j].Methods, ",")
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func normalizeStatusCodes(values []int) []int {
	items := append([]int(nil), values...)
	sort.Ints(items)
	return slices.Compact(items)
}

func normalizeCustomLimitRules(values []CustomLimitRule) []CustomLimitRule {
	if len(values) == 0 {
		return nil
	}
	items := make([]CustomLimitRule, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		rate := normalizeLimitReqRate(value.Rate)
		if path == "" || rate == "" {
			continue
		}
		key := strings.ToLower(path) + "\x00" + rate
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, CustomLimitRule{Path: path, Rate: rate})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path == items[j].Path {
			return items[i].Rate < items[j].Rate
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func normalizeAntibotChallengeRules(values []AntibotChallengeRule) []AntibotChallengeRule {
	if len(values) == 0 {
		return nil
	}
	items := make([]AntibotChallengeRule, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		challenge := strings.ToLower(strings.TrimSpace(value.Challenge))
		if path == "" || challenge == "" {
			continue
		}
		key := strings.ToLower(path) + "\x00" + challenge
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, AntibotChallengeRule{
			Path:      path,
			Challenge: challenge,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path == items[j].Path {
			return items[i].Challenge < items[j].Challenge
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func normalizeAuthUsers(values []SecurityAuthUser) []SecurityAuthUser {
	if len(values) == 0 {
		return nil
	}
	out := make([]SecurityAuthUser, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		username := strings.TrimSpace(value.Username)
		if username == "" {
			continue
		}
		key := strings.ToLower(username)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, SecurityAuthUser{
			Username:    username,
			Password:    strings.TrimSpace(value.Password),
			Enabled:     value.Enabled,
			LastLoginAt: strings.TrimSpace(value.LastLoginAt),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Username) < strings.ToLower(out[j].Username)
	})
	return out
}

func hasEnabledUserWithPassword(values []SecurityAuthUser) bool {
	for _, value := range values {
		if !value.Enabled {
			continue
		}
		if strings.TrimSpace(value.Password) == "" {
			continue
		}
		return true
	}
	return false
}

func normalizeLimitReqRate(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return value
	}
	if strings.HasSuffix(value, "r/s") {
		return value
	}
	if num, err := strconv.Atoi(value); err == nil && num > 0 {
		return fmt.Sprintf("%dr/s", num)
	}
	return value
}

func normalizeBanEscalationStages(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value < 0 {
			continue
		}
		out = append(out, value)
		if value == 0 {
			// Permanent stage is terminal.
			break
		}
	}
	return out
}

func isValidCountrySelector(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	if _, ok := allowedCountrySelectors[value]; ok {
		return true
	}
	return len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z'
}

func isSafeHeaderValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return safeHeaderValueRegexp.MatchString(trimmed)
}

func isAllowedReferrerPolicy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "no-referrer",
		"no-referrer-when-downgrade",
		"same-origin",
		"origin",
		"strict-origin",
		"origin-when-cross-origin",
		"strict-origin-when-cross-origin",
		"unsafe-url":
		return true
	default:
		return false
	}
}
