package compiler

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type easySiteData struct {
	SiteID                       string
	RateLimitCookieVar           string
	RateLimitEscalationCookieVar string
	ExceptionVar                 string
	AllowedMethodsPattern        string
	MaxClientSize                string

	ReferrerPolicy        string
	ContentSecurityPolicy string
	PermissionsPolicy     string
	UseCORS               bool
	CORSAllowedOrigins    string

	ReverseProxyCustomHost string
	ReverseProxySSLSNI     bool
	ReverseProxySSLSNIName string
	ReverseProxyWebsocket  bool
	ReverseProxyKeepalive  bool
	PassHostHeader         bool
	SendXForwardedFor      bool
	SendXForwardedProto    bool
	SendXRealIP            bool
	RateLimitBanSeconds    int
	AdminBypassPathPattern string

	UseAuthBasic      bool
	AuthBasicRealm    string
	AuthBasicUserFile string
	AuthGateLoginURI  string
	AuthGateVerifyURI string
	AuthGateCookieKey string
	AuthGateCookieVal string
	AuthGateCookieTTL int

	AntibotEnabled           bool
	AntibotTwoLayerEnabled   bool
	AntibotUsesInterstitial  bool
	AntibotChallenge         string
	AntibotEscalationMode    string
	AntibotURI               string
	AntibotVerifyURI         string
	AntibotStage1URI         string
	AntibotStage1VerifyURI   string
	AntibotRedirectURI       string
	AntibotStage1RedirectURI string
	AntibotStage1CookieName  string
	AntibotStage1CookieValue string
	AntibotCookieName        string
	AntibotCookieValue       string
	AntibotRecaptchaHint     string
	AntibotHcaptchaHint      string
	AntibotTurnstileHint     string
	AntibotRuleOverrides     []easyAntibotRuleData
	AntibotScannerAutoBan    bool
	AntibotScannerPattern    string

	BlacklistIP        []string
	BlacklistUserAgent []string
	BlacklistURI       []string

	BlacklistCountryGuardPattern string
	WhitelistCountryGuardPattern string

	UseModSecurity         bool
	UseModSecurityEasyFile bool
	ModSecurityEasyRules   string
	ModSecurityEasyRulesOn bool
}

type l4GuardConfigData struct {
	Enabled       bool   `json:"enabled"`
	ChainMode     string `json:"chain_mode"`
	ConnLimit     int    `json:"conn_limit"`
	RatePerSec    int    `json:"rate_per_second"`
	RateBurst     int    `json:"rate_burst"`
	Ports         []int  `json:"ports"`
	Target        string `json:"target"`
	DestinationIP string `json:"destination_ip"`
}

type antibotChallengePageData struct {
	VerifyURI string
}

type authGatePageData struct {
	VerifyURI string
}

type easyAntibotRuleData struct {
	GuardPattern string
	Challenge    string
	RedirectURI  string
}

