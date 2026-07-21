package releaseartifacts

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateReleaseArtifacts(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	outputDir := filepath.Join(t.TempDir(), "release-3.0.0")

	result, err := Generate(Options{
		RepoRoot:   repoRoot,
		Version:    "3.0.0",
		CommitSHA:  "deadbeef",
		Tag:        "v3.0.0",
		OutputDir:  outputDir,
		DockerTags: []string{"tarinio:3.0.0", "ghcr.io/berkutsolutions/tarinio:3.0.0"},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	required := []string{
		result.ManifestPath,
		result.SignaturePath,
		result.SBOMPath,
		result.ProvenancePath,
		result.PublicKeyPath,
		filepath.Join(outputDir, "checksums.txt"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}

	var manifest map[string]any
	if err := decodeJSONFile(result.ManifestPath, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if got := manifest["format"]; got != "tarinio-release-artifacts/v1" {
		t.Fatalf("manifest format = %v", got)
	}
	if got := manifest["version"]; got != "3.0.0" {
		t.Fatalf("manifest version = %v", got)
	}
	sourceInputs, ok := manifest["source_inputs"].([]any)
	if !ok {
		t.Fatalf("manifest source_inputs missing or invalid: %#v", manifest["source_inputs"])
	}
	for _, item := range sourceInputs {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("manifest source_inputs entry invalid: %#v", item)
		}
		if row["path"] == "scripts/local-ci-preflight.ps1" {
			t.Fatalf("manifest should not publish local preflight script in source_inputs")
		}
	}
	generatedFiles, ok := manifest["generated_files"].([]any)
	if !ok || len(generatedFiles) < 4 {
		t.Fatalf("manifest generated_files missing or too short: %#v", manifest["generated_files"])
	}

	var sbom map[string]any
	if err := decodeJSONFile(result.SBOMPath, &sbom); err != nil {
		t.Fatalf("decode sbom: %v", err)
	}
	if got := sbom["bomFormat"]; got != "CycloneDX" {
		t.Fatalf("sbom bomFormat = %v", got)
	}
	trustedKeyPath := filepath.Join(t.TempDir(), "release-public-key.pem")
	trustedKey, err := os.ReadFile(result.PublicKeyPath)
	if err != nil {
		t.Fatalf("read generated public key: %v", err)
	}
	if err := os.WriteFile(trustedKeyPath, trustedKey, 0o600); err != nil {
		t.Fatalf("write trusted public key: %v", err)
	}
	if err := Verify(outputDir, trustedKeyPath); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyReleaseArtifactsFailsOnTamper(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	outputDir := filepath.Join(t.TempDir(), "release-3.0.0")

	if _, err := Generate(Options{
		RepoRoot:   repoRoot,
		Version:    "3.0.0",
		CommitSHA:  "deadbeef",
		Tag:        "v3.0.0",
		OutputDir:  outputDir,
		DockerTags: []string{"tarinio:3.0.0"},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "provenance.json"), []byte(`{"tampered":true}`), 0o644); err != nil {
		t.Fatalf("tamper provenance: %v", err)
	}
	trustedKeyPath := filepath.Join(t.TempDir(), "release-public-key.pem")
	trustedKey, err := os.ReadFile(filepath.Join(outputDir, "release-public-key.pem"))
	if err != nil {
		t.Fatalf("read generated public key: %v", err)
	}
	if err := os.WriteFile(trustedKeyPath, trustedKey, 0o600); err != nil {
		t.Fatalf("write trusted public key: %v", err)
	}
	if err := Verify(outputDir, trustedKeyPath); err == nil {
		t.Fatal("expected Verify() to fail after artifact tampering")
	}
}

func TestVerifyReleaseArtifactsRejectsArtifactProvidedReplacementKey(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	trustedRelease := filepath.Join(t.TempDir(), "trusted")
	if _, err := Generate(Options{RepoRoot: repoRoot, Version: "3.0.0", CommitSHA: "trusted", Tag: "v3.0.0", OutputDir: trustedRelease}); err != nil {
		t.Fatalf("generate trusted release: %v", err)
	}
	trustedKeyPath := filepath.Join(t.TempDir(), "trust.pem")
	trustedKey, err := os.ReadFile(filepath.Join(trustedRelease, "release-public-key.pem"))
	if err != nil {
		t.Fatalf("read trusted key: %v", err)
	}
	if err := os.WriteFile(trustedKeyPath, trustedKey, 0o600); err != nil {
		t.Fatalf("write trusted key: %v", err)
	}
	replacement := filepath.Join(t.TempDir(), "replacement")
	if err := os.MkdirAll(replacement, 0o755); err != nil {
		t.Fatalf("create replacement directory: %v", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate replacement key: %v", err)
	}
	manifest := []byte(`{"format":"tarinio-release-artifacts/v1","signing_key_id":"replacement"}`)
	signature := ed25519.Sign(privateKey, manifest)
	files := map[string][]byte{
		"release-manifest.json":  manifest,
		"checksums.txt":          nil,
		"sbom.cdx.json":          []byte(`{}`),
		"provenance.json":        []byte(`{}`),
		"release-public-key.pem": pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}),
		"signature.json":         []byte(fmt.Sprintf(`{"algorithm":"ed25519","key_id":"replacement","signed_object":"release-manifest.json","manifest_sha256":"%s","signature":"%s"}`, sha256Hex(manifest), hex.EncodeToString(signature))),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(replacement, name), content, 0o644); err != nil {
			t.Fatalf("write replacement artifact %s: %v", name, err)
		}
	}
	if err := Verify(replacement, trustedKeyPath); err == nil {
		t.Fatal("expected replacement artifact directory key to be rejected")
	}
}

func TestEnsureSigningKeyDerivesIDWhenCIKeyIDFileIsAbsent(t *testing.T) {
	root := t.TempDir()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release-ed25519-private.pem"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey}), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release-ed25519-public.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey}), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	keyID, _, loadedPrivate, err := ensureSigningKey(root)
	if err != nil {
		t.Fatalf("ensure signing key: %v", err)
	}
	if want := "release-" + hex.EncodeToString(publicKey[:6]); keyID != want {
		t.Fatalf("key ID=%q, want %q", keyID, want)
	}
	if !loadedPrivate.Equal(privateKey) {
		t.Fatal("loaded private key differs from CI signing key")
	}
}

func decodeJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
