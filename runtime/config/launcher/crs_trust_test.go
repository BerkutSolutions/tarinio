package main

import "testing"

func TestLoadCRSTrustedDigestsRejectsMalformedValues(t *testing.T) {
	t.Setenv(crsTrustedDigestsEnv, "4.1.0=not-a-digest")
	if _, err := loadCRSTrustedDigests(); err == nil {
		t.Fatal("expected malformed digest to be rejected")
	}
}

func TestLoadCRSTrustedDigestsLoadsPinnedRelease(t *testing.T) {
	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv(crsTrustedDigestsEnv, "v4.1.0="+digest)
	values, err := loadCRSTrustedDigests()
	if err != nil || values["4.1.0"] != digest {
		t.Fatalf("expected pinned digest, got %v, %v", values, err)
	}
}

func TestLoadCRSTrustedDigestsIncludesBuiltInReleasePin(t *testing.T) {
	t.Setenv(crsTrustedDigestsEnv, "")
	values, err := loadCRSTrustedDigests()
	if err != nil {
		t.Fatalf("load built-in pins: %v", err)
	}
	if values["4.28.0"] != "fca67fe46adafeeee61b9d1a03f38c25b9b2a799577df03fa51d99589e6d03b9" {
		t.Fatalf("unexpected built-in CRS 4.28.0 pin: %q", values["4.28.0"])
	}
}

func TestNewCRSManagerKeepsBuiltInPinsWhenEnvOverrideIsInvalid(t *testing.T) {
	t.Setenv(crsTrustedDigestsEnv, "4.28.0=invalid")
	manager := newCRSManager(t.TempDir(), "")
	if manager.trustedDigests["4.28.0"] != builtInCRSTrustedDigests["4.28.0"] {
		t.Fatal("invalid environment override must not clear built-in CRS pins")
	}
}
