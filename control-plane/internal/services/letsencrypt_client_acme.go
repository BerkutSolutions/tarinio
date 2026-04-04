package services

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
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

const (
	letsEncryptDirectoryURL        = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStagingDirectoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	zeroSSLDirectoryURL            = "https://acme.zerossl.com/v2/DV90"
)

var envVarKeyRegexp = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

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

func (c *ACMELetsEncryptClient) Issue(commonName string, sanList []string, options *ACMEIssueOptions) (IssuedMaterial, error) {
	return c.obtain(commonName, sanList, options)
}

func (c *ACMELetsEncryptClient) Renew(commonName string, sanList []string, options *ACMEIssueOptions) (IssuedMaterial, error) {
	return c.obtain(commonName, sanList, options)
}

func (c *ACMELetsEncryptClient) obtain(commonName string, sanList []string, options *ACMEIssueOptions) (IssuedMaterial, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	domains, err := normalizeDomains(commonName, sanList)
	if err != nil {
		return IssuedMaterial{}, err
	}
	resolvedOptions, err := c.resolveIssueOptions(options)
	if err != nil {
		return IssuedMaterial{}, err
	}
	restoreEnv, err := applyProcessEnv(resolvedOptions.DNSProviderEnv)
	if err != nil {
		return IssuedMaterial{}, err
	}
	defer restoreEnv()

	user, err := c.loadOrCreateUser(resolvedOptions.AccountEmail, resolvedOptions.DirectoryURL)
	if err != nil {
		return IssuedMaterial{}, err
	}

	legoCfg := lego.NewConfig(user)
	legoCfg.CADirURL = resolvedOptions.DirectoryURL
	legoCfg.Certificate.KeyType = certcrypto.EC256
	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return IssuedMaterial{}, fmt.Errorf("init acme client: %w", err)
	}

	if resolvedOptions.ChallengeType == "dns-01" {
		provider, err := buildDNSChallengeProvider(resolvedOptions)
		if err != nil {
			return IssuedMaterial{}, fmt.Errorf("configure dns-01 provider: %w", err)
		}
		dnsOptions := make([]dns01.ChallengeOption, 0, 2)
		if len(resolvedOptions.DNSResolvers) > 0 {
			dnsOptions = append(dnsOptions, dns01.AddRecursiveNameservers(resolvedOptions.DNSResolvers))
		}
		if resolvedOptions.DNSPropagationSeconds > 0 {
			dnsOptions = append(dnsOptions, dns01.PropagationWait(time.Duration(resolvedOptions.DNSPropagationSeconds)*time.Second, true))
		}
		if err := client.Challenge.SetDNS01Provider(provider, dnsOptions...); err != nil {
			return IssuedMaterial{}, fmt.Errorf("configure dns-01 challenge provider: %w", err)
		}
	} else {
		provider := &acmeHTTPChallengeProvider{dir: c.cfg.ChallengeDir}
		if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
			return IssuedMaterial{}, fmt.Errorf("configure http-01 challenge provider: %w", err)
		}
	}

	if user.Registration == nil {
		var reg *registration.Resource
		if resolvedOptions.CertificateAuthorityServer == "zerossl" {
			reg, err = client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
				TermsOfServiceAgreed: true,
				Kid:                  resolvedOptions.ZeroSSLEABKID,
				HmacEncoded:          resolvedOptions.ZeroSSLEABHMACKey,
			})
			if err != nil {
				return IssuedMaterial{}, fmt.Errorf("register ZeroSSL account: %w", err)
			}
		} else {
			reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err != nil {
				return IssuedMaterial{}, fmt.Errorf("register acme account: %w", err)
			}
		}
		user.Registration = reg
		if err := c.saveUser(user, resolvedOptions.AccountEmail, resolvedOptions.DirectoryURL); err != nil {
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

type resolvedACMEIssueOptions struct {
	CertificateAuthorityServer string
	DirectoryURL               string
	AccountEmail               string
	ChallengeType              string
	DNSProvider                string
	DNSProviderEnv             map[string]string
	DNSResolvers               []string
	DNSPropagationSeconds      int
	ZeroSSLEABKID              string
	ZeroSSLEABHMACKey          string
}

func (c *ACMELetsEncryptClient) resolveIssueOptions(options *ACMEIssueOptions) (resolvedACMEIssueOptions, error) {
	resolved := resolvedACMEIssueOptions{
		CertificateAuthorityServer: "letsencrypt",
		DirectoryURL:               strings.TrimSpace(c.cfg.DirectoryURL),
		AccountEmail:               strings.TrimSpace(c.cfg.Email),
		ChallengeType:              "http-01",
		DNSProviderEnv:             map[string]string{},
		DNSResolvers:               []string{},
	}
	if options == nil {
		return resolved, nil
	}

	if value := strings.ToLower(strings.TrimSpace(options.CertificateAuthorityServer)); value != "" {
		resolved.CertificateAuthorityServer = value
	}
	if value := strings.TrimSpace(options.AccountEmail); value != "" {
		resolved.AccountEmail = value
	}
	if value := strings.ToLower(strings.TrimSpace(options.ChallengeType)); value != "" {
		resolved.ChallengeType = value
	}
	if value := strings.TrimSpace(options.ZeroSSLEABKID); value != "" {
		resolved.ZeroSSLEABKID = value
	}
	if value := strings.TrimSpace(options.ZeroSSLEABHMACKey); value != "" {
		resolved.ZeroSSLEABHMACKey = value
	}
	if value := strings.TrimSpace(options.CustomDirectoryURL); value != "" {
		resolved.DirectoryURL = value
	}
	if len(options.DNSProviderEnv) > 0 {
		envValues := make(map[string]string, len(options.DNSProviderEnv))
		for key, value := range options.DNSProviderEnv {
			key = strings.ToUpper(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			if key == "" || value == "" {
				continue
			}
			if !envVarKeyRegexp.MatchString(key) {
				return resolvedACMEIssueOptions{}, fmt.Errorf("dns provider env key %q is invalid", key)
			}
			envValues[key] = value
		}
		resolved.DNSProviderEnv = envValues
	}

	resolvers := make([]string, 0, len(options.DNSResolvers))
	for _, value := range options.DNSResolvers {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		resolvers = append(resolvers, value)
	}
	resolved.DNSResolvers = resolvers
	if options.DNSPropagationSeconds > 0 {
		resolved.DNSPropagationSeconds = options.DNSPropagationSeconds
	}
	if value := strings.ToLower(strings.TrimSpace(options.DNSProvider)); value != "" {
		resolved.DNSProvider = value
	}

	switch resolved.CertificateAuthorityServer {
	case "letsencrypt":
		if options.UseLetsEncryptStaging {
			resolved.DirectoryURL = letsEncryptStagingDirectoryURL
		} else if strings.TrimSpace(options.CustomDirectoryURL) == "" {
			resolved.DirectoryURL = letsEncryptDirectoryURL
		}
	case "zerossl":
		if strings.TrimSpace(options.CustomDirectoryURL) == "" {
			resolved.DirectoryURL = zeroSSLDirectoryURL
		}
		if resolved.ZeroSSLEABKID == "" || resolved.ZeroSSLEABHMACKey == "" {
			return resolvedACMEIssueOptions{}, fmt.Errorf("zerossl requires eab kid and eab hmac key")
		}
	case "custom":
		if strings.TrimSpace(resolved.DirectoryURL) == "" {
			return resolvedACMEIssueOptions{}, fmt.Errorf("custom certificate authority requires a directory url")
		}
	default:
		return resolvedACMEIssueOptions{}, fmt.Errorf("unsupported certificate authority server %q", resolved.CertificateAuthorityServer)
	}

	if resolved.AccountEmail == "" {
		return resolvedACMEIssueOptions{}, fmt.Errorf("acme account email is required")
	}
	if resolved.DirectoryURL == "" {
		return resolvedACMEIssueOptions{}, fmt.Errorf("acme directory url is required")
	}
	if resolved.ChallengeType != "http-01" && resolved.ChallengeType != "dns-01" {
		return resolvedACMEIssueOptions{}, fmt.Errorf("challenge_type must be http-01 or dns-01")
	}
	if resolved.ChallengeType == "dns-01" && resolved.DNSProvider == "" {
		return resolvedACMEIssueOptions{}, fmt.Errorf("dns provider is required for dns-01 challenge")
	}
	if resolved.ChallengeType != "dns-01" {
		resolved.DNSProvider = ""
		resolved.DNSProviderEnv = map[string]string{}
		resolved.DNSResolvers = []string{}
		resolved.DNSPropagationSeconds = 0
	}

	return resolved, nil
}

func applyProcessEnv(values map[string]string) (func(), error) {
	if len(values) == 0 {
		return func() {}, nil
	}

	type previous struct {
		value string
		set   bool
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	prev := make(map[string]previous, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(values[key])
		if key == "" || value == "" {
			continue
		}
		oldValue, oldSet := os.LookupEnv(key)
		prev[key] = previous{value: oldValue, set: oldSet}
		if err := os.Setenv(key, value); err != nil {
			for restoreKey, restoreValue := range prev {
				if restoreValue.set {
					_ = os.Setenv(restoreKey, restoreValue.value)
				} else {
					_ = os.Unsetenv(restoreKey)
				}
			}
			return nil, fmt.Errorf("set env %s: %w", key, err)
		}
	}
	return func() {
		for _, key := range keys {
			previousValue, ok := prev[key]
			if !ok {
				continue
			}
			if previousValue.set {
				_ = os.Setenv(key, previousValue.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}, nil
}

func buildDNSChallengeProvider(options resolvedACMEIssueOptions) (challenge.Provider, error) {
	switch options.DNSProvider {
	case "cloudflare":
		token := strings.TrimSpace(options.DNSProviderEnv["CLOUDFLARE_API_TOKEN"])
		if token == "" {
			token = strings.TrimSpace(options.DNSProviderEnv["CF_API_TOKEN"])
		}
		return newCloudflareDNSProvider(token)
	default:
		return nil, fmt.Errorf("dns provider %q is not supported; supported providers: cloudflare", options.DNSProvider)
	}
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

func (c *ACMELetsEncryptClient) userStatePath(email string, directoryURL string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	directoryURL = strings.ToLower(strings.TrimSpace(directoryURL))
	if email == strings.ToLower(strings.TrimSpace(c.cfg.Email)) && directoryURL == strings.ToLower(strings.TrimSpace(c.cfg.DirectoryURL)) {
		return filepath.Join(c.cfg.StateDir, "account.json")
	}
	sum := sha256.Sum256([]byte(directoryURL + "|" + email))
	return filepath.Join(c.cfg.StateDir, "accounts", hex.EncodeToString(sum[:8])+".json")
}

func (c *ACMELetsEncryptClient) loadOrCreateUser(email string, directoryURL string) (*acmeUser, error) {
	path := c.userStatePath(email, directoryURL)
	content, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read acme account state: %w", err)
		}
		return c.createUser(email, directoryURL)
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
	user.Email = email
	return &user, nil
}

func (c *ACMELetsEncryptClient) createUser(email string, directoryURL string) (*acmeUser, error) {
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
		Email:  email,
		KeyPEM: string(keyPEM),
		key:    key,
	}
	if err := c.saveUser(user, email, directoryURL); err != nil {
		return nil, err
	}
	return user, nil
}

func (c *ACMELetsEncryptClient) saveUser(user *acmeUser, email string, directoryURL string) error {
	content, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("encode acme account state: %w", err)
	}
	content = append(content, '\n')
	path := c.userStatePath(email, directoryURL)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create acme account state dir: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
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
