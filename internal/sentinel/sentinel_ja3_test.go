package sentinel

import "testing"

func TestDeriveSignalWeights_JA3Risk_Present(t *testing.T) {
	signals := deriveSignalWeights(&ipStat{
		Count:            5,
		JA3BlacklistHits: 2,
		UniquePaths:      map[string]struct{}{},
		Sites:            map[string]struct{}{},
	})
	weight, ok := signals["signal_ja3_risk"]
	if !ok {
		t.Fatal("expected signal_ja3_risk to be present")
	}
	if weight <= 0 {
		t.Fatalf("expected positive signal_ja3_risk weight, got %f", weight)
	}
	// 2 hits * 2.0 = 4.0, capped at 4.0
	if weight > 4.0 {
		t.Fatalf("expected signal_ja3_risk <= 4.0, got %f", weight)
	}
}

func TestDeriveSignalWeights_JA3Risk_Absent_WhenNoHits(t *testing.T) {
	signals := deriveSignalWeights(&ipStat{
		Count:            10,
		JA3BlacklistHits: 0,
		UniquePaths:      map[string]struct{}{},
		Sites:            map[string]struct{}{},
	})
	if _, ok := signals["signal_ja3_risk"]; ok {
		t.Fatal("expected signal_ja3_risk to be absent when JA3BlacklistHits=0")
	}
}

func TestDeriveSignalWeights_JA3Risk_Cap(t *testing.T) {
	signals := deriveSignalWeights(&ipStat{
		Count:            100,
		JA3BlacklistHits: 10,
		UniquePaths:      map[string]struct{}{},
		Sites:            map[string]struct{}{},
	})
	weight := signals["signal_ja3_risk"]
	if weight > 4.0 {
		t.Fatalf("expected signal_ja3_risk capped at 4.0, got %f", weight)
	}
}

func TestIsKnownBadJA3_EmptyReturnsFalse(t *testing.T) {
	if isKnownBadJA3("") {
		t.Fatal("expected empty JA3 to return false")
	}
}

func TestIsKnownBadJA3_KnownBadReturnsTrue(t *testing.T) {
	if !isKnownBadJA3("c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8") {
		t.Fatal("expected known bad JA3 fingerprint to return true")
	}
}

func TestIsKnownBadJA3_UnknownReturnsFalse(t *testing.T) {
	if isKnownBadJA3("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("expected unknown JA3 fingerprint to return false")
	}
}

// TestIsJA3Blacklisted_OptIn verifies that signal fires only when operator has
// configured at least one site-level JA3 blacklist entry.
func TestIsJA3Blacklisted_OptIn_NoConfigNoSignal(t *testing.T) {
	// Known-bad fingerprint, but no operator config → must NOT fire.
	cfg := Config{JA3BlacklistFingerprints: nil}
	if isJA3Blacklisted("c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8", cfg) {
		t.Fatal("expected isJA3Blacklisted=false when no operator JA3 config present")
	}
}

func TestIsJA3Blacklisted_OptIn_WithConfig_KnownBad(t *testing.T) {
	// Operator has configured something → known-bad list is also active.
	cfg := Config{JA3BlacklistFingerprints: []string{"aabbccddeeff00112233445566778899"}}
	if !isJA3Blacklisted("c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8", cfg) {
		t.Fatal("expected isJA3Blacklisted=true for known-bad fp when operator config present")
	}
}

func TestIsJA3Blacklisted_OptIn_WithConfig_OperatorEntry(t *testing.T) {
	// Operator-provided fingerprint must match.
	fp := "aabbccddeeff00112233445566778899"
	cfg := Config{JA3BlacklistFingerprints: []string{fp}}
	if !isJA3Blacklisted(fp, cfg) {
		t.Fatal("expected isJA3Blacklisted=true for operator-provided fingerprint")
	}
}

func TestIsJA3Blacklisted_OptIn_WithConfig_UnknownDoesNotMatch(t *testing.T) {
	// Operator has config, but this fingerprint is neither known-bad nor in the list.
	cfg := Config{JA3BlacklistFingerprints: []string{"aabbccddeeff00112233445566778899"}}
	if isJA3Blacklisted("ffffffffffffffffffffffffffffffff", cfg) {
		t.Fatal("expected isJA3Blacklisted=false for unknown fingerprint not in any list")
	}
}

func TestJA3BlacklistHits_IncrementedOnKnownBad(t *testing.T) {
	cfg := Config{
		ModelEnabled:             true,
		MaxUniquePathsPerIP:      100,
		// Operator opted in with at least one fingerprint.
		JA3BlacklistFingerprints: []string{"c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8"},
	}
	badJA3 := "c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8"
	items := []parsedAccess{
		{ip: "1.2.3.4", site: "site-a", status: 200, ja3: badJA3, path: "/"},
		{ip: "1.2.3.4", site: "site-a", status: 200, ja3: badJA3, path: "/about"},
		{ip: "1.2.3.4", site: "site-a", status: 200, ja3: "unknown-safe-ja3", path: "/safe"},
	}

	perIPStats := map[string]*ipStat{}
	for _, item := range items {
		stats := perIPStats[item.ip]
		if stats == nil {
			stats = &ipStat{
				UniquePaths: map[string]struct{}{},
				Sites:       map[string]struct{}{},
				Site:        item.site,
			}
			perIPStats[item.ip] = stats
		}
		stats.Count++
		if item.ja3 != "" {
			stats.JA3Hits++
			if isJA3Blacklisted(item.ja3, cfg) {
				stats.JA3BlacklistHits++
			}
		}
	}

	stats := perIPStats["1.2.3.4"]
	if stats.JA3Hits != 3 {
		t.Fatalf("expected JA3Hits=3, got %d", stats.JA3Hits)
	}
	if stats.JA3BlacklistHits != 2 {
		t.Fatalf("expected JA3BlacklistHits=2 (2 known bad), got %d", stats.JA3BlacklistHits)
	}
}
