# Backups (EN)

Documentation baseline: `1.0.15`

## Goal

Define an operational backup and restore baseline for single-node deployments.
This page is written for operators who use Docker Compose and local volumes.

## What to back up

- PostgreSQL data volume (system state, users, policy entities, revisions metadata).
- Runtime/state volumes used by the deployment profile.
- Certificate/state volumes if TLS material is generated and stored locally.
- `.env` and any external secret files (outside the repository, encrypted at rest).

## Backup policy (minimum)

- Before every upgrade: mandatory full backup.
- During normal operations: at least daily backup.
- Keep at least 7 recent restore points.
- Store at least one copy outside the host (remote storage or encrypted off-host disk).

## Pre-backup checklist

1. Confirm stack health (`/healthz` and dashboard access).
2. Confirm no active incident or ongoing rollback.
3. Record current active revision id.
4. Record current release/build version.

## Example workflow (Compose)

1. Stop write-heavy administrative changes in UI.
2. Snapshot database and runtime volumes.
3. Archive `.env` and deployment-level overrides.
4. Save backup metadata:
   - timestamp (UTC)
   - host
   - app version
   - active revision id
   - operator name

## Restore drill (required)

At least once per month, validate restore on a separate environment:

1. Deploy the same app version.
2. Restore database and required volumes.
3. Start stack.
4. Validate:
   - login works,
   - sites and policies are present,
   - active revision can be applied,
   - runtime serves expected host(s).

If restore is not tested, backup quality is unknown.

## Recovery priorities

- Priority 1: recover control-plane state (database + secrets).
- Priority 2: recover runtime artifacts and certificate state.
- Priority 3: re-run compile/apply and validate traffic.

## Related documents

- `docs/eng/upgrade.md` (mandatory pre-upgrade flow)
- `docs/eng/runbook.md` (incident and rollback operations)
