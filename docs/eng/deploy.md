# Deployment

This page belongs to the current documentation branch.

This document describes practical deployment paths for TARINIO across local, test, and production-like environments.

## What Gets Deployed

The minimal stack includes:

- `control-plane`
- `runtime`
- `postgres`
- related volumes, networks, and compose profile configuration

Depending on the profile, additional supporting services may exist.

## Deployment Options

### AIO One-Command Install

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

This path is useful for:

- quick proof-of-concept environments;
- labs and test stands;
- first contact with the product.

### Docker Compose

The main compose profile documentation lives in:

- `deploy/compose/README.md`
- `deploy/compose/default/README.md`
- `deploy/compose/auto-start/README.md`

Base startup command:

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## After Startup

Typical post-start checks:

- administrative UI: `/login`
- health endpoint: `/healthz`
- runtime ingress on the HTTP/HTTPS ports exposed by the profile

If bootstrap has not been completed yet, the product can redirect into onboarding.

## Production Checklist

- use non-default secrets;
- restrict network access to the control-plane;
- enable HTTPS;
- configure trusted proxies correctly;
- ensure volume and secret backup coverage;
- verify that retention and update-check behavior match your operational policy;
- predefine rollback ownership and rollback order.

## What To Prepare Before Production

- DNS and network rules;
- a log/event retention policy;
- a TLS strategy: import, self-signed, ACME, or DNS-01;
- an operator access model;
- a backup/restore process;
- a compile/apply/rollback operating model.

## Initial Steps After Deployment

1. Open `/login`.
2. If the system is not initialized, complete onboarding.
3. Create the first administrator.
4. Create the first site and upstream.
5. Configure TLS.
6. Verify that healthcheck passes.
7. Run the first compile/apply cycle.

## What Counts As A Successful Deployment

The deployment should be considered complete only when:

- `/healthz` is stable;
- login and onboarding work;
- `GET /api/app/meta` returns `2.0.2`;
- a site can be created and turned into a working revision;
- runtime serves the expected host after apply;
- events, requests, and audit entries appear in the UI.

## Related Documents

- `docs/eng/security.md`
- `docs/eng/runbook.md`
- `docs/eng/upgrade.md`
- `docs/eng/backups.md`
- `docs/eng/ui.md`
