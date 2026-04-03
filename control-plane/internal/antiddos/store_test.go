package antiddos

import (
	"path/filepath"
	"testing"
)

func TestStore_GetReturnsDefaultsWhenEmpty(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	item, err := store.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defaults := DefaultSettings()
	if !item.UseL4Guard || item.ConnLimit != defaults.ConnLimit || item.RatePerSecond != defaults.RatePerSecond {
		t.Fatalf("unexpected defaults: %+v", item)
	}
}

func TestStore_UpsertPersistsSettings(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "antiddos"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	updated, err := store.Upsert(Settings{
		UseL4Guard:    true,
		ChainMode:     ChainModeInput,
		ConnLimit:     320,
		RatePerSecond: 160,
		RateBurst:     320,
		Ports:         []int{443, 8443},
		Target:        TargetReject,
		EnforceL7Rate: true,
		L7RequestsPS:  30,
		L7Burst:       60,
		L7StatusCode:  429,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if updated.CreatedAt == "" || updated.UpdatedAt == "" {
		t.Fatalf("expected timestamps, got %+v", updated)
	}
	reloaded, err := store.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if reloaded.ConnLimit != 320 || reloaded.RatePerSecond != 160 || reloaded.ChainMode != ChainModeInput {
		t.Fatalf("unexpected settings after reload: %+v", reloaded)
	}
}
