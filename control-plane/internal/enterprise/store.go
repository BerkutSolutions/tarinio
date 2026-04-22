package enterprise

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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

type RoleMapping struct {
	ExternalGroup string   `json:"external_group"`
	RoleIDs       []string `json:"role_ids"`
}

type OIDCSettings struct {
	Enabled              bool          `json:"enabled"`
	DisplayName          string        `json:"display_name,omitempty"`
	IssuerURL            string        `json:"issuer_url,omitempty"`
	ClientID             string        `json:"client_id,omitempty"`
	ClientSecret         string        `json:"client_secret,omitempty"`
	RedirectURL          string        `json:"redirect_url,omitempty"`
	Scopes               []string      `json:"scopes,omitempty"`
	UsernameClaim        string        `json:"username_claim,omitempty"`
	EmailClaim           string        `json:"email_claim,omitempty"`
	FullNameClaim        string        `json:"full_name_claim,omitempty"`
	GroupsClaim          string        `json:"groups_claim,omitempty"`
	AllowedEmailDomains  []string      `json:"allowed_email_domains,omitempty"`
	DefaultRoleIDs       []string      `json:"default_role_ids,omitempty"`
	GroupRoleMappings    []RoleMapping `json:"group_role_mappings,omitempty"`
	AutoProvision        bool          `json:"auto_provision"`
	RequireVerifiedEmail bool          `json:"require_verified_email"`
}

type ApprovalPolicy struct {
	Enabled           bool     `json:"enabled"`
	RequiredApprovals int      `json:"required_approvals"`
	AllowSelfApproval bool     `json:"allow_self_approval"`
	ReviewerRoleIDs   []string `json:"reviewer_role_ids,omitempty"`
}

type SCIMToken struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Prefix      string `json:"prefix"`
	SecretHash  string `json:"secret_hash"`
	CreatedAt   string `json:"created_at"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	Disabled    bool   `json:"disabled"`
}

type SCIMSettings struct {
	Enabled           bool          `json:"enabled"`
	DefaultRoleIDs    []string      `json:"default_role_ids,omitempty"`
	GroupRoleMappings []RoleMapping `json:"group_role_mappings,omitempty"`
	Tokens            []SCIMToken   `json:"tokens,omitempty"`
}

type EvidenceSettings struct {
	KeyID         string `json:"key_id"`
	PublicKeyPEM  string `json:"public_key_pem"`
	PrivateKeyPEM string `json:"private_key_pem"`
}

type Settings struct {
	OIDC      OIDCSettings     `json:"oidc"`
	Approvals ApprovalPolicy   `json:"approvals"`
	SCIM      SCIMSettings     `json:"scim"`
	Evidence  EvidenceSettings `json:"evidence"`
}

type state struct {
	Settings Settings `json:"settings"`
}

type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

type SettingsView struct {
	OIDC struct {
		Enabled              bool          `json:"enabled"`
		DisplayName          string        `json:"display_name,omitempty"`
		IssuerURL            string        `json:"issuer_url,omitempty"`
		ClientID             string        `json:"client_id,omitempty"`
		RedirectURL          string        `json:"redirect_url,omitempty"`
		Scopes               []string      `json:"scopes,omitempty"`
		UsernameClaim        string        `json:"username_claim,omitempty"`
		EmailClaim           string        `json:"email_claim,omitempty"`
		FullNameClaim        string        `json:"full_name_claim,omitempty"`
		GroupsClaim          string        `json:"groups_claim,omitempty"`
		AllowedEmailDomains  []string      `json:"allowed_email_domains,omitempty"`
		DefaultRoleIDs       []string      `json:"default_role_ids,omitempty"`
		GroupRoleMappings    []RoleMapping `json:"group_role_mappings,omitempty"`
		AutoProvision        bool          `json:"auto_provision"`
		RequireVerifiedEmail bool          `json:"require_verified_email"`
		HasClientSecret      bool          `json:"has_client_secret"`
	} `json:"oidc"`
	Approvals ApprovalPolicy `json:"approvals"`
	SCIM      struct {
		Enabled           bool            `json:"enabled"`
		DefaultRoleIDs    []string        `json:"default_role_ids,omitempty"`
		GroupRoleMappings []RoleMapping   `json:"group_role_mappings,omitempty"`
		Tokens            []SCIMTokenView `json:"tokens,omitempty"`
	} `json:"scim"`
	Evidence struct {
		KeyID        string `json:"key_id"`
		PublicKeyPEM string `json:"public_key_pem"`
	} `json:"evidence"`
}

type SCIMTokenView struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Prefix      string `json:"prefix"`
	CreatedAt   string `json:"created_at"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	Disabled    bool   `json:"disabled"`
}

