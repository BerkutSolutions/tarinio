# Known Limitations and Product Boundaries

This page belongs to the current documentation branch.

Release baseline for this revision: `3.0.4`.

## Why This Page Exists

Precise limitations increase trust more than vague marketing claims.

## Product Boundaries

TARINIO is:

- a self-hosted traffic protection and control platform;
- a revision-driven WAF and Anti-DDoS control surface;
- an operator-facing runtime and control-plane product.

TARINIO is not:

- a SIEM;
- a directory service;
- an application security testing suite;
- a universal replacement for every edge component.

## Data Stores In Scope

Current product-aligned data services:

- `PostgreSQL` for control-plane state (users, roles, revisions, settings, metadata);
- `Redis` for High Availability coordination (locks, leader-only jobs);
- `OpenSearch` as the primary request/event backend when enabled;
- `ClickHouse` as the optional cold-tier backend for long-term request analytics.

Important boundary:

- `PostgreSQL` is not the request-log analytics database;
- local request archive files are a compatibility spool/fallback path, not the primary system of record.

## Synchronization Boundaries

The product provides controlled synchronization flows:

- revision compile/apply with distributed serialization in High Availability mode;
- synchronization between easy profile and policy entities through control-plane workflows;
- controlled migration/synchronization of legacy local request archives into `OpenSearch` and optionally into `ClickHouse`;
- optional hot-to-cold request history flow (`OpenSearch` -> `ClickHouse`) in enterprise deployments.

The product does not provide:

- cross-region global active-active data replication orchestration;
- external SIEM data pipeline ownership beyond documented integrations;
- arbitrary third-party database synchronization outside documented backends and flows.

## Operational Boundaries

Operators still need:

- proper network design;
- backup and DR ownership;
- secure application code;
- secret management;
- monitoring and response procedures.

Operators are also responsible for:

- sizing and retention policies for `OpenSearch` and `ClickHouse`;
- backup and restore validation for every enabled backend;
- confirming synchronization health after upgrades or topology changes.

## Documentation Boundaries

The official docs focus on documented Docker-based product flows and should not be interpreted as universal deployment support for every platform variation.
