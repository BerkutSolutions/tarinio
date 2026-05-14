package sentinel

import "time"

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
	ModelEnabled                bool
	MLEnabled                   bool
	MLArtifactPath              string
	MLMinProbability            float64
	MLMaxWeight                 float64
	LogPath                     string
	StatePath                   string
	OutputPath                  string
	RuntimeRoot                 string
	SourceBackend               string
	EnabledSiteIDs              []string
	PollInterval                time.Duration
	DecayLambda                 float64
	ThrottleThreshold           float64
	DropThreshold               float64
	HoldSeconds                 int
	ThrottleRatePerSecond       int
	ThrottleBurst               int
	ThrottleTarget              string
	Weight429                   float64
	Weight403                   float64
	Weight444                   float64
	EmergencyRPS                int
	EmergencyUniqueIPs          int
	EmergencyPerIPRPS           int
	WeightEmergencyBotnet       float64
	WeightEmergencySingle       float64
	ConnLimit                   int
	RatePerSecond               int
	RateBurst                   int
	EnforceL7RateLimit          bool
	L7RequestsPerSecond         int
	L7Burst                     int
	WatchThreshold              float64
	TempBanThreshold            float64
	PromotionMinSignals         int
	PromotionConsecutiveTicks   int
	ActionCooldownSeconds       int
	MaxActiveIPs                int
	MaxUniquePathsPerIP         int
	MaxPublishedEntries         int
	MaxActionsPerMinute         int
	InactiveTTLSeconds          int
	PublishInterval             time.Duration
	SuggestInactiveTTLSeconds   int
	SuggestionsOutputPath       string
	SuggestMinHits              int
	SuggestMinUniqueIPs         int
	SuggestShadowPromoteHits    int
	SuggestTemporaryPromoteHits int
	SuggestPermanentPromoteHits int
	SuggestShadowMaxFPRate      float64
	SuggestTemporaryHoldSeconds int
	SuggestPermanentMinLifetime time.Duration
}

// Record is per (site,ip) adaptive state.
type Record struct {
	Score           float64            `json:"score"`
	RiskScore       float64            `json:"risk_score,omitempty"`
	TrustScore      float64            `json:"trust_score,omitempty"`
	ModelVersion    string             `json:"model_version,omitempty"`
	FirstSeen       string             `json:"first_seen,omitempty"`
	LastSeen        string             `json:"last_seen"`
	LastUpdated     string             `json:"last_updated"`
	Stage           string             `json:"stage"`
	ExpiresAt       string             `json:"expires_at"`
	ReasonCodes     []string           `json:"reason_codes,omitempty"`
	TopSignals      map[string]float64 `json:"top_signals,omitempty"`
	ExplainSummary  string             `json:"explain_summary,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
	LastAction      string             `json:"last_action,omitempty"`
	LastActionAt    string             `json:"last_action_at,omitempty"`
	CandidateAction string             `json:"candidate_action,omitempty"`
	CandidateCount  int                `json:"candidate_count,omitempty"`
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
	IP              string               `json:"ip"`
	Action          string               `json:"action"`
	ExpiresAt       string               `json:"expires_at,omitempty"`
	Score           float64              `json:"score,omitempty"`
	TrustScore      float64              `json:"trust_score,omitempty"`
	ModelVersion    string               `json:"model_version,omitempty"`
	Source          string               `json:"source,omitempty"`
	FirstSeen       string               `json:"first_seen,omitempty"`
	LastSeen        string               `json:"last_seen,omitempty"`
	ReasonCodes     []string             `json:"reason_codes,omitempty"`
	TopSignals      map[string]float64   `json:"top_signals,omitempty"`
	ExplainSummary  string               `json:"explain_summary,omitempty"`
	ReasonDetails   []SignalReasonDetail `json:"reason_details,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
}

type SignalReasonDetail struct {
	Code        string  `json:"code"`
	Weight      float64 `json:"weight"`
	Explanation string  `json:"explanation"`
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
	ID              string `json:"id"`
	PathPrefix      string `json:"path_prefix"`
	Status          string `json:"status"`
	Hits            int    `json:"hits"`
	UniqueIPs       int    `json:"unique_ips"`
	WouldBlock      int    `json:"would_block_hits,omitempty"`
	ShadowHits      int    `json:"shadow_hits,omitempty"`
	ShadowFP        int    `json:"shadow_false_positive_hits,omitempty"`
	ShadowRate      string `json:"shadow_false_positive_rate,omitempty"`
	TemporaryUntil  string `json:"temporary_until,omitempty"`
	PromotionReason string `json:"promotion_reason,omitempty"`
	Source          string `json:"source,omitempty"`
	FirstSeen       string `json:"first_seen,omitempty"`
	LastSeen        string `json:"last_seen,omitempty"`
	GeneratedAt     string `json:"generated_at,omitempty"`
}

type scannerPathStat struct {
	Hits        int
	SuccessHits int
	IPs         map[string]struct{}
	FirstSeen   time.Time
	LastSeen    time.Time
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
	Sites            map[string]struct{}
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
