package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConnLimit    = 50
	defaultRatePerSec   = 30
	defaultRateBurst    = 60
	defaultPorts        = "80,443"
	customChainName     = "WAF-RUNTIME-L4"
	defaultChainMode    = "auto"
	defaultTarget       = "DROP"
	defaultEnabled      = true
	defaultConfigPath   = "/etc/waf/l4guard/config.json"
	defaultAdaptivePath = "/etc/waf/l4guard-adaptive/adaptive.json"
	iptablesIPv4        = "iptables"
	iptablesIPv6        = "ip6tables"
)

type config struct {
	Enabled       bool
	ChainMode     string
	ConnLimit     int
	RatePerSec    int
	RateBurst     int
	Ports         []int
	Target        string
	DestinationIP string
	Adaptive      adaptiveConfig
}

type fileConfig struct {
	Enabled       *bool  `json:"enabled"`
	ChainMode     string `json:"chain_mode"`
	ConnLimit     int    `json:"conn_limit"`
	RatePerSec    int    `json:"rate_per_second"`
	RateBurst     int    `json:"rate_burst"`
	Ports         []int  `json:"ports"`
	Target        string `json:"target"`
	DestinationIP string `json:"destination_ip"`
}

type adaptiveEntry struct {
	IP        string `json:"ip"`
	Action    string `json:"action"`
	ExpiresAt string `json:"expires_at"`
}

type adaptiveFileConfig struct {
	Entries               []adaptiveEntry `json:"entries"`
	ThrottleRatePerSecond int             `json:"throttle_rate_per_second"`
	ThrottleBurst         int             `json:"throttle_burst"`
	ThrottleTarget        string          `json:"throttle_target"`
}

type adaptiveConfig struct {
	Entries               []adaptiveEntry
	ThrottleRatePerSecond int
	ThrottleBurst         int
	ThrottleTarget        string
}

type executor interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
}

type osExecutor struct{}

func (osExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func (osExecutor) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = io.Discard
	return cmd.Output()
}

func main() {
	if err := run(osExecutor{}, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "waf-runtime-l4-guard: %v\n", err)
		os.Exit(1)
	}
}

func run(exec executor, args []string) error {
	command := "bootstrap"
	if len(args) > 0 {
		command = strings.TrimSpace(args[0])
	}
	if command != "bootstrap" {
		return fmt.Errorf("unsupported command %q", command)
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return nil
	}
	return bootstrap(exec, cfg)
}

func loadConfig() (config, error) {
	cfg := config{
		Enabled:    defaultEnabled,
		ChainMode:  defaultChainMode,
		ConnLimit:  defaultConnLimit,
		RatePerSec: defaultRatePerSec,
		RateBurst:  defaultRateBurst,
		Target:     defaultTarget,
		Adaptive: adaptiveConfig{
			ThrottleRatePerSecond: 5,
			ThrottleBurst:         10,
			ThrottleTarget:        "REJECT",
		},
	}
	if err := applyFileConfig(&cfg); err != nil {
		return config{}, err
	}
	if err := applyAdaptiveFileConfig(&cfg); err != nil {
		return config{}, err
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_ENABLED")); value != "" {
		cfg.Enabled = !strings.EqualFold(value, "false") && value != "0"
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_CHAIN_MODE")); value != "" {
		cfg.ChainMode = strings.ToLower(value)
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_CONN_LIMIT")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number <= 0 {
			return config{}, errors.New("WAF_L4_GUARD_CONN_LIMIT must be a positive integer")
		}
		cfg.ConnLimit = number
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_RATE_PER_SECOND")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number <= 0 {
			return config{}, errors.New("WAF_L4_GUARD_RATE_PER_SECOND must be a positive integer")
		}
		cfg.RatePerSec = number
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_RATE_BURST")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number <= 0 {
			return config{}, errors.New("WAF_L4_GUARD_RATE_BURST must be a positive integer")
		}
		cfg.RateBurst = number
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_TARGET")); value != "" {
		cfg.Target = strings.ToUpper(value)
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_DESTINATION_IP")); value != "" {
		ip := net.ParseIP(value)
		if ip == nil {
			return config{}, errors.New("WAF_L4_GUARD_DESTINATION_IP must be a valid IP address")
		}
		cfg.DestinationIP = ip.String()
	}
	if value := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_PORTS")); value != "" {
		ports, err := parsePorts(value)
		if err != nil {
			return config{}, err
		}
		cfg.Ports = ports
	} else if len(cfg.Ports) == 0 {
		ports, err := parsePorts(defaultPorts)
		if err != nil {
			return config{}, err
		}
		cfg.Ports = ports
	}
	if cfg.ChainMode != "auto" && cfg.ChainMode != "docker-user" && cfg.ChainMode != "input" {
		return config{}, errors.New("WAF_L4_GUARD_CHAIN_MODE must be auto, docker-user, or input")
	}
	if cfg.Target != "DROP" && cfg.Target != "REJECT" {
		return config{}, errors.New("WAF_L4_GUARD_TARGET must be DROP or REJECT")
	}
	if cfg.Adaptive.ThrottleRatePerSecond <= 0 {
		cfg.Adaptive.ThrottleRatePerSecond = 5
	}
	if cfg.Adaptive.ThrottleBurst <= 0 {
		cfg.Adaptive.ThrottleBurst = cfg.Adaptive.ThrottleRatePerSecond * 2
	}
	if cfg.Adaptive.ThrottleTarget != "DROP" && cfg.Adaptive.ThrottleTarget != "REJECT" {
		cfg.Adaptive.ThrottleTarget = "REJECT"
	}
	return cfg, nil
}

