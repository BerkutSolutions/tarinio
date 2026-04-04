# TARINIO Compose Profile - default (production baseline)

`default` is the production-oriented compose profile.

## Key properties

- No automatic bootstrap admin creation by default.
- No dev fast start flow by default.
- Operators configure admin, sites, certificates, and revisions manually.
- Designed for controlled deployment scenarios.

## Start

From repository root:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Required preparation

1. Review and update `deploy/compose/default/.env`.
2. Replace all placeholder secrets/passwords.
3. Configure trusted network, TLS strategy, and backup policy.

## After startup

1. Open UI and complete initial setup.
2. Configure upstreams, sites, TLS, and policies.
3. Compile and apply revision.

## Localhost auto bootstrap profile

For localhost fast-start and auto bootstrap use:

- `deploy/compose/auto-start/docker-compose.yml`
- `deploy/compose/auto-start/README.md`
