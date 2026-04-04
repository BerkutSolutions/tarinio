# TARINIO Architecture (overview)

Documentation baseline: `1.0.10`

TARINIO is a standalone self-hosted WAF built around NGINX + ModSecurity + OWASP CRS.

## Core principles

- The source of truth for operator intent is the control-plane (storage + revisions).
- Runtime is not edited manually; it consumes only the active compiled bundle.
- All changes go through revisions: compile → validate → apply → rollback.

## Stage 0 source of truth

The full architecture doc set (binding input):
- `docs/architecture/adr-001-runtime-control-plane-split.md`
- `docs/architecture/adr-002-config-compilation-model.md`
- `docs/architecture/adr-003-config-rollout-and-rollback.md`
- `docs/architecture/core-domain-model.md`
- `docs/architecture/mvp-deployment-topology.md`
- `docs/architecture/mvp-ui-information-architecture.md`