func applyFileConfig(cfg *config) error {
	path := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_CONFIG_PATH"))
	if path == "" {
		path = defaultConfigPath
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read l4 guard config file: %w", err)
	}
	var item fileConfig
	if err := json.Unmarshal(content, &item); err != nil {
		return fmt.Errorf("decode l4 guard config file: %w", err)
	}
	if item.Enabled != nil {
		cfg.Enabled = *item.Enabled
	}
	if strings.TrimSpace(item.ChainMode) != "" {
		cfg.ChainMode = strings.ToLower(strings.TrimSpace(item.ChainMode))
	}
	if item.ConnLimit > 0 {
		cfg.ConnLimit = item.ConnLimit
	}
	if item.RatePerSec > 0 {
		cfg.RatePerSec = item.RatePerSec
	}
	if item.RateBurst > 0 {
		cfg.RateBurst = item.RateBurst
	}
	if len(item.Ports) > 0 {
		ports := append([]int(nil), item.Ports...)
		sort.Ints(ports)
		cfg.Ports = ports
	}
	if strings.TrimSpace(item.Target) != "" {
		cfg.Target = strings.ToUpper(strings.TrimSpace(item.Target))
	}
	if strings.TrimSpace(item.DestinationIP) != "" {
		cfg.DestinationIP = strings.TrimSpace(item.DestinationIP)
	}
	return nil
}

func applyAdaptiveFileConfig(cfg *config) error {
	path := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_ADAPTIVE_PATH"))
	if path == "" {
		path = defaultAdaptivePath
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read adaptive l4 guard config file: %w", err)
	}
	var item adaptiveFileConfig
	if err := json.Unmarshal(content, &item); err != nil {
		return fmt.Errorf("decode adaptive l4 guard config file: %w", err)
	}
	if item.ThrottleRatePerSecond > 0 {
		cfg.Adaptive.ThrottleRatePerSecond = item.ThrottleRatePerSecond
	}
	if item.ThrottleBurst > 0 {
		cfg.Adaptive.ThrottleBurst = item.ThrottleBurst
	}
	if strings.TrimSpace(item.ThrottleTarget) != "" {
		cfg.Adaptive.ThrottleTarget = strings.ToUpper(strings.TrimSpace(item.ThrottleTarget))
	}

	now := time.Now().UTC()
	filtered := make([]adaptiveEntry, 0, len(item.Entries))
	seen := make(map[string]struct{}, len(item.Entries))
	for _, entry := range item.Entries {
		ip := strings.TrimSpace(entry.IP)
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			continue
		}
		action := strings.ToLower(strings.TrimSpace(entry.Action))
		if action != "throttle" && action != "drop" {
			continue
		}
		expiresAt := strings.TrimSpace(entry.ExpiresAt)
		if expiresAt != "" {
			if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil && parsed.Before(now) {
				continue
			}
		}
		key := ip + "|" + action
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, adaptiveEntry{
			IP:        ip,
			Action:    action,
			ExpiresAt: expiresAt,
		})
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Action == filtered[j].Action {
			return filtered[i].IP < filtered[j].IP
		}
		return filtered[i].Action < filtered[j].Action
	})
	cfg.Adaptive.Entries = filtered
	return nil
}

