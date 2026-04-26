package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"waf/control-plane/internal/sites"
)

func TestSentinelBanSyncServicePromotesToManualBans(t *testing.T) {
	tmp := t.TempDir()
	adaptivePath := filepath.Join(tmp, "adaptive.json")
	statePath := filepath.Join(tmp, "state.json")

	payload := map[string]any{
		"entries": []map[string]any{
			{"ip": "203.0.113.10", "action": "drop", "score": 15.0},
			{"ip": "203.0.113.11", "action": "temp_ban", "score": 12.5},
			{"ip": "203.0.113.12", "action": "throttle", "score": 30.0},
			{"ip": "203.0.113.13", "action": "drop", "score": 5.0},
		},
	}
	if err := writeSentinelAdaptiveFixture(adaptivePath, payload); err != nil {
		t.Fatalf("write adaptive fixture: %v", err)
	}

	store := &fakeAccessPolicyStore{}
	manualBans := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{
		{ID: "site-a"},
		{ID: "site-b"},
	}}, nil)
	service := NewSentinelBanSyncService(manualBans, adaptivePath, statePath, 0, 10, 10)

	if err := service.syncOnce(context.Background()); err != nil {
		t.Fatalf("sync once failed: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list access policies: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected policies for both sites, got %d", len(items))
	}
	for _, item := range items {
		if len(item.DenyList) != 2 {
			t.Fatalf("expected 2 promoted bans in %+v", item)
		}
		if !containsString(item.DenyList, "203.0.113.10") || !containsString(item.DenyList, "203.0.113.11") {
			t.Fatalf("expected promoted deny entries in %+v", item.DenyList)
		}
	}

	if err := service.syncOnce(context.Background()); err != nil {
		t.Fatalf("second sync once failed: %v", err)
	}
	items, err = store.List()
	if err != nil {
		t.Fatalf("list access policies after second sync: %v", err)
	}
	for _, item := range items {
		if len(item.DenyList) != 2 {
			t.Fatalf("expected idempotent promotions without duplicates, got %+v", item.DenyList)
		}
	}
}

func TestSentinelBanSyncServiceHonorsMaxPromotionsPerTick(t *testing.T) {
	tmp := t.TempDir()
	adaptivePath := filepath.Join(tmp, "adaptive.json")

	payload := map[string]any{
		"entries": []map[string]any{
			{"ip": "203.0.113.50", "action": "drop", "score": 20.0},
			{"ip": "203.0.113.51", "action": "drop", "score": 19.0},
			{"ip": "203.0.113.52", "action": "drop", "score": 18.0},
		},
	}
	if err := writeSentinelAdaptiveFixture(adaptivePath, payload); err != nil {
		t.Fatalf("write adaptive fixture: %v", err)
	}

	store := &fakeAccessPolicyStore{}
	manualBans := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)
	service := NewSentinelBanSyncService(manualBans, adaptivePath, "", 0, 10, 2)

	if err := service.syncOnce(context.Background()); err != nil {
		t.Fatalf("sync once failed: %v", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("list access policies: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected single policy, got %d", len(items))
	}
	if len(items[0].DenyList) != 2 {
		t.Fatalf("expected max 2 promotions, got %+v", items[0].DenyList)
	}
	if !containsString(items[0].DenyList, "203.0.113.50") || !containsString(items[0].DenyList, "203.0.113.51") {
		t.Fatalf("expected top-score ips promoted first, got %+v", items[0].DenyList)
	}
}

func writeSentinelAdaptiveFixture(path string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
