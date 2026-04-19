package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/users"
)

type healthRevisionService interface {
	RevisionCount() (int, error)
}

type healthRevisionCatalogProbeService interface {
	Probe(ctx context.Context) (services.RevisionCatalogProbe, error)
}

type healthSetupService interface {
	Status() (services.SetupStatus, error)
}

type healthSessionCounter interface {
	Count() (int, error)
}

type healthUserCounter interface {
	Count() (int, error)
	List() ([]users.User, error)
}

type healthRoleReader interface {
	List() ([]roles.Role, error)
}

type healthCompilerPreviewer interface {
	Preview() (revisionsnapshots.Snapshot, error)
}

type healthRuntimeReadyProber interface {
	Probe() error
}

type healthRuntimeSecurityProber interface {
	Probe() error
}

type healthRuntimeRequestsProber interface {
	Probe(query url.Values) error
}

type healthRuntimeCRSStatuser interface {
	Status(ctx context.Context) (services.RuntimeCRSStatus, error)
}

type healthComponent map[string]any

type HealthHandler struct {
	revisions      healthRevisionService
	revisionsAPI   healthRevisionCatalogProbeService
	setup          healthSetupService
	sessions       healthSessionCounter
	users          healthUserCounter
	roles          healthRoleReader
	compiler       healthCompilerPreviewer
	runtimeReady   healthRuntimeReadyProber
	runtimeEvents  healthRuntimeSecurityProber
	runtimeRequest healthRuntimeRequestsProber
	runtimeCRS     healthRuntimeCRSStatuser
}

func NewHealthHandler(
	revisions healthRevisionService,
	revisionsAPI healthRevisionCatalogProbeService,
	setup healthSetupService,
	sessions healthSessionCounter,
	users healthUserCounter,
	roles healthRoleReader,
	compiler healthCompilerPreviewer,
	runtimeReady healthRuntimeReadyProber,
	runtimeEvents healthRuntimeSecurityProber,
	runtimeRequest healthRuntimeRequestsProber,
	runtimeCRS healthRuntimeCRSStatuser,
) *HealthHandler {
	return &HealthHandler{
		revisions:      revisions,
		revisionsAPI:   revisionsAPI,
		setup:          setup,
		sessions:       sessions,
		users:          users,
		roles:          roles,
		compiler:       compiler,
		runtimeReady:   runtimeReady,
		runtimeEvents:  runtimeEvents,
		runtimeRequest: runtimeRequest,
		runtimeCRS:     runtimeCRS,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	components := map[string]healthComponent{}
	status := "ok"
	httpStatus := http.StatusOK
	hardFailure := false

	setDegraded := func(name string, payload healthComponent) {
		payload["status"] = "degraded"
		components[name] = payload
		status = "degraded"
		if hardFailure {
			httpStatus = http.StatusServiceUnavailable
		}
	}
	setOK := func(name string, payload healthComponent) {
		payload["status"] = "ok"
		components[name] = payload
	}

	count, err := h.revisions.RevisionCount()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded",
			"error":  "revision store unavailable",
		})
		return
	}
	setOK("revision_store", healthComponent{
		"revision_count": count,
	})

	if catalogProbe, err := h.revisionsAPI.Probe(r.Context()); err != nil {
		hardFailure = true
		setDegraded("revisions_api", healthComponent{"error": "catalog probe failed"})
	} else {
		setOK("revisions_api", healthComponent{
			"service_count":  catalogProbe.ServiceCount,
			"revision_count": catalogProbe.RevisionCount,
			"timeline_count": catalogProbe.TimelineCount,
		})
	}

	if h.setup != nil {
		setupStatus, err := h.setup.Status()
		if err != nil {
			hardFailure = true
			setDegraded("setup", healthComponent{"error": "setup status unavailable"})
		} else {
			setOK("setup", healthComponent{
				"has_users":           setupStatus.HasUsers,
				"has_sites":           setupStatus.HasSites,
				"has_active_revision": setupStatus.HasActiveRevision,
				"needs_bootstrap":     setupStatus.NeedsBootstrap,
			})
		}
	}

	if h.users != nil && h.sessions != nil {
		userCount, userErr := h.users.Count()
		sessionCount, sessionErr := h.sessions.Count()
		if userErr != nil || sessionErr != nil {
			hardFailure = true
			setDegraded("auth_state", healthComponent{"error": "auth/session backend unavailable"})
		} else {
			setOK("auth_state", healthComponent{
				"user_count":    userCount,
				"session_count": sessionCount,
			})
		}
	}

	if h.users != nil && h.roles != nil {
		userItems, userErr := h.users.List()
		roleItems, roleErr := h.roles.List()
		if userErr != nil || roleErr != nil {
			hardFailure = true
			setDegraded("zero_trust", healthComponent{"error": "user/role integrity check unavailable"})
		} else {
			zeroTrustStatus, payload := zeroTrustHealthPayload(userItems, roleItems)
			if zeroTrustStatus != "ok" {
				setDegraded("zero_trust", payload)
			} else {
				setOK("zero_trust", payload)
			}
		}
	}

	if h.compiler != nil {
		if _, err := h.compiler.Preview(); err != nil {
			hardFailure = true
			setDegraded("compiler_preview", healthComponent{"error": "compiler preview failed"})
		} else {
			setOK("compiler_preview", healthComponent{"preview": "ok"})
		}
	}

	shouldProbeRuntime := h.setup == nil
	if h.setup != nil {
		if setupStatus, err := h.setup.Status(); err == nil {
			shouldProbeRuntime = setupStatus.HasActiveRevision
		}
	}

	if shouldProbeRuntime && h.runtimeReady != nil {
		if err := runHealthProbe(1200*time.Millisecond, func() error { return h.runtimeReady.Probe() }); err != nil {
			setDegraded("runtime_ready", healthComponent{"error": "runtime ready probe failed"})
		} else {
			setOK("runtime_ready", healthComponent{"ready": true})
		}
	} else if h.runtimeReady != nil {
		setOK("runtime_ready", healthComponent{"probe": "skipped"})
	}

	if shouldProbeRuntime && h.runtimeEvents != nil {
		if err := runHealthProbe(1200*time.Millisecond, func() error { return h.runtimeEvents.Probe() }); err != nil {
			setDegraded("runtime_security_events", healthComponent{"error": "runtime security events probe failed"})
		} else {
			setOK("runtime_security_events", healthComponent{"probe": "ok"})
		}
	} else if h.runtimeEvents != nil {
		setOK("runtime_security_events", healthComponent{"probe": "skipped"})
	}

	if shouldProbeRuntime && h.runtimeRequest != nil {
		if err := runHealthProbe(1200*time.Millisecond, func() error { return h.runtimeRequest.Probe(nil) }); err != nil {
			setDegraded("runtime_requests", healthComponent{"error": "runtime requests probe failed"})
		} else {
			setOK("runtime_requests", healthComponent{"probe": "ok"})
		}
	} else if h.runtimeRequest != nil {
		setOK("runtime_requests", healthComponent{"probe": "skipped"})
	}

	if shouldProbeRuntime && h.runtimeCRS != nil {
		probeCtx, cancel := context.WithTimeout(r.Context(), 1200*time.Millisecond)
		crsStatus, err := h.runtimeCRS.Status(probeCtx)
		cancel()
		if err != nil {
			setDegraded("runtime_crs", healthComponent{"error": "runtime CRS status unavailable"})
		} else {
			setOK("runtime_crs", healthComponent{
				"active_version":             crsStatus.ActiveVersion,
				"has_update":                 crsStatus.HasUpdate,
				"hourly_auto_update_enabled": crsStatus.HourlyAutoUpdateEnabled,
			})
		}
	} else if h.runtimeCRS != nil {
		setOK("runtime_crs", healthComponent{"probe": "skipped"})
	}

	writeJSON(w, httpStatus, map[string]any{
		"status":         status,
		"revision_count": count,
		"components":     components,
	})
}

