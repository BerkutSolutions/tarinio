package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/coordination/redis"
)

const (
	defaultHTTPAddr         = "127.0.0.1:8080"
	defaultRuntimeRoot      = "/var/lib/waf"
	defaultRevisionsDir     = "control-plane"
	defaultRuntimeHealthURL = "http://127.0.0.1:8081/readyz"
	defaultRuntimeReloadURL = "http://127.0.0.1:8081/reload"
)

// Config contains the minimal control-plane process configuration for the MVP skeleton.
type Config struct {
	HTTPAddr         string
	RuntimeRoot      string
	RevisionStoreDir string
	PostgresDSN      string
	AuthIssuer       string
	StartupSelfTest  bool
	Security         SecurityConfig
	ACME             ACMEConfig
	RuntimeHealthURL string
	RuntimeReloadURL string
	RuntimeAPIToken  string
	BootstrapAdmin   BootstrapAdminConfig
	DevFastStart     DevFastStartConfig
	Redis            redis.Config
	HA               HAConfig
	Metrics          MetricsConfig
	SentinelBanSync  SentinelBanSyncConfig
}

type ACMEConfig struct {
	Enabled              bool
	UseDevelopmentClient bool
	Email                string
	DirectoryURL         string
	StateDir             string
	ChallengeDir         string
}

type SecurityConfig struct {
	Pepper   string
	WebAuthn WebAuthnConfig
}

type WebAuthnConfig struct {
	Enabled bool
	RPID    string
	RPName  string
	Origins []string
}

type BootstrapAdminConfig struct {
	Enabled  bool
	ID       string
	Username string
	Email    string
	Password string
}

type DevFastStartConfig struct {
	Enabled           bool
	Host              string
	CertificateID     string
	ManagementSiteID  string
	UpstreamHost      string
	UpstreamPort      int
	RetryDelaySeconds int
	MaxAttempts       int
}

type HAConfig struct {
	Enabled                 bool
	NodeID                  string
	OperationLockTTLSeconds int
	LeaderLockTTLSeconds    int
}

type MetricsConfig struct {
	Token string
}

