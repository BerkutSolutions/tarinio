//go:build e2e

package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EUIRootlessImageReadsRestrictiveContextConfigs(t *testing.T) {
	composeDir, err := filepath.Abs(filepath.Join("..", "..", "deploy", "compose", "e2e"))
	if err != nil {
		t.Fatalf("resolve E2E Compose directory: %v", err)
	}
	containerID := runUIRootlessCommand(t, composeDir, "docker", "compose", "-f", "docker-compose.yml", "ps", "-q", "ui")
	if containerID == "" {
		t.Fatal("E2E UI container is missing")
	}

	state := runUIRootlessCommand(t, composeDir, "docker", "inspect", "--format", "{{.State.Status}}|{{.Config.User}}", containerID)
	if state != "running|101:101" {
		t.Fatalf("UI container must run rootless as 101:101, got %q", state)
	}

	modes := runUIRootlessCommand(t, composeDir, "docker", "exec", containerID, "stat", "-c", "%a|%u|%g|%n",
		"/etc/nginx/nginx.conf", "/etc/nginx/conf.d/default.conf")
	for _, configPath := range []string{"/etc/nginx/nginx.conf", "/etc/nginx/conf.d/default.conf"} {
		expected := "644|0|0|" + configPath
		if !strings.Contains(modes, expected) {
			t.Fatalf("rootless-readable nginx config %q is absent from container modes:\n%s", expected, modes)
		}
	}

	runUIRootlessCommand(t, composeDir, "docker", "exec", containerID, "nginx", "-t")
}

func runUIRootlessCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
