# English Wiki

This page belongs to the current documentation branch.

This is the main English index for TARINIO documentation. It is structured as an operator wiki for the real `current release` product and is meant to complement the Russian branch, not lag behind it.

## Start Here

- Product overview: `README.en.md`
- Documentation navigator: [Navigator](navigator.md)
- Full UI and workflow guide: [UI](core-docs/ui.md)
- Architecture and product boundaries: [Architecture](architecture-docs/architecture.md)
- API catalog: [API](core-docs/api.md)

## Core Documents

1. [Architecture](architecture-docs/architecture.md)
2. [UI and operator workflows](core-docs/ui.md)
3. [API](core-docs/api.md)
4. [Security](core-docs/security.md)
5. [Deployment](core-docs/deploy.md)
6. [Security Profiles](core-docs/security-profiles.md)
7. [API Positive Security](core-docs/api-positive-security.md)
8. [High Availability / multi-node](high-availability-docs/high-availability.md)
9. [Observability](core-docs/observability.md)
10. [Runbook](core-docs/runbook.md)
11. [Troubleshooting](core-docs/troubleshooting.md)
12. [Upgrade and rollback](core-docs/upgrade.md)
13. [Backups and restore](core-docs/backups.md)
14. [CLI commands](core-docs/cli-commands.md)
15. [WAF env parameter reference](core-docs/waf-env-reference.md)
16. [Logging architecture](architecture-docs/logging-architecture.md)
17. [Secret management](core-docs/secret-management.md)
18. [Migration and compatibility](core-docs/migration-compatibility.md)
19. [Enterprise identity](core-docs/enterprise-identity.md)
20. [Evidence and releases](core-docs/evidence-and-releases.md)

## Enterprise Trust Documents

- [Support And Lifecycle Policy](core-docs/support-lifecycle.md)
- [Enterprise Identity](core-docs/enterprise-identity.md)
- [Evidence And Releases](core-docs/evidence-and-releases.md)
- [Compatibility Matrix](core-docs/compatibility-matrix.md)
- [Sizing Guide](core-docs/sizing.md)
- [Disaster Recovery Guide](core-docs/disaster-recovery.md)
- [Hardening Guide](core-docs/hardening.md)
- [Threat Model](core-docs/threat-model.md)
- [Reference Architectures](architecture-docs/reference-architectures.md)
- [Operations Cookbook](core-docs/cookbook.md)
- [Release Notes Policy](core-docs/release-policy.md)
- [Known Limitations](core-docs/limitations.md)
- [Compliance Mapping](core-docs/compliance-mapping.md)

## Operator Guides

- [Anti-DDoS runbook](model-docs/anti-ddos-runbook.md)
- [Anti-DDoS model](model-docs/anti-ddos-model.md)
- [TARINIO Sentinel](model-docs/tarinio-sentinel.md)
- [Runtime L4 guard](model-docs/runtime-l4-guard.md)
- [Runtime filesystem contract](model-docs/runtime-filesystem-contract.md)
- [WAF tuning guide](operators/waf-tuning-guide.md)
- [Stage 1 E2E validation](high-availability-docs/stage-1-e2e-validation.md)
- [Service protection enterprise validation](high-availability-docs/service-protection-enterprise-validation.md)
- [OWASP CRS operations](operators/owasp-crs.md)
- [Let's Encrypt DNS-01 operations](operators/letsencrypt-dns.md)

## What Matters In current release

- Documentation is aligned with the application version from `control-plane/internal/appmeta/meta.go`.
- The wiki covers the real UI sections: `Dashboard`, `Sites`, `Anti-DDoS`, `OWASP CRS`, `TLS`, `Requests`, `Revisions`, `Events`, `Bans`, `Administration`, `Activity`, `Settings`, and `Profile`.
- The High Availability, observability, and PostgreSQL-backed storage changes are documented as first-class product capabilities.
- Onboarding, login, `2FA`, passkeys, and healthcheck are covered as operator flows.

## Recommended Reading Paths

### For First Product Familiarization

1. `README.en.md`
2. [Architecture](architecture-docs/architecture.md)
3. [UI](core-docs/ui.md)

### For Deployment And Adoption

1. [Deployment](core-docs/deploy.md)
2. [Security](core-docs/security.md)
3. [Runbook](core-docs/runbook.md)
4. [Backups](core-docs/backups.md)
5. [Upgrade](core-docs/upgrade.md)

### For Daily Operations

1. [UI](core-docs/ui.md)
2. [Runbook](core-docs/runbook.md)
3. [API](core-docs/api.md)
4. `operators/`

## Source Of Truth

Stage 0 architecture decisions remain the binding foundation:

- `docs/architecture/`

Those documents define product boundaries, revision semantics, compilation, and deployment assumptions.




