package certificatematerials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	certificateFileName = "certificate.pem"
	privateKeyFileName  = "private.key"
)

// MaterialRecord keeps deterministic refs for uploaded certificate material.
type MaterialRecord struct {
	CertificateID  string `json:"certificate_id"`
	CertificateRef string `json:"certificate_ref"`
	PrivateKeyRef  string `json:"private_key_ref"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type state struct {
	Materials []MaterialRecord `json:"materials"`
}

// Store persists certificate material refs and stores files on disk.
type Store struct {
	root    string
	path    string
	files   string
	refBase string
	mu      sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("certificate materials store root is required")
	}
	if err := os.MkdirAll(filepath.Join(root, "files"), 0o755); err != nil {
		return nil, fmt.Errorf("create certificate materials store root: %w", err)
	}
	return &Store{
		root:    root,
		path:    filepath.Join(root, "materials.json"),
		files:   filepath.Join(root, "files"),
		refBase: filepath.Base(root),
	}, nil
}

func (s *Store) Put(certificateID string, certificatePEM []byte, privateKeyPEM []byte) (MaterialRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	certificateID = normalizeID(certificateID)
	if err := validateMaterialID(certificateID); err != nil {
		return MaterialRecord{}, err
	}
	if certificateID == "" {
		return MaterialRecord{}, errors.New("certificate material certificate_id is required")
	}
	if len(certificatePEM) == 0 {
		return MaterialRecord{}, errors.New("certificate file is required")
	}
	if len(privateKeyPEM) == 0 {
		return MaterialRecord{}, errors.New("private key file is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return MaterialRecord{}, err
	}

	record := MaterialRecord{
		CertificateID:  certificateID,
		CertificateRef: filepath.ToSlash(filepath.Join(s.refBase, "files", certificateID, certificateFileName)),
		PrivateKeyRef:  filepath.ToSlash(filepath.Join(s.refBase, "files", certificateID, privateKeyFileName)),
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	record.CreatedAt = now
	record.UpdatedAt = now
	for _, existing := range current.Materials {
		if existing.CertificateID == certificateID {
			record.CreatedAt = existing.CreatedAt
			break
		}
	}

	targetDir := filepath.Join(s.files, certificateID)
	if err := os.RemoveAll(targetDir); err != nil {
		return MaterialRecord{}, fmt.Errorf("reset certificate materials dir: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return MaterialRecord{}, fmt.Errorf("create certificate materials dir: %w", err)
	}
	if err := writeFileAtomically(targetDir, certificateFileName, certificatePEM, 0o600); err != nil {
		return MaterialRecord{}, fmt.Errorf("write certificate file: %w", err)
	}
	if err := writeFileAtomically(targetDir, privateKeyFileName, privateKeyPEM, 0o600); err != nil {
		return MaterialRecord{}, fmt.Errorf("write private key file: %w", err)
	}

	upserted := false
	for i := range current.Materials {
		if current.Materials[i].CertificateID == certificateID {
			current.Materials[i] = record
			upserted = true
			break
		}
	}
	if !upserted {
		current.Materials = append(current.Materials, record)
	}
	sortMaterials(current.Materials)
	if err := s.saveLocked(current); err != nil {
		return MaterialRecord{}, err
	}
	return record, nil
}

func (s *Store) Get(certificateID string) (MaterialRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	certificateID = normalizeID(certificateID)
	if err := validateMaterialID(certificateID); err != nil {
		return MaterialRecord{}, err
	}
	if certificateID == "" {
		return MaterialRecord{}, errors.New("certificate material certificate_id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return MaterialRecord{}, err
	}
	for _, item := range current.Materials {
		if item.CertificateID == certificateID {
			return item, nil
		}
	}
	return MaterialRecord{}, fmt.Errorf("certificate material %s not found", certificateID)
}

func (s *Store) Read(certificateID string) (MaterialRecord, []byte, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	certificateID = normalizeID(certificateID)
	if err := validateMaterialID(certificateID); err != nil {
		return MaterialRecord{}, nil, nil, err
	}
	if certificateID == "" {
		return MaterialRecord{}, nil, nil, errors.New("certificate material certificate_id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return MaterialRecord{}, nil, nil, err
	}
	for _, item := range current.Materials {
		if item.CertificateID != certificateID {
			continue
		}
		certificatePEM, err := os.ReadFile(filepath.Join(s.files, certificateID, certificateFileName))
		if err != nil {
			return MaterialRecord{}, nil, nil, fmt.Errorf("read certificate material file: %w", err)
		}
		privateKeyPEM, err := os.ReadFile(filepath.Join(s.files, certificateID, privateKeyFileName))
		if err != nil {
			return MaterialRecord{}, nil, nil, fmt.Errorf("read private key material file: %w", err)
		}
		return item, certificatePEM, privateKeyPEM, nil
	}
	return MaterialRecord{}, nil, nil, fmt.Errorf("certificate material %s not found", certificateID)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read certificate materials store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode certificate materials store: %w", err)
	}
	sortMaterials(current.Materials)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode certificate materials store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write certificate materials store: %w", err)
	}
	return nil
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateMaterialID(value string) error {
	if value == "" {
		return errors.New("certificate material certificate_id is required")
	}
	if value == "." || value == ".." {
		return errors.New("certificate material certificate_id is invalid")
	}
	if strings.Contains(value, "..") {
		return errors.New("certificate material certificate_id must not contain '..'")
	}
	if strings.ContainsAny(value, `/\`) {
		return errors.New("certificate material certificate_id must not contain path separators")
	}
	if strings.HasPrefix(value, "~") || strings.HasPrefix(value, ":") {
		return errors.New("certificate material certificate_id is invalid")
	}
	if strings.Contains(value, "\x00") {
		return errors.New("certificate material certificate_id is invalid")
	}
	return nil
}

func sortMaterials(items []MaterialRecord) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CertificateID < items[j].CertificateID
	})
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
