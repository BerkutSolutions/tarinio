package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	t.Setenv("CONTROL_PLANE_HTTP_ADDR", "")
	t.Setenv("WAF_RUNTIME_ROOT", "")
	t.Setenv("CONTROL_PLANE_REVISION_STORE_DIR", "")
	t.Setenv("CONTROL_PLANE_STARTUP_SELF_TEST_ENABLED", "")
	t.Setenv("CONTROL_PLANE_SECURITY_PEPPER", "test-pepper")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.HTTPAddr == "" || cfg.RuntimeRoot == "" || cfg.RevisionStoreDir == "" {
		t.Fatal("expected defaults to be populated")
	}
	if cfg.Redis.Addr != "" {
		t.Fatalf("expected redis to be optional by default, got %q", cfg.Redis.Addr)
	}
	if !cfg.StartupSelfTest {
		t.Fatal("expected startup self-test to be enabled by default")
	}
	if cfg.SentinelBanSync.Enabled {
		t.Fatal("expected sentinel ban sync to be disabled by default")
	}
}

func TestLoadFromEnv_Overrides(t *testing.T) {
	t.Setenv("CONTROL_PLANE_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("CONTROL_PLANE_REVISION_STORE_DIR", "/tmp/control-plane")
	t.Setenv("WAF_RUNTIME_ROOT", "/tmp/runtime")
	t.Setenv("CONTROL_PLANE_REDIS_ADDR", "127.0.0.1:6380")
	t.Setenv("CONTROL_PLANE_REDIS_DB", "2")
	t.Setenv("CONTROL_PLANE_STARTUP_SELF_TEST_ENABLED", "false")
	t.Setenv("CONTROL_PLANE_SECURITY_PEPPER", "test-pepper")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_ENABLED", "true")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_ADAPTIVE_PATH", "/tmp/adaptive.json")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_STATE_PATH", "/tmp/sentinel-sync-state.json")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_POLL_SECONDS", "7")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_MIN_SCORE", "11.5")
	t.Setenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_MAX_PROMOTIONS_PER_TICK", "8")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("unexpected addr: %s", cfg.HTTPAddr)
	}
	if cfg.RevisionStoreDir != "/tmp/control-plane" {
		t.Fatalf("unexpected revision store dir: %s", cfg.RevisionStoreDir)
	}
	if cfg.Redis.Addr != "127.0.0.1:6380" || cfg.Redis.DB != 2 {
		t.Fatalf("unexpected redis config: %+v", cfg.Redis)
	}
	if cfg.StartupSelfTest {
		t.Fatal("expected startup self-test to be disabled by env override")
	}
	if !cfg.SentinelBanSync.Enabled {
		t.Fatal("expected sentinel ban sync to be enabled")
	}
	if cfg.SentinelBanSync.AdaptivePath != "/tmp/adaptive.json" {
		t.Fatalf("unexpected sentinel adaptive path: %s", cfg.SentinelBanSync.AdaptivePath)
	}
	if cfg.SentinelBanSync.StatePath != "/tmp/sentinel-sync-state.json" {
		t.Fatalf("unexpected sentinel state path: %s", cfg.SentinelBanSync.StatePath)
	}
	if cfg.SentinelBanSync.PollIntervalSeconds != 7 || cfg.SentinelBanSync.MinScore != 11.5 || cfg.SentinelBanSync.MaxPromotionsPerTick != 8 {
		t.Fatalf("unexpected sentinel ban sync config: %+v", cfg.SentinelBanSync)
	}
}

func TestLoadFromEnv_InvalidAddr(t *testing.T) {
	t.Setenv("CONTROL_PLANE_HTTP_ADDR", "invalid")
	t.Setenv("CONTROL_PLANE_SECURITY_PEPPER", "test-pepper")
	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected invalid addr error")
	}
}

func TestLoadFromEnv_RequiresRedisOnlyInHA(t *testing.T) {
	t.Setenv("CONTROL_PLANE_HA_ENABLED", "true")
	t.Setenv("CONTROL_PLANE_REDIS_ADDR", "")
	t.Setenv("CONTROL_PLANE_SECURITY_PEPPER", "test-pepper")
	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected redis validation error when HA is enabled")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
