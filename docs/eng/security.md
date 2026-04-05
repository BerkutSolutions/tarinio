# Security (EN)

Documentation baseline: `1.0.18`

## Production baseline

- `APP_ENV=prod`
- default secrets are forbidden
- HTTPS is required (built-in TLS or trusted reverse-proxy TLS)
- explicitly restrict `BERKUT_SECURITY_TRUSTED_PROXIES`
- enable tamper-evident audit with `BERKUT_AUDIT_SIGNING_KEY`

## Secrets handling

- Keep `.env` outside public repositories.
- Rotate admin and integration secrets on a fixed schedule.
- Never log secret values in tickets or chat.
- Limit host-level access to secret files.

## Network baseline

- Expose only required public ports.
- Restrict admin/control-plane access to trusted networks.
- Keep Docker network isolation enabled.
- Use host firewall + runtime L4 guard together.

## TLS and certificates

- Enforce HTTPS for all operator and site traffic.
- Monitor certificate expiration.
- Test renewal flow before expiry windows.
- Treat failed issuance as an operational incident.

## Access control

- Use named operator accounts only.
- Remove stale admin accounts promptly.
- Review audit events for privileged actions.
- Enforce least privilege on host and CI/CD users.

## Change security

- Every security-relevant change must go through revision compile/apply.
- Avoid manual edits inside runtime filesystem.
- Keep rollback path ready before enforcement changes.

## Incident readiness

- Keep backup/restore process validated.
- Keep a known-good revision ready for emergency rollback.
- Document incident timeline and remediation actions.

## Related documents

- `docs/eng/backups.md`
- `docs/eng/upgrade.md`
- `docs/eng/runbook.md`
