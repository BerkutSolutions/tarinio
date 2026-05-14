package sentinel

import (
	"math"
	"sort"
	"strings"
	"time"
)

func deriveSignalWeights(stats *ipStat) map[string]float64 {
	out := map[string]float64{}
	if stats == nil || stats.Count <= 0 {
		return out
	}
	total := float64(stats.Count)
	if stats.Count > 30 {
		out["signal_rps"] = float64(stats.Count-30) * 0.05
	}
	ratio404 := float64(stats.NotFound) / total
	if stats.Count >= 10 && ratio404 >= 0.4 {
		out["signal_404_ratio"] = ratio404 * 2.0
	}
	if len(stats.UniquePaths) >= 20 {
		out["signal_unique_paths"] = float64(len(stats.UniquePaths)-20) * 0.08
	}
	if stats.ScannerHits > 0 {
		out["signal_scanner_paths"] = float64(stats.ScannerHits) * 1.2
	}
	if stats.SuspiciousUAHits > 0 {
		out["signal_ua_risk"] = math.Min(2.5, float64(stats.SuspiciousUAHits)*0.6)
	}
	if len(stats.Sites) >= 2 {
		out["signal_cross_site_spread"] = math.Min(2.0, float64(len(stats.Sites)-1)*0.8)
	}
	blockedRatio := float64(stats.Blocked) / total
	if stats.Count >= 10 && blockedRatio >= 0.6 {
		out["signal_blocked_ratio"] = blockedRatio * 1.5
	}
	return out
}

// EffectiveEmergencyThresholds scales thresholds based on configured limits.
func EffectiveEmergencyThresholds(cfg Config) (int, int, int) {
	globalRPS := cfg.EmergencyRPS
	uniqueIPs := cfg.EmergencyUniqueIPs
	perIPRPS := cfg.EmergencyPerIPRPS

	if cfg.RatePerSecond > 0 {
		globalRPS = maxInt(globalRPS, minInt(cfg.RatePerSecond*3, globalRPS*3))
		perIPRPS = maxInt(perIPRPS, minInt(cfg.RatePerSecond*2, perIPRPS*3))
	}
	if cfg.EnforceL7RateLimit && cfg.L7RequestsPerSecond > 0 {
		globalRPS = maxInt(globalRPS, minInt(cfg.L7RequestsPerSecond*4, globalRPS*3))
		perIPRPS = maxInt(perIPRPS, minInt(cfg.L7RequestsPerSecond*2, perIPRPS*3))
	}
	if cfg.ConnLimit > 0 {
		scaledUnique := minInt(50, maxInt(20, cfg.ConnLimit/12))
		uniqueIPs = maxInt(uniqueIPs, scaledUnique)
	}
	return globalRPS, uniqueIPs, perIPRPS
}

func scoreWeight(cfg Config, status int) float64 {
	switch status {
	case 429:
		return cfg.Weight429
	case 403:
		return cfg.Weight403
	case 444:
		return cfg.Weight444
	default:
		return 0
	}
}

func selectStage(cfg Config, score float64) string {
	if score >= cfg.TempBanThreshold {
		return "temp_ban"
	}
	if score >= cfg.DropThreshold {
		return "drop"
	}
	if score >= cfg.ThrottleThreshold {
		return "throttle"
	}
	if score >= cfg.WatchThreshold {
		return "watch"
	}
	return ""
}

func decideAction(cfg Config, score float64) string {
	return selectStage(cfg, score)
}

func requiresMultiSignalGate(action string, minSignals int) bool {
	if minSignals <= 1 {
		return false
	}
	return action == "drop" || action == "temp_ban"
}

func countStrongSignals(topSignals map[string]float64) int {
	if len(topSignals) == 0 {
		return 0
	}
	count := 0
	for _, value := range topSignals {
		if value >= 0.3 {
			count++
		}
	}
	return count
}

func hasEmergencySignal(topSignals map[string]float64) bool {
	if len(topSignals) == 0 {
		return false
	}
	for signal := range topSignals {
		if strings.HasPrefix(signal, "signal_emergency_") {
			return true
		}
	}
	return false
}

func currentAction(rec Record) string {
	if strings.TrimSpace(rec.LastAction) != "" {
		return rec.LastAction
	}
	return strings.TrimSpace(rec.Stage)
}

func stageRank(stage string) int {
	switch strings.TrimSpace(stage) {
	case "watch":
		return 1
	case "throttle":
		return 2
	case "drop":
		return 3
	case "temp_ban":
		return 4
	default:
		return 0
	}
}

func isCooldownDeescalation(previous, next string, lastActionAt string, cooldownSeconds int, now time.Time) bool {
	if cooldownSeconds <= 0 {
		return false
	}
	if stageRank(next) >= stageRank(previous) {
		return false
	}
	last, err := parseTime(lastActionAt)
	if err != nil || last.IsZero() {
		return false
	}
	return now.Sub(last) < time.Duration(cooldownSeconds)*time.Second
}

