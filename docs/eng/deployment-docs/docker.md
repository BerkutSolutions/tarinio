---
sidebar_position: 2
---

# Docker Deployment

This is the primary deployment guide for TARINIO v1.3.5+ with Docker/Docker Compose in production-like environments.

## Full Service Stack

| Service | Role |
|---|---|
| `control-plane` | API, config compiler, UI backend |
| `runtime` | nginx/OpenResty, L4-guard, traffic handling |
| `tarinio-sentinel` | DDoS model, credential stuffing detection, JA3 |
| `postgres` | Primary database (sites, revisions, virtual patches) |
| `opensearch` | Request and event index |
| `vault` | Secret store (mTLS certificates, tokens) |
| `ui` | Static frontend |

Vault is mandatory from v1.3.x — it stores mTLS certificates (ClientCA, upstream client keys).

## Option 1: AIO Bootstrap

### sh

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

### PowerShell

```powershell
iwr https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh -UseBasicParsing | sh
```

## Option 2: Docker Compose

Main compose docs:

- `deploy/compose/README.md`
- `deploy/compose/default/README.md`
- `deploy/compose/auto-start/README.md`

### sh

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

### PowerShell

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Key Volumes

| Volume | Purpose |
|---|---|
| `waf-runtime-data` | Compiled nginx configs |
| `waf-control-plane-data` | Control-plane data |
| `waf-certificates-data` | mTLS certificates (incoming + upstream) |
| `waf-l4-adaptive` | Adaptive L4 ban-list from sentinel |
| `waf-sentinel-state` | DDoS model state |
| `waf-vault-data` | Vault data |

## Post-Deploy Checklist

- `/login` is reachable;
- `/healthz` is stable;
- `GET /core-docs/api/app/meta` returns the expected version;
- first compile/apply succeeds;
- runtime serves expected host traffic;
- `tarinio-sentinel` healthcheck is green (`/out/adaptive.json` is non-empty);
- Vault is initialized and unsealed (`vault status` shows `Sealed: false`).

## Production Notes

- replace default secrets (`CONTROL_PLANE_SECURITY_PEPPER`, `WAF_RUNTIME_API_TOKEN`, `POSTGRES_PASSWORD`) before first exposure;
- restrict control-plane access (port 8080) with network rules;
- enforce HTTPS for admin and operator flows;
- for mTLS — load certificates via Vault, do not mount them as raw files;
- define backup and rollback ownership before go-live.
