package httpserver

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/handlers"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/storage"
	"waf/control-plane/internal/telemetry"
	"waf/control-plane/internal/users"
)

type Server struct {
	httpServer *http.Server
}

func New(
	addr string,
	runtimeRoot string,
	revisionStoreDir string,
	runtimeHealthURL string,
	haEnabled bool,
	haNodeID string,
	metricsToken string,
	stateBackend storage.Backend,
	setupService interface {
		Status() (services.SetupStatus, error)
	},
	revisionService *services.RevisionService,
	authService *services.AuthService,
	enterpriseService *services.EnterpriseService,
	sessionStore *sessions.Store,
	userStore *users.Store,
	roleStore *roles.Store,
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
	revisionCatalogService *services.RevisionCatalogService,
	auditService *services.AuditService,
	reportService *services.ReportService,
	dashboardService *services.DashboardService,
	containerRuntimeService *services.ContainerRuntimeService,
	runtimeCRSService *services.RuntimeCRSService,
	requestCollector services.RuntimeRequestCollector,
	runtimeReadyProbe interface{ Probe() error },
	runtimeSecurityProbe interface{ Probe() error },
	runtimeRequestProbe interface{ Probe(url.Values) error },
	adminScriptService *services.AdminScriptService,
) *Server {
	mux := http.NewServeMux()
	metrics := telemetry.Default()
	settingsRuntimeHandler := handlers.NewSettingsRuntimeHandlerWithBackend(
		filepath.Join(revisionStoreDir, "settings"),
		runtimeHealthURL,
		stateBackend,
		strings.TrimSpace(os.Getenv("CONTROL_PLANE_SECURITY_PEPPER")),
	)
	administrationUsersHandler := handlers.NewAdministrationUsersHandlerWithSessions(userStore, roleStore, sessionStore)
	administrationRolesHandler := handlers.NewAdministrationRolesHandler(roleStore, userStore)
	zeroTrustHealthHandler := handlers.NewZeroTrustHealthHandler(userStore, roleStore)
	mux.Handle("/healthz", handlers.NewHealthHandler(revisionService, revisionCatalogService, setupService, sessionStore, userStore, roleStore, revisionCompileService, runtimeReadyProbe, runtimeSecurityProbe, runtimeRequestProbe, runtimeCRSService))
	mux.Handle("/metrics", metricsHandler(metrics.Registry(), metricsToken))
	mux.Handle("/api/setup/status", handlers.NewSetupHandler(setupService))
	mux.Handle("/api/app/meta", withAuth(authService, "", handlers.NewAppMetaHandler(haEnabled, haNodeID)))
	mux.Handle("/api/app/ping", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppPingHandler()))
	mux.Handle("/api/app/compat", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppCompatHandler(runtimeRoot, revisionStoreDir)))
	mux.Handle("/api/app/compat/fix", withAuth(authService, rbac.PermissionAuthSelf, handlers.NewAppCompatHandler(runtimeRoot, revisionStoreDir)))
	mux.Handle("/api/settings/runtime", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionSettingsGeneralRead},
		http.MethodPut: {rbac.PermissionSettingsGeneralWrite},
	}, settingsRuntimeHandler))
	mux.Handle("/api/settings/runtime/check-updates", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodPost: {rbac.PermissionSettingsGeneralWrite},
	}, settingsRuntimeHandler))
	mux.Handle("/api/settings/runtime/storage-indexes", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:    {rbac.PermissionSettingsStorageRead},
		http.MethodDelete: {rbac.PermissionSettingsStorageWrite},
	}, settingsRuntimeHandler))
	mux.Handle("/api/owasp-crs/status", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionOWASPCRSRead, rbac.PermissionPoliciesRead},
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/owasp-crs/check-updates", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodPost: {rbac.PermissionOWASPCRSRead, rbac.PermissionPoliciesRead},
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/owasp-crs/update", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodPost: {rbac.PermissionOWASPCRSWrite, rbac.PermissionPoliciesWrite},
	}, handlers.NewOWASPCRSHandler(runtimeCRSService)))
	mux.Handle("/api/auth/bootstrap", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/login", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/login/2fa", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/passkeys/login/begin", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/passkeys/login/finish", handlers.NewAuthHandler(authService))
	mux.Handle("/api/auth/providers", handlers.NewOIDCHandler(enterpriseService))
	mux.Handle("/api/auth/oidc/start", handlers.NewOIDCHandler(enterpriseService))
	mux.Handle("/api/auth/oidc/callback", handlers.NewOIDCHandler(enterpriseService))
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
	mux.Handle("/api/sites", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionSitesRead},
		http.MethodPost: {rbac.PermissionSitesWrite},
	}, handlers.NewSitesHandler(siteService, siteBanService)))
	mux.Handle("/api/sites/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:    {rbac.PermissionSitesRead},
		http.MethodPost:   {rbac.PermissionAccessWrite},
		http.MethodPut:    {rbac.PermissionSitesWrite},
		http.MethodDelete: {rbac.PermissionSitesWrite},
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
	mux.Handle("/api/certificates", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionTLSRead, rbac.PermissionCertificatesRead},
		http.MethodPost: {rbac.PermissionTLSWrite, rbac.PermissionCertificatesWrite},
	}, handlers.NewCertificatesHandler(certificateService)))
	mux.Handle("/api/certificates/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:    {rbac.PermissionTLSRead, rbac.PermissionCertificatesRead},
		http.MethodPut:    {rbac.PermissionTLSWrite, rbac.PermissionCertificatesWrite},
		http.MethodDelete: {rbac.PermissionTLSWrite, rbac.PermissionCertificatesWrite},
	}, handlers.NewCertificatesHandler(certificateService)))
	mux.Handle("/api/tls-configs", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionTLSRead},
		http.MethodPost: {rbac.PermissionTLSWrite},
	}, handlers.NewTLSConfigsHandler(tlsConfigService)))
	mux.Handle("/api/tls-configs/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:    {rbac.PermissionTLSRead},
		http.MethodPut:    {rbac.PermissionTLSWrite},
		http.MethodDelete: {rbac.PermissionTLSWrite},
	}, handlers.NewTLSConfigsHandler(tlsConfigService)))
	mux.Handle("/api/tls/auto-renew", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionTLSRead},
		http.MethodPut: {rbac.PermissionTLSWrite},
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
	mux.Handle("/api/anti-ddos/settings", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionAntiDDoSRead, rbac.PermissionPoliciesRead},
		http.MethodPut:  {rbac.PermissionAntiDDoSWrite, rbac.PermissionPoliciesWrite},
		http.MethodPost: {rbac.PermissionAntiDDoSWrite, rbac.PermissionPoliciesWrite},
	}, handlers.NewAntiDDoSHandler(antiDDoSService)))
	mux.Handle("/api/events", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionEventsRead, rbac.PermissionReportsRead},
		http.MethodPost: {rbac.PermissionEventsRead, rbac.PermissionReportsRead},
	}, handlers.NewEventsHandler(eventService)))
	mux.Handle("/api/requests", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionRequestsRead, rbac.PermissionReportsRead},
		http.MethodPost: {rbac.PermissionRequestsRead, rbac.PermissionReportsRead},
	}, handlers.NewRequestsHandler(requestCollector)))
	mux.Handle("/api/reports/revisions", withAuth(authService, rbac.PermissionReportsRead, handlers.NewReportsHandler(reportService)))
	mux.Handle("/api/revisions", withAuth(authService, rbac.PermissionRevisionsRead, handlers.NewRevisionCatalogHandler(revisionCatalogService)))
	mux.Handle("/api/revisions/statuses", withAuth(authService, rbac.PermissionRevisionsWrite, handlers.NewRevisionStatusHandler(revisionCatalogService)))
	mux.Handle("/api/dashboard/stats", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionDashboardRead, rbac.PermissionReportsRead},
	}, handlers.NewDashboardHandler(dashboardService)))
	mux.Handle("/api/dashboard/containers/overview", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionDashboardRead, rbac.PermissionReportsRead},
	}, handlers.NewDashboardContainersHandler(containerRuntimeService)))
	mux.Handle("/api/dashboard/containers/logs", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionDashboardRead, rbac.PermissionReportsRead, rbac.PermissionAdministrationRead},
	}, handlers.NewDashboardContainersHandler(containerRuntimeService)))
	mux.Handle("/api/dashboard/containers/issues", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionDashboardRead, rbac.PermissionReportsRead},
	}, handlers.NewDashboardContainersHandler(containerRuntimeService)))
	mux.Handle("/api/audit", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionActivityRead},
	}, handlers.NewAuditHandler(auditService)))
	mux.Handle("/api/administration/users", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionAdministrationRead, rbac.PermissionAdministrationUsersRead},
		http.MethodPost: {rbac.PermissionAdministrationWrite, rbac.PermissionAdministrationUsersWrite},
	}, administrationUsersHandler))
	mux.Handle("/api/administration/users/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionAdministrationRead, rbac.PermissionAdministrationUsersRead},
		http.MethodPut: {rbac.PermissionAdministrationWrite, rbac.PermissionAdministrationUsersWrite},
	}, administrationUsersHandler))
	mux.Handle("/api/administration/roles", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionAdministrationRead, rbac.PermissionAdministrationRolesRead},
		http.MethodPost: {rbac.PermissionAdministrationWrite, rbac.PermissionAdministrationRolesWrite},
	}, administrationRolesHandler))
	mux.Handle("/api/administration/roles/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionAdministrationRead, rbac.PermissionAdministrationRolesRead},
		http.MethodPut: {rbac.PermissionAdministrationWrite, rbac.PermissionAdministrationRolesWrite},
	}, administrationRolesHandler))
	mux.Handle("/api/administration/zero-trust/health", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionHealthcheckRead},
	}, zeroTrustHealthHandler))
	mux.Handle("/api/administration/scripts", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionAdministrationRead},
	}, handlers.NewAdministrationScriptsHandler(adminScriptService)))
	mux.Handle("/api/administration/scripts/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet:  {rbac.PermissionAdministrationRead},
		http.MethodPost: {rbac.PermissionAdministrationWrite},
	}, handlers.NewAdministrationScriptsHandler(adminScriptService)))
	mux.Handle("/api/administration/enterprise", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionAdministrationRead},
		http.MethodPut: {rbac.PermissionAdministrationWrite},
	}, handlers.NewEnterpriseHandler(enterpriseService)))
	mux.Handle("/api/administration/enterprise/scim-tokens", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodPost: {rbac.PermissionAdministrationWrite},
	}, handlers.NewEnterpriseHandler(enterpriseService)))
	mux.Handle("/api/administration/enterprise/scim-tokens/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodDelete: {rbac.PermissionAdministrationWrite},
	}, handlers.NewEnterpriseHandler(enterpriseService)))
	mux.Handle("/api/administration/support-bundle", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodGet: {rbac.PermissionAdministrationRead},
	}, handlers.NewEnterpriseHandler(enterpriseService)))
	mux.Handle("/api/revisions/compile", withAuth(authService, rbac.PermissionRevisionsWrite, handlers.NewRevisionCompileHandler(revisionCompileService)))
	mux.Handle("/api/revisions/", withMethodAllPermissions(authService, map[string][]rbac.Permission{
		http.MethodDelete: {rbac.PermissionRevisionsWrite},
		http.MethodPost:   {rbac.PermissionRevisionsWrite},
	}, handlers.NewRevisionApplyHandler(applyService, revisionCatalogService, enterpriseService)))
	mux.Handle("/scim/v2/ServiceProviderConfig", handlers.NewSCIMHandler(enterpriseService))
	mux.Handle("/scim/v2/Users", handlers.NewSCIMHandler(enterpriseService))
	mux.Handle("/scim/v2/Users/", handlers.NewSCIMHandler(enterpriseService))
	mux.Handle("/scim/v2/Groups", handlers.NewSCIMHandler(enterpriseService))

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           metrics.InstrumentHTTP(mux, haNodeID),
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
