package services

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type ACMEClientConfig struct {
	Email        string
	DirectoryURL string
	StateDir     string
	ChallengeDir string
}

type ACMELetsEncryptClient struct {
	cfg ACMEClientConfig
	mu  sync.Mutex
}

func NewACMELetsEncryptClient(cfg ACMEClientConfig) (*ACMELetsEncryptClient, error) {
	cfg.Email = strings.TrimSpace(cfg.Email)
	cfg.DirectoryURL = strings.TrimSpace(cfg.DirectoryURL)
	cfg.StateDir = strings.TrimSpace(cfg.StateDir)
	cfg.ChallengeDir = strings.TrimSpace(cfg.ChallengeDir)
	if cfg.Email == "" {
		return nil, fmt.Errorf("acme email is required")
	}
	if cfg.DirectoryURL == "" {
		return nil, fmt.Errorf("acme directory url is required")
	}
	if cfg.StateDir == "" {
		return nil, fmt.Errorf("acme state dir is required")
	}
	if cfg.ChallengeDir == "" {
		return nil, fmt.Errorf("acme challenge dir is required")
	}
	if err := os.MkdirAll(cfg.StateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create acme state dir: %w", err)
	}
	if err := os.MkdirAll(cfg.ChallengeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create acme challenge dir: %w", err)
	}
	return &ACMELetsEncryptClient{cfg: cfg}, nil
}

func (c *ACMELetsEncryptClient) Issue(commonName string, sanList []string) (IssuedMaterial, error) {
	return c.obtain(commonName, sanList)
}

func (c *ACMELetsEncryptClient) Renew(commonName string, sanList []string) (IssuedMaterial, error) {
	return c.obtain(commonName, sanList)
}

func (c *ACMELetsEncryptClient) obtain(commonName string, sanList []string) (IssuedMaterial, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	domains, err := normalizeDomains(commonName, sanList)
	if err != nil {
		return IssuedMaterial{}, err
	}
	user, err := c.loadOrCreateUser()
	if err != nil {
		return IssuedMaterial{}, err
	}

	legoCfg := lego.NewConfig(user)
	legoCfg.CADirURL = c.cfg.DirectoryURL
	legoCfg.Certificate.KeyType = certcrypto.EC256
	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return IssuedMaterial{}, fmt.Errorf("init acme client: %w", err)
	}

	provider := &acmeHTTPChallengeProvider{dir: c.cfg.ChallengeDir}
	if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
		return IssuedMaterial{}, fmt.Errorf("configure http-01 challenge provider: %w", err)
	}

	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return IssuedMaterial{}, fmt.Errorf("register acme account: %w", err)
		}
		user.Registration = reg
		if err := c.saveUser(user); err != nil {
			return IssuedMaterial{}, err
		}
	}

	issued, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return IssuedMaterial{}, fmt.Errorf("issue certificate: %w", err)
	}

	notBefore, notAfter, err := parseCertificateValidity(issued.Certificate)
	if err != nil {
		return IssuedMaterial{}, err
	}

	return IssuedMaterial{
		CertificatePEM: issued.Certificate,
		PrivateKeyPEM:  issued.PrivateKey,
		NotBefore:      notBefore.Format(time.RFC3339),
		NotAfter:       notAfter.Format(time.RFC3339),
	}, nil
}

func normalizeDomains(commonName string, sanList []string) ([]string, error) {
	seen := map[string]struct{}{}
	add := func(in string, out *[]string) {
		value := strings.ToLower(strings.TrimSpace(in))
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		*out = append(*out, value)
	}

	domains := make([]string, 0, len(sanList)+1)
	add(commonName, &domains)
	for _, item := range sanList {
		add(item, &domains)
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("common name is required")
	}
	return domains, nil
}

type acmeUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration,omitempty"`
	KeyPEM       string                 `json:"key_pem"`
	key          crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string {
	return u.Email
}

func (u *acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (c *ACMELetsEncryptClient) userStatePath() string {
	return filepath.Join(c.cfg.StateDir, "account.json")
}

func (c *ACMELetsEncryptClient) loadOrCreateUser() (*acmeUser, error) {
	content, err := os.ReadFile(c.userStatePath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read acme account state: %w", err)
		}
		return c.createUser()
	}
	var user acmeUser
	if err := json.Unmarshal(content, &user); err != nil {
		return nil, fmt.Errorf("decode acme account state: %w", err)
	}
	key, err := parsePrivateKeyPEM([]byte(user.KeyPEM))
	if err != nil {
		return nil, err
	}
	user.key = key
	user.Email = c.cfg.Email
	return &user, nil
}

func (c *ACMELetsEncryptClient) createUser() (*acmeUser, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate acme account key: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal acme account key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	user := &acmeUser{
		Email:  c.cfg.Email,
		KeyPEM: string(keyPEM),
		key:    key,
	}
	if err := c.saveUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (c *ACMELetsEncryptClient) saveUser(user *acmeUser) error {
	content, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("encode acme account state: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(c.userStatePath(), content, 0o600); err != nil {
		return fmt.Errorf("write acme account state: %w", err)
	}
	return nil
}

func parsePrivateKeyPEM(content []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("decode acme account key: invalid pem")
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse acme account key: %w", err)
	}
	return pkcs8Key, nil
}

func parseCertificateValidity(certPEM []byte) (time.Time, time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("decode issued certificate: invalid pem")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse issued certificate: %w", err)
	}
	return parsed.NotBefore.UTC(), parsed.NotAfter.UTC(), nil
}

type acmeHTTPChallengeProvider struct {
	dir string
}

var _ challenge.Provider = (*acmeHTTPChallengeProvider)(nil)

func (p *acmeHTTPChallengeProvider) Present(_ string, token, keyAuth string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("acme challenge token is required")
	}
	if strings.Contains(token, "/") || strings.Contains(token, "\\") {
		return fmt.Errorf("acme challenge token is invalid")
	}
	path := filepath.Join(p.dir, token)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create acme challenge dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(keyAuth)), 0o644); err != nil {
		return fmt.Errorf("write acme challenge file: %w", err)
	}
	return nil
}

func (p *acmeHTTPChallengeProvider) CleanUp(_ string, token, _ string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	if strings.Contains(token, "/") || strings.Contains(token, "\\") {
		return nil
	}
	path := filepath.Join(p.dir, token)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove acme challenge file: %w", err)
	}
	return nil
}
