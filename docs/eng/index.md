# English Wiki

This page belongs to the current documentation branch.

This is the main English index for TARINIO documentation. It is structured as an operator wiki for the real `2.0.3` product and is meant to complement the Russian branch, not lag behind it.

## Start Here

- Product overview: `README.en.md`
- Documentation navigator: [Navigator](navigator.md)
- Full UI and workflow guide: [UI](ui.md)
- Architecture and product boundaries: [Architecture](architecture.md)
- API catalog: [API](api.md)

## Core Documents

1. [Architecture](architecture.md)
2. [UI and operator workflows](ui.md)
3. [API](api.md)
4. [Security](security.md)
5. [Deployment](deploy.md)
6. [HA / multi-node](ha.md)
7. [Observability](observability.md)
8. [Benchmarks](benchmarks.md)
9. [Runbook](runbook.md)
10. [Troubleshooting](troubleshooting.md)
11. [Upgrade and rollback](upgrade.md)
12. [Backups and restore](backups.md)
13. [CLI commands](cli-commands.md)

## Enterprise Trust Documents

- [Support And Lifecycle Policy](support-lifecycle.md)
- [Compatibility Matrix](compatibility-matrix.md)
- [Sizing Guide](sizing.md)
- [Disaster Recovery Guide](disaster-recovery.md)
- [Hardening Guide](hardening.md)
- [Threat Model](threat-model.md)
- [Reference Architectures](reference-architectures.md)
- [Operations Cookbook](cookbook.md)
- [Release Notes Policy](release-policy.md)
- [Known Limitations](limitations.md)
- [Compliance Mapping](compliance-mapping.md)

## Operator Guides

- [Anti-DDoS runbook](operators/anti-ddos-runbook.md)
- [Anti-DDoS model](operators/anti-ddos-model.md)
- [Runtime L4 guard](operators/runtime-l4-guard.md)
- [Runtime filesystem contract](operators/runtime-filesystem-contract.md)
- [WAF tuning guide](operators/waf-tuning-guide.md)
- [Stage 1 E2E validation](operators/stage-1-e2e-validation.md)
- [OWASP CRS operations](operators/owasp-crs.md)
- [Let's Encrypt DNS-01 operations](operators/letsencrypt-dns.md)

## What Matters In 2.0.3

- Documentation is aligned with the application version from `control-plane/internal/appmeta/meta.go`.
- The wiki covers the real UI sections: `Dashboard`, `Sites`, `Anti-DDoS`, `OWASP CRS`, `TLS`, `Requests`, `Revisions`, `Events`, `Bans`, `Administration`, `Activity`, `Settings`, and `Profile`.
- The HA, observability, benchmark, and PostgreSQL-backed storage changes are documented as first-class product capabilities.
- Onboarding, login, `2FA`, passkeys, and healthcheck are covered as operator flows.

## Recommended Reading Paths

### For First Product Familiarization

1. `README.en.md`
2. [Architecture](architecture.md)
3. [UI](ui.md)

### For Deployment And Adoption

1. [Deployment](deploy.md)
2. [Security](security.md)
3. [Runbook](runbook.md)
4. [Backups](backups.md)
5. [Upgrade](upgrade.md)

### For Daily Operations

1. [UI](ui.md)
2. [Runbook](runbook.md)
3. [API](api.md)
4. `operators/`

## Source Of Truth

Stage 0 architecture decisions remain the binding foundation:

- `docs/architecture/`

Those documents define product boundaries, revision semantics, compilation, and deployment assumptions.
