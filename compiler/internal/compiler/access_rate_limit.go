package compiler

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type accessSiteData struct {
	TrustedProxyCIDRs []string
	AllowCIDRs        []string
	DenyCIDRs         []string
	DefaultAction     string
}

type rateLimitHTTPEntry struct {
	ZoneName        string
	ConnZoneName    string
	LimitReqKeyVar  string
	ConnKeyVar      string
	BaseKeyVar      string
	Rate            string
	LimitReqURLVar  string
	LimitReqPattern string
}

type rateLimitExceptionEntry struct {
	ExceptionVar string
	KeyVar       string
	AllowCIDRs   []string
}

type rateLimitHTTPData struct {
	Exceptions []rateLimitExceptionEntry
	Entries    []rateLimitHTTPEntry
}

type rateLimitSiteData struct {
	Enabled         bool
	ZoneName        string
	ConnZoneName    string
	LimitReqURLVar  string
	Burst           int
	StatusCode      int
	ConnectionLimit int
}

// RenderAccessRateLimitArtifacts produces deterministic nginx snippets for MVP
// access policies and basic rate limiting.
func RenderAccessRateLimitArtifacts(
	sites []SiteInput,
	accessPolicies []AccessPolicyInput,
	rateLimitPolicies []RateLimitPolicyInput,
) ([]ArtifactOutput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool {
		return sortedSites[i].ID < sortedSites[j].ID
	})

	accessBySite := make(map[string]AccessPolicyInput, len(accessPolicies))
	for _, policy := range accessPolicies {
		if policy.ID == "" {
			return nil, errors.New("access policy id is required")
		}
		if policy.SiteID == "" {
			return nil, fmt.Errorf("access policy %s site id is required", policy.ID)
		}
		policy.AllowCIDRs = sortedUnique(policy.AllowCIDRs)
		policy.DenyCIDRs = sortedUnique(policy.DenyCIDRs)
		policy.TrustedProxyCIDRs = sortedUnique(policy.TrustedProxyCIDRs)
		for _, cidr := range policy.TrustedProxyCIDRs {
			if err := validateNginxCIDROrIP(cidr, fmt.Sprintf("access policy %s trusted proxy CIDR", policy.ID)); err != nil {
				return nil, err
			}
		}
		policy.DefaultAction = normalizeDefaultAction(policy.DefaultAction)
		accessBySite[policy.SiteID] = policy
	}

	rateBySite := make(map[string]RateLimitPolicyInput, len(rateLimitPolicies))
	for _, policy := range rateLimitPolicies {
		if policy.ID == "" {
			return nil, errors.New("rate limit policy id is required")
		}
		if policy.SiteID == "" {
			return nil, fmt.Errorf("rate limit policy %s site id is required", policy.ID)
		}
		if policy.Enabled {
			if policy.Requests <= 0 {
				return nil, fmt.Errorf("rate limit policy %s requests must be positive", policy.ID)
			}
			if policy.WindowSeconds <= 0 {
				return nil, fmt.Errorf("rate limit policy %s window_seconds must be positive", policy.ID)
			}
			if policy.Burst < 0 {
				return nil, fmt.Errorf("rate limit policy %s burst must be zero or positive", policy.ID)
			}
			if policy.StatusCode == 0 {
				policy.StatusCode = 429
			}
		}
		rateBySite[policy.SiteID] = policy
	}

	httpData := rateLimitHTTPData{}
	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}
		accessPolicy, ok := accessBySite[site.ID]
		if isManagementSite(site) || !ok {
			accessPolicy = AccessPolicyInput{
				SiteID:        site.ID,
				DefaultAction: "allow",
			}
		}
		httpData.Exceptions = append(httpData.Exceptions, rateLimitExceptionEntry{
			ExceptionVar: siteExceptionVar(site.ID),
			KeyVar:       siteRateLimitKeyVar(site.ID),
			AllowCIDRs:   append([]string(nil), accessPolicy.AllowCIDRs...),
		})
		if policy, ok := rateBySite[site.ID]; ok && policy.Enabled {
			baseKeyVar := siteRateLimitKeyVar(site.ID)
			limitReqKeyVar := baseKeyVar
			limitReqURLVar := ""
			limitReqPattern := strings.TrimSpace(policy.LimitReqURL)
			if limitReqPattern != "" {
				limitReqURLVar = siteRateLimitReqKeyVar(site.ID)
				limitReqKeyVar = limitReqURLVar
			}
			httpData.Entries = append(httpData.Entries, rateLimitHTTPEntry{
				ZoneName:        reqZoneName(site.ID),
				ConnZoneName:    connZoneName(site.ID),
				LimitReqKeyVar:  limitReqKeyVar,
				ConnKeyVar:      baseKeyVar,
				BaseKeyVar:      baseKeyVar,
				Rate:            formatRate(policy.Requests, policy.WindowSeconds),
				LimitReqURLVar:  limitReqURLVar,
				LimitReqPattern: limitReqPattern,
			})
		}
	}

	httpContent, err := renderTemplate("templates/nginx/ratelimits/http.conf.tmpl", httpData)
	if err != nil {
		return nil, fmt.Errorf("render rate limit http template: %w", err)
	}

	// Собираем все уникальные TrustedProxyCIDRs из всех access policies
	// и выносим set_real_ip_from в http context (conf.d) чтобы real_ip модуль
	// работал в POST_READ_PHASE и корректно заменял $remote_addr до обработки deny.
	allTrustedCIDRs := map[string]struct{}{}
	for _, policy := range accessPolicies {
		for _, cidr := range policy.TrustedProxyCIDRs {
			if cidr != "" {
				allTrustedCIDRs[cidr] = struct{}{}
			}
		}
	}
	trustedCIDRSlice := sortedUnique(func() []string {
		out := make([]string, 0, len(allTrustedCIDRs))
		for cidr := range allTrustedCIDRs {
			out = append(out, cidr)
		}
		return out
	}())
	realIPContent, err := renderTemplate("templates/nginx/conf.d/real_ip.conf.tmpl", struct {
		TrustedProxyCIDRs []string
	}{TrustedProxyCIDRs: trustedCIDRSlice})
	if err != nil {
		return nil, fmt.Errorf("render real_ip template: %w", err)
	}

	artifacts := []ArtifactOutput{
		newArtifact("nginx/conf.d/ratelimits.conf", ArtifactKindNginxConfig, httpContent),
		newArtifact("nginx/conf.d/real_ip.conf", ArtifactKindNginxConfig, realIPContent),
	}

	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}

		accessPolicy, ok := accessBySite[site.ID]
		if isManagementSite(site) || !ok {
			accessPolicy = AccessPolicyInput{
				SiteID:        site.ID,
				DefaultAction: "allow",
			}
		}
		// transparent/monitor mode: disable all active blocking — deny rules must not apply.
		mode := strings.ToLower(strings.TrimSpace(accessPolicy.SecurityMode))
		if mode == "transparent" || mode == "monitor" {
			accessPolicy.DenyCIDRs = nil
			accessPolicy.AllowCIDRs = nil
			accessPolicy.DefaultAction = "allow"
		} else if shouldDefaultDenySite(site.ID, accessPolicy) {
			accessPolicy.DefaultAction = "deny"
		}

		accessContent, err := renderTemplate("templates/nginx/access/site.conf.tmpl", accessSiteData{
			TrustedProxyCIDRs: accessPolicy.TrustedProxyCIDRs,
			AllowCIDRs:        accessPolicy.AllowCIDRs,
			DenyCIDRs:         accessPolicy.DenyCIDRs,
			DefaultAction:     accessPolicy.DefaultAction,
		})
		if err != nil {
			return nil, fmt.Errorf("render access template for %s: %w", site.ID, err)
		}

		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/access/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			accessContent,
		))

		ratePolicy, ok := rateBySite[site.ID]
		if !ok || !ratePolicy.Enabled {
			ratePolicy = RateLimitPolicyInput{
				SiteID:  site.ID,
				Enabled: false,
			}
		}

		rateContent, err := renderTemplate("templates/nginx/ratelimits/site.conf.tmpl", rateLimitSiteData{
			Enabled:         ratePolicy.Enabled,
			ZoneName:        reqZoneName(site.ID),
			ConnZoneName:    connZoneName(site.ID),
			Burst:           ratePolicy.Burst,
			StatusCode:      effectiveStatusCode(ratePolicy.StatusCode),
			ConnectionLimit: connectionLimit(ratePolicy),
		})
		if err != nil {
			return nil, fmt.Errorf("render rate limit site template for %s: %w", site.ID, err)
		}

		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/ratelimits/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			rateContent,
		))
	}

	return artifacts, nil
}