// RenderEasyArtifacts compiles Easy-mode site directives into per-site nginx snippets.
func RenderEasyArtifacts(sites []SiteInput, profiles []EasyProfileInput) ([]ArtifactOutput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool { return sortedSites[i].ID < sortedSites[j].ID })

	profileBySite := make(map[string]EasyProfileInput, len(profiles))
	for _, profile := range profiles {
		siteID := strings.TrimSpace(profile.SiteID)
		if siteID == "" {
			return nil, fmt.Errorf("easy profile site id is required")
		}
		profile.AllowedMethods = sortedUnique(profile.AllowedMethods)
		if len(profile.AllowedMethods) == 0 {
			profile.AllowedMethods = []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
		}
		profile.PermissionsPolicy = sortedUnique(profile.PermissionsPolicy)
		profile.CORSAllowedOrigins = sortedUnique(profile.CORSAllowedOrigins)
		profile.BlacklistIP = sortedUnique(profile.BlacklistIP)
		profile.BlacklistUserAgent = sortedUnique(profile.BlacklistUserAgent)
		profile.BlacklistURI = sortedUnique(profile.BlacklistURI)
		profile.BlacklistURI = normalizeBlacklistURIPatterns(profile.BlacklistURI)
		profile.BlacklistCountry = sortedUniqueUpper(profile.BlacklistCountry)
		profile.WhitelistCountry = sortedUniqueUpper(profile.WhitelistCountry)
		profile.MaxClientSize = strings.TrimSpace(profile.MaxClientSize)
		if profile.MaxClientSize == "" {
			profile.MaxClientSize = "100m"
		}
		profile.ReferrerPolicy = strings.TrimSpace(profile.ReferrerPolicy)
		profile.ContentSecurityPolicy = strings.TrimSpace(profile.ContentSecurityPolicy)
		profile.ReverseProxyCustomHost = strings.TrimSpace(profile.ReverseProxyCustomHost)
		profile.ReverseProxySSLSNIName = strings.TrimSpace(profile.ReverseProxySSLSNIName)
		profile.AuthBasicUser = strings.TrimSpace(profile.AuthBasicUser)
		profile.AuthBasicPassword = strings.TrimSpace(profile.AuthBasicPassword)
		profile.AuthBasicText = strings.TrimSpace(profile.AuthBasicText)
		if profile.AuthBasicText == "" {
			profile.AuthBasicText = "Restricted area"
		}
		profile.AuthUsers = normalizeAuthUsers(profile.AuthUsers)
		if len(profile.AuthUsers) == 0 && profile.AuthBasicUser != "" {
			profile.AuthUsers = []ServiceAuthUserInput{
				{
					Username: profile.AuthBasicUser,
					Password: profile.AuthBasicPassword,
					Enabled:  true,
				},
			}
		}
		if profile.AuthBasicUser == "" && len(profile.AuthUsers) > 0 {
			profile.AuthBasicUser = profile.AuthUsers[0].Username
			profile.AuthBasicPassword = profile.AuthUsers[0].Password
		}
		if profile.AuthSessionTTLMin < -1 {
			profile.AuthSessionTTLMin = -1
		}
		if profile.AuthSessionTTLMin == 0 {
			profile.AuthSessionTTLMin = 60
		}
		if profile.AuthSessionTTLMin > 1440 {
			profile.AuthSessionTTLMin = 1440
		}
		profile.AntibotChallenge = strings.ToLower(strings.TrimSpace(profile.AntibotChallenge))
		profile.SecurityMode = strings.ToLower(strings.TrimSpace(profile.SecurityMode))
		switch profile.SecurityMode {
		case "block", "monitor", "transparent":
		default:
			profile.SecurityMode = "block"
		}
		profile.AntibotURI = strings.TrimSpace(profile.AntibotURI)
		profile.ChallengeEscalationMode = strings.ToLower(strings.TrimSpace(profile.ChallengeEscalationMode))
		if profile.ChallengeEscalationMode == "" {
			profile.ChallengeEscalationMode = "javascript"
		}
		profile.AntibotChallengeRules = normalizeCompilerAntibotRules(profile.AntibotChallengeRules)
		if profile.AntibotChallenge == "" {
			profile.AntibotChallenge = "no"
		}
		if profile.AntibotURI == "" {
			profile.AntibotURI = "/challenge"
		}
		if !strings.HasPrefix(profile.AntibotURI, "/") {
			profile.AntibotURI = "/" + profile.AntibotURI
		}
		profile.ModSecurityCRSVersion = strings.TrimSpace(profile.ModSecurityCRSVersion)
		if profile.ModSecurityCRSVersion == "" {
			profile.ModSecurityCRSVersion = "4"
		}
		profile.ModSecurityCRSPlugins = sortedUnique(profile.ModSecurityCRSPlugins)
		profile.ModSecurityCustomPath = strings.TrimSpace(profile.ModSecurityCustomPath)
		if profile.ModSecurityCustomPath == "" {
			profile.ModSecurityCustomPath = "modsec/anomaly_score.conf"
		}
		profile.ModSecurityCustomContent = strings.TrimSpace(profile.ModSecurityCustomContent)
		profile.OpenAPISchemaRef = strings.TrimSpace(profile.OpenAPISchemaRef)
		profile.APIEnforcementMode = strings.ToLower(strings.TrimSpace(profile.APIEnforcementMode))
		if profile.APIEnforcementMode == "" {
			profile.APIEnforcementMode = "monitor"
		}
		profile.APIDefaultAction = strings.ToLower(strings.TrimSpace(profile.APIDefaultAction))
		if profile.APIDefaultAction == "" {
			profile.APIDefaultAction = "allow"
		}
		for idx := range profile.APIEndpointPolicies {
			policy := &profile.APIEndpointPolicies[idx]
			policy.Path = strings.TrimSpace(policy.Path)
			policy.Mode = strings.ToLower(strings.TrimSpace(policy.Mode))
			policy.Methods = sortedUniqueUpper(policy.Methods)
			policy.TokenIDs = sortedUnique(policy.TokenIDs)
			contentTypes := sortedUnique(policy.ContentTypes)
			for i := range contentTypes {
				contentTypes[i] = strings.ToLower(strings.TrimSpace(contentTypes[i]))
			}
			policy.ContentTypes = contentTypes
		}
		if profile.BadBehaviorBanTimeSeconds < 0 {
			profile.BadBehaviorBanTimeSeconds = 0
		}

		profileBySite[siteID] = profile
	}

	artifacts := make([]ArtifactOutput, 0, len(sortedSites)*2)
	l4ConnLimit := 200
	l4RatePerSec := 100
	l4Enabled := false
	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}
		profile, ok := profileBySite[site.ID]
		if !ok {
			profile = EasyProfileInput{
				SiteID:                    site.ID,
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
			}
		}
		if profile.UseLimitConn {
			l4Enabled = true
			if profile.LimitConnMaxHTTP1 > l4ConnLimit {
				l4ConnLimit = profile.LimitConnMaxHTTP1
			}
		}
		if profile.UseLimitReq {
			l4Enabled = true
			if parsed := parseRatePerSecond(profile.LimitReqRate); parsed > l4RatePerSec {
				l4RatePerSec = parsed
			}
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
			UseAuthBasic:                 profile.UseAuthBasic && len(authUsers) > 0,
			AuthBasicRealm:               profile.AuthBasicText,
			AuthBasicUserFile:            fmt.Sprintf("/etc/waf/nginx/auth-basic/%s.htpasswd", site.ID),
			AuthGateLoginURI:             authGateLoginURI(),
			AuthGateVerifyURI:            authGateVerifyURI(),
			AuthGateCookieKey:            authGateCookieName(site.ID),
			AuthGateCookieVal:            authGateCookieValue(site.ID, profile),
			AuthGateCookieTTL:            authGateCookieTTLSeconds(profile),
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
			AntibotRuleOverrides:         buildEasyAntibotRuleData(profile.AntibotChallengeRules, profile.AntibotURI),
			AntibotScannerAutoBan:        profile.AntibotScannerAutoBan,
			AntibotScannerPattern:        antibotScannerPattern(),
			BlacklistIP:                  profile.BlacklistIP,
			BlacklistUserAgent:           profile.BlacklistUserAgent,
			BlacklistURI:                 profile.BlacklistURI,
			BlacklistCountryGuardPattern: blacklistCountryGuardPattern(profile.BlacklistCountry),
			WhitelistCountryGuardPattern: whitelistCountryGuardPattern(profile.WhitelistCountry),
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
			),
		}

		content, err := renderTemplate(filepath.Join(templatesRoot(), "easy", "site.conf.tmpl"), data)
		if err != nil {
			return nil, fmt.Errorf("render easy site template for %s: %w", site.ID, err)
		}
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/easy/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			content,
		))

		if data.UseAuthBasic {
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
			authPage, err := renderTemplate(filepath.Join(templatesRoot(), "..", "errors", "auth.html.tmpl"), authGatePageData{
				VerifyURI: data.AuthGateVerifyURI,
			})
			if err != nil {
				return nil, fmt.Errorf("render auth gate page for %s: %w", site.ID, err)
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
		if data.AntibotUsesInterstitial {
			challengePage, err := renderTemplate(filepath.Join(templatesRoot(), "..", "errors", "antibot.html.tmpl"), antibotChallengePageData{
				VerifyURI: data.AntibotVerifyURI,
			})
			if err != nil {
				return nil, fmt.Errorf("render antibot challenge page for %s: %w", site.ID, err)
			}
			artifacts = append(artifacts, newArtifact(
				fmt.Sprintf("errors/%s/antibot.html", site.ID),
				ArtifactKindNginxConfig,
				challengePage,
			))
		}
	}
	if l4Enabled {
		l4 := l4GuardConfigData{
			Enabled:       true,
			ChainMode:     "auto",
			ConnLimit:     l4ConnLimit,
			RatePerSec:    l4RatePerSec,
			RateBurst:     l4RatePerSec * 2,
			Ports:         []int{80, 443},
			Target:        "DROP",
			DestinationIP: "",
		}
		raw, err := json.MarshalIndent(l4, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode l4 guard config: %w", err)
		}
		raw = append(raw, '\n')
		artifacts = append(artifacts, newArtifact(
			"l4guard/config.json",
			ArtifactKindNginxConfig,
			raw,
		))
	}

	return artifacts, nil
}

