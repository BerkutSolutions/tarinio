package sentinel

import (
	"testing"
)

// --- TASK-4.1: signal_credential_stuffing ---

func TestDeriveSignalWeights_CredentialStuffing_Above5(t *testing.T) {
	stats := &ipStat{
		Count:        10,
		AuthFailures: 10,
	}
	signals := deriveSignalWeights(stats)
	w, ok := signals["signal_credential_stuffing"]
	if !ok {
		t.Fatal("expected signal_credential_stuffing to be present")
	}
	// (10-5)*0.4 = 2.0
	expected := float64(10-5) * 0.4
	if w != expected {
		t.Fatalf("expected weight %.2f, got %.2f", expected, w)
	}
}

func TestDeriveSignalWeights_CredentialStuffing_Below5(t *testing.T) {
	stats := &ipStat{
		Count:        5,
		AuthFailures: 3,
	}
	signals := deriveSignalWeights(stats)
	if _, ok := signals["signal_credential_stuffing"]; ok {
		t.Fatal("expected signal_credential_stuffing to be absent for AuthFailures < 5")
	}
}

func TestDeriveSignalWeights_CredentialStuffing_Cap(t *testing.T) {
	stats := &ipStat{
		Count:        100,
		AuthFailures: 100,
	}
	signals := deriveSignalWeights(stats)
	w, ok := signals["signal_credential_stuffing"]
	if !ok {
		t.Fatal("expected signal_credential_stuffing to be present")
	}
	if w > 3.0 {
		t.Fatalf("weight %.2f exceeds cap 3.0", w)
	}
}

// --- TASK-4.2: signal_antibot_fail ---

func TestDeriveSignalWeights_AntibotFail_Above3(t *testing.T) {
	stats := &ipStat{
		Count:       10,
		AntibotFails: 5,
	}
	signals := deriveSignalWeights(stats)
	w, ok := signals["signal_antibot_fail"]
	if !ok {
		t.Fatal("expected signal_antibot_fail to be present")
	}
	// 5*0.8 = 4.0 → cap 4.0
	expected := 4.0
	if w != expected {
		t.Fatalf("expected weight %.2f, got %.2f", expected, w)
	}
}

func TestDeriveSignalWeights_AntibotFail_Below3(t *testing.T) {
	stats := &ipStat{
		Count:       5,
		AntibotFails: 2,
	}
	signals := deriveSignalWeights(stats)
	if _, ok := signals["signal_antibot_fail"]; ok {
		t.Fatal("expected signal_antibot_fail to be absent for AntibotFails < 3")
	}
}

// --- TASK-4.2: signal_bad_behavior ---

func TestDeriveSignalWeights_BadBehavior_Above5(t *testing.T) {
	stats := &ipStat{
		Count:           20,
		BadBehaviorHits: 10,
	}
	signals := deriveSignalWeights(stats)
	w, ok := signals["signal_bad_behavior"]
	if !ok {
		t.Fatal("expected signal_bad_behavior to be present")
	}
	// (10-5)*0.3 = 1.5
	expected := float64(10-5) * 0.3
	if w != expected {
		t.Fatalf("expected weight %.2f, got %.2f", expected, w)
	}
}

func TestDeriveSignalWeights_BadBehavior_Below5(t *testing.T) {
	stats := &ipStat{
		Count:           5,
		BadBehaviorHits: 4,
	}
	signals := deriveSignalWeights(stats)
	if _, ok := signals["signal_bad_behavior"]; ok {
		t.Fatal("expected signal_bad_behavior to be absent for BadBehaviorHits < 5")
	}
}

// --- TASK-4.2: JA3 + antibot correlation ---

