package compiler

import (
	"fmt"
	"path/filepath"
	"strings"
)

func defaultEasyProfileForSite(siteID string) EasyProfileInput {
	return EasyProfileInput{
		SiteID:                    siteID,
		SecurityMode:              "block",
		AllowedMethods:            []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"},
		MaxClientSize:             "100m",
		UseModSecurity:            true,
		UseModSecurityCRSPlugins:  true,
		UseLimitConn:              true,
		LimitConnMaxHTTP1:         200,
		UseLimitReq:               true,
		LimitReqRate:              "100r/s",
		PassHostHeader:            true,
		SendXForwardedFor:         true,
		SendXForwardedProto:       true,
		SendXRealIP:               false,
		AntibotScannerAutoBan:     true,
		UseBadBehavior:            true,
		BadBehaviorStatusCodes:    []int{400, 401, 403, 404, 405, 429, 444},
		BadBehaviorBanTimeSeconds: 30,
		AuthMode:                  authModeBasic,
		AuthOrder:                 authOrderAuthFirst,
	}
}

func renderEasySiteArtifacts(site SiteInput, profile EasyProfileInput) ([]ArtifactOutput, bool, int, int, error) {
	l4Enabled := false
	l4ConnLimit := 0
	l4RatePerSec := 0
	if profile.UseLimitConn {
		l4Enabled = true
		l4ConnLimit = profile.LimitConnMaxHTTP1
	}
	if profile.UseLimitReq {
		l4Enabled = true
		l4RatePerSec = parseRatePerSecond(profile.LimitReqRate)
	}

	permissionsPolicy := strings.Join(profile.PermissionsPolicy, ", ")
	corsAllowedOrigins := strings.Join(profile.CORSAllowedOrigins, " ")
	if corsAllowedOrigins == "" {
		corsAllowedOrigins = "*"
	}
	antibotEnabled := profile.AntibotChallenge != "" && profile.AntibotChallenge != "no"
	antibotTwoLayer := profile.ChallengeEscalationEnabled && antibotEnabled
	defaultChallenge := profile.AntibotChallenge
	if antibotTwoLayer && profile.ChallengeEscalationMode != "" && profile.ChallengeEscalationMode != "no" {
		defaultChallenge = profile.ChallengeEscalationMode
	}
	authUsers := enabledAuthUsers(profile.AuthUsers)
	if len(authUsers) == 0 && strings.TrimSpace(profile.AuthBasicUser) != "" && strings.TrimSpace(profile.AuthBasicPassword) != "" {
		authUsers = []ServiceAuthUserInput{{
			Username: strings.TrimSpace(profile.AuthBasicUser),
			Password: strings.TrimSpace(profile.AuthBasicPassword),
			Enabled:  true,
		}}
	}
	authTokens := enabledAuthServiceTokens(profile.AuthServiceTokens)
	authMode := normalizeCompilerAuthMode(profile.AuthMode)
	authOrder := normalizeCompilerAuthOrder(profile.AuthOrder)
	authEnabled := profile.UseAuthBasic && (len(authUsers) > 0 || len(authTokens) > 0)
	authBasicEnabled := authEnabled && authMode != authModeServiceToken && len(authUsers) > 0
	authTokenEnabled := authEnabled && authMode != authModeBasic && len(authTokens) > 0
	data := easySiteData{
		SiteID:                       site.ID,
		RateLimitCookieVar:           rateLimitCookieVar(site.ID),
		RateLimitEscalationCookieVar: rateLimitEscalationCookieVar(site.ID),
		ExceptionVar:                 siteExceptionVar(site.ID),
		AllowedMethodsPattern:        methodPattern(profile.AllowedMethods),
		MaxClientSize:                profile.MaxClientSize,
		ReferrerPolicy:               profile.ReferrerPolicy,
		ContentSecurityPolicy:        profile.ContentSecurityPolicy,
		PermissionsPolicy:            permissionsPolicy,
		HSTSHeader:                   buildHSTSHeaderValue(profile),
		UseCORS:                      profile.UseCORS,
		CORSAllowedOrigins:           corsAllowedOrigins,
		ReverseProxyCustomHost:       profile.ReverseProxyCustomHost,
		ReverseProxySSLSNI:           profile.ReverseProxySSLSNI,
		ReverseProxySSLSNIName:       profile.ReverseProxySSLSNIName,
		ReverseProxyWebsocket:        profile.ReverseProxyWebsocket,
		ReverseProxyKeepalive:        profile.ReverseProxyKeepalive,
		PassHostHeader:               profile.PassHostHeader,
		SendXForwardedFor:            profile.SendXForwardedFor,
		SendXForwardedProto:          profile.SendXForwardedProto,
		SendXRealIP:                  profile.SendXRealIP,
		RateLimitBanSeconds:          rateLimitBanSeconds(profile),
		AdminBypassPathPattern:       easyAdminBypassPathPatternForSite(site.ID),
		AuthEnabled:                  authEnabled,
		AuthBasicEnabled:             authBasicEnabled,
		AuthTokenEnabled:             authTokenEnabled,
		AuthMode:                     authMode,
		AuthOrder:                    authOrder,
		AuthRunsBeforeAntibot:        authOrder == authOrderAuthFirst,
		AuthBasicRealm:               profile.AuthBasicText,
		AuthBasicUserFile:            fmt.Sprintf("/etc/waf/nginx/auth-basic/%s.htpasswd", site.ID),
		AuthGateLoginURI:             authGateLoginURI(),
		AuthGateVerifyBasicURI:       authGateVerifyBasicURI(),
		AuthGateVerifyTokenURI:       authGateVerifyTokenURI(),
		AuthGateCookieKey:            authGateCookieName(site.ID),
		AuthGateCookieVal:            authGateCookieValue(site.ID, profile),
		AuthGateCookieTTL:            authGateCookieTTLSeconds(profile),
		AuthExclusionRules:           buildEasyAuthExclusionRuleData(profile.AuthExclusionRules),
		AuthTokenRules:               buildEasyAuthTokenRuleData(authTokens),
		AntibotEnabled:               antibotEnabled,
		AntibotTwoLayerEnabled:       antibotTwoLayer,
		AntibotUsesInterstitial:      antibotUsesInterstitial(defaultChallenge),
		AntibotChallenge:             defaultChallenge,
		AntibotEscalationMode:        profile.ChallengeEscalationMode,
		AntibotURI:                   profile.AntibotURI,
		AntibotVerifyURI:             antibotVerifyURI(profile.AntibotURI),
		AntibotStage1URI:             antibotStage1URI(profile.AntibotURI),
		AntibotStage1VerifyURI:       antibotStage1VerifyURI(profile.AntibotURI),
		AntibotRedirectURI:           antibotRedirectURI(defaultChallenge, profile.AntibotURI),
		AntibotStage1RedirectURI:     antibotVerifyURI(antibotStage1URI(profile.AntibotURI)),
		AntibotStage1CookieName:      antibotStage1CookieName(site.ID),
		AntibotStage1CookieValue:     antibotStage1CookieValue(site.ID, profile),
		AntibotCookieName:            antibotCookieName(site.ID),
		AntibotCookieValue:           antibotCookieValue(site.ID, profile),
		AntibotRecaptchaHint:         strings.TrimSpace(profile.AntibotRecaptchaKey),
		AntibotHcaptchaHint:          strings.TrimSpace(profile.AntibotHcaptchaKey),
		AntibotTurnstileHint:         strings.TrimSpace(profile.AntibotTurnstileKey),
		AntibotExclusionRules:        buildEasyAntibotExclusionRuleData(profile.AntibotExclusionRules),
		AntibotRuleOverrides:         buildEasyAntibotRuleData(profile.AntibotChallengeRules, profile.AntibotURI),
		AntibotScannerAutoBan:        profile.AntibotScannerAutoBan,
		AntibotScannerPattern:        antibotScannerPattern(),
		BlacklistIP:                  profile.BlacklistIP,
		BlacklistUserAgent:           profile.BlacklistUserAgent,
		BlacklistURI:                 profile.BlacklistURI,
		BlacklistJA3:                 profile.BlacklistJA3,
		ExceptionsURI:                profile.ExceptionsURI,
		BlacklistCountryGuardPattern: blacklistCountryGuardPattern(profile.BlacklistCountry),
		WhitelistCountryGuardPattern: whitelistCountryGuardPattern(profile.WhitelistCountry),
		ShowGeoBlockPage:             profile.ShowGeoBlockPage,
		GeoTimeWindowSnippet:         buildGeoTimeWindowServerSnippet(site.ID, profile.GeoTimeWindows, "$"+siteExceptionVar(site.ID)),
		WSInspectionSnippet:          buildWSInspectionServerSnippet(site.ID, profile.WSInspection),
		MTLSSnippet:                  buildMTLSServerSnippet(profile.MTLS),
		UpstreamMTLSSnippet:          buildUpstreamMTLSSnippet(profile.UpstreamMTLS),
		UseModSecurity:               profile.UseModSecurity,
		UseModSecurityEasyFile:       profile.UseModSecurity,
		ModSecurityEasyRulesOn:       profile.UseModSecurity,
		ModSecurityEasyRules: buildEasyModSecurityRules(
			site.ID,
			profile.SecurityMode,
			profile.UseModSecurityCRSPlugins,
			profile.ModSecurityCRSVersion,
			profile.ModSecurityCRSPlugins,
			profile.UseModSecurityCustomConfiguration,
			profile.ModSecurityCustomPath,
			profile.ModSecurityCustomContent,
			profile.UseAPIPositiveSecurity,
			profile.OpenAPISchemaRef,
			profile.APIEnforcementMode,
			profile.APIDefaultAction,
			profile.APIEndpointPolicies,
			profile.VirtualPatches,
		),
		HttpStrictParsing:   profile.HttpStrictParsing,
		CookieFlags:         profile.CookieFlags,
		KeepUpstreamHeaders: profile.KeepUpstreamHeaders,
		HealthCheckEnabled:         profile.HealthCheckEnabled,
		HealthCheckPath:            profile.HealthCheckPath,
		HealthCheckIntervalSeconds: profile.HealthCheckIntervalSeconds,
		HealthCheckFailThreshold:   profile.HealthCheckFailThreshold,
	}

	content, err := renderTemplate(filepath.Join(templatesRoot(), "easy", "site.conf.tmpl"), data)
	if err != nil {
		return nil, false, 0, 0, fmt.Errorf("render easy site template for %s: %w", site.ID, err)
	}
	artifacts := []ArtifactOutput{
		newArtifact(
			fmt.Sprintf("nginx/easy/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			content,
		),
	}

	if data.AuthBasicEnabled {
		lines := make([]string, 0, len(authUsers))
		for _, user := range authUsers {
			lines = append(lines, buildSHA1HTPasswdLine(user.Username, user.Password))
		}
		htpasswd := []byte(strings.Join(lines, "\n"))
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/auth-basic/%s.htpasswd", site.ID),
			ArtifactKindNginxConfig,
			htpasswd,
		))
	}
	if data.AuthEnabled {
		authPage, err := renderTemplate(filepath.Join(templatesRoot(), "..", "errors", "auth.html.tmpl"), authGatePageData{
			BasicVerifyURI:  data.AuthGateVerifyBasicURI,
			TokenVerifyURI:  data.AuthGateVerifyTokenURI,
			UseBasic:        data.AuthBasicEnabled,
			UseServiceToken: data.AuthTokenEnabled,
		})
		if err != nil {
			return nil, false, 0, 0, fmt.Errorf("render auth gate page for %s: %w", site.ID, err)
		}
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("errors/%s/auth.html", site.ID),
			ArtifactKindNginxConfig,
			authPage,
		))
	}
	if data.ModSecurityEasyRulesOn {
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("modsecurity/easy/%s.conf", site.ID),
			ArtifactKindModSecurity,
			[]byte(data.ModSecurityEasyRules),
		))
	}
	// geo time-window http-context maps (must be included from http block)
	if geoConf := buildGeoTimeWindowHttpConf(site.ID, profile.GeoTimeWindows); geoConf != "" {
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/geo-timewindow/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			[]byte(geoConf),
		))
	}
	if data.AntibotUsesInterstitial {
		challengePage, err := renderTemplate(filepath.Join(templatesRoot(), "..", "errors", "antibot.html.tmpl"), antibotChallengePageData{
			VerifyURI: data.AntibotVerifyURI,
		})
		if err != nil {
			return nil, false, 0, 0, fmt.Errorf("render antibot challenge page for %s: %w", site.ID, err)
		}
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("errors/%s/antibot.html", site.ID),
			ArtifactKindNginxConfig,
			challengePage,
		))
	}
	return artifacts, l4Enabled, l4ConnLimit, l4RatePerSec, nil
}
