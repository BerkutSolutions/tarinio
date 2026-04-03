package compiler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CandidateStager writes a validated revision bundle into a deterministic
// candidate directory without activating it.
type CandidateStager struct {
	Root string
}

// Stage materializes the bundle in candidates/<revision-id> and returns the
// staged candidate path.
func (s CandidateStager) Stage(bundle *RevisionBundle) (string, error) {
	if strings.TrimSpace(s.Root) == "" {
		return "", errors.New("staging root is required")
	}
	if bundle == nil {
		return "", errors.New("bundle is required")
	}
	if err := ValidateRevisionBundle(bundle); err != nil {
		return "", err
	}
	if strings.TrimSpace(bundle.Revision.ID) == "" {
		return "", errors.New("bundle revision id is required")
	}

	candidateRoot := filepath.Join(s.Root, "candidates", bundle.Revision.ID)
	if err := os.RemoveAll(candidateRoot); err != nil {
		return "", fmt.Errorf("reset candidate directory: %w", err)
	}
	if err := os.MkdirAll(candidateRoot, 0o755); err != nil {
		return "", fmt.Errorf("create candidate directory: %w", err)
	}

	files := append([]BundleFile(nil), bundle.Files...)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	for _, file := range files {
		targetPath := filepath.Join(candidateRoot, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return "", fmt.Errorf("create staged file directory for %s: %w", file.Path, err)
		}
		if err := os.WriteFile(targetPath, file.Content, 0o644); err != nil {
			return "", fmt.Errorf("write staged file %s: %w", file.Path, err)
		}
	}

	return candidateRoot, nil
}
