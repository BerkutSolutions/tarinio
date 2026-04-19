package revisionsnapshots

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/wafpolicies"
)

// Snapshot is the immutable control-plane state captured for one revision.
type Snapshot struct {
	Sites                []sites.Site                        `json:"sites"`
	Upstreams            []upstreams.Upstream                `json:"upstreams"`
	Certificates         []certificates.Certificate          `json:"certificates"`
	TLSConfigs           []tlsconfigs.TLSConfig              `json:"tls_configs"`
	WAFPolicies          []wafpolicies.WAFPolicy             `json:"waf_policies"`
	AccessPolicies       []accesspolicies.AccessPolicy       `json:"access_policies"`
	RateLimitPolicies    []ratelimitpolicies.RateLimitPolicy `json:"rate_limit_policies"`
	EasySiteProfiles     []easysiteprofiles.EasySiteProfile  `json:"easy_site_profiles"`
	AntiDDoSSettings     antiddos.Settings                   `json:"anti_ddos_settings"`
	CertificateMaterials []CertificateMaterialSnapshot       `json:"certificate_materials"`
}

type CertificateMaterialSnapshot struct {
	CertificateID  string `json:"certificate_id"`
	CertificateRef string `json:"certificate_ref"`
	PrivateKeyRef  string `json:"private_key_ref"`
}

type MaterialContent struct {
	CertificateID  string
	CertificatePEM []byte
	PrivateKeyPEM  []byte
}

// Store persists immutable revision snapshots without runtime coupling.
type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("revision snapshots store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create revision snapshots root: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) Save(revisionID string, snapshot Snapshot, materials []MaterialContent) (string, string, error) {
	revisionID = normalizeID(revisionID)
	if revisionID == "" {
		return "", "", errors.New("revision id is required")
	}
	materialSnapshots, err := s.writeMaterials(revisionID, materials)
	if err != nil {
		return "", "", err
	}
	snapshot.CertificateMaterials = materialSnapshots

	content, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("encode revision snapshot: %w", err)
	}
	content = append(content, '\n')
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])

	filename := revisionID + ".json"
	fullPath := filepath.Join(s.root, filename)
	tempPath := fullPath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return "", "", fmt.Errorf("write revision snapshot temp file: %w", err)
	}
	if err := os.Rename(tempPath, fullPath); err != nil {
		_ = os.Remove(tempPath)
		return "", "", fmt.Errorf("rename revision snapshot file: %w", err)
	}
	return filepath.ToSlash(filepath.Join("snapshots", filename)), checksum, nil
}

func (s *Store) Load(snapshotPath string) (Snapshot, error) {
	relative := strings.TrimSpace(snapshotPath)
	if relative == "" {
		return Snapshot{}, errors.New("snapshot path is required")
	}
	relative = strings.TrimPrefix(filepath.ToSlash(relative), "snapshots/")
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(relative)))
	if err != nil {
		return Snapshot{}, fmt.Errorf("read revision snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode revision snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) ReadMaterial(ref string) ([]byte, error) {
	relative := strings.TrimSpace(ref)
	if relative == "" {
		return nil, errors.New("material ref is required")
	}
	basePrefix := filepath.ToSlash(filepath.Base(s.root)) + "/"
	relative = strings.TrimPrefix(filepath.ToSlash(relative), basePrefix)
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(relative)))
	if err != nil {
		return nil, fmt.Errorf("read revision snapshot material: %w", err)
	}
	return content, nil
}

func (s *Store) Delete(snapshotPath string) error {
	relative := strings.TrimSpace(snapshotPath)
	if relative == "" {
		return errors.New("snapshot path is required")
	}
	revisionID := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(relative), "snapshots/"), ".json")
	if revisionID == "" {
		return errors.New("snapshot path is invalid")
	}

	fullPath := filepath.Join(s.root, filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(relative), "snapshots/")))
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete revision snapshot: %w", err)
	}
	materialsRoot := filepath.Join(s.root, "files", normalizeID(revisionID))
	if err := os.RemoveAll(materialsRoot); err != nil {
		return fmt.Errorf("delete revision snapshot materials: %w", err)
	}
	return nil
}

func (s *Store) writeMaterials(revisionID string, materials []MaterialContent) ([]CertificateMaterialSnapshot, error) {
	if len(materials) == 0 {
		return nil, nil
	}

	sort.Slice(materials, func(i, j int) bool {
		return normalizeID(materials[i].CertificateID) < normalizeID(materials[j].CertificateID)
	})

	out := make([]CertificateMaterialSnapshot, 0, len(materials))
	for _, item := range materials {
		certificateID := normalizeID(item.CertificateID)
		if certificateID == "" {
			return nil, errors.New("snapshot material certificate_id is required")
		}
		if len(item.CertificatePEM) == 0 || len(item.PrivateKeyPEM) == 0 {
			return nil, fmt.Errorf("snapshot material for certificate %s is incomplete", certificateID)
		}

		targetDir := filepath.Join(s.root, "files", revisionID, certificateID)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return nil, fmt.Errorf("create revision snapshot material dir: %w", err)
		}
		if err := writeFileAtomically(targetDir, "certificate.pem", item.CertificatePEM, 0o600); err != nil {
			return nil, fmt.Errorf("write revision snapshot certificate: %w", err)
		}
		if err := writeFileAtomically(targetDir, "private.key", item.PrivateKeyPEM, 0o600); err != nil {
			return nil, fmt.Errorf("write revision snapshot private key: %w", err)
		}

		out = append(out, CertificateMaterialSnapshot{
			CertificateID:  certificateID,
			CertificateRef: filepath.ToSlash(filepath.Join(filepath.Base(s.root), "files", revisionID, certificateID, "certificate.pem")),
			PrivateKeyRef:  filepath.ToSlash(filepath.Join(filepath.Base(s.root), "files", revisionID, certificateID, "private.key")),
		})
	}
	return out, nil
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func writeFileAtomically(dir string, name string, content []byte, mode os.FileMode) error {
	tempPath := filepath.Join(dir, name+".tmp."+strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
	if err := os.WriteFile(tempPath, content, mode); err != nil {
		return err
	}
	targetPath := filepath.Join(dir, name)
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Chmod(targetPath, mode); err != nil {
		return err
	}
	return nil
}