type CreateSCIMTokenResult struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Token       string `json:"token"`
	Prefix      string `json:"prefix"`
	CreatedAt   string `json:"created_at"`
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("enterprise store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create enterprise store root: %w", err)
	}
	store := &Store{state: storage.NewFileJSONState(filepath.Join(root, "enterprise.json"))}
	if err := store.ensureInitialized(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("enterprise store root is required")
	}
	store := &Store{
		state: storage.NewBackendJSONState(backend, "enterprise/enterprise.json", filepath.Join(root, "enterprise.json")),
	}
	if err := store.ensureInitialized(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Get() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	return current.Settings, nil
}

func (s *Store) View() (SettingsView, error) {
	settings, err := s.Get()
	if err != nil {
		return SettingsView{}, err
	}
	return settingsToView(settings), nil
}

func (s *Store) Update(settings Settings) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	settings = normalizeSettings(settings, current.Settings)
	if err := validateSettings(settings); err != nil {
		return Settings{}, err
	}
	current.Settings = settings
	if err := s.saveLocked(current); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (s *Store) CreateSCIMToken(displayName string) (CreateSCIMTokenResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return CreateSCIMTokenResult{}, err
	}
	tokenValue, prefix, tokenHash, err := newSCIMToken()
	if err != nil {
		return CreateSCIMTokenResult{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := SCIMToken{
		ID:          "scim-" + strings.ToLower(hex.EncodeToString(randomBytes(8))),
		DisplayName: normalizeDisplayName(displayName),
		Prefix:      prefix,
		SecretHash:  tokenHash,
		CreatedAt:   now,
	}
	if item.DisplayName == "" {
		item.DisplayName = "SCIM Token"
	}
	current.Settings.SCIM.Tokens = append(current.Settings.SCIM.Tokens, item)
	sort.Slice(current.Settings.SCIM.Tokens, func(i, j int) bool {
		return current.Settings.SCIM.Tokens[i].CreatedAt > current.Settings.SCIM.Tokens[j].CreatedAt
	})
	if err := s.saveLocked(current); err != nil {
		return CreateSCIMTokenResult{}, err
	}
	return CreateSCIMTokenResult{
		ID:          item.ID,
		DisplayName: item.DisplayName,
		Token:       tokenValue,
		Prefix:      item.Prefix,
		CreatedAt:   item.CreatedAt,
	}, nil
}

func (s *Store) DeleteSCIMToken(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	needle := strings.TrimSpace(id)
	filtered := current.Settings.SCIM.Tokens[:0]
	found := false
	for _, item := range current.Settings.SCIM.Tokens {
		if item.ID == needle {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return fmt.Errorf("scim token %s not found", needle)
	}
	current.Settings.SCIM.Tokens = filtered
	return s.saveLocked(current)
}

func (s *Store) AuthenticateSCIMToken(raw string) (SCIMToken, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return SCIMToken{}, false, err
	}
	hash := hashToken(raw)
	for i := range current.Settings.SCIM.Tokens {
		item := current.Settings.SCIM.Tokens[i]
		if item.Disabled || item.SecretHash != hash {
			continue
		}
		current.Settings.SCIM.Tokens[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveLocked(current); err != nil {
			return SCIMToken{}, false, err
		}
		return current.Settings.SCIM.Tokens[i], true, nil
	}
	return SCIMToken{}, false, nil
}

func (s *Store) SignJSON(value any) (string, string, error) {
	settings, err := s.Get()
	if err != nil {
		return "", "", err
	}
	content, err := json.Marshal(value)
	if err != nil {
		return "", "", err
	}
	privateKey, err := decodePrivateKey(settings.Evidence.PrivateKeyPEM)
	if err != nil {
		return "", "", err
	}
	signature := ed25519.Sign(privateKey, content)
	return settings.Evidence.KeyID, base64.StdEncoding.EncodeToString(signature), nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{Settings: defaultSettings()}, nil
		}
		return nil, fmt.Errorf("read enterprise store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode enterprise store: %w", err)
	}
	current.Settings = normalizeSettings(current.Settings, defaultSettings())
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode enterprise store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write enterprise store: %w", err)
	}
	return nil
}

func (s *Store) ensureInitialized() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	current.Settings = normalizeSettings(current.Settings, defaultSettings())
	return s.saveLocked(current)
}

