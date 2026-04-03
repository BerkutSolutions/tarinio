package rbac

import "sort"

type Permission string

const (
	PermissionSitesRead           Permission = "sites.read"
	PermissionSitesWrite          Permission = "sites.write"
	PermissionUpstreamsRead       Permission = "upstreams.read"
	PermissionUpstreamsWrite      Permission = "upstreams.write"
	PermissionCertificatesRead    Permission = "certificates.read"
	PermissionCertificatesWrite   Permission = "certificates.write"
	PermissionTLSRead             Permission = "tls.read"
	PermissionTLSWrite            Permission = "tls.write"
	PermissionPoliciesRead        Permission = "policies.read"
	PermissionPoliciesWrite       Permission = "policies.write"
	PermissionAccessRead          Permission = "access.read"
	PermissionAccessWrite         Permission = "access.write"
	PermissionRateLimitsRead      Permission = "ratelimits.read"
	PermissionRateLimitsWrite     Permission = "ratelimits.write"
	PermissionRevisionsRead       Permission = "revisions.read"
	PermissionRevisionsWrite      Permission = "revisions.write"
	PermissionReportsRead         Permission = "reports.read"
	PermissionAdministrationRead  Permission = "administration.read"
	PermissionAdministrationWrite Permission = "administration.write"
	PermissionAuthSelf            Permission = "auth.self"
)

func DefaultRolePermissions() map[string][]Permission {
	admin := []Permission{
		PermissionSitesRead,
		PermissionSitesWrite,
		PermissionUpstreamsRead,
		PermissionUpstreamsWrite,
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
		PermissionRevisionsRead,
		PermissionRevisionsWrite,
		PermissionReportsRead,
		PermissionAdministrationRead,
		PermissionAdministrationWrite,
		PermissionAuthSelf,
	}
	operator := []Permission{
		PermissionSitesRead,
		PermissionSitesWrite,
		PermissionUpstreamsRead,
		PermissionUpstreamsWrite,
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
		PermissionRevisionsRead,
		PermissionRevisionsWrite,
		PermissionReportsRead,
		PermissionAdministrationRead,
		PermissionAuthSelf,
	}
	viewer := []Permission{
		PermissionSitesRead,
		PermissionUpstreamsRead,
		PermissionCertificatesRead,
		PermissionTLSRead,
		PermissionPoliciesRead,
		PermissionAccessRead,
		PermissionRateLimitsRead,
		PermissionRevisionsRead,
		PermissionReportsRead,
		PermissionAdministrationRead,
		PermissionAuthSelf,
	}
	return map[string][]Permission{
		"admin":    admin,
		"operator": operator,
		"viewer":   viewer,
	}
}

func SortedPermissions(items []Permission) []Permission {
	out := append([]Permission(nil), items...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
