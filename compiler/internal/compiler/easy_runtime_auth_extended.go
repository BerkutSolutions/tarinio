package compiler

import (
	"sort"
	"strconv"
	"strings"
)

const (
	authModeBasic        = "basic"
	authModeServiceToken = "service_token"
	authModeBasicOrToken = "basic_or_token"
	authOrderAuthFirst   = "auth_first"
)

type easyAuthExclusionRuleData struct {
	MatchPattern string
}

type easyAuthTokenRuleData struct {
	ServiceGuard string
	BearerGuard  string
}

func normalizeCompilerAuthMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case authModeServiceToken:
		return authModeServiceToken
	case authModeBasicOrToken:
		return authModeBasicOrToken
	default:
		return authModeBasic
	}
}

func normalizeCompilerAuthOrder(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == "antibot_first" {
		return "antibot_first"
	}
	return authOrderAuthFirst
}

func enabledAuthServiceTokens(values []ServiceAuthTokenInput) []ServiceAuthTokenInput {
	out := make([]ServiceAuthTokenInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		serviceName := strings.TrimSpace(item.ServiceName)
		token := strings.TrimSpace(item.Token)
		if serviceName == "" || token == "" || !item.Enabled {
			continue
		}
		key := strings.ToLower(serviceName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ServiceAuthTokenInput{
			ServiceName: serviceName,
			Token:       token,
			Enabled:     true,
			LastUsedAt:  strings.TrimSpace(item.LastUsedAt),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ServiceName) < strings.ToLower(out[j].ServiceName)
	})
	return out
}

func buildEasyAuthExclusionRuleData(values []AuthExclusionRuleInput) []easyAuthExclusionRuleData {
	if len(values) == 0 {
		return nil
	}
	items := make([]easyAuthExclusionRuleData, 0, len(values))
	for _, value := range values {
		methods := value.Methods
		if len(methods) == 0 {
			methods = []string{"*"}
		}
		matchMethods := ".*"
		normalized := sortedUniqueUpper(methods)
		if len(normalized) > 0 && !(len(normalized) == 1 && normalized[0] == "*") {
			matchMethods = "(?:" + strings.Join(normalized, "|") + ")"
		}
		items = append(items, easyAuthExclusionRuleData{
			MatchPattern: "^" + matchMethods + ":" + wildcardPathToRegex(strings.TrimSpace(value.Path)) + "$",
		})
	}
	return items
}

func buildEasyAuthTokenRuleData(values []ServiceAuthTokenInput) []easyAuthTokenRuleData {
	if len(values) == 0 {
		return nil
	}
	items := make([]easyAuthTokenRuleData, 0, len(values))
	for _, value := range values {
		serviceName := strconv.Quote(strings.TrimSpace(value.ServiceName))
		token := strconv.Quote(strings.TrimSpace(value.Token))
		token = strings.Trim(token, "\"")
		serviceName = strings.Trim(serviceName, "\"")
		items = append(items, easyAuthTokenRuleData{
			ServiceGuard: serviceName + "|" + token,
			BearerGuard:  serviceName + "|Bearer " + token,
		})
	}
	return items
}
