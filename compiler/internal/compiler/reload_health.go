package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HealthChecker isolates immediate post-reload health evaluation.
type HealthChecker interface {
	Check(active *ActivePointer) error
}

// ReloadHealthResult is the explicit outcome of reload and immediate health evaluation.
type ReloadHealthResult struct {
	RevisionID           string
	ReloadSucceeded      bool
	HealthCheckSucceeded bool
	ReloadError          error
	HealthCheckError     error
}

// ReloadHealthRunner executes runtime reload and immediate post-reload health
// evaluation without rollback or control-plane orchestration.
type ReloadHealthRunner struct {
	NginxBinary   string
	Executor      CommandExecutor
	HealthChecker HealthChecker
}

// Run executes reload first and health-check only if reload succeeded.
func (r ReloadHealthRunner) Run(active *ActivePointer) ReloadHealthResult {
	result := ReloadHealthResult{}
	if active == nil {
		result.ReloadError = errors.New("active pointer is required")
		return result
	}
	if strings.TrimSpace(active.RevisionID) == "" {
		result.ReloadError = errors.New("active revision id is required")
		return result
	}
	if strings.TrimSpace(active.CandidatePath) == "" {
		result.ReloadError = errors.New("active candidate path is required")
		return result
	}
	if r.Executor == nil {
		result.ReloadError = errors.New("command executor is required")
		return result
	}
	if r.HealthChecker == nil {
		result.ReloadError = errors.New("health checker is required")
		return result
	}

	result.RevisionID = active.RevisionID
	nginxBinary := strings.TrimSpace(r.NginxBinary)
	if nginxBinary == "" {
		nginxBinary = "nginx"
	}

	nginxRoot := filepath.Join(active.CandidatePath, "nginx")
	args := []string{"-p", nginxRoot, "-c", "nginx.conf", "-s", "reload"}
	if err := r.Executor.Run(nginxBinary, args, nginxRoot); err != nil {
		result.ReloadError = fmt.Errorf("runtime reload failed: %w", err)
		return result
	}
	result.ReloadSucceeded = true

	if err := r.HealthChecker.Check(active); err != nil {
		result.HealthCheckError = fmt.Errorf("post-reload health-check failed: %w", err)
		return result
	}
	result.HealthCheckSucceeded = true

	return result
}

// LoadActivePointer reads the current activation pointer from disk.
func LoadActivePointer(root string) (*ActivePointer, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("activation root is required")
	}

	content, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("active pointer is missing")
		}
		return nil, fmt.Errorf("read active pointer: %w", err)
	}

	var pointer ActivePointer
	if err := json.Unmarshal(content, &pointer); err != nil {
		return nil, fmt.Errorf("decode active pointer: %w", err)
	}
	if strings.TrimSpace(pointer.RevisionID) == "" {
		return nil, errors.New("active pointer revision id is required")
	}
	if strings.TrimSpace(pointer.CandidatePath) == "" {
		return nil, errors.New("active pointer candidate path is required")
	}

	return &pointer, nil
}
