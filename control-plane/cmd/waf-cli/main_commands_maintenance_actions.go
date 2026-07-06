package main

import (
	"fmt"
	"strings"
)

func runMaintenanceStackAudit(c *cli, _ *maintenanceSession) error {
	fmt.Println("Expected remote Linux VM commands:")
	for _, cmd := range maintenanceStackAuditCommands() {
		fmt.Printf("  %s\n", cmd)
	}
	fmt.Println("CLI integration checks:")
	if err := runMaintenanceAPICommand(c, httpMethodGet, "/api/sites", nil); err != nil {
		return err
	}
	if err := runMaintenanceAPICommand(c, httpMethodGet, "/api/upstreams", nil); err != nil {
		return err
	}
	return runMaintenanceRevisionsList(c)
}

func runMaintenanceHealthProbes(c *cli, session *maintenanceSession) error {
	fmt.Println("Control-plane probes:")
	if err := cmdHealth(c); err != nil {
		return err
	}
	if err := cmdAuthMe(c, false); err != nil {
		return err
	}
	if err := cmdSetup(c, []string{"status"}, false); err != nil {
		return err
	}
	fmt.Println("Revision/status probes:")
	return runMaintenanceRevisionsList(c)
}

func runMaintenanceLogs(c *cli, _ *maintenanceSession) error {
	fmt.Println("Scoped log commands for remote Linux VM:")
	for _, cmd := range maintenanceLogCommands() {
		fmt.Printf("  %s\n", cmd)
	}
	fmt.Println("Recent security events via CLI:")
	return cmdEvents(c, []string{"--limit", "10"}, false)
}

func runMaintenancePlaceholderSecrets(_ *cli, _ *maintenanceSession) error {
	paths := []string{
		"/opt/tarinio/deploy/compose/default/.env",
		"/opt/tarinio/deploy/compose/enterprise/.env",
	}
	fmt.Println("Placeholder-secret check commands (values never printed):")
	for _, path := range paths {
		fmt.Printf("  awk -F= 'BEGIN{IGNORECASE=1} /^[A-Z0-9_]+=/{v=$2; gsub(/^[[:space:]]+|[[:space:]]+$/,\"\",v); if (v ~ /^(|change-me.*|changeme|default|please-change.*|todo|secret|password|replace-me.*)$/) print FILENAME\":\"$1}' %s\n", path)
	}
	return nil
}

func runMaintenancePostgresAuth(c *cli, _ *maintenanceSession) error {
	fmt.Println("Postgres consistency commands for remote VM:")
	for _, cmd := range maintenancePostgresCommands() {
		fmt.Printf("  %s\n", cmd)
	}
	return runMaintenanceAPICommand(c, httpMethodGet, "/api/setup/status", nil)
}

func runMaintenanceBackupSnapshot(_ *cli, session *maintenanceSession) error {
	if session.readOnly {
		fmt.Println("Read-only mode: printing snapshot commands only.")
		for _, cmd := range maintenanceBackupCommands() {
			fmt.Printf("  %s\n", cmd)
		}
		return nil
	}
	name, err := promptLine(session, "Snapshot name", "maintenance-snapshot")
	if err != nil {
		return err
	}
	fmt.Println("Create remote snapshot with commands:")
	for _, cmd := range maintenanceBackupCommandsWithName(name) {
		fmt.Printf("  %s\n", cmd)
	}
	return nil
}

