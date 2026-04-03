# Revision Manifest Contract

Status: MVP contract for `S1-02`

## Purpose

This file defines the minimal manifest structure for one compiled `Revision` bundle.

It is the concrete Stage 1 implementation contract for:
- the `Revision` domain entity
- ADR-002 bundle structure
- validation and apply flow used later by ADR-003

## File Location

Each compiled bundle must include:
- `manifest.json`

The manifest JSON must follow:
- [manifest.schema.json](/C:/Trash/Development/WAF/compiler/bundles/manifest.schema.json)

## Required Fields

- `schema_version`
  Meaning: manifest format version for MVP. Fixed to `v1`.

- `revision_id`
  Meaning: control-plane `Revision.id`.

- `revision_version`
  Meaning: control-plane `Revision.version`.

- `created_at`
  Meaning: bundle creation timestamp from the control plane.

- `bundle_checksum`
  Meaning: integrity checksum for the whole compiled bundle.

- `contents`
  Meaning: artifact list included in the bundle.

## Bundle Contents Entry

Each entry in `contents` must include:
- `path`
  Relative path of the artifact inside the bundle.

- `kind`
  Allowed MVP values:
  - `nginx_config`
  - `modsecurity_config`
  - `crs_config`
  - `tls_ref`
  - `error_asset`

- `checksum`
  Integrity checksum for the individual artifact.

## Relation to Domain Model

This manifest maps directly to the accepted `Revision` entity:
- `revision_id` -> `Revision.id`
- `revision_version` -> `Revision.version`
- `bundle_checksum` -> `Revision.checksum`
- `created_at` -> `Revision.created_at`

The manifest does not contain raw domain entities such as `Site`, `WAFPolicy`, or `AccessPolicy`.
Those are compiler inputs only.
The manifest describes compiled artifacts only.

## Relation to ADR-002

This contract reflects the accepted bundle model from ADR-002:
- compiled bundle is the runtime-facing output
- runtime reads compiled artifacts only
- `manifest.json` is part of the bundle
- validation happens against the bundle before apply

The manifest is intentionally minimal for MVP.
It does not add future extension hooks or non-MVP metadata.

## Apply Usage

During apply, the manifest is used to:
- identify which `Revision` is being staged or activated
- confirm the expected bundle version
- confirm the expected bundle checksum
- enumerate the expected artifacts inside the candidate bundle

The apply flow must treat the manifest as the index of the candidate bundle, not as a source of domain truth.

## Validation Usage

During validation, the control plane must check:
- `schema_version` is supported
- `revision_id` is present and matches the target revision record
- `revision_version` is present and matches the target revision record
- `created_at` is present
- `bundle_checksum` is present
- every `contents` entry has path, kind, and checksum
- every listed artifact exists in the bundle
- every listed artifact checksum matches
- bundle structure is consistent with the accepted ADR-002 layout

Validation must fail if:
- the manifest is missing
- the manifest is malformed
- revision identity does not match
- a listed artifact is missing
- a checksum does not match
- an unexpected partial bundle is detected

## MVP Boundaries

For MVP, the manifest does not include:
- plugin metadata
- multi-node deployment metadata
- fleet rollout data
- SCC integration data
- runtime-generated state

It exists only to support bundle integrity, revision identity, and safe apply/validation.
