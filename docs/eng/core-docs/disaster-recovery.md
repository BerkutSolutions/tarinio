# Disaster Recovery Guide

## Scope

This guide covers recovery planning for node loss, service loss, and restoration from backups.

## Recovery Priorities

Recover in this order:

1. persistent state;
2. control-plane availability;
3. runtime readiness;
4. ingress validation;
5. observability and secondary tooling.

## Single Control-Plane Node Loss

If High Availability exists:

- keep traffic on the remaining node;
- restore the failed node from deployment artifacts;
- rejoin it with the same shared PostgreSQL and Redis.

If single-node:

- restore the host;
- reattach volumes;
- validate `/healthz`, `/login`, and `/healthcheck`.

## PostgreSQL Loss

Primary actions:

- stop unsafe write activity;
- restore PostgreSQL from the latest valid backup;
- verify schema and migrated control-plane state;
- validate the latest known-good revision after recovery.

## Redis Loss In High Availability

Primary actions:

- restore Redis service;
- verify lock and leader behavior resumes cleanly;
- confirm no split-brain compile/apply activity occurred.

## Vault Loss

Primary actions:

- restore Vault from backup or snapshot;
- confirm that mTLS certificate paths (`mtls/<site_id>/client_ca`, `upstream_cert`, `upstream_key`) and logging secret paths are accessible;
- verify that control-plane successfully resolves secrets after restore;
- if Vault was unavailable, confirm that mTLS-enabled sites resume correctly after reconnection.

## Full Host Rebuild

Recommended order:

1. restore deployment artifacts;
2. restore PostgreSQL;
3. restore Redis if used;
4. restore Vault with mTLS certificates and logging secrets;
5. restore runtime and certificate volumes if needed;
6. start services;
7. run smoke validation.

## DR Drill Expectations

Enterprise-style operation should include:

- at least one test restore on a fresh host;
- documented recovery timings;
- documented operator sequence;
- confirmation that backups are actually usable.
