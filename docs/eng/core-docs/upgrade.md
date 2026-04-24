# Upgrade and Rollback

This page belongs to the current documentation branch.

This document describes the recommended upgrade lifecycle and rollback criteria for TARINIO.

## Supported Approach

Recommended upgrade path:

- `latest-1 -> latest`

In production, upgrades should be treated as controlled change windows, not as background actions.

## Required Lifecycle

1. Preflight
2. Backup
3. Upgrade
4. Smoke validation
5. Rollback decision

## 1. Preflight

Before upgrading:

- record current and target versions;
- verify that the system is healthy;
- confirm that no apply jobs are stuck;
- check available disk space and volume health;
- assign rollback ownership;
- define the maintenance window.

## 2. Backup

Before upgrade, a full backup is mandatory:

- database;
- runtime state;
- TLS/certificate materials;
- `.env` and secrets.

See `docs/eng/core-docs/backups.md` for the detailed policy.

## 3. Upgrade

Recommended sequence:

1. Update deployment artifacts.
2. Apply compose/config changes.
3. Build new images.
4. For High Availability environments, upgrade control-plane nodes one at a time behind the load balancer.
5. Wait for readiness after each step.
6. Verify that migrations finished successfully.
7. Run the strict post-upgrade smoke validation.

### Rolling / Zero-Downtime Upgrade

For High Availability topologies the preferred control-plane upgrade path is rolling:

1. keep `api-lb` online;
2. upgrade `control-plane-a`;
3. wait until it is healthy and serving traffic again;
4. upgrade `control-plane-b`;
5. confirm that API traffic remained available throughout the process.

The bundled High Availability lab includes a validation helper:

```powershell
cd deploy/compose/High Availability-lab
powershell -ExecutionPolicy Bypass -File .\upgrade\rolling-upgrade.ps1
```

or on Unix-like hosts:

```sh
cd deploy/compose/High Availability-lab
./core-docs/upgrade/rolling-upgrade.sh
```

The rolling helper continuously probes `api-lb` during each node rebuild and fails if API availability drops.

## 4. Smoke Validation

After upgrade, check at minimum:

- `GET /healthz`
- `/login`
- `/healthcheck`
- `GET /core-docs/api/app/meta` and the expected version
- key UI sections opening successfully
- compile/apply of a new or existing revision
- HTTP/HTTPS ingress availability
- `/metrics` for control-plane and runtime when metrics tokens are configured

When using `scripts/install-aio.sh`, enable the stricter validation pass:

```sh
RUN_STRICT_POST_UPGRADE_VALIDATION=1 PROFILE=default sh scripts/install-aio.sh
```

This executes `scripts/post-upgrade-smoke.sh` after the regular health gate and verifies:

- control-plane health;
- setup status and app metadata path;
- runtime health and readiness;
- host `/healthcheck`;
- metrics endpoints when protected by tokens.

## 5. Rollback Decision

Rollback is required when:

- health does not stabilize;
- login, onboarding, or the baseline UI stops working;
- compile/apply fails repeatedly;
- runtime behavior becomes unsafe or clearly degraded;
- state integrity is in doubt.

## Rollback Path

1. Restore the previous deployment version.
2. Restore backup if needed.
3. Verify `/healthz`.
4. Re-apply a known-good revision.
5. Repeat smoke checks.

## What Not To Do

- do not upgrade production without backup;
- do not combine upgrade with a large batch of unrelated risky policy changes;
- do not skip smoke validation;
- do not consider the upgrade complete only because containers started.

## Related Documents

- `docs/eng/core-docs/backups.md`
- `docs/eng/core-docs/runbook.md`
- `docs/eng/core-docs/security.md`


