package users

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

	"golang.org/x/crypto/argon2"
)

type User struct {
	ID                 string             `json:"id"`
	Username           string             `json:"username"`
	Email              string             `json:"email"`
	FullName           string             `json:"full_name,omitempty"`
	Department         string             `json:"department,omitempty"`
	Position           string             `json:"position,omitempty"`
	PasswordHash       string             `json:"password_hash"`
	IsActive           bool               `json:"is_active"`
	RoleIDs            []string           `json:"role_ids"`
	TOTPEnabled        bool               `json:"totp_enabled"`
	TOTPSecret         string             `json:"totp_secret,omitempty"`
	TOTPSecretEnc      string             `json:"totp_secret_enc,omitempty"`
	TOTPRecoveryHashes []TOTPRecoveryHash `json:"totp_recovery_hashes,omitempty"`
	LastLoginAt        string             `json:"last_login_at,omitempty"`
	CreatedAt          string             `json:"created_at"`
	UpdatedAt          string             `json:"updated_at"`
}

type TOTPRecoveryHash struct {
	Hash   string `json:"hash"`
	Salt   string `json:"salt"`
	UsedAt string `json:"used_at,omitempty"`
}

type BootstrapUser struct {
	Enabled  bool
	ID       string
	Username string
	Email    string
	Password string
	RoleIDs  []string
}

type state struct {
	Users []User `json:"users"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string, bootstrap BootstrapUser) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("users store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create users store root: %w", err)
	}
	store := &Store{path: filepath.Join(root, "users.json")}
	if err := store.seedBootstrap(bootstrap); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Get(id string) (User, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return User{}, false, err
	}
	id = normalizeID(id)
	for _, item := range current.Users {
		if item.ID == id {
			return item, true, nil
		}
	}
	return User{}, false, nil
}

func (s *Store) FindByUsername(username string) (User, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return User{}, false, err
	}
	username = normalizeUsername(username)
	for _, item := range current.Users {
		if item.Username == username {
			return item, true, nil
		}
	}
	return User{}, false, nil
}

func (s *Store) List() ([]User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]User(nil), current.Users...)
	sortUsers(items)
	return items, nil
}

func (s *Store) Update(user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return User{}, err
	}
	user = normalizeUser(user)
	for i := range current.Users {
		if current.Users[i].ID != user.ID {
			continue
		}
		user.CreatedAt = current.Users[i].CreatedAt
		user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Users[i] = user
		sortUsers(current.Users)
		if err := s.saveLocked(current); err != nil {
			return User{}, err
		}
		return user, nil
	}
	return User{}, fmt.Errorf("user %s not found", user.ID)
}

func (s *Store) Create(user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return User{}, err
	}
	user = normalizeUser(user)
	if user.ID == "" {
		return User{}, errors.New("user id is required")
	}
	if user.Username == "" {
		return User{}, errors.New("username is required")
	}
	for _, item := range current.Users {
		if item.ID == user.ID {
			return User{}, fmt.Errorf("user %s already exists", user.ID)
		}
		if item.Username == user.Username {
			return User{}, fmt.Errorf("username %s already exists", user.Username)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = now
	user.UpdatedAt = now
	current.Users = append(current.Users, user)
	sortUsers(current.Users)
	if err := s.saveLocked(current); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) Count() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	return len(current.Users), nil
}

func (s *Store) MarkLogin(userID string, when time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Users {
		if current.Users[i].ID != normalizeID(userID) {
			continue
		}
		current.Users[i].LastLoginAt = when.UTC().Format(time.RFC3339)
		current.Users[i].UpdatedAt = when.UTC().Format(time.RFC3339)
		return s.saveLocked(current)
	}
	return fmt.Errorf("user %s not found", userID)
}

func (s *Store) seedBootstrap(bootstrap BootstrapUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !bootstrap.Enabled {
		return nil
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	if len(current.Users) > 0 {
		return nil
	}
	passwordHash, err := HashPassword(bootstrap.Password)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	current.Users = append(current.Users, normalizeUser(User{
		ID:           bootstrap.ID,
		Username:     bootstrap.Username,
		Email:        bootstrap.Email,
		PasswordHash: passwordHash,
		IsActive:     true,
		RoleIDs:      append([]string(nil), bootstrap.RoleIDs...),
		CreatedAt:    now,
		UpdatedAt:    now,
	}))
	return s.saveLocked(current)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read users store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode users store: %w", err)
	}
	sortUsers(current.Users)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode users store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o600); err != nil {
		return fmt.Errorf("write users store: %w", err)
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password is required")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := deriveArgon2idKey([]byte(password), salt)
	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemoryKiB,
		argonTimeCost,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

func VerifyPassword(password, encoded string) bool {
	return verifyPasswordDetails(password, encoded)
}

func NeedsPasswordRehash(encoded string) bool {
	return !strings.HasPrefix(strings.TrimSpace(encoded), "argon2id$")
}

const (
	argonTimeCost    uint32 = 3
	argonMemoryKiB   uint32 = 64 * 1024
	argonParallelism uint8  = 2
	argonKeyLength   uint32 = 32
)

func verifyPasswordDetails(password, encoded string) bool {
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, "argon2id$") {
		return verifyArgon2idPassword(password, encoded)
	}
	return verifyLegacyPassword(password, encoded)
}

func verifyLegacyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := deriveKey([]byte(password), salt)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func verifyArgon2idPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false
	}
	if parts[0] != "argon2id" || parts[1] != "v=19" {
		return false
	}
	var memory uint32
	var timeCost uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &timeCost, &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	keyLength := uint32(len(expected))
	if keyLength == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, timeCost, memory, parallelism, keyLength)
	return hmac.Equal(actual, expected)
}

func deriveArgon2idKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argonTimeCost, argonMemoryKiB, argonParallelism, argonKeyLength)
}

func deriveKey(password, salt []byte) []byte {
	sum := sha256.Sum256(append(append([]byte(nil), password...), salt...))
	for i := 0; i < 120000; i++ {
		combined := append(append([]byte(nil), sum[:]...), salt...)
		sum = sha256.Sum256(combined)
	}
	return sum[:]
}

func normalizeUser(user User) User {
	user.ID = normalizeID(user.ID)
	user.Username = normalizeUsername(user.Username)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.FullName = strings.TrimSpace(user.FullName)
	user.Department = strings.TrimSpace(user.Department)
	user.Position = strings.TrimSpace(user.Position)
	user.RoleIDs = normalizeValues(user.RoleIDs)
	user.TOTPSecret = strings.TrimSpace(user.TOTPSecret)
	user.TOTPSecretEnc = strings.TrimSpace(user.TOTPSecretEnc)
	for i := range user.TOTPRecoveryHashes {
		user.TOTPRecoveryHashes[i].Hash = strings.TrimSpace(user.TOTPRecoveryHashes[i].Hash)
		user.TOTPRecoveryHashes[i].Salt = strings.TrimSpace(user.TOTPRecoveryHashes[i].Salt)
		user.TOTPRecoveryHashes[i].UsedAt = strings.TrimSpace(user.TOTPRecoveryHashes[i].UsedAt)
	}
	return user
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeValues(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		value := normalizeID(item)
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

func sortUsers(items []User) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}
