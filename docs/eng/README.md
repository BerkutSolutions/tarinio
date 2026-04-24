# English Wiki

This page belongs to the current documentation branch.

This is the main English index for TARINIO documentation. It is structured as an operator wiki for the real `3.0.2` product and is meant to complement the Russian branch, not lag behind it.

## Start Here

- Product overview: `README.en.md`
- Full UI and workflow guide: `docs/eng/core-docs/ui.md`
- Architecture and product boundaries: `docs/eng/architecture-docs/architecture.md`
- API catalog: `docs/eng/core-docs/api.md`

## Core Documents

1. Architecture: `docs/eng/architecture-docs/architecture.md`
2. UI and operator workflows: `docs/eng/core-docs/ui.md`
3. API: `docs/eng/core-docs/api.md`
4. Security: `docs/eng/core-docs/security.md`
5. Deployment: `docs/eng/core-docs/deploy.md`
6. High Availability / multi-node: `docs/eng/high-availability-docs/high-availability.md`
7. Runbook: `docs/eng/core-docs/runbook.md`
8. Upgrade and rollback: `docs/eng/core-docs/upgrade.md`
9. Backups and restore: `docs/eng/core-docs/backups.md`
10. CLI commands: `docs/eng/core-docs/cli-commands.md`
11. WAF env parameter reference: `docs/eng/core-docs/waf-env-reference.md`

## Operator Guides

- Anti-DDoS runbook: `docs/eng/model-docs/anti-ddos-runbook.md`
- Anti-DDoS model: `docs/eng/model-docs/anti-ddos-model.md`
- Security Benchmark Pack: `docs/eng/security-benchmark-pack/README.md`
- High Availability architecture and operations: `docs/eng/high-availability-docs/high-availability.md`
- Runtime L4 guard: `docs/eng/model-docs/runtime-l4-guard.md`
- Runtime filesystem contract: `docs/eng/model-docs/runtime-filesystem-contract.md`
- WAF tuning guide: `docs/eng/operators/waf-tuning-guide.md`
- Stage 1 E2E validation: `docs/eng/high-availability-docs/stage-1-e2e-validation.md`
- OWASP CRS operations: `docs/eng/operators/owasp-crs.md`
- Let's Encrypt DNS-01 operations: `docs/eng/operators/letsencrypt-dns.md`

## What Matters In 3.0.2

- Documentation is aligned with the application version from `control-plane/internal/appmeta/meta.go`.
- The wiki covers the real UI sections: `Dashboard`, `Sites`, `Anti-DDoS`, `OWASP CRS`, `TLS`, `Requests`, `Revisions`, `Events`, `Bans`, `Administration`, `Activity`, `Settings`, and `Profile`.
- The new revision catalog API `GET /core-docs/api/revisions` and revision status cleanup flow are documented.
- Onboarding, login, `2FA`, passkeys, and healthcheck are covered as first-class product flows.

## Recommended Reading Paths

### For First Product Familiarization

1. `README.en.md`
2. `docs/eng/architecture-docs/architecture.md`
3. `docs/eng/core-docs/ui.md`

### For Deployment And Adoption

1. `docs/eng/core-docs/deploy.md`
2. `docs/eng/core-docs/security.md`
3. `docs/eng/core-docs/runbook.md`
4. `docs/eng/core-docs/backups.md`

### For Daily Operations

1. `docs/eng/core-docs/ui.md`
2. `docs/eng/core-docs/runbook.md`
3. `docs/eng/core-docs/api.md`
4. `docs/eng/operators/`

## Source Of Truth

Stage 0 architecture decisions remain the binding foundation:

- `docs/architecture/`

Those documents define product boundaries, revision semantics, compilation, and deployment assumptions.