func rateLimitBanSeconds(profile EasyProfileInput) int {
	if !profile.UseBadBehavior {
		return 0
	}
	for _, code := range profile.BadBehaviorStatusCodes {
		if code == 429 {
			if profile.BadBehaviorBanTimeSeconds == 0 {
				// "Permanent" for local ban semantics: keep a very long max-age.
				return 2147483647
			}
			if profile.BadBehaviorBanTimeSeconds < 0 {
				return 0
			}
			return profile.BadBehaviorBanTimeSeconds
		}
	}
	return 0
}

func antibotCookieName(siteID string) string {
	return "waf_antibot_" + shortStableHash(siteID)
}

func authGateLoginURI() string {
	return "/auth"
}

func authGateVerifyURI() string {
	return "/auth/verify"
}

func authGateCookieName(siteID string) string {
	return "waf_auth_" + shortStableHash(siteID)
}

func authGateCookieValue(siteID string, profile EasyProfileInput) string {
	enabled := enabledAuthUsers(profile.AuthUsers)
	parts := make([]string, 0, len(enabled)+4)
	parts = append(parts, siteID, profile.AuthBasicText, strconv.Itoa(profile.AuthSessionTTLMin))
	for _, user := range enabled {
		parts = append(parts, strings.TrimSpace(user.Username)+":"+strings.TrimSpace(user.Password))
	}
	return shortStableHash(strings.Join(parts, "|"))
}