func runMaintenanceRevisions(c *cli, session *maintenanceSession) error {
	action, err := promptChoice(session, "Revision flow", []string{"list", "current/inspect", "apply/rollback"}, "list")
	if err != nil {
		return err
	}
	switch action {
	case "list":
		return runMaintenanceRevisionsList(c)
	case "current/inspect":
		return runMaintenanceAPICommand(c, httpMethodGet, "/api/revisions/current", nil)
	case "apply/rollback":
		if session.readOnly {
			return fmt.Errorf("apply/rollback requires destructive mode")
		}
		revID, err := promptLine(session, "Revision ID", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(revID) == "" {
			return fmt.Errorf("revision id is required")
		}
		confirm, err := promptChoice(session, "Apply revision and verify after apply?", []string{"no", "yes"}, "no")
		if err != nil {
			return err
		}
		if confirm != "yes" {
			return fmt.Errorf("apply cancelled")
		}
		if err := cmdRevisions(c, []string{"apply", revID}, false); err != nil {
			return err
		}
		fmt.Println("Post-apply verification:")
		return runMaintenanceHealthProbes(c, session)
	default:
		return fmt.Errorf("unsupported revision action %q", action)
	}
}

func runMaintenanceBans(c *cli, session *maintenanceSession) error {
	action, err := promptChoice(session, "Ban flow", []string{"list", "ban", "unban"}, "list")
	if err != nil {
		return err
	}
	if action == "list" {
		return runMaintenanceBansList(c, session.defaultSiteID)
	}
	if session.readOnly {
		return fmt.Errorf("ban/unban requires destructive mode")
	}
	ip, err := promptLine(session, "IP", "")
	if err != nil {
		return err
	}
	siteID, err := promptLine(session, "Site ID", session.defaultSiteID)
	if err != nil {
		return err
	}
	confirm, err := promptChoice(session, fmt.Sprintf("Confirm %s for %s on site %s", action, strings.TrimSpace(ip), strings.TrimSpace(siteID)), []string{"no", "yes"}, "no")
	if err != nil {
		return err
	}
	if confirm != "yes" {
		return fmt.Errorf("%s cancelled", action)
	}
	if err := cmdBanLike(c, action, []string{strings.TrimSpace(ip), "--site", strings.TrimSpace(siteID)}, false); err != nil {
		return err
	}
	fmt.Println("Post-check via bans list:")
	return runMaintenanceBansList(c, strings.TrimSpace(siteID))
}

func runMaintenanceEmergency(_ *cli, session *maintenanceSession) error {
	fmt.Println("Emergency workflow (safe-by-default):")
	for _, cmd := range maintenanceEmergencyCommands() {
		fmt.Printf("  %s\n", cmd)
	}
	if session.readOnly {
		fmt.Println("Read-only mode active: no destructive switch executed.")
		return nil
	}
	confirm, err := promptChoice(session, "Mark emergency plan as acknowledged?", []string{"no", "yes"}, "no")
	if err != nil {
		return err
	}
	if confirm != "yes" {
		return fmt.Errorf("emergency action cancelled")
	}
	fmt.Println("No direct emergency toggle is executed automatically. Use printed commands intentionally on the target VM.")
	return nil
}

func runMaintenanceAPICommand(c *cli, method, path string, payload any) error {
	value, err := c.requestJSON(method, path, payload, true)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	return renderMaintenanceSummary(path, value)
}

func runMaintenanceRevisionsList(c *cli) error {
	return runMaintenanceAPICommand(c, httpMethodGet, "/api/revisions", nil)
}

func runMaintenanceBansList(c *cli, siteID string) error {
	args := []string{"list"}
	if strings.TrimSpace(siteID) != "" {
		args = append(args, "--site", strings.TrimSpace(siteID))
	}
	return cmdBans(c, args, false)
}

func maintenanceStackAuditCommands() []string {
	return []string{
		"cd /opt/tarinio/deploy/compose/default && docker compose ps",
		"cd /opt/tarinio/deploy/compose/default && docker compose images",
		"docker system df",
		"df -h / /var/lib/docker",
		"free -m",
		"grep -E '^(POSTGRES_|WAF_RUNTIME_API_TOKEN|CONTROL_PLANE_SECURITY_PEPPER|CLICKHOUSE_|OPENSEARCH_)' /opt/tarinio/deploy/compose/default/.env | sed 's/=.*$/=<redacted>/'",
	}
}

func maintenanceLogCommands() []string {
	return []string{
		"cd /opt/tarinio/deploy/compose/default && docker compose logs --tail=200 control-plane",
		"cd /opt/tarinio/deploy/compose/default && docker compose logs --tail=200 waf-runtime",
		"cd /opt/tarinio/deploy/compose/default && docker compose logs --tail=200 postgres",
		"cd /opt/tarinio/deploy/compose/default && docker compose logs --tail=200 worker",
		"cd /opt/tarinio/deploy/compose/default && docker compose logs -f control-plane",
	}
}

func maintenancePostgresCommands() []string {
	return []string{
		"cd /opt/tarinio/deploy/compose/default && awk -F= '/^(POSTGRES_USER|POSTGRES_DB|POSTGRES_DSN)=/{print $1\"=<redacted>\"}' .env",
		"cd /opt/tarinio/deploy/compose/default && docker compose exec -T postgres pg_isready -U \"$POSTGRES_USER\" -d \"$POSTGRES_DB\"",
		"cd /opt/tarinio/deploy/compose/default && docker compose exec -T control-plane sh -lc 'printf %s \"$POSTGRES_DSN\" | sed -E \"s#://[^:]+:[^@]+@#://<user>:<redacted>@#\"'",
	}
}

func maintenanceBackupCommands() []string {
	return maintenanceBackupCommandsWithName("maintenance-snapshot")
}

func maintenanceBackupCommandsWithName(name string) []string {
	escaped := shellQuoteSingle(name)
	base := "$(date +%Y%m%d-%H%M%S)-" + escaped
	return []string{
		"SNAPSHOT_DIR=/opt/tarinio/backups/" + base,
		"mkdir -p \"$SNAPSHOT_DIR\"",
		"cp /opt/tarinio/deploy/compose/default/docker-compose.yml \"$SNAPSHOT_DIR\"/docker-compose.yml",
		"awk -F= '/^[A-Z0-9_]+=/{print $1\"=<redacted>\"}' /opt/tarinio/deploy/compose/default/.env > \"$SNAPSHOT_DIR\"/.env.redacted",
		"curl -fsS http://127.0.0.1:8080/api/revisions > \"$SNAPSHOT_DIR\"/revisions.json",
		"tar -czf \"$SNAPSHOT_DIR\".tar.gz -C \"$(dirname \"$SNAPSHOT_DIR\")\" \"$(basename \"$SNAPSHOT_DIR\")\"",
	}
}

func maintenanceEmergencyCommands() []string {
	return []string{
		"1) Capture redacted snapshot before any switch (see backup snapshot action).",
		"2) Run health probes and docker compose ps before toggles.",
		"3) For temporary relax/block toggles edit only intended env/profile values, then: docker compose down && docker compose up -d",
		"4) After switch verify /healthz, waf-cli me, waf-cli revisions, and target probe URLs.",
		"5) Revert using captured snapshot and repeat verification.",
	}
}

const httpMethodGet = "GET"
