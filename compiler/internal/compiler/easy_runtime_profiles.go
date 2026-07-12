package compiler

import (
	"fmt"
	"slices"
	"strings"
)

func buildEasyProfileBySite(profiles []EasyProfileInput) (map[string]EasyProfileInput, error) {
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
		profile.BlacklistJA3 = sortedUnique(profile.BlacklistJA3)
		profile.BlacklistURI = normalizeBlacklistURIPatterns(profile.BlacklistURI)
		profile.BlacklistCountry = sortedUniqueUpper(profile.BlacklistCountry)
		profile.WhitelistCountry = sortedUniqueUpper(profile.WhitelistCountry)
		profile.MaxClientSize = strings.TrimSpace(profile.MaxClientSize)
		if profile.MaxClientSize == "" {
			profile.MaxClientSize = "100m"
		}
		profile.ReferrerPolicy = strings.TrimSpace(profile.ReferrerPolicy)
		profile.ContentSecurityPolicy = strings.TrimSpace(profile.ContentSecurityPolicy)
		if profile.HSTSMaxAgeSeconds <= 0 {
			profile.HSTSMaxAgeSeconds = 15552000
		}
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
			profile.AuthUsers = []ServiceAuthUserInput{{
				Username: profile.AuthBasicUser,
				Password: profile.AuthBasicPassword,
				Enabled:  true,
			}}
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
		profile.AntibotExclusionRules = normalizeCompilerAntibotExclusionRules(profile.AntibotExclusionRules)
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
		profile.ModSecurityExclusionRules = normalizeCompilerModSecurityExclusionRules(profile.ModSecurityExclusionRules)
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
		profile = applySecurityModePolicy(profile)

		profileBySite[siteID] = profile
	}
	return profileBySite, nil
}

func applySecurityModePolicy(profile EasyProfileInput) EasyProfileInput {
	mode := strings.ToLower(strings.TrimSpace(profile.SecurityMode))
	switch mode {
	case "transparent":
		profile.UseModSecurity = false
		profile.UseModSecurityCRSPlugins = false
		profile.UseModSecurityCustomConfiguration = false
		profile.ModSecurityExclusionRules = nil
		profile.UseAPIPositiveSecurity = false
		profile.UseLimitConn = false
		profile.UseLimitReq = false
		profile.CustomLimitRules = nil
		profile.UseBadBehavior = false
		profile.BadBehaviorStatusCodes = nil
		profile.BadBehaviorBanTimeSeconds = 0
		profile.BlacklistIP = nil
		profile.BlacklistUserAgent = nil
		profile.BlacklistURI = nil
		profile.BlacklistJA3 = nil
		profile.BlacklistCountry = nil
		profile.WhitelistCountry = nil
		profile.AntibotChallenge = "no"
		profile.AntibotScannerAutoBan = false
		profile.ChallengeEscalationEnabled = false
		profile.ChallengeEscalationMode = "no"
		profile.AntibotExclusionRules = nil
		profile.AntibotChallengeRules = nil
		profile.UseAuthBasic = false
	case "monitor":
		profile.UseLimitConn = false
		profile.UseLimitReq = false
		profile.CustomLimitRules = nil
		profile.UseBadBehavior = false
		profile.BadBehaviorStatusCodes = nil
		profile.BadBehaviorBanTimeSeconds = 0
		profile.BlacklistIP = nil
		profile.BlacklistUserAgent = nil
		profile.BlacklistURI = nil
		profile.BlacklistJA3 = nil
		profile.BlacklistCountry = nil
		profile.WhitelistCountry = nil
		profile.AntibotChallenge = "no"
		profile.AntibotScannerAutoBan = false
		profile.ChallengeEscalationEnabled = false
		profile.ChallengeEscalationMode = "no"
		profile.AntibotExclusionRules = nil
		profile.AntibotChallengeRules = nil
		profile.UseAuthBasic = false
	}
	return profile
}

func normalizeCompilerModSecurityExclusionRules(value []ModSecurityExclusionRuleInput) []ModSecurityExclusionRuleInput {
	items := make([]ModSecurityExclusionRuleInput, 0, len(value))
	seen := make(map[string]struct{}, len(value))
	for _, item := range value {
		rule := ModSecurityExclusionRuleInput{
			Path:        strings.TrimSpace(item.Path),
			PathPattern: strings.TrimSpace(item.PathPattern),
			Methods:     sortedUniqueUpper(item.Methods),
			Mode:        strings.ToLower(strings.TrimSpace(item.Mode)),
			RuleIDs:     normalizeCompilerPositiveInts(item.RuleIDs),
			Targets:     sortedUnique(item.Targets),
			Comment:     strings.TrimSpace(item.Comment),
		}
		if rule.Mode == "" {
			rule.Mode = "exact"
		}
		if len(rule.Methods) == 0 || slices.Contains(rule.Methods, "*") {
			rule.Methods = []string{"*"}
		}
		key := strings.ToLower(rule.Path) + "\x00" + rule.PathPattern + "\x00" + rule.Mode + "\x00" + strings.Join(rule.Methods, ",") + "\x00" + joinCompilerInts(rule.RuleIDs) + "\x00" + strings.Join(rule.Targets, ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, rule)
	}
	return items
}

func normalizeCompilerPositiveInts(values []int) []int {
	items := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, value)
	}
	slices.Sort(items)
	return slices.Compact(items)
}

func joinCompilerInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ",")
}
