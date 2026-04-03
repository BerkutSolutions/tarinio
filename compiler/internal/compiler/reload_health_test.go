package compiler

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeHealthChecker struct {
	active *ActivePointer
	err    error
	calls  int
}

func (f *fakeHealthChecker) Check(active *ActivePointer) error {
	f.active = active
	f.calls++
	return f.err
}

func TestReloadHealthRunner_RunSuccess(t *testing.T) {
	exec := &fakeCommandExecutor{}
	health := &fakeHealthChecker{}
	runner := ReloadHealthRunner{
		NginxBinary:   "nginx-test",
		Executor:      exec,
		HealthChecker: health,
	}

	active := &ActivePointer{
		RevisionID:    "rev-001",
		CandidatePath: filepath.Join("c:", "staging", "candidates", "rev-001"),
	}
	result := runner.Run(active)

	if !result.ReloadSucceeded {
		t.Fatal("expected reload success")
	}
	if !result.HealthCheckSucceeded {
		t.Fatal("expected health-check success")
	}
	if exec.calls != 1 {
		t.Fatalf("expected one reload call, got %d", exec.calls)
	}
	if health.calls != 1 {
		t.Fatalf("expected one health-check call, got %d", health.calls)
	}
	if exec.name != "nginx-test" {
		t.Fatalf("unexpected binary: %s", exec.name)
	}
	if exec.args[0] != "-p" || exec.args[2] != "-c" || exec.args[4] != "-s" || exec.args[5] != "reload" {
		t.Fatalf("unexpected reload args: %#v", exec.args)
	}
}

func TestReloadHealthRunner_ReloadFailureStopsHealthCheck(t *testing.T) {
	exec := &fakeCommandExecutor{err: errors.New("exit status 1")}
	health := &fakeHealthChecker{}
	runner := ReloadHealthRunner{
		Executor:      exec,
		HealthChecker: health,
	}

	result := runner.Run(&ActivePointer{
		RevisionID:    "rev-001",
		CandidatePath: filepath.Join("candidates", "rev-001"),
	})

	if result.ReloadSucceeded {
		t.Fatal("did not expect reload success")
	}
	if result.ReloadError == nil {
		t.Fatal("expected reload error")
	}
	if health.calls != 0 {
		t.Fatal("health-check should not run after reload failure")
	}
}

func TestReloadHealthRunner_HealthCheckFailure(t *testing.T) {
	exec := &fakeCommandExecutor{}
	health := &fakeHealthChecker{err: errors.New("unhealthy")}
	runner := ReloadHealthRunner{
		Executor:      exec,
		HealthChecker: health,
	}

	result := runner.Run(&ActivePointer{
		RevisionID:    "rev-001",
		CandidatePath: filepath.Join("candidates", "rev-001"),
	})

	if !result.ReloadSucceeded {
		t.Fatal("expected reload success")
	}
	if result.HealthCheckSucceeded {
		t.Fatal("did not expect health-check success")
	}
	if result.HealthCheckError == nil {
		t.Fatal("expected health-check error")
	}
}

func TestLoadActivePointer(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	content := []byte("{\n  \"revision_id\": \"rev-001\",\n  \"candidate_path\": \"candidates/rev-001\"\n}\n")
	if err := os.WriteFile(filepath.Join(activeDir, "current.json"), content, 0o644); err != nil {
		t.Fatalf("write active pointer failed: %v", err)
	}

	active, err := LoadActivePointer(root)
	if err != nil {
		t.Fatalf("load active pointer failed: %v", err)
	}
	if active.RevisionID != "rev-001" {
		t.Fatalf("unexpected revision id: %s", active.RevisionID)
	}
	if active.CandidatePath != "candidates/rev-001" {
		t.Fatalf("unexpected candidate path: %s", active.CandidatePath)
	}
}
