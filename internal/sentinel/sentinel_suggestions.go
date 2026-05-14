package sentinel

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func buildRuleSuggestions(cfg Config, scannerStats map[string]*scannerPathStat, previous []RuleSuggestion, now time.Time) []RuleSuggestion {
	minHits := cfg.SuggestMinHits
	if minHits <= 0 {
		minHits = 20
	}
	minUniqueIPs := cfg.SuggestMinUniqueIPs
	if minUniqueIPs <= 0 {
		minUniqueIPs = 5
	}
	shadowPromoteHits := cfg.SuggestShadowPromoteHits
	if shadowPromoteHits <= 0 {
		shadowPromoteHits = minHits * 2
	}
	temporaryPromoteHits := cfg.SuggestTemporaryPromoteHits
	if temporaryPromoteHits <= 0 {
		temporaryPromoteHits = minHits * 4
	}
	permanentPromoteHits := cfg.SuggestPermanentPromoteHits
	if permanentPromoteHits <= 0 {
		permanentPromoteHits = minHits * 8
	}
	shadowMaxFPRate := cfg.SuggestShadowMaxFPRate
	if shadowMaxFPRate <= 0 {
		shadowMaxFPRate = 0.02
	}
	temporaryHoldSeconds := cfg.SuggestTemporaryHoldSeconds
	if temporaryHoldSeconds <= 0 {
		temporaryHoldSeconds = 14400
	}
	suggestInactiveTTLSeconds := cfg.SuggestInactiveTTLSeconds
	if suggestInactiveTTLSeconds <= 0 {
		suggestInactiveTTLSeconds = 86400
	}

	prevByPath := map[string]RuleSuggestion{}
	for _, item := range previous {
		path := strings.TrimSpace(item.PathPrefix)
		if path == "" {
			continue
		}
		prevByPath[path] = item
	}
	out := make([]RuleSuggestion, 0, maxInt(len(scannerStats), len(previous)))
	processed := map[string]struct{}{}
	for path, stat := range scannerStats {
		if stat == nil {
			continue
		}
		processed[path] = struct{}{}
		unique := len(stat.IPs)
		if stat.Hits < minHits || unique < minUniqueIPs {
			continue
		}
		prev := prevByPath[path]
		status := normalizeSuggestionStatus(prev.Status)
		shadowHits := prev.ShadowHits
		shadowFP := prev.ShadowFP
		if status == "shadow" || status == "temporary" || status == "permanent" {
			shadowHits += stat.Hits
			shadowFP += stat.SuccessHits
		}
		firstSeen := stat.FirstSeen
		if existing, err := parseTime(prev.FirstSeen); err == nil && !existing.IsZero() && (firstSeen.IsZero() || existing.Before(firstSeen)) {
			firstSeen = existing
		}
		lastSeen := stat.LastSeen
		if existing, err := parseTime(prev.LastSeen); err == nil && !existing.IsZero() && existing.After(lastSeen) {
			lastSeen = existing
		}
		fpRate := 0.0
		if shadowHits > 0 {
			fpRate = float64(shadowFP) / float64(shadowHits)
		}
		temporaryUntil, _ := parseTime(prev.TemporaryUntil)
		promotionReason := strings.TrimSpace(prev.PromotionReason)
		switch status {
		case "suggested":
			if stat.Hits >= shadowPromoteHits && unique >= minUniqueIPs {
				status = "shadow"
				promotionReason = "auto_promote_shadow"
			}
		case "shadow":
			if fpRate > shadowMaxFPRate*2 {
				status = "suggested"
				promotionReason = "rollback_high_shadow_fp"
			} else if shadowHits >= temporaryPromoteHits && fpRate <= shadowMaxFPRate && unique >= minUniqueIPs {
				status = "temporary"
				temporaryUntil = now.Add(time.Duration(temporaryHoldSeconds) * time.Second).UTC()
				promotionReason = "auto_promote_temporary"
			}
		case "temporary":
			if fpRate > shadowMaxFPRate*2 {
				status = "suggested"
				temporaryUntil = time.Time{}
				promotionReason = "rollback_high_shadow_fp"
			} else if !temporaryUntil.IsZero() && now.After(temporaryUntil) {
				lifetimeOK := cfg.SuggestPermanentMinLifetime <= 0 || (!firstSeen.IsZero() && now.Sub(firstSeen) >= cfg.SuggestPermanentMinLifetime)
				if shadowHits >= permanentPromoteHits && fpRate <= shadowMaxFPRate*0.5 && lifetimeOK {
					status = "permanent"
					promotionReason = "candidate_permanent_review_required"
				} else {
					status = "shadow"
					promotionReason = "temporary_expired_return_shadow"
				}
				temporaryUntil = time.Time{}
			}
		case "permanent":
			if fpRate > shadowMaxFPRate*2 {
				status = "shadow"
				promotionReason = "rollback_permanent_high_fp"
			}
		}
		s := RuleSuggestion{
			ID:              "path-" + strings.TrimLeft(strings.ReplaceAll(path, "/", "-"), "-"),
			PathPrefix:      path,
			Status:          status,
			Hits:            stat.Hits,
			UniqueIPs:       unique,
			WouldBlock:      stat.Hits,
			ShadowHits:      shadowHits,
			ShadowFP:        shadowFP,
			PromotionReason: promotionReason,
			Source:          "tarinio-sentinel",
			GeneratedAt:     now.Format(time.RFC3339),
		}
		if shadowHits > 0 {
			s.ShadowRate = strconv.FormatFloat(fpRate, 'f', 4, 64)
		}
		if !firstSeen.IsZero() {
			s.FirstSeen = firstSeen.Format(time.RFC3339)
		}
		if !lastSeen.IsZero() {
			s.LastSeen = lastSeen.Format(time.RFC3339)
		}
		if !temporaryUntil.IsZero() {
			s.TemporaryUntil = temporaryUntil.Format(time.RFC3339)
		}
		out = append(out, s)
	}
	for path, prev := range prevByPath {
		if _, ok := processed[path]; ok {
			continue
		}
		if isSuggestionExpired(prev, now, suggestInactiveTTLSeconds) {
			continue
		}
		copyItem := prev
		if strings.TrimSpace(copyItem.Status) == "" {
			copyItem.Status = "suggested"
		}
		out = append(out, copyItem)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hits == out[j].Hits {
			return out[i].PathPrefix < out[j].PathPrefix
		}
		return out[i].Hits > out[j].Hits
	})
	if cfg.MaxPublishedEntries > 0 && len(out) > cfg.MaxPublishedEntries {
		out = out[:cfg.MaxPublishedEntries]
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PathPrefix < out[j].PathPrefix
	})
	return out
}

func normalizeSuggestionStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "shadow", "temporary", "permanent":
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return "suggested"
	}
}

func isSuggestionExpired(item RuleSuggestion, now time.Time, ttlSeconds int) bool {
	if ttlSeconds <= 0 {
		return false
	}
	lastSeen, err := parseTime(item.LastSeen)
	if err != nil || lastSeen.IsZero() {
		return false
	}
	return now.Sub(lastSeen) > time.Duration(ttlSeconds)*time.Second
}

func ruleSuggestionsEqual(left, right []RuleSuggestion) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

type suggestionsOutput struct {
	UpdatedAt string           `json:"updated_at"`
	Items     []RuleSuggestion `json:"items"`
}

func SaveSuggestionsIfChanged(path string, st State, now time.Time, minInterval time.Duration, lastWrite *time.Time) (bool, error) {
	raw, err := renderSuggestionsPayload(st, now)
	if err != nil {
		return false, err
	}
	if existing, readErr := os.ReadFile(path); readErr == nil {
		if bytes.Equal(existing, raw) {
			return false, nil
		}
	}
	if minInterval > 0 && lastWrite != nil && !lastWrite.IsZero() && now.Sub(*lastWrite) < minInterval {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return false, err
	}
	if lastWrite != nil {
		*lastWrite = now
	}
	return true, nil
}

func renderSuggestionsPayload(st State, now time.Time) ([]byte, error) {
	items := append([]RuleSuggestion(nil), st.L7Suggestions...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].PathPrefix < items[j].PathPrefix
	})
	out := suggestionsOutput{
		UpdatedAt: now.Format(time.RFC3339),
		Items:     items,
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}
