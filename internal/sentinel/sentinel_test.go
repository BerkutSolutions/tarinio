package sentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sentinelsource "waf/internal/sentinel/source"
)

func TestDecideActionThresholds(t *testing.T) {
	cfg := Config{
		WatchThreshold:    1.0,
		ThrottleThreshold: 3.0,
		DropThreshold:     6.0,
		TempBanThreshold:  10.0,
	}
	cases := []struct {
		score float64
		want  string
	}{
		{score: 0.5, want: ""},
		{score: 1.2, want: "watch"},
		{score: 3.1, want: "throttle"},
		{score: 6.4, want: "drop"},
		{score: 10.5, want: "temp_ban"},
	}
	for _, tc := range cases {
		if got := decideAction(cfg, tc.score); got != tc.want {
			t.Fatalf("score %.2f: expected %q, got %q", tc.score, tc.want, got)
		}
	}
}

func TestNewSourceBackendModes(t *testing.T) {
	fileBackend := newSourceBackend(Config{SourceBackend: "file", LogPath: "access.log"})
	if _, ok := fileBackend.(*sentinelsource.FileBackend); !ok {
		t.Fatalf("expected file backend for file mode, got %T", fileBackend)
	}
	redisBackend := newSourceBackend(Config{SourceBackend: "redis", LogPath: "access.log"})
	if _, ok := redisBackend.(*sentinelsource.FallbackBackend); !ok {
		t.Fatalf("expected fallback backend for redis mode, got %T", redisBackend)
	}
	unknownBackend := newSourceBackend(Config{SourceBackend: "unknown", LogPath: "access.log"})
	if _, ok := unknownBackend.(*sentinelsource.FallbackBackend); !ok {
		t.Fatalf("expected fallback backend for unknown mode, got %T", unknownBackend)
	}
}

func TestEvictOverflow(t *testing.T) {
	now := time.Now().UTC()
	st := State{
		IPs: map[string]Record{
			MakeScoreKey("*", "203.0.113.1"): {Score: 1, LastSeen: now.Add(-10 * time.Minute).Format(time.RFC3339)},
			MakeScoreKey("*", "203.0.113.2"): {Score: 9, LastSeen: now.Format(time.RFC3339)},
			MakeScoreKey("*", "203.0.113.3"): {Score: 5, LastSeen: now.Format(time.RFC3339)},
		},
	}
	changed := evictOverflow(&st, Config{MaxActiveIPs: 2}, now)
	if !changed {
		t.Fatal("expected overflow eviction to report change")
	}
	if len(st.IPs) != 2 {
		t.Fatalf("expected 2 records after eviction, got %d", len(st.IPs))
	}
	if _, ok := st.IPs[MakeScoreKey("*", "203.0.113.1")]; ok {
		t.Fatal("expected lowest score record to be evicted")
	}
}

func TestSaveAdaptiveRespectsMaxPublishedEntries(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "adaptive.json")
	now := time.Now().UTC()
	st := State{
		IPs: map[string]Record{
			MakeScoreKey("*", "203.0.113.10"): {Stage: "drop", Score: 11, TrustScore: 20, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)},
			MakeScoreKey("*", "203.0.113.11"): {Stage: "drop", Score: 9, TrustScore: 25, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)},
			MakeScoreKey("*", "203.0.113.12"): {Stage: "throttle", Score: 7, TrustScore: 40, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)},
		},
	}
	cfg := Config{
		ThrottleRatePerSecond: 3,
		ThrottleBurst:         6,
		ThrottleTarget:        "REJECT",
		MaxPublishedEntries:   2,
	}
	if err := SaveAdaptive(path, cfg, st, now); err != nil {
		t.Fatalf("save adaptive: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read adaptive: %v", err)
	}
	var out AdaptiveOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode adaptive: %v", err)
	}
	if len(out.Entries) != 2 {
		t.Fatalf("expected 2 entries due to max_published_entries, got %d", len(out.Entries))
	}
}

