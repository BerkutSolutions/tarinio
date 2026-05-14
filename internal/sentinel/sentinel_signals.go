package sentinel

import (
	"sort"
	"strings"
)

func normalizePath(path string) string {
	p := strings.TrimSpace(strings.ToLower(path))
	if p == "" || p == "-" {
		return ""
	}
	if idx := strings.Index(p, "?"); idx >= 0 {
		p = p[:idx]
	}
	return p
}

func isScannerPath(path string) bool {
	p := normalizePath(path)
	for _, prefix := range scannerPathPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func matchedScannerPrefix(path string) string {
	p := normalizePath(path)
	for _, prefix := range scannerPathPrefixes {
		if strings.HasPrefix(p, prefix) {
			return prefix
		}
	}
	return ""
}

func isSuspiciousUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" || ua == "-" {
		return true
	}
	return strings.Contains(ua, "sqlmap") ||
		strings.Contains(ua, "nikto") ||
		strings.Contains(ua, "nmap") ||
		strings.Contains(ua, "masscan") ||
		strings.Contains(ua, "python-requests") ||
		strings.Contains(ua, "go-http-client")
}

func topSignalKeys(items map[string]float64, limit int) []string {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	type pair struct {
		Key   string
		Value float64
	}
	all := make([]pair, 0, len(items))
	for key, value := range items {
		if value <= 0 {
			continue
		}
		all = append(all, pair{Key: key, Value: value})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Value == all[j].Value {
			return all[i].Key < all[j].Key
		}
		return all[i].Value > all[j].Value
	})
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]string, 0, len(all))
	for _, item := range all {
		out = append(out, item.Key)
	}
	return out
}

func topSignalMap(items map[string]float64, limit int) map[string]float64 {
	keys := topSignalKeys(items, limit)
	if len(keys) == 0 {
		return nil
	}
	out := make(map[string]float64, len(keys))
	for _, key := range keys {
		out[key] = items[key]
	}
	return out
}

func buildExplainability(items map[string]float64, action string) (string, []SignalReasonDetail, []string) {
	keys := topSignalKeys(items, 5)
	if len(keys) == 0 {
		return "", nil, nil
	}
	details := make([]SignalReasonDetail, 0, len(keys))
	recommendations := make([]string, 0, len(keys)+1)
	seenRecommendations := map[string]struct{}{}
	for _, key := range keys {
		explanation := signalExplanation(key)
		details = append(details, SignalReasonDetail{
			Code:        key,
			Weight:      items[key],
			Explanation: explanation,
		})
		for _, item := range signalRecommendations(key) {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, ok := seenRecommendations[trimmed]; ok {
				continue
			}
			seenRecommendations[trimmed] = struct{}{}
			recommendations = append(recommendations, trimmed)
		}
	}
	if actionHint := actionRecommendation(action); actionHint != "" {
		if _, ok := seenRecommendations[actionHint]; !ok {
			recommendations = append([]string{actionHint}, recommendations...)
		}
	}
	summary := "Primary drivers: " + strings.Join(keys, ", ")
	return summary, details, recommendations
}

func signalExplanation(code string) string {
	switch strings.TrimSpace(code) {
	case "signal_rps":
		return "Request rate exceeded baseline for this source."
	case "signal_404_ratio":
		return "High share of 404 responses suggests probing of unknown paths."
	case "signal_unique_paths":
		return "Large number of unique paths suggests scan-like behavior."
	case "signal_scanner_paths":
		return "Scanner signatures were detected in requested paths."
	case "signal_ua_risk":
		return "Suspicious or automation-like user-agent was observed."
	case "signal_blocked_ratio":
		return "High ratio of blocked responses indicates persistent abusive traffic."
	case "signal_cross_site_spread":
		return "Same source touched multiple protected sites in a short window."
	case "signal_ml_risk":
		return "CPU-only logistic classifier raised anomaly probability for this source."
	case "signal_emergency_botnet":
		return "Distributed emergency pattern detected (many unique sources per second)."
	case "signal_emergency_single":
		return "Single-source emergency flood pattern detected."
	case "signal_emergency_drop":
		return "Immediate drop was applied due to emergency flood threshold."
	default:
		return "Contributed to risk score by anomaly model."
	}
}

func signalRecommendations(code string) []string {
	switch strings.TrimSpace(code) {
	case "signal_rps", "signal_emergency_single", "signal_emergency_botnet":
		return []string{
			"Review upstream capacity and edge rate limits for the affected site.",
			"Verify this source against allowlists before escalating to permanent blocking.",
		}
	case "signal_404_ratio", "signal_unique_paths", "signal_scanner_paths":
		return []string{
			"Add or tighten path-based rules for high-noise scanner endpoints.",
			"Enable stricter challenge policy for authentication and admin paths.",
		}
	case "signal_ua_risk":
		return []string{
			"Enable stronger anti-bot challenge for suspicious user-agents.",
		}
	case "signal_blocked_ratio":
		return []string{
			"Audit false positives and tune thresholds if legitimate clients are impacted.",
		}
	case "signal_cross_site_spread":
		return []string{
			"Review source activity across sites and enforce tenant/site segmentation controls.",
		}
	case "signal_ml_risk":
		return []string{
			"Validate ML-driven anomaly against deterministic signals before promoting to permanent ban.",
		}
	default:
		return nil
	}
}

func actionRecommendation(action string) string {
	switch strings.TrimSpace(action) {
	case "watch":
		return "Keep traffic in watch mode and continue monitoring signal drift."
	case "throttle":
		return "Throttle is active: monitor latency/error budget for legitimate users."
	case "drop", "temp_ban":
		return "Hard mitigation is active: validate business impact and prepare incident notes."
	default:
		return ""
	}
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
