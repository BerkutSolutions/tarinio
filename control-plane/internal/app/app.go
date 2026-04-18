package app

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/appcompat"
	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/config"
	"waf/control-plane/internal/coordination/redis"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/events"
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
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/users"
	"waf/control-plane/internal/wafpolicies"
)

// App wires the minimal control-plane foundation without runtime coupling.
type App struct {
	Config                   config.Config
	RedisBackend             *redis.Backend
	RevisionStore            *revisions.Store
	RevisionSnapshotStore    *revisionsnapshots.Store
	SetupService             *services.SetupService
	RevisionService          *services.RevisionService
	RevisionCompileService   *services.RevisionCompileService
	ApplyService             *services.ApplyService
	EventStore               *events.Store
	EventService             *services.EventService
	AdminScriptService       *services.AdminScriptService
	AuditStore               *audits.Store
	AuditService             *services.AuditService
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
	if err := appcompat.EnsureLegacyDataTransferred(cfg.RuntimeRoot, cfg.RevisionStoreDir); err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(cfg.Redis)
	redisBackend := redis.NewBackend(redisClient)

	revisionStore, err := revisions.NewStore(filepath.Join(cfg.RevisionStoreDir, "revisions"))
	if err != nil {
		return nil, err
	}
	revisionSnapshotStore, err := revisionsnapshots.NewStore(filepath.Join(cfg.RevisionStoreDir, "revision-snapshots"))
	if err != nil {
		return nil, err
	}
	eventStore, err := events.NewStore(filepath.Join(cfg.RevisionStoreDir, "events"))
	if err != nil {
		return nil, err
	}
	auditStore, err := audits.NewStore(filepath.Join(cfg.RevisionStoreDir, "audits"))
	if err != nil {
		return nil, err
	}
	jobStore, err := jobs.NewStore(filepath.Join(cfg.RevisionStoreDir, "jobs"))
	if err != nil {
		return nil, err
	}
	roleStore, err := roles.NewStore(filepath.Join(cfg.RevisionStoreDir, "roles"))
	if err != nil {
		return nil, err
	}
	userStore, err := users.NewStore(filepath.Join(cfg.RevisionStoreDir, "users"), users.BootstrapUser{
		Enabled:  cfg.BootstrapAdmin.Enabled,
		ID:       cfg.BootstrapAdmin.ID,
		Username: cfg.BootstrapAdmin.Username,
		Email:    cfg.BootstrapAdmin.Email,
		Password: cfg.BootstrapAdmin.Password,
		RoleIDs:  []string{"admin"},
	})
	if err != nil {
		return nil, err
	}
	sessionStore, err := sessions.NewStore(filepath.Join(cfg.RevisionStoreDir, "sessions"))
	if err != nil {
		return nil, err
	}
	passkeyStore, err := passkeys.NewStore(filepath.Join(cfg.RevisionStoreDir, "passkeys"))
	if err != nil {
		return nil, err
	}
	siteStore, err := sites.NewStore(filepath.Join(cfg.RevisionStoreDir, "sites"))
	if err != nil {
		return nil, err
	}
	upstreamStore, err := upstreams.NewStore(filepath.Join(cfg.RevisionStoreDir, "upstreams"))
	if err != nil {
		return nil, err
	}
	certificateStore, err := certificates.NewStore(filepath.Join(cfg.RevisionStoreDir, "certificates"))
	if err != nil {
		return nil, err
	}
	certificateMaterialStore, err := certificatematerials.NewStore(filepath.Join(cfg.RevisionStoreDir, "certificate-materials"))
	if err != nil {
		return nil, err
	}
	tlsConfigStore, err := tlsconfigs.NewStore(filepath.Join(cfg.RevisionStoreDir, "tlsconfigs"))
	if err != nil {
		return nil, err
	}
	wafPolicyStore, err := wafpolicies.NewStore(filepath.Join(cfg.RevisionStoreDir, "wafpolicies"))
	if err != nil {
		return nil, err
	}
	accessPolicyStore, err := accesspolicies.NewStore(filepath.Join(cfg.RevisionStoreDir, "accesspolicies"))
	if err != nil {
		return nil, err
	}
	rateLimitPolicyStore, err := ratelimitpolicies.NewStore(filepath.Join(cfg.RevisionStoreDir, "ratelimitpolicies"))
	if err != nil {
		return nil, err
	}
	easySiteProfileStore, err := easysiteprofiles.NewStore(filepath.Join(cfg.RevisionStoreDir, "easysiteprofiles"))
	if err != nil {
		return nil, err
	}
	antiDDoSStore, err := antiddos.NewStore(filepath.Join(cfg.RevisionStoreDir, "antiddos"))
	if err != nil {
		return nil, err
	}

	revisionService := services.NewRevisionService(revisionStore)
	setupService := services.NewSetupService(userStore, siteStore, revisionStore)
	auditService := services.NewAuditService(auditStore)
	revisionCompileService := services.NewRevisionCompileService(revisionStore, revisionSnapshotStore, jobStore, siteStore, upstreamStore, certificateStore, tlsConfigStore, wafPolicyStore, accessPolicyStore, rateLimitPolicyStore, easySiteProfileStore, antiDDoSStore, certificateMaterialStore, auditService)
	eventService := services.NewEventService(
		eventStore,
		services.WithRuntimeSecurityCollector(services.NewHTTPRuntimeSecurityEventCollector(cfg.RuntimeHealthURL)),
	)
	jobService := services.NewJobService(jobStore)
	authService := services.NewAuthService(userStore, roleStore, sessionStore, passkeyStore, cfg.AuthIssuer, services.AuthSecurityConfig{
		Pepper: cfg.Security.Pepper,
		WebAuthn: services.WebAuthnConfig{
			Enabled: cfg.Security.WebAuthn.Enabled,
			RPID:    cfg.Security.WebAuthn.RPID,
			RPName:  cfg.Security.WebAuthn.RPName,
			Origins: append([]string(nil), cfg.Security.WebAuthn.Origins...),
		},
	}, auditService)
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
	tlsAutoRenewService, err := services.NewTLSAutoRenewService(filepath.Join(cfg.RevisionStoreDir, "tls-auto-renew"), certificateStore, tlsConfigStore, letsEncryptService)
	if err != nil {
		return nil, err
	}
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
		services.HTTPReloadExecutor{URL: cfg.RuntimeReloadURL},
		services.HTTPHealthChecker{URL: cfg.RuntimeHealthURL},
		auditService,
	)
	easySiteProfileService := services.NewEasySiteProfileService(easySiteProfileStore, siteStore, wafPolicyStore, accessPolicyStore, rateLimitPolicyStore, revisionCompileService, applyService, auditService)
	antiDDoSService := services.NewAntiDDoSService(antiDDoSStore, revisionCompileService, applyService, auditService)
	services.ConfigureAutoApply(revisionCompileService, applyService)
	reportService := services.NewReportService(eventStore, jobStore, revisionStore)
	runtimeRequestCollector := services.NewHTTPRuntimeRequestCollector(cfg.RuntimeHealthURL)
	dashboardService := services.NewDashboardService(eventService, runtimeRequestCollector, cfg.RuntimeHealthURL)
	runtimeCRSService := services.NewRuntimeCRSService(services.RuntimeBaseURLFromHealthURL(cfg.RuntimeHealthURL))
	containerRuntimeService := services.NewContainerRuntimeService()
	adminScriptService := services.NewAdminScriptService(cfg.RevisionStoreDir, detectScriptsRoot())
	httpServer := httpserver.New(cfg.HTTPAddr, cfg.RuntimeRoot, cfg.RevisionStoreDir, cfg.RuntimeHealthURL, setupService, revisionService, authService, siteService, manualBanService, upstreamService, certificateService, tlsConfigService, tlsAutoRenewService, certificateUploadService, certificateMaterialStore, letsEncryptService, selfSignedCertificateService, wafPolicyService, accessPolicyService, rateLimitPolicyService, easySiteProfileService, antiDDoSService, eventService, revisionCompileService, applyService, auditService, reportService, dashboardService, containerRuntimeService, runtimeCRSService, runtimeRequestCollector, adminScriptService)
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
	}

	return &App{
		Config:                   cfg,
		RedisBackend:             redisBackend,
		RevisionStore:            revisionStore,
		RevisionSnapshotStore:    revisionSnapshotStore,
		SetupService:             setupService,
		RevisionService:          revisionService,
		RevisionCompileService:   revisionCompileService,
		ApplyService:             applyService,
		EventStore:               eventStore,
		EventService:             eventService,
		AdminScriptService:       adminScriptService,
		AuditStore:               auditStore,
		AuditService:             auditService,
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
