package compiler

import "testing"

func TestEasyAdminBypassPathPatternForSite_Localhost(t *testing.T) {
	pattern := easyAdminBypassPathPatternForSite("localhost")
	if pattern == "^$" {
		t.Fatalf("expected management bypass pattern for localhost")
	}
}

