package sessions

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

	"waf/control-plane/internal/storage"
)

type Session struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	RoleIDs   []string `json:"role_ids"`
	CreatedAt string   `json:"created_at"`
	ExpiresAt string   `json:"expires_at"`
}

type LoginChallenge struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type TOTPSetupChallenge struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Secret    string `json:"secret"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type OIDCLoginChallenge struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Nonce     string `json:"nonce"`
	NextPath  string `json:"next_path,omitempty"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type state struct {
	Sessions            []Session            `json:"sessions"`
	LoginChallenges     []LoginChallenge     `json:"login_challenges"`
	TOTPSetupChallenges []TOTPSetupChallenge `json:"totp_setup_challenges"`
	OIDCLoginChallenges []OIDCLoginChallenge `json:"oidc_login_challenges"`
}

type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("sessions store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions store root: %w", err)
	}
	return &Store{state: storage.NewFileJSONState(filepath.Join(root, "sessions.json"))}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("sessions store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "sessions/sessions.json", filepath.Join(root, "sessions.json")),
	}, nil
}

func (s *Store) CreateSession(userID string, username string, roleIDs []string, ttl time.Duration) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	session := Session{
		ID:        newToken(),
		UserID:    normalize(userID),
		Username:  normalize(username),
		RoleIDs:   normalizeValues(roleIDs),
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(ttl).Format(time.RFC3339),
	}
	current.Sessions = append(current.Sessions, session)
	pruneExpired(current, now)
	if err := s.saveLocked(current); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Store) GetSession(id string) (Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Session{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	for _, item := range current.Sessions {
		if item.ID == strings.TrimSpace(id) {
			if err := s.saveLocked(current); err != nil {
				return Session{}, false, err
			}
			return item, true, nil
		}
	}
	if err := s.saveLocked(current); err != nil {
		return Session{}, false, err
	}
	return Session{}, false, nil
}

func (s *Store) Count() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	if err := s.saveLocked(current); err != nil {
		return 0, err
	}
	return len(current.Sessions), nil
}

func (s *Store) TouchSession(id string, ttl time.Duration) (Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Session{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	trimmedID := strings.TrimSpace(id)
	for i := range current.Sessions {
		if current.Sessions[i].ID != trimmedID {
			continue
		}
		current.Sessions[i].ExpiresAt = now.Add(ttl).Format(time.RFC3339)
		session := current.Sessions[i]
		if err := s.saveLocked(current); err != nil {
			return Session{}, false, err
		}
		return session, true, nil
	}
	if err := s.saveLocked(current); err != nil {
		return Session{}, false, err
	}
	return Session{}, false, nil
}

func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	filtered := current.Sessions[:0]
	for _, item := range current.Sessions {
		if item.ID != strings.TrimSpace(id) {
			filtered = append(filtered, item)
		}
	}
	current.Sessions = filtered
	return s.saveLocked(current)
}

func (s *Store) DeleteSessionsByUser(userID string) error {
	return s.deleteSessionsByUser(userID, "")
}

func (s *Store) DeleteSessionsByUserExcept(userID, exceptSessionID string) error {
	return s.deleteSessionsByUser(userID, exceptSessionID)
}

func (s *Store) deleteSessionsByUser(userID, exceptSessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	normalizedUserID := normalize(userID)
	trimmedExceptSessionID := strings.TrimSpace(exceptSessionID)
	filtered := current.Sessions[:0]
	for _, item := range current.Sessions {
		if item.UserID == normalizedUserID && item.ID != trimmedExceptSessionID {
			continue
		}
		filtered = append(filtered, item)
	}
	current.Sessions = filtered
	return s.saveLocked(current)
}

func (s *Store) CreateLoginChallenge(userID string, ttl time.Duration) (LoginChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return LoginChallenge{}, err
	}
	now := time.Now().UTC()
	challenge := LoginChallenge{
		ID:        newToken(),
		UserID:    normalize(userID),
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(ttl).Format(time.RFC3339),
	}
	current.LoginChallenges = append(current.LoginChallenges, challenge)
	pruneExpired(current, now)
	if err := s.saveLocked(current); err != nil {
		return LoginChallenge{}, err
	}
	return challenge, nil
}

