package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ValidateRevisionBundle checks manifest presence, artifact presence, per-file
// checksums, and overall bundle checksum using only bundle files.
func ValidateRevisionBundle(bundle *RevisionBundle) error {
	if bundle == nil {
		return errors.New("bundle is required")
	}
	if len(bundle.Files) == 0 {
		return errors.New("bundle files are required")
	}

	fileByPath := make(map[string]BundleFile, len(bundle.Files))
	var manifestFile BundleFile
	manifestFound := false
	for _, file := range bundle.Files {
		if strings.TrimSpace(file.Path) == "" {
			return errors.New("bundle file path is required")
		}
		if !isSafeBundlePath(file.Path) {
			return fmt.Errorf("bundle file path %q escapes the staging root", file.Path)
		}
		if _, exists := fileByPath[file.Path]; exists {
			return fmt.Errorf("duplicate bundle file path %s", file.Path)
		}
		fileByPath[file.Path] = file
		if file.Path == "manifest.json" {
			manifestFile = file
			manifestFound = true
		}
	}
	if !manifestFound {
		return errors.New("manifest.json is required")
	}
	if !isSafeBundleIdentifier(bundle.Revision.ID) {
		return errors.New("bundle revision id must be a safe identifier")
	}

	var manifest RevisionManifest
	if err := json.Unmarshal(manifestFile.Content, &manifest); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.SchemaVersion != manifestSchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %s", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.RevisionID) == "" {
		return errors.New("manifest revision_id is required")
	}
	if manifest.RevisionVersion <= 0 {
		return errors.New("manifest revision_version must be positive")
	}
	if strings.TrimSpace(manifest.CreatedAt) == "" {
		return errors.New("manifest created_at is required")
	}
	if strings.TrimSpace(manifest.BundleChecksum) == "" {
		return errors.New("manifest bundle_checksum is required")
	}
	if len(manifest.Contents) == 0 {
		return errors.New("manifest contents are required")
	}

	contents := append([]ManifestContent(nil), manifest.Contents...)
	sort.Slice(contents, func(i, j int) bool {
		return contents[i].Path < contents[j].Path
	})

	seenContentPaths := make(map[string]struct{}, len(contents))
	for _, entry := range contents {
		if strings.TrimSpace(entry.Path) == "" {
			return errors.New("manifest content path is required")
		}
		if !isSafeBundlePath(entry.Path) {
			return fmt.Errorf("manifest content path %q escapes the staging root", entry.Path)
		}
		if _, exists := seenContentPaths[entry.Path]; exists {
			return fmt.Errorf("duplicate manifest content path %s", entry.Path)
		}
		seenContentPaths[entry.Path] = struct{}{}
		if strings.TrimSpace(entry.Checksum) == "" {
			return fmt.Errorf("manifest content checksum is required for %s", entry.Path)
		}
		file, ok := fileByPath[entry.Path]
		if !ok {
			return fmt.Errorf("manifest content %s is missing from bundle", entry.Path)
		}
		if checksumBytes(file.Content) != entry.Checksum {
			return fmt.Errorf("checksum mismatch for %s", entry.Path)
		}
	}

	for path := range fileByPath {
		if path == "manifest.json" {
			continue
		}
		if _, ok := seenContentPaths[path]; !ok {
			return fmt.Errorf("bundle file %s is not declared in manifest", path)
		}
	}

	if checksumContents(contents) != manifest.BundleChecksum {
		return errors.New("bundle checksum mismatch")
	}

	return nil
}

func isSafeBundleIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isSafeBundlePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.ContainsRune(value, '\\') {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func checksumBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
