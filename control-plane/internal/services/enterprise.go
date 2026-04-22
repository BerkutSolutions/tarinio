package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/enterprise"
	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/users"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

type EnterpriseRevisionStore interface {
	Get(revisionID string) (revisions.Revision, bool, error)
	UpdateRevision(revision revisions.Revision) error
	Approve(revisionID string, approval revisions.ApprovalRecord, requiredApprovals int) (revisions.Revision, error)
	List() ([]revisions.Revision, error)
}

type EnterpriseUserStore interface {
	Get(id string) (users.User, bool, error)
	FindByUsername(username string) (users.User, bool, error)
	FindByEmail(email string) (users.User, bool, error)
	FindByExternalIdentity(authSource, externalID string) (users.User, bool, error)
	Create(user users.User) (users.User, error)
	Update(user users.User) (users.User, error)
	List() ([]users.User, error)
}

type EnterpriseRoleStore interface {
	List() ([]roles.Role, error)
	PermissionsForRoleIDs(roleIDs []string) []rbac.Permission
}

type EnterpriseSessionStore interface {
	CreateSession(userID string, username string, roleIDs []string, ttl time.Duration) (sessions.Session, error)
	CreateOIDCLoginChallenge(nextPath string, ttl time.Duration) (sessions.OIDCLoginChallenge, error)
	ConsumeOIDCLoginChallenge(state string) (sessions.OIDCLoginChallenge, bool, error)
}

type EnterpriseAuditStore interface {
	List(query audits.Query) (audits.ListResult, error)
}

type EnterpriseEventStore interface {
	List() ([]events.Event, error)
}

type EnterpriseJobStore interface {
	List() ([]jobs.Job, error)
}

type EnterpriseService struct {
	store      *enterprise.Store
	users      EnterpriseUserStore
	roles      EnterpriseRoleStore
	sessions   EnterpriseSessionStore
	revisions  EnterpriseRevisionStore
	audits     EnterpriseAuditStore
	events     EnterpriseEventStore
	jobs       EnterpriseJobStore
	sessionTTL time.Duration
	httpClient *http.Client
	auditLog   *AuditService
}

type OIDCProviderStatus struct {
	Enabled     bool   `json:"enabled"`
	DisplayName string `json:"display_name,omitempty"`
}

type OIDCCallbackResult struct {
	SessionID string
	NextPath  string
	User      AuthUser
}

type EnterpriseSettingsInput struct {
	OIDC      enterprise.OIDCSettings   `json:"oidc"`
	Approvals enterprise.ApprovalPolicy `json:"approvals"`
	SCIM      enterprise.SCIMSettings   `json:"scim"`
}

func NewEnterpriseService(store *enterprise.Store, users EnterpriseUserStore, roles EnterpriseRoleStore, sessions EnterpriseSessionStore, revisions EnterpriseRevisionStore, audits EnterpriseAuditStore, events EnterpriseEventStore, jobs EnterpriseJobStore, auditLog *AuditService) *EnterpriseService {
	return &EnterpriseService{
		store:      store,
		users:      users,
		roles:      roles,
		sessions:   sessions,
		revisions:  revisions,
		audits:     audits,
		events:     events,
		jobs:       jobs,
		sessionTTL: 12 * time.Hour,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		auditLog:   auditLog,
	}
}

func (s *EnterpriseService) GetSettingsView() (enterprise.SettingsView, error) {
	if s == nil || s.store == nil {
		return enterprise.SettingsView{}, errors.New("enterprise store is unavailable")
	}
	return s.store.View()
}