func authGateCookieTTLSeconds(profile EasyProfileInput) int {
	if profile.AuthSessionTTLMin < 0 {
		return -1
	}
	if profile.AuthSessionTTLMin == 0 {
		return 3600
	}
	return profile.AuthSessionTTLMin * 60
}

func antibotUsesInterstitial(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha":
		return true
	default:
		return false
	}
}

func antibotVerifyURI(challengeURI string) string {
	trimmed := strings.TrimSpace(challengeURI)
	if trimmed == "" || trimmed == "/" {
		return "/challenge/verify"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed + "/verify"
}

func antibotStage1URI(challengeURI string) string {
	trimmed := strings.TrimSpace(challengeURI)
	if trimmed == "" || trimmed == "/" {
		return "/challenge/stage1"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed + "/stage1"
}

func antibotStage1VerifyURI(challengeURI string) string {
	return antibotVerifyURI(antibotStage1URI(challengeURI))
}

func antibotRedirectURI(mode string, challengeURI string) string {
	if antibotUsesInterstitial(mode) {
		return strings.TrimSpace(challengeURI)
	}
	return antibotVerifyURI(challengeURI)
}

func antibotCookieValue(siteID string, profile EasyProfileInput) string {
	return shortStableHash(strings.Join([]string{
		siteID,
		profile.AntibotChallenge,
		profile.AntibotURI,
		profile.AntibotRecaptchaKey,
		profile.AntibotHcaptchaKey,
		profile.AntibotTurnstileKey,
	}, "|"))
}

func antibotStage1CookieName(siteID string) string {
	return "waf_antibot_s1_" + shortStableHash(siteID)
}

func antibotStage1CookieValue(siteID string, profile EasyProfileInput) string {
	return shortStableHash(strings.Join([]string{
		siteID,
		profile.AntibotChallenge,
		profile.AntibotURI,
		profile.ChallengeEscalationMode,
	}, "|"))
}

func shortStableHash(value string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(value)))
	return fmt.Sprintf("%x", sum[:6])
}

