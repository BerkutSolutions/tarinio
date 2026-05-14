package main

import "strings"

func normalizeBaseURL(value string) string {
	return strings.TrimRight(value, "/")
}
