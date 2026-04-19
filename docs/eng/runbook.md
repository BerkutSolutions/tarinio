# TARINIO 2.0.0 Runbook

Wiki baseline: `2.0.0`

This runbook describes daily operator checks, standard change flow, triage, and rollback patterns for TARINIO.

## Fast Health Checks

- `GET /healthz`
- `GET /api/app/meta`
- UI access through `/login`
- successful `/healthcheck`
- expected data visible on `Dashboard`
- healthy state in `Revisions`

## Daily Operator Minimum

1. Verify that `/healthz` is healthy.
2. Confirm that the UI opens without degradation and healthcheck reports no critical issue.
3. Check `Revisions` and confirm the last working revision was applied successfully.
4. Review `Dashboard` and `Events` for attack spikes, `403`, `429`, or `5xx` anomalies.
5. Review `TLS` for certificates approaching expiration.
6. Use `Activity` when investigating unexpected configuration changes.

## Standard Change Flow

1. Make the change through `Sites`, `TLS`, `Anti-DDoS`, `OWASP CRS`, or the API.
2. Verify that the intended model matches the desired outcome.
3. Compile a new revision.
4. Apply the revision.
5. Check `Dashboard`, `Requests`, `Events`, and `Bans`.
6. Record the result in the team’s change log.

## Where To Look For Different Problems

- Application availability issue: `Dashboard`, `Requests`, `TLS`, `Sites`
- Suspected false blocks: `Events`, `Requests`, `Bans`, `Sites`
- Suspected network attack: `Dashboard`, `Anti-DDoS`, `Events`, `Bans`
- Need to know who changed settings: `Activity`
- Issue after deploy or upgrade: `/healthz`, `healthcheck`, `Revisions`, container logs

## Incident Triage

### 1. Determine The Problem Layer

- login/control-plane;
- TLS and certificates;
- routing/upstream;
- WAF/CRS;
- rate-limit and bans;
- L4/L7 Anti-DDoS;
- runtime/container health.

### 2. Review Recent Changes

- `Activity`
- `Revisions`
- `Settings`

### 3. Review The Observable Impact

- `Dashboard`
- `Requests`
- `Events`
- `Bans`

### 4. Decide On The Action

- correct the setting;
- roll back;
- temporarily relax a policy;
- escalate the incident.

## Rollback Playbook

1. Open the `Revisions` section.
2. Find the last known-good revision for the affected service.
3. Run `Apply`.
4. Wait for the apply result.
5. Validate recovery through `Dashboard` and `Requests`.
6. Confirm that user-facing traffic is restored.

If revision rollback is not sufficient, continue with deployment rollback and, if needed, backup restoration.

## Working With Bans

If an attack source must be isolated quickly:

1. Open `Bans`.
2. Create a manual ban for the IP and site.
3. Extend the ban if required.
4. If the IP was blocked incorrectly, unban it and review the site allowlist/denylist logic.

## TLS Incident Handling

1. Verify that the site has a valid TLS binding.
2. Check certificate status and expiration.
3. For ACME issues, inspect challenge mode and DNS/env parameters.
4. Re-issue or import a certificate when necessary.
5. Revalidate the ingress and target host after the fix.

## When To Escalate

Escalation is mandatory when:

- `/healthz` remains unstable for more than 5 minutes;
- onboarding or login fails;
- compile/apply fails repeatedly;
- a policy change causes widespread `4xx/5xx`;
- there is evidence of false blocking on business-critical traffic;
- a production certificate is at risk of expiring without a clear recovery path.

## Related Documents

- `docs/eng/ui.md`
- `docs/eng/security.md`
- `docs/eng/upgrade.md`
- `docs/eng/backups.md`
- `docs/eng/operators/anti-ddos-runbook.md`