func (s *EnterpriseService) UpdateSettings(ctx context.Context, input EnterpriseSettingsInput) (enterprise.SettingsView, error) {
	if s == nil || s.store == nil {
		return enterprise.SettingsView{}, errors.New("enterprise store is unavailable")
	}
	current, err := s.store.Get()
	if err != nil {
		return enterprise.SettingsView{}, err
	}
	current.OIDC = input.OIDC
	current.Approvals = input.Approvals
	current.SCIM.Enabled = input.SCIM.Enabled
	current.SCIM.DefaultRoleIDs = append([]string(nil), input.SCIM.DefaultRoleIDs...)
	current.SCIM.GroupRoleMappings = append([]enterprise.RoleMapping(nil), input.SCIM.GroupRoleMappings...)
	if _, err := s.store.Update(current); err != nil {
		return enterprise.SettingsView{}, err
	}
	recordAudit(ctx, s.auditLog, audits.AuditEvent{
		Action:       "enterprise.settings_update",
		ResourceType: "enterprise_settings",
		ResourceID:   "default",
		Status:       auditStatus(err),
		Summary:      "enterprise settings updated",
	})
	return s.store.View()
}

func (s *EnterpriseService) CreateSCIMToken(ctx context.Context, displayName string) (enterprise.CreateSCIMTokenResult, error) {
	result, err := s.store.CreateSCIMToken(displayName)
	recordAudit(ctx, s.auditLog, audits.AuditEvent{
		Action:       "enterprise.scim_token_create",
		ResourceType: "scim_token",
		ResourceID:   result.ID,
		Status:       auditStatus(err),
		Summary:      "scim provisioning token created",
	})
	return result, err
}

func (s *EnterpriseService) DeleteSCIMToken(ctx context.Context, id string) error {
	err := s.store.DeleteSCIMToken(id)
	recordAudit(ctx, s.auditLog, audits.AuditEvent{
		Action:       "enterprise.scim_token_delete",
		ResourceType: "scim_token",
		ResourceID:   strings.TrimSpace(id),
		Status:       auditStatus(err),
		Summary:      "scim provisioning token deleted",
	})
	return err
}

func (s *EnterpriseService) ProviderStatus() (OIDCProviderStatus, error) {
	settings, err := s.store.Get()
	if err != nil {
		return OIDCProviderStatus{}, err
	}
	return OIDCProviderStatus{
		Enabled:     settings.OIDC.Enabled,
		DisplayName: settings.OIDC.DisplayName,
	}, nil
}

func (s *EnterpriseService) BeginOIDCLogin(nextPath string) (string, error) {
	settings, err := s.store.Get()
	if err != nil {
		return "", err
	}
	if !settings.OIDC.Enabled {
		return "", errors.New("oidc is disabled")
	}
	discovery, err := s.fetchOIDCDiscovery(settings.OIDC.IssuerURL)
	if err != nil {
		return "", err
	}
	challenge, err := s.sessions.CreateOIDCLoginChallenge(normalizeNextPath(nextPath), 10*time.Minute)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("client_id", settings.OIDC.ClientID)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(settings.OIDC.Scopes, " "))
	query.Set("redirect_uri", settings.OIDC.RedirectURL)
	query.Set("state", challenge.State)
	query.Set("nonce", challenge.Nonce)
	return discovery.AuthorizationEndpoint + "?" + query.Encode(), nil
}

func (s *EnterpriseService) HandleOIDCCallback(ctx context.Context, state, code string) (OIDCCallbackResult, error) {
	settings, err := s.store.Get()
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	if !settings.OIDC.Enabled {
		return OIDCCallbackResult{}, errors.New("oidc is disabled")
	}
	challenge, ok, err := s.sessions.ConsumeOIDCLoginChallenge(state)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	if !ok {
		return OIDCCallbackResult{}, errors.New("oidc login challenge not found")
	}
	discovery, err := s.fetchOIDCDiscovery(settings.OIDC.IssuerURL)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	tokenResponse, err := s.exchangeOIDCCode(discovery.TokenEndpoint, settings.OIDC, code)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	claims, err := s.verifyIDToken(discovery, settings.OIDC, tokenResponse.IDToken, challenge.Nonce)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	user, err := s.resolveOIDCUser(settings.OIDC, claims)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	session, err := s.sessions.CreateSession(user.ID, user.Username, user.RoleIDs, s.sessionTTL)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	authUser := AuthUser{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    user.FullName,
		Department:  user.Department,
		Position:    user.Position,
		Language:    user.Language,
		RoleIDs:     append([]string(nil), user.RoleIDs...),
		Permissions: permissionsToStrings(s.roles.PermissionsForRoleIDs(user.RoleIDs)),
	}
	recordAudit(ctx, s.auditLog, audits.AuditEvent{
		ActorUserID:  user.ID,
		Action:       "auth.oidc_login",
		ResourceType: "auth",
		ResourceID:   user.ID,
		Status:       audits.StatusSucceeded,
		Summary:      "oidc login",
	})
	return OIDCCallbackResult{
		SessionID: session.ID,
		NextPath:  firstNonEmptyValue(challenge.NextPath, "/healthcheck"),
		User:      authUser,
	}, nil
}

