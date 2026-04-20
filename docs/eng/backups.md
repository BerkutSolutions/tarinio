# TARINIO 2.0.1 Backups And Restore

Wiki baseline: `2.0.1`

This document describes what must be backed up and how restore readiness should be validated for TARINIO.

## What Counts As Critical Data

Critical data includes:

- PostgreSQL data;
- runtime state and revision store data;
- TLS and certificate materials;
- `.env`, secrets, and external override configuration;
- the recorded application version and active revision context.

## What Must Be Backed Up

- database volumes;
- runtime and state volumes used by the deployment profile;
- certificate-related volumes when materials are stored locally;
- secrets and deployment overrides stored outside the repository.

## Minimum Policy

- full backup before every upgrade;
- at least daily backups in normal operation;
- keep at least 7 restore points;
- store at least one copy off the primary host;
- validate restore procedures regularly in a separate environment.

## Pre-Backup Checklist

1. Check `/healthz`.
2. Make sure there is no active incident or stuck apply flow.
3. Record the application version.
4. Record the active revision.
5. Record the operator and backup timestamp.

## Metadata To Store Alongside The Backup

- timestamp;
- hostname;
- application version;
- active revision ID;
- compose profile;
- a short reason for the backup: scheduled, pre-upgrade, emergency.

## Restore Drill

At least monthly, restore should be validated:

1. Deploy the same TARINIO version.
2. Restore the database and required volumes.
3. Start the stack.
4. Verify login and `/healthz`.
5. Confirm that sites, certificates, and policies are present.
6. Verify compile/apply.
7. Verify that runtime serves the expected host.

If restore is never tested, backup quality is unknown.

## Restore Priorities

1. Control-plane state and secrets.
2. TLS materials and runtime state.
3. Re-establish the active configuration through compile/apply and revisions.

## When Full Restore Is Needed

- host loss;
- database corruption;
- failed upgrade that damaged state integrity;
- compromise requiring redeployment onto a clean platform.

## Related Documents

- `docs/eng/upgrade.md`
- `docs/eng/runbook.md`
- `docs/eng/security.md`