func defaultSettings() Settings {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	return Settings{
		OIDC: OIDCSettings{
			DisplayName:          "Enterprise SSO",
			Scopes:               []string{"openid", "profile", "email", "groups"},
			UsernameClaim:        "preferred_username",
			EmailClaim:           "email",
			FullNameClaim:        "name",
			GroupsClaim:          "groups",
			AutoProvision:        true,
			RequireVerifiedEmail: true,
		},
		Approvals: ApprovalPolicy{
			Enabled:           false,
			RequiredApprovals: 1,
			AllowSelfApproval: false,
			ReviewerRoleIDs:   []string{"admin", "manager"},
		},
		SCIM: SCIMSettings{},
		Evidence: EvidenceSettings{
			KeyID:         "evidence-" + strings.ToLower(hex.EncodeToString(randomBytes(6))),
			PublicKeyPEM:  encodePublicKey(publicKey),
			PrivateKeyPEM: encodePrivateKey(privateKey),
		},
	}
}

func normalizeSettings(settings Settings, defaults Settings) Settings {
	settings.OIDC.DisplayName = firstNonEmpty(settings.OIDC.DisplayName, defaults.OIDC.DisplayName)
	settings.OIDC.IssuerURL = strings.TrimSpace(settings.OIDC.IssuerURL)
	settings.OIDC.ClientID = strings.TrimSpace(settings.OIDC.ClientID)
	settings.OIDC.ClientSecret = firstNonEmpty(strings.TrimSpace(settings.OIDC.ClientSecret), defaults.OIDC.ClientSecret)
	settings.OIDC.RedirectURL = strings.TrimSpace(settings.OIDC.RedirectURL)
	settings.OIDC.Scopes = normalizeScopes(settings.OIDC.Scopes, defaults.OIDC.Scopes)
	settings.OIDC.UsernameClaim = firstNonEmpty(settings.OIDC.UsernameClaim, defaults.OIDC.UsernameClaim)
	settings.OIDC.EmailClaim = firstNonEmpty(settings.OIDC.EmailClaim, defaults.OIDC.EmailClaim)
	settings.OIDC.FullNameClaim = firstNonEmpty(settings.OIDC.FullNameClaim, defaults.OIDC.FullNameClaim)
	settings.OIDC.GroupsClaim = firstNonEmpty(settings.OIDC.GroupsClaim, defaults.OIDC.GroupsClaim)
	settings.OIDC.AllowedEmailDomains = normalizeStringList(settings.OIDC.AllowedEmailDomains)
	settings.OIDC.DefaultRoleIDs = normalizeRoleIDs(settings.OIDC.DefaultRoleIDs)
	settings.OIDC.GroupRoleMappings = normalizeRoleMappings(settings.OIDC.GroupRoleMappings)
	settings.Approvals.RequiredApprovals = max(1, settings.Approvals.RequiredApprovals)
	settings.Approvals.ReviewerRoleIDs = normalizeRoleIDs(settings.Approvals.ReviewerRoleIDs)
	if len(settings.Approvals.ReviewerRoleIDs) == 0 {
		settings.Approvals.ReviewerRoleIDs = append([]string(nil), defaults.Approvals.ReviewerRoleIDs...)
	}
	settings.SCIM.DefaultRoleIDs = normalizeRoleIDs(settings.SCIM.DefaultRoleIDs)
	settings.SCIM.GroupRoleMappings = normalizeRoleMappings(settings.SCIM.GroupRoleMappings)
	for i := range settings.SCIM.Tokens {
		settings.SCIM.Tokens[i].ID = strings.TrimSpace(settings.SCIM.Tokens[i].ID)
		settings.SCIM.Tokens[i].DisplayName = normalizeDisplayName(settings.SCIM.Tokens[i].DisplayName)
		settings.SCIM.Tokens[i].Prefix = strings.TrimSpace(settings.SCIM.Tokens[i].Prefix)
		settings.SCIM.Tokens[i].SecretHash = strings.TrimSpace(settings.SCIM.Tokens[i].SecretHash)
		settings.SCIM.Tokens[i].CreatedAt = strings.TrimSpace(settings.SCIM.Tokens[i].CreatedAt)
		settings.SCIM.Tokens[i].LastUsedAt = strings.TrimSpace(settings.SCIM.Tokens[i].LastUsedAt)
	}
	settings.Evidence.KeyID = firstNonEmpty(settings.Evidence.KeyID, defaults.Evidence.KeyID)
	settings.Evidence.PublicKeyPEM = firstNonEmpty(settings.Evidence.PublicKeyPEM, defaults.Evidence.PublicKeyPEM)
	settings.Evidence.PrivateKeyPEM = firstNonEmpty(settings.Evidence.PrivateKeyPEM, defaults.Evidence.PrivateKeyPEM)
	return settings
}

