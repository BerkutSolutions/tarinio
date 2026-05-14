package main

import "fmt"

func renderSetupStatus(value any) error {
	item, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	fmt.Println("Setup status:")
	printKV(
		"needs_bootstrap", stringify(item["needs_bootstrap"]),
		"bootstrap_allowed", stringify(item["bootstrap_allowed"]),
		"users_count", stringify(item["users_count"]),
		"needs_2fa", stringify(item["needs_2fa"]),
		"has_active_revision", stringify(item["has_active_revision"]),
	)
	return nil
}
