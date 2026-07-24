package tests

import (
	"os"
	"strings"
	"testing"
)

func TestAdministrationModalLayersAboveSidebar(t *testing.T) {
	content, err := os.ReadFile("../app/static/waf.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(content)
	if !strings.Contains(css, "#administration-entity-modal {\n  z-index: 1400;") {
		t.Fatal("administration entity modal must layer above the z-index 1300 sidebar")
	}
}

func TestAdministrationMutationControlsRespectWritePermissions(t *testing.T) {
	pageContent, err := os.ReadFile("../app/static/js/pages/administration.js")
	if err != nil {
		t.Fatal(err)
	}
	page := string(pageContent)
	for _, required := range []string{
		`permissions.has("administration.write") && permissions.has("administration.users.write")`,
		`permissions.has("administration.write") && permissions.has("administration.roles.write")`,
		`renderUsersTable(state.users, state.roles, ctx, canWriteUsers)`,
		`renderRolesTable(state.roles, ctx, canWriteRoles)`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("administration page is missing write-permission gate %q", required)
		}
	}

	helperContent, err := os.ReadFile("../app/static/js/pages/administration.helpers-base.js")
	if err != nil {
		t.Fatal(err)
	}
	helper := string(helperContent)
	for _, required := range []string{
		"canWrite && !user?.is_builtin",
		`canWrite && !["admin", "auditor", "manager", "soc"].includes(roleID)`,
	} {
		if !strings.Contains(helper, required) {
			t.Fatalf("administration table is missing protected delete control %q", required)
		}
	}
}
