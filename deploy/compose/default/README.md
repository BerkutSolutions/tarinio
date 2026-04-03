# Stage 1 Operator Quickstart

This document is the practical startup guide for the Stage 1 single-node WAF system.

It is written for the current MVP stack:
- `ui`
- `control-plane`
- `runtime`
- `redis`
- `postgres`

The goal is simple:

`docker compose up` -> ready local admin -> automatic localhost certificate + revision -> working HTTPS WAF on `https://localhost`

---

## Quickstart

Minimal path:

1. Clone the repository.
2. Open a terminal in the repository root.
3. Change into the default profile directory:

```powershell
cd deploy/compose/default
```

4. Copy the environment file:

```powershell
Copy-Item .env.example .env
```

5. Return to the repository root:

```powershell
cd ../../..
```

6. Start the full stack:

```powershell
make up
```

After the stack is up:
- UI bootstrap entry: `http://localhost`
- Control-plane API: `http://localhost:8080`
- Protected HTTPS entrypoint: `https://localhost`

Default local development shortcut:
- the compose env file enables bootstrap admin creation automatically
- the compose env file also enables full dev fast start automatically
- default username: `admin`
- default password: `admin`
- this is intended for local debugging only

---

## First Run

On first startup with the default compose env:

1. Start the stack.
2. Wait for `control-plane` and `runtime` to become healthy.
3. Open `https://localhost/login`

Default development credentials:
- username: `admin`
- password: `admin`

What the startup path does automatically:
- creates the bootstrap admin user
- creates an internal management `Site`
- creates an internal management `Upstream`
- creates a development certificate for `localhost`
- creates `TLSConfig`
- creates a `Revision`
- runs `compile + apply`

After fast start completes:
- `https://localhost/login` is ready immediately
- the active revision is written to `/var/lib/waf/active/current.json`
- runtime reloads the active bundle
- HTTPS is available on `https://localhost`

Important Stage 1 note:
- the current development certificate flow produces a local self-signed certificate for the management path
- this is expected for the single-node dev stack

Important local-dev behavior:
- `http://localhost` remains the pre-init shell/onboarding entrypoint
- with dev fast start enabled, operators should use `https://localhost/login` for normal access
- the internal management site is created automatically so deleting ordinary user sites does not remove control-plane access
- the management site is not meant to be edited as a normal protected site

## Dev Fast Start

The compose files ship with a local-only bootstrap admin and full dev fast start enabled by default:

```env
CONTROL_PLANE_BOOTSTRAP_ADMIN_ENABLED=true
CONTROL_PLANE_BOOTSTRAP_ADMIN_ID=admin
CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME=admin
CONTROL_PLANE_BOOTSTRAP_ADMIN_EMAIL=admin@localhost
CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD=admin
CONTROL_PLANE_DEV_FAST_START_ENABLED=true
CONTROL_PLANE_DEV_FAST_START_HOST=localhost
CONTROL_PLANE_DEV_FAST_START_CERTIFICATE_ID=control-plane-localhost-tls
CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID=control-plane-access
CONTROL_PLANE_DEV_FAST_START_UPSTREAM_HOST=ui
CONTROL_PLANE_DEV_FAST_START_UPSTREAM_PORT=80
CONTROL_PLANE_DEV_FAST_START_RETRY_DELAY_SECONDS=2
CONTROL_PLANE_DEV_FAST_START_MAX_ATTEMPTS=30
```

Use this when you want:
- a predictable local admin on every clean start
- immediate `https://localhost/login` availability on a clean reset
- localhost certificate issuance and first active revision without manual onboarding
- repeated UI or runtime debugging without rebuilding first-run state

Do not use these static credentials outside local development.

---

## Architecture

The stack is intentionally split into separate responsibilities:

- `ui`
  - static frontend shell
  - proxies `/api/*` to control-plane
- `control-plane`
  - owns system state
  - stores configuration entities
  - creates revision snapshots
  - runs compile/apply orchestration
- `compiler`
  - turns revision snapshot data into a deterministic bundle
- `runtime`
  - reads only the active revision bundle
  - does not read control-plane state directly
  - reloads NGINX using the active bundle only
- `redis`
  - coordination baseline
- `postgres`
  - schema baseline for Stage 1 stack completeness

Flow:

```text
UI -> API -> compile -> apply -> runtime
```

More explicitly:

```text
UI
  -> control-plane API
  -> revision snapshot
  -> compiler bundle
  -> active/current.json switch
  -> runtime internal reload
  -> runtime serves traffic on 443
```

Reload behavior:
- control-plane triggers apply
- apply writes the new active pointer
- control-plane calls the runtime internal reload endpoint
- runtime reloads NGINX against the active bundle

---

## Volumes And Data

The compose stack uses these persistent directories:

