package compiler

import "testing"

func TestNginxInputValidationRejectsDirectiveBoundaries(t *testing.T) {
	for name, err := range map[string]error{
		"site-id": validateNginxIdentifier("site-a\ninclude /tmp/evil", "site id"),
		"host":    validateNginxHost("backend;proxy_pass http://evil", "upstream host"),
		"alias":   validateNginxHost("good.example.com\nlisten 8080", "alias"),
	} {
		if err == nil {
			t.Fatalf("expected %s injection to be rejected", name)
		}
	}
	if err := validateNginxHost("*.example.com", "host"); err != nil {
		t.Fatalf("expected wildcard host to remain supported: %v", err)
	}
}
