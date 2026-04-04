package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyRuntimeProfile_DisabledModelIsRespected(t *testing.T) {
	base := modelConfig{
		ModelEnabled:      true,
		ThrottleThreshold: 2.5,
		DropThreshold:     6.0,
	}
	profile := runtimeProfile{
		ModelEnabled: false,
	}

	got := applyRuntimeProfile(base, profile)
	if got.ModelEnabled {
		t.Fatal("expected model to be disabled by runtime profile")
	}
}

func TestEffectiveEmergencyThresholds_ScalesWithConfiguredLimits(t *testing.T) {
	cfg := modelConfig{
		EmergencyRPS:       180,
		EmergencyUniqueIPs: 40,
		EmergencyPerIPRPS:  60,
		ConnLimit:          500,
		RatePerSecond:      300,
	}

	globalRPS, uniqueIPs, perIPRPS := effectiveEmergencyThresholds(cfg)
	if globalRPS <= 180 {
		t.Fatalf("expected global emergency rps to scale up, got %d", globalRPS)
	}
	if uniqueIPs < 40 || uniqueIPs > 50 {
		t.Fatalf("expected unique ip threshold in [40,50], got %d", uniqueIPs)
	}
	if perIPRPS <= 60 {
		t.Fatalf("expected per-ip threshold to scale up, got %d", perIPRPS)
	}
}

func TestSaveAdaptive_DoesNotEmitSingleSitePenalty(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "adaptive.json")
	now := time.Now().UTC()
	st := state{
		IPs: map[string]record{
			makeScoreKey("service1", "203.0.113.10"): {
				Score:       3.1,
				Stage:       "throttle",
				ExpiresAt:   now.Add(1 * time.Minute).Format(time.RFC3339),
				LastSeen:    now.Format(time.RFC3339),
				LastUpdated: now.Format(time.RFC3339),
			},
		},
	}
	if err := saveAdaptive(outPath, modelConfig{ThrottleRatePerSecond: 3, ThrottleBurst: 6, ThrottleTarget: "REJECT"}, st, now); err != nil {
		t.Fatalf("save adaptive: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read adaptive: %v", err)
	}
	var out adaptiveOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode adaptive: %v", err)
	}
	if len(out.Entries) != 0 {
		t.Fatalf("expected no global entries for single-site penalty, got %d", len(out.Entries))
	}
}
