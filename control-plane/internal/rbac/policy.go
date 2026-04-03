package rbac

import (
	"sort"
	"strings"
)

type RoleReader interface {
	PermissionsForRoleIDs(roleIDs []string) []Permission
}

type Policy struct {
	roles RoleReader
}

func NewPolicy(roles RoleReader) *Policy {
	return &Policy{roles: roles}
}

func (p *Policy) Allowed(roleIDs []string, permission Permission) bool {
	if p == nil || p.roles == nil {
		return false
	}
	for _, item := range p.roles.PermissionsForRoleIDs(roleIDs) {
		if item == permission {
			return true
		}
	}
	return false
}

func (p *Policy) Permissions(roleIDs []string) []string {
	if p == nil || p.roles == nil {
		return nil
	}
	items := p.roles.PermissionsForRoleIDs(roleIDs)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, string(item))
	}
	sort.Strings(out)
	return out
}

func NormalizeRoleID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
