# Migration And Compatibility Contract

This document defines the compatibility rules for the hot/cold logging architecture and secret-management flow introduced in `2.0.5`.

## Backward Compatibility

Existing standalone installations remain supported:

- legacy file-based request archives still work;
- older persisted runtime settings are normalized into the new model;
- missing `OpenSearch` or `Vault` settings must not break startup;
- `ClickHouse` remains a safe cold layer and falls back cleanly when incompletely configured.

The main rule is simple: an upgrade must not turn an existing installation into a half-working environment.

## Backend Cutover Rules

The current routing model is:

1. `file` remains available as the local fallback layer;
2. `OpenSearch` is used as the hot backend;
3. `ClickHouse` is used as the cold backend;
4. if a backend is only partially configured, the product returns to a safe fallback state instead of staying in an undefined middle state.

## Secret Migration Rules

For logging secrets, the order is:

1. the operator selects `vault`;
2. TARINIO validates Vault connectivity and token access;
3. backend secrets are written to Vault;
4. the product verifies that the secret can be read back;
5. local persisted backend credential copies are cleared.

The key principle is:

- successful write and validation first;
- cleanup of the old local copy second.

## Upgrade Safety

The default profile now includes `OpenSearch`, `ClickHouse`, and `Vault` as the normal `2.0.5` posture.

That still needs to remain safe during upgrades:

- `OpenSearch` must not mutate or destroy existing local request archives;
- the local request archive continues to act as the compatibility layer;
- `Vault` in the default profile now bootstraps automatically instead of remaining just a container that still needs manual init/unseal.

## What Counts As A Successful Migration

A migration should be treated as successful only if all of the following are true:

1. the stack starts without manual compose surgery;
2. `OpenSearch`, `ClickHouse`, and `Vault` are healthy;
3. fresh requests are visible in the UI;
4. historical requests remain available through the cold tier;
5. backend secrets no longer survive as open local copies;
6. the local fallback archive is still available as a compatible reserve path.

## What Counts As A Regression

The following should be treated as regressions:

- requiring manual Vault init/unseal on the normal default profile;
- losing access to old request archives after upgrade;
- showing a new backend in the UI while runtime cannot actually use it;
- clearing local secrets before Vault storage is confirmed;
- a state where the backend appears enabled but data is not actually written anywhere.

## Practical Summary

The `2.0.5` contract is:

- the new architecture must enable safely;
- old installs must continue to work;
- logging must degrade predictably;
- the secret flow must migrate without losing access and without leaving unsafe legacy copies behind.
