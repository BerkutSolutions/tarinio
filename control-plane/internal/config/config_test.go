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
}

func TestLoadFromEnv_Overrides(t *testing.T) {
	t.Setenv("CONTROL_PLANE_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("CONTROL_PLANE_REVISION_STORE_DIR", "/tmp/control-plane")
	t.Setenv("WAF_RUNTIME_ROOT", "/tmp/runtime")
	t.Setenv("CONTROL_PLANE_REDIS_ADDR", "127.0.0.1:6380")
	t.Setenv("CONTROL_PLANE_REDIS_DB", "2")
	t.Setenv("CONTROL_PLANE_STARTUP_SELF_TEST_ENABLED", "false")
	t.Setenv("CONTROL_PLANE_SECURITY_PEPPER", "test-pepper")

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
