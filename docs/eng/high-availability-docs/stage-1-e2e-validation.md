# Stage 1 End-To-End Validation

This document describes the minimum end-to-end validation expected for a working Stage 1 TARINIO deployment.

## Validation Goals

The goal is to confirm that:

- the stack is reachable;
- control-plane works;
- runtime serves traffic;
- revisions compile and apply correctly;
- protection layers are active without breaking baseline traffic.

## Minimum Validation Path

1. Verify `/healthz`.
2. Verify `/login` and `/healthcheck`.
3. Confirm that `GET /core-docs/api/app/meta` returns the expected version.
4. Confirm that at least one site, upstream, and TLS path are working.
5. Compile and apply a revision.
6. Verify that runtime serves the expected host after apply.
7. Confirm that events, requests, and audit data are visible.

## UI Validation Targets

At minimum, validate:

- `Dashboard`
- `Sites`
- `TLS`
- `Revisions`
- `Events`
- `Requests`
- `Activity`
- `Settings`

## Security Validation Targets

- confirm that ModSecurity/CRS is active where expected;
- confirm that Anti-DDoS settings are readable and valid;
- confirm that manual ban/unban flows work;
- confirm that authentication, `2FA`, or passkeys behave as expected for the chosen operator account.

## Failure Signals

Validation should be considered failed if:

- `/healthz` is unstable;
- compile/apply fails;
- runtime does not serve the expected host;
- the UI opens but critical sections cannot load their data;
- operator changes do not show up in revisions, audit, or runtime behavior.

## Related Documents

- `docs/eng/core-docs/runbook.md`
- `docs/eng/core-docs/deploy.md`
- `docs/eng/high-availability-docs/high-availability.md`
- `docs/eng/core-docs/ui.md`


