package main

import "strings"

func listUsageCommand(title string) string {
	return strings.ReplaceAll(title, " ", "-")
}