func TestDeriveSignalWeights_JA3AntibotCorrelation(t *testing.T) {
	stats := &ipStat{
		Count:            10,
		JA3BlacklistHits: 1,
		AntibotFails:     5,
	}
	signals := deriveSignalWeights(stats)
	if _, ok := signals["signal_ja3_risk"]; !ok {
		t.Fatal("expected signal_ja3_risk")
	}
	if _, ok := signals["signal_antibot_fail"]; !ok {
		t.Fatal("expected signal_antibot_fail")
	}
	corrW, ok := signals["signal_ja3_antibot_correlation"]
	if !ok {
		t.Fatal("expected signal_ja3_antibot_correlation when both signals active")
	}
	if corrW != 1.0 {
		t.Fatalf("expected correlation weight 1.0, got %.2f", corrW)
	}
	// total score with correlation > sum of individual signals
	ja3W := signals["signal_ja3_risk"]
	abW := signals["signal_antibot_fail"]
	total := ja3W + abW + corrW
	without := ja3W + abW
	if total <= without {
		t.Fatalf("correlation should increase total: %.2f vs %.2f", total, without)
	}
}

func TestDeriveSignalWeights_NoCorrelation_WhenOnlyOneSignal(t *testing.T) {
	// only antibot, no JA3 → no correlation
	stats := &ipStat{
		Count:       10,
		AntibotFails: 5,
	}
	signals := deriveSignalWeights(stats)
	if _, ok := signals["signal_ja3_antibot_correlation"]; ok {
		t.Fatal("should not have correlation signal when JA3 is absent")
	}
}

// --- Tick parsing: AuthFailures ---

func TestTickParsing_AuthFailures(t *testing.T) {
	authPaths := effectiveAuthPaths(Config{})
	// /login + 401 → auth failure
	if !isAuthPath("/login", authPaths) {
		t.Fatal("expected /login to be an auth path")
	}
	if !isAuthPath("/api/auth/token", authPaths) {
		t.Fatal("expected /api/auth/token to match /api/auth prefix")
	}
	// /dashboard + 401 → not an auth path
	if isAuthPath("/dashboard", authPaths) {
		t.Fatal("expected /dashboard to NOT be an auth path")
	}
}

func TestTickParsing_CustomAuthPaths(t *testing.T) {
	cfg := Config{AuthPaths: []string{"/custom/login"}}
	authPaths := effectiveAuthPaths(cfg)
	if !isAuthPath("/custom/login", authPaths) {
		t.Fatal("expected /custom/login to match custom auth path")
	}
	// default paths should not be present when custom list is given
	if isAuthPath("/login", authPaths) {
		t.Fatal("expected /login to NOT match when custom auth paths override defaults")
	}
}

// --- Explanations and recommendations for new signals ---

func TestSignalExplanation_NewSignals(t *testing.T) {
	codes := []string{
		"signal_credential_stuffing",
		"signal_antibot_fail",
		"signal_bad_behavior",
		"signal_ja3_antibot_correlation",
	}
	for _, code := range codes {
		exp := signalExplanation(code)
		if exp == "" || exp == "Contributed to risk score by anomaly model." {
			t.Fatalf("signal %q has no specific explanation", code)
		}
	}
}

func TestSignalRecommendations_NewSignals(t *testing.T) {
	codes := []string{
		"signal_credential_stuffing",
		"signal_antibot_fail",
		"signal_bad_behavior",
		"signal_ja3_antibot_correlation",
	}
	for _, code := range codes {
		recs := signalRecommendations(code)
		if len(recs) == 0 {
			t.Fatalf("signal %q has no recommendations", code)
		}
	}
}

// --- isBadBehaviorHit ---

func TestIsBadBehaviorHit(t *testing.T) {
	if !isBadBehaviorHit(429) {
		t.Fatal("expected 429 to be a bad behavior hit")
	}
	if isBadBehaviorHit(403) {
		t.Fatal("expected 403 to NOT be a bad behavior hit")
	}
	if isBadBehaviorHit(200) {
		t.Fatal("expected 200 to NOT be a bad behavior hit")
	}
}