func validateSettings(settings Settings) error {
	if settings.OIDC.Enabled {
		if settings.OIDC.IssuerURL == "" {
			return errors.New("oidc issuer_url is required when oidc is enabled")
		}
		if settings.OIDC.ClientID == "" {
			return errors.New("oidc client_id is required when oidc is enabled")
		}
		if settings.OIDC.ClientSecret == "" {
			return errors.New("oidc client_secret is required when oidc is enabled")
		}
		if settings.OIDC.RedirectURL == "" {
			return errors.New("oidc redirect_url is required when oidc is enabled")
		}
	}
	if settings.Evidence.KeyID == "" || settings.Evidence.PublicKeyPEM == "" || settings.Evidence.PrivateKeyPEM == "" {
		return errors.New("evidence signing keys are required")
	}
	if _, err := decodePrivateKey(settings.Evidence.PrivateKeyPEM); err != nil {
		return err
	}
	return nil
}

func settingsToView(settings Settings) SettingsView {
	var view SettingsView
	view.OIDC.Enabled = settings.OIDC.Enabled
	view.OIDC.DisplayName = settings.OIDC.DisplayName
	view.OIDC.IssuerURL = settings.OIDC.IssuerURL
	view.OIDC.ClientID = settings.OIDC.ClientID
	view.OIDC.RedirectURL = settings.OIDC.RedirectURL
	view.OIDC.Scopes = append([]string(nil), settings.OIDC.Scopes...)
	view.OIDC.UsernameClaim = settings.OIDC.UsernameClaim
	view.OIDC.EmailClaim = settings.OIDC.EmailClaim
	view.OIDC.FullNameClaim = settings.OIDC.FullNameClaim
	view.OIDC.GroupsClaim = settings.OIDC.GroupsClaim
	view.OIDC.AllowedEmailDomains = append([]string(nil), settings.OIDC.AllowedEmailDomains...)
	view.OIDC.DefaultRoleIDs = append([]string(nil), settings.OIDC.DefaultRoleIDs...)
	view.OIDC.GroupRoleMappings = append([]RoleMapping(nil), settings.OIDC.GroupRoleMappings...)
	view.OIDC.AutoProvision = settings.OIDC.AutoProvision
	view.OIDC.RequireVerifiedEmail = settings.OIDC.RequireVerifiedEmail
	view.OIDC.HasClientSecret = strings.TrimSpace(settings.OIDC.ClientSecret) != ""
	view.Approvals = settings.Approvals
	view.SCIM.Enabled = settings.SCIM.Enabled
	view.SCIM.DefaultRoleIDs = append([]string(nil), settings.SCIM.DefaultRoleIDs...)
	view.SCIM.GroupRoleMappings = append([]RoleMapping(nil), settings.SCIM.GroupRoleMappings...)
	for _, token := range settings.SCIM.Tokens {
		view.SCIM.Tokens = append(view.SCIM.Tokens, SCIMTokenView{
			ID:          token.ID,
			DisplayName: token.DisplayName,
			Prefix:      token.Prefix,
			CreatedAt:   token.CreatedAt,
			LastUsedAt:  token.LastUsedAt,
			Disabled:    token.Disabled,
		})
	}
	view.Evidence.KeyID = settings.Evidence.KeyID
	view.Evidence.PublicKeyPEM = settings.Evidence.PublicKeyPEM
	return view
}

func newSCIMToken() (string, string, string, error) {
	value := "scim_" + strings.ToLower(base64.RawURLEncoding.EncodeToString(randomBytes(32)))
	if strings.TrimSpace(value) == "" {
		return "", "", "", errors.New("failed to create scim token")
	}
	prefix := value
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	return value, prefix, hashToken(value), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func normalizeScopes(items []string, defaults []string) []string {
	values := normalizeStringList(items)
	if len(values) == 0 {
		return append([]string(nil), defaults...)
	}
	return values
}

func normalizeStringList(items []string) []string {
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

func normalizeRoleIDs(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		value := strings.ToLower(strings.TrimSpace(item))
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

func normalizeRoleMappings(items []RoleMapping) []RoleMapping {
	out := make([]RoleMapping, 0, len(items))
	for _, item := range items {
		externalGroup := strings.TrimSpace(item.ExternalGroup)
		roleIDs := normalizeRoleIDs(item.RoleIDs)
		if externalGroup == "" || len(roleIDs) == 0 {
			continue
		}
		out = append(out, RoleMapping{
			ExternalGroup: externalGroup,
			RoleIDs:       roleIDs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExternalGroup < out[j].ExternalGroup })
	return out
}

func encodePrivateKey(key ed25519.PrivateKey) string {
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: []byte(key)}
	return string(pem.EncodeToMemory(block))
}

func encodePublicKey(key ed25519.PublicKey) string {
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: []byte(key)}
	return string(pem.EncodeToMemory(block))
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil {
		return nil, errors.New("invalid evidence private key")
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, errors.New("unsupported evidence private key")
	}
	return ed25519.PrivateKey(block.Bytes), nil
}

func normalizeDisplayName(value string) string {
	return strings.TrimSpace(value)
}

func randomBytes(size int) []byte {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return buf
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

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
