package tests

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func nodeESMCommand(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	command := "node"
	if runtime.GOOS == "windows" {
		command = "node.exe"
	}
	version, err := exec.Command(command, "--version").Output()
	if err != nil {
		t.Skipf("%s is required for frontend contract tests", command)
	}
	majorText := strings.TrimPrefix(strings.TrimSpace(string(version)), "v")
	majorText = strings.SplitN(majorText, ".", 2)[0]
	major, err := strconv.Atoi(majorText)
	if err != nil {
		t.Fatalf("parse Node.js version %q: %v", version, err)
	}
	args := []string{"--input-type=module", "--eval", script}
	if major < 20 {
		args = append([]string{"--experimental-default-type=module"}, args...)
	}
	return exec.Command(command, args...)
}
