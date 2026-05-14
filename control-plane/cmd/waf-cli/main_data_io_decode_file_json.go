package main

import (
	"encoding/json"
	"fmt"
)

func decodeJSONFilePayload(path string, raw []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode json %s: %w", path, err)
	}
	return payload, nil
}
