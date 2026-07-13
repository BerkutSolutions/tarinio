package managementhosts

import (
	"os"
	"testing"
)

func TestStoreNormalizesAndUsesOptimisticConcurrency(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update([]string{"PREWAF.example.", "192.0.2.4"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := updated.Hosts[0], "192.0.2.4"; got != want {
		t.Fatalf("host = %q, want %q", got, want)
	}
	if _, err := store.Update([]string{"next.example"}, 0); err == nil {
		t.Fatal("expected version conflict")
	}
}

func TestStoreReportsSetupRequiredBeforeMigration(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := store.Get()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.SetupRequired {
		t.Fatal("expected setup-required state for ambiguous legacy installation")
	}
}

func TestBootstrapIsIdempotentAndSurvivesStoreRestart(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, migrated, err := store.Bootstrap("PANEL.example.")
	if err != nil || !migrated {
		t.Fatalf("first migration: item=%+v migrated=%v err=%v", first, migrated, err)
	}
	second, migrated, err := store.Bootstrap("other.example")
	if err != nil || migrated {
		t.Fatalf("second migration must be noop: item=%+v migrated=%v err=%v", second, migrated, err)
	}
	restarted, err := NewStore(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := restarted.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Hosts) != 1 || stored.Hosts[0] != "panel.example" || !stored.Migrated {
		t.Fatalf("persisted migration lost after restart: %+v", stored)
	}
}

func TestBootstrapLegacyEnvironmentSetAndEmpty(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_HOST", "legacy.panel.example")
	configured, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	item, migrated, err := configured.Bootstrap(os.Getenv("CONTROL_PLANE_DEV_FAST_START_HOST"))
	if err != nil || !migrated || len(item.Hosts) != 1 || item.Hosts[0] != "legacy.panel.example" {
		t.Fatalf("legacy env migration: item=%+v migrated=%v err=%v", item, migrated, err)
	}
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_HOST", "")
	ambiguous, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	item, migrated, err = ambiguous.Bootstrap(os.Getenv("CONTROL_PLANE_DEV_FAST_START_HOST"))
	if err != nil || migrated || !item.SetupRequired {
		t.Fatalf("empty legacy env must require guided setup: item=%+v migrated=%v err=%v", item, migrated, err)
	}
}

func TestBootstrapPreservesLegacyLocalhost(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	item, migrated, err := store.Bootstrap("LOCALHOST")
	if err != nil || !migrated || len(item.Hosts) != 1 || item.Hosts[0] != "localhost" {
		t.Fatalf("localhost migration: item=%+v migrated=%v err=%v", item, migrated, err)
	}
}