func resolveActionTransition(rec *Record, previous, desired string, cfg Config, emergency bool) (string, bool) {
	if rec == nil {
		return previous, false
	}
	if strings.TrimSpace(desired) == strings.TrimSpace(previous) {
		rec.CandidateAction = ""
		rec.CandidateCount = 0
		return previous, false
	}
	requiredTicks := cfg.PromotionConsecutiveTicks
	if requiredTicks <= 0 {
		requiredTicks = 1
	}
	if rec.CandidateAction != desired {
		rec.CandidateAction = desired
		rec.CandidateCount = 1
	} else {
		rec.CandidateCount++
	}
	if emergency || rec.CandidateCount >= requiredTicks {
		rec.CandidateAction = ""
		rec.CandidateCount = 0
		return desired, true
	}
	return previous, false
}

func actionExpiry(now time.Time, cfg Config, action string) time.Time {
	hold := cfg.HoldSeconds
	if hold <= 0 {
		hold = 60
	}
	switch action {
	case "temp_ban":
		return now.Add(time.Duration(hold*6) * time.Second).UTC()
	case "drop", "throttle", "watch":
		return now.Add(time.Duration(hold) * time.Second).UTC()
	default:
		return now.Add(time.Duration(hold) * time.Second).UTC()
	}
}

func consumeActionBudget(st *State, cfg Config, now time.Time) bool {
	if st == nil {
		return false
	}
	limit := cfg.MaxActionsPerMinute
	if limit <= 0 {
		limit = 300
	}
	windowStart, err := parseTime(st.ActionWindowStartedAt)
	if err != nil || windowStart.IsZero() || now.Sub(windowStart) >= time.Minute {
		st.ActionWindowStartedAt = now.Format(time.RFC3339)
		st.ActionsInWindow = 0
	}
	if st.ActionsInWindow >= limit {
		return false
	}
	st.ActionsInWindow++
	return true
}

func isExpiredByInactivity(rec Record, now time.Time, inactiveTTLSeconds int) bool {
	if inactiveTTLSeconds <= 0 {
		return false
	}
	lastSeen, err := parseTime(rec.LastSeen)
	if err != nil || lastSeen.IsZero() {
		return false
	}
	return now.Sub(lastSeen) > time.Duration(inactiveTTLSeconds)*time.Second
}

func evictOverflow(st *State, cfg Config, now time.Time) bool {
	if st == nil || len(st.IPs) == 0 {
		return false
	}
	limit := cfg.MaxActiveIPs
	if limit <= 0 || len(st.IPs) <= limit {
		return false
	}
	type item struct {
		Key      string
		Score    float64
		LastSeen time.Time
	}
	all := make([]item, 0, len(st.IPs))
	for key, rec := range st.IPs {
		lastSeen, _ := parseTime(rec.LastSeen)
		if lastSeen.IsZero() {
			lastSeen = now
		}
		all = append(all, item{Key: key, Score: rec.Score, LastSeen: lastSeen})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Score == all[j].Score {
			return all[i].LastSeen.Before(all[j].LastSeen)
		}
		return all[i].Score < all[j].Score
	})
	toDrop := len(all) - limit
	if toDrop <= 0 {
		return false
	}
	for i := 0; i < toDrop; i++ {
		delete(st.IPs, all[i].Key)
	}
	return true
}

func applyImmediateDrop(st *State, cfg Config, ip string, when time.Time) bool {
	if st == nil || strings.TrimSpace(ip) == "" {
		return false
	}
	key := MakeScoreKey(globalSiteMarker, ip)
	rec := st.IPs[key]
	score := rec.Score
	if score < cfg.DropThreshold {
		score = cfg.DropThreshold
	}
	expires := when.Add(time.Duration(cfg.HoldSeconds) * time.Second).UTC()
	if rec.TopSignals == nil {
		rec.TopSignals = map[string]float64{}
	}
	rec.TopSignals["signal_emergency_drop"] += 1
	rec.Score = score
	rec.RiskScore = clamp(score, 0, 100)
	rec.TrustScore = clamp(100-rec.RiskScore, 0, 100)
	rec.LastSeen = when.UTC().Format(time.RFC3339)
	if strings.TrimSpace(rec.FirstSeen) == "" {
		rec.FirstSeen = when.UTC().Format(time.RFC3339)
	}
	rec.LastUpdated = when.UTC().Format(time.RFC3339)
	rec.Stage = "drop"
	rec.LastAction = "drop"
	rec.LastActionAt = when.UTC().Format(time.RFC3339)
	rec.ExpiresAt = expires.Format(time.RFC3339)
	rec.ReasonCodes = topSignalKeys(rec.TopSignals, 5)
	rec.ExplainSummary, _, rec.Recommendations = buildExplainability(rec.TopSignals, rec.Stage)
	st.IPs[key] = rec
	return true
}
