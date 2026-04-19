# TARINIO 2.0.0 Upgrade And Rollback

Wiki baseline: `2.0.0`

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

See `docs/eng/backups.md` for the detailed policy.

## 3. Upgrade

Recommended sequence:

1. Update deployment artifacts.
2. Apply compose/config changes.
3. Restart services.
4. Wait for readiness.
5. Verify that migrations finished successfully.

## 4. Smoke Validation

After upgrade, check at minimum:

- `GET /healthz`
- `/login`
- `/healthcheck`
- `GET /api/app/meta` and the expected version
- key UI sections opening successfully
- compile/apply of a new or existing revision
- HTTP/HTTPS ingress availability

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

- `docs/eng/backups.md`
- `docs/eng/runbook.md`
- `docs/eng/security.md`