func (s *Store) ConsumeLoginChallenge(id string) (LoginChallenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return LoginChallenge{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	for i, item := range current.LoginChallenges {
		if item.ID != strings.TrimSpace(id) {
			continue
		}
		current.LoginChallenges = append(current.LoginChallenges[:i], current.LoginChallenges[i+1:]...)
		if err := s.saveLocked(current); err != nil {
			return LoginChallenge{}, false, err
		}
		return item, true, nil
	}
	if err := s.saveLocked(current); err != nil {
		return LoginChallenge{}, false, err
	}
	return LoginChallenge{}, false, nil
}

func (s *Store) GetLoginChallenge(id string) (LoginChallenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return LoginChallenge{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	needle := strings.TrimSpace(id)
	for _, item := range current.LoginChallenges {
		if item.ID == needle {
			if err := s.saveLocked(current); err != nil {
				return LoginChallenge{}, false, err
			}
			return item, true, nil
		}
	}
	if err := s.saveLocked(current); err != nil {
		return LoginChallenge{}, false, err
	}
	return LoginChallenge{}, false, nil
}

func (s *Store) CreateTOTPSetupChallenge(userID, secret string, ttl time.Duration) (TOTPSetupChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return TOTPSetupChallenge{}, err
	}
	now := time.Now().UTC()
	filtered := current.TOTPSetupChallenges[:0]
	for _, item := range current.TOTPSetupChallenges {
		if item.UserID != normalize(userID) {
			filtered = append(filtered, item)
		}
	}
	current.TOTPSetupChallenges = filtered
	challenge := TOTPSetupChallenge{
		ID:        newToken(),
		UserID:    normalize(userID),
		Secret:    strings.TrimSpace(secret),
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(ttl).Format(time.RFC3339),
	}
	current.TOTPSetupChallenges = append(current.TOTPSetupChallenges, challenge)
	pruneExpired(current, now)
	if err := s.saveLocked(current); err != nil {
		return TOTPSetupChallenge{}, err
	}
	return challenge, nil
}

func (s *Store) CreateOIDCLoginChallenge(nextPath string, ttl time.Duration) (OIDCLoginChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return OIDCLoginChallenge{}, err
	}
	now := time.Now().UTC()
	challenge := OIDCLoginChallenge{
		ID:        newToken(),
		State:     newToken(),
		Nonce:     newToken(),
		NextPath:  strings.TrimSpace(nextPath),
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(ttl).Format(time.RFC3339),
	}
	current.OIDCLoginChallenges = append(current.OIDCLoginChallenges, challenge)
	pruneExpired(current, now)
	if err := s.saveLocked(current); err != nil {
		return OIDCLoginChallenge{}, err
	}
	return challenge, nil
}

func (s *Store) ConsumeOIDCLoginChallenge(state string) (OIDCLoginChallenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return OIDCLoginChallenge{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	needle := strings.TrimSpace(state)
	for i, item := range current.OIDCLoginChallenges {
		if item.State != needle {
			continue
		}
		current.OIDCLoginChallenges = append(current.OIDCLoginChallenges[:i], current.OIDCLoginChallenges[i+1:]...)
		if err := s.saveLocked(current); err != nil {
			return OIDCLoginChallenge{}, false, err
		}
		return item, true, nil
	}
	if err := s.saveLocked(current); err != nil {
		return OIDCLoginChallenge{}, false, err
	}
	return OIDCLoginChallenge{}, false, nil
}

func (s *Store) ConsumeTOTPSetupChallenge(id string) (TOTPSetupChallenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return TOTPSetupChallenge{}, false, err
	}
	now := time.Now().UTC()
	pruneExpired(current, now)
	for i, item := range current.TOTPSetupChallenges {
		if item.ID != strings.TrimSpace(id) {
			continue
		}
		current.TOTPSetupChallenges = append(current.TOTPSetupChallenges[:i], current.TOTPSetupChallenges[i+1:]...)
		if err := s.saveLocked(current); err != nil {
			return TOTPSetupChallenge{}, false, err
		}
		return item, true, nil
	}
	if err := s.saveLocked(current); err != nil {
		return TOTPSetupChallenge{}, false, err
	}
	return TOTPSetupChallenge{}, false, nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read sessions store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode sessions store: %w", err)
	}
	sortSessions(current.Sessions)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sessions store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write sessions store: %w", err)
	}
	return nil
}

func pruneExpired(current *state, now time.Time) {
	current.Sessions = pruneSessions(current.Sessions, now)
	current.LoginChallenges = pruneLoginChallenges(current.LoginChallenges, now)
	current.TOTPSetupChallenges = pruneSetupChallenges(current.TOTPSetupChallenges, now)
	current.OIDCLoginChallenges = pruneOIDCChallenges(current.OIDCLoginChallenges, now)
}

func pruneSessions(items []Session, now time.Time) []Session {
	filtered := items[:0]
	for _, item := range items {
		expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			filtered = append(filtered, item)
		}
	}
	sortSessions(filtered)
	return filtered
}

func pruneLoginChallenges(items []LoginChallenge, now time.Time) []LoginChallenge {
	filtered := items[:0]
	for _, item := range items {
		expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func pruneSetupChallenges(items []TOTPSetupChallenge, now time.Time) []TOTPSetupChallenge {
	filtered := items[:0]
	for _, item := range items {
		expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func pruneOIDCChallenges(items []OIDCLoginChallenge, now time.Time) []OIDCLoginChallenge {
	filtered := items[:0]
	for _, item := range items {
		expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func newToken() string {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeValues(items []string) []string {
	out := append([]string(nil), items...)
	for i := range out {
		out[i] = normalize(out[i])
	}
	sort.Strings(out)
	return out
}

func sortSessions(items []Session) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}