func normalizeAuthUsers(values []ServiceAuthUserInput) []ServiceAuthUserInput {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]ServiceAuthUserInput, 0, len(values))
	for _, item := range values {
		username := strings.TrimSpace(item.Username)
		if username == "" {
			continue
		}
		key := strings.ToLower(username)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ServiceAuthUserInput{
			Username:    username,
			Password:    strings.TrimSpace(item.Password),
			Enabled:     item.Enabled,
			LastLoginAt: strings.TrimSpace(item.LastLoginAt),
		})
	}
	return out
}

func enabledAuthUsers(values []ServiceAuthUserInput) []ServiceAuthUserInput {
	out := make([]ServiceAuthUserInput, 0, len(values))
	for _, item := range normalizeAuthUsers(values) {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(item.Password) == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func rateLimitCookieVar(siteID string) string {
	return rateLimitCookieName(siteID)
}

func rateLimitEscalationCookieVar(siteID string) string {
	return rateLimitEscalationCookieName(siteID)
}

func methodPattern(methods []string) string {
	escaped := make([]string, 0, len(methods))
	for _, method := range methods {
		escaped = append(escaped, strings.ToUpper(strings.TrimSpace(method)))
	}
	return "^(" + strings.Join(escaped, "|") + ")$"
}

func sortedUniqueUpper(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return compactStrings(items)
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value == out[len(out)-1] {
			continue
		}
		out = append(out, value)
	}
	return out
}

func toQuotedList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Quote(v))
	}
	return strings.Join(parts, " ")
}

func blacklistCountrySelectorPattern(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.ToUpper(strings.TrimSpace(value))
		if token == "" {
			continue
		}
		parts = append(parts, regexp.QuoteMeta(token))
	}
	if len(parts) == 0 {
		return ""
	}
	return "^(?:" + strings.Join(parts, "|") + ")$"
}

func whitelistCountryGuardPattern(values []string) string {
	basePattern := blacklistCountrySelectorPattern(values)
	if basePattern == "" {
		return ""
	}
	if strings.HasPrefix(basePattern, "^(?:") && strings.HasSuffix(basePattern, ")$") {
		core := strings.TrimSuffix(strings.TrimPrefix(basePattern, "^(?:"), ")$")
		// Allow empty country code (GeoIP unavailable) to preserve existing behavior.
		return "^(?:1:.*|0:(?:|" + core + "))$"
	}
	return ""
}

func blacklistCountryGuardPattern(values []string) string {
	basePattern := blacklistCountrySelectorPattern(values)
	if basePattern == "" {
		return ""
	}
	if strings.HasPrefix(basePattern, "^(?:") && strings.HasSuffix(basePattern, ")$") {
		core := strings.TrimSuffix(strings.TrimPrefix(basePattern, "^(?:"), ")$")
		return "^(?:0:(?:" + core + "))$"
	}
	return ""
}

func buildSHA1HTPasswdLine(user, password string) string {
	sum := sha1.Sum([]byte(password))
	hash := base64.StdEncoding.EncodeToString(sum[:])
	return user + ":{SHA}" + hash + "\n"
}