func bootstrap(exec executor, cfg config) error {
	var errs []string
	if err := bootstrapFamily(exec, cfg, iptablesIPv4, false); err != nil {
		errs = append(errs, err.Error())
	}
	if err := bootstrapFamily(exec, cfg, iptablesIPv6, true); err != nil && !errors.Is(err, errFamilyNotConfigured) {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

var errFamilyNotConfigured = errors.New("ip family not configured")

func bootstrapFamily(exec executor, cfg config, table string, ipv6 bool) error {
	parentChain, destinationIP, err := resolveParentChain(exec, cfg, table, ipv6)
	if err != nil {
		return err
	}
	if parentChain == "" {
		return errFamilyNotConfigured
	}
	if err := ensureChain(exec, table, customChainName); err != nil {
		return err
	}
	if err := flushChain(exec, table, customChainName); err != nil {
		return err
	}
	for _, rule := range adaptiveRules(cfg, ipv6) {
		if err := exec.Run(table, append([]string{"-w", "-A", customChainName}, rule...)...); err != nil {
			return fmt.Errorf("append adaptive guard rule (%s): %w", table, err)
		}
	}
	for _, rule := range guardRules(cfg.Target, cfg.ConnLimit, cfg.RatePerSec, cfg.RateBurst, ipv6) {
		if err := exec.Run(table, append([]string{"-w", "-A", customChainName}, rule...)...); err != nil {
			return fmt.Errorf("append guard rule (%s): %w", table, err)
		}
	}
	jumpRule := parentJumpRule(parentChain, destinationIP, cfg.Ports)
	if err := deleteJumpRule(exec, table, parentChain, jumpRule); err != nil {
		return err
	}
	if err := exec.Run(table, append([]string{"-w", "-I", parentChain, "1"}, jumpRule...)...); err != nil {
		return fmt.Errorf("insert jump rule into %s via %s: %w", parentChain, table, err)
	}
	return nil
}

func adaptiveRules(cfg config, ipv6 bool) [][]string {
	if len(cfg.Adaptive.Entries) == 0 {
		return nil
	}
	rules := make([][]string, 0, len(cfg.Adaptive.Entries))
	for _, entry := range cfg.Adaptive.Entries {
		parsedIP := net.ParseIP(entry.IP)
		if parsedIP == nil {
			continue
		}
		if ipv6 != (parsedIP.To4() == nil) {
			continue
		}
		switch entry.Action {
		case "drop":
			rules = append(rules, []string{"-p", "tcp", "-s", entry.IP, "-j", cfg.Target})
		case "throttle":
			name := adaptiveHashlimitName(entry.IP)
			rules = append(rules, []string{
				"-p", "tcp",
				"-s", entry.IP,
				"-m", "conntrack",
				"--ctstate", "NEW",
				"-m", "hashlimit",
				"--hashlimit-name", name,
				"--hashlimit-mode", "srcip",
				"--hashlimit-above", fmt.Sprintf("%d/second", cfg.Adaptive.ThrottleRatePerSecond),
				"--hashlimit-burst", strconv.Itoa(cfg.Adaptive.ThrottleBurst),
				"--hashlimit-htable-expire", "600000",
				"-j", cfg.Adaptive.ThrottleTarget,
			})
		}
	}
	return rules
}

func adaptiveHashlimitName(ip string) string {
	sum := sha1.Sum([]byte(ip))
	return "waf-l4-ad-" + fmt.Sprintf("%x", sum[:4])
}

func resolveParentChain(exec executor, cfg config, table string, ipv6 bool) (string, string, error) {
	switch cfg.ChainMode {
	case "docker-user":
		ip, err := resolveDestinationIP(cfg.DestinationIP, ipv6)
		if err != nil {
			if errors.Is(err, errFamilyNotConfigured) {
				return "", "", errFamilyNotConfigured
			}
			return "", "", err
		}
		if !chainExists(exec, table, "DOCKER-USER") {
			return "", "", errors.New("DOCKER-USER chain not found in current namespace")
		}
		return "DOCKER-USER", ip, nil
	case "input":
		return "INPUT", "", nil
	default:
		if chainExists(exec, table, "DOCKER-USER") {
			ip, err := resolveDestinationIP(cfg.DestinationIP, ipv6)
			if err == nil && ip != "" {
				return "DOCKER-USER", ip, nil
			}
		}
		return "INPUT", "", nil
	}
}

func resolveDestinationIP(override string, ipv6 bool) (string, error) {
	if value := strings.TrimSpace(override); value != "" {
		ip := net.ParseIP(value)
		if ip == nil {
			return "", errors.New("invalid destination ip")
		}
		if ipv6 {
			if ip.To4() != nil {
				return "", errFamilyNotConfigured
			}
			return ip.String(), nil
		}
		if ip.To4() == nil {
			return "", errFamilyNotConfigured
		}
		return ip.To4().String(), nil
	}
	if ipv6 {
		ip, err := primaryIPv6()
		if err != nil {
			return "", errFamilyNotConfigured
		}
		return ip, nil
	}
	return primaryIPv4()
}

func chainExists(exec executor, table string, chain string) bool {
	_, err := exec.Output(table, "-w", "-S", chain)
	return err == nil
}

func ensureChain(exec executor, table string, chain string) error {
	if chainExists(exec, table, chain) {
		return nil
	}
	if err := exec.Run(table, "-w", "-N", chain); err != nil {
		return fmt.Errorf("create chain %s: %w", chain, err)
	}
	return nil
}

func flushChain(exec executor, table string, chain string) error {
	if err := exec.Run(table, "-w", "-F", chain); err != nil {
		return fmt.Errorf("flush chain %s: %w", chain, err)
	}
	return nil
}

func deleteJumpRule(exec executor, table string, parentChain string, rule []string) error {
	checkArgs := append([]string{"-w", "-C", parentChain}, rule...)
	deleteArgs := append([]string{"-w", "-D", parentChain}, rule...)
	for {
		if err := exec.Run(table, checkArgs...); err != nil {
			return nil
		}
		if err := exec.Run(table, deleteArgs...); err != nil {
			return fmt.Errorf("delete stale jump rule from %s: %w", parentChain, err)
		}
	}
}

func parentJumpRule(parentChain, destinationIP string, ports []int) []string {
	rule := []string{"-p", "tcp"}
	if parentChain == "DOCKER-USER" && destinationIP != "" {
		rule = append(rule, "-d", destinationIP)
	}
	rule = append(rule, "-m", "multiport", "--dports", joinPorts(ports), "-j", customChainName)
	return rule
}

func guardRules(target string, connLimit int, ratePerSec int, rateBurst int, ipv6 bool) [][]string {
	connlimitMask := "32"
	if ipv6 {
		connlimitMask = "128"
	}
	return [][]string{
		{
			"-p", "tcp",
			"-m", "connlimit",
			"--connlimit-mask", connlimitMask,
			"--connlimit-above", strconv.Itoa(connLimit),
			"-j", target,
		},
		{
			"-p", "tcp",
			"-m", "conntrack",
			"--ctstate", "NEW",
			"-m", "hashlimit",
			"--hashlimit-name", "waf-runtime-l4-rate",
			"--hashlimit-mode", "srcip",
			"--hashlimit-above", fmt.Sprintf("%d/second", ratePerSec),
			"--hashlimit-burst", strconv.Itoa(rateBurst),
			"--hashlimit-htable-expire", "600000",
			"-j", target,
		},
		{"-j", "RETURN"},
	}
}

func primaryIPv4() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
				continue
			}
			if ip := ipNet.IP.To4(); ip != nil {
				return ip.String(), nil
			}
		}
	}
	return "", errors.New("no non-loopback ipv4 address found")
}

func primaryIPv6() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP
			if ip.To4() != nil || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("no non-loopback ipv6 address found")
}

func parsePorts(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	ports := make([]int, 0, len(parts))
	seen := map[int]struct{}{}
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 || port > 65535 {
			return nil, fmt.Errorf("invalid L4 guard port %q", value)
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		ports = append(ports, port)
	}
	sort.Ints(ports)
	if len(ports) == 0 {
		return nil, errors.New("at least one L4 guard port is required")
	}
	return ports, nil
}

func joinPorts(ports []int) string {
	parts := make([]string, 0, len(ports))
	for _, port := range ports {
		parts = append(parts, strconv.Itoa(port))
	}
	return strings.Join(parts, ",")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
