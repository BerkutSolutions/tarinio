# Architecture

This page belongs to the current documentation branch.

TARINIO is a self-hosted web traffic protection and control platform where the control-plane defines the desired state and the runtime executes only compiled and applied configuration artifacts.

## Architectural Model

In practical terms, the product consists of four main layers:

- `UI`: operator interface, onboarding, login, healthcheck.
- `Control-plane`: API, RBAC, state storage, audit, revision catalog, and change orchestration.
- `Compiler / revision pipeline`: translates operator intent into deterministic bundles.
- `Runtime`: NGINX + ModSecurity + OWASP CRS + the enforcement mechanisms around them.

## Core Principles

- The source of truth for operator intent lives in the control-plane.
- Runtime is not treated as a human-editable source of truth.
- Every configuration change must be expressible as a revision.
- Rollout and rollback are normal lifecycle operations, not emergency exceptions.
- Permissions are checked server-side on every endpoint.

## Key Domains In 2.0.5

The main architectural domains in version `2.0.5` are:

- sites and upstreams;
- TLS and certificate materials;
- access, WAF, and rate-limit policies;
- easy site profiles as the high-level application configuration layer;
- Anti-DDoS;
- revisions, snapshots, and apply jobs;
- audit and observability data;
- runtime settings and retention policy.

## Change Lifecycle

The standard change path is:

1. The operator updates entities through UI, API, or CLI.
2. The control-plane stores the desired state.
3. A revision is created through compile.
4. The revision is validated and applied into runtime.
5. The platform keeps apply history and rollback capability.

This is why the `Revisions` section is a central product capability rather than a simple history page.

## Why Revisions Matter

In `2.0.5`, the revision subsystem includes:

- an aggregated revision catalog by service;
- active, pending, and failed classification;
- a status/event timeline for rollout;
- deletion of inactive revisions;
- persistence of the revision’s last apply result even after timeline cleanup.

That makes change management reproducible and operationally useful.

## Runtime Boundary

Runtime is responsible for:

- traffic termination and proxying;
- executing NGINX configuration;
- ModSecurity and OWASP CRS enforcement;
- applying Anti-DDoS and low-level protective limits;
- request and security event logging.

Control-plane is responsible for:

- storing operator intent;
- authentication and authorization;
- compiling and publishing configuration;
- audit and API;
- the operator UI.

## The UI As An Architectural Layer

In `2.0.5`, the UI is not a thin shell over a few backend objects. It reflects multiple operational layers:

- operational overview through `Dashboard`;
- configuration through `Sites`, `TLS`, `Anti-DDoS`, and `OWASP CRS`;
- observability through `Requests`, `Events`, `Bans`, and `Activity`;
- change management through `Revisions`;
- platform-level workflows through `Settings`, `Profile`, and `Administration`.

## Stage 0 Source Of Truth

The mandatory architecture foundation remains in:

- `docs/architecture/adr-001-runtime-control-plane-split.md`
- `docs/architecture/adr-002-config-compilation-model.md`
- `docs/architecture/adr-003-config-rollout-and-rollback.md`
- `docs/architecture/core-domain-model.md`
- `docs/architecture/mvp-deployment-topology.md`
- `docs/architecture/mvp-ui-information-architecture.md`

Those documents define the foundation, while this wiki maps that foundation to the real product surface in `2.0.5`.