func zeroTrustHealthPayload(userItems []users.User, roleItems []roles.Role) (string, healthComponent) {
	rolesByID := map[string]roles.Role{}
	for _, item := range roleItems {
		rolesByID[item.ID] = item
	}
	missingDefaults := []string{}
	for _, item := range []string{"admin", "auditor", "manager", "soc"} {
		if _, ok := rolesByID[item]; !ok {
			missingDefaults = append(missingDefaults, item)
		}
	}
	adminRole, hasAdminRole := rolesByID["admin"]
	unknownRoleRefs := 0
	inactiveAdmins := 0
	for _, user := range userItems {
		for _, roleID := range user.RoleIDs {
			if _, ok := rolesByID[roleID]; !ok && roleID != "admin" {
				unknownRoleRefs++
			}
		}
		if isHealthBuiltInAdmin(user) && !user.IsActive {
			inactiveAdmins++
		}
	}
	status := "ok"
	if len(missingDefaults) > 0 || !hasAdminRole || len(adminRole.Permissions) < len(rbac.AllPermissions()) || unknownRoleRefs > 0 || inactiveAdmins > 0 {
		status = "degraded"
	}
	return status, healthComponent{
		"user_count":              len(userItems),
		"role_count":              len(roleItems),
		"missing_default_roles":   missingDefaults,
		"unknown_role_references": unknownRoleRefs,
		"inactive_admins":         inactiveAdmins,
		"admin_permissions":       len(adminRole.Permissions),
		"known_permissions":       len(rbac.AllPermissions()),
	}
}

func isHealthBuiltInAdmin(item users.User) bool {
	return strings.EqualFold(item.ID, "admin") || strings.EqualFold(item.Username, "admin")
}

func runHealthProbe(timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	writeJSON(w, status, payload)
}
