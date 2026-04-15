package compiler

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var sentryEnvelopePathRE = regexp.MustCompile(`^/api/\d+/envelope/?$`)

type easyRateLimitHTTPEntry struct {
	ZoneName string
	KeyVar   string
	Rate     string
}

type easyRateLimitHTTPData struct {
	Entries []easyRateLimitHTTPEntry
}

type easyLocationRuleData struct {
	LocationModifier            string
	LocationPath                string
	DisableProxyInterceptErrors bool
	ZoneName                    string
	Burst                       int
	StatusCode                  int
	PassHostHeader              bool
	ProxyPassTarget             string
}

type easyLocationData struct {
	SiteID string
	Rules  []easyLocationRuleData
}

func RenderEasyRateLimitArtifacts(sites []SiteInput, upstreams []UpstreamInput, profiles []EasyProfileInput) ([]ArtifactOutput, error) {
	sortedSites, upstreamByID, err := normalizeInputs(sites, upstreams)
	if err != nil {
		return nil, err
	}

	profileBySite := make(map[string]EasyProfileInput, len(profiles))
	for _, profile := range profiles {
		profileBySite[strings.TrimSpace(profile.SiteID)] = profile
	}

	httpData := easyRateLimitHTTPData{}
	artifacts := []ArtifactOutput{}
	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}
		profile, ok := profileBySite[site.ID]
		if !ok {
			profile = EasyProfileInput{SiteID: site.ID}
		}
		statusCode := easyCustomLimitStatusCode(profile.BadBehaviorStatusCodes)
		rules := normalizeCompilerCustomLimitRules(profile.CustomLimitRules)
		for index, rule := range rules {
			httpData.Entries = append(httpData.Entries, easyRateLimitHTTPEntry{
				ZoneName: easyCustomReqZoneName(site.ID, index),
				KeyVar:   siteRateLimitKeyVar(site.ID),
				Rate:     rule.Rate,
			})
		}

		upstream := upstreamByID[site.DefaultUpstreamID]
		locationData := easyLocationData{SiteID: site.ID, Rules: make([]easyLocationRuleData, 0, len(rules))}
		for index, rule := range rules {
			canonicalPath := customLimitCanonicalPath(rule.Path)
			locationData.Rules = append(locationData.Rules, easyLocationRuleData{
				LocationModifier:            customLimitLocationModifier(rule.Path),
				LocationPath:                customLimitLocationPath(rule.Path),
				DisableProxyInterceptErrors: strings.HasPrefix(canonicalPath, "/api/"),
				ZoneName:                    easyCustomReqZoneName(site.ID, index),
				Burst:                       customLimitBurst(rule.Rate),
				StatusCode:                  statusCode,
				PassHostHeader:              upstream.PassHostHeader,
				ProxyPassTarget:             "http://" + upstreamBlockName(site.ID, upstream.ID),
			})
		}
		locationContent, err := renderTemplate(filepath.Join(templatesRoot(), "easy", "locations.conf.tmpl"), locationData)
		if err != nil {
			return nil, fmt.Errorf("render easy locations template for %s: %w", site.ID, err)
		}
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/easy-locations/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			locationContent,
		))
	}

	httpContent, err := renderTemplate(filepath.Join(templatesRoot(), "conf.d", "easy-ratelimits.conf.tmpl"), httpData)
	if err != nil {
		return nil, fmt.Errorf("render easy rate limit http template: %w", err)
	}
	artifacts = append([]ArtifactOutput{newArtifact("nginx/conf.d/easy-ratelimits.conf", ArtifactKindNginxConfig, httpContent)}, artifacts...)
	return artifacts, nil
}

func normalizeCompilerCustomLimitRules(values []CustomRateLimitRuleInput) []CustomRateLimitRuleInput {
	if len(values) == 0 {
		return nil
	}
	items := make([]CustomRateLimitRuleInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		rate := strings.ToLower(strings.TrimSpace(value.Rate))
		if path == "" || rate == "" {
			continue
		}
		expanded := expandCustomLimitRuleAliases(path, rate)
		for _, rule := range expanded {
			if isReservedBaseLocationPath(rule.Path) {
				continue
			}
			key := strings.ToLower(rule.Path) + "\x00" + rule.Rate
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, rule)
		}
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
			return items[i].Rate < items[j].Rate
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func customLimitPriority(path string) int {
	trimmed := strings.TrimSpace(path)
	if hasComplexWildcard(trimmed) {
		return 1
	}
	if strings.HasSuffix(trimmed, "*") || strings.HasSuffix(trimmed, "/") {
		return 2
	}
	return 3
}

func customLimitLocationModifier(path string) string {
	trimmed := strings.TrimSpace(path)
	if hasComplexWildcard(trimmed) {
		return "~*"
	}
	if strings.HasSuffix(trimmed, "*") || strings.HasSuffix(trimmed, "/") {
		return "^~"
	}
	return "="
}

func customLimitLocationPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if hasComplexWildcard(trimmed) {
		return wildcardPathToRegex(trimmed)
	}
	if strings.HasSuffix(trimmed, "*") {
		trimmed = strings.TrimSuffix(trimmed, "*")
		if trimmed == "" {
			return "/"
		}
	}
	return trimmed
}

func customLimitBurst(rate string) int {
	if parsed := parseRatePerSecond(rate); parsed > 0 {
		return parsed
	}
	return 1
}

func easyCustomReqZoneName(siteID string, index int) string {
	// Keep zone names versioned to avoid nginx reload failures when zone key changes
	// between releases (shared-memory zones are immutable across reloads).
	return fmt.Sprintf("easy_%s_req_v2_%d", slugSiteID(siteID), index)
}

func isReservedBaseLocationPath(path string) bool {
	canonical := customLimitCanonicalPath(path)
	return canonical == "/" || canonical == "/api/"
}

func customLimitCanonicalPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasSuffix(trimmed, "*") && !hasComplexWildcard(trimmed) {
		trimmed = strings.TrimSuffix(trimmed, "*")
		if trimmed == "" {
			return "/"
		}
	}
	return trimmed
}

func hasComplexWildcard(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	star := strings.Count(path, "*")
	qm := strings.Count(path, "?")
	if qm > 0 {
		return true
	}
	if star == 0 {
		return false
	}
	// Simple trailing * is already supported via prefix location.
	if star == 1 && strings.HasSuffix(path, "*") {
		return false
	}
	return true
}

func wildcardPathToRegex(path string) string {
	quoted := regexp.QuoteMeta(strings.TrimSpace(path))
	quoted = strings.ReplaceAll(quoted, `\*`, `.*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `.`)
	return "^" + quoted
}

func expandCustomLimitRuleAliases(path, rate string) []CustomRateLimitRuleInput {
	base := CustomRateLimitRuleInput{Path: path, Rate: rate}
	normalizedPath := strings.TrimSpace(path)
	if !sentryEnvelopePathRE.MatchString(normalizedPath) {
		return []CustomRateLimitRuleInput{base}
	}
	// Sentry envelope endpoint uses project id in URL (/api/{project_id}/envelope/).
	// Keep compatibility for common project ids to avoid accidental throttling by global rules.
	return []CustomRateLimitRuleInput{
		{Path: "/api/*/envelope/", Rate: rate},
		base,
	}
}

func easyCustomLimitStatusCode(values []int) int {
	// Preserve existing ban/escalation behavior when 429 is present.
	for _, value := range values {
		if value == 429 {
			return 429
		}
	}
	// Otherwise allow any explicitly configured valid status code.
	for _, value := range values {
		if (value >= 100 && value <= 599) || value == 444 {
			return value
		}
	}
	return 429
}