type SentinelBanSyncConfig struct {
	Enabled             bool
	AdaptivePath        string
	StatePath           string
	PollIntervalSeconds int
	MinScore            float64
	MaxPromotionsPerTick int
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:         defaultHTTPAddr,
		RuntimeRoot:      defaultRuntimeRoot,
		RevisionStoreDir: filepath.Join(defaultRuntimeRoot, defaultRevisionsDir),
		PostgresDSN:      "",
		AuthIssuer:       "WAF",
		StartupSelfTest:  true,
		Security: SecurityConfig{
			Pepper: "",
			WebAuthn: WebAuthnConfig{
				Enabled: true,
				RPName:  "TARINIO",
			},
		},
		ACME: ACMEConfig{
			Enabled:              true,
			UseDevelopmentClient: false,
			Email:                "admin@example.com",
			DirectoryURL:         "https://acme-v02.api.letsencrypt.org/directory",
			StateDir:             filepath.Join(defaultRuntimeRoot, defaultRevisionsDir, "acme-state"),
			ChallengeDir:         filepath.Join(defaultRuntimeRoot, defaultRevisionsDir, "acme-challenges"),
		},
		RuntimeHealthURL: defaultRuntimeHealthURL,
		RuntimeReloadURL: defaultRuntimeReloadURL,
		RuntimeAPIToken:  "",
		BootstrapAdmin: BootstrapAdminConfig{
			Enabled:  false,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@localhost",
			Password: "admin",
		},
		DevFastStart: DevFastStartConfig{
			Enabled:           false,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 2,
			MaxAttempts:       30,
		},
		Redis: redis.Config{
			DB:          redis.DefaultConfig().DB,
			DialTimeout: redis.DefaultConfig().DialTimeout,
		},
		HA: HAConfig{
			Enabled:                 false,
			NodeID:                  hostnameOrDefault("waf-control-plane"),
			OperationLockTTLSeconds: 120,
			LeaderLockTTLSeconds:    30,
		},
		Metrics: MetricsConfig{},
		SentinelBanSync: SentinelBanSyncConfig{
			Enabled:              false,
			AdaptivePath:         "/etc/waf/l4guard-adaptive/adaptive.json",
			StatePath:            filepath.Join(defaultRuntimeRoot, defaultRevisionsDir, "sentinel-ban-sync", "state.json"),
			PollIntervalSeconds:  5,
			MinScore:             10,
			MaxPromotionsPerTick: 5,
		},
	}

	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_HTTP_ADDR")); value != "" {
		cfg.HTTPAddr = value
	}
	if value := strings.TrimSpace(os.Getenv("WAF_RUNTIME_ROOT")); value != "" {
		cfg.RuntimeRoot = value
		cfg.RevisionStoreDir = filepath.Join(value, defaultRevisionsDir)
		cfg.ACME.StateDir = filepath.Join(value, defaultRevisionsDir, "acme-state")
		cfg.ACME.ChallengeDir = filepath.Join(value, defaultRevisionsDir, "acme-challenges")
		cfg.SentinelBanSync.StatePath = filepath.Join(value, defaultRevisionsDir, "sentinel-ban-sync", "state.json")
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REVISION_STORE_DIR")); value != "" {
		cfg.RevisionStoreDir = value
		cfg.SentinelBanSync.StatePath = filepath.Join(value, "sentinel-ban-sync", "state.json")
	}
	if value := strings.TrimSpace(os.Getenv("POSTGRES_DSN")); value != "" {
		cfg.PostgresDSN = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_AUTH_ISSUER")); value != "" {
		cfg.AuthIssuer = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_STARTUP_SELF_TEST_ENABLED")); value != "" {
		cfg.StartupSelfTest = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SECURITY_PEPPER")); value != "" {
		cfg.Security.Pepper = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_ENABLED")); value != "" {
		cfg.ACME.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_USE_DEVELOPMENT_CLIENT")); value != "" {
		cfg.ACME.UseDevelopmentClient = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_EMAIL")); value != "" {
		cfg.ACME.Email = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_DIRECTORY_URL")); value != "" {
		cfg.ACME.DirectoryURL = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_STATE_DIR")); value != "" {
		cfg.ACME.StateDir = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_ACME_CHALLENGE_DIR")); value != "" {
		cfg.ACME.ChallengeDir = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_WEBAUTHN_ENABLED")); value != "" {
		cfg.Security.WebAuthn.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_WEBAUTHN_RPID")); value != "" {
		cfg.Security.WebAuthn.RPID = strings.ToLower(value)
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_WEBAUTHN_RP_NAME")); value != "" {
		cfg.Security.WebAuthn.RPName = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_WEBAUTHN_ORIGINS")); value != "" {
		parts := strings.Split(value, ",")
		origins := make([]string, 0, len(parts))
		for _, item := range parts {
			origin := strings.TrimSpace(item)
			if origin == "" {
				continue
			}
			origins = append(origins, origin)
		}
		cfg.Security.WebAuthn.Origins = origins
	}
	if value := strings.TrimSpace(os.Getenv("WAF_RUNTIME_HEALTH_URL")); value != "" {
		cfg.RuntimeHealthURL = value
	}
	if value := strings.TrimSpace(os.Getenv("WAF_RUNTIME_RELOAD_URL")); value != "" {
		cfg.RuntimeReloadURL = value
	}
	if value := strings.TrimSpace(os.Getenv("WAF_RUNTIME_API_TOKEN")); value != "" {
		cfg.RuntimeAPIToken = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_BOOTSTRAP_ADMIN_ID")); value != "" {
		cfg.BootstrapAdmin.ID = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_BOOTSTRAP_ADMIN_ENABLED")); value != "" {
		cfg.BootstrapAdmin.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME")); value != "" {
		cfg.BootstrapAdmin.Username = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_BOOTSTRAP_ADMIN_EMAIL")); value != "" {
		cfg.BootstrapAdmin.Email = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD")); value != "" {
		cfg.BootstrapAdmin.Password = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_ENABLED")); value != "" {
		cfg.DevFastStart.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_HOST")); value != "" {
		cfg.DevFastStart.Host = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_CERTIFICATE_ID")); value != "" {
		cfg.DevFastStart.CertificateID = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID")); value != "" {
		cfg.DevFastStart.ManagementSiteID = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_UPSTREAM_HOST")); value != "" {
		cfg.DevFastStart.UpstreamHost = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_UPSTREAM_PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 || port > 65535 {
			return Config{}, fmt.Errorf("dev fast start upstream port must be between 1 and 65535")
		}
		cfg.DevFastStart.UpstreamPort = port
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_RETRY_DELAY_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("dev fast start retry delay must be a positive integer")
		}
		cfg.DevFastStart.RetryDelaySeconds = seconds
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_MAX_ATTEMPTS")); value != "" {
		attempts, err := strconv.Atoi(value)
		if err != nil || attempts <= 0 {
			return Config{}, fmt.Errorf("dev fast start max attempts must be a positive integer")
		}
		cfg.DevFastStart.MaxAttempts = attempts
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REDIS_ADDR")); value != "" {
		cfg.Redis.Addr = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REDIS_USERNAME")); value != "" {
		cfg.Redis.Username = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REDIS_PASSWORD")); value != "" {
		cfg.Redis.Password = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REDIS_DB")); value != "" {
		db, err := redis.ParseDB(value)
		if err != nil {
			return Config{}, err
		}
		cfg.Redis.DB = db
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_REDIS_DIAL_TIMEOUT_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("redis dial timeout must be a positive integer")
		}
		cfg.Redis.DialTimeout = time.Duration(seconds) * time.Second
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_HA_ENABLED")); value != "" {
		cfg.HA.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_HA_NODE_ID")); value != "" {
		cfg.HA.NodeID = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_HA_OPERATION_LOCK_TTL_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("ha operation lock ttl must be a positive integer")
		}
		cfg.HA.OperationLockTTLSeconds = seconds
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_HA_LEADER_LOCK_TTL_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("ha leader lock ttl must be a positive integer")
		}
		cfg.HA.LeaderLockTTLSeconds = seconds
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_METRICS_TOKEN")); value != "" {
		cfg.Metrics.Token = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_ENABLED")); value != "" {
		cfg.SentinelBanSync.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_ADAPTIVE_PATH")); value != "" {
		cfg.SentinelBanSync.AdaptivePath = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_STATE_PATH")); value != "" {
		cfg.SentinelBanSync.StatePath = value
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_POLL_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("sentinel ban sync poll seconds must be a positive integer")
		}
		cfg.SentinelBanSync.PollIntervalSeconds = seconds
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_MIN_SCORE")); value != "" {
		minScore, err := strconv.ParseFloat(value, 64)
		if err != nil || minScore <= 0 {
			return Config{}, fmt.Errorf("sentinel ban sync min score must be a positive number")
		}
		cfg.SentinelBanSync.MinScore = minScore
	}
	if value := strings.TrimSpace(os.Getenv("CONTROL_PLANE_SENTINEL_BAN_SYNC_MAX_PROMOTIONS_PER_TICK")); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 {
			return Config{}, fmt.Errorf("sentinel ban sync max promotions per tick must be a positive integer")
		}
		cfg.SentinelBanSync.MaxPromotionsPerTick = limit
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return fmt.Errorf("http addr is required")
	}
	host, port, err := splitHostPort(cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("invalid http addr: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("http addr host is required")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("http addr port must be numeric")
	}
	if strings.TrimSpace(cfg.RevisionStoreDir) == "" {
		return fmt.Errorf("revision store dir is required")
	}
	if strings.TrimSpace(cfg.AuthIssuer) == "" {
		return fmt.Errorf("auth issuer is required")
	}
	if strings.TrimSpace(cfg.Security.Pepper) == "" {
		return fmt.Errorf("security pepper is required")
	}
	if cfg.ACME.Enabled && !cfg.ACME.UseDevelopmentClient {
		if strings.TrimSpace(cfg.ACME.Email) == "" {
			return fmt.Errorf("acme email is required")
		}
		if strings.TrimSpace(cfg.ACME.DirectoryURL) == "" {
			return fmt.Errorf("acme directory url is required")
		}
		if strings.TrimSpace(cfg.ACME.StateDir) == "" {
			return fmt.Errorf("acme state dir is required")
		}
		if strings.TrimSpace(cfg.ACME.ChallengeDir) == "" {
			return fmt.Errorf("acme challenge dir is required")
		}
	}
	if cfg.Security.WebAuthn.Enabled && strings.TrimSpace(cfg.Security.WebAuthn.RPName) == "" {
		return fmt.Errorf("webauthn rp name is required when enabled")
	}
	if strings.TrimSpace(cfg.RuntimeHealthURL) == "" {
		return fmt.Errorf("runtime health url is required")
	}
	if strings.TrimSpace(cfg.RuntimeReloadURL) == "" {
		return fmt.Errorf("runtime reload url is required")
	}
	if cfg.BootstrapAdmin.Enabled {
		if strings.TrimSpace(cfg.BootstrapAdmin.ID) == "" || strings.TrimSpace(cfg.BootstrapAdmin.Username) == "" || strings.TrimSpace(cfg.BootstrapAdmin.Email) == "" || strings.TrimSpace(cfg.BootstrapAdmin.Password) == "" {
			return fmt.Errorf("bootstrap admin config is required when enabled")
		}
	}
	if cfg.DevFastStart.Enabled {
		if !cfg.BootstrapAdmin.Enabled {
			return fmt.Errorf("dev fast start requires bootstrap admin to be enabled")
		}
		if strings.TrimSpace(cfg.DevFastStart.Host) == "" {
			return fmt.Errorf("dev fast start host is required when enabled")
		}
		if strings.TrimSpace(cfg.DevFastStart.CertificateID) == "" {
			return fmt.Errorf("dev fast start certificate id is required when enabled")
		}
		if strings.TrimSpace(cfg.DevFastStart.ManagementSiteID) == "" {
			return fmt.Errorf("dev fast start management site id is required when enabled")
		}
		if strings.TrimSpace(cfg.DevFastStart.UpstreamHost) == "" {
			return fmt.Errorf("dev fast start upstream host is required when enabled")
		}
		if cfg.DevFastStart.UpstreamPort <= 0 || cfg.DevFastStart.UpstreamPort > 65535 {
			return fmt.Errorf("dev fast start upstream port must be between 1 and 65535")
		}
	}
	if cfg.HA.Enabled {
		if strings.TrimSpace(cfg.Redis.Addr) == "" {
			return fmt.Errorf("redis addr is required when ha is enabled")
		}
		if strings.TrimSpace(cfg.HA.NodeID) == "" {
			return fmt.Errorf("ha node id is required when ha is enabled")
		}
		if cfg.HA.OperationLockTTLSeconds <= 0 {
			return fmt.Errorf("ha operation lock ttl must be positive")
		}
		if cfg.HA.LeaderLockTTLSeconds <= 0 {
			return fmt.Errorf("ha leader lock ttl must be positive")
		}
	}
	if cfg.SentinelBanSync.Enabled {
		if strings.TrimSpace(cfg.SentinelBanSync.AdaptivePath) == "" {
			return fmt.Errorf("sentinel ban sync adaptive path is required when enabled")
		}
		if cfg.SentinelBanSync.PollIntervalSeconds <= 0 {
			return fmt.Errorf("sentinel ban sync poll interval must be positive")
		}
		if cfg.SentinelBanSync.MinScore <= 0 {
			return fmt.Errorf("sentinel ban sync min score must be positive")
		}
		if cfg.SentinelBanSync.MaxPromotionsPerTick <= 0 {
			return fmt.Errorf("sentinel ban sync max promotions per tick must be positive")
		}
	}
	return nil
}

func splitHostPort(addr string) (string, string, error) {
	index := strings.LastIndex(addr, ":")
	if index == -1 || index == len(addr)-1 {
		return "", "", fmt.Errorf("expected host:port")
	}
	return addr[:index], addr[index+1:], nil
}

func hostnameOrDefault(fallback string) string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return fallback
	}
	return hostname
}
