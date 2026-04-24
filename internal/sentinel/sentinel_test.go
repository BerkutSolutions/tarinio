package sentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	got := buildRuleSuggestions(Config{SuggestMinHits: 10, SuggestMinUniqueIPs: 2}, nil, previous, now)
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
