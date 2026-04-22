package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/appcompat"
	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/config"
	"waf/control-plane/internal/coordination/redis"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/enterprise"
	"waf/control-plane/internal/events"
	"waf/control-plane/internal/handlers"
	"waf/control-plane/internal/httpserver"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/storage"
	storemigrations "waf/control-plane/internal/store"
	"waf/control-plane/internal/telemetry"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/users"
	"waf/control-plane/internal/wafpolicies"
)

// App wires the minimal control-plane foundation without runtime coupling.
type App struct {
	Config                   config.Config
	PostgresBackend          *storage.PostgresBackend
	RedisBackend             *redis.Backend
	Coordinator              services.DistributedCoordinator
	RevisionStore            *revisions.Store
	RevisionSnapshotStore    *revisionsnapshots.Store
	SetupService             *services.SetupService
	RevisionService          *services.RevisionService
	RevisionCompileService   *services.RevisionCompileService
	ApplyService             *services.ApplyService
	RevisionCatalogService   *services.RevisionCatalogService
	EventStore               *events.Store
	EventService             *services.EventService
	AdminScriptService       *services.AdminScriptService
	AuditStore               *audits.Store
	AuditService             *services.AuditService
	EnterpriseStore          *enterprise.Store
	EnterpriseService        *services.EnterpriseService
	ReportService            *services.ReportService
	DashboardService         *services.DashboardService
	ContainerRuntimeService  *services.ContainerRuntimeService
	JobStore                 *jobs.Store
	JobService               *services.JobService
	RoleStore                *roles.Store
	SessionStore             *sessions.Store
	SiteStore                *sites.Store
	SiteService              *services.SiteService
	ManualBanService         *services.ManualBanService
	UpstreamStore            *upstreams.Store
	UpstreamService          *services.UpstreamService
	CertificateStore         *certificates.Store
	CertificateService       *services.CertificateService
	CertificateMaterialStore *certificatematerials.Store
	CertificateUploadService *services.CertificateUploadService
	LetsEncryptService       *services.LetsEncryptService
	TLSConfigStore           *tlsconfigs.Store
	TLSConfigService         *services.TLSConfigService
	TLSAutoRenewService      *services.TLSAutoRenewService
	WAFPolicyStore           *wafpolicies.Store
	WAFPolicyService         *services.WAFPolicyService
	AccessPolicyStore        *accesspolicies.Store
	AccessPolicyService      *services.AccessPolicyService
	RateLimitPolicyStore     *ratelimitpolicies.Store
	RateLimitPolicyService   *services.RateLimitPolicyService
	EasySiteProfileStore     *easysiteprofiles.Store
	EasySiteProfileService   *services.EasySiteProfileService
	AntiDDoSStore            *antiddos.Store
	AntiDDoSService          *services.AntiDDoSService
	RuntimeCRSService        *services.RuntimeCRSService
	UserStore                *users.Store
	AuthService              *services.AuthService
	PasskeyStore             *passkeys.Store
	DevFastStartBootstrapper *services.DevFastStartBootstrapper
	HTTPServer               *httpserver.Server
}

