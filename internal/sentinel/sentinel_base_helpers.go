package sentinel

import (
	"errors"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func applyScore(st *State, cfg Config, ip string, site string, when time.Time, weight float64, reason string) bool {
	if st == nil || strings.TrimSpace(ip) == "" || weight <= 0 {
		return false
	}
	key := MakeScoreKey(site, ip)
	rec := st.IPs[key]
	score := rec.Score
	lastUpdated, _ := parseTime(rec.LastUpdated)
	if !lastUpdated.IsZero() {
		delta := when.Sub(lastUpdated).Seconds()
		if delta > 0 {
			score = score * math.Exp(-cfg.DecayLambda*delta)
		}
	}
	score += weight
	expires := when.Add(time.Duration(cfg.HoldSeconds) * time.Second).UTC()
	if rec.TopSignals == nil {
		rec.TopSignals = map[string]float64{}
	}
	if strings.TrimSpace(reason) != "" {
		rec.TopSignals[reason] += weight
	}
	rec.Score = score
	rec.RiskScore = clamp(score, 0, 100)
	rec.TrustScore = clamp(100-rec.RiskScore, 0, 100)
	rec.LastSeen = when.UTC().Format(time.RFC3339)
	if strings.TrimSpace(rec.FirstSeen) == "" {
		rec.FirstSeen = when.UTC().Format(time.RFC3339)
	}
	rec.LastUpdated = when.UTC().Format(time.RFC3339)
	rec.ExpiresAt = expires.Format(time.RFC3339)
	rec.ReasonCodes = topSignalKeys(rec.TopSignals, 5)
	rec.ExplainSummary, _, rec.Recommendations = buildExplainability(rec.TopSignals, rec.Stage)
	st.IPs[key] = rec
	return true
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty")
	}
	return time.Parse(time.RFC3339, value)
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func normalizeSiteID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "-" {
		return ""
	}
	return value
}

func normalizeSiteList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		site := normalizeSiteID(value)
		if site == "" {
			continue
		}
		seen[site] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isSiteModelEnabled(cfg Config, site string) bool {
	if len(cfg.EnabledSiteIDs) == 0 {
		return true
	}
	site = normalizeSiteID(site)
	if site == "" {
		return false
	}
	for _, allowed := range cfg.EnabledSiteIDs {
		if site == allowed {
			return true
		}
	}
	return false
}

// MakeScoreKey produces a stable key for (site, ip).
func MakeScoreKey(site, ip string) string {
	site = normalizeSiteID(site)
	if site == "" {
		site = globalSiteMarker
	}
	return site + "|" + strings.TrimSpace(ip)
}

func parseScoreKey(key string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(key), "|", 2)
	if len(parts) == 2 {
		return normalizeSiteID(parts[0]), strings.TrimSpace(parts[1])
	}
	return globalSiteMarker, strings.TrimSpace(key)
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func maxInt(left, right int) int {
	if left >= right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left <= right {
		return left
	}
	return right
}
