package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeExecutor struct {
	chains       map[string]bool
	runCommands  [][]string
	outputErrors map[string]error
	rules        map[string]bool
	failDelete   map[string]int
}

func (f *fakeExecutor) Run(name string, args ...string) error {
	command := append([]string{name}, args...)
	f.runCommands = append(f.runCommands, command)
	if f.rules == nil {
		f.rules = map[string]bool{}
	}
	if len(args) >= 3 && args[1] == "-N" {
		f.chains[args[2]] = true
		return nil
	}
	if len(args) >= 3 && args[1] == "-I" {
		f.rules[strings.Join(command, " ")] = true
		return nil
	}
	if len(args) >= 2 && args[1] == "-C" {
		inserted := append([]string{name, args[0], "-I", args[2]}, args[3:]...)
		if !f.rules[strings.Join(inserted, " ")] {
			return errors.New("rule not found")
		}
		return nil
	}
	if len(args) >= 2 && args[1] == "-D" {
		key := strings.Join(command, " ")
		if f.failDelete != nil {
			if remain, ok := f.failDelete[key]; ok && remain > 0 {
				f.failDelete[key] = remain - 1
				return errors.New("simulated delete race")
			}
		}
		inserted := append([]string{name, args[0], "-I", args[2]}, args[3:]...)
		delete(f.rules, strings.Join(inserted, " "))
		return nil
	}
	return nil
}

func (f *fakeExecutor) Output(name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	if err := f.outputErrors[key]; err != nil {
		return nil, err
	}
	if len(args) == 3 && args[1] == "-S" && f.chains[args[2]] {
		return []byte("-N " + args[2]), nil
	}
	return nil, errors.New("not found")
}

func TestParsePorts_SortsAndDeduplicates(t *testing.T) {
	ports, err := parsePorts("443,80,443")
	if err != nil {
		t.Fatalf("parse ports: %v", err)
	}
	if !reflect.DeepEqual(ports, []int{80, 443}) {
		t.Fatalf("unexpected ports: %+v", ports)
	}
}

func TestBootstrap_AutoFallsBackToInputWhenDockerUserMissing(t *testing.T) {
	exec := &fakeExecutor{chains: map[string]bool{"INPUT": true, customChainName: true}, outputErrors: map[string]error{}}
	cfg := config{Enabled: true, ChainMode: "auto", ConnLimit: 50, RatePerSec: 30, RateBurst: 60, Ports: []int{80, 443}, Target: "DROP"}

	if err := bootstrap(exec, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	found := false
	for _, command := range exec.runCommands {
		if strings.Join(command, " ") == "iptables -w -I INPUT 1 -p tcp -m multiport --dports 80,443 -j WAF-RUNTIME-L4" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected INPUT jump rule, got %+v", exec.runCommands)
	}
}

func TestBootstrap_DockerUserUsesDestinationIP(t *testing.T) {
	exec := &fakeExecutor{chains: map[string]bool{"DOCKER-USER": true, customChainName: true}, outputErrors: map[string]error{}}
	cfg := config{
		Enabled:       true,
		ChainMode:     "docker-user",
		ConnLimit:     50,
		RatePerSec:    30,
		RateBurst:     60,
		Ports:         []int{80, 443},
		Target:        "DROP",
		DestinationIP: "172.18.0.10",
	}

	if err := bootstrap(exec, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	found := false
	for _, command := range exec.runCommands {
		if strings.Join(command, " ") == "iptables -w -I DOCKER-USER 1 -p tcp -d 172.18.0.10 -m multiport --dports 80,443 -j WAF-RUNTIME-L4" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DOCKER-USER jump rule with destination IP, got %+v", exec.runCommands)
	}
}

func TestDeleteJumpRule_ToleratesConcurrentRemovalRace(t *testing.T) {
	exec := &fakeExecutor{
		chains:       map[string]bool{"INPUT": true, customChainName: true},
		outputErrors: map[string]error{},
		rules:        map[string]bool{},
		failDelete:   map[string]int{},
	}
	rule := []string{"-p", "tcp", "-m", "multiport", "--dports", "80,443", "-j", customChainName}
	inserted := append([]string{iptablesIPv4, "-w", "-I", "INPUT", "1"}, rule...)
	exec.rules[strings.Join(inserted, " ")] = true
	deleteCmd := append([]string{iptablesIPv4, "-w", "-D", "INPUT"}, rule...)
	exec.failDelete[strings.Join(deleteCmd, " ")] = 1

	if err := deleteJumpRule(exec, iptablesIPv4, "INPUT", rule); err != nil {
		t.Fatalf("deleteJumpRule should tolerate race: %v", err)
	}
}

func TestGuardRules_ContainConnlimitAndHashlimit(t *testing.T) {
	rules := guardRules("DROP", 40, 25, 50, false)
	if len(rules) != 3 {
		t.Fatalf("unexpected rule count: %d", len(rules))
	}
	if !strings.Contains(strings.Join(rules[0], " "), "--connlimit-above 40") {
		t.Fatalf("missing connlimit rule: %+v", rules[0])
	}
	if !strings.Contains(strings.Join(rules[1], " "), "--hashlimit-above 25/second") {
		t.Fatalf("missing hashlimit rule: %+v", rules[1])
	}
}

func TestGuardRules_IPv6Uses128Mask(t *testing.T) {
	rules := guardRules("DROP", 40, 25, 50, true)
	if !strings.Contains(strings.Join(rules[0], " "), "--connlimit-mask 128") {
		t.Fatalf("expected ipv6 connlimit mask, got %+v", rules[0])
	}
}

func TestLoadConfig_ParsesEnv(t *testing.T) {
	t.Setenv("WAF_L4_GUARD_CHAIN_MODE", "input")
	t.Setenv("WAF_L4_GUARD_CONN_LIMIT", "75")
	t.Setenv("WAF_L4_GUARD_RATE_PER_SECOND", "20")
	t.Setenv("WAF_L4_GUARD_RATE_BURST", "40")
	t.Setenv("WAF_L4_GUARD_PORTS", "8443,8080")
	t.Setenv("WAF_L4_GUARD_TARGET", "REJECT")
	t.Setenv("WAF_L4_GUARD_DESTINATION_IP", "172.18.0.10")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ChainMode != "input" || cfg.ConnLimit != 75 || cfg.RatePerSec != 20 || cfg.RateBurst != 40 || cfg.Target != "REJECT" || cfg.DestinationIP != "172.18.0.10" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443}) {
		t.Fatalf("unexpected ports: %+v", cfg.Ports)
	}
}

