package tests

import (
	"testing"
)

func TestE2EUniqueIDIsCompactAndDistinct(t *testing.T) {
	first, second := e2eUniqueID(t, "e2e-dashboard"), e2eUniqueID(t, "e2e-dashboard")
	if first == second { t.Fatal("E2E resource IDs must be distinct") }
	if len(first) > 36 { t.Fatalf("E2E ID is too long for Nginx derived identifiers: %q", first) }
}
