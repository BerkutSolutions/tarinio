# TARINIO Compose Profile - auto-start (localhost)

`auto-start` is the localhost operator profile for quick environment startup.

## What this profile enables

- Automatic bootstrap admin creation.
- Dev fast start flow for localhost management site and revision apply.
- Ready-to-use local debugging loop.

Default local credentials:
- username: `admin`
- password: `admin`

Use only for local development.

## Start

From repository root:

```powershell
docker compose -f deploy/compose/auto-start/docker-compose.yml up -d --build
```

After startup:
- UI: `http://localhost`
- Login: `https://localhost/login`
- API health: `http://localhost:8080/healthz`

