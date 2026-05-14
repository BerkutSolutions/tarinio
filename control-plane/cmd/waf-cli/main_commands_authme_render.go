package main

import "fmt"

func renderAuthMe(value any) error {
	item, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	fmt.Println("Current user:")
	printKV(
		"id", stringify(item["id"]),
		"username", stringify(item["username"]),
		"email", stringify(item["email"]),
		"role", stringify(item["role"]),
		"totp_enabled", stringify(item["totp_enabled"]),
	)
	return nil
}
