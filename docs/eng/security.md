# Security

This page belongs to the current documentation branch.

This document defines the minimum production baseline and the main secure-operation practices for TARINIO.

## Security Baseline

- `APP_ENV=prod`
- default secrets are unacceptable;
- control-plane access should be restricted to trusted networks;
- administrative access should run over HTTPS;
- trusted proxies must be configured explicitly;
- audit and secret storage must be protected at the platform level.

## Authentication And Access

In version `2.0.10`, the platform supports:

- session-based login;
- enterprise login through `OIDC`;
- `SCIM` provisioning with external group mappings;
- `2FA` through TOTP;
- passkeys for login and as a second factor;
- server-side permission checks on every endpoint.

Important boundary:

- directory-backed `LDAP/AD` group mapping is supported when groups are projected into `OIDC` or `SCIM`;
- standalone direct `LDAP` password login is not part of `2.0.10`.

Recommendations:

- use named operator accounts only;
- enable a second factor for privileged operators;
- remove unused passkeys;
- review the roles and permissions granted to operators regularly.
- keep login brute-force protection enabled in `Settings -> Security`.

## Secrets And Sensitive Data

- do not store `.env` in public repositories;
- keep secrets in a dedicated protected store;
- restrict access to keys and certificate materials;
- do not paste EAB values, DNS API tokens, or private keys into tickets or chats.
- never rely on development defaults for `CONTROL_PLANE_SECURITY_PEPPER`, `WAF_RUNTIME_API_TOKEN`, and `POSTGRES_PASSWORD`.

## Network Model

- publish only the ports you actually need;
- separate runtime ingress from administrative access;
- restrict PostgreSQL and internal network access;
- use host firewall controls together with the runtime L4 guard.

## TLS And Certificates

Recommended baseline:

- use HTTPS for the administrative boundary;
- monitor certificate expiration dates;
- test certificate renewal flows before they are needed;
- treat exported certificate archives as sensitive artifacts;
- document which mode is in use: import, self-signed, ACME `http-01`, or ACME `dns-01`.
- keep Vault TLS verification enabled; insecure skip-verify mode should be disabled unless a temporary emergency exception is approved.

## Change Safety

- do not edit runtime manually;
- push risk-sensitive changes through compile/apply;
- keep a known-good revision available before tightening protection;
- validate `Dashboard`, `Events`, `Requests`, and `Bans` after every security-impacting change.

## Audit And Investigation

Use the following for investigation:

- `Activity` for administrative actions;
- `Events` for security detections;
- `Requests` for request-level detail;
- `Revisions` to correlate incidents with rollout history.

For evidence-grade export:

- use the signed support bundle from `Administration -> Enterprise`;
- archive the generated release manifest, signature, `SBOM`, and provenance for promoted builds.

## Backup And Restore As A Security Capability

Secure operation also requires:

- regular backups;
- at least one off-host copy;
- a tested restore drill;
- a clear rollback path for failed or unsafe changes.

## Related Documents

- `docs/eng/backups.md`
- `docs/eng/runbook.md`
- `docs/eng/upgrade.md`
- `docs/eng/ui.md`
- `docs/eng/enterprise-identity.md`
- `docs/eng/evidence-and-releases.md`