func TestLoadConfig_InvalidPortFails(t *testing.T) {
	t.Setenv("WAF_L4_GUARD_PORTS", "80,abc")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadConfig_ReadsConfigFileWhenPresent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "l4guard.json")
	if err := os.WriteFile(path, []byte(`{
  "enabled": true,
  "chain_mode": "input",
  "conn_limit": 222,
  "rate_per_second": 111,
  "rate_burst": 333,
  "ports": [443, 8443],
  "target": "REJECT"
}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	t.Setenv("WAF_L4_GUARD_CONFIG_PATH", path)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ChainMode != "input" || cfg.ConnLimit != 222 || cfg.RatePerSec != 111 || cfg.RateBurst != 333 || cfg.Target != "REJECT" {
		t.Fatalf("unexpected config from file: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{443, 8443}) {
		t.Fatalf("unexpected file ports: %+v", cfg.Ports)
	}
}

func TestLoadConfig_EnvOverridesFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "l4guard.json")
	if err := os.WriteFile(path, []byte(`{
  "enabled": true,
  "chain_mode": "input",
  "conn_limit": 222,
  "rate_per_second": 111,
  "rate_burst": 333,
  "ports": [443],
  "target": "REJECT"
}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	t.Setenv("WAF_L4_GUARD_CONFIG_PATH", path)
	t.Setenv("WAF_L4_GUARD_CONN_LIMIT", "300")
	t.Setenv("WAF_L4_GUARD_PORTS", "80,443")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ConnLimit != 300 {
		t.Fatalf("expected env override for conn limit, got %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{80, 443}) {
		t.Fatalf("expected env override ports, got %+v", cfg.Ports)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", " ", "a"); got != "a" {
		t.Fatalf("unexpected value %q", got)
	}
}

func TestMainDoesNotReadUnexpectedEnv(t *testing.T) {
	for _, key := range []string{
		"WAF_L4_GUARD_CHAIN_MODE",
		"WAF_L4_GUARD_CONN_LIMIT",
		"WAF_L4_GUARD_RATE_PER_SECOND",
		"WAF_L4_GUARD_RATE_BURST",
		"WAF_L4_GUARD_PORTS",
		"WAF_L4_GUARD_TARGET",
		"WAF_L4_GUARD_DESTINATION_IP",
	} {
		if _, ok := os.LookupEnv(key); ok {
			t.Fatalf("unexpected inherited env %s", key)
		}
	}
}
