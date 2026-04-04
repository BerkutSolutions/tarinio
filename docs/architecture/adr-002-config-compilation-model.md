# ADR-002: Config Compilation Model

Status: Accepted
Date: 2026-03-31

## Context

The standalone WAF MVP stores operator intent as control-plane domain entities.
The runtime must not read those entities directly.

Runtime behavior must be produced through a controlled compilation pipeline so that:
- generated config is deterministic
- apply operations are traceable
- validation happens before activation
- rollback works on versioned runtime snapshots

## Decision

The product uses a compiler-centered configuration pipeline:

`domain model -> compiler -> revision bundle -> validation -> apply -> runtime`

The control plane owns the full pipeline.
The runtime consumes only compiled artifacts from a selected `Revision`.

## Logical Model vs Compiled Artifacts

The logical model is the control-plane source of truth.
It includes:
- `Site`
- `Upstream`
- `TLSConfig`
- `Certificate`
- `WAFPolicy`
- `AccessPolicy`
- `RateLimitPolicy`

The logical model is not loaded directly by NGINX or ModSecurity.

The compiled artifacts are runtime-facing outputs.
They include:
- concrete NGINX config files
- concrete ModSecurity config files
- CRS includes and overrides
- certificate and key references used by runtime
- generated error-page wiring and related runtime includes

Compiled artifacts are derived, versioned outputs.
They are not editable source-of-truth state.

## Pipeline

The pipeline for every runtime-affecting change is:
1. control plane persists the logical model
2. compiler reads the current logical model
3. compiler renders a new revision bundle
4. control plane assigns version, checksum, and revision status
5. control plane validates the bundle before activation
6. control plane atomically applies the validated revision
7. runtime reloads and enforces the selected revision
8. control plane records success, failure, or rollback against that `Revision`

Every config apply must reference exactly one `Revision`.
No apply path may bypass revision creation.

## Bundle Structure

Each compiled `Revision` bundle must have a stable internal structure.

Minimal MVP bundle structure:
- `nginx/nginx.conf`
- `nginx/conf.d/*.conf`
- `nginx/sites/*.conf`
- `modsecurity/modsecurity.conf`
- `modsecurity/sites/*.conf`
- `modsecurity/crs-setup.conf`
- `modsecurity/crs-overrides/*.conf`
- `tls/` references or mounted-file metadata for cert/key material
- `errors/` generated error-page assets when enabled
- `manifest.json`

`manifest.json` must contain at least:
- revision `id`
- revision `version`
- creation timestamp
- checksum
- referenced site ids
- artifact list

## Versioning

Every compiled bundle is represented by a `Revision`.

Minimal `Revision` tracking for MVP:
- `id`: stable internal identifier
- `version`: monotonically increasing applyable revision number
- `status`: `created`, `validated`, `active`, `failed`, `rolled_back`, or equivalent
- `checksum`: integrity fingerprint of the compiled bundle
- timestamps for create, validate, apply, failure, and rollback moments

Versioning rules:
- a new runtime-affecting logical change produces a new `Revision`
- re-apply always targets a specific existing or newly built `Revision`
- rollback always points to a previously known good `Revision`
- runtime does not invent or mutate revision versions

## Atomic Apply

Apply must be atomic at the revision level.

For MVP, atomic apply means:
- the control plane prepares a complete candidate bundle before activation
- validation runs against that complete candidate bundle
- runtime activation switches from one full revision to another full revision
- partial file-by-file live mutation of the active runtime config is prohibited

Allowed MVP approach:
- stage bundle in a candidate location
- validate candidate config
- switch active reference or active directory to the validated revision
- reload runtime against the validated revision

Disallowed approach:
- editing active runtime files in place as the main apply method

## Rollback Strategy

Rollback is revision-based.

If validation fails:
- the candidate `Revision` is marked failed
- it is not activated
- the current active `Revision` remains unchanged

If reload or post-apply health-check fails:
- the candidate `Revision` is marked failed
- control plane re-activates the last known good `Revision`
- rollback is recorded in `Revision`, `Job`, `Event`, and `AuditEvent` history

Rollback never reconstructs config from scratch under pressure.
It reuses an already compiled and previously valid `Revision`.

## Constraints

The following constraints are mandatory:
- runtime never knows the domain model directly
- runtime reads only compiled bundle artifacts
- compiler is the only supported path from logical model to runtime config
- dynamic runtime mutation outside revision apply is prohibited
- plugin systems and ad hoc extension hooks are out of scope for MVP

## Stage 1 Consequences

This decision requires the following Stage 1 order:
1. define compiler inputs and artifact templates
2. implement compiler output as revision bundles
3. implement bundle validation
4. implement atomic apply and revision rollback
5. only then build API and UI flows on top of the compiler-backed model

Restrictions for Stage 1:
- API endpoints must mutate logical entities, not runtime files
- UI screens must edit logical entities, not compiled artifacts
- runtime implementation must assume bundle-driven activation only
- no feature is complete unless it can be rendered into a revision bundle

## Resulting Rule

In MVP, runtime configuration is never hand-authored as the primary product workflow.
It is always the output of:
- logical control-plane state
- compiler render
- revision bundle validation
- revision-based apply


