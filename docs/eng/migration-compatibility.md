# Migration And Compatibility Contract

This document defines the `2.0.10` compatibility contract for logging, secret handling, and standalone-to-enterprise cutover.

## Backward Compatibility

Existing standalone installs remain supported:

- older runtime settings are normalized into the current model;
- missing optional backends do not break startup;
- legacy request archives remain readable until migration completes;
- cleanup only happens after the target backend has been verified.

The core rule is simple: an upgrade must not leave the installation in a half-working state.

## Request Archive Migration Rules

For request data, the migration order is:

1. keep the local legacy archive intact;
2. import legacy day files into the active backend;
3. validate that the expected records exist in that backend;
4. remove only the verified legacy file;
5. persist migration state.

This applies to:

- `2.0.5 -> 2.0.10` standalone upgrades where request history lived in local `*.jsonl` archives;
- enterprise setups that still need to move validated data from hot to cold tiers.

## Backend Cutover Rules

The effective request-routing model in `2.0.10` is:

1. `OpenSearch` is the primary request backend when enabled;
2. `ClickHouse` is the optional cold tier for enterprise history;
3. the local archive remains a spool/fallback layer, not the primary UI query source;
4. if a backend is only partially configured, the product falls back safely instead of pretending the backend is active.

## Secret Migration Rules

For logging secrets, the order is:

1. the operator enables `vault`;
2. TARINIO validates Vault connectivity and token access;
3. backend secrets are written to Vault;
4. the product verifies that the secret can be read back;
5. legacy local secret copies are cleared.

The invariant is:

- write and validate first;
- clean up the old copy second.

## Default And Enterprise Profiles

### Default

The normal `2.0.10` default profile is:

- `PostgreSQL`
- `OpenSearch`
- `Vault`
- no mandatory `Redis`
- no mandatory `ClickHouse`

### Enterprise

The enterprise profile adds:

- multi-node control-plane
- `Redis` for HA coordination
- `ClickHouse` for cold analytics

Upgrades must remain safe in both modes.

## What Counts As A Successful Migration

A migration is successful only if all of the following are true:

1. the stack starts without manual compose surgery;
2. fresh requests are visible through the API/UI;
3. legacy request files are migrated before deletion;
4. backend secrets no longer remain as unsafe local copies after Vault migration;
5. standalone still has a safe spool/fallback path if the backend is temporarily unavailable;
6. enterprise still preserves validated hot-to-cold history movement.

## What Counts As A Regression

The following are regressions:

- deleting legacy request archives before backend validation;
- showing a backend as active while runtime cannot actually use it;
- making the UI depend on local request files instead of the API/backend path;
- clearing local secrets before Vault storage is confirmed;
- making standalone require enterprise-only dependencies just to stay functional.

