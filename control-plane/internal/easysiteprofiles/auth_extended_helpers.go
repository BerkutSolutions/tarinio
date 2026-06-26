package easysiteprofiles

import (
	"sort"
	"strings"
)

var allowedAuthModes = []string{
	AuthModeBasic,
	AuthModeServiceToken,
	AuthModeBasicOrToken,
}

var allowedAuthOrders = []string{
	AuthOrderAuthFirst,
	AuthOrderAntibotFirst,
}

func normalizeAuthMode(value string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowedAuthModes {
		if mode == candidate {
			return mode
		}
	}
	return AuthModeBasic
}

func normalizeAuthOrder(value string) string {
	order := strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowedAuthOrders {
		if order == candidate {
			return order
		}
	}
	return AuthOrderAuthFirst
}

func normalizeAuthExclusionRules(values []SecurityAuthExclusionRule) []SecurityAuthExclusionRule {
	if len(values) == 0 {
		return nil
	}
	items := make([]SecurityAuthExclusionRule, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		methods := normalizeUpperList(value.Methods)
		if len(methods) == 0 {
			methods = []string{"*"}
		}
		if slicesContains(methods, "*") {
			methods = []string{"*"}
		}
		if path == "" {
			continue
		}
		key := strings.ToLower(path) + "\x00" + strings.Join(methods, ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, SecurityAuthExclusionRule{
			Path:    path,
			Methods: methods,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Path == items[j].Path {
			return strings.Join(items[i].Methods, ",") < strings.Join(items[j].Methods, ",")
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func normalizeAuthServiceTokens(values []SecurityAuthServiceToken) []SecurityAuthServiceToken {
	if len(values) == 0 {
		return nil
	}
	out := make([]SecurityAuthServiceToken, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		serviceName := strings.TrimSpace(value.ServiceName)
		if serviceName == "" {
			continue
		}
		key := strings.ToLower(serviceName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, SecurityAuthServiceToken{
			ServiceName: serviceName,
			Token:       strings.TrimSpace(value.Token),
			Enabled:     value.Enabled,
			LastUsedAt:  strings.TrimSpace(value.LastUsedAt),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ServiceName) < strings.ToLower(out[j].ServiceName)
	})
	return out
}

func hasEnabledServiceToken(values []SecurityAuthServiceToken) bool {
	for _, value := range values {
		if !value.Enabled {
			continue
		}
		if strings.TrimSpace(value.Token) == "" {
			continue
		}
		return true
	}
	return false
}

func slicesContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
