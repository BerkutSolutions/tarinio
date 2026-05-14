package main

import (
	"os"
	"strings"
)

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return formatBoolString(v)
	case float64:
		return formatFloatString(v)
	default:
		return formatDefaultString(v)
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
