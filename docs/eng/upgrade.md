# Upgrade / rollback (EN)

Documentation baseline: `1.1.6`

Minimum supported upgrade path: `latest-1 -> latest`.

## Required lifecycle

1. Preflight
2. Backup
3. Upgrade/migrate
4. Smoke validation
5. Rollback decision

## 1. Preflight

Before touching production:

- Confirm maintenance window and rollback owner.
- Confirm current app version and target version.
- Confirm required environment variables for target version.
- Confirm runtime health and no active apply jobs.

## 2. Backup (mandatory)

Create a complete backup before the upgrade:

- PostgreSQL data volume
- runtime and certificate state volumes (if present)
- `.env` and secret material

Reference: `docs/eng/backups.md`.

## 3. Upgrade/migrate

Recommended sequence:

1. Pull/update deployment artifacts.
2. Apply compose/config updates.
3. Start or restart services.
4. Wait for control-plane and runtime readiness.
5. If migration jobs exist, ensure they finish successfully.

## 4. Smoke validation

Minimum checks after upgrade:

- `GET /healthz` is healthy.
- UI login works.
- `GET /api/app/meta` returns expected version/build.
- Compile + apply succeeds for a small safe revision.
- HTTPS serving is healthy on the protected host.

## 5. Rollback decision

Rollback immediately if:

- health does not stabilize,
- login/control-plane functions are degraded,
- compile/apply cannot succeed,
- traffic behavior is unsafe.

Rollback path:

1. Restore previous deployment version.
2. Restore backup if state is inconsistent.
3. Re-apply last known good revision.
4. Run smoke checks again.

## Notes

- Do not skip backup to save time.
- Do not batch unrelated risky changes in the same window.
- Keep an upgrade log with timestamps and operator actions.



