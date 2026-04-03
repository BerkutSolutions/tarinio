package main

import (
	"bufio"
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
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

type modelConfig struct {
	ModelEnabled          bool
	LogPath               string
	StatePath             string
	OutputPath            string
	RuntimeRoot           string
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
}

type record struct {
	Score       float64 `json:"score"`
	LastSeen    string  `json:"last_seen"`
	LastUpdated string  `json:"last_updated"`
	Stage       string  `json:"stage"`
	ExpiresAt   string  `json:"expires_at"`
}

type state struct {
	Offset int64             `json:"offset"`
	IPs    map[string]record `json:"ips"`
}

type adaptiveEntry struct {
	IP        string `json:"ip"`
	Action    string `json:"action"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type adaptiveOutput struct {
	UpdatedAt             string          `json:"updated_at"`
	ThrottleRatePerSecond int             `json:"throttle_rate_per_second"`
	ThrottleBurst         int             `json:"throttle_burst"`
	ThrottleTarget        string          `json:"throttle_target"`
	Entries               []adaptiveEntry `json:"entries"`
}

type parsedAccess struct {
	ip     string
	site   string
	status int
	when   time.Time
}

type jsonAccess struct {
	Timestamp string `json:"timestamp"`
	ClientIP  string `json:"client_ip"`
	Site      string `json:"site"`
	Status    int    `json:"status"`
}

const globalSiteMarker = "*"

type secondStat struct {
	Count int
	IPs   map[string]struct{}
}

type runtimeActivePointer struct {
	CandidatePath string `json:"candidate_path"`
}

type runtimeProfile struct {
	ConnLimit                  int     `json:"conn_limit"`
	RatePerSecond              int     `json:"rate_per_second"`
	RateBurst                  int     `json:"rate_burst"`
	EnforceL7RateLimit         bool    `json:"enforce_l7_rate_limit"`
	L7RequestsPerSecond        int     `json:"l7_requests_per_second"`
	L7Burst                    int     `json:"l7_burst"`
	ModelEnabled               bool    `json:"model_enabled"`
	ModelPollIntervalSeconds   int     `json:"model_poll_interval_seconds"`
	ModelDecayLambda           float64 `json:"model_decay_lambda"`
	ModelThrottleThreshold     float64 `json:"model_throttle_threshold"`
	ModelDropThreshold         float64 `json:"model_drop_threshold"`
	ModelHoldSeconds           int     `json:"model_hold_seconds"`
	ModelThrottleRatePerSecond int     `json:"model_throttle_rate_per_second"`
	ModelThrottleBurst         int     `json:"model_throttle_burst"`
	ModelThrottleTarget        string  `json:"model_throttle_target"`
	ModelWeight429             float64 `json:"model_weight_429"`
	ModelWeight403             float64 `json:"model_weight_403"`
	ModelWeight444             float64 `json:"model_weight_444"`
	ModelEmergencyRPS          int     `json:"model_emergency_rps"`
	ModelEmergencyUniqueIPs    int     `json:"model_emergency_unique_ips"`
	ModelEmergencyPerIPRPS     int     `json:"model_emergency_per_ip_rps"`
	ModelWeightEmergencyBotnet float64 `json:"model_weight_emergency_botnet"`
	ModelWeightEmergencySingle float64 `json:"model_weight_emergency_single"`
}

func main() {
	cfg := loadConfig()
	st := loadState(cfg.StatePath)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		now := time.Now().UTC()
		effective := cfg
		if profile, ok := loadRuntimeProfile(cfg.RuntimeRoot); ok {
			effective = applyRuntimeProfile(cfg, profile)
		}
		next, changed, err := processTick(effective, st, now)
		if err != nil {
			log.Printf("ddos-model: tick failed: %v", err)
		} else {
			st = next
			if changed {
				if err := saveState(cfg.StatePath, st); err != nil {
					log.Printf("ddos-model: save state failed: %v", err)
				}
			}
			if err := saveAdaptive(cfg.OutputPath, effective, st, now); err != nil {
				log.Printf("ddos-model: save adaptive output failed: %v", err)
			}
		}
		<-ticker.C
	}
}

func loadConfig() modelConfig {
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
	return modelConfig{
		ModelEnabled:          envBool("MODEL_ENABLED", true),
		LogPath:               envString("MODEL_LOG_PATH", "/logs/access.log"),
		StatePath:             envString("MODEL_STATE_PATH", "/state/model-state.json"),
		OutputPath:            envString("MODEL_OUTPUT_PATH", "/out/adaptive.json"),
		RuntimeRoot:           envString("MODEL_RUNTIME_ROOT", "/var/lib/waf"),
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
	}
}

func loadRuntimeProfile(runtimeRoot string) (runtimeProfile, bool) {
	root := strings.TrimSpace(runtimeRoot)
	if root == "" {
		return runtimeProfile{}, false
	}
	rawPointer, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		return runtimeProfile{}, false
	}
	var pointer runtimeActivePointer
	if err := json.Unmarshal(rawPointer, &pointer); err != nil {
		return runtimeProfile{}, false
	}
	candidate := strings.TrimSpace(pointer.CandidatePath)
	if candidate == "" {
		return runtimeProfile{}, false
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, filepath.FromSlash(candidate))
	}
	rawConfig, err := os.ReadFile(filepath.Join(candidate, "ddos-model", "config.json"))
	if err != nil {
		return runtimeProfile{}, false
	}
	var profile runtimeProfile
	if err := json.Unmarshal(rawConfig, &profile); err != nil {
		return runtimeProfile{}, false
	}
	return profile, true
}

func applyRuntimeProfile(base modelConfig, profile runtimeProfile) modelConfig {
	out := base
	out.ModelEnabled = profile.ModelEnabled
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

func processTick(cfg modelConfig, current state, now time.Time) (state, bool, error) {
	if !cfg.ModelEnabled {
		next := state{
			Offset: current.Offset,
			IPs:    map[string]record{},
		}
		items, offset, err := readNewLines(cfg.LogPath, current.Offset)
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
		next.IPs = map[string]record{}
	}
	items, offset, err := readNewLines(cfg.LogPath, current.Offset)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return next, false, nil
		}
		return current, false, err
	}
	changed := offset != current.Offset
	next.Offset = offset
	emergencyRPS, emergencyUniqueIPs, emergencyPerIPRPS := effectiveEmergencyThresholds(cfg)
	perSecond := map[string]*secondStat{}
	perIPSecond := map[string]int{}
	for _, item := range items {
		second := item.when.UTC().Format("2006-01-02T15:04:05Z")
		stat := perSecond[second]
		if stat == nil {
			stat = &secondStat{IPs: map[string]struct{}{}}
			perSecond[second] = stat
		}
		stat.Count++
		stat.IPs[item.ip] = struct{}{}
		perIPSecond[item.ip+"|"+second]++

		weight := scoreWeight(cfg, item.status)
		if weight <= 0 {
			continue
		}
		changed = applyScore(&next, cfg, item.ip, item.site, item.when, weight) || changed
	}

	// Emergency botnet-like burst detector:
	// very high RPS with many unique clients in the same second.
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
		log.Printf("ddos-model: emergency botnet burst detected: rps=%d unique_ips=%d", stat.Count, len(stat.IPs))
		emergencyDrop := stat.Count >= emergencyRPS*2 && len(stat.IPs) >= emergencyUniqueIPs*2
		for _, ip := range ips {
			if emergencyDrop {
				changed = applyImmediateDrop(&next, cfg, ip, secondTS) || changed
				continue
			}
			changed = applyScore(&next, cfg, ip, globalSiteMarker, secondTS, cfg.WeightEmergencyBotnet) || changed
		}
	}

	// Emergency single-source flood detector.
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
		log.Printf("ddos-model: emergency single-source flood detected: ip=%s rps=%d", parts[0], count)
		if count >= emergencyPerIPRPS*2 {
			changed = applyImmediateDrop(&next, cfg, parts[0], secondTS) || changed
			continue
		}
		changed = applyScore(&next, cfg, parts[0], globalSiteMarker, secondTS, cfg.WeightEmergencySingle) || changed
	}

	for ip, rec := range next.IPs {
		lastUpdated, _ := parseTime(rec.LastUpdated)
		if !lastUpdated.IsZero() {
			delta := now.Sub(lastUpdated).Seconds()
			if delta > 0 {
				rec.Score = rec.Score * math.Exp(-cfg.DecayLambda*delta)
				rec.LastUpdated = now.Format(time.RFC3339)
			}
		}
		rec.Stage = selectStage(cfg, rec.Score)
		exp, _ := parseTime(rec.ExpiresAt)
		if (rec.Stage == "" && rec.Score < 0.2) || (!exp.IsZero() && now.After(exp)) {
			delete(next.IPs, ip)
			changed = true
			continue
		}
		next.IPs[ip] = rec
	}

	return next, changed, nil
}

func effectiveEmergencyThresholds(cfg modelConfig) (int, int, int) {
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

func scoreWeight(cfg modelConfig, status int) float64 {
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

func selectStage(cfg modelConfig, score float64) string {
	if score >= cfg.DropThreshold {
		return "drop"
	}
	if score >= cfg.ThrottleThreshold {
		return "throttle"
	}
	return ""
}

func applyImmediateDrop(st *state, cfg modelConfig, ip string, when time.Time) bool {
	if st == nil || strings.TrimSpace(ip) == "" {
		return false
	}
	key := makeScoreKey(globalSiteMarker, ip)
	rec := st.IPs[key]
	score := rec.Score
	if score < cfg.DropThreshold {
		score = cfg.DropThreshold
	}
	expires := when.Add(time.Duration(cfg.HoldSeconds) * time.Second).UTC()
	next := record{
		Score:       score,
		LastSeen:    when.UTC().Format(time.RFC3339),
		LastUpdated: when.UTC().Format(time.RFC3339),
		Stage:       "drop",
		ExpiresAt:   expires.Format(time.RFC3339),
	}
	changed := next.Score != rec.Score || next.Stage != rec.Stage || next.ExpiresAt != rec.ExpiresAt || next.LastSeen != rec.LastSeen
	st.IPs[key] = next
	return changed
}

func saveAdaptive(path string, cfg modelConfig, st state, now time.Time) error {
	out := adaptiveOutput{
		UpdatedAt:             now.Format(time.RFC3339),
		ThrottleRatePerSecond: cfg.ThrottleRatePerSecond,
		ThrottleBurst:         cfg.ThrottleBurst,
		ThrottleTarget:        cfg.ThrottleTarget,
		Entries:               make([]adaptiveEntry, 0, len(st.IPs)),
	}
	type ipAggregate struct {
		HasGlobal bool
		Sites     map[string]struct{}
		Action    string
		ExpiresAt time.Time
	}
	aggregated := map[string]ipAggregate{}

	for key, rec := range st.IPs {
		if rec.Stage != "throttle" && rec.Stage != "drop" {
			continue
		}
		site, ip := parseScoreKey(key)
		if ip == "" {
			continue
		}
		agg := aggregated[ip]
		if agg.Sites == nil {
			agg.Sites = map[string]struct{}{}
		}
		if site == globalSiteMarker {
			agg.HasGlobal = true
		} else if site != "" {
			agg.Sites[site] = struct{}{}
		}
		if rec.Stage == "drop" || agg.Action == "" {
			agg.Action = rec.Stage
		}
		if exp, err := parseTime(rec.ExpiresAt); err == nil && exp.After(agg.ExpiresAt) {
			agg.ExpiresAt = exp
		}
		aggregated[ip] = agg
	}

	for ip, agg := range aggregated {
		if !agg.HasGlobal && len(agg.Sites) < 2 {
			continue
		}
		entry := adaptiveEntry{
			IP:     ip,
			Action: agg.Action,
		}
		if !agg.ExpiresAt.IsZero() {
			entry.ExpiresAt = agg.ExpiresAt.Format(time.RFC3339)
		}
		out.Entries = append(out.Entries, entry)
	}
	sort.Slice(out.Entries, func(i, j int) bool { return out.Entries[i].IP < out.Entries[j].IP })
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func loadState(path string) state {
	content, err := os.ReadFile(path)
	if err != nil {
		return state{IPs: map[string]record{}}
	}
	var out state
	if err := json.Unmarshal(content, &out); err != nil {
		return state{IPs: map[string]record{}}
	}
	if out.IPs == nil {
		out.IPs = map[string]record{}
	}
	return out
}

func saveState(path string, st state) error {
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

func readNewLines(path string, offset int64) ([]parsedAccess, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, offset, err
	}
	if offset > stat.Size() {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, offset, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]parsedAccess, 0, 32)
	for scanner.Scan() {
		item, ok := parseAccessLine(strings.TrimSpace(scanner.Text()))
		if ok {
			out = append(out, item)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}
	pos, err := file.Seek(0, 1)
	if err != nil {
		return nil, offset, err
	}
	return out, pos, nil
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
						ip:     ip,
						site:   normalizeSiteID(item.Site),
						status: item.Status,
						when:   when.UTC(),
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
		ip:     ip,
		site:   normalizeSiteID(matches[9]),
		status: status,
		when:   when.UTC(),
	}, true
}

func applyScore(st *state, cfg modelConfig, ip string, site string, when time.Time, weight float64) bool {
	if st == nil || strings.TrimSpace(ip) == "" || weight <= 0 {
		return false
	}
	key := makeScoreKey(site, ip)
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
	stage := selectStage(cfg, score)
	expires := when.Add(time.Duration(cfg.HoldSeconds) * time.Second).UTC()
	next := record{
		Score:       score,
		LastSeen:    when.UTC().Format(time.RFC3339),
		LastUpdated: when.UTC().Format(time.RFC3339),
		Stage:       stage,
		ExpiresAt:   expires.Format(time.RFC3339),
	}
	changed := next.Score != rec.Score || next.Stage != rec.Stage || next.ExpiresAt != rec.ExpiresAt || next.LastSeen != rec.LastSeen
	st.IPs[key] = next
	return changed
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

func makeScoreKey(site, ip string) string {
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
