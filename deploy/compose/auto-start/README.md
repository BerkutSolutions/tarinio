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
- UI (loopback-only by default): `http://127.0.0.1:18080`
- Edge login (public WAF entry): `https://localhost/login`
- Edge HTTP endpoint: `http://localhost` (expected redirect/edge behavior depends on site policy)

Port model in this profile:
- runtime publishes external edge ports `80/443` (`WAF_RUNTIME_HTTP_PORT`, `WAF_RUNTIME_HTTPS_PORT`);
- UI is intentionally bound to loopback by default (`WAF_UI_BIND_ADDR=127.0.0.1`) to avoid exposing management UI on public interfaces.


