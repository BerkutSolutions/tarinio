# Runbook (EN)

Documentation baseline: `1.0.2`

## Basic checks

- Liveness: `GET /healthz`
- Version: `GET /api/app/meta`

## Common actions

- After configuration changes: compile and apply a new revision.
- On traffic issues: rollback to the last known good revision.

## Logs

Use container logs:
- `control-plane`
- `runtime`
- `worker`

