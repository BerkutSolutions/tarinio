package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ActivePointer is the stable active reference written by the activation step.
type ActivePointer struct {
	RevisionID    string `json:"revision_id"`
	CandidatePath string `json:"candidate_path"`
}

// AtomicActivator switches the active reference to an already staged
// candidate bundle without copying files or touching runtime.
type AtomicActivator struct {
	Root string
}

// Activate updates the active pointer to candidates/<revision-id> atomically.
func (a AtomicActivator) Activate(revisionID string) (*ActivePointer, error) {
	if strings.TrimSpace(a.Root) == "" {
		return nil, errors.New("activation root is required")
	}
	if strings.TrimSpace(revisionID) == "" {
		return nil, errors.New("revision id is required")
	}

	candidatePath := filepath.Join(a.Root, "candidates", revisionID)
	if _, err := os.Stat(candidatePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("candidate %s not found", revisionID)
		}
		return nil, fmt.Errorf("stat candidate %s: %w", revisionID, err)
	}
	if _, err := os.Stat(filepath.Join(candidatePath, "manifest.json")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("candidate %s is missing manifest.json", revisionID)
		}
		return nil, fmt.Errorf("stat candidate manifest for %s: %w", revisionID, err)
	}

	activeDir := filepath.Join(a.Root, "active")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create active directory: %w", err)
	}

	pointer := &ActivePointer{
		RevisionID:    revisionID,
		CandidatePath: candidatePath,
	}
	content, err := json.MarshalIndent(pointer, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal active pointer: %w", err)
	}
	content = append(content, '\n')

	targetPath := filepath.Join(activeDir, "current.json")
	tempPath := filepath.Join(activeDir, fmt.Sprintf(".current.%s.tmp", revisionID))
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return nil, fmt.Errorf("write temp active pointer: %w", err)
	}
	if err := replaceFileAtomically(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("activate revision %s: %w", revisionID, err)
	}

	return pointer, nil
}
