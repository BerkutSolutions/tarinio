package releaseartifacts

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type signaturePayload struct {
	Algorithm      string `json:"algorithm"`
	KeyID          string `json:"key_id"`
	SignedObject   string `json:"signed_object"`
	ManifestSHA256 string `json:"manifest_sha256"`
	Signature      string `json:"signature"`
}

func Verify(outputDir string) error {
	dir := strings.TrimSpace(outputDir)
	if dir == "" {
		return errors.New("output directory is required")
	}
	requiredFiles := []string{
		"release-manifest.json",
		"signature.json",
		"checksums.txt",
		"sbom.cdx.json",
		"provenance.json",
		"release-public-key.pem",
	}
	for _, name := range requiredFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("missing required artifact %s: %w", name, err)
		}
	}

	if err := verifyChecksums(dir); err != nil {
		return err
	}
	return verifySignature(dir)
}

func verifyChecksums(outputDir string) error {
	raw, err := os.ReadFile(filepath.Join(outputDir, "checksums.txt"))
	if err != nil {
		return fmt.Errorf("read checksums.txt: %w", err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return errors.New("invalid checksums.txt format")
		}
		expected := strings.ToLower(strings.TrimSpace(parts[0]))
		relative := filepath.Clean(strings.TrimSpace(parts[1]))
		if expected == "" || relative == "." || strings.HasPrefix(relative, "..") {
			return errors.New("invalid checksums.txt entry")
		}
		target := filepath.Join(outputDir, relative)
		content, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("read checksum target %s: %w", relative, err)
		}
		actual := sha256.Sum256(content)
		if hex.EncodeToString(actual[:]) != expected {
			return fmt.Errorf("checksum mismatch for %s", relative)
		}
	}
	return nil
}

func verifySignature(outputDir string) error {
	manifestRaw, err := os.ReadFile(filepath.Join(outputDir, "release-manifest.json"))
	if err != nil {
		return fmt.Errorf("read release-manifest.json: %w", err)
	}
	signatureRaw, err := os.ReadFile(filepath.Join(outputDir, "signature.json"))
	if err != nil {
		return fmt.Errorf("read signature.json: %w", err)
	}
	publicKeyRaw, err := os.ReadFile(filepath.Join(outputDir, "release-public-key.pem"))
	if err != nil {
		return fmt.Errorf("read release-public-key.pem: %w", err)
	}

	var signature signaturePayload
	if err := json.Unmarshal(signatureRaw, &signature); err != nil {
		return fmt.Errorf("decode signature.json: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(signature.Algorithm)) != "ed25519" {
		return errors.New("unsupported signature algorithm")
	}
	if strings.TrimSpace(signature.SignedObject) != "release-manifest.json" {
		return errors.New("signature signed_object must be release-manifest.json")
	}

	manifestSum := sha256.Sum256(manifestRaw)
	if hex.EncodeToString(manifestSum[:]) != strings.ToLower(strings.TrimSpace(signature.ManifestSHA256)) {
		return errors.New("manifest hash mismatch in signature.json")
	}

	block, _ := pem.Decode(publicKeyRaw)
	if block == nil || len(block.Bytes) != ed25519.PublicKeySize {
		return errors.New("invalid public key format")
	}
	sig, err := hex.DecodeString(strings.TrimSpace(signature.Signature))
	if err != nil {
		return fmt.Errorf("decode signature hex: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(block.Bytes), manifestRaw, sig) {
		return errors.New("ed25519 signature verification failed")
	}

	var manifest map[string]any
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return fmt.Errorf("decode release-manifest.json: %w", err)
	}
	manifestKeyID := strings.TrimSpace(asManifestString(manifest["signing_key_id"]))
	if manifestKeyID == "" || manifestKeyID != strings.TrimSpace(signature.KeyID) {
		return errors.New("signature key id mismatch with manifest signing_key_id")
	}
	return nil
}

func asManifestString(value any) string {
	typed, _ := value.(string)
	return typed
}