- `deploy/compose/data/revisions`
  - mounted as `/var/lib/waf`
  - shared between control-plane and runtime
- `deploy/compose/data/control-plane`
  - mounted as `/var/lib/waf/control-plane`
  - stores file-backed control-plane state
- `deploy/compose/data/certificates`
  - mounted as `/var/lib/waf/control-plane/certificate-materials`
  - stores certificate PEM and private key material
- `deploy/compose/data/postgres`
  - mounted as PostgreSQL data directory

Important bundle layout inside `/var/lib/waf`:

- `active/`
  - contains `current.json`
  - this is the only source of truth for the active revision
- `candidates/`
  - contains staged compiled revisions

Key rule:

`runtime` reads only:
- `active/current.json`
- compiled bundle files under `candidates/`

`runtime` does not read:
- `Site`
- `Upstream`
- policies
- users
- sessions
- audit records
- other control-plane state

That separation is intentional and must remain true.

---

## Useful Commands

Project helpers:

```powershell
make up
make down
make logs
```

Direct compose logs:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml logs -f control-plane
docker compose -f deploy/compose/default/docker-compose.yml logs -f runtime
docker compose -f deploy/compose/default/docker-compose.yml logs -f ui
```

Inspect the active revision pointer:

```powershell
Get-Content deploy/compose/data/revisions/active/current.json
```

Inspect staged revisions:

```powershell
Get-ChildItem deploy/compose/data/revisions/candidates
```

---

## Troubleshooting

### Onboarding does not complete

Check that the API is reachable:

```powershell
Invoke-WebRequest http://localhost:8080/healthz
```

Check site creation state:

```powershell
Invoke-WebRequest http://localhost:8080/api/sites -WebSession (New-Object Microsoft.PowerShell.Commands.WebRequestSession)
```

Check revision/apply summary:

```powershell
Invoke-WebRequest http://localhost:8080/api/reports/revisions -WebSession (New-Object Microsoft.PowerShell.Commands.WebRequestSession)
```

What to look for:
- no `Site` was created
- a revision exists but never becomes `active`
- recent apply summaries show `failed`

### Runtime does not apply the config

Check the active pointer:

```powershell
Get-Content deploy/compose/data/revisions/active/current.json
```

Check that the referenced bundle exists:

```powershell
Get-ChildItem deploy/compose/data/revisions/candidates -Recurse
```

Then inspect runtime logs:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml logs -f runtime
```

What to look for:
- missing `manifest.json`
- missing `nginx/nginx.conf`
- missing `modsecurity/modsecurity.conf`
- reload errors after pointer switch

### HTTPS does not work

Check certificate material:

```powershell
Get-ChildItem deploy/compose/data/certificates -Recurse
```

Check that `TLSConfig` was created and compile/apply completed.

In practice, if onboarding created the site but HTTPS is unavailable, inspect:
- runtime logs
- active bundle under `deploy/compose/data/revisions/candidates/<revision-id>/tls`
- certificate files under `deploy/compose/data/certificates`

Likely causes:
- certificate material missing
- TLS bundle files missing
- `TLSConfig` was not created
- apply failed before runtime reload

### CPU is high

Stage 1 includes only baseline L4 protection.

Check runtime logs first:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml logs -f runtime
```

If needed, inspect the L4 guard rules inside the runtime container:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml exec runtime iptables -S
```

What to verify:
- `WAF-RUNTIME-L4` chain exists
- jump rule is installed before NGINX traffic is processed
- `connlimit` rule exists
- `hashlimit` rule exists

If rules are missing:
- confirm runtime started with `NET_ADMIN`
- confirm `WAF_L4_GUARD_ENABLED=true`

### UI loads but API actions fail

Check UI logs:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml logs -f ui
```

Check control-plane logs:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml logs -f control-plane
```

The UI is static and proxies `/api/*` to control-plane. If the shell loads but actions fail, the issue is usually:
- control-plane not healthy
- auth/me failure
- backend validation failure

---

## Stage 1 Limits

This stack is production-minded, but it is still Stage 1 MVP.

Current limitations:

- development onboarding uses a self-signed certificate flow
- no distributed deployment
- no high availability
- no multi-node rollout
- no fleet management
- no production-grade DDoS system
- only baseline L4 protection is included
- no external secret manager
- no cloud-specific infrastructure

This is intentional.

The goal of Stage 1 is:
- one node
- deterministic apply flow
- working UI
- working HTTPS runtime
- clear separation between control-plane and runtime

---

## What вЂњDoneвЂќ Looks Like

A new operator should be able to:

1. start the stack
2. open `http://localhost`
3. login with the bootstrap admin
4. complete onboarding
5. get a working active revision
6. open `https://localhost`
7. see traffic served by the runtime

They should be able to do that without reading the codebase first.
