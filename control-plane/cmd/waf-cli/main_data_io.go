package main

import (
	"encoding/json"
	"strings"
)

func readJSONFile(path string) (map[string]any, error) {
	raw, err := readFileBytes(path)
	if err != nil {
		return nil, err
	}
	return decodeJSONFilePayload(path, raw)
}

func extractErr(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if msg := extractErrorField(payload); msg != "" {
			return msg
		}
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "(empty response)"
	}
	return text
}
