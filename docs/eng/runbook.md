# Runbook (EN)

Documentation baseline: `1.1.3`

## Fast health checks

- Liveness: `GET /healthz`
- Version/build: `GET /api/app/meta`
- UI reachability: `/login`, `/dashboard`

## Daily operator checks

1. Confirm services are healthy.
2. Confirm recent revision apply status is `succeeded`.
3. Confirm no unusual spike in blocked requests or `429`.
4. Confirm audit stream is updated.

## Standard change flow

For any policy/config change:

1. Change config in UI/API.
2. Compile a new revision.
3. Apply the revision.
4. Verify traffic and activity.
5. Roll back if behavior is unsafe.

## Incident triage

When traffic breaks:

1. Identify layer first:
   - L4 anti-DDoS,
   - WAF policy,
   - rate-limit,
   - TLS/certificate,
   - upstream availability.
2. Check latest changes in audit and revision reports.
3. Re-apply last known good revision if blast radius is high.

## Rollback playbook

1. Open revisions page.
2. Select previous known-good revision.
3. Apply revision.
4. Confirm `apply succeeded`.
5. Confirm runtime traffic recovery.

## Logs

Primary container logs:

- `control-plane`
- `runtime`
- `worker`

## Escalation checklist

Escalate to incident owner when one of these holds:

- repeated apply failures,
- health endpoint unstable > 5 minutes,
- widespread 4xx/5xx after policy change,
- certificate expiration/issuance failure for production host.

## Related guides

- `docs/eng/operators/anti-ddos-runbook.md`
- `docs/eng/operators/waf-tuning-guide.md`
- `docs/eng/security.md`



