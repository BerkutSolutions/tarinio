package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func renderHealthResponse(outputJSON bool, status int, body []byte) error {
	if outputJSON {
		var payload any
		if json.Unmarshal(body, &payload) == nil {
			return printJSON(payload)
		}
		fmt.Printf("{\"status\":%d,\"body\":%q}\n", status, strings.TrimSpace(string(body)))
		return nil
	}
	fmt.Printf("Health: HTTP %d\n", status)
	fmt.Printf("Response: %s\n", strings.TrimSpace(string(body)))
	return nil
}
