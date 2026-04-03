package passkeys

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Passkey struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	Name            string   `json:"name"`
	CredentialID    string   `json:"credential_id"`
	PublicKey       []byte   `json:"public_key,omitempty"`
	AttestationType string   `json:"attestation_type,omitempty"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	SignCount       uint32   `json:"sign_count,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	LastUsedAt      string   `json:"last_used_at,omitempty"`
}

type Challenge struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	UserID          string `json:"user_id,omitempty"`
	RefID           string `json:"ref_id,omitempty"`
	Name            string `json:"name,omitempty"`
	SessionDataJSON string `json:"session_data_json"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
	ClientIP        string `json:"client_ip,omitempty"`
	UserAgent       string `json:"user_agent,omitempty"`
}

type state struct {
	Passkeys   []Passkey   `json:"passkeys"`
	Challenges []Challenge `json:"challenges"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("passkeys store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create passkeys store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "passkeys.json")}, nil
}

func (s *Store) ListByUser(userID string) ([]Passkey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	needle := normalize(userID)
	out := make([]Passkey, 0)
	for _, item := range current.Passkeys {
		if item.UserID == needle {
			out = append(out, item)
		}
	}
	sortPasskeys(out)
	return out, nil
}

func (s *Store) FindByCredentialID(credentialID string) (Passkey, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Passkey{}, false, err
	}
	needle := strings.TrimSpace(credentialID)
	for _, item := range current.Passkeys {
		if item.CredentialID == needle {
			return item, true, nil
		}
	}
	return Passkey{}, false, nil
}

func (s *Store) FindByID(id string) (Passkey, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Passkey{}, false, err
	}
	needle := strings.TrimSpace(id)
	for _, item := range current.Passkeys {
		if item.ID == needle {
			return item, true, nil
		}
	}
	return Passkey{}, false, nil
}

func (s *Store) Create(item Passkey) (Passkey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Passkey{}, err
	}
	item = normalizePasskey(item)
	if item.UserID == "" {
		return Passkey{}, errors.New("user id is required")
	}
	if item.CredentialID == "" {
		return Passkey{}, errors.New("credential id is required")
	}
	for _, existing := range current.Passkeys {
		if existing.CredentialID == item.CredentialID {
			return Passkey{}, errors.New("passkey already exists")
		}
	}
	if item.ID == "" {
		item.ID = newToken(16)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Passkeys = append(current.Passkeys, item)
	sortPasskeys(current.Passkeys)
	if err := s.saveLocked(current); err != nil {
		return Passkey{}, err
	}
	return item, nil
}

func (s *Store) Rename(id, name string) (Passkey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Passkey{}, err
	}
	needle := strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "My Passkey"
	}
	for i := range current.Passkeys {
		if current.Passkeys[i].ID != needle {
			continue
		}
		current.Passkeys[i].Name = name
		current.Passkeys[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveLocked(current); err != nil {
			return Passkey{}, err
		}
		return current.Passkeys[i], nil
	}
	return Passkey{}, errors.New("passkey not found")
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	needle := strings.TrimSpace(id)
	filtered := current.Passkeys[:0]
	for _, item := range current.Passkeys {
		if item.ID != needle {
			filtered = append(filtered, item)
		}
	}
	current.Passkeys = filtered
	return s.saveLocked(current)
}

func (s *Store) MarkUsed(credentialID string, signCount uint32, when time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	needle := strings.TrimSpace(credentialID)
	for i := range current.Passkeys {
		if current.Passkeys[i].CredentialID != needle {
			continue
		}
		current.Passkeys[i].LastUsedAt = when.UTC().Format(time.RFC3339Nano)
		current.Passkeys[i].UpdatedAt = when.UTC().Format(time.RFC3339Nano)
		if signCount > current.Passkeys[i].SignCount {
			current.Passkeys[i].SignCount = signCount
		}
		return s.saveLocked(current)
	}
	return nil
}

func (s *Store) CreateChallenge(item Challenge, ttl time.Duration) (Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Challenge{}, err
	}
	item = normalizeChallenge(item)
	if item.Kind == "" {
		return Challenge{}, errors.New("challenge kind is required")
	}
	if item.SessionDataJSON == "" {
		return Challenge{}, errors.New("challenge session data is required")
	}
	if item.ID == "" {
		item.ID = newToken(24)
	}
	now := time.Now().UTC()
	item.CreatedAt = now.Format(time.RFC3339Nano)
	item.ExpiresAt = now.Add(ttl).Format(time.RFC3339Nano)
	current.Challenges = append(current.Challenges, item)
	pruneChallenges(current, now)
	if err := s.saveLocked(current); err != nil {
		return Challenge{}, err
	}
	return item, nil
}

func (s *Store) GetChallenge(id string) (Challenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Challenge{}, false, err
	}
	now := time.Now().UTC()
	pruneChallenges(current, now)
	needle := strings.TrimSpace(id)
	for _, item := range current.Challenges {
		if item.ID == needle {
			if err := s.saveLocked(current); err != nil {
				return Challenge{}, false, err
			}
			return item, true, nil
		}
	}
	if err := s.saveLocked(current); err != nil {
		return Challenge{}, false, err
	}
	return Challenge{}, false, nil
}

func (s *Store) DeleteChallenge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	needle := strings.TrimSpace(id)
	filtered := current.Challenges[:0]
	for _, item := range current.Challenges {
		if item.ID != needle {
			filtered = append(filtered, item)
		}
	}
	current.Challenges = filtered
	return s.saveLocked(current)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read passkeys store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode passkeys store: %w", err)
	}
	sortPasskeys(current.Passkeys)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode passkeys store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o600); err != nil {
		return fmt.Errorf("write passkeys store: %w", err)
	}
	return nil
}

func pruneChallenges(current *state, now time.Time) {
	filtered := current.Challenges[:0]
	for _, item := range current.Challenges {
		expiresAt, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			filtered = append(filtered, item)
		}
	}
	current.Challenges = filtered
}

func sortPasskeys(items []Passkey) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].UserID != items[j].UserID {
			return items[i].UserID < items[j].UserID
		}
		return items[i].ID < items[j].ID
	})
}

func normalizePasskey(item Passkey) Passkey {
	item.ID = strings.TrimSpace(item.ID)
	item.UserID = normalize(item.UserID)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		item.Name = "My Passkey"
	}
	item.CredentialID = strings.TrimSpace(item.CredentialID)
	item.AttestationType = strings.TrimSpace(item.AttestationType)
	item.CreatedAt = strings.TrimSpace(item.CreatedAt)
	item.UpdatedAt = strings.TrimSpace(item.UpdatedAt)
	item.LastUsedAt = strings.TrimSpace(item.LastUsedAt)
	item.Transports = normalizeValues(item.Transports)
	return item
}

func normalizeChallenge(item Challenge) Challenge {
	item.ID = strings.TrimSpace(item.ID)
	item.Kind = strings.TrimSpace(item.Kind)
	item.UserID = normalize(item.UserID)
	item.RefID = strings.TrimSpace(item.RefID)
	item.Name = strings.TrimSpace(item.Name)
	item.SessionDataJSON = strings.TrimSpace(item.SessionDataJSON)
	item.ClientIP = strings.TrimSpace(item.ClientIP)
	item.UserAgent = strings.TrimSpace(item.UserAgent)
	return item
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeValues(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func newToken(size int) string {
	if size <= 0 {
		size = 32
	}
	raw := make([]byte, size)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}
