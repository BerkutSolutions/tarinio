# Docker Deployment

This page belongs to the current documentation branch.

This is the primary deployment guide for TARINIO with Docker/Docker Compose in production-like environments.

## Minimum Stack

- `control-plane`
- `runtime`
- `postgres`
- Docker networks, volumes, and environment-specific secrets

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

## Post-Deploy Checklist

- `/login` is reachable;
- `/healthz` is stable;
- `GET /core-docs/api/app/meta` returns the expected version;
- first compile/apply succeeds;
- runtime serves expected host traffic.

## Production Notes

- replace default secrets before first exposure;
- restrict control-plane access;
- enforce HTTPS for admin and operator flows;
- define backup and rollback ownership before go-live.
