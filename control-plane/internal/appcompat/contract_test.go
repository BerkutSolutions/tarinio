package appcompat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityContract_UIAndInstallerInSync(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	appJSPath := filepath.Join(root, "ui", "app", "static", "js", "app.js")
	healthcheckJSPath := filepath.Join(root, "ui", "app", "static", "js", "healthcheck.js")
	installerPath := filepath.Join(root, "scripts", "install-aio.sh")

	appJS, err := os.ReadFile(appJSPath)
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	healthcheckJS, err := os.ReadFile(healthcheckJSPath)
	if err != nil {
		t.Fatalf("read healthcheck.js: %v", err)
	}
	installer, err := os.ReadFile(installerPath)
	if err != nil {
		t.Fatalf("read install-aio.sh: %v", err)
	}

	registry := DefaultRegistry()
	for _, item := range registry {
		moduleID := strings.TrimSpace(item.ModuleID)
		if moduleID == "" {
			t.Fatalf("registry contains empty module id")
		}
		if !strings.Contains(string(appJS), `id: "`+moduleID+`"`) {
			t.Fatalf("compat contract broken: module %q missing in ui/app/static/js/app.js sections; update UI section list and diff", moduleID)
		}
		if !strings.Contains(string(healthcheckJS), `tab.`+moduleID) {
			t.Fatalf("compat contract broken: module %q missing in ui/app/static/js/healthcheck.js TAB_PROBES; update healthcheck probes and diff", moduleID)
		}
	}

	if !strings.Contains(string(healthcheckJS), `const CONTRACT_VERSION = "`+ContractVersion+`"`) {
		t.Fatalf("compat contract broken: healthcheck CONTRACT_VERSION does not match appcompat.ContractVersion=%q; update both according to diff", ContractVersion)
	}
	if !strings.Contains(string(installer), `COMPAT_CONTRACT_VERSION="${COMPAT_CONTRACT_VERSION:-`+ContractVersion+`}"`) {
		t.Fatalf("compat contract broken: install-aio.sh COMPAT_CONTRACT_VERSION does not match appcompat.ContractVersion=%q; update installer according to diff", ContractVersion)
	}
}
