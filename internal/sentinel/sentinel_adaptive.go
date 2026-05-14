package sentinel

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SaveAdaptive writes adaptive file for l4guard.
func SaveAdaptive(path string, cfg Config, st State, now time.Time) error {
	raw, err := renderAdaptivePayload(cfg, st, now)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// SaveAdaptiveIfChanged writes adaptive output only when payload changed and interval elapsed.
func SaveAdaptiveIfChanged(path string, cfg Config, st State, now time.Time, minInterval time.Duration, lastWrite *time.Time) (bool, error) {
	raw, err := renderAdaptivePayload(cfg, st, now)
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

func renderAdaptivePayload(cfg Config, st State, now time.Time) ([]byte, error) {
	out := AdaptiveOutput{
		UpdatedAt:             now.Format(time.RFC3339),
		ThrottleRatePerSecond: cfg.ThrottleRatePerSecond,
		ThrottleBurst:         cfg.ThrottleBurst,
		ThrottleTarget:        cfg.ThrottleTarget,
		Entries:               make([]AdaptiveEntry, 0, len(st.IPs)),
	}
	type ipAggregate struct {
		HasGlobal  bool
		Sites      map[string]struct{}
		Action     string
		ExpiresAt  time.Time
		Score      float64
		TrustScore float64
		ModelVer   string
		TopSignals map[string]float64
		FirstSeen  time.Time
		LastSeen   time.Time
	}
	aggregated := map[string]*ipAggregate{}

	for key, rec := range st.IPs {
		if rec.Stage != "throttle" && rec.Stage != "drop" && rec.Stage != "temp_ban" {
			continue
		}
		site, ip := parseScoreKey(key)
		if ip == "" {
			continue
		}
		agg := aggregated[ip]
		if agg == nil {
			agg = &ipAggregate{
				Sites:      map[string]struct{}{},
				TopSignals: map[string]float64{},
			}
			aggregated[ip] = agg
		}
		if site == globalSiteMarker {
			agg.HasGlobal = true
		} else if site != "" {
			agg.Sites[site] = struct{}{}
		}
		mappedAction := rec.Stage
		if mappedAction == "temp_ban" {
			mappedAction = "drop"
		}
		if mappedAction == "drop" || agg.Action == "" {
			agg.Action = mappedAction
		}
		if exp, err := parseTime(rec.ExpiresAt); err == nil && exp.After(agg.ExpiresAt) {
			agg.ExpiresAt = exp
		}
		if seen, err := parseTime(rec.FirstSeen); err == nil && !seen.IsZero() && (agg.FirstSeen.IsZero() || seen.Before(agg.FirstSeen)) {
			agg.FirstSeen = seen
		}
		if seen, err := parseTime(rec.LastSeen); err == nil && !seen.IsZero() && seen.After(agg.LastSeen) {
			agg.LastSeen = seen
		}
		if rec.Score > agg.Score {
			agg.Score = rec.Score
		}
		if agg.TrustScore == 0 || rec.TrustScore < agg.TrustScore {
			agg.TrustScore = rec.TrustScore
		}
		if strings.TrimSpace(agg.ModelVer) == "" && strings.TrimSpace(rec.ModelVersion) != "" {
			agg.ModelVer = strings.TrimSpace(rec.ModelVersion)
		}
		for signal, value := range rec.TopSignals {
			agg.TopSignals[signal] += value
		}
	}

	for ip, agg := range aggregated {
		if !agg.HasGlobal && len(agg.Sites) < 2 {
			continue
		}
		entry := AdaptiveEntry{
			IP:           ip,
			Action:       agg.Action,
			Score:        agg.Score,
			TrustScore:   agg.TrustScore,
			ModelVersion: agg.ModelVer,
			Source:       "tarinio-sentinel",
			ReasonCodes:  topSignalKeys(agg.TopSignals, 5),
			TopSignals:   topSignalMap(agg.TopSignals, 8),
		}
		entry.ExplainSummary, entry.ReasonDetails, entry.Recommendations = buildExplainability(agg.TopSignals, entry.Action)
		if !agg.ExpiresAt.IsZero() {
			entry.ExpiresAt = agg.ExpiresAt.Format(time.RFC3339)
		}
		if !agg.FirstSeen.IsZero() {
			entry.FirstSeen = agg.FirstSeen.Format(time.RFC3339)
		}
		if !agg.LastSeen.IsZero() {
			entry.LastSeen = agg.LastSeen.Format(time.RFC3339)
		}
		out.Entries = append(out.Entries, entry)
	}
	if cfg.MaxPublishedEntries > 0 && len(out.Entries) > cfg.MaxPublishedEntries {
		sort.Slice(out.Entries, func(i, j int) bool {
			if out.Entries[i].Score == out.Entries[j].Score {
				return out.Entries[i].IP < out.Entries[j].IP
			}
			return out.Entries[i].Score > out.Entries[j].Score
		})
		out.Entries = out.Entries[:cfg.MaxPublishedEntries]
	}
	sort.Slice(out.Entries, func(i, j int) bool { return out.Entries[i].IP < out.Entries[j].IP })
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}
