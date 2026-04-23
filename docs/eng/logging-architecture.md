# Logging Architecture

This document describes the real request-log behavior in TARINIO `2.0.7`.

## Overview

The request path is now split into three roles:

- `OpenSearch` is the primary request backend for reads and writes.
- `ClickHouse` is the optional cold tier for enterprise and long-term analytics.
- the local JSONL archive is no longer a user-facing database; it is a transient compatibility spool and recovery path.

`PostgreSQL` continues to store product state such as users, roles, revisions, settings, and control-plane metadata. It is not the request-log database.

## Request Flow

For each request, runtime:

1. tails the NGINX access log;
2. normalizes the line into a stable request record;
3. writes the record to the configured backend path;
4. keeps a local JSONL copy as a short-lived spool/fallback;
5. serves `GET /api/requests` from the backend path first, not from local files.

In practice:

- standalone default uses `OpenSearch` as the primary source of truth for request reads;
- enterprise can use `OpenSearch` for hot reads and `ClickHouse` for older history;
- the UI calls the API and no longer reads request archive files directly through `nginx`.

## Local Archive Role

The local archive still exists, but its role changed.

It is kept for:

- backward-compatible upgrades from older standalone installs;
- short-lived buffering if the backend is temporarily unavailable;
- controlled migration of legacy `*.jsonl` archives into `OpenSearch` or `ClickHouse`;
- operator diagnostics and recovery.

It is not intended to remain the main query path for the `Requests` page.

## Upgrade Behavior From Older Installs

When a node upgrades from an older build that already has local request archives:

1. runtime scans legacy day files from the archive spool;
2. imports them into the active backend;
3. validates that the imported records are actually present in the backend;
4. removes only the verified legacy file;
5. records the migration state to avoid unsafe partial cleanup.

This keeps the `2.0.5 -> 2.0.7` transition safe:

- old request history is not dropped before verification;
- standalone installs move toward `OpenSearch` as the primary store;
- enterprise installs can still continue hot-to-cold migration into `ClickHouse`.

## Default And Enterprise Modes

### Default

The normal `deploy/compose/default` posture is:

- `PostgreSQL`
- `OpenSearch`
- `Vault`
- `runtime`
- `ui`
- `ddos-model`

In this mode:

- `OpenSearch` is both the operational read backend and the default request store;
- the local archive is only a spool/fallback layer.

### Enterprise

The `deploy/compose/enterprise` posture adds:

- multi-node control-plane
- `Redis` for HA coordination
- `ClickHouse` for cold analytics

In this mode:

- `OpenSearch` remains the hot path;
- `ClickHouse` becomes the cold historical tier;
- legacy local archives can still be ingested safely before cleanup.

## What Operators Should Expect

Healthy `2.0.7` behavior looks like this:

1. fresh requests appear in `Requests` through the API without direct file reads;
2. `OpenSearch` receives recent traffic;
3. old legacy archive days are migrated and removed only after validation;
4. if `ClickHouse` is enabled, older hot data can be moved there without losing validated records;
5. the local archive remains available as a fallback spool rather than the main database.
