package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sentinelComposeContract struct {
	path     string
	services []string
	volumes  []string
}

func TestSentinelComposeProfilesRemainRootless(t *testing.T) {
	contracts := []sentinelComposeContract{
		{"default", []string{"tarinio-sentinel"}, []string{"waf-sentinel-state-v2", "waf-l4-adaptive-v2"}},
		{"auto-start", []string{"tarinio-sentinel"}, []string{"waf-ddos-model-state-auto-start-v2", "waf-l4-adaptive-auto-start-v2"}},
		{"enterprise", []string{"tarinio-sentinel"}, []string{"waf-ha-ddos-model-state-v2", "waf-ha-l4-adaptive-v2"}},
		{"ha-lab", []string{"tarinio-sentinel"}, []string{"waf-ha-sentinel-state-v2", "waf-ha-l4-adaptive-v2"}},
		{"e2e", []string{"tarinio-sentinel"}, []string{"waf-e2e-sentinel-state-v2", "waf-e2e-l4-adaptive-v2"}},
		{"testpage", []string{"tarinio-sentinel-mgmt", "tarinio-sentinel-app"}, []string{
			"waf-test-mgmt-sentinel-state-v2", "waf-test-mgmt-l4-adaptive-v2",
			"waf-test-app-sentinel-state-v2", "waf-test-app-l4-adaptive-v2",
		}},
	}

	for _, contract := range contracts {
		contract := contract
		t.Run(contract.path, func(t *testing.T) {
			path := filepath.Join("..", "..", "deploy", "compose", contract.path, "docker-compose.yml")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			source := strings.ReplaceAll(string(raw), "\r\n", "\n")
			if strings.Contains(source, "sentinel-volume-init") {
				t.Fatal("root-owned sentinel volume init service is forbidden")
			}
			for _, service := range contract.services {
				section := composeServiceSection(t, source, service)
				for _, marker := range []string{
					`user: "65532:4"`,
					"cap_drop:\n      - ALL",
					"security_opt:\n      - no-new-privileges:true",
					"read_only: true",
					"/tmp:rw,noexec,nosuid,nodev,mode=1777",
				} {
					if !strings.Contains(section, marker) {
						t.Fatalf("service %s missing rootless marker %q", service, marker)
					}
				}
			}
			for _, volume := range contract.volumes {
				if !strings.Contains(source, volume) {
					t.Fatalf("missing fresh rootless volume %q", volume)
				}
			}
		})
	}
}

func TestAdaptiveVolumeMountpointsSeedSentinelOwnership(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "control-plane", "Dockerfile"),
		filepath.Join("..", "..", "runtime", "image", "Dockerfile"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		source := string(raw)
		for _, marker := range []string{
			"/etc/waf/l4guard-adaptive",
			"chown 65532:4 /etc/waf/l4guard-adaptive",
			"chmod 0770 /etc/waf/l4guard-adaptive",
		} {
			if !strings.Contains(source, marker) {
				t.Fatalf("%s missing adaptive volume ownership marker %q", path, marker)
			}
		}
	}
}

func composeServiceSection(t *testing.T, source, service string) string {
	t.Helper()
	startMarker := "\n  " + service + ":\n"
	start := strings.Index(source, startMarker)
	if start < 0 {
		t.Fatalf("service %s is missing", service)
	}
	rest := source[start+len(startMarker):]
	for index := 0; index+3 < len(rest); index++ {
		if rest[index] == '\n' && rest[index+1] == ' ' && rest[index+2] == ' ' && rest[index+3] != ' ' && rest[index+3] != '\n' {
			return rest[:index]
		}
	}
	return rest
}
