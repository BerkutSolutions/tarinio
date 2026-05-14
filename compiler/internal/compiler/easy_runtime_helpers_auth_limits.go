package compiler

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

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