func New(cfg config.Config) (*App, error) {
	handlers.SetSessionBootToken(cfg.Security.Pepper)
	telemetry.Default().RecordBuild(cfg.HA.NodeID, cfg.HA.Enabled)

	if err := appcompat.EnsureLegacyDataTransferred(cfg.RuntimeRoot, cfg.RevisionStoreDir); err != nil {
		return nil, err
	}

	var postgresBackend *storage.PostgresBackend
	if strings.TrimSpace(cfg.PostgresDSN) != "" {
		backend, err := storage.NewPostgresBackend(cfg.PostgresDSN)
		if err != nil {
			return nil, err
		}
		if err := backend.Ping(); err != nil {
			return nil, err
		}
		if err := storemigrations.RunMigrations(backend.DB()); err != nil {
			return nil, err
		}
		if err := migrateLegacyStateToPostgres(backend, cfg.RevisionStoreDir); err != nil {
			return nil, err
		}
		postgresBackend = backend
	}

	var redisBackend *redis.Backend
	coord := services.NewNoopDistributedCoordinator()
	if strings.TrimSpace(cfg.Redis.Addr) != "" {
		redisClient := redis.NewClient(cfg.Redis)
		redisBackend = redis.NewBackend(redisClient)
		if cfg.HA.Enabled {
			if err := redisClient.Ping(context.Background()); err != nil {
				return nil, err
			}
			coord = services.NewRedisDistributedCoordinator(
				redisBackend,
				cfg.HA.NodeID,
				time.Duration(cfg.HA.OperationLockTTLSeconds)*time.Second,
				time.Duration(cfg.HA.LeaderLockTTLSeconds)*time.Second,
			)
		}
	} else if cfg.HA.Enabled {
		return nil, fmt.Errorf("redis addr is required when ha is enabled")
	}

	revisionsRoot := filepath.Join(cfg.RevisionStoreDir, "revisions")
	revisionSnapshotsRoot := filepath.Join(cfg.RevisionStoreDir, "revision-snapshots")
	eventsRoot := filepath.Join(cfg.RevisionStoreDir, "events")
	auditsRoot := filepath.Join(cfg.RevisionStoreDir, "audits")
	jobsRoot := filepath.Join(cfg.RevisionStoreDir, "jobs")
	rolesRoot := filepath.Join(cfg.RevisionStoreDir, "roles")
	usersRoot := filepath.Join(cfg.RevisionStoreDir, "users")
	sessionsRoot := filepath.Join(cfg.RevisionStoreDir, "sessions")
	passkeysRoot := filepath.Join(cfg.RevisionStoreDir, "passkeys")
	sitesRoot := filepath.Join(cfg.RevisionStoreDir, "sites")
	upstreamsRoot := filepath.Join(cfg.RevisionStoreDir, "upstreams")
	certificatesRoot := filepath.Join(cfg.RevisionStoreDir, "certificates")
	certificateMaterialsRoot := filepath.Join(cfg.RevisionStoreDir, "certificate-materials")
	tlsConfigsRoot := filepath.Join(cfg.RevisionStoreDir, "tlsconfigs")
	wafPoliciesRoot := filepath.Join(cfg.RevisionStoreDir, "wafpolicies")
	accessPoliciesRoot := filepath.Join(cfg.RevisionStoreDir, "accesspolicies")
	rateLimitPoliciesRoot := filepath.Join(cfg.RevisionStoreDir, "ratelimitpolicies")
	easySiteProfilesRoot := filepath.Join(cfg.RevisionStoreDir, "easysiteprofiles")
	antiDDoSRoot := filepath.Join(cfg.RevisionStoreDir, "antiddos")
	enterpriseRoot := filepath.Join(cfg.RevisionStoreDir, "enterprise")

	var (
		err                      error
		revisionStore            *revisions.Store
		revisionSnapshotStore    *revisionsnapshots.Store
		eventStore               *events.Store
		auditStore               *audits.Store
		jobStore                 *jobs.Store
		roleStore                *roles.Store
		userStore                *users.Store
		sessionStore             *sessions.Store
		passkeyStore             *passkeys.Store
		siteStore                *sites.Store
		upstreamStore            *upstreams.Store
		certificateStore         *certificates.Store
		certificateMaterialStore *certificatematerials.Store
		tlsConfigStore           *tlsconfigs.Store
		wafPolicyStore           *wafpolicies.Store
		accessPolicyStore        *accesspolicies.Store
		rateLimitPolicyStore     *ratelimitpolicies.Store
		easySiteProfileStore     *easysiteprofiles.Store
		antiDDoSStore            *antiddos.Store
		enterpriseStore          *enterprise.Store
	)

	bootstrapUser := users.BootstrapUser{
		Enabled:  cfg.BootstrapAdmin.Enabled,
		ID:       cfg.BootstrapAdmin.ID,
		Username: cfg.BootstrapAdmin.Username,
		Email:    cfg.BootstrapAdmin.Email,
		Password: cfg.BootstrapAdmin.Password,
		RoleIDs:  []string{"admin"},
	}

	if postgresBackend != nil {
		revisionStore, err = revisions.NewPostgresStore(revisionsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		revisionSnapshotStore, err = revisionsnapshots.NewPostgresStore(revisionSnapshotsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		eventStore, err = events.NewPostgresStore(eventsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		auditStore, err = audits.NewPostgresStore(auditsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		jobStore, err = jobs.NewPostgresStore(jobsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		roleStore, err = roles.NewPostgresStore(rolesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		userStore, err = users.NewPostgresStore(usersRoot, postgresBackend, bootstrapUser)
		if err != nil {
			return nil, err
		}
		sessionStore, err = sessions.NewPostgresStore(sessionsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		passkeyStore, err = passkeys.NewPostgresStore(passkeysRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		siteStore, err = sites.NewPostgresStore(sitesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		upstreamStore, err = upstreams.NewPostgresStore(upstreamsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		certificateStore, err = certificates.NewPostgresStore(certificatesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		certificateMaterialStore, err = certificatematerials.NewPostgresStore(certificateMaterialsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		tlsConfigStore, err = tlsconfigs.NewPostgresStore(tlsConfigsRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		wafPolicyStore, err = wafpolicies.NewPostgresStore(wafPoliciesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		accessPolicyStore, err = accesspolicies.NewPostgresStore(accessPoliciesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		rateLimitPolicyStore, err = ratelimitpolicies.NewPostgresStore(rateLimitPoliciesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		easySiteProfileStore, err = easysiteprofiles.NewPostgresStore(easySiteProfilesRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		antiDDoSStore, err = antiddos.NewPostgresStore(antiDDoSRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
		enterpriseStore, err = enterprise.NewPostgresStore(enterpriseRoot, postgresBackend)
		if err != nil {
			return nil, err
		}
	} else {
		revisionStore, err = revisions.NewStore(revisionsRoot)
		if err != nil {
			return nil, err
		}
		revisionSnapshotStore, err = revisionsnapshots.NewStore(revisionSnapshotsRoot)
		if err != nil {
			return nil, err
		}
		eventStore, err = events.NewStore(eventsRoot)
		if err != nil {
			return nil, err
		}
		auditStore, err = audits.NewStore(auditsRoot)
		if err != nil {
			return nil, err
		}
		jobStore, err = jobs.NewStore(jobsRoot)
		if err != nil {
			return nil, err
		}
		roleStore, err = roles.NewStore(rolesRoot)
		if err != nil {
			return nil, err
		}
		userStore, err = users.NewStore(usersRoot, bootstrapUser)
		if err != nil {
			return nil, err
		}
		sessionStore, err = sessions.NewStore(sessionsRoot)
		if err != nil {
			return nil, err
		}
		passkeyStore, err = passkeys.NewStore(passkeysRoot)
		if err != nil {
			return nil, err
		}
		siteStore, err = sites.NewStore(sitesRoot)
		if err != nil {
			return nil, err
		}
		upstreamStore, err = upstreams.NewStore(upstreamsRoot)
		if err != nil {
			return nil, err
		}
		certificateStore, err = certificates.NewStore(certificatesRoot)
		if err != nil {
			return nil, err
		}
		certificateMaterialStore, err = certificatematerials.NewStore(certificateMaterialsRoot)
		if err != nil {
			return nil, err
		}
		tlsConfigStore, err = tlsconfigs.NewStore(tlsConfigsRoot)
		if err != nil {
			return nil, err
		}
		wafPolicyStore, err = wafpolicies.NewStore(wafPoliciesRoot)
		if err != nil {
			return nil, err
		}
		accessPolicyStore, err = accesspolicies.NewStore(accessPoliciesRoot)
		if err != nil {
			return nil, err
		}
		rateLimitPolicyStore, err = ratelimitpolicies.NewStore(rateLimitPoliciesRoot)
		if err != nil {
			return nil, err
		}
		easySiteProfileStore, err = easysiteprofiles.NewStore(easySiteProfilesRoot)
		if err != nil {
			return nil, err
		}
		antiDDoSStore, err = antiddos.NewStore(antiDDoSRoot)
		if err != nil {
			return nil, err
		}
		enterpriseStore, err = enterprise.NewStore(enterpriseRoot)
		if err != nil {
			return nil, err
		}
	}

	revisionService := services.NewRevisionService(revisionStore)
	setupService := services.NewSetupService(userStore, siteStore, revisionStore)
	auditService := services.NewAuditService(auditStore)
	revisionCompileService := services.NewRevisionCompileService(revisionStore, revisionSnapshotStore, jobStore, siteStore, upstreamStore, certificateStore, tlsConfigStore, wafPolicyStore, accessPolicyStore, rateLimitPolicyStore, easySiteProfileStore, antiDDoSStore, certificateMaterialStore, auditService)
	revisionCompileService.SetCoordinator(coord)
	revisionCatalogService := services.NewRevisionCatalogService(revisionStore, revisionSnapshotStore, jobStore, eventStore, siteStore)
	runtimeSecurityCollector := services.NewHTTPRuntimeSecurityEventCollector(cfg.RuntimeHealthURL, cfg.RuntimeAPIToken)
	eventService := services.NewEventService(
		eventStore,
		services.WithRuntimeSecurityCollector(runtimeSecurityCollector),
	)
	jobService := services.NewJobService(jobStore)
	enterpriseService := services.NewEnterpriseService(enterpriseStore, userStore, roleStore, sessionStore, revisionStore, auditStore, eventStore, jobStore, auditService)
	authService := services.NewAuthService(userStore, roleStore, sessionStore, passkeyStore, cfg.AuthIssuer, services.AuthSecurityConfig{
		Pepper: cfg.Security.Pepper,
		WebAuthn: services.WebAuthnConfig{
			Enabled: cfg.Security.WebAuthn.Enabled,
			RPID:    cfg.Security.WebAuthn.RPID,
			RPName:  cfg.Security.WebAuthn.RPName,
			Origins: append([]string(nil), cfg.Security.WebAuthn.Origins...),
		},
	}, auditService)
	revisionCompileService.SetGovernance(enterpriseService)
	siteService := services.NewSiteService(siteStore, auditService)
	manualBanService := services.NewManualBanService(accessPolicyStore, siteStore, auditService)
	upstreamService := services.NewUpstreamService(upstreamStore, siteStore, auditService)
	certificateService := services.NewCertificateService(certificateStore, auditService)
	certificateUploadService := services.NewCertificateUploadService(certificateStore, certificateMaterialStore, auditService)
	var letsEncryptClient services.LetsEncryptClient
	if cfg.ACME.Enabled && !cfg.ACME.UseDevelopmentClient {
		client, err := services.NewACMELetsEncryptClient(services.ACMEClientConfig{
			Email:        cfg.ACME.Email,
			DirectoryURL: cfg.ACME.DirectoryURL,
			StateDir:     cfg.ACME.StateDir,
			ChallengeDir: cfg.ACME.ChallengeDir,
		})
		if err != nil {
			return nil, err
		}
		letsEncryptClient = client
	} else {
		letsEncryptClient = services.NewDevelopmentLetsEncryptClient()
	}
	letsEncryptService := services.NewLetsEncryptService(letsEncryptClient, jobStore, certificateStore, certificateMaterialStore, auditService)
	selfSignedCertificateService := services.NewLetsEncryptService(services.NewDevelopmentLetsEncryptClient(), jobStore, certificateStore, certificateMaterialStore, auditService)
	tlsConfigService := services.NewTLSConfigService(tlsConfigStore, siteStore, certificateStore, auditService)
	tlsAutoRenewService, err := services.NewTLSAutoRenewServiceWithBackend(filepath.Join(cfg.RevisionStoreDir, "tls-auto-renew"), postgresBackend, certificateStore, tlsConfigStore, letsEncryptService)
	if err != nil {
		return nil, err
	}
	tlsAutoRenewService.SetCoordinator(coord)
	tlsAutoRenewService.Start()
	wafPolicyService := services.NewWAFPolicyService(wafPolicyStore, siteStore, auditService)
	accessPolicyService := services.NewAccessPolicyService(accessPolicyStore, siteStore, auditService)
	rateLimitPolicyService := services.NewRateLimitPolicyService(rateLimitPolicyStore, siteStore, auditService)
	applyService := services.NewApplyService(
		cfg.RuntimeRoot,
		revisionStore,
		revisionSnapshotStore,
		jobStore,
		eventService,
		services.NoopCommandExecutor{},
		services.HTTPReloadExecutor{URL: cfg.RuntimeReloadURL, Token: cfg.RuntimeAPIToken},
		services.HTTPHealthChecker{URL: cfg.RuntimeHealthURL, Token: cfg.RuntimeAPIToken},
		auditService,
	)
	applyService.SetCoordinator(coord)
	applyService.SetGovernance(enterpriseService)
	easySiteProfileService := services.NewEasySiteProfileService(easySiteProfileStore, siteStore, wafPolicyStore, accessPolicyStore, rateLimitPolicyStore, revisionCompileService, applyService, auditService)
	antiDDoSService := services.NewAntiDDoSService(antiDDoSStore, revisionCompileService, applyService, auditService)
	services.ConfigureAutoApply(revisionCompileService, applyService, coord)
	reportService := services.NewReportService(eventStore, jobStore, revisionStore)
	runtimeRequestCollector := services.NewHTTPRuntimeRequestCollector(cfg.RuntimeHealthURL, cfg.RuntimeAPIToken)
	runtimeReadyProbe := services.NewHTTPRuntimeReadyProbe(cfg.RuntimeHealthURL, cfg.RuntimeAPIToken)
	dashboardService := services.NewDashboardService(eventService, runtimeRequestCollector, runtimeReadyProbe)
	runtimeCRSService := services.NewRuntimeCRSService(services.RuntimeBaseURLFromHealthURL(cfg.RuntimeHealthURL), cfg.RuntimeAPIToken)
	containerRuntimeService := services.NewContainerRuntimeService()
	adminScriptService := services.NewAdminScriptService(cfg.RevisionStoreDir, detectScriptsRoot())
	httpServer := httpserver.New(cfg.HTTPAddr, cfg.RuntimeRoot, cfg.RevisionStoreDir, cfg.RuntimeHealthURL, coord.Enabled(), coord.NodeID(), cfg.Metrics.Token, postgresBackend, setupService, revisionService, authService, enterpriseService, sessionStore, userStore, roleStore, siteService, manualBanService, upstreamService, certificateService, tlsConfigService, tlsAutoRenewService, certificateUploadService, certificateMaterialStore, letsEncryptService, selfSignedCertificateService, wafPolicyService, accessPolicyService, rateLimitPolicyService, easySiteProfileService, antiDDoSService, eventService, revisionCompileService, applyService, revisionCatalogService, auditService, reportService, dashboardService, containerRuntimeService, runtimeCRSService, runtimeRequestCollector, runtimeReadyProbe, runtimeSecurityCollector, runtimeRequestCollector, adminScriptService)
	var devFastStartBootstrapper *services.DevFastStartBootstrapper
	if cfg.DevFastStart.Enabled {
		devFastStartCertificateIssuer := letsEncryptService
		if shouldUseSelfSignedForDevFastStartHost(cfg.DevFastStart.Host) {
			devFastStartCertificateIssuer = selfSignedCertificateService
		}
		devFastStartBootstrapper = services.NewDevFastStartBootstrapper(
			cfg.DevFastStart,
			siteService,
			upstreamService,
			certificateService,
			certificateMaterialStore,
			tlsConfigService,
			revisionStore,
			revisionCompileService,
			applyService,
			devFastStartCertificateIssuer,
			easySiteProfileStore,
			easySiteProfileService,
			rateLimitPolicyStore,
		)
		devFastStartBootstrapper.SetCoordinator(coord)
	}

	return &App{
		Config:                   cfg,
		PostgresBackend:          postgresBackend,
		RedisBackend:             redisBackend,
		Coordinator:              coord,
		RevisionStore:            revisionStore,
		RevisionSnapshotStore:    revisionSnapshotStore,
		SetupService:             setupService,
		RevisionService:          revisionService,
		RevisionCompileService:   revisionCompileService,
		ApplyService:             applyService,
		RevisionCatalogService:   revisionCatalogService,
		EventStore:               eventStore,
		EventService:             eventService,
		AdminScriptService:       adminScriptService,
		AuditStore:               auditStore,
		AuditService:             auditService,
		EnterpriseStore:          enterpriseStore,
		EnterpriseService:        enterpriseService,
		ReportService:            reportService,
		DashboardService:         dashboardService,
		ContainerRuntimeService:  containerRuntimeService,
		JobStore:                 jobStore,
		JobService:               jobService,
		RoleStore:                roleStore,
		SessionStore:             sessionStore,
		SiteStore:                siteStore,
		SiteService:              siteService,
		ManualBanService:         manualBanService,
		UpstreamStore:            upstreamStore,
		UpstreamService:          upstreamService,
		CertificateStore:         certificateStore,
		CertificateService:       certificateService,
		CertificateMaterialStore: certificateMaterialStore,
		CertificateUploadService: certificateUploadService,
		LetsEncryptService:       letsEncryptService,
		TLSConfigStore:           tlsConfigStore,
		TLSConfigService:         tlsConfigService,
		TLSAutoRenewService:      tlsAutoRenewService,
		WAFPolicyStore:           wafPolicyStore,
		WAFPolicyService:         wafPolicyService,
		AccessPolicyStore:        accessPolicyStore,
		AccessPolicyService:      accessPolicyService,
		RateLimitPolicyStore:     rateLimitPolicyStore,
		RateLimitPolicyService:   rateLimitPolicyService,
		EasySiteProfileStore:     easySiteProfileStore,
		EasySiteProfileService:   easySiteProfileService,
		AntiDDoSStore:            antiDDoSStore,
		AntiDDoSService:          antiDDoSService,
		RuntimeCRSService:        runtimeCRSService,
		UserStore:                userStore,
		AuthService:              authService,
		PasskeyStore:             passkeyStore,
		DevFastStartBootstrapper: devFastStartBootstrapper,
		HTTPServer:               httpServer,
	}, nil
}

func detectScriptsRoot() string {
	if value := strings.TrimSpace(os.Getenv("WAF_SCRIPTS_ROOT")); value != "" {
		return value
	}
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		candidates = append(candidates, filepath.Join(cwd, "scripts"))
	}
	candidates = append(candidates, "/src/scripts")
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func shouldUseSelfSignedForDevFastStartHost(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.Trim(value, "[]")
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.IsLoopback()
	}
	if strings.Contains(value, ":") {
		parsedHost, _, err := net.SplitHostPort(value)
		if err == nil {
			parsedHost = strings.Trim(strings.ToLower(strings.TrimSpace(parsedHost)), "[]")
			if ip := net.ParseIP(parsedHost); ip != nil {
				return ip.IsLoopback()
			}
			value = parsedHost
		}
	}
	return value == "localhost" || strings.HasSuffix(value, ".localhost")
}
