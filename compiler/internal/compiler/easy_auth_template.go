package compiler

import "strings"

func normalizeAuthBasicTemplate(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "v1"
	}
}