func (s *EnterpriseService) PrepareCompiledRevision(ctx context.Context, revision revisions.Revision) (revisions.Revision, error) {
	settings, err := s.store.Get()
	if err != nil {
		return revision, err
	}
	session, _ := auth.SessionFromContext(ctx)
	revision.CompiledByUserID = session.UserID
	revision.CompiledByName = session.Username
	if settings.Approvals.Enabled {
		revision.ApprovalStatus = revisions.ApprovalPending
		revision.RequiredApprovals = settings.Approvals.RequiredApprovals
		revision.Status = revisions.StatusPendingApproval
	} else {
		revision.ApprovalStatus = revisions.ApprovalNotRequired
		revision.RequiredApprovals = 0
		revision.Status = revisions.StatusPending
	}
	keyID, signature, err := s.store.SignJSON(map[string]any{
		"id":                  revision.ID,
		"version":             revision.Version,
		"created_at":          revision.CreatedAt,
		"checksum":            revision.Checksum,
		"compiled_by_user_id": revision.CompiledByUserID,
		"approval_status":     revision.ApprovalStatus,
		"required_approvals":  revision.RequiredApprovals,
	})
	if err != nil {
		return revision, err
	}
	revision.SignatureKeyID = keyID
	revision.Signature = signature
	return revision, nil
}

func (s *EnterpriseService) EnsureRevisionCanApply(revision revisions.Revision) error {
	if revision.ApprovalStatus == revisions.ApprovalPending {
		return fmt.Errorf("revision %s is waiting for approval", revision.ID)
	}
	return nil
}

func (s *EnterpriseService) ApproveRevision(ctx context.Context, revisionID, comment string) (revisions.Revision, error) {
	settings, err := s.store.Get()
	if err != nil {
		return revisions.Revision{}, err
	}
	session, ok := auth.SessionFromContext(ctx)
	if !ok {
		return revisions.Revision{}, errors.New("approval requires authenticated session")
	}
	if !slices.ContainsFunc(settings.Approvals.ReviewerRoleIDs, func(roleID string) bool {
		return slices.Contains(session.RoleIDs, roleID)
	}) && !slices.Contains(session.RoleIDs, "admin") {
		return revisions.Revision{}, errors.New("user is not allowed to approve revisions")
	}
	revision, exists, err := s.revisions.Get(revisionID)
	if err != nil {
		return revisions.Revision{}, err
	}
	if !exists {
		return revisions.Revision{}, fmt.Errorf("revision %s not found", revisionID)
	}
	if settings.Approvals.Enabled && !settings.Approvals.AllowSelfApproval && revision.CompiledByUserID != "" && revision.CompiledByUserID == session.UserID {
		return revisions.Revision{}, errors.New("self approval is disabled")
	}
	updated, err := s.revisions.Approve(revisionID, revisions.ApprovalRecord{
		UserID:     session.UserID,
		Username:   session.Username,
		Comment:    strings.TrimSpace(comment),
		ApprovedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}, settings.Approvals.RequiredApprovals)
	recordAudit(ctx, s.auditLog, audits.AuditEvent{
		ActorUserID:       session.UserID,
		Action:            "revision.approve",
		ResourceType:      "revision",
		ResourceID:        revisionID,
		RelatedRevisionID: revisionID,
		Status:            auditStatus(err),
		Summary:           "revision approved",
		Details: map[string]any{
			"comment": strings.TrimSpace(comment),
		},
	})
	return updated, err
}