func TestSaveAdaptiveIfChangedHonorsPublishInterval(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "adaptive.json")
	now := time.Now().UTC()
	st := State{
		IPs: map[string]Record{
			MakeScoreKey("*", "203.0.113.10"): {Stage: "drop", Score: 11, TrustScore: 20, FirstSeen: now.Add(-time.Minute).Format(time.RFC3339), LastSeen: now.Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)},
		},
	}
	cfg := Config{
		ThrottleRatePerSecond: 3,
		ThrottleBurst:         6,
		ThrottleTarget:        "REJECT",
		MaxPublishedEntries:   100,
	}
	var lastWrite time.Time
	wrote, err := SaveAdaptiveIfChanged(path, cfg, st, now, 5*time.Second, &lastWrite)
	if err != nil {
		t.Fatalf("first save adaptive if changed: %v", err)
	}
	if !wrote {
		t.Fatal("expected first adaptive write")
	}

	st.IPs[MakeScoreKey("*", "203.0.113.10")] = Record{Stage: "drop", Score: 15, TrustScore: 10, FirstSeen: now.Add(-2 * time.Minute).Format(time.RFC3339), LastSeen: now.Add(time.Second).Format(time.RFC3339), ExpiresAt: now.Add(2 * time.Minute).Format(time.RFC3339)}
	wrote, err = SaveAdaptiveIfChanged(path, cfg, st, now.Add(2*time.Second), 5*time.Second, &lastWrite)
	if err != nil {
		t.Fatalf("second save adaptive if changed: %v", err)
	}
	if wrote {
		t.Fatal("expected write to be skipped by publish interval")
	}

	wrote, err = SaveAdaptiveIfChanged(path, cfg, st, now.Add(6*time.Second), 5*time.Second, &lastWrite)
	if err != nil {
		t.Fatalf("third save adaptive if changed: %v", err)
	}
	if !wrote {
		t.Fatal("expected write after publish interval elapsed")
	}
}

