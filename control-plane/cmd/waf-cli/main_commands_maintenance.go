package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type maintenanceCommand struct {
	label       string
	description string
	run         func(*cli, *maintenanceSession) error
}

type maintenanceSession struct {
	reader            *bufio.Reader
	readOnly          bool
	defaultSiteID     string
	usernameConfirmed bool
	passwordConfirmed bool
}

func cmdMaintenance(c *cli, args []string, noAuth bool) error {
	if noAuth {
		return fmt.Errorf("maintenance toolkit requires auth; remove --no-auth")
	}
	if len(args) > 0 {
		return fmt.Errorf("usage: waf-cli maintenance")
	}
	session := newMaintenanceSession(defaultMaintenanceSiteID())
	return runMaintenanceMenu(c, session)
}

func newMaintenanceSession(defaultSiteID string) *maintenanceSession {
	return &maintenanceSession{
		reader:        bufio.NewReader(os.Stdin),
		readOnly:      true,
		defaultSiteID: normalizeSiteID(defaultSiteID),
	}
}

func runMaintenanceMenu(c *cli, session *maintenanceSession) error {
	commands := maintenanceCommands()
	for {
		printMaintenanceMenu(commands, session)
		choice, err := promptMaintenanceChoice(session, len(commands))
		if err != nil {
			return err
		}
		if choice == 0 {
			fmt.Println("Maintenance toolkit exited.")
			return nil
		}
		if choice == -1 {
			session.readOnly = !session.readOnly
			fmt.Printf("Mode switched: %s\n", maintenanceModeLabel(session.readOnly))
			continue
		}
		cmd := commands[choice-1]
		fmt.Printf("\n== %s ==\n", cmd.label)
		if err := ensureMaintenanceAuth(c, session); err != nil {
			return err
		}
		if err := cmd.run(c, session); err != nil {
			fmt.Printf("[FAIL] %v\n", err)
		} else {
			fmt.Println("[OK] Done.")
		}
		fmt.Println()
	}
}

func maintenanceCommands() []maintenanceCommand {
	return []maintenanceCommand{
		{
			label:       "Stack audit",
			description: "Compose ps, image tags, disk/memory headroom, mounted env metadata",
			run:         runMaintenanceStackAudit,
		},
		{
			label:       "Health probes",
			description: "Control-plane healthz, auth/me, setup status, revisions list",
			run:         runMaintenanceHealthProbes,
		},
		{
			label:       "Logs / diagnostics",
			description: "Show container log commands and quick runtime diagnostics",
			run:         runMaintenanceLogs,
		},
		{
			label:       "Placeholder-secret checks",
			description: "Scan env files for placeholder/default secret markers without printing values",
			run:         runMaintenancePlaceholderSecrets,
		},
		{
			label:       "Postgres auth consistency",
			description: "Compare .env / compose refs and verify postgres readiness without secret rotation",
			run:         runMaintenancePostgresAuth,
		},
		{
			label:       "Backup config snapshot",
			description: "Create redacted compose/env/revision snapshot manifest",
			run:         runMaintenanceBackupSnapshot,
		},
		{
			label:       "Revision operations",
			description: "List/current/apply flows with post-apply verification",
			run:         runMaintenanceRevisions,
		},
		{
			label:       "Ban / unban",
			description: "Guided IP ban, unban, and bans list flow with post-check",
			run:         runMaintenanceBans,
		},
		{
			label:       "Emergency mode switch",
			description: "Guided safe toggles with mandatory preflight snapshot",
			run:         runMaintenanceEmergency,
		},
	}
}

func printMaintenanceMenu(commands []maintenanceCommand, session *maintenanceSession) {
	fmt.Println("TARINIO maintenance toolkit")
	fmt.Printf("Mode: %s\n", maintenanceModeLabel(session.readOnly))
	fmt.Println("0) Exit")
	fmt.Println("m) Toggle read-only / destructive mode")
	for i, cmd := range commands {
		fmt.Printf("%d) %s — %s\n", i+1, cmd.label, cmd.description)
	}
	fmt.Print("Select action: ")
}

func promptMaintenanceChoice(session *maintenanceSession, max int) (int, error) {
	text, err := session.readLine()
	if err != nil {
		return 0, err
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "0", "q", "quit", "exit":
		return 0, nil
	case "m", "mode":
		return -1, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 1 || value > max {
		return 0, fmt.Errorf("invalid choice %q", strings.TrimSpace(text))
	}
	return value, nil
}

func ensureMaintenanceAuth(c *cli, session *maintenanceSession) error {
	if session.usernameConfirmed && session.passwordConfirmed && c.loggedIn {
		return nil
	}
	if strings.TrimSpace(c.username) == "" {
		username, err := promptLine(session, "Username", defaultCLIUsername())
		if err != nil {
			return err
		}
		c.username = username
	}
	if strings.TrimSpace(c.password) == "" {
		password, err := promptLine(session, "Password", "")
		if err != nil {
			return err
		}
		c.password = password
	}
	if err := c.ensureLogin(); err != nil {
		return err
	}
	session.usernameConfirmed = true
	session.passwordConfirmed = true
	return maybePersistMaintenanceCredentials(session, c)
}

func maybePersistMaintenanceCredentials(session *maintenanceSession, c *cli) error {
	answer, err := promptChoice(
		session,
		"Save credentials to current shell env for this session?",
		[]string{"no", "yes"},
		"no",
	)
	if err != nil {
		return err
	}
	if answer != "yes" {
		return nil
	}
	fmt.Println("Export these in your shell if desired:")
	if strings.TrimSpace(c.username) != "" {
		fmt.Printf("  export WAF_CLI_USERNAME=%s\n", shellQuoteSingle(c.username))
	}
	if strings.TrimSpace(c.password) != "" {
		fmt.Printf("  export WAF_CLI_PASSWORD=%s\n", shellQuoteSingle(c.password))
	}
	return nil
}

func renderMaintenanceSummary(path string, value any) error {
	item, ok := value.(map[string]any)
	if ok {
		keys := make([]string, 0, len(item))
		for key := range item {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Printf("%s -> object keys: %s\n", path, strings.Join(keys, ", "))
		return nil
	}
	list := asList(value)
	if list != nil {
		fmt.Printf("%s -> %d items\n", path, len(list))
		return nil
	}
	return printJSON(value)
}

func (s *maintenanceSession) readLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptLine(session *maintenanceSession, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, err := session.readLine()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return strings.TrimSpace(defaultValue), nil
	}
	return strings.TrimSpace(text), nil
}

func promptChoice(session *maintenanceSession, label string, options []string, defaultValue string) (string, error) {
	fmt.Printf("%s [%s]: ", label, strings.Join(options, "/"))
	text, err := session.readLine()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return defaultValue, nil
	}
	value := strings.ToLower(strings.TrimSpace(text))
	for _, option := range options {
		if value == strings.ToLower(option) {
			return option, nil
		}
	}
	return "", fmt.Errorf("unsupported choice %q", value)
}

func maintenanceModeLabel(readOnly bool) string {
	if readOnly {
		return "read-only / dry-run"
	}
	return "destructive actions enabled"
}

func defaultMaintenanceSiteID() string {
	return envOrDefault(
		"WAF_CLI_DEFAULT_SITE",
		envOrDefault("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access"),
	)
}

func shellQuoteSingle(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
