package sentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	promotionConsecutiveTicks := envInt("MODEL_PROMOTION_CONSECUTIVE_TICKS", 2)
	if promotionConsecutiveTicks <= 0 {
		promotionConsecutiveTicks = 2
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
	suggestShadowPromote := envInt("MODEL_SUGGEST_SHADOW_PROMOTE_HITS", suggestMinHits*2)
	if suggestShadowPromote <= 0 {
		suggestShadowPromote = suggestMinHits * 2
	}
	suggestTemporaryPromote := envInt("MODEL_SUGGEST_TEMPORARY_PROMOTE_HITS", suggestMinHits*4)
	if suggestTemporaryPromote <= 0 {
		suggestTemporaryPromote = suggestMinHits * 4
	}
	suggestPermanentPromote := envInt("MODEL_SUGGEST_PERMANENT_PROMOTE_HITS", suggestMinHits*8)
	if suggestPermanentPromote <= 0 {
		suggestPermanentPromote = suggestMinHits * 8
	}
	suggestShadowMaxFPRate := envFloat("MODEL_SUGGEST_SHADOW_MAX_FP_RATE", 0.02)
	if suggestShadowMaxFPRate <= 0 {
		suggestShadowMaxFPRate = 0.02
	}
	suggestTemporaryHold := envInt("MODEL_SUGGEST_TEMPORARY_HOLD_SECONDS", 14400)
	if suggestTemporaryHold <= 0 {
		suggestTemporaryHold = 14400
	}
	suggestPermanentMinLifetime := envInt("MODEL_SUGGEST_PERMANENT_MIN_LIFETIME_SECONDS", 21600)
	if suggestPermanentMinLifetime <= 0 {
		suggestPermanentMinLifetime = 21600
	}
	suggestInactiveTTL := envInt("MODEL_SUGGEST_INACTIVE_TTL_SECONDS", 86400)
	if suggestInactiveTTL <= 0 {
		suggestInactiveTTL = 86400
	}
	mlMinProbability := envFloat("MODEL_ML_MIN_PROBABILITY", 0.65)
	if mlMinProbability <= 0 || mlMinProbability >= 1 {
		mlMinProbability = 0.65
	}
	mlMaxWeight := envFloat("MODEL_ML_MAX_WEIGHT", 1.5)
	if mlMaxWeight <= 0 {
		mlMaxWeight = 1.5
	}
	return Config{
		ModelEnabled:                envBool("MODEL_ENABLED", true),
		MLEnabled:                   envBool("MODEL_ML_ENABLED", false),
		MLArtifactPath:              envString("MODEL_ML_ARTIFACT_PATH", "/etc/waf/ddos-model/ml-model.json"),
		MLMinProbability:            mlMinProbability,
		MLMaxWeight:                 mlMaxWeight,
		LogPath:                     envString("MODEL_LOG_PATH", "/logs/access.log"),
		StatePath:                   envString("MODEL_STATE_PATH", "/state/model-state.json"),
		OutputPath:                  envString("MODEL_OUTPUT_PATH", "/out/adaptive.json"),
		RuntimeRoot:                 envString("MODEL_RUNTIME_ROOT", "/var/lib/waf"),
		SourceBackend:               strings.ToLower(envString("MODEL_SOURCE_BACKEND", "file")),
		PollInterval:                time.Duration(poll) * time.Second,
		DecayLambda:                 envFloat("MODEL_DECAY_LAMBDA", 0.08),
		ThrottleThreshold:           envFloat("MODEL_THROTTLE_THRESHOLD", 2.5),
		DropThreshold:               envFloat("MODEL_DROP_THRESHOLD", 6.0),
		HoldSeconds:                 hold,
		ThrottleRatePerSecond:       rate,
		ThrottleBurst:               burst,
		ThrottleTarget:              target,
		Weight429:                   envFloat("MODEL_WEIGHT_429", 1.0),
		Weight403:                   envFloat("MODEL_WEIGHT_403", 1.8),
		Weight444:                   envFloat("MODEL_WEIGHT_444", 2.2),
		EmergencyRPS:                envInt("MODEL_EMERGENCY_RPS", 180),
		EmergencyUniqueIPs:          envInt("MODEL_EMERGENCY_UNIQUE_IPS", 40),
		EmergencyPerIPRPS:           envInt("MODEL_EMERGENCY_PER_IP_RPS", 60),
		WeightEmergencyBotnet:       envFloat("MODEL_WEIGHT_EMERGENCY_BOTNET", 6.0),
		WeightEmergencySingle:       envFloat("MODEL_WEIGHT_EMERGENCY_SINGLE", 4.0),
		WatchThreshold:              watchThreshold,
		TempBanThreshold:            tempBanThreshold,
		PromotionMinSignals:         promotionMinSignals,
		PromotionConsecutiveTicks:   promotionConsecutiveTicks,
		ActionCooldownSeconds:       actionCooldown,
		MaxActiveIPs:                maxActiveIPs,
		MaxUniquePathsPerIP:         maxUniquePaths,
		MaxPublishedEntries:         maxPublishedEntries,
		MaxActionsPerMinute:         maxActionsPerMinute,
		InactiveTTLSeconds:          inactiveTTL,
		PublishInterval:             time.Duration(publishInterval) * time.Second,
		SuggestInactiveTTLSeconds:   suggestInactiveTTL,
		SuggestionsOutputPath:       envString("MODEL_SUGGESTIONS_OUTPUT_PATH", "/out/l7-suggestions.json"),
		SuggestMinHits:              suggestMinHits,
		SuggestMinUniqueIPs:         suggestMinUnique,
		SuggestShadowPromoteHits:    suggestShadowPromote,
		SuggestTemporaryPromoteHits: suggestTemporaryPromote,
		SuggestPermanentPromoteHits: suggestPermanentPromote,
		SuggestShadowMaxFPRate:      suggestShadowMaxFPRate,
		SuggestTemporaryHoldSeconds: suggestTemporaryHold,
		SuggestPermanentMinLifetime: time.Duration(suggestPermanentMinLifetime) * time.Second,
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
