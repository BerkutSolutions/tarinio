package main

import (
	"encoding/json"
	"fmt"
)

func formatDefaultString(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}
