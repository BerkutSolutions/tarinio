package antiddos

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	ChainModeAuto       = "auto"
	ChainModeDockerUser = "docker-user"
	ChainModeInput      = "input"

	TargetDrop   = "DROP"
	TargetReject = "REJECT"
)

type Settings struct {
	UseL4Guard                 bool    `json:"use_l4_guard"`
	ChainMode                  string  `json:"chain_mode"`
	ConnLimit                  int     `json:"conn_limit"`
	RatePerSecond              int     `json:"rate_per_second"`
	RateBurst                  int     `json:"rate_burst"`
	Ports                      []int   `json:"ports"`
	Target                     string  `json:"target"`
	DestinationIP              string  `json:"destination_ip"`
	EnforceL7Rate              bool    `json:"enforce_l7_rate_limit"`
	L7RequestsPS               int     `json:"l7_requests_per_second"`
	L7Burst                    int     `json:"l7_burst"`
	L7StatusCode               int     `json:"l7_status_code"`
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
	CreatedAt                  string  `json:"created_at"`
	UpdatedAt                  string  `json:"updated_at"`
}

func DefaultSettings() Settings {
	connLimit := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_CONN_LIMIT", 200)
	ratePerSecond := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND", 100)
	rateBurst := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_RATE_BURST", 200)
	ports := ParsePortsCSV(envStringOrDefault("WAF_DEFAULT_ANTIDDOS_PORTS", "80,443"))
	if len(ports) == 0 {
		ports = []int{80, 443}
	}
	destinationIP := strings.TrimSpace(os.Getenv("WAF_DEFAULT_ANTIDDOS_DESTINATION_IP"))
	enforceL7 := envBoolOrDefault("WAF_DEFAULT_ANTIDDOS_ENFORCE_L7_RATE_LIMIT", true)
	l7RequestsPS := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_L7_REQUESTS_PER_SECOND", 100)
	l7Burst := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_L7_BURST", 200)
	l7StatusCode := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_L7_STATUS_CODE", 429)
	modelPoll := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_POLL_INTERVAL_SECONDS", 2)
	modelHold := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_HOLD_SECONDS", 60)
	modelThrottleRPS := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_THROTTLE_RATE_PER_SECOND", 3)
	modelThrottleBurst := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_THROTTLE_BURST", 6)
	modelEmergencyRPS := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_EMERGENCY_RPS", 180)
	modelEmergencyUnique := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_EMERGENCY_UNIQUE_IPS", 40)
	modelEmergencyPerIP := envIntOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_EMERGENCY_PER_IP_RPS", 60)

	return Settings{
		UseL4Guard:                 true,
		ChainMode:                  ChainModeAuto,
		ConnLimit:                  connLimit,
		RatePerSecond:              ratePerSecond,
		RateBurst:                  rateBurst,
		Ports:                      ports,
		Target:                     TargetDrop,
		DestinationIP:              destinationIP,
		EnforceL7Rate:              enforceL7,
		L7RequestsPS:               l7RequestsPS,
		L7Burst:                    l7Burst,
		L7StatusCode:               l7StatusCode,
		ModelEnabled:               envBoolOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED", true),
		ModelPollIntervalSeconds:   modelPoll,
		ModelDecayLambda:           envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_DECAY_LAMBDA", 0.08),
		ModelThrottleThreshold:     envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_THROTTLE_THRESHOLD", 2.5),
		ModelDropThreshold:         envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_DROP_THRESHOLD", 6.0),
		ModelHoldSeconds:           modelHold,
		ModelThrottleRatePerSecond: modelThrottleRPS,
		ModelThrottleBurst:         modelThrottleBurst,
		ModelThrottleTarget:        strings.ToUpper(envStringOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_THROTTLE_TARGET", "REJECT")),
		ModelWeight429:             envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_WEIGHT_429", 1.0),
		ModelWeight403:             envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_WEIGHT_403", 1.8),
		ModelWeight444:             envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_WEIGHT_444", 2.2),
		ModelEmergencyRPS:          modelEmergencyRPS,
		ModelEmergencyUniqueIPs:    modelEmergencyUnique,
		ModelEmergencyPerIPRPS:     modelEmergencyPerIP,
		ModelWeightEmergencyBotnet: envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_WEIGHT_EMERGENCY_BOTNET", 6.0),
		ModelWeightEmergencySingle: envFloatOrDefault("WAF_DEFAULT_ANTIDDOS_MODEL_WEIGHT_EMERGENCY_SINGLE", 4.0),
	}
}

func NormalizeSettings(item Settings) Settings {
	out := item
	out.ChainMode = strings.ToLower(strings.TrimSpace(out.ChainMode))
	if out.ChainMode == "" {
		out.ChainMode = ChainModeAuto
	}
	out.Target = strings.ToUpper(strings.TrimSpace(out.Target))
	if out.Target == "" {
		out.Target = TargetDrop
	}
	out.DestinationIP = strings.TrimSpace(out.DestinationIP)
	out.Ports = normalizePorts(out.Ports)
	if len(out.Ports) == 0 {
		out.Ports = []int{80, 443}
	}
	if out.ConnLimit <= 0 {
		out.ConnLimit = 200
	}
	if out.RatePerSecond <= 0 {
		out.RatePerSecond = 100
	}
	if out.RateBurst <= 0 {
		out.RateBurst = out.RatePerSecond * 2
	}
	if out.L7RequestsPS <= 0 {
		out.L7RequestsPS = 100
	}
	if out.L7Burst < 0 {
		out.L7Burst = out.L7RequestsPS * 2
	}
	if out.L7StatusCode <= 0 {
		out.L7StatusCode = 429
	}
	if out.ModelPollIntervalSeconds <= 0 {
		out.ModelPollIntervalSeconds = 2
	}
	if out.ModelDecayLambda <= 0 {
		out.ModelDecayLambda = 0.08
	}
	if out.ModelThrottleThreshold <= 0 {
		out.ModelThrottleThreshold = 2.5
	}
	if out.ModelDropThreshold <= out.ModelThrottleThreshold {
		out.ModelDropThreshold = out.ModelThrottleThreshold + 0.5
	}
	if out.ModelHoldSeconds <= 0 {
		out.ModelHoldSeconds = 60
	}
	if out.ModelThrottleRatePerSecond <= 0 {
		out.ModelThrottleRatePerSecond = 3
	}
	if out.ModelThrottleBurst <= 0 {
		out.ModelThrottleBurst = out.ModelThrottleRatePerSecond * 2
	}
	out.ModelThrottleTarget = strings.ToUpper(strings.TrimSpace(out.ModelThrottleTarget))
	if out.ModelThrottleTarget != TargetDrop && out.ModelThrottleTarget != TargetReject {
		out.ModelThrottleTarget = TargetReject
	}
	if out.ModelWeight429 <= 0 {
		out.ModelWeight429 = 1.0
	}
	if out.ModelWeight403 <= 0 {
		out.ModelWeight403 = 1.8
	}
	if out.ModelWeight444 <= 0 {
		out.ModelWeight444 = 2.2
	}
	if out.ModelEmergencyRPS <= 0 {
		out.ModelEmergencyRPS = 180
	}
	if out.ModelEmergencyUniqueIPs <= 0 {
		out.ModelEmergencyUniqueIPs = 40
	}
	if out.ModelEmergencyPerIPRPS <= 0 {
		out.ModelEmergencyPerIPRPS = 60
	}
	if out.ModelWeightEmergencyBotnet <= 0 {
		out.ModelWeightEmergencyBotnet = 6.0
	}
	if out.ModelWeightEmergencySingle <= 0 {
		out.ModelWeightEmergencySingle = 4.0
	}
	return out
}

func ValidateSettings(item Settings) error {
	switch item.ChainMode {
	case ChainModeAuto, ChainModeDockerUser, ChainModeInput:
	default:
		return errors.New("anti-ddos chain_mode must be auto, docker-user, or input")
	}
	switch item.Target {
	case TargetDrop, TargetReject:
	default:
		return errors.New("anti-ddos target must be DROP or REJECT")
	}
	if item.ConnLimit <= 0 {
		return errors.New("anti-ddos conn_limit must be positive")
	}
	if item.RatePerSecond <= 0 {
		return errors.New("anti-ddos rate_per_second must be positive")
	}
	if item.RateBurst < item.RatePerSecond {
		return errors.New("anti-ddos rate_burst must be greater than or equal to rate_per_second")
	}
	for _, port := range item.Ports {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("anti-ddos ports contains invalid value %d", port)
		}
	}
	if item.DestinationIP != "" && net.ParseIP(item.DestinationIP) == nil {
		return errors.New("anti-ddos destination_ip must be a valid ip address")
	}
	if item.EnforceL7Rate {
		if item.L7RequestsPS <= 0 {
			return errors.New("anti-ddos l7_requests_per_second must be positive when l7 rate limit is enabled")
		}
		if item.L7Burst < 0 {
			return errors.New("anti-ddos l7_burst must be zero or positive")
		}
		if item.L7StatusCode <= 0 {
			return errors.New("anti-ddos l7_status_code must be positive")
		}
	}
	if item.ModelEnabled {
		if item.ModelPollIntervalSeconds <= 0 {
			return errors.New("anti-ddos model_poll_interval_seconds must be positive")
		}
		if item.ModelDecayLambda <= 0 {
			return errors.New("anti-ddos model_decay_lambda must be positive")
		}
		if item.ModelThrottleThreshold <= 0 {
			return errors.New("anti-ddos model_throttle_threshold must be positive")
		}
		if item.ModelDropThreshold <= item.ModelThrottleThreshold {
			return errors.New("anti-ddos model_drop_threshold must be greater than model_throttle_threshold")
		}
		if item.ModelHoldSeconds <= 0 {
			return errors.New("anti-ddos model_hold_seconds must be positive")
		}
		if item.ModelThrottleRatePerSecond <= 0 {
			return errors.New("anti-ddos model_throttle_rate_per_second must be positive")
		}
		if item.ModelThrottleBurst <= 0 {
			return errors.New("anti-ddos model_throttle_burst must be positive")
		}
		if item.ModelThrottleTarget != TargetDrop && item.ModelThrottleTarget != TargetReject {
			return errors.New("anti-ddos model_throttle_target must be DROP or REJECT")
		}
	}
	return nil
}

func normalizePorts(items []int) []int {
	if len(items) == 0 {
		return nil
	}
	set := make(map[int]struct{}, len(items))
	for _, item := range items {
		if item <= 0 || item > 65535 {
			continue
		}
		set[item] = struct{}{}
	}
	out := make([]int, 0, len(set))
	for port := range set {
		out = append(out, port)
	}
	sort.Ints(out)
	return out
}

func ParsePortsCSV(value string) []int {
	parts := strings.Split(value, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return normalizePorts(out)
}

func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envStringOrDefault(key, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	return raw
}

func envBoolOrDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envFloatOrDefault(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