func (s *EnterpriseService) ProvisionSCIMUser(ctx context.Context, input SCIMProvisionUserInput) (users.User, error) {
	settings, err := s.store.Get()
	if err != nil {
		return users.User{}, err
	}
	if !settings.SCIM.Enabled {
		return users.User{}, errors.New("scim is disabled")
	}
	return s.resolveExternalUser("scim", input.ExternalID, input.UserName, input.Email, input.FullName, input.Department, input.Position, input.Active, input.Groups, settings.SCIM.DefaultRoleIDs, settings.SCIM.GroupRoleMappings)
}

func (s *EnterpriseService) BuildSupportBundle() ([]byte, string, error) {
	settings, err := s.store.Get()
	if err != nil {
		return nil, "", err
	}
	auditItems, err := s.audits.List(audits.Query{Limit: 10000})
	if err != nil {
		return nil, "", err
	}
	eventItems, err := s.events.List()
	if err != nil {
		return nil, "", err
	}
	jobItems, err := s.jobs.List()
	if err != nil {
		return nil, "", err
	}
	revisionItems, err := s.revisions.List()
	if err != nil {
		return nil, "", err
	}
	auditChain := audits.SummarizeChain(auditItems.Items)
	revisionSignatureCount := 0
	for _, item := range revisionItems {
		if strings.TrimSpace(item.Signature) != "" && strings.TrimSpace(item.SignatureKeyID) != "" {
			revisionSignatureCount++
		}
	}
	archive := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(archive)
	tw := tar.NewWriter(gz)
	files := map[string][]byte{}
	files["evidence/public_key.pem"] = []byte(settings.Evidence.PublicKeyPEM)
	if files["audits.json"], err = json.MarshalIndent(auditItems, "", "  "); err != nil {
		return nil, "", err
	}
	if files["events.json"], err = json.MarshalIndent(eventItems, "", "  "); err != nil {
		return nil, "", err
	}
	if files["jobs.json"], err = json.MarshalIndent(jobItems, "", "  "); err != nil {
		return nil, "", err
	}
	if files["revisions.json"], err = json.MarshalIndent(revisionItems, "", "  "); err != nil {
		return nil, "", err
	}
	fileDigests := make(map[string]map[string]any, len(files))
	fileNames := make([]string, 0, len(files)+2)
	for name, content := range files {
		fileNames = append(fileNames, name)
		fileDigests[name] = map[string]any{
			"sha256": sha256Hex(content),
			"size":   len(content) + 1,
		}
	}
	slices.Sort(fileNames)
	manifest := map[string]any{
		"generated_at":                time.Now().UTC().Format(time.RFC3339Nano),
		"bundle_format":               "tarinio-support-bundle/v2",
		"public_key_id":               settings.Evidence.KeyID,
		"audit_items":                 len(auditItems.Items),
		"event_items":                 len(eventItems),
		"job_items":                   len(jobItems),
		"revision_items":              len(revisionItems),
		"signed_revision_items":       revisionSignatureCount,
		"audit_chain":                 auditChain,
		"files":                       fileNames,
		"file_digests":                fileDigests,
		"verification_recommendation": "verify manifest signature, file sha256 values, audit chain continuity, and revision signatures before sharing the bundle externally",
	}
	if files["manifest.json"], err = json.MarshalIndent(manifest, "", "  "); err != nil {
		return nil, "", err
	}
	keyID, signature, err := s.store.SignJSON(manifest)
	if err != nil {
		return nil, "", err
	}
	if files["signature.json"], err = json.MarshalIndent(map[string]any{
		"algorithm":       "ed25519",
		"key_id":          keyID,
		"signature":       signature,
		"signed_object":   "manifest.json",
		"manifest_sha256": sha256Hex(files["manifest.json"]),
	}, "", "  "); err != nil {
		return nil, "", err
	}
	fileNames = append(fileNames, "manifest.json", "signature.json")
	slices.Sort(fileNames)
	for _, name := range fileNames {
		content := files[name]
		if err := writeTarFile(tw, name, append(content, '\n')); err != nil {
			return nil, "", err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, "", err
	}
	if err := gz.Close(); err != nil {
		return nil, "", err
	}
	return archive.Bytes(), fmt.Sprintf("tarinio-support-bundle-%s.tar.gz", time.Now().UTC().Format("20060102-150405")), nil
}

func (s *EnterpriseService) AuthenticateSCIMBearer(raw string) error {
	settings, err := s.store.Get()
	if err != nil {
		return err
	}
	if !settings.SCIM.Enabled {
		return errors.New("scim is disabled")
	}
	if _, ok, err := s.store.AuthenticateSCIMToken(raw); err != nil {
		return err
	} else if !ok {
		return errors.New("invalid scim token")
	}
	return nil
}

func (s *EnterpriseService) ListSCIMUsers() ([]SCIMUserRecord, error) {
	items, err := s.users.List()
	if err != nil {
		return nil, err
	}
	out := make([]SCIMUserRecord, 0, len(items))
	for _, item := range items {
		if item.AuthSource != "scim" && item.AuthSource != "oidc" {
			continue
		}
		out = append(out, scimUserFromUser(item))
	}
	return out, nil
}

func (s *EnterpriseService) GetSCIMUser(id string) (SCIMUserRecord, bool, error) {
	item, ok, err := s.users.Get(id)
	if err != nil || !ok {
		return SCIMUserRecord{}, ok, err
	}
	return scimUserFromUser(item), true, nil
}

func (s *EnterpriseService) ListSCIMGroups() ([]SCIMGroupRecord, error) {
	settings, err := s.store.Get()
	if err != nil {
		return nil, err
	}
	out := make([]SCIMGroupRecord, 0, len(settings.SCIM.GroupRoleMappings))
	for _, item := range settings.SCIM.GroupRoleMappings {
		out = append(out, SCIMGroupRecord{
			ID:          item.ExternalGroup,
			DisplayName: item.ExternalGroup,
			RoleIDs:     append([]string(nil), item.RoleIDs...),
		})
	}
	return out, nil
}

func (s *EnterpriseService) DeactivateSCIMUser(id string) (SCIMUserRecord, error) {
	current, ok, err := s.users.Get(id)
	if err != nil {
		return SCIMUserRecord{}, err
	}
	if !ok {
		return SCIMUserRecord{}, fmt.Errorf("user %s not found", id)
	}
	current.IsActive = false
	updated, err := s.users.Update(current)
	if err != nil {
		return SCIMUserRecord{}, err
	}
	return scimUserFromUser(updated), nil
}

type SCIMProvisionUserInput struct {
	ExternalID string
	UserName   string
	Email      string
	FullName   string
	Department string
	Position   string
	Active     bool
	Groups     []string
}

type SCIMUserRecord struct {
	ID         string   `json:"id"`
	UserName   string   `json:"userName"`
	ExternalID string   `json:"externalId,omitempty"`
	Active     bool     `json:"active"`
	Email      string   `json:"email,omitempty"`
	FullName   string   `json:"full_name,omitempty"`
	Department string   `json:"department,omitempty"`
	Position   string   `json:"position,omitempty"`
	Groups     []string `json:"groups,omitempty"`
}

type SCIMGroupRecord struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	RoleIDs     []string `json:"role_ids"`
}

type oidcDiscoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type oidcTokenResponse struct {
	IDToken string `json:"id_token"`
}

func (s *EnterpriseService) fetchOIDCDiscovery(issuerURL string) (oidcDiscoveryDocument, error) {
	url := strings.TrimRight(strings.TrimSpace(issuerURL), "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return oidcDiscoveryDocument{}, fmt.Errorf("oidc discovery returned %d", resp.StatusCode)
	}
	var discovery oidcDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return oidcDiscoveryDocument{}, err
	}
	return discovery, nil
}

func (s *EnterpriseService) exchangeOIDCCode(tokenEndpoint string, settings enterprise.OIDCSettings, code string) (oidcTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", settings.RedirectURL)
	form.Set("client_id", settings.ClientID)
	form.Set("client_secret", settings.ClientSecret)
	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return oidcTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return oidcTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return oidcTokenResponse{}, fmt.Errorf("oidc token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload oidcTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return oidcTokenResponse{}, err
	}
	if strings.TrimSpace(payload.IDToken) == "" {
		return oidcTokenResponse{}, errors.New("oidc token response does not contain id_token")
	}
	return payload, nil
}

func (s *EnterpriseService) verifyIDToken(discovery oidcDiscoveryDocument, settings enterprise.OIDCSettings, rawToken, expectedNonce string) (jwt.MapClaims, error) {
	keySet, err := s.fetchJWKSet(discovery.JWKSURI)
	if err != nil {
		return nil, err
	}
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		for _, key := range keySet.Keys {
			if key.KeyID != kid {
				continue
			}
			return key.Key, nil
		}
		if len(keySet.Keys) == 1 {
			return keySet.Keys[0].Key, nil
		}
		return nil, errors.New("oidc signing key not found")
	}, jwt.WithAudience(settings.ClientID), jwt.WithIssuer(discovery.Issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid oidc id_token")
	}
	if nonce := strings.TrimSpace(asStringClaim(claims, "nonce")); nonce != strings.TrimSpace(expectedNonce) {
		return nil, errors.New("oidc nonce validation failed")
	}
	if settings.RequireVerifiedEmail {
		if verified, _ := claims["email_verified"].(bool); !verified {
			return nil, errors.New("oidc email is not verified")
		}
	}
	return claims, nil
}

