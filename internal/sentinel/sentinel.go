package sentinel

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sentinelsource "waf/internal/sentinel/source"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

const globalSiteMarker = "*"

var scannerPathPrefixes = []string{
	"/.env",
	"/wp-admin",
	"/phpmyadmin",
	"/vendor/phpunit",
	"/.git",
	"/boaform",
	"/cgi-bin",
}

// Config controls sentinel behavior and thresholds.
type Config struct {
	ModelEnabled          bool
	LogPath               string
	StatePath             string
	OutputPath            string
	RuntimeRoot           string
	SourceBackend         string
	EnabledSiteIDs        []string
	PollInterval          time.Duration
	DecayLambda           float64
	ThrottleThreshold     float64
	DropThreshold         float64
	HoldSeconds           int
	ThrottleRatePerSecond int
	ThrottleBurst         int
	ThrottleTarget        string
	Weight429             float64
	Weight403             float64
	Weight444             float64
	EmergencyRPS          int
	EmergencyUniqueIPs    int
	EmergencyPerIPRPS     int
	WeightEmergencyBotnet float64
	WeightEmergencySingle float64
	ConnLimit             int
	RatePerSecond         int
	RateBurst             int
	EnforceL7RateLimit    bool
	L7RequestsPerSecond   int
	L7Burst               int
	WatchThreshold        float64
	TempBanThreshold      float64
	PromotionMinSignals   int
	ActionCooldownSeconds int
	MaxActiveIPs          int
	MaxUniquePathsPerIP   int
	MaxPublishedEntries   int
	MaxActionsPerMinute   int
	InactiveTTLSeconds    int
	PublishInterval       time.Duration
	SuggestionsOutputPath string
	SuggestMinHits        int
	SuggestMinUniqueIPs   int
}

// Record is per (site,ip) adaptive state.
type Record struct {
	Score        float64            `json:"score"`
	RiskScore    float64            `json:"risk_score,omitempty"`
	TrustScore   float64            `json:"trust_score,omitempty"`
	FirstSeen    string             `json:"first_seen,omitempty"`
	LastSeen     string             `json:"last_seen"`
	LastUpdated  string             `json:"last_updated"`
	Stage        string             `json:"stage"`
	ExpiresAt    string             `json:"expires_at"`
	ReasonCodes  []string           `json:"reason_codes,omitempty"`
	TopSignals   map[string]float64 `json:"top_signals,omitempty"`
	LastAction   string             `json:"last_action,omitempty"`
	LastActionAt string             `json:"last_action_at,omitempty"`
}

// State stores persistent sentinel offset and adaptive records.
type State struct {
	Offset                int64             `json:"offset"`
	IPs                   map[string]Record `json:"ips"`
	L7Suggestions         []RuleSuggestion  `json:"l7_suggestions,omitempty"`
	ActionWindowStartedAt string            `json:"action_window_started_at,omitempty"`
	ActionsInWindow       int               `json:"actions_in_window,omitempty"`
}

// AdaptiveEntry is a single adaptive action entry for L4 guard.
type AdaptiveEntry struct {
	IP          string             `json:"ip"`
	Action      string             `json:"action"`
	ExpiresAt   string             `json:"expires_at,omitempty"`
	Score       float64            `json:"score,omitempty"`
	TrustScore  float64            `json:"trust_score,omitempty"`
	Source      string             `json:"source,omitempty"`
	FirstSeen   string             `json:"first_seen,omitempty"`
	LastSeen    string             `json:"last_seen,omitempty"`
	ReasonCodes []string           `json:"reason_codes,omitempty"`
	TopSignals  map[string]float64 `json:"top_signals,omitempty"`
}

// AdaptiveOutput is backward compatible with existing l4guard consumer.
type AdaptiveOutput struct {
	UpdatedAt             string          `json:"updated_at"`
	ThrottleRatePerSecond int             `json:"throttle_rate_per_second"`
	ThrottleBurst         int             `json:"throttle_burst"`
	ThrottleTarget        string          `json:"throttle_target"`
	Entries               []AdaptiveEntry `json:"entries"`
}

