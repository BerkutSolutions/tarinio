package httpserver

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/handlers"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/services"
)

type Server struct {
	httpServer *http.Server
}

func New(
	addr string,
	runtimeRoot string,
	revisionStoreDir string,
	runtimeHealthURL string,
	setupService interface {
		Status() (services.SetupStatus, error)
	},
	revisionService *services.RevisionService,
	authService *services.AuthService,
	siteService *services.SiteService,
	siteBanService *services.ManualBanService,
	upstreamService *services.UpstreamService,
	certificateService *services.CertificateService,
	tlsConfigService *services.TLSConfigService,
	tlsAutoRenewService interface {
		Settings() (services.TLSAutoRenewSettings, error)
		UpdateSettings(input services.TLSAutoRenewSettings) (services.TLSAutoRenewSettings, error)
	},
	certificateUploadService *services.CertificateUploadService,
	certificateMaterialReader interface {
		Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error)
	},
	certificateACMEService interface {
		Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *services.ACMEIssueOptions) (jobs.Job, error)
		Renew(ctx context.Context, certificateID string, options *services.ACMEIssueOptions) (jobs.Job, error)
	},
	certificateSelfSignedService interface {
		Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *services.ACMEIssueOptions) (jobs.Job, error)
	},
	wafPolicyService *services.WAFPolicyService,
	accessPolicyService *services.AccessPolicyService,
	rateLimitPolicyService *services.RateLimitPolicyService,
	easySiteProfileService *services.EasySiteProfileService,
	antiDDoSService *services.AntiDDoSService,
	eventService *services.EventService,
	revisionCompileService *services.RevisionCompileService,
	applyService *services.ApplyService,
	auditService *services.AuditService,
	reportService *services.ReportService,
	dashboardService *services.DashboardService,
	containerRuntimeService *services.ContainerRuntimeService,
	runtimeCRSService *services.RuntimeCRSService,
	requestCollector services.RuntimeRequestCollector,
) *Server {
	mux := http.NewServeMux()
	settingsRuntimeHandler := handlers.NewSettingsRuntimeHandler(filepath.Join(revisionStoreDir, "settings"), runtimeHealthURL)
	mux.Handle("/healthz", handlers.NewHealthHandler(revisionService))
	mux.Handle("/api/setup/status", handlers.NewSetupHandler(setupService))
	mux.Handle("/api/app/meta", withAuth(authService, "", handlers.NewAppMetaHandler()))
	mux.Handle("/api/app/ping", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppPingHandler()))
	mux.Handle("/api/app/compat", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppCompatHandler(runtimeRoot, revisionStoreDir)))
	mux.Handle("/api/app/compat/fix", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppCompatHandler(runtimeRoot, revisionStoreDir)))
	mux.Handle("/api/settings/runtime", withAuth(authService, rbac.PermissionAuthSelf, settingsRuntimeHandler))
	mux.Handle("/api/settings/runtime/check-updates", withAuth(authService, rbac.PermissionAuthSelf, settingsRuntimeHandler))
	mux.Handle("/api/settings/runtime/storage-indexes", withAuth(authService, rbac.PermissionAuthSelf, settingsRuntimeHandler))
	mux.Handle("/api/owasp-crs/status", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet: rbac.PermissionPoliciesRead,
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/owasp-crs/check-updates", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodPost: rbac.PermissionPoliciesRead,
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/owasp-crs/update", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodPost: rbac.PermissionPoliciesWrite,
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/auth/bootstrap", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/login", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/login/2fa", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/passkeys/login/begin", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/passkeys/login/finish", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/logout", withAuth(authService, "", handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/me", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/2fa/status", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/2fa/setup", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/2fa/enable", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/2fa/disable", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/change-password", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/login/2fa/passkey/begin", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/login/2fa/passkey/finish", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/passkeys", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/passkeys/register/begin", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/passkeys/register/finish", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/auth/passkeys/", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAuthHandler(authService)))
	mux.Handle("/api/sites", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionSitesRead,
		http.MethodPost: rbac.PermissionSitesWrite,
	}, handlers.NewSitesHandler(siteService, siteBanService)))
	mux.Handle("/api/sites/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionSitesRead,
		http.MethodPost:   rbac.PermissionAccessWrite,
		http.MethodPut:    rbac.PermissionSitesWrite,
		http.MethodDelete: rbac.PermissionSitesWrite,
	}, handlers.NewSitesHandler(siteService, siteBanService)))
	mux.Handle("/api/upstreams", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionUpstreamsRead,
		http.MethodPost: rbac.PermissionUpstreamsWrite,
	}, handlers.NewUpstreamsHandler(upstreamService)))
	mux.Handle("/api/upstreams/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionUpstreamsRead,
		http.MethodPut:    rbac.PermissionUpstreamsWrite,
		http.MethodDelete: rbac.PermissionUpstreamsWrite,
	}, handlers.NewUpstreamsHandler(upstreamService)))
	mux.Handle("/api/certificates", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionCertificatesRead,
		http.MethodPost: rbac.PermissionCertificatesWrite,
	}, handlers.NewCertificatesHandler(certificateService)))
	mux.Handle("/api/certificates/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionCertificatesRead,
		http.MethodPut:    rbac.PermissionCertificatesWrite,
		http.MethodDelete: rbac.PermissionCertificatesWrite,
	}, handlers.NewCertificatesHandler(certificateService)))
	mux.Handle("/api/tls-configs", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionTLSRead,
		http.MethodPost: rbac.PermissionTLSWrite,
	}, handlers.NewTLSConfigsHandler(tlsConfigService)))
	mux.Handle("/api/tls-configs/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionTLSRead,
		http.MethodPut:    rbac.PermissionTLSWrite,
		http.MethodDelete: rbac.PermissionTLSWrite,
	}, handlers.NewTLSConfigsHandler(tlsConfigService)))
	mux.Handle("/api/tls/auto-renew", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet: rbac.PermissionTLSRead,
		http.MethodPut: rbac.PermissionTLSWrite,
	}, handlers.NewTLSAutoRenewHandler(tlsAutoRenewService)))
	certificateACMEHandler := handlers.NewCertificateACMEHandler(certificateACMEService, certificateSelfSignedService)
	mux.Handle("/api/certificate-materials/upload", withAuth(authService, rbac.PermissionCertificatesWrite, handlers.NewCertificateUploadHandler(certificateUploadService)))
	mux.Handle("/api/certificate-materials/import-archive", withAuth(authService, rbac.PermissionCertificatesWrite, handlers.NewCertificateUploadHandler(certificateUploadService)))
	mux.Handle("/api/certificate-materials/export", withAuth(authService, rbac.PermissionCertificatesRead, handlers.NewCertificateMaterialExportHandler(certificateMaterialReader)))
	mux.Handle("/api/certificate-materials/export/", withAuth(authService, rbac.PermissionCertificatesRead, handlers.NewCertificateMaterialExportHandler(certificateMaterialReader)))
	mux.Handle("/api/certificates/acme/issue", withAuth(authService, rbac.PermissionCertificatesWrite, certificateACMEHandler))
	mux.Handle("/api/certificates/acme/renew/", withAuth(authService, rbac.PermissionCertificatesWrite, certificateACMEHandler))
	mux.Handle("/api/certificates/self-signed/issue", withAuth(authService, rbac.PermissionCertificatesWrite, certificateACMEHandler))
	mux.Handle("/api/waf-policies", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionPoliciesRead,
		http.MethodPost: rbac.PermissionPoliciesWrite,
	}, handlers.NewWAFPoliciesHandler(wafPolicyService)))
	mux.Handle("/api/waf-policies/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionPoliciesRead,
		http.MethodPut:    rbac.PermissionPoliciesWrite,
		http.MethodDelete: rbac.PermissionPoliciesWrite,
	}, handlers.NewWAFPoliciesHandler(wafPolicyService)))
	mux.Handle("/api/access-policies", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionAccessRead,
		http.MethodPost: rbac.PermissionAccessWrite,
	}, handlers.NewAccessPoliciesHandler(accessPolicyService)))
	mux.Handle("/api/access-policies/upsert", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodPost: rbac.PermissionAccessWrite,
		http.MethodPut:  rbac.PermissionAccessWrite,
	}, handlers.NewAccessPoliciesHandler(accessPolicyService)))
	mux.Handle("/api/access-policies/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionAccessRead,
		http.MethodPut:    rbac.PermissionAccessWrite,
		http.MethodDelete: rbac.PermissionAccessWrite,
	}, handlers.NewAccessPoliciesHandler(accessPolicyService)))
	mux.Handle("/api/rate-limit-policies", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionRateLimitsRead,
		http.MethodPost: rbac.PermissionRateLimitsWrite,
	}, handlers.NewRateLimitPoliciesHandler(rateLimitPolicyService)))
	mux.Handle("/api/rate-limit-policies/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    rbac.PermissionRateLimitsRead,
		http.MethodPut:    rbac.PermissionRateLimitsWrite,
		http.MethodDelete: rbac.PermissionRateLimitsWrite,
	}, handlers.NewRateLimitPoliciesHandler(rateLimitPolicyService)))
	mux.Handle("/api/easy-site-profiles/", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionSitesRead,
		http.MethodPut:  rbac.PermissionSitesWrite,
		http.MethodPost: rbac.PermissionSitesWrite,
	}, handlers.NewEasySiteProfilesHandler(easySiteProfileService)))
	mux.Handle("/api/easy-site-profiles/catalog/countries", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet: rbac.PermissionSitesRead,
	}, handlers.NewEasySiteProfileCatalogHandler()))
	mux.Handle("/api/anti-ddos/settings", withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:  rbac.PermissionPoliciesRead,
		http.MethodPut:  rbac.PermissionPoliciesWrite,
		http.MethodPost: rbac.PermissionPoliciesWrite,
	}, handlers.NewAntiDDoSHandler(antiDDoSService)))
	mux.Handle("/api/events", withAuth(authService, rbac.PermissionReportsRead, handlers.NewEventsHandler(eventService)))
	mux.Handle("/api/requests", withAuth(authService, rbac.PermissionReportsRead, handlers.NewRequestsHandler(requestCollector)))
	mux.Handle("/api/reports/revisions", withAuth(authService, rbac.PermissionReportsRead, handlers.NewReportsHandler(reportService)))
	mux.Handle("/api/dashboard/stats", withAuth(authService, rbac.PermissionReportsRead, handlers.NewDashboardHandler(dashboardService)))
	mux.Handle("/api/dashboard/containers/overview", withAuth(authService, rbac.PermissionReportsRead, handlers.NewDashboardContainersHandler(containerRuntimeService)))
	mux.Handle("/api/dashboard/containers/logs", withAuth(authService, rbac.PermissionReportsRead, handlers.NewDashboardContainersHandler(containerRuntimeService)))
	mux.Handle("/api/audit", withAuth(authService, rbac.PermissionAdministrationRead, handlers.NewAuditHandler(auditService)))
	mux.Handle("/api/revisions/compile", withAuth(authService, rbac.PermissionRevisionsWrite, handlers.NewRevisionCompileHandler(revisionCompileService)))
	mux.Handle("/api/revisions/", withAuth(authService, rbac.PermissionRevisionsWrite, handlers.NewRevisionApplyHandler(applyService)))

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
