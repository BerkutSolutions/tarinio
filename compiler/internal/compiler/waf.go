package compiler

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

type modSecurityBaseData struct{}

type modSecuritySiteData struct {
	EngineMode   string
	IncludeCRS   bool
	OverridesRef string
}

type crsSetupData struct{}

type crsOverrideData struct {
	RuleIncludes []string
}

// RenderWAFArtifacts produces deterministic ModSecurity and CRS artifacts for
// the MVP WAFPolicy compiler mapping.
func RenderWAFArtifacts(sites []SiteInput, policies []WAFPolicyInput) ([]ArtifactOutput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool {
		return sortedSites[i].ID < sortedSites[j].ID
	})

	policyBySite := make(map[string]WAFPolicyInput, len(policies))
	for _, policy := range policies {
		if policy.ID == "" {
			return nil, errors.New("waf policy id is required")
		}
		if policy.SiteID == "" {
			return nil, fmt.Errorf("waf policy %s site id is required", policy.ID)
		}
		if policy.Enabled {
			switch policy.Mode {
			case WAFModeDetection, WAFModePrevention:
			default:
				return nil, fmt.Errorf("waf policy %s mode must be detection or prevention", policy.ID)
			}
		}
		includes, err := normalizeTrustedModSecurityIncludes(policy.CustomRuleIncludes)
		if err != nil {
			return nil, fmt.Errorf("waf policy %s custom_rule_includes: %w", policy.ID, err)
		}
		policy.CustomRuleIncludes = includes
		policyBySite[policy.SiteID] = policy
	}

	baseContent, err := renderTemplate("templates/modsecurity/modsecurity.conf.tmpl", modSecurityBaseData{})
	if err != nil {
		return nil, fmt.Errorf("render modsecurity base template: %w", err)
	}

	crsSetupContent, err := renderTemplate("templates/modsecurity/crs-setup.conf.tmpl", crsSetupData{})
	if err != nil {
		return nil, fmt.Errorf("render crs setup template: %w", err)
	}

	artifacts := []ArtifactOutput{
		newArtifact("modsecurity/modsecurity.conf", ArtifactKindModSecurity, baseContent),
		newArtifact("modsecurity/crs-setup.conf", ArtifactKindCRSConfig, crsSetupContent),
	}

	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}

		policy, ok := policyBySite[site.ID]
		if !ok {
			policy = WAFPolicyInput{
				SiteID:     site.ID,
				Enabled:    false,
				Mode:       "",
				CRSEnabled: false,
			}
		}
		if !policy.Enabled {
			policy.Mode = ""
		}

		siteContent, err := renderTemplate("templates/modsecurity/sites/site.conf.tmpl", modSecuritySiteData{
			EngineMode:   engineMode(policy.Mode),
			IncludeCRS:   policy.Enabled && policy.CRSEnabled,
			OverridesRef: fmt.Sprintf("/etc/waf/modsecurity/crs-overrides/%s.conf", site.ID),
		})
		if err != nil {
			return nil, fmt.Errorf("render modsecurity site template for %s: %w", site.ID, err)
		}

		overridesContent, err := renderTemplate("templates/modsecurity/crs-overrides/site-overrides.conf.tmpl", crsOverrideData{
			RuleIncludes: policy.CustomRuleIncludes,
		})
		if err != nil {
			return nil, fmt.Errorf("render crs overrides template for %s: %w", site.ID, err)
		}

		artifacts = append(artifacts,
			newArtifact(fmt.Sprintf("modsecurity/sites/%s.conf", site.ID), ArtifactKindModSecurity, siteContent),
			newArtifact(fmt.Sprintf("modsecurity/crs-overrides/%s.conf", site.ID), ArtifactKindCRSConfig, overridesContent),
		)
	}

	return artifacts, nil
}

func normalizeTrustedModSecurityIncludes(values []string) ([]string, error) {
	trusted := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "/rules/") {
			value = strings.TrimPrefix(value, "/")
		}
		if !strings.HasPrefix(value, "rules/") || strings.ContainsAny(value, "\r\n\"';{}") {
			return nil, errors.New("must be a relative rules/*.conf path")
		}
		clean := path.Clean(value)
		if clean == "." || clean == "rules" || strings.HasPrefix(clean, "../") || !strings.HasSuffix(clean, ".conf") {
			return nil, errors.New("must be a relative rules/*.conf path")
		}
		trusted = append(trusted, "/etc/waf/modsecurity/"+clean)
	}
	return sortedUnique(trusted), nil
}

func engineMode(mode WAFMode) string {
	if mode == WAFModePrevention {
		return "On"
	}
	if mode == "" {
		return "Off"
	}
	if mode == WAFModeDetection {
		return "DetectionOnly"
	}
	return "DetectionOnly"
}
