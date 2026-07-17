package services

import "testing"

func TestNormalizeNextPathRejectsOpenRedirectVariants(t *testing.T) {
	for _, value := range []string{
		"//attacker.example",
		"///attacker.example",
		"https://attacker.example",
		"/\\attacker.example",
		"/%5cattacker.example",
		"/%2f%2fattacker.example",
	} {
		if got := normalizeNextPath(value); got != "/healthcheck" {
			t.Fatalf("next path %q was not rejected: %q", value, got)
		}
	}
}

func TestNormalizeNextPathPreservesCanonicalRelativePath(t *testing.T) {
	if got := normalizeNextPath("/dashboard?tab=security#summary"); got != "/dashboard?tab=security#summary" {
		t.Fatalf("unexpected normalized path: %q", got)
	}
}
