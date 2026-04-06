# Stage 1 End-to-End Validation

Date: `2026-04-01`

Scope:
- clean single-node `docker compose` startup
- onboarding-driven bootstrap
- compile/apply to active runtime
- HTTPS handoff
- login and audit trail
- clean UI routing without `.html`
- runtime L4 guard bootstrap without warning noise

Environment:
- Windows host
- Docker Compose stack from `deploy/compose/`
- fresh reset before validation: `docker compose down --remove-orphans`

Notes:
- Validation was executed after fixing several blocking defects found during the run:
  - control-plane image was missing `compiler/templates`
  - syntax validation runner did not shadow absolute `/etc/waf/*` paths
  - compiler emitted invalid upstream server syntax for NGINX upstream blocks
  - onboarding could trigger duplicate apply jobs
  - UI routing still exposed `.html` paths
  - onboarding still allowed auth noise before bootstrap
  - runtime L4 guard emitted avoidable `iptables` warning noise during idempotent checks
- The final validation run used onboarding-only bootstrap. No env-seeded admin user was used for that specific verification pass.
- The compose stack may expose a local-dev fast-start shortcut through `deploy/compose/.env` for repeated debugging, but this validation document records the onboarding-driven verification pass.

## Validation Matrix

| Scenario | Expected Result | Actual Result | Pass / Fail | Notes / Root Cause |
| --- | --- | --- | --- | --- |
| A. Clean first-run onboarding entry | `http://localhost` opens onboarding, `setup status` reports no users and no sites | `GET /api/setup/status` returned `needs_bootstrap=true`, `has_users=false`, `has_sites=false`; `http://localhost` returned onboarding page | PASS | Clean start is now onboarding-first |
| B. First admin bootstrap | First user is created only through onboarding/bootstrap flow | `POST /api/auth/bootstrap` created `admin`; persisted `users.json` contains exactly one admin with requested email | PASS | Validation run intentionally used onboarding bootstrap instead of the optional dev env shortcut |
| C. Admin email capture | Email is optional, validated, stored, and visible in audit details when provided | Bootstrap with `admin@example.test` succeeded; `users.json` stored email; `audit_events.json` stored `details_json.email` | PASS | Email is now part of Stage 1 bootstrap |
| D. Site + Upstream + TLS setup | Onboarding creates `Site`, `Upstream`, certificate metadata/material, and `TLSConfig` | Persisted state contains `localhost`, `localhost-upstream`, `localhost-tls`, and TLS binding for `localhost` | PASS | Uses existing site/upstream/certificate/tls APIs only |
| E. Compile + apply | Initial revision compiles and applies successfully from snapshot-only state | `rev-000001` created and `apply-rev-000001` completed with `succeeded` / `revision applied` | PASS | Blocking compiler/runtime path defects were fixed during validation |
| F. Active runtime state | Successful apply marks the revision active and emits success event | `revisions.json` shows `current_active_revision_id=rev-000001`; `events.json` contains `apply_started` and `apply_succeeded` | PASS | Runtime stays snapshot-driven |
| G. HTTPS runtime serving | Runtime serves HTTPS on `443` using the generated development certificate | Runtime-internal `wget --no-check-certificate https://localhost/` returned `HTTP/1.1 200 OK` | PASS | Certificate is self-signed dev material produced by the existing WAF issuance flow |
| H. Login after setup | After setup, operator can authenticate normally through local auth | `POST /api/auth/login` succeeded for the bootstrapped admin after apply | PASS | Login path remains separate from onboarding bootstrap |
| I. Audit trail | Critical actions are traceable in `AuditEvent` | Audit shows `auth.bootstrap`, `site.create`, `upstream.create`, `certificate.acme_issue`, `tlsconfig.create`, `revision.compile_request`, `revision.apply_trigger`, `auth.login` | PASS | Actor, resource, status, and related ids are present |
| J. Clean user-facing routing | Clean routes exist without `.html`; legacy `.html` URLs redirect | `GET /login` -> `200`; `GET /dashboard` -> `200`; `GET /login.html` -> `302 /login`; `GET /index.html` -> `302 /dashboard` | PASS | UI now uses clean route surface |
| K. UI lock before setup | Operator cannot use the app shell before initial setup | Before bootstrap, setup state required onboarding; shell logic now checks setup before auth and redirects to `/` | PASS | Removes pre-setup login/shell bypass path |
| L. Apply UX idempotency guard | Initial apply action must not fan out duplicate jobs from repeated clicks | Onboarding UI now disables Apply while request is in flight and shows explicit status | PASS | UI fix; repeated-click job fan-out path removed |
| M. Runtime L4 guard bootstrap | Runtime starts with installed L4 guard rules and without noisy warning logs | Runtime logs are clean; `iptables -S INPUT` and `iptables -S WAF-RUNTIME-L4` show installed jump + `connlimit` + `hashlimit` rules | PASS | Warning noise removed from idempotent checks; baseline rules confirmed |

## Summary

Stage 1 clean first-run now behaves as a product flow:
- empty stack -> onboarding
- onboarding bootstrap creates the first admin
- onboarding creates site, upstream, certificate, and TLS binding
- compile/apply succeeds
- runtime becomes active
- HTTPS is served on `443`
- clean login route and dashboard route are available without `.html`

## Remaining Stage 1 Constraints

- Development certificate flow is still a local WAF-managed development certificate, not real public Let's Encrypt issuance.
- HTTPS verification from Windows CLI tooling may depend on local Schannel behavior; runtime-internal HTTPS serving is verified directly in-container.
- Stage 1 still remains single-node only and baseline-only for anti-DDoS protection.



