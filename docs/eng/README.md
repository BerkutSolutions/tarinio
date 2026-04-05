# Berkut Solutions - TARINIO Documentation (EN)

Documentation baseline: `1.0.17`

## Sections

1. Architecture (overview): `docs/eng/architecture.md`
2. API: `docs/eng/api.md`
3. Security: `docs/eng/security.md`
4. Deploy: `docs/eng/deploy.md`
5. Runbook (start and recovery): `docs/eng/runbook.md`
6. Upgrade and rollback: `docs/eng/upgrade.md`
7. Backups: `docs/eng/backups.md`
8. CLI commands: `docs/CLI_COMMANDS.md`
9. Operator docs:
   - Anti-DDoS: `docs/eng/operators/anti-ddos-runbook.md`
   - L4 guard: `docs/eng/operators/runtime-l4-guard.md`
   - Runtime filesystem: `docs/eng/operators/runtime-filesystem-contract.md`
   - WAF tuning: `docs/eng/operators/waf-tuning-guide.md`
   - OWASP CRS: `docs/eng/operators/owasp-crs.md`
   - Let's Encrypt DNS-01: `docs/eng/operators/letsencrypt-dns.md`
   - E2E validation: `docs/eng/operators/stage-1-e2e-validation.md`
10. Stage 0 architecture documents: `docs/architecture/`
11. OSS docs (EN):
   - `docs/eng/oss/SECURITY.md`
   - `docs/eng/oss/CONTRIBUTING.md`
   - `docs/eng/oss/CODE_OF_CONDUCT.md`
   - `docs/eng/oss/SUPPORT.md`

## Release context for 1.0.17

- Product branding unified as `Berkut Solutions - TARINIO`.
- Application version source centralized via `control-plane/internal/appmeta/meta.go`.
- UI and i18n dictionaries aligned for Easy site validation and Anti-DDoS flows.
- Global Anti-DDoS settings added and integrated into the revision apply pipeline.
- Documentation synced with the current standalone WAF runtime model.

## Important

Stage 0 architecture decisions are frozen and are the source of truth:
- `docs/architecture/`






