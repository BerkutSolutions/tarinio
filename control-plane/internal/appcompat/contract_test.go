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
	enterpriseInstallerPath := filepath.Join(root, "scripts", "install-aio-enterprise.sh")
	rotateSecretsPath := filepath.Join(root, "scripts", "rotate-env-secrets.sh")

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
	enterpriseInstaller, err := os.ReadFile(enterpriseInstallerPath)
	if err != nil {
		t.Fatalf("read install-aio-enterprise.sh: %v", err)
	}
	rotateSecretsScript, err := os.ReadFile(rotateSecretsPath)
	if err != nil {
		t.Fatalf("read rotate-env-secrets.sh: %v", err)
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
	for _, candidate := range []struct {
		name    string
		content string
	}{
		{name: "install-aio.sh", content: string(installer)},
		{name: "install-aio-enterprise.sh", content: string(enterpriseInstaller)},
	} {
		if !strings.Contains(candidate.content, `COMPAT_CONTRACT_VERSION="${COMPAT_CONTRACT_VERSION:-`+ContractVersion+`}"`) {
			t.Fatalf("compat contract broken: %s COMPAT_CONTRACT_VERSION does not match appcompat.ContractVersion=%q; update installer according to diff", candidate.name, ContractVersion)
		}
		if !strings.Contains(candidate.content, `scripts/rotate-env-secrets.sh`) {
			t.Fatalf("compat contract broken: %s must reference scripts/rotate-env-secrets.sh for explicit secret rotation guidance", candidate.name)
		}
		if !strings.Contains(candidate.content, `IS_UPGRADE=1`) {
			t.Fatalf("compat contract broken: %s must mark existing installations as upgrade flow", candidate.name)
		}
	}
	if !strings.Contains(string(rotateSecretsScript), `postgres password rotated`) {
		t.Fatalf("compat contract broken: rotate-env-secrets.sh must rotate postgres credentials before restarting the stack")
	}
	if !strings.Contains(string(rotateSecretsScript), `stack health verified after secret rotation`) {
		t.Fatalf("compat contract broken: rotate-env-secrets.sh must verify stack health after secret rotation")
	}
}