func (s *EnterpriseService) fetchJWKSet(jwksURI string) (jose.JSONWebKeySet, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSpace(jwksURI), nil)
	if err != nil {
		return jose.JSONWebKeySet{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return jose.JSONWebKeySet{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jose.JSONWebKeySet{}, fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}
	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return jose.JSONWebKeySet{}, err
	}
	return keySet, nil
}

func (s *EnterpriseService) resolveOIDCUser(settings enterprise.OIDCSettings, claims jwt.MapClaims) (users.User, error) {
	subject := asStringClaim(claims, "sub")
	username := firstNonEmptyValue(asStringClaim(claims, settings.UsernameClaim), asStringClaim(claims, "preferred_username"), asStringClaim(claims, "email"))
	email := firstNonEmptyValue(asStringClaim(claims, settings.EmailClaim), asStringClaim(claims, "email"))
	fullName := firstNonEmptyValue(asStringClaim(claims, settings.FullNameClaim), asStringClaim(claims, "name"), username)
	if len(settings.AllowedEmailDomains) > 0 && email != "" {
		allowed := false
		for _, item := range settings.AllowedEmailDomains {
			if strings.HasSuffix(strings.ToLower(email), "@"+strings.ToLower(item)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return users.User{}, errors.New("oidc email domain is not allowed")
		}
	}
	if !settings.AutoProvision {
		user, ok, err := s.users.FindByExternalIdentity("oidc", subject)
		if err != nil {
			return users.User{}, err
		}
		if !ok {
			return users.User{}, errors.New("oidc user is not provisioned")
		}
		return user, nil
	}
	return s.resolveExternalUser("oidc", subject, username, email, fullName, "", "", true, claimStringSlice(claims, settings.GroupsClaim), settings.DefaultRoleIDs, settings.GroupRoleMappings)
}

func (s *EnterpriseService) resolveExternalUser(authSource, externalID, username, email, fullName, department, position string, active bool, groups []string, defaultRoles []string, mappings []enterprise.RoleMapping) (users.User, error) {
	existing, ok, err := s.users.FindByExternalIdentity(authSource, externalID)
	if err != nil {
		return users.User{}, err
	}
	roleIDs := resolveMappedRoles(groups, defaultRoles, mappings)
	if !ok {
		if existingByUsername, usernameExists, err := s.users.FindByUsername(username); err != nil {
			return users.User{}, err
		} else if usernameExists && existingByUsername.ExternalID == "" {
			return users.User{}, fmt.Errorf("username %s already exists and is not linked to %s", username, authSource)
		}
		passwordHash, err := users.HashPassword(base64.RawURLEncoding.EncodeToString([]byte(externalID + time.Now().UTC().Format(time.RFC3339Nano))))
		if err != nil {
			return users.User{}, err
		}
		created, err := s.users.Create(users.User{
			ID:             username,
			Username:       username,
			Email:          email,
			FullName:       fullName,
			Department:     department,
			Position:       position,
			AuthSource:     authSource,
			ExternalID:     externalID,
			ExternalGroups: append([]string(nil), groups...),
			LastSyncedAt:   time.Now().UTC().Format(time.RFC3339Nano),
			PasswordHash:   passwordHash,
			IsActive:       active,
			RoleIDs:        roleIDs,
		})
		if err != nil {
			return users.User{}, err
		}
		return created, nil
	}
	existing.Username = username
	existing.Email = email
	existing.FullName = fullName
	existing.Department = department
	existing.Position = position
	existing.AuthSource = authSource
	existing.ExternalID = externalID
	existing.ExternalGroups = append([]string(nil), groups...)
	existing.LastSyncedAt = time.Now().UTC().Format(time.RFC3339Nano)
	existing.IsActive = active
	existing.RoleIDs = roleIDs
	return s.users.Update(existing)
}

func resolveMappedRoles(groups []string, defaultRoles []string, mappings []enterprise.RoleMapping) []string {
	set := map[string]struct{}{}
	for _, roleID := range defaultRoles {
		set[strings.ToLower(strings.TrimSpace(roleID))] = struct{}{}
	}
	for _, group := range groups {
		for _, mapping := range mappings {
			if !strings.EqualFold(strings.TrimSpace(group), strings.TrimSpace(mapping.ExternalGroup)) {
				continue
			}
			for _, roleID := range mapping.RoleIDs {
				set[strings.ToLower(strings.TrimSpace(roleID))] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for roleID := range set {
		if roleID == "" {
			continue
		}
		out = append(out, roleID)
	}
	slices.Sort(out)
	return out
}

func asStringClaim(claims jwt.MapClaims, key string) string {
	if key == "" {
		return ""
	}
	value, ok := claims[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func claimStringSlice(claims jwt.MapClaims, key string) []string {
	if key == "" {
		return nil
	}
	value, ok := claims[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func writeTarFile(tw *tar.Writer, name string, content []byte) error {
	header := &tar.Header{
		Name:    filepath.ToSlash(name),
		Mode:    0o644,
		Size:    int64(len(content)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write(content)
	return err
}

func normalizeNextPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return "/healthcheck"
	}
	return trimmed
}

func permissionsToStrings(items []rbac.Permission) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, string(item))
	}
	return out
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func scimUserFromUser(item users.User) SCIMUserRecord {
	return SCIMUserRecord{
		ID:         item.ID,
		UserName:   item.Username,
		ExternalID: item.ExternalID,
		Active:     item.IsActive,
		Email:      item.Email,
		FullName:   item.FullName,
		Department: item.Department,
		Position:   item.Position,
		Groups:     append([]string(nil), item.ExternalGroups...),
	}
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
