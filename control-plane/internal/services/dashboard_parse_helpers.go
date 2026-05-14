package services

import (
	"strconv"
	"strings"
	"time"
)

func parseAnyTime(value any) time.Time {
	raw := strings.TrimSpace(asString(value))
	if raw == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts.UTC()
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC()
	}
	return time.Time{}
}

func parseAnyInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.Itoa(int(typed))
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.Itoa(int(typed))
	case float32:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