func buildEasyModSecurityRules(
	siteID,
	securityMode string,
	useCRSPlugins bool,
	crsVersion string,
	plugins []string,
	useCustomConfiguration bool,
	customPath,
	customContent string,
	useAPIPositiveSecurity bool,
	openAPISchemaRef,
	apiEnforcementMode,
	apiDefaultAction string,
	apiEndpointPolicies []APIPositiveEndpointPolicyInput,
) string {
	engineDirective := "SecRuleEngine On"
	switch strings.ToLower(strings.TrimSpace(securityMode)) {
	case "monitor":
		engineDirective = "SecRuleEngine DetectionOnly"
	case "transparent":
		engineDirective = "SecRuleEngine Off"
	default:
		engineDirective = "SecRuleEngine On"
	}
	lines := []string{
		"# Easy ModSecurity directives",
		"# site: " + siteID,
		"# security_mode: " + strings.TrimSpace(securityMode),
		"# crs_version: " + strings.TrimSpace(crsVersion),
		engineDirective,
	}
	if engineDirective != "SecRuleEngine Off" {
		lines = append(lines,
			"Include /etc/waf/modsecurity/crs-setup.conf",
			"Include /etc/waf/modsecurity/coreruleset/rules/*.conf",
		)
	}
	if useCRSPlugins && len(plugins) > 0 {
		lines = append(lines, "# crs_plugins: "+strings.Join(plugins, " "))
		for _, plugin := range plugins {
			lines = append(lines, fmt.Sprintf("Include /etc/waf/modsecurity/crs-overrides/%s.conf", strings.TrimSpace(plugin)))
		}
	}
	if useCustomConfiguration && strings.TrimSpace(customPath) != "" {
		lines = append(lines, "# custom_path: "+strings.TrimSpace(customPath))
	}
	if useCustomConfiguration && strings.TrimSpace(customContent) != "" {
		lines = append(lines, customContent)
	}
	if useAPIPositiveSecurity {
		lines = append(lines, buildAPIPositiveModSecurityRules(siteID, openAPISchemaRef, apiEnforcementMode, apiDefaultAction, apiEndpointPolicies)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildAPIPositiveModSecurityRules(siteID, schemaRef, enforcementMode, defaultAction string, policies []APIPositiveEndpointPolicyInput) []string {
	lines := []string{
		"# API Positive Security directives",
		"# api_positive_site: " + siteID,
		"# api_positive_schema_ref: " + strings.TrimSpace(schemaRef),
		"# api_positive_enforcement_mode: " + strings.TrimSpace(enforcementMode),
		"# api_positive_default_action: " + strings.TrimSpace(defaultAction),
	}

	blocking := strings.ToLower(strings.TrimSpace(enforcementMode)) == "block"
	defaultDeny := strings.ToLower(strings.TrimSpace(defaultAction)) == "deny"
	ruleID := 190000
	knownPaths := make([]string, 0, len(policies))

	for _, policy := range policies {
		path := strings.TrimSpace(policy.Path)
		if path == "" {
			continue
		}
		knownPaths = append(knownPaths, regexp.QuoteMeta(path))

		mode := strings.ToLower(strings.TrimSpace(policy.Mode))
		policyBlocking := blocking
		if mode == "block" {
			policyBlocking = true
		}
		if mode == "monitor" {
			policyBlocking = false
		}
		action := "pass,log"
		if policyBlocking {
			action = "deny,status:403,log"
		}

		if len(policy.Methods) > 0 {
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_METHOD "!@within %s" "t:none,%s,msg:'API positive security: method mismatch for %s'"`, strings.Join(policy.Methods, " "), action, path),
			)
			ruleID += 2
		}
		if len(policy.TokenIDs) > 0 {
			tokenPattern := "(?:" + strings.Join(quoteMetaList(policy.TokenIDs), "|") + ")"
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_HEADERS:X-WAF-API-TOKEN-ID "!@rx ^%s$" "t:none,%s,msg:'API positive security: token mismatch for %s'"`, tokenPattern, action, path),
			)
			ruleID += 2
		}
		if len(policy.ContentTypes) > 0 {
			contentTypePattern := "(?:" + strings.Join(quoteMetaList(policy.ContentTypes), "|") + ")"
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_HEADERS:Content-Type "!@rx ^%s(?:;|$)" "t:none,%s,msg:'API positive security: content-type mismatch for %s'"`, contentTypePattern, action, path),
			)
			ruleID += 2
		}
	}

	if defaultDeny && len(knownPaths) > 0 {
		action := "pass,log"
		if blocking {
			action = "deny,status:403,log"
		}
		lines = append(lines,
			fmt.Sprintf(`SecRule REQUEST_URI "!@rx ^(?:%s)(?:$|/)" "id:%d,phase:1,t:none,%s,msg:'API positive security: unknown endpoint'"`, strings.Join(knownPaths, "|"), ruleID, action),
		)
	}

	return lines
}

func quoteMetaList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		out = append(out, regexp.QuoteMeta(v))
	}
	return out
}

func parseRatePerSecond(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimSuffix(value, "r/s")
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func normalizeCompilerAntibotRules(values []AntibotChallengeRuleInput) []AntibotChallengeRuleInput {
	if len(values) == 0 {
		return nil
	}
	items := make([]AntibotChallengeRuleInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		challenge := strings.ToLower(strings.TrimSpace(value.Challenge))
		if path == "" || challenge == "" || challenge == "no" {
			continue
		}
		key := strings.ToLower(path) + "\x00" + challenge
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, AntibotChallengeRuleInput{
			Path:      path,
			Challenge: challenge,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftPriority := customLimitPriority(items[i].Path)
		rightPriority := customLimitPriority(items[j].Path)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if len(items[i].Path) != len(items[j].Path) {
			return len(items[i].Path) > len(items[j].Path)
		}
		if items[i].Path == items[j].Path {
			return items[i].Challenge < items[j].Challenge
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func buildEasyAntibotRuleData(rules []AntibotChallengeRuleInput, challengeURI string) []easyAntibotRuleData {
	if len(rules) == 0 {
		return nil
	}
	out := make([]easyAntibotRuleData, 0, len(rules))
	for _, rule := range rules {
		pattern := antibotRuleGuardPattern(rule.Path)
		if pattern == "" {
			continue
		}
		out = append(out, easyAntibotRuleData{
			GuardPattern: pattern,
			Challenge:    rule.Challenge,
			RedirectURI:  antibotRedirectURI(rule.Challenge, challengeURI),
		})
	}
	return out
}

func antibotRuleGuardPattern(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if hasComplexWildcard(trimmed) {
		return wildcardPathToRegex(trimmed)
	}
	if strings.HasSuffix(trimmed, "*") {
		base := strings.TrimSuffix(trimmed, "*")
		if base == "" {
			return "^/"
		}
		return "^" + regexp.QuoteMeta(base)
	}
	if strings.HasSuffix(trimmed, "/") {
		return "^" + regexp.QuoteMeta(trimmed)
	}
	return "^" + regexp.QuoteMeta(trimmed) + "$"
}

func normalizeBlacklistURIPatterns(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		pattern := strings.TrimSpace(value)
		if pattern == "" {
			continue
		}
		pattern = toSafeBlacklistURIRegex(pattern)
		if pattern == "" {
			continue
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		out = append(out, pattern)
	}
	sort.Strings(out)
	return out
}

func toSafeBlacklistURIRegex(pattern string) string {
	// Keep existing regex behavior when the input is already a valid expression.
	if _, err := regexp.Compile(pattern); err == nil {
		return pattern
	}
	// Fallback: interpret as wildcard expression (`*`, `?`) and escape everything else.
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, `.*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `.`)
	return quoted
}

func antibotScannerPattern() string {
	patterns := []string{
		`/(?:\.env|\.git|phpmyadmin|wp-admin|vendor/phpunit|cgi-bin|boaform)`,
		`acunetix`,
		`dirbuster`,
		`dirsearch`,
		`feroxbuster`,
		`ffuf`,
		`gobuster`,
		`gospider`,
		`hakrawler`,
		`masscan`,
		`nessus`,
		`nikto`,
		`nmap`,
		`sqlmap`,
		`wfuzz`,
		`wpscan`,
		`zgrab`,
		`securityscanner`,
	}
	return strings.Join(patterns, "|")
}
