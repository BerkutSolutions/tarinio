# Compose Profiles

This directory contains deployment profiles for TARINIO:

- `default/` - production-oriented baseline (no auto bootstrap, no dev fast start).
- `auto-start/` - localhost operator profile with auto bootstrap and dev fast start.
- `testpage/` - local test profile with extra protected test page.
- `ha-lab/` - resource-limited HA / multi-node control-plane validation lab with 20-service provisioning helpers.

## AIO one-command start

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

## Run from repository root

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
docker compose -f deploy/compose/auto-start/docker-compose.yml up -d --build
docker compose -f deploy/compose/ha-lab/docker-compose.yml --profile tools up -d --build
```

## Run from `deploy/compose`

```powershell
cd deploy/compose
docker compose -f .\default\docker-compose.yml up -d --build
docker compose -f .\auto-start\docker-compose.yml up -d --build
docker compose -f .\ha-lab\docker-compose.yml --profile tools up -d --build
```

Profile docs:

- `deploy/compose/default/README.md`
- `deploy/compose/auto-start/README.md`
- `deploy/compose/testpage/README.md`
- `deploy/compose/ha-lab/README.md`


