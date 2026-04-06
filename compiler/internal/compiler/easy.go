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
	RateLimitBanSeconds    int

	UseAuthBasic      bool
	AuthBasicRealm    string
	AuthBasicUserFile string

	AntibotEnabled       bool
	AntibotChallenge     string
	AntibotURI           string
	AntibotRecaptchaHint string
	AntibotHcaptchaHint  string
	AntibotTurnstileHint string

	BlacklistIP        []string
	BlacklistUserAgent []string
	BlacklistURI       []string

	BlacklistCountryPattern string
	WhitelistCountryPattern string

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
		profile.AntibotChallenge = strings.ToLower(strings.TrimSpace(profile.AntibotChallenge))
		profile.SecurityMode = strings.ToLower(strings.TrimSpace(profile.SecurityMode))
		switch profile.SecurityMode {
		case "block", "monitor", "transparent":
		default:
			profile.SecurityMode = "block"
		}
		profile.AntibotURI = strings.TrimSpace(profile.AntibotURI)
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
		data := easySiteData{
			SiteID:                       site.ID,
			RateLimitCookieVar:           rateLimitCookieVar(site.ID),
			RateLimitEscalationCookieVar: rateLimitEscalationCookieVar(site.ID),
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
			RateLimitBanSeconds:          rateLimitBanSeconds(profile),
			UseAuthBasic:                 profile.UseAuthBasic && profile.AuthBasicUser != "" && profile.AuthBasicPassword != "",
			AuthBasicRealm:               profile.AuthBasicText,
			AuthBasicUserFile:            fmt.Sprintf("/etc/waf/nginx/auth-basic/%s.htpasswd", site.ID),
			AntibotEnabled:               profile.AntibotChallenge != "" && profile.AntibotChallenge != "no",
			AntibotChallenge:             profile.AntibotChallenge,
			AntibotURI:                   profile.AntibotURI,
			AntibotRecaptchaHint:         strings.TrimSpace(profile.AntibotRecaptchaKey),
			AntibotHcaptchaHint:          strings.TrimSpace(profile.AntibotHcaptchaKey),
			AntibotTurnstileHint:         strings.TrimSpace(profile.AntibotTurnstileKey),
			BlacklistIP:                  profile.BlacklistIP,
			BlacklistUserAgent:           profile.BlacklistUserAgent,
			BlacklistURI:                 profile.BlacklistURI,
			BlacklistCountryPattern:      blacklistCountrySelectorPattern(profile.BlacklistCountry),
			WhitelistCountryPattern:      whitelistCountrySelectorPattern(profile.WhitelistCountry),
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
			htpasswd := []byte(buildSHA1HTPasswdLine(profile.AuthBasicUser, profile.AuthBasicPassword))
			artifacts = append(artifacts, newArtifact(
				fmt.Sprintf("nginx/auth-basic/%s.htpasswd", site.ID),
				ArtifactKindNginxConfig,
				htpasswd,
			))
		}
		if data.ModSecurityEasyRulesOn {
			artifacts = append(artifacts, newArtifact(
				fmt.Sprintf("modsecurity/easy/%s.conf", site.ID),
				ArtifactKindModSecurity,
				[]byte(data.ModSecurityEasyRules),
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

func whitelistCountrySelectorPattern(values []string) string {
	blacklistPattern := blacklistCountrySelectorPattern(values)
	if blacklistPattern == "" {
		return ""
	}
	// Allow empty country code (GeoIP unavailable) to preserve existing behavior.
	if strings.HasPrefix(blacklistPattern, "^(?:") && strings.HasSuffix(blacklistPattern, ")$") {
		core := strings.TrimSuffix(strings.TrimPrefix(blacklistPattern, "^(?:"), ")$")
		return "^(?:" + "|" + core + ")$"
	}
	return blacklistPattern
}

func buildSHA1HTPasswdLine(user, password string) string {
	sum := sha1.Sum([]byte(password))
	hash := base64.StdEncoding.EncodeToString(sum[:])
	return user + ":{SHA}" + hash + "\n"
}

func buildEasyModSecurityRules(siteID, securityMode string, useCRSPlugins bool, crsVersion string, plugins []string, useCustomConfiguration bool, customPath, customContent string) string {
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
	return strings.Join(lines, "\n") + "\n"
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
