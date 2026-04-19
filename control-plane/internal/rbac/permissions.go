package rbac

import "sort"

type Permission string

const (
	PermissionAuthSelf Permission = "auth.self"

	PermissionHealthcheckRead Permission = "healthcheck.read"
	PermissionProfileRead     Permission = "profile.read"

	PermissionDashboardRead Permission = "dashboard.read"

	PermissionSitesRead      Permission = "sites.read"
	PermissionSitesWrite     Permission = "sites.write"
	PermissionUpstreamsRead  Permission = "upstreams.read"
	PermissionUpstreamsWrite Permission = "upstreams.write"

	PermissionAntiDDoSRead  Permission = "antiddos.read"
	PermissionAntiDDoSWrite Permission = "antiddos.write"
	PermissionOWASPCRSRead  Permission = "owaspcrs.read"
	PermissionOWASPCRSWrite Permission = "owaspcrs.write"

	PermissionCertificatesRead  Permission = "certificates.read"
	PermissionCertificatesWrite Permission = "certificates.write"
	PermissionTLSRead           Permission = "tls.read"
	PermissionTLSWrite          Permission = "tls.write"

	PermissionPoliciesRead    Permission = "policies.read"
	PermissionPoliciesWrite   Permission = "policies.write"
	PermissionAccessRead      Permission = "access.read"
	PermissionAccessWrite     Permission = "access.write"
	PermissionRateLimitsRead  Permission = "ratelimits.read"
	PermissionRateLimitsWrite Permission = "ratelimits.write"

	PermissionRequestsRead Permission = "requests.read"
	PermissionReportsRead  Permission = "reports.read"
	PermissionEventsRead   Permission = "events.read"
	PermissionBansRead     Permission = "bans.read"

	PermissionRevisionsRead  Permission = "revisions.read"
	PermissionRevisionsWrite Permission = "revisions.write"

	PermissionAdministrationRead       Permission = "administration.read"
	PermissionAdministrationWrite      Permission = "administration.write"
	PermissionAdministrationUsersRead  Permission = "administration.users.read"
	PermissionAdministrationUsersWrite Permission = "administration.users.write"
	PermissionAdministrationRolesRead  Permission = "administration.roles.read"
	PermissionAdministrationRolesWrite Permission = "administration.roles.write"

	PermissionActivityRead Permission = "activity.read"

	PermissionSettingsGeneralRead  Permission = "settings.general.read"
	PermissionSettingsGeneralWrite Permission = "settings.general.write"
	PermissionSettingsStorageRead  Permission = "settings.storage.read"
	PermissionSettingsStorageWrite Permission = "settings.storage.write"
	PermissionSettingsAboutRead    Permission = "settings.about.read"
)

type RoleDefinition struct {
	ID          string
	Name        string
	Permissions []Permission
}

func AllPermissions() []Permission {
	return []Permission{
		PermissionAuthSelf,
		PermissionHealthcheckRead,
		PermissionProfileRead,
		PermissionDashboardRead,
		PermissionSitesRead,
		PermissionSitesWrite,
		PermissionUpstreamsRead,
		PermissionUpstreamsWrite,
		PermissionAntiDDoSRead,
		PermissionAntiDDoSWrite,
		PermissionOWASPCRSRead,
		PermissionOWASPCRSWrite,
		PermissionCertificatesRead,
		PermissionCertificatesWrite,
		PermissionTLSRead,
		PermissionTLSWrite,
		PermissionPoliciesRead,
		PermissionPoliciesWrite,
		PermissionAccessRead,
		PermissionAccessWrite,
		PermissionRateLimitsRead,
		PermissionRateLimitsWrite,
		PermissionRequestsRead,
		PermissionReportsRead,
		PermissionEventsRead,
		PermissionBansRead,
		PermissionRevisionsRead,
		PermissionRevisionsWrite,
		PermissionAdministrationRead,
		PermissionAdministrationWrite,
		PermissionAdministrationUsersRead,
		PermissionAdministrationUsersWrite,
		PermissionAdministrationRolesRead,
		PermissionAdministrationRolesWrite,
		PermissionActivityRead,
		PermissionSettingsGeneralRead,
		PermissionSettingsGeneralWrite,
		PermissionSettingsStorageRead,
		PermissionSettingsStorageWrite,
		PermissionSettingsAboutRead,
	}
}

func IsKnownPermission(permission Permission) bool {
	for _, item := range AllPermissions() {
		if item == permission {
			return true
		}
	}
	return false
}