// RuleSuggestion is an L7 candidate generated from hot scanner paths.
type RuleSuggestion struct {
	ID          string `json:"id"`
	PathPrefix  string `json:"path_prefix"`
	Status      string `json:"status"`
	Hits        int    `json:"hits"`
	UniqueIPs   int    `json:"unique_ips"`
	WouldBlock  int    `json:"would_block_hits,omitempty"`
	ShadowHits  int    `json:"shadow_hits,omitempty"`
	ShadowFP    int    `json:"shadow_false_positive_hits,omitempty"`
	ShadowRate  string `json:"shadow_false_positive_rate,omitempty"`
	Source      string `json:"source,omitempty"`
	FirstSeen   string `json:"first_seen,omitempty"`
	LastSeen    string `json:"last_seen,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

type scannerPathStat struct {
	Hits      int
	IPs       map[string]struct{}
	FirstSeen time.Time
	LastSeen  time.Time
}

type parsedAccess struct {
	ip        string
	site      string
	status    int
	method    string
	path      string
	userAgent string
	when      time.Time
}

type jsonAccess struct {
	Timestamp string `json:"timestamp"`
	ClientIP  string `json:"client_ip"`
	Site      string `json:"site"`
	Status    int    `json:"status"`
	Method    string `json:"method"`
	URI       string `json:"uri"`
	UserAgent string `json:"user_agent"`
}

type secondStat struct {
	Count int
	IPs   map[string]struct{}
}

type ipStat struct {
	Count            int
	NotFound         int
	Blocked          int
	ScannerHits      int
	SuspiciousUAHits int
	UniquePaths      map[string]struct{}
	Site             string
}

type runtimeActivePointer struct {
	CandidatePath string `json:"candidate_path"`
}

// RuntimeProfile represents ddos-model profile from active runtime revision.
type RuntimeProfile struct {
	ConnLimit                  int      `json:"conn_limit"`
	RatePerSecond              int      `json:"rate_per_second"`
	RateBurst                  int      `json:"rate_burst"`
	EnforceL7RateLimit         bool     `json:"enforce_l7_rate_limit"`
	L7RequestsPerSecond        int      `json:"l7_requests_per_second"`
	L7Burst                    int      `json:"l7_burst"`
	ModelEnabled               bool     `json:"model_enabled"`
	ModelEnabledSites          []string `json:"model_enabled_sites"`
	ModelPollIntervalSeconds   int      `json:"model_poll_interval_seconds"`
	ModelDecayLambda           float64  `json:"model_decay_lambda"`
	ModelThrottleThreshold     float64  `json:"model_throttle_threshold"`
	ModelDropThreshold         float64  `json:"model_drop_threshold"`
	ModelHoldSeconds           int      `json:"model_hold_seconds"`
	ModelThrottleRatePerSecond int      `json:"model_throttle_rate_per_second"`
	ModelThrottleBurst         int      `json:"model_throttle_burst"`
	ModelThrottleTarget        string   `json:"model_throttle_target"`
	ModelWeight429             float64  `json:"model_weight_429"`
	ModelWeight403             float64  `json:"model_weight_403"`
	ModelWeight444             float64  `json:"model_weight_444"`
	ModelEmergencyRPS          int      `json:"model_emergency_rps"`
	ModelEmergencyUniqueIPs    int      `json:"model_emergency_unique_ips"`
	ModelEmergencyPerIPRPS     int      `json:"model_emergency_per_ip_rps"`
	ModelWeightEmergencyBotnet float64  `json:"model_weight_emergency_botnet"`
	ModelWeightEmergencySingle float64  `json:"model_weight_emergency_single"`
}

func logInfof(format string, args ...any) {
	log.Printf("[info] "+format, args...)
}

func logWarnf(format string, args ...any) {
	log.Printf("[warn] "+format, args...)
}

func logErrorf(format string, args ...any) {
	log.Printf("[error] "+format, args...)
}

// Run starts sentinel loop.
func Run(componentName string) {
	name := strings.TrimSpace(componentName)
	if name == "" {
		name = "tarinio-sentinel"
	}
	cfg := LoadConfig()
	st := loadState(cfg.StatePath)
	logInfof(
		"%s: started (enabled=%t poll_interval=%s log_path=%s state_path=%s output_path=%s runtime_root=%s)",
		name,
		cfg.ModelEnabled,
		cfg.PollInterval,
		cfg.LogPath,
		cfg.StatePath,
		cfg.OutputPath,
		cfg.RuntimeRoot,
	)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	var lastAdaptiveWrite time.Time
	var lastSuggestionWrite time.Time

	for {
		now := time.Now().UTC()
		effective := cfg
		if profile, ok := LoadRuntimeProfile(cfg.RuntimeRoot); ok {
			effective = ApplyRuntimeProfile(cfg, profile)
		}
		next, changed, err := processTick(effective, st, now)
		if err != nil {
			logErrorf("%s: tick failed: %v", name, err)
		} else {
			st = next
			if changed {
				if err := saveState(cfg.StatePath, st); err != nil {
					logErrorf("%s: save state failed: %v", name, err)
				}
			}
			if _, err := SaveAdaptiveIfChanged(cfg.OutputPath, effective, st, now, cfg.PublishInterval, &lastAdaptiveWrite); err != nil {
				logErrorf("%s: save adaptive output failed: %v", name, err)
			}
			if _, err := SaveSuggestionsIfChanged(cfg.SuggestionsOutputPath, st, now, cfg.PublishInterval, &lastSuggestionWrite); err != nil {
				logErrorf("%s: save suggestions output failed: %v", name, err)
			}
		}
		<-ticker.C
	}
}

// LoadConfig reads env-based config.
func LoadConfig() Config {
	poll := envInt("MODEL_POLL_INTERVAL_SECONDS", 2)
	if poll <= 0 {
		poll = 2
	}
	hold := envInt("MODEL_HOLD_SECONDS", 60)
	if hold <= 0 {
		hold = 60
	}
	rate := envInt("MODEL_THROTTLE_RATE_PER_SECOND", 3)
	if rate <= 0 {
		rate = 3
	}
	burst := envInt("MODEL_THROTTLE_BURST", 6)
	if burst <= 0 {
		burst = 6
	}
	target := strings.ToUpper(strings.TrimSpace(envString("MODEL_THROTTLE_TARGET", "REJECT")))
	if target != "DROP" && target != "REJECT" {
		target = "REJECT"
	}
	watchThreshold := envFloat("MODEL_WATCH_THRESHOLD", 1.0)
	if watchThreshold <= 0 {
		watchThreshold = 1.0
	}
	tempBanThreshold := envFloat("MODEL_TEMP_BAN_THRESHOLD", 12.0)
	if tempBanThreshold <= 0 {
		tempBanThreshold = 12.0
	}
	promotionMinSignals := envInt("MODEL_PROMOTION_MIN_SIGNALS", 2)
	if promotionMinSignals <= 0 {
		promotionMinSignals = 2
	}
	actionCooldown := envInt("MODEL_ACTION_COOLDOWN_SECONDS", 30)
	if actionCooldown < 0 {
		actionCooldown = 30
	}
	maxActiveIPs := envInt("MODEL_MAX_ACTIVE_IPS", 50000)
	if maxActiveIPs <= 0 {
		maxActiveIPs = 50000
	}
	maxUniquePaths := envInt("MODEL_MAX_UNIQUE_PATHS_PER_IP", 256)
	if maxUniquePaths <= 0 {
		maxUniquePaths = 256
	}
	maxPublishedEntries := envInt("MODEL_MAX_PUBLISHED_ENTRIES", 1000)
	if maxPublishedEntries <= 0 {
		maxPublishedEntries = 1000
	}
	maxActionsPerMinute := envInt("MODEL_MAX_ACTIONS_PER_MINUTE", 300)
	if maxActionsPerMinute <= 0 {
		maxActionsPerMinute = 300
	}
	inactiveTTL := envInt("MODEL_INACTIVE_TTL_SECONDS", 900)
	if inactiveTTL <= 0 {
		inactiveTTL = 900
	}
	publishInterval := envInt("MODEL_PUBLISH_INTERVAL_SECONDS", 5)
	if publishInterval <= 0 {
		publishInterval = 5
	}
	suggestMinHits := envInt("MODEL_SUGGEST_MIN_HITS", 20)
	if suggestMinHits <= 0 {
		suggestMinHits = 20
	}
	suggestMinUnique := envInt("MODEL_SUGGEST_MIN_UNIQUE_IPS", 5)
	if suggestMinUnique <= 0 {
		suggestMinUnique = 5
	}
	return Config{
		ModelEnabled:          envBool("MODEL_ENABLED", true),
		LogPath:               envString("MODEL_LOG_PATH", "/logs/access.log"),
		StatePath:             envString("MODEL_STATE_PATH", "/state/model-state.json"),
		OutputPath:            envString("MODEL_OUTPUT_PATH", "/out/adaptive.json"),
		RuntimeRoot:           envString("MODEL_RUNTIME_ROOT", "/var/lib/waf"),
		SourceBackend:         strings.ToLower(envString("MODEL_SOURCE_BACKEND", "file")),
		PollInterval:          time.Duration(poll) * time.Second,
		DecayLambda:           envFloat("MODEL_DECAY_LAMBDA", 0.08),
		ThrottleThreshold:     envFloat("MODEL_THROTTLE_THRESHOLD", 2.5),
		DropThreshold:         envFloat("MODEL_DROP_THRESHOLD", 6.0),
		HoldSeconds:           hold,
		ThrottleRatePerSecond: rate,
		ThrottleBurst:         burst,
		ThrottleTarget:        target,
		Weight429:             envFloat("MODEL_WEIGHT_429", 1.0),
		Weight403:             envFloat("MODEL_WEIGHT_403", 1.8),
		Weight444:             envFloat("MODEL_WEIGHT_444", 2.2),
		EmergencyRPS:          envInt("MODEL_EMERGENCY_RPS", 180),
		EmergencyUniqueIPs:    envInt("MODEL_EMERGENCY_UNIQUE_IPS", 40),
		EmergencyPerIPRPS:     envInt("MODEL_EMERGENCY_PER_IP_RPS", 60),
		WeightEmergencyBotnet: envFloat("MODEL_WEIGHT_EMERGENCY_BOTNET", 6.0),
		WeightEmergencySingle: envFloat("MODEL_WEIGHT_EMERGENCY_SINGLE", 4.0),
		WatchThreshold:        watchThreshold,
		TempBanThreshold:      tempBanThreshold,
		PromotionMinSignals:   promotionMinSignals,
		ActionCooldownSeconds: actionCooldown,
		MaxActiveIPs:          maxActiveIPs,
		MaxUniquePathsPerIP:   maxUniquePaths,
		MaxPublishedEntries:   maxPublishedEntries,
		MaxActionsPerMinute:   maxActionsPerMinute,
		InactiveTTLSeconds:    inactiveTTL,
		PublishInterval:       time.Duration(publishInterval) * time.Second,
		SuggestionsOutputPath: envString("MODEL_SUGGESTIONS_OUTPUT_PATH", "/out/l7-suggestions.json"),
		SuggestMinHits:        suggestMinHits,
		SuggestMinUniqueIPs:   suggestMinUnique,
	}
}

// LoadRuntimeProfile loads current runtime adaptive profile.
func LoadRuntimeProfile(runtimeRoot string) (RuntimeProfile, bool) {
	root := strings.TrimSpace(runtimeRoot)
	if root == "" {
		return RuntimeProfile{}, false
	}
	rawPointer, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		return RuntimeProfile{}, false
	}
	var pointer runtimeActivePointer
	if err := json.Unmarshal(rawPointer, &pointer); err != nil {
		return RuntimeProfile{}, false
	}
	candidate := strings.TrimSpace(pointer.CandidatePath)
	if candidate == "" {
		return RuntimeProfile{}, false
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, filepath.FromSlash(candidate))
	}
	rawConfig, err := os.ReadFile(filepath.Join(candidate, "ddos-model", "config.json"))
	if err != nil {
		return RuntimeProfile{}, false
	}
	var profile RuntimeProfile
	if err := json.Unmarshal(rawConfig, &profile); err != nil {
		return RuntimeProfile{}, false
	}
	return profile, true
}

// ApplyRuntimeProfile overlays revision profile onto base config.
func ApplyRuntimeProfile(base Config, profile RuntimeProfile) Config {
	out := base
	out.ModelEnabled = profile.ModelEnabled
	out.EnabledSiteIDs = normalizeSiteList(profile.ModelEnabledSites)
	out.ConnLimit = profile.ConnLimit
	out.RatePerSecond = profile.RatePerSecond
	out.RateBurst = profile.RateBurst
	out.EnforceL7RateLimit = profile.EnforceL7RateLimit
	out.L7RequestsPerSecond = profile.L7RequestsPerSecond
	out.L7Burst = profile.L7Burst
	if !out.ModelEnabled {
		return out
	}
	if profile.ModelPollIntervalSeconds > 0 {
		out.PollInterval = time.Duration(profile.ModelPollIntervalSeconds) * time.Second
	}
	if profile.ModelDecayLambda > 0 {
		out.DecayLambda = profile.ModelDecayLambda
	}
	if profile.ModelThrottleThreshold > 0 {
		out.ThrottleThreshold = profile.ModelThrottleThreshold
	}
	if profile.ModelDropThreshold > out.ThrottleThreshold {
		out.DropThreshold = profile.ModelDropThreshold
	}
	if out.WatchThreshold >= out.ThrottleThreshold {
		out.WatchThreshold = out.ThrottleThreshold * 0.5
	}
	if out.TempBanThreshold <= out.DropThreshold {
		out.TempBanThreshold = out.DropThreshold + 2
	}
	if profile.ModelHoldSeconds > 0 {
		out.HoldSeconds = profile.ModelHoldSeconds
	}
	if profile.ModelThrottleRatePerSecond > 0 {
		out.ThrottleRatePerSecond = profile.ModelThrottleRatePerSecond
	}
	if profile.ModelThrottleBurst > 0 {
		out.ThrottleBurst = profile.ModelThrottleBurst
	}
	target := strings.ToUpper(strings.TrimSpace(profile.ModelThrottleTarget))
	if target == "DROP" || target == "REJECT" {
		out.ThrottleTarget = target
	}
	if profile.ModelWeight429 > 0 {
		out.Weight429 = profile.ModelWeight429
	}
	if profile.ModelWeight403 > 0 {
		out.Weight403 = profile.ModelWeight403
	}
	if profile.ModelWeight444 > 0 {
		out.Weight444 = profile.ModelWeight444
	}
	if profile.ModelEmergencyRPS > 0 {
		out.EmergencyRPS = profile.ModelEmergencyRPS
	}
	if profile.ModelEmergencyUniqueIPs > 0 {
		out.EmergencyUniqueIPs = profile.ModelEmergencyUniqueIPs
	}
	if profile.ModelEmergencyPerIPRPS > 0 {
		out.EmergencyPerIPRPS = profile.ModelEmergencyPerIPRPS
	}
	if profile.ModelWeightEmergencyBotnet > 0 {
		out.WeightEmergencyBotnet = profile.ModelWeightEmergencyBotnet
	}
	if profile.ModelWeightEmergencySingle > 0 {
		out.WeightEmergencySingle = profile.ModelWeightEmergencySingle
	}
	return out
}

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
			ipStats = &ipStat{UniquePaths: map[string]struct{}{}, Site: item.site}
			perIPStats[item.ip] = ipStats
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
		if desiredAction != prevAction {
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
		rec.ReasonCodes = topSignalKeys(rec.TopSignals, 3)
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
	rec.ReasonCodes = topSignalKeys(rec.TopSignals, 3)
	st.IPs[key] = rec
	return true
}

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
		for signal, value := range rec.TopSignals {
			agg.TopSignals[signal] += value
		}
	}

	for ip, agg := range aggregated {
		if !agg.HasGlobal && len(agg.Sites) < 2 {
			continue
		}
		entry := AdaptiveEntry{
			IP:          ip,
			Action:      agg.Action,
			Score:       agg.Score,
			TrustScore:  agg.TrustScore,
			Source:      "tarinio-sentinel",
			ReasonCodes: topSignalKeys(agg.TopSignals, 3),
			TopSignals:  topSignalMap(agg.TopSignals, 5),
		}
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

func loadState(path string) State {
	content, err := os.ReadFile(path)
	if err != nil {
		return State{IPs: map[string]Record{}}
	}
	var out State
	if err := json.Unmarshal(content, &out); err != nil {
		return State{IPs: map[string]Record{}}
	}
	if out.IPs == nil {
		out.IPs = map[string]Record{}
	}
	return out
}

func saveState(path string, st State) error {
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func newSourceBackend(cfg Config) sentinelsource.Backend {
	switch strings.ToLower(strings.TrimSpace(cfg.SourceBackend)) {
	case "redis":
		return sentinelsource.NewRedisBackend()
	default:
		return sentinelsource.NewFileBackend(cfg.LogPath)
	}
}

func readNewLines(path string, offset int64) ([]parsedAccess, int64, error) {
	return readNewEvents(sentinelsource.NewFileBackend(path), offset)
}

func readNewEvents(backend sentinelsource.Backend, offset int64) ([]parsedAccess, int64, error) {
	items, nextOffset, err := backend.Read(offset)
	if err != nil {
		return nil, offset, err
	}
	out := make([]parsedAccess, 0, len(items))
	for _, item := range items {
		out = append(out, parsedAccess{
			ip:        item.IP,
			site:      normalizeSiteID(item.Site),
			status:    item.Status,
			method:    item.Method,
			path:      item.Path,
			userAgent: item.UserAgent,
			when:      item.When,
		})
	}
	return out, nextOffset, nil
}

func parseAccessLine(line string) (parsedAccess, bool) {
	if strings.HasPrefix(line, "{") {
		var item jsonAccess
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			ip := strings.TrimSpace(item.ClientIP)
			if ip != "" && item.Status > 0 {
				when, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
				if err == nil {
					return parsedAccess{
						ip:        ip,
						site:      normalizeSiteID(item.Site),
						status:    item.Status,
						method:    strings.TrimSpace(item.Method),
						path:      strings.TrimSpace(item.URI),
						userAgent: strings.TrimSpace(item.UserAgent),
						when:      when.UTC(),
					}, true
				}
			}
		}
	}

	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) != 10 {
		return parsedAccess{}, false
	}
	ip := strings.TrimSpace(matches[1])
	if ip == "" {
		return parsedAccess{}, false
	}
	status, err := strconv.Atoi(matches[5])
	if err != nil {
		return parsedAccess{}, false
	}
	when, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		return parsedAccess{}, false
	}
	return parsedAccess{
		ip:        ip,
		site:      normalizeSiteID(matches[9]),
		status:    status,
		method:    strings.TrimSpace(matches[3]),
		path:      strings.TrimSpace(matches[4]),
		userAgent: strings.TrimSpace(matches[8]),
		when:      when.UTC(),
	}, true
}

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
	rec.ReasonCodes = topSignalKeys(rec.TopSignals, 3)
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

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func buildRuleSuggestions(cfg Config, scannerStats map[string]*scannerPathStat, previous []RuleSuggestion, now time.Time) []RuleSuggestion {
	if len(scannerStats) == 0 {
		return append([]RuleSuggestion(nil), previous...)
	}
	prevByPath := map[string]RuleSuggestion{}
	for _, item := range previous {
		path := strings.TrimSpace(item.PathPrefix)
		if path == "" {
			continue
		}
		prevByPath[path] = item
	}
	out := make([]RuleSuggestion, 0, len(scannerStats))
	for path, stat := range scannerStats {
		if stat == nil {
			continue
		}
		unique := len(stat.IPs)
		if stat.Hits < cfg.SuggestMinHits || unique < cfg.SuggestMinUniqueIPs {
			continue
		}
		prev := prevByPath[path]
		status := strings.TrimSpace(prev.Status)
		if status == "" {
			status = "suggested"
		}
		shadowHits := prev.ShadowHits
		shadowFP := prev.ShadowFP
		if status == "shadow" {
			shadowHits += stat.Hits
		}
		firstSeen := stat.FirstSeen
		if existing, err := parseTime(prev.FirstSeen); err == nil && !existing.IsZero() && (firstSeen.IsZero() || existing.Before(firstSeen)) {
			firstSeen = existing
		}
		lastSeen := stat.LastSeen
		if existing, err := parseTime(prev.LastSeen); err == nil && !existing.IsZero() && existing.After(lastSeen) {
			lastSeen = existing
		}
		s := RuleSuggestion{
			ID:          "path-" + strings.TrimLeft(strings.ReplaceAll(path, "/", "-"), "-"),
			PathPrefix:  path,
			Status:      status,
			Hits:        stat.Hits,
			UniqueIPs:   unique,
			WouldBlock:  stat.Hits,
			ShadowHits:  shadowHits,
			ShadowFP:    shadowFP,
			Source:      "tarinio-sentinel",
			GeneratedAt: now.Format(time.RFC3339),
		}
		if shadowHits > 0 {
			s.ShadowRate = strconv.FormatFloat(float64(shadowFP)/float64(shadowHits), 'f', 4, 64)
		}
		if !firstSeen.IsZero() {
			s.FirstSeen = firstSeen.Format(time.RFC3339)
		}
		if !lastSeen.IsZero() {
			s.LastSeen = lastSeen.Format(time.RFC3339)
		}
		out = append(out, s)
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
