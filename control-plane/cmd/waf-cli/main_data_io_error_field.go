package main

import "strings"

func extractErrorField(payload map[string]any) string {
	msg, ok := payload["error"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(msg)
}
