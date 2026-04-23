# English Wiki

This page belongs to the current documentation branch.

This is the main English index for TARINIO documentation. It is structured as an operator wiki for the real `2.0.8` product and is meant to complement the Russian branch, not lag behind it.

## Start Here

- Product overview: `README.en.md`
- Full UI and workflow guide: `docs/eng/ui.md`
- Architecture and product boundaries: `docs/eng/architecture.md`
- API catalog: `docs/eng/api.md`

## Core Documents

1. Architecture: `docs/eng/architecture.md`
2. UI and operator workflows: `docs/eng/ui.md`
3. API: `docs/eng/api.md`
4. Security: `docs/eng/security.md`
5. Deployment: `docs/eng/deploy.md`
6. HA / multi-node: `docs/eng/ha.md`
7. Runbook: `docs/eng/runbook.md`
8. Upgrade and rollback: `docs/eng/upgrade.md`
9. Backups and restore: `docs/eng/backups.md`
10. CLI commands: `docs/CLI_COMMANDS.md`
11. WAF env parameter reference: `docs/eng/waf-env-reference.md`

## Operator Guides

- Anti-DDoS runbook: `docs/eng/operators/anti-ddos-runbook.md`
- Anti-DDoS model: `docs/eng/operators/anti-ddos-model.md`
- HA architecture and operations: `docs/eng/ha.md`
- Runtime L4 guard: `docs/eng/operators/runtime-l4-guard.md`
- Runtime filesystem contract: `docs/eng/operators/runtime-filesystem-contract.md`
- WAF tuning guide: `docs/eng/operators/waf-tuning-guide.md`
- Stage 1 E2E validation: `docs/eng/operators/stage-1-e2e-validation.md`
- OWASP CRS operations: `docs/eng/operators/owasp-crs.md`
- Let's Encrypt DNS-01 operations: `docs/eng/operators/letsencrypt-dns.md`

## What Matters In 2.0.8

- Documentation is aligned with the application version from `control-plane/internal/appmeta/meta.go`.
- The wiki covers the real UI sections: `Dashboard`, `Sites`, `Anti-DDoS`, `OWASP CRS`, `TLS`, `Requests`, `Revisions`, `Events`, `Bans`, `Administration`, `Activity`, `Settings`, and `Profile`.
- The new revision catalog API `GET /api/revisions` and revision status cleanup flow are documented.
- Onboarding, login, `2FA`, passkeys, and healthcheck are covered as first-class product flows.

## Recommended Reading Paths

### For First Product Familiarization

1. `README.en.md`
2. `docs/eng/architecture.md`
3. `docs/eng/ui.md`

### For Deployment And Adoption

1. `docs/eng/deploy.md`
2. `docs/eng/security.md`
3. `docs/eng/runbook.md`
4. `docs/eng/backups.md`

### For Daily Operations

1. `docs/eng/ui.md`
2. `docs/eng/runbook.md`
3. `docs/eng/api.md`
4. `docs/eng/operators/`

## Source Of Truth

Stage 0 architecture decisions remain the binding foundation:

- `docs/architecture/`

Those documents define product boundaries, revision semantics, compilation, and deployment assumptions.
