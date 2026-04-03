package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const manifestSchemaVersion = "v1"

// RevisionInput is the minimal control-plane revision identity required to
// assemble one self-contained runtime bundle.
type RevisionInput struct {
	ID        string
	Version   int
	CreatedAt string
}

// ManifestContent is one manifest entry describing a compiled bundle artifact.
type ManifestContent struct {
	Path     string       `json:"path"`
	Kind     ArtifactKind `json:"kind"`
	Checksum string       `json:"checksum"`
}

// RevisionManifest is the MVP manifest.json structure for one compiled bundle.
type RevisionManifest struct {
	SchemaVersion   string            `json:"schema_version"`
	RevisionID      string            `json:"revision_id"`
	RevisionVersion int               `json:"revision_version"`
	CreatedAt       string            `json:"created_at"`
	BundleChecksum  string            `json:"bundle_checksum"`
	Contents        []ManifestContent `json:"contents"`
}

// BundleFile is one concrete file in the assembled revision bundle.
type BundleFile struct {
	Path    string
	Content []byte
}

// RevisionBundle is one fully assembled, self-contained compiled revision.
type RevisionBundle struct {
	Revision RevisionInput
	Files    []BundleFile
	Manifest RevisionManifest
}

// AssembleRevisionBundle builds one deterministic revision bundle from the
// rendered compiler artifacts and manifest contract.
func AssembleRevisionBundle(revision RevisionInput, artifacts ...[]ArtifactOutput) (*RevisionBundle, error) {
	if strings.TrimSpace(revision.ID) == "" {
		return nil, errors.New("revision id is required")
	}
	if revision.Version <= 0 {
		return nil, errors.New("revision version must be positive")
	}
	if strings.TrimSpace(revision.CreatedAt) == "" {
		return nil, errors.New("revision created_at is required")
	}

	var flat []ArtifactOutput
	for _, group := range artifacts {
		flat = append(flat, group...)
	}
	if len(flat) == 0 {
		return nil, errors.New("at least one artifact is required")
	}

	sort.Slice(flat, func(i, j int) bool {
		return flat[i].Path < flat[j].Path
	})

	seenPaths := make(map[string]struct{}, len(flat))
	contents := make([]ManifestContent, 0, len(flat))
	files := make([]BundleFile, 0, len(flat)+1)
	for _, artifact := range flat {
		if strings.TrimSpace(artifact.Path) == "" {
			return nil, errors.New("artifact path is required")
		}
		if _, exists := seenPaths[artifact.Path]; exists {
			return nil, fmt.Errorf("duplicate artifact path %s", artifact.Path)
		}
		seenPaths[artifact.Path] = struct{}{}
		if strings.TrimSpace(artifact.Checksum) == "" {
			return nil, fmt.Errorf("artifact %s checksum is required", artifact.Path)
		}

		contents = append(contents, ManifestContent{
			Path:     artifact.Path,
			Kind:     artifact.Kind,
			Checksum: artifact.Checksum,
		})
		files = append(files, BundleFile{
			Path:    artifact.Path,
			Content: artifact.Content,
		})
	}

	manifest := RevisionManifest{
		SchemaVersion:   manifestSchemaVersion,
		RevisionID:      revision.ID,
		RevisionVersion: revision.Version,
		CreatedAt:       revision.CreatedAt,
		BundleChecksum:  checksumContents(contents),
		Contents:        contents,
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manifestJSON = append(manifestJSON, '\n')
	files = append(files, BundleFile{
		Path:    "manifest.json",
		Content: manifestJSON,
	})

	return &RevisionBundle{
		Revision: revision,
		Files:    files,
		Manifest: manifest,
	}, nil
}

func checksumContents(contents []ManifestContent) string {
	h := sha256.New()
	for _, entry := range contents {
		h.Write([]byte(entry.Path))
		h.Write([]byte{0})
		h.Write([]byte(entry.Checksum))
		h.Write([]byte{0})
		h.Write([]byte(entry.Kind))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
