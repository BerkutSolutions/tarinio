package main

import "strings"

func normalizeSiteID(value string) string {
	return strings.TrimSpace(value)
}