func normalizeDefaultAction(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "deny":
		return "deny"
	default:
		return "allow"
	}
}

func shouldDefaultDenySite(siteID string, policy AccessPolicyInput) bool {
	if !isManagementSiteID(siteID) {
		return false
	}
	if normalizeDefaultAction(policy.DefaultAction) == "deny" {
		return true
	}
	return len(policy.AllowCIDRs) > 0
}

func reqZoneName(siteID string) string {
	return fmt.Sprintf("site_%s_req", siteID)
}

func connZoneName(siteID string) string {
	return fmt.Sprintf("site_%s_conn", siteID)
}

func siteExceptionVar(siteID string) string {
	return fmt.Sprintf("waf_allow_bypass_%s", slugSiteID(siteID))
}

func siteRateLimitKeyVar(siteID string) string {
	return fmt.Sprintf("waf_rate_limit_key_%s", slugSiteID(siteID))
}

func siteRateLimitReqKeyVar(siteID string) string {
	return fmt.Sprintf("waf_rate_limit_req_key_%s", slugSiteID(siteID))
}

func formatRate(requests, windowSeconds int) string {
	if windowSeconds == 1 {
		return fmt.Sprintf("%dr/s", requests)
	}
	if windowSeconds%60 == 0 {
		return fmt.Sprintf("%dr/m", requests/(windowSeconds/60))
	}
	return fmt.Sprintf("%dr/%ds", requests, windowSeconds)
}

func effectiveStatusCode(value int) int {
	if value == 0 {
		return 429
	}
	return value
}

func connectionLimit(policy RateLimitPolicyInput) int {
	if policy.Burst > 0 {
		return policy.Burst
	}
	if policy.Requests > 0 {
		return policy.Requests
	}
	return 1
}
