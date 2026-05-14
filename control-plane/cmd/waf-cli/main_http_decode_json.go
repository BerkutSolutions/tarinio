package main

import (
	"encoding/json"
	"fmt"
)

func decodeJSONBody(body []byte) (any, error) {
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out, nil
}
