# Berkut Solutions - TARINIO CLI

`waf-cli` is an isolated CLI container for full control-plane operations.

Use the helper script from repo root:

```powershell
.\waf-cli.ps1 <command>
```

The helper already uses:
- `deploy/compose/default/docker-compose.yml`
- `--profile tools`
- `run --rm --no-deps`
- pre-check that `control-plane` is running (clear error if not)

## Prerequisite

Start platform services first:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d
```

## Quick examples

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 sites list
.\waf-cli.ps1 events --limit 50
.\waf-cli.ps1 bans list
.\waf-cli.ps1 unban 172.18.0.1
```

## Output modes

Default output is human-readable (tables and summaries).

Raw JSON mode:

```powershell
.\waf-cli.ps1 --json events --limit 20
```

## Core commands

### Health and auth

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 setup status
.\waf-cli.ps1 me
```

### Sites and bans

```powershell
.\waf-cli.ps1 sites list
.\waf-cli.ps1 sites delete site-a

.\waf-cli.ps1 ban 172.18.0.1 --site control-plane-access
.\waf-cli.ps1 unban 172.18.0.1 --site control-plane-access
.\waf-cli.ps1 bans list
.\waf-cli.ps1 bans list --site control-plane-access
```

### Events and audit

```powershell
.\waf-cli.ps1 events --limit 100
.\waf-cli.ps1 events --type security_waf --severity warning --site-id control-plane-access

.\waf-cli.ps1 audit --limit 50
.\waf-cli.ps1 audit --action site.create --site-id control-plane-access --status success --limit 100 --offset 0
```

### Revisions and reports

```powershell
.\waf-cli.ps1 revisions compile
.\waf-cli.ps1 revisions apply rev-000001

# compatibility aliases
.\waf-cli.ps1 compile
.\waf-cli.ps1 apply rev-000001

.\waf-cli.ps1 reports revisions
```

### Backend modules (list/get)

```powershell
.\waf-cli.ps1 upstreams list
.\waf-cli.ps1 tls list
.\waf-cli.ps1 certificates list

.\waf-cli.ps1 access-policies list
.\waf-cli.ps1 waf-policies list
.\waf-cli.ps1 rate-limit-policies list
```

### Easy profile and Anti-DDoS from files

```powershell
.\waf-cli.ps1 easy get control-plane-access
.\waf-cli.ps1 easy upsert control-plane-access --file .\profile.json

.\waf-cli.ps1 antiddos get
.\waf-cli.ps1 antiddos upsert --file .\antiddos.json
```

### Universal API mode (full backend coverage)

Use this for any endpoint not yet wrapped by a dedicated command.

```powershell
.\waf-cli.ps1 api GET /api/sites
.\waf-cli.ps1 api GET /api/reports/revisions
.\waf-cli.ps1 api PUT /api/anti-ddos/settings --file .\antiddos.json
.\waf-cli.ps1 api DELETE /api/sites/site-a
```

## Auth and connection flags

Defaults:
- base URL: `http://127.0.0.1:8080`
- username/password: `admin/admin`

Override:

```powershell
.\waf-cli.ps1 --base-url http://control-plane:8080 --username admin --password admin sites list
```

No-auth mode (for public endpoints only):

```powershell
.\waf-cli.ps1 --no-auth health
```

TLS skip verify:

```powershell
.\waf-cli.ps1 --insecure --base-url https://control-plane:8443 health
```

## Direct compose command (without helper script)

```powershell
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli sites list
```

## Test profile (full stack + test container)

Dedicated one-command profile with:
- WAF/UI on `https://localhost:8080`
- protected test page on `https://localhost:8081`

```powershell
cd deploy/compose/testpage
docker compose up -d --build
```

`ui` image build runs `go test ./ui/tests`, so broken UI/i18n keys stop the compose startup during image build.

Open:
- `https://localhost:8080`
- `https://localhost:8081`

Useful checks:

```powershell
cd deploy/compose/testpage
docker compose logs -f control-plane runtime control-plane-test runtime-test request-archive
docker compose ps
```

Adaptive model checks:

```powershell
cd deploy/compose/testpage
docker compose logs -f ddos-model-mgmt ddos-model-app runtime runtime-test
docker compose exec ddos-model-mgmt sh -lc "cat /out/adaptive.json"
docker compose exec ddos-model-app sh -lc "cat /out/adaptive.json"
```

Request archive checks:

```powershell
cd deploy/compose/testpage
docker compose exec request-archive sh -lc "tail -n 40 /archive/requests.jsonl"
```

Request archive env limits:
- `REQUEST_ARCHIVE_MAX_FILE_SIZE_MB` (default `256`)
- `REQUEST_ARCHIVE_MAX_FILES` (default `7`)
- `REQUEST_ARCHIVE_RETENTION_DAYS` (default `2`)

Adaptive model env limits/tuning:
- `WAF_L4_GUARD_REAPPLY_INTERVAL_SECONDS` (default `5`)
- `DDOS_MODEL_POLL_INTERVAL_SECONDS` (default `2`)
- `DDOS_MODEL_DECAY_LAMBDA` (default `0.08`)
- `DDOS_MODEL_THROTTLE_THRESHOLD` (default `2.5`)
- `DDOS_MODEL_DROP_THRESHOLD` (default `6.0`)
- `DDOS_MODEL_HOLD_SECONDS` (default `60`)
- `DDOS_MODEL_THROTTLE_RATE_PER_SECOND` (default `3`)
- `DDOS_MODEL_THROTTLE_BURST` (default `6`)
- `DDOS_MODEL_THROTTLE_TARGET` (default `REJECT`)
- `DDOS_MODEL_WEIGHT_429` (default `1.0`)
- `DDOS_MODEL_WEIGHT_403` (default `1.8`)
- `DDOS_MODEL_WEIGHT_444` (default `2.2`)
- `DDOS_MODEL_EMERGENCY_RPS` (default `180`)
- `DDOS_MODEL_EMERGENCY_UNIQUE_IPS` (default `40`)
- `DDOS_MODEL_EMERGENCY_PER_IP_RPS` (default `60`)
- `DDOS_MODEL_WEIGHT_EMERGENCY_BOTNET` (default `6.0`)
- `DDOS_MODEL_WEIGHT_EMERGENCY_SINGLE` (default `4.0`)

Example:

```powershell
cd deploy/compose/testpage
$env:REQUEST_ARCHIVE_MAX_FILE_SIZE_MB="64"
$env:REQUEST_ARCHIVE_MAX_FILES="10"
$env:REQUEST_ARCHIVE_RETENTION_DAYS="7"
docker compose up -d --build
```

The same build-time UI/i18n test gate applies here as well.

Stop and cleanup:

```powershell
cd deploy/compose/testpage
docker compose down -v --remove-orphans --rmi local
```

Full guide:
- `deploy/compose/testpage/README.md`
- `deploy/compose/testpage/DDOS_ADAPTIVE_MODEL.md`

## Full deployment (GUI + test page)

`testpage` profile already includes both endpoints in one command.

