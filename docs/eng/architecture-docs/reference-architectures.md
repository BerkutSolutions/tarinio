# Reference Architectures

## Purpose

This page gives operators reference deployment shapes to follow instead of inventing topology from scratch.

## Stack Components By Topology (v1.3.5+)

| Component | Single-Node | HA | Enterprise |
|---|---|---|---|
| `control-plane` | yes | yes (x2) | yes (x2+) |
| `postgresql` | yes | shared | shared |
| `vault` | yes | yes | yes |
| `opensearch` | yes | yes | yes |
| `redis` | no | yes | yes |
| `clickhouse` | no | no | optional |
| `tarinio-sentinel` | optional | optional | yes |

## 1. Single-Node Starter

Use when:

- evaluating TARINIO;
- protecting a small number of services;
- High Availability is not yet required.

Shape:

- one control-plane;
- one runtime;
- bundled PostgreSQL;
- Vault for secrets;
- optional local observability.

## 2. High Availability Control-Plane

Use when:

- administrative availability matters;
- rolling upgrades should avoid API downtime;
- multiple operators need dependable control-plane access.

Shape:

- `api-lb`
- `control-plane-a`
- `control-plane-b`
- shared PostgreSQL
- shared Redis
- Vault
- runtime

## 3. Enterprise-Style Segmented Deployment

Use when:

- administrative access must be separated;
- data services must stay internal;
- observability is centralized.

Shape:

- dedicated admin boundary;
- internal PostgreSQL, Redis, and Vault;
- runtime in protected ingress segment;
- Prometheus / Grafana on internal monitoring network;
- tarinio-sentinel for anomaly detection.

## 4. Lab / Validation Architecture

Use when:

- validating failover;
- benchmarking;
- testing upgrades and migrations.

Shape:

- `deploy/compose/ha-lab`

This is the recommended rehearsal environment before production promotion.
