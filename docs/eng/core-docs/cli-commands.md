# CLI Commands

This page belongs to the current documentation branch.

`waf-cli` is the containerized CLI for `control-plane` API operations.

## Run

Windows (from repository root):

```powershell
.\waf-cli.ps1 <command>
```

Linux/macOS (`docker compose`):

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli <command>
```

## Global Flags

- `--base-url` (default `http://127.0.0.1:8080`)
- `--username` (default `admin`)
- `--password` (default `admin`)
- `--insecure` (skip TLS certificate verification)
- `--no-auth` (skip login for command execution)
- `--json` (print raw JSON responses)

Example:

```powershell
.\waf-cli.ps1 --base-url https://control-plane:8443 --username admin --password admin --insecure sites list
```

## Environment Variables

- `WAF_CLI_BASE_URL`
- `WAF_CLI_USERNAME`
- `WAF_CLI_PASSWORD`
- `CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME`
- `CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD`
- `WAF_CLI_DEFAULT_SITE` (for `ban` / `unban`)
- `CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID` (default-site fallback)

## Full Command Catalog

### Health and Auth

- `health`
- `setup status`
- `me`

### Sites

- `sites list`
- `sites delete <site-id>`

### Policy and Certificate Catalogs

- `upstreams list`
- `tls list`
- `certificates list`
- `access-policies list`
- `waf-policies list`
- `rate-limit-policies list`

### Events and Audit

- `events [--limit N] [--type TYPE] [--site-id SITE] [--severity LEVEL]`
- `audit [--action ACTION] [--site-id SITE] [--status STATUS] [--limit N] [--offset N]`

### IP Blocking

- `ban <ip> [--site <site-id>]`
- `unban <ip> [--site <site-id>]`
- `bans list [--site <site-id>]`

### Easy Profile and Anti-DDoS

- `easy get <site-id>`
- `easy upsert <site-id> --file <profile.json>`
- `antiddos get`
- `antiddos upsert --file <antiddos.json>`

### Revisions and Reports

- `revisions compile`
- `revisions apply <revision-id>`
- `reports revisions`

Compatible aliases:

- `compile` = `revisions compile`
- `apply <revision-id>` = `revisions apply <revision-id>`

### Generic API Mode

- `api <GET|POST|PUT|DELETE> <path> [--file <body.json>]`

`<path>` should be an API path (`/api/...`). If `/` is omitted, CLI prepends it automatically.

Examples:

```powershell
.\waf-cli.ps1 api GET /api/sites
.\waf-cli.ps1 api GET /api/reports/revisions
.\waf-cli.ps1 api PUT /api/anti-ddos/settings --file .\antiddos.json
.\waf-cli.ps1 api DELETE /api/sites/site-a
```

## Quick Examples

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 sites list
.\waf-cli.ps1 events --limit 50
.\waf-cli.ps1 bans list --site control-plane-access
.\waf-cli.ps1 revisions compile
.\waf-cli.ps1 --json reports revisions
```

## Related Documents

- `docs/eng/core-docs/api.md`
- `docs/eng/core-docs/ui.md`
- `docs/eng/core-docs/security-profiles.md`
