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

func isPathWithinStagingRoot(path, root string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(absRoot), filepath.Clean(absPath))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
	if !isPathWithinStagingRoot(candidateRoot, filepath.Join(s.Root, "candidates")) {
		return "", errors.New("candidate path escapes staging root")
	}
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
		if !isPathWithinStagingRoot(targetPath, candidateRoot) {
			return "", fmt.Errorf("staged file path %q escapes candidate root", file.Path)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return "", fmt.Errorf("create staged file directory for %s: %w", file.Path, err)
		}
		if !isStagingDirectorySafe(filepath.Dir(targetPath), candidateRoot) {
			return "", fmt.Errorf("staged file directory for %s is outside candidate root", file.Path)
		}
		if err := os.WriteFile(targetPath, file.Content, 0o644); err != nil {
			return "", fmt.Errorf("write staged file %s: %w", file.Path, err)
		}
	}

	return candidateRoot, nil
}

// isStagingDirectorySafe rejects symlink traversal below candidateRoot without
// resolving the root itself. The root is created by Stage; resolving it through
// EvalSymlinks is not portable on Windows volumes that deny reparse-point
// inspection even for ordinary directories.
func isStagingDirectorySafe(directory, candidateRoot string) bool {
	if !isPathWithinStagingRoot(directory, candidateRoot) {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(candidateRoot), filepath.Clean(directory))
	if err != nil || rel == "." {
		return err == nil
	}
	current := candidateRoot
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}
	return true
}
