# TARINIO Compose Profile - default (standalone baseline)

`default` is the standalone production-oriented compose profile.

## Key properties

- No automatic bootstrap admin creation by default.
- No dev fast start flow by default.
- Operators configure admin, sites, certificates, and revisions manually.
- `Vault` is enabled by default for secret management.
- `OpenSearch` is the default logging backend for both fresh and historical data retention in standalone mode.
- `Redis` and `ClickHouse` are intentionally not part of the standalone footprint.
- Designed for controlled single-node deployment scenarios.

## Start

From repository root:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Required preparation

1. Review and update `deploy/compose/default/.env`.
2. Replace all placeholder secrets/passwords.
3. Configure trusted network, TLS strategy, and backup policy.
4. If you need multi-node control-plane, Redis coordination, and dedicated cold analytics storage, use `deploy/compose/enterprise/`.

## After startup

1. Open UI and complete initial setup.
2. Configure upstreams, sites, TLS, and policies.
3. Compile and apply revision.

## Localhost auto bootstrap profile

For localhost fast-start and auto bootstrap use:

- `deploy/compose/auto-start/docker-compose.yml`
- `deploy/compose/auto-start/README.md`

## Full enterprise profile

For multi-node / full-stack deployments use:

- `deploy/compose/enterprise/docker-compose.yml`
- `deploy/compose/enterprise/README.md`
