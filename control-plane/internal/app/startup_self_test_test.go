package app

import (
	"context"
	"testing"

	"waf/control-plane/internal/config"
)

func TestRunStartupSelfTest_SucceedsWhenEnabled(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:         "127.0.0.1:8080",
		RuntimeRoot:      "/tmp/runtime",
		RevisionStoreDir: t.TempDir(),
		AuthIssuer:       "WAF",
		StartupSelfTest:  true,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Enabled:  true,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@example.test",
			Password: "admin",
		},
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("app bootstrap failed: %v", err)
	}

	if err := application.RunStartupSelfTest(context.Background()); err != nil {
		t.Fatalf("startup self-test failed: %v", err)
	}
}

func TestRunStartupSelfTest_NoOpWhenDisabled(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:         "127.0.0.1:8080",
		RuntimeRoot:      "/tmp/runtime",
		RevisionStoreDir: t.TempDir(),
		AuthIssuer:       "WAF",
		StartupSelfTest:  false,
		BootstrapAdmin: config.BootstrapAdminConfig{
			Enabled:  true,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@example.test",
			Password: "admin",
		},
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("app bootstrap failed: %v", err)
	}

	if err := application.RunStartupSelfTest(context.Background()); err != nil {
		t.Fatalf("startup self-test should be skipped, got: %v", err)
	}
}
