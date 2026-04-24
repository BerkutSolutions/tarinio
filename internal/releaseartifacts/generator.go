package releaseartifacts

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
)

type Options struct {
	RepoRoot   string
	Version    string
	CommitSHA  string
	Tag        string
	OutputDir  string
	DockerTags []string
}

type Result struct {
	OutputDir      string
	ManifestPath   string
	SignaturePath  string
	SBOMPath       string
	ProvenancePath string
	PublicKeyPath  string
}

type packageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type packageLockJSON struct {
	Name     string                     `json:"name"`
	Version  string                     `json:"version"`
	Packages map[string]packageLockNode `json:"packages"`
}

type packageLockNode struct {
	Version string `json:"version"`
}

type sourceInput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

type component struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	PURL    string `json:"purl,omitempty"`
}

func Generate(opts Options) (Result, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	version := strings.TrimSpace(opts.Version)
	if repoRoot == "" {
		return Result{}, errors.New("repo root is required")
	}
	if version == "" {
		return Result{}, errors.New("version is required")
	}
	if strings.TrimSpace(opts.OutputDir) == "" {
		opts.OutputDir = filepath.Join(repoRoot, "build", "release", version)
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output dir: %w", err)
	}

	signingRoot := filepath.Join(repoRoot, ".work", "release-signing")
	keyID, publicKeyPEM, privateKey, err := ensureSigningKey(signingRoot)
	if err != nil {
		return Result{}, err
	}
	publicKeyPath := filepath.Join(opts.OutputDir, "release-public-key.pem")
	if err := os.WriteFile(publicKeyPath, append([]byte(publicKeyPEM), '\n'), 0o644); err != nil {
		return Result{}, fmt.Errorf("write public key: %w", err)
	}

	inputPaths := []string{
		"go.mod",
		"go.sum",
		"package.json",
		"package-lock.json",
		"control-plane/Dockerfile",
		"deploy/compose/default/docker-compose.yml",
		"deploy/compose/enterprise/docker-compose.yml",
		"scripts/generate-release-artifacts.ps1",
		"scripts/install-aio.sh",
		"scripts/install-aio-enterprise.sh",
	}
	inputs, err := collectSourceInputs(repoRoot, inputPaths)
	if err != nil {
		return Result{}, err
	}

	sbom, err := buildSBOM(repoRoot, version)
	if err != nil {
		return Result{}, err
	}
	provenance := map[string]any{
		"_type":         "https://in-toto.io/Statement/v1",
		"subject":       []map[string]any{{"name": "TARINIO", "digest": map[string]string{"sha256": sha256Hex([]byte(version))}}},
		"predicateType": "https://slsa.dev/provenance/v1",
		"predicate": map[string]any{
			"buildDefinition": map[string]any{
				"buildType": "tarinio/release-artifacts@v1",
				"externalParameters": map[string]any{
					"version":     version,
					"tag":         strings.TrimSpace(opts.Tag),
					"docker_tags": append([]string(nil), opts.DockerTags...),
				},
				"resolvedDependencies": inputs,
			},
			"runDetails": map[string]any{
				"builder": map[string]any{
					"id": "tarinio/scripts/generate-release-artifacts.ps1",
				},
				"metadata": map[string]any{
					"invocation_id": "release-" + version,
					"started_at":    time.Now().UTC().Format(time.RFC3339Nano),
					"commit_sha":    strings.TrimSpace(opts.CommitSHA),
				},
			},
		},
	}

	sbomPath := filepath.Join(opts.OutputDir, "sbom.cdx.json")
	provenancePath := filepath.Join(opts.OutputDir, "provenance.json")
	if err := writePrettyJSON(sbomPath, sbom); err != nil {
		return Result{}, err
	}
	if err := writePrettyJSON(provenancePath, provenance); err != nil {
		return Result{}, err
	}

	outputFiles, err := collectSourceInputs(opts.OutputDir, []string{
		"release-public-key.pem",
		"sbom.cdx.json",
		"provenance.json",
	})
	if err != nil {
		return Result{}, err
	}
	checksumsPath := filepath.Join(opts.OutputDir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(renderChecksums(outputFiles)), 0o644); err != nil {
		return Result{}, fmt.Errorf("write checksums: %w", err)
	}
	outputFiles, err = collectSourceInputs(opts.OutputDir, []string{
		"release-public-key.pem",
		"sbom.cdx.json",
		"provenance.json",
		"checksums.txt",
	})
	if err != nil {
		return Result{}, err
	}

	manifest := map[string]any{
		"format":          "tarinio-release-artifacts/v1",
		"generated_at":    time.Now().UTC().Format(time.RFC3339Nano),
		"product":         "TARINIO",
		"version":         version,
		"tag":             strings.TrimSpace(opts.Tag),
		"commit_sha":      strings.TrimSpace(opts.CommitSHA),
		"docker_tags":     append([]string(nil), opts.DockerTags...),
		"signing_key_id":  keyID,
		"source_inputs":   inputs,
		"generated_files": outputFiles,
	}
	manifestPath := filepath.Join(opts.OutputDir, "release-manifest.json")
	if err := writePrettyJSON(manifestPath, manifest); err != nil {
		return Result{}, err
	}
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Result{}, fmt.Errorf("read manifest for signing: %w", err)
	}
	signature := ed25519.Sign(privateKey, manifestRaw)
	signaturePath := filepath.Join(opts.OutputDir, "signature.json")
	if err := writePrettyJSON(signaturePath, map[string]any{
		"algorithm":       "ed25519",
		"key_id":          keyID,
		"signed_object":   "release-manifest.json",
		"manifest_sha256": sha256Hex(manifestRaw),
		"signature":       hex.EncodeToString(signature),
	}); err != nil {
		return Result{}, err
	}

	return Result{
		OutputDir:      opts.OutputDir,
		ManifestPath:   manifestPath,
		SignaturePath:  signaturePath,
		SBOMPath:       sbomPath,
		ProvenancePath: provenancePath,
		PublicKeyPath:  publicKeyPath,
	}, nil
}