func TestSaveSuggestionsIfChangedWritesCandidates(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "l7-suggestions.json")
	now := time.Now().UTC()
	st := State{
		L7Suggestions: []RuleSuggestion{
			{
				ID:         "path-.env",
				PathPrefix: "/.env",
				Status:     "suggested",
				Hits:       30,
				UniqueIPs:  12,
				WouldBlock: 30,
				Source:     "tarinio-sentinel",
				FirstSeen:  now.Add(-time.Minute).Format(time.RFC3339),
				LastSeen:   now.Format(time.RFC3339),
			},
		},
	}
	var lastWrite time.Time
	wrote, err := SaveSuggestionsIfChanged(path, st, now, 3*time.Second, &lastWrite)
	if err != nil {
		t.Fatalf("save suggestions: %v", err)
	}
	if !wrote {
		t.Fatal("expected suggestions write")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read suggestions: %v", err)
	}
	var payload struct {
		Items []RuleSuggestion `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode suggestions: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].PathPrefix != "/.env" {
		t.Fatalf("unexpected suggestions payload: %+v", payload.Items)
	}
}

func TestIsSiteModelEnabledUsesRuntimeScope(t *testing.T) {
	cfg := Config{EnabledSiteIDs: []string{"control-plane-access", "waf.example.test"}}
	if !isSiteModelEnabled(cfg, "waf.example.test") {
		t.Fatal("expected host from runtime scope to be enabled")
	}
	if isSiteModelEnabled(cfg, "app.example.test") {
		t.Fatal("expected non-scoped site to be disabled")
	}
}

func TestBuildRuleSuggestionsAccumulatesShadowHits(t *testing.T) {
	now := time.Now().UTC()
	stats := map[string]*scannerPathStat{
		"/.env": {
			Hits:      20,
			IPs:       map[string]struct{}{"203.0.113.1": {}, "203.0.113.2": {}},
			FirstSeen: now.Add(-time.Minute),
			LastSeen:  now,
		},
	}
	previous := []RuleSuggestion{{PathPrefix: "/.env", Status: "shadow", ShadowHits: 5}}

	got := buildRuleSuggestions(Config{SuggestMinHits: 10, SuggestMinUniqueIPs: 2}, stats, previous, now)
	if len(got) != 1 {
		t.Fatalf("expected one suggestion, got %d", len(got))
	}
	if got[0].Status != "shadow" {
		t.Fatalf("expected shadow status, got %q", got[0].Status)
	}
	if got[0].ShadowHits != 25 {
		t.Fatalf("expected accumulated shadow hits, got %d", got[0].ShadowHits)
	}
}

func TestBuildRuleSuggestionsPipelinePromotions(t *testing.T) {
	now := time.Now().UTC()
	cfg := Config{
		SuggestMinHits:              10,
		SuggestMinUniqueIPs:         2,
		SuggestShadowPromoteHits:    15,
		SuggestTemporaryPromoteHits: 30,
		SuggestPermanentPromoteHits: 50,
		SuggestShadowMaxFPRate:      0.05,
		SuggestTemporaryHoldSeconds: 60,
		SuggestPermanentMinLifetime: 30 * time.Second,
	}
	stats := map[string]*scannerPathStat{
		"/.env": {
			Hits:      20,
			IPs:       map[string]struct{}{"203.0.113.1": {}, "203.0.113.2": {}},
			FirstSeen: now.Add(-2 * time.Minute),
			LastSeen:  now,
		},
	}
	suggestions := buildRuleSuggestions(cfg, stats, nil, now)
	if len(suggestions) != 1 || suggestions[0].Status != "shadow" {
		t.Fatalf("expected promote to shadow, got %+v", suggestions)
	}

	stats["/.env"].Hits = 35
	stats["/.env"].FirstSeen = now.Add(-3 * time.Minute)
	stats["/.env"].LastSeen = now.Add(5 * time.Second)
	suggestions = buildRuleSuggestions(cfg, stats, suggestions, now.Add(5*time.Second))
	if len(suggestions) != 1 || suggestions[0].Status != "temporary" {
		t.Fatalf("expected promote to temporary, got %+v", suggestions)
	}
	if suggestions[0].TemporaryUntil == "" {
		t.Fatal("expected temporary_until for temporary stage")
	}

	stats["/.env"].Hits = 25
	stats["/.env"].FirstSeen = now.Add(-4 * time.Minute)
	stats["/.env"].LastSeen = now.Add(70 * time.Second)
	suggestions = buildRuleSuggestions(cfg, stats, suggestions, now.Add(70*time.Second))
	if len(suggestions) != 1 || suggestions[0].Status != "permanent" {
		t.Fatalf("expected candidate promote to permanent, got %+v", suggestions)
	}
	if suggestions[0].PromotionReason == "" {
		t.Fatal("expected promotion reason for permanent candidate")
	}
}

func TestBuildRuleSuggestionsRollsBackOnHighShadowFalsePositiveRate(t *testing.T) {
	now := time.Now().UTC()
	cfg := Config{
		SuggestMinHits:              10,
		SuggestMinUniqueIPs:         2,
		SuggestShadowPromoteHits:    15,
		SuggestTemporaryPromoteHits: 25,
		SuggestPermanentPromoteHits: 50,
		SuggestShadowMaxFPRate:      0.05,
	}
	stats := map[string]*scannerPathStat{
		"/.env": {
			Hits:        30,
			SuccessHits: 20,
			IPs:         map[string]struct{}{"203.0.113.1": {}, "203.0.113.2": {}},
			FirstSeen:   now.Add(-time.Minute),
			LastSeen:    now,
		},
	}
	previous := []RuleSuggestion{{PathPrefix: "/.env", Status: "shadow", ShadowHits: 30, ShadowFP: 0}}
	suggestions := buildRuleSuggestions(cfg, stats, previous, now)
	if len(suggestions) != 1 {
		t.Fatalf("expected one suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Status != "suggested" {
		t.Fatalf("expected rollback to suggested on high fp rate, got %q", suggestions[0].Status)
	}
}

func TestBuildRuleSuggestionsPrunesStaleSuggestions(t *testing.T) {
	now := time.Now().UTC()
	cfg := Config{
		SuggestMinHits:            10,
		SuggestMinUniqueIPs:       2,
		SuggestInactiveTTLSeconds: 60,
	}
	previous := []RuleSuggestion{
		{PathPrefix: "/.env", Status: "shadow", LastSeen: now.Add(-2 * time.Minute).Format(time.RFC3339)},
		{PathPrefix: "/wp-admin", Status: "suggested", LastSeen: now.Add(-20 * time.Second).Format(time.RFC3339)},
	}
	got := buildRuleSuggestions(cfg, nil, previous, now)
	if len(got) != 1 || got[0].PathPrefix != "/wp-admin" {
		t.Fatalf("expected stale suggestion to be pruned, got %+v", got)
	}
}

func TestBuildRuleSuggestionsPreservesPreviousOnQuietTick(t *testing.T) {
	now := time.Now().UTC()
	previous := []RuleSuggestion{
		{
			ID:          "path-env",
			PathPrefix:  "/.env",
			Status:      "suggested",
			Hits:        24,
			UniqueIPs:   6,
			GeneratedAt: now.Add(-time.Minute).Format(time.RFC3339),
		},
	}

	got := buildRuleSuggestions(Config{SuggestMinHits: 10, SuggestMinUniqueIPs: 2, SuggestInactiveTTLSeconds: 3600}, nil, previous, now)
	if len(got) != 1 {
		t.Fatalf("expected previous suggestion to be preserved, got %d", len(got))
	}
	if got[0].PathPrefix != "/.env" {
		t.Fatalf("unexpected preserved suggestion: %+v", got[0])
	}
	previous[0].PathPrefix = "/changed"
	if got[0].PathPrefix != "/.env" {
		t.Fatal("expected preserved suggestions to be copied")
	}
}

func TestConsumeActionBudgetRespectsLimitAndWindowReset(t *testing.T) {
	now := time.Now().UTC()
	st := State{}
	cfg := Config{MaxActionsPerMinute: 2}

	if !consumeActionBudget(&st, cfg, now) {
		t.Fatal("expected first action in window")
	}
	if !consumeActionBudget(&st, cfg, now.Add(10*time.Second)) {
		t.Fatal("expected second action in window")
	}
	if consumeActionBudget(&st, cfg, now.Add(20*time.Second)) {
		t.Fatal("expected third action to be rejected by max actions per minute")
	}
	if !consumeActionBudget(&st, cfg, now.Add(70*time.Second)) {
		t.Fatal("expected action budget reset after one minute")
	}
}

func TestNewSourceBackendRedisFallsBackToFile(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "access.log")
	line := `{"timestamp":"2026-04-26T12:00:00Z","client_ip":"203.0.113.44","site":"control-plane-access","status":404,"method":"GET","uri":"/.env","user_agent":"sqlmap"}` + "\n"
	if err := os.WriteFile(logPath, []byte(line), 0o644); err != nil {
		t.Fatalf("write access log: %v", err)
	}

	cfg := Config{
		SourceBackend: "redis",
		LogPath:       logPath,
	}
	items, nextOffset, err := readNewEvents(newSourceBackend(cfg), 0)
	if err != nil {
		t.Fatalf("read events with redis fallback: %v", err)
	}
	if nextOffset <= 0 {
		t.Fatalf("expected next offset > 0, got %d", nextOffset)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 parsed event, got %d", len(items))
	}
	if items[0].ip != "203.0.113.44" {
		t.Fatalf("unexpected event ip: %+v", items[0])
	}
}

func TestRenderAdaptivePayload_ExplainabilityFields(t *testing.T) {
	now := time.Now().UTC()
	st := State{
		IPs: map[string]Record{
			MakeScoreKey("*", "203.0.113.10"): {
				Stage:      "drop",
				Score:      11,
				TrustScore: 20,
				ExpiresAt:  now.Add(time.Minute).Format(time.RFC3339),
				TopSignals: map[string]float64{
					"signal_rps":           2.1,
					"signal_scanner_paths": 1.4,
					"signal_ua_risk":       0.8,
				},
			},
			MakeScoreKey("site-a", "203.0.113.10"): {
				Stage:      "drop",
				Score:      10,
				TrustScore: 22,
				ExpiresAt:  now.Add(time.Minute).Format(time.RFC3339),
				TopSignals: map[string]float64{"signal_blocked_ratio": 0.9},
			},
		},
	}
	cfg := Config{
		ThrottleRatePerSecond: 3,
		ThrottleBurst:         6,
		ThrottleTarget:        "REJECT",
	}
	raw, err := renderAdaptivePayload(cfg, st, now)
	if err != nil {
		t.Fatalf("render adaptive payload: %v", err)
	}
	var out AdaptiveOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode adaptive payload: %v", err)
	}
	if len(out.Entries) != 1 {
		t.Fatalf("expected one published entry, got %d", len(out.Entries))
	}
	entry := out.Entries[0]
	if entry.ExplainSummary == "" {
		t.Fatal("expected explain_summary to be populated")
	}
	if len(entry.ReasonCodes) == 0 || len(entry.ReasonDetails) == 0 {
		t.Fatalf("expected reason codes/details, got %+v", entry)
	}
	if len(entry.Recommendations) == 0 {
		t.Fatal("expected recommendations in adaptive entry")
	}
}

func TestResolveActionTransitionRequiresConsecutiveTicks(t *testing.T) {
	cfg := Config{PromotionConsecutiveTicks: 2}
	rec := Record{}

	action, transitioned := resolveActionTransition(&rec, "watch", "drop", cfg, false)
	if transitioned {
		t.Fatal("expected first promotion tick to stay pending")
	}
	if action != "watch" {
		t.Fatalf("expected action to remain watch, got %q", action)
	}
	if rec.CandidateAction != "drop" || rec.CandidateCount != 1 {
		t.Fatalf("unexpected pending candidate state: %+v", rec)
	}

	action, transitioned = resolveActionTransition(&rec, "watch", "drop", cfg, false)
	if !transitioned {
		t.Fatal("expected transition on second consistent tick")
	}
	if action != "drop" {
		t.Fatalf("expected drop action, got %q", action)
	}
	if rec.CandidateAction != "" || rec.CandidateCount != 0 {
		t.Fatalf("expected candidate state reset, got %+v", rec)
	}
}

func TestResolveActionTransitionEmergencyBypassesPendingTicks(t *testing.T) {
	cfg := Config{PromotionConsecutiveTicks: 3}
	rec := Record{}

	action, transitioned := resolveActionTransition(&rec, "throttle", "drop", cfg, true)
	if !transitioned {
		t.Fatal("expected emergency transition to bypass pending ticks")
	}
	if action != "drop" {
		t.Fatalf("expected drop action, got %q", action)
	}
}