func DefaultRoleDefinitions() []RoleDefinition {
	commonSelf := []Permission{
		PermissionAuthSelf,
		PermissionHealthcheckRead,
		PermissionProfileRead,
		PermissionSettingsAboutRead,
	}
	readOnlyOps := []Permission{
		PermissionDashboardRead,
		PermissionSitesRead,
		PermissionUpstreamsRead,
		PermissionAntiDDoSRead,
		PermissionOWASPCRSRead,
		PermissionCertificatesRead,
		PermissionTLSRead,
		PermissionPoliciesRead,
		PermissionAccessRead,
		PermissionRateLimitsRead,
		PermissionRequestsRead,
		PermissionReportsRead,
		PermissionEventsRead,
		PermissionBansRead,
		PermissionRevisionsRead,
		PermissionActivityRead,
	}
	adminPermissions := append(append([]Permission(nil), commonSelf...), AllPermissions()...)
	managerPermissions := append(append([]Permission(nil), commonSelf...),
		PermissionDashboardRead,
		PermissionSitesRead,
		PermissionSitesWrite,
		PermissionUpstreamsRead,
		PermissionUpstreamsWrite,
		PermissionCertificatesRead,
		PermissionCertificatesWrite,
		PermissionTLSRead,
		PermissionTLSWrite,
		PermissionRevisionsRead,
		PermissionRevisionsWrite,
		PermissionActivityRead,
		PermissionAdministrationRead,
		PermissionAdministrationUsersRead,
		PermissionAdministrationUsersWrite,
		PermissionAdministrationRolesRead,
	)
	auditorPermissions := append(append([]Permission(nil), commonSelf...), readOnlyOps...)
	socPermissions := append(append([]Permission(nil), commonSelf...),
		PermissionDashboardRead,
		PermissionSitesRead,
		PermissionAntiDDoSRead,
		PermissionAntiDDoSWrite,
		PermissionOWASPCRSRead,
		PermissionOWASPCRSWrite,
		PermissionPoliciesRead,
		PermissionPoliciesWrite,
		PermissionRequestsRead,
		PermissionReportsRead,
		PermissionEventsRead,
		PermissionBansRead,
		PermissionRevisionsRead,
		PermissionActivityRead,
	)
	operatorPermissions := append(append([]Permission(nil), commonSelf...),
		PermissionDashboardRead,
		PermissionSitesRead,
		PermissionSitesWrite,
		PermissionUpstreamsRead,
		PermissionUpstreamsWrite,
		PermissionCertificatesRead,
		PermissionCertificatesWrite,
		PermissionTLSRead,
		PermissionTLSWrite,
		PermissionAntiDDoSRead,
		PermissionAntiDDoSWrite,
		PermissionOWASPCRSRead,
		PermissionOWASPCRSWrite,
		PermissionPoliciesRead,
		PermissionPoliciesWrite,
		PermissionAccessRead,
		PermissionAccessWrite,
		PermissionRateLimitsRead,
		PermissionRateLimitsWrite,
		PermissionRevisionsRead,
		PermissionRevisionsWrite,
		PermissionRequestsRead,
		PermissionReportsRead,
		PermissionEventsRead,
		PermissionBansRead,
		PermissionActivityRead,
		PermissionAdministrationRead,
	)
	viewerPermissions := append(append([]Permission(nil), commonSelf...), readOnlyOps...)

	return []RoleDefinition{
		{ID: "admin", Name: "Administrator", Permissions: SortedPermissions(adminPermissions)},
		{ID: "auditor", Name: "Auditor", Permissions: SortedPermissions(auditorPermissions)},
		{ID: "manager", Name: "Manager", Permissions: SortedPermissions(managerPermissions)},
		{ID: "soc", Name: "SOC", Permissions: SortedPermissions(socPermissions)},
		{ID: "operator", Name: "Operator", Permissions: SortedPermissions(operatorPermissions)},
		{ID: "viewer", Name: "Viewer", Permissions: SortedPermissions(viewerPermissions)},
	}
}

func DefaultRolePermissions() map[string][]Permission {
	out := make(map[string][]Permission)
	for _, item := range DefaultRoleDefinitions() {
		out[item.ID] = append([]Permission(nil), item.Permissions...)
	}
	return out
}

func SortedPermissions(items []Permission) []Permission {
	set := map[Permission]struct{}{}
	out := make([]Permission, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := set[item]; ok {
			continue
		}
		set[item] = struct{}{}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
