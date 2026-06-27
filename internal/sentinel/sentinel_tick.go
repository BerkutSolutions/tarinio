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

func processTick(cfg Config, current State, now time.Time) (State, bool, error) {
	if !cfg.ModelEnabled {
		next := State{
			Offset: current.Offset,
			IPs:    map[string]Record{},
		}
		items, offset, err := readNewEvents(newSourceBackend(cfg), current.Offset)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return next, len(current.IPs) > 0, nil
			}
			return current, false, err
		}
		_ = items
		next.Offset = offset
		return next, offset != current.Offset || len(current.IPs) > 0, nil
	}

	next := current
	if next.IPs == nil {
		next.IPs = map[string]Record{}
	}
	items, offset, err := readNewEvents(newSourceBackend(cfg), current.Offset)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return next, false, nil
		}
		return current, false, err
	}
	changed := offset != current.Offset
	next.Offset = offset

	emergencyRPS, emergencyUniqueIPs, emergencyPerIPRPS := EffectiveEmergencyThresholds(cfg)
	perSecond := map[string]*secondStat{}
	perIPSecond := map[string]int{}
	perIPStats := map[string]*ipStat{}
	scannerStats := map[string]*scannerPathStat{}
	var mlModel *mlLogisticModel
	if cfg.MLEnabled {
		loaded, err := loadMLModel(cfg.MLArtifactPath)
		if err != nil {
			logWarnf("tarinio-sentinel: ml disabled for this tick: %v", err)
		} else {
			mlModel = loaded
		}
	}

	for _, item := range items {
		if !isSiteModelEnabled(cfg, item.site) {
			continue
		}
		second := item.when.UTC().Format("2006-01-02T15:04:05Z")
		stat := perSecond[second]
		if stat == nil {
			stat = &secondStat{IPs: map[string]struct{}{}}
			perSecond[second] = stat
		}
		stat.Count++
		stat.IPs[item.ip] = struct{}{}
		perIPSecond[item.ip+"|"+second]++

		ipStats := perIPStats[item.ip]
		if ipStats == nil {
			ipStats = &ipStat{UniquePaths: map[string]struct{}{}, Sites: map[string]struct{}{}, Site: item.site}
			perIPStats[item.ip] = ipStats
		}
		if site := normalizeSiteID(item.site); site != "" {
			ipStats.Sites[site] = struct{}{}
		}
		ipStats.Count++
		if item.status == 404 {
			ipStats.NotFound++
		}
		if item.status == 403 || item.status == 429 || item.status == 444 {
			ipStats.Blocked++
		}
		if isScannerPath(item.path) {
			ipStats.ScannerHits++
			if scannerPath := matchedScannerPrefix(item.path); scannerPath != "" {
				stat := scannerStats[scannerPath]
				if stat == nil {
					stat = &scannerPathStat{IPs: map[string]struct{}{}, FirstSeen: item.when, LastSeen: item.when}
					scannerStats[scannerPath] = stat
				}
				stat.Hits++
				if item.status >= 200 && item.status < 400 {
					stat.SuccessHits++
				}
				stat.IPs[item.ip] = struct{}{}
				if item.when.Before(stat.FirstSeen) {
					stat.FirstSeen = item.when
				}
				if item.when.After(stat.LastSeen) {
					stat.LastSeen = item.when
				}
			}
		}
		if isSuspiciousUserAgent(item.userAgent) {
			ipStats.SuspiciousUAHits++
		}
		if item.ja3 != "" {
			ipStats.JA3Hits++
			if isJA3Blacklisted(item.ja3, cfg) {
				ipStats.JA3BlacklistHits++
			}
		}
		// credential stuffing: 401 on auth paths
		authPaths := effectiveAuthPaths(cfg)
		if item.status == 401 && isAuthPath(item.path, authPaths) {
			ipStats.AuthFailures++
			if ipStats.AuthPaths == nil {
				ipStats.AuthPaths = map[string]struct{}{}
			}
			if p := normalizePath(item.path); p != "" {
				ipStats.AuthPaths[p] = struct{}{}
			}
		}
		// antibot fail
		if item.antibotFail {
			ipStats.AntibotFails++
		}
		// bad behavior: 429 from bad-behavior zone
		if isBadBehaviorHit(item.status) {
			ipStats.BadBehaviorHits++
		}
		if path := normalizePath(item.path); path != "" {
			if len(ipStats.UniquePaths) < cfg.MaxUniquePathsPerIP {
				ipStats.UniquePaths[path] = struct{}{}
			}
		}

		weight := scoreWeight(cfg, item.status)
		if weight > 0 {
			changed = applyScore(&next, cfg, item.ip, item.site, item.when, weight, "signal_status_"+strconv.Itoa(item.status)) || changed
		}
	}

	for ip, stats := range perIPStats {
		signals := deriveSignalWeights(stats)
		if mlWeight, modelVersion, ok := inferMLWeight(stats, cfg, mlModel); ok {
			signals["signal_ml_risk"] = mlWeight
			key := MakeScoreKey(stats.Site, ip)
			rec := next.IPs[key]
			rec.ModelVersion = modelVersion
			next.IPs[key] = rec
			changed = true
		}
		if len(signals) == 0 {
			continue
		}
		for reason, weight := range signals {
			changed = applyScore(&next, cfg, ip, stats.Site, now, weight, reason) || changed
		}
	}

	for second, stat := range perSecond {
		if stat.Count < emergencyRPS || len(stat.IPs) < emergencyUniqueIPs {
			continue
		}
		secondTS, err := time.Parse("2006-01-02T15:04:05Z", second)
		if err != nil {
			secondTS = now
		}
		ips := make([]string, 0, len(stat.IPs))
		for ip := range stat.IPs {
			ips = append(ips, ip)
		}
		sort.Strings(ips)
		logWarnf("tarinio-sentinel: emergency botnet burst detected: rps=%d unique_ips=%d", stat.Count, len(stat.IPs))
		emergencyDrop := stat.Count >= emergencyRPS*2 && len(stat.IPs) >= emergencyUniqueIPs*2
		for _, ip := range ips {
			if emergencyDrop {
				changed = applyImmediateDrop(&next, cfg, ip, secondTS) || changed
				continue
			}
			changed = applyScore(&next, cfg, ip, globalSiteMarker, secondTS, cfg.WeightEmergencyBotnet, "signal_emergency_botnet") || changed
		}
	}

	for key, count := range perIPSecond {
		if count < emergencyPerIPRPS {
			continue
		}
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		secondTS, err := time.Parse("2006-01-02T15:04:05Z", parts[1])
		if err != nil {
			secondTS = now
		}
		logWarnf("tarinio-sentinel: emergency single-source flood detected: ip=%s rps=%d", parts[0], count)
		if count >= emergencyPerIPRPS*2 {
			changed = applyImmediateDrop(&next, cfg, parts[0], secondTS) || changed
			continue
		}
		changed = applyScore(&next, cfg, parts[0], globalSiteMarker, secondTS, cfg.WeightEmergencySingle, "signal_emergency_single") || changed
	}

	for key, rec := range next.IPs {
		lastUpdated, _ := parseTime(rec.LastUpdated)
		if !lastUpdated.IsZero() {
			delta := now.Sub(lastUpdated).Seconds()
			if delta > 0 {
				factor := math.Exp(-cfg.DecayLambda * delta)
				rec.Score = rec.Score * factor
				for signal, value := range rec.TopSignals {
					decayed := value * factor
					if decayed < 0.01 {
						delete(rec.TopSignals, signal)
						continue
					}
					rec.TopSignals[signal] = decayed
				}
				rec.LastUpdated = now.Format(time.RFC3339)
			}
		}

		if isExpiredByInactivity(rec, now, cfg.InactiveTTLSeconds) {
			delete(next.IPs, key)
			changed = true
			continue
		}

		signalCount := countStrongSignals(rec.TopSignals)
		desiredAction := decideAction(cfg, rec.Score)
		if requiresMultiSignalGate(desiredAction, cfg.PromotionMinSignals) && signalCount < cfg.PromotionMinSignals && !hasEmergencySignal(rec.TopSignals) {
			desiredAction = "throttle"
		}
		prevAction := currentAction(rec)
		if isCooldownDeescalation(prevAction, desiredAction, rec.LastActionAt, cfg.ActionCooldownSeconds, now) {
			desiredAction = prevAction
		}
		desiredAction, transitioned := resolveActionTransition(&rec, prevAction, desiredAction, cfg, hasEmergencySignal(rec.TopSignals))
		if desiredAction != prevAction && transitioned {
			if !consumeActionBudget(&next, cfg, now) {
				desiredAction = prevAction
			} else {
				rec.LastAction = desiredAction
				rec.LastActionAt = now.Format(time.RFC3339)
				changed = true
			}
		}

		rec.Stage = desiredAction
		rec.RiskScore = clamp(rec.Score, 0, 100)
		rec.TrustScore = clamp(100-rec.RiskScore, 0, 100)
		rec.ReasonCodes = topSignalKeys(rec.TopSignals, 5)
		rec.ExplainSummary, _, rec.Recommendations = buildExplainability(rec.TopSignals, rec.Stage)
		rec.ExpiresAt = actionExpiry(now, cfg, rec.Stage).Format(time.RFC3339)

		exp, _ := parseTime(rec.ExpiresAt)
		if (rec.Stage == "" && rec.Score < 0.2) || (!exp.IsZero() && now.After(exp)) {
			delete(next.IPs, key)
			changed = true
			continue
		}
		next.IPs[key] = rec
	}
	if evictOverflow(&next, cfg, now) {
		changed = true
	}
	suggestions := buildRuleSuggestions(cfg, scannerStats, next.L7Suggestions, now)
	if !ruleSuggestionsEqual(next.L7Suggestions, suggestions) {
		next.L7Suggestions = suggestions
		changed = true
	}

	return next, changed, nil
}
