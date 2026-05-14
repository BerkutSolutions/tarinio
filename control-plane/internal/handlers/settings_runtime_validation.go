package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"waf/internal/loggingconfig"
)

func parseStorageRetention(raw map[string]any, current StorageRetention) (StorageRetention, error) {
	out := current
	if value, ok := raw["logs_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.LogsDays = parsed
	}
	if value, ok := raw["activity_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.ActivityDays = parsed
	}
	if value, ok := raw["events_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.EventsDays = parsed
	}
	if value, ok := raw["bans_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.BansDays = parsed
	}
	if value, ok := raw["hot_index_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.HotIndexDays = parsed
	}
	if value, ok := raw["cold_index_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.ColdIndexDays = parsed
	}
	return normalizeStorageRetention(out), nil
}

func parsePositiveRetentionInt(value any) (int, error) {
	switch typed := value.(type) {
	case float64:
		parsed := int(typed)
		if parsed <= 0 {
			return 0, strconv.ErrSyntax
		}
		return parsed, nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil || parsed <= 0 {
			return 0, strconv.ErrSyntax
		}
		return parsed, nil
	default:
		return 0, strconv.ErrSyntax
	}
}

func normalizeStorageRetention(input StorageRetention) StorageRetention {
	if input.LogsDays <= 0 {
		input.LogsDays = 14
	}
	if input.ActivityDays <= 0 {
		input.ActivityDays = 30
	}
	if input.EventsDays <= 0 {
		input.EventsDays = 30
	}
	if input.BansDays <= 0 {
		input.BansDays = 30
	}
	if input.HotIndexDays <= 0 {
		input.HotIndexDays = loggingconfig.DefaultHotDays
	}
	if input.ColdIndexDays <= 0 {
		input.ColdIndexDays = loggingconfig.DefaultColdDays
	}
	if input.HotIndexDays > loggingconfig.MaxHotDays {
		input.HotIndexDays = loggingconfig.MaxHotDays
	}
	if input.ColdIndexDays > loggingconfig.MaxColdDays {
		input.ColdIndexDays = loggingconfig.MaxColdDays
	}
	return input
}

func normalizeRuntimeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ru", "de", "sr", "zh":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "en"
	}
}

func normalizeRuntimeSecuritySettings(input RuntimeSecuritySettings) RuntimeSecuritySettings {
	out := input
	if out.LoginRateLimitMaxAttempts <= 0 {
		out.LoginRateLimitMaxAttempts = 10
	}
	if out.LoginRateLimitWindowSecond <= 0 {
		out.LoginRateLimitWindowSecond = 300
	}
	if out.LoginRateLimitBlockSecond <= 0 {
		out.LoginRateLimitBlockSecond = 600
	}
	if out.LoginRateLimitMaxAttempts > 100 {
		out.LoginRateLimitMaxAttempts = 100
	}
	if out.LoginRateLimitWindowSecond > 24*60*60 {
		out.LoginRateLimitWindowSecond = 24 * 60 * 60
	}
	if out.LoginRateLimitBlockSecond > 24*60*60 {
		out.LoginRateLimitBlockSecond = 24 * 60 * 60
	}
	return out
}

func parseRuntimeSecuritySettings(raw map[string]any, current RuntimeSecuritySettings) (RuntimeSecuritySettings, error) {
	content, err := json.Marshal(raw)
	if err != nil {
		return RuntimeSecuritySettings{}, fmt.Errorf("encode security settings: %w", err)
	}
	next := normalizeRuntimeSecuritySettings(current)
	if err := json.Unmarshal(content, &next); err != nil {
		return RuntimeSecuritySettings{}, fmt.Errorf("decode security settings: %w", err)
	}
	next = normalizeRuntimeSecuritySettings(next)
	if next.LoginRateLimitMaxAttempts < 3 || next.LoginRateLimitMaxAttempts > 100 {
		return RuntimeSecuritySettings{}, fmt.Errorf("security.login_rate_limit_max_attempts must be between 3 and 100")
	}
	if next.LoginRateLimitWindowSecond < 60 || next.LoginRateLimitWindowSecond > 24*60*60 {
		return RuntimeSecuritySettings{}, fmt.Errorf("security.login_rate_limit_window_seconds must be between 60 and 86400")
	}
	if next.LoginRateLimitBlockSecond < 60 || next.LoginRateLimitBlockSecond > 24*60*60 {
		return RuntimeSecuritySettings{}, fmt.Errorf("security.login_rate_limit_block_seconds must be between 60 and 86400")
	}
	return next, nil
}
