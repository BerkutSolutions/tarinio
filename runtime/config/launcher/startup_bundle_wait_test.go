package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartupBundleWaitDurationFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  time.Duration
		bad   bool
	}{
		{name: "disabled when empty", value: "", want: 0},
		{name: "disabled when zero", value: "0", want: 0},
		{name: "enabled", value: "15", want: 15 * time.Second},
		{name: "rejects invalid", value: "soon", bad: true},
		{name: "rejects excessive", value: "3601", bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(runtimeStartupBundleWaitEnv, tc.value)
			got, err := startupBundleWaitDurationFromEnv()
			if tc.bad {
				if err == nil {
					t.Fatal("expected invalid setting to be rejected")
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got duration=%s err=%v, want %s", got, err, tc.want)
			}
		})
	}
}

func TestWaitForInitialBundleAcceptsCompletePublishedCandidate(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidates", "rev-000001")
	go func() {
		time.Sleep(20 * time.Millisecond)
		for _, relative := range []string{"manifest.json", "nginx/nginx.conf", "modsecurity/modsecurity.conf"} {
			path := filepath.Join(candidate, relative)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return
			}
			if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
				return
			}
		}
		active := filepath.Join(root, "active")
		if err := os.MkdirAll(active, 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(active, "current.json"), []byte(`{"revision_id":"rev-000001","candidate_path":"candidates/rev-000001"}`), 0o600)
		}
	}()

	if err := waitForInitialBundle(root, time.Second); err != nil {
		t.Fatalf("waitForInitialBundle: %v", err)
	}
}

func TestWaitForInitialBundleWaitsForCandidateAfterPointer(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidates", "rev-000001")
	active := filepath.Join(root, "active")
	if err := os.MkdirAll(active, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(active, "current.json"), []byte(`{"revision_id":"rev-000001","candidate_path":"candidates/rev-000001"}`), 0o600); err != nil {
		t.Fatalf("write active pointer: %v", err)
	}

	go func() {
		time.Sleep(120 * time.Millisecond)
		for _, relative := range []string{"manifest.json", "nginx/nginx.conf", "modsecurity/modsecurity.conf"} {
			path := filepath.Join(candidate, relative)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return
			}
			if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
				return
			}
		}
	}()

	if err := waitForInitialBundle(root, time.Second); err != nil {
		t.Fatalf("waitForInitialBundle after early pointer: %v", err)
	}
}

func TestWaitForInitialBundleTimesOutWithoutPointer(t *testing.T) {
	if err := waitForInitialBundle(t.TempDir(), 20*time.Millisecond); err == nil {
		t.Fatal("expected timeout when no initial bundle is published")
	}
}
