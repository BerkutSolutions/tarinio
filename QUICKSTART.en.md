# Quick Start (EN)

Minimal local startup for Berkut Solutions - TARINIO.

## 1. Prepare

```powershell
Copy-Item .env.example .env
```

Check `.env`:
- use non-default secrets for non-dev environments
- set your operational timezone (`TZ`)

## 2. Start

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## 3. Verify

- UI: `https://localhost/` (reverse-proxy profile) or your control-plane URL.
- API liveness: `GET /healthz`
- Product version: `GET /api/app/meta`

## 4. First actions after login

1. Complete initial admin bootstrap.
2. Create an upstream and a site.
3. Compile and apply a revision.

## 5. Useful links

- Full deploy: [`docs/eng/deploy.md`](docs/eng/deploy.md)
- Runbook: [`docs/eng/runbook.md`](docs/eng/runbook.md)
- Upgrade/rollback: [`docs/eng/upgrade.md`](docs/eng/upgrade.md)
- CLI: [`docs/CLI_COMMANDS.md`](docs/CLI_COMMANDS.md)