func buildSBOM(repoRoot, version string) (map[string]any, error) {
	goModPath := filepath.Join(repoRoot, "go.mod")
	goModRaw, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	mod, err := modfile.Parse("go.mod", goModRaw, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	pkg := packageJSON{}
	if err := readJSON(filepath.Join(repoRoot, "package.json"), &pkg); err != nil {
		return nil, err
	}
	lock := packageLockJSON{}
	if err := readJSON(filepath.Join(repoRoot, "package-lock.json"), &lock); err != nil {
		return nil, err
	}

	components := []component{
		{Type: "application", Name: firstNonEmpty(pkg.Name, "tarinio"), Version: version, PURL: "pkg:generic/tarinio@" + version},
	}
	for _, req := range mod.Require {
		modVersion := strings.TrimSpace(req.Mod.Version)
		if modVersion == "" {
			continue
		}
		name := strings.TrimSpace(req.Mod.Path)
		components = append(components, component{
			Type:    "library",
			Name:    name,
			Version: modVersion,
			PURL:    "pkg:golang/" + name + "@" + modVersion,
		})
	}
	paths := make([]string, 0, len(lock.Packages))
	for path := range lock.Packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if !strings.HasPrefix(path, "node_modules/") {
			continue
		}
		node := lock.Packages[path]
		if strings.TrimSpace(node.Version) == "" {
			continue
		}
		name := strings.TrimPrefix(path, "node_modules/")
		components = append(components, component{
			Type:    "library",
			Name:    name,
			Version: strings.TrimSpace(node.Version),
			PURL:    "pkg:npm/" + strings.ReplaceAll(name, "@", "%40") + "@" + strings.TrimSpace(node.Version),
		})
	}
	sort.Slice(components, func(i, j int) bool {
		if components[i].Type == components[j].Type {
			if components[i].Name == components[j].Name {
				return components[i].Version < components[j].Version
			}
			return components[i].Name < components[j].Name
		}
		return components[i].Type < components[j].Type
	})

	return map[string]any{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
		"metadata": map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"tools": []map[string]string{
				{"vendor": "Berkut Solutions", "name": "tarinio-release-artifacts", "version": version},
			},
			"component": map[string]any{
				"type":    "application",
				"name":    firstNonEmpty(pkg.Name, "tarinio"),
				"version": version,
			},
		},
		"components": components,
	}, nil
}

func collectSourceInputs(root string, relPaths []string) ([]sourceInput, error) {
	out := make([]sourceInput, 0, len(relPaths))
	for _, rel := range relPaths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		out = append(out, sourceInput{
			Path:   filepath.ToSlash(rel),
			SHA256: sha256Hex(raw),
			Size:   len(raw),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func renderChecksums(items []sourceInput) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s  %s", item.SHA256, item.Path))
	}
	return strings.Join(lines, "\n") + "\n"
}

func ensureSigningKey(root string) (string, string, ed25519.PrivateKey, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", "", nil, fmt.Errorf("create signing root: %w", err)
	}
	keyIDPath := filepath.Join(root, "key-id.txt")
	privatePath := filepath.Join(root, "release-ed25519-private.pem")
	publicPath := filepath.Join(root, "release-ed25519-public.pem")
	privateRaw, err := os.ReadFile(privatePath)
	if err == nil {
		block, _ := pem.Decode(privateRaw)
		if block == nil || len(block.Bytes) != ed25519.PrivateKeySize {
			return "", "", nil, errors.New("invalid release signing private key")
		}
		publicRaw, err := os.ReadFile(publicPath)
		if err != nil {
			return "", "", nil, fmt.Errorf("read release signing public key: %w", err)
		}
		keyIDRaw, err := os.ReadFile(keyIDPath)
		if err != nil {
			return "", "", nil, fmt.Errorf("read release signing key id: %w", err)
		}
		return strings.TrimSpace(string(keyIDRaw)), strings.TrimSpace(string(publicRaw)), ed25519.PrivateKey(block.Bytes), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", "", nil, fmt.Errorf("read release signing key: %w", err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate release signing key: %w", err)
	}
	keyID := "release-" + hex.EncodeToString(publicKey[:6])
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte(privateKey)})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte(publicKey)})
	if err := os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		return "", "", nil, fmt.Errorf("write release signing private key: %w", err)
	}
	if err := os.WriteFile(publicPath, publicPEM, 0o644); err != nil {
		return "", "", nil, fmt.Errorf("write release signing public key: %w", err)
	}
	if err := os.WriteFile(keyIDPath, []byte(keyID+"\n"), 0o644); err != nil {
		return "", "", nil, fmt.Errorf("write release signing key id: %w", err)
	}
	return keyID, strings.TrimSpace(string(publicPEM)), privateKey, nil
}

func writePrettyJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
