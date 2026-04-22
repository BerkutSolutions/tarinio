# Logging Architecture

This document explains how TARINIO `2.0.5` stores requests, events, and activity data, which backends are used by default, and what an operator should treat as the normal production posture.

## Overview

The `2.0.5` release introduces a tiered logging model:

- `OpenSearch` is the hot storage layer.
- `ClickHouse` is the cold storage layer.
- the local file archive remains a compatibility and diagnostic fallback.
- `PostgreSQL` continues to store product state such as users, roles, revisions, settings, and control-plane metadata.

The goal is to solve two different problems at the same time:

- fast investigation of recent incidents;
- long-term history without overloading the hot tier.

## What Is Stored Where

### Control-plane state

`PostgreSQL` stores:

- users;
- roles and RBAC;
- revisions and related metadata;
- runtime settings;
- control-plane state;
- product entities that are not request telemetry streams.

`PostgreSQL` is not the primary request-log backend.

### Requests

The runtime processes incoming requests as follows:

1. it tails the NGINX access log;
2. normalizes each record into a request entry;
3. keeps a local JSONL archive as fallback;
4. writes recent data to `OpenSearch`;
5. keeps long-term history in `ClickHouse`.

In practice this means:

- recent requests usually come from `OpenSearch`;
- older history should come from `ClickHouse`;
- the local archive exists for safety, compatibility, and diagnostics.

### Events and Activity

The `Events` and `Activity` pages now follow the same hot/cold tier model.

Operators can see:

- which hot backend is active;
- which cold backend is active;
- which retention window is configured;
- which indexes are available for cleanup.

## Backend Roles

### OpenSearch

`OpenSearch` is used for:

- fast search over recent data;
- investigation workflows;
- alert-oriented access patterns;
- quick filtering by IP, URI, host, status, and similar fields.

It is the operational hot tier.

### ClickHouse

`ClickHouse` is used for:

- long-term history;
- reports and aggregations;
- lower-cost storage for large volumes;
- antifraud, behavioral analytics, and ML-oriented pipelines.

It complements the hot tier rather than replacing it.

### Local archive

The local archive is not the primary enterprise log store.

It exists for:

- backward compatibility;
- graceful degradation;
- local investigations and exports;
- safe migration from older standalone installs.

## Default Behavior In `2.0.5`

The normal default model is:

- hot backend: `OpenSearch`;
- cold backend: `ClickHouse`;
- integration secrets: `Vault`;
- local archive: enabled as fallback.

The standalone `deploy/compose/default` profile is expected to bring up that full stack without leaving the operator in a half-configured state.

## Retention Policy

### Hot retention

The hot tier is capped at:

- `30` days maximum.

This protects single-node and medium-sized environments from turning the hot layer into long-term storage.

### Cold retention

The cold tier is capped at:

- `730` days maximum.

This window is intended for historical investigations and reporting.

## What Operators Should Expect

If the system is healthy, operators should see the following:

1. recent requests appear quickly and remain searchable;
2. older requests do not disappear after they age out of the hot window;
3. the UI shows active hot and cold tiers;
4. index cleanup is available for `requests`, `events`, and `activity`;
5. if an external backend is unavailable, the product still retains a compatible fallback path.

## What Counts As Normal

The expected `2.0.5` behavior is:

1. `OpenSearch` is healthy and accepts new records.
2. `ClickHouse` is healthy and available for historical data.
3. `Vault` is healthy and used for integration secrets.
4. `Requests` shows new traffic without silent gaps.
5. old data remains available through the cold tier.
6. operators can inspect and clean indexes by stream.

## What To Verify After Upgrade

After upgrading to `2.0.5`, verify that:

1. `OpenSearch`, `ClickHouse`, and `Vault` are healthy;
2. the logging settings show `OpenSearch` as hot and `ClickHouse` as cold;
3. hot retention does not exceed `30` days;
4. cold retention does not exceed `730` days;
5. the `Requests` page shows fresh traffic;
6. index cleanup works for the intended stream.

## Degradation And Resilience

If `OpenSearch` is unavailable:

- hot search degrades;
- the local archive and cold tier remain sources of data;
- the product should not lose history just because the hot backend is down.

If `ClickHouse` is unavailable:

- recent investigations still work;
- long-term history and reporting degrade;
- the local archive remains a safety net.

If both external backends are unavailable:

- the standalone deployment should still preserve request visibility through the local archive;
- the archive remains the last compatibility and diagnostics layer.

## Practical Summary

For operators, the simple rule is:

- `OpenSearch` is for recent operational work;
- `ClickHouse` is for long-term history;
- `Vault` protects integration secrets;
- the local archive is a fallback, not the main log database.

If the product behaves that way, the documentation and the actual `2.0.5` release behavior are aligned.
