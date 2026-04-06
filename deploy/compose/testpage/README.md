# Testpage Profile (One Command)

Start:

```powershell
cd deploy/compose/testpage
docker compose up -d --build
```

Endpoints:
- `https://localhost:8080` -> WAF UI / management stack
- `https://localhost:8081` -> protected test app stack

## Architecture

This profile runs two isolated WAF stacks in one compose project:

- management stack:
  - `control-plane`
  - `runtime` (`8080 -> 443`)
  - `ui`
  - `redis-mgmt`
  - `postgres-mgmt`

- test app stack:
  - `control-plane-test`
  - `runtime-test` (`8081 -> 443`)
  - `test-login-app`
  - `postgres-app`

Additionally:
- `request-archive` container collects request logs from both runtimes.
- `ddos-model-mgmt` and `ddos-model-app` provide independent adaptive DDoS scoring per runtime.

Notes:
- `test-login-app` has no host port mapping (traffic goes only through WAF on `8081`).
- state and DB volumes are separated between management and test stacks.
- upstream defaults use static IPs inside docker network (`172.30.0.0/24`) to show IPs in UI instead of container names.

## Request Archive Container (WAF Independence)

`request-archive` stores request-level log entries in its own volume:

- archive file: `/archive/requests.jsonl`
- source logs:
  - `/logs/mgmt/access.log` (from `runtime`)
  - `/logs/app/access.log` (from `runtime-test`)

Why this proves independence:
- WAF runtimes do not depend on `request-archive` for request processing.
- if `request-archive` stops or crashes, both WAF runtimes continue serving traffic and enforcing protections.
- archive overload affects only archive container/volume, not control-plane state and not runtime traffic path.

## Adaptive DDoS Model (Independent from WAF core)

This profile includes two adaptive model containers:

- `ddos-model-mgmt` for management runtime (`8080`)
- `ddos-model-app` for app runtime (`8081`)

They parse runtime access logs, compute per-IP risk score with decay, and publish adaptive L4 rules (`throttle` -> `drop`) into dedicated volumes.

Runtime periodically re-applies L4 rules and merges:

- static anti-DDoS config from control-plane revision
- adaptive IP actions from model output

Important behavior:

- adaptation is isolated per runtime (no cross-site accidental block),
- if model fails, WAF runtime still works with static protections.

Detailed design, math model, and tuning:

- `deploy/compose/testpage/DDOS_ADAPTIVE_MODEL.md`

## Request Archive Limits (ENV)

Configure in compose env (service `request-archive`):

- `REQUEST_ARCHIVE_MAX_FILE_SIZE_MB` (default `256`)
  - rotate when `requests.jsonl` exceeds this size.
- `REQUEST_ARCHIVE_MAX_FILES` (default `7`)
  - keep up to this number of rotated files (`requests.jsonl.1`, `.2`, ...).
- `REQUEST_ARCHIVE_RETENTION_DAYS` (default `2`)
  - delete archive files older than this many days.

Example:

```powershell
$env:REQUEST_ARCHIVE_MAX_FILE_SIZE_MB="64"
$env:REQUEST_ARCHIVE_MAX_FILES="10"
$env:REQUEST_ARCHIVE_RETENTION_DAYS="7"
docker compose up -d --build
```

## Main WAF Tuning ENV

Management stack:
- `MGMT_WAF_HTTPS_PORT` (default `8080`)
- `MGMT_WAF_DOMAIN` (default `localhost`)
- `MGMT_WAF_UPSTREAM_HOST` (default `172.30.0.10`)
- `MGMT_WAF_UPSTREAM_PORT` (default `80`)
- `MGMT_WAF_DEFAULT_LIMIT_REQ_RATE` (default `20r/s`)

Test stack:
- `TEST_APP_WAF_HTTPS_PORT` (default `8081`)
- `TEST_WAF_DOMAIN` (default `localhost`)
- `TEST_WAF_UPSTREAM_HOST` (default `172.30.0.20`)
- `TEST_WAF_UPSTREAM_PORT` (default `80`)
- `TEST_WAF_DEFAULT_LIMIT_REQ_RATE` (default `5r/s`)
- `TEST_WAF_DEFAULT_BAD_BEHAVIOR_BAN_TIME_SECONDS` (default `10`)

Adaptive model:
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

## Useful Commands

Logs:

```powershell
docker compose logs -f control-plane runtime control-plane-test runtime-test request-archive
```

Inspect request archive:

```powershell
docker compose exec request-archive sh -lc "tail -n 40 /archive/requests.jsonl"
```

Inspect adaptive model output:

```powershell
docker compose exec ddos-model-mgmt sh -lc "cat /out/adaptive.json"
docker compose exec ddos-model-app sh -lc "cat /out/adaptive.json"
```

Status:

```powershell
docker compose ps
```

Stop and cleanup:

```powershell
docker compose down --remove-orphans
```

## Services UI Import/Export (JSON + ENV)

In `Services` page:

- `Export` with no selected checkboxes:
  - exports full snapshot to `waf-services-export.json`.
- select one or more services with checkboxes, then `Export`:
  - exports each selected service into its own file: `<site-id>.env`.

Import supports multiple files at once:
- `.json` (full snapshot format),
- `.env` (per-service format).

ENV import behavior:
- unknown parameter -> import error, file is rejected;
- missing newly added fields -> defaults are applied automatically and warning is shown;
- if service already exists -> update is applied and a diff is shown in UI.


