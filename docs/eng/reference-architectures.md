# Reference Architectures

This page belongs to the current documentation branch.

## Purpose

This page gives operators reference deployment shapes to follow instead of inventing topology from scratch.

## 1. Single-Node Starter

Use when:

- evaluating TARINIO;
- protecting a small number of services;
- HA is not yet required.

Shape:

- one control-plane;
- one runtime;
- bundled PostgreSQL;
- optional local observability.

## 2. HA Control-Plane

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
- runtime

## 3. Enterprise-Style Segmented Deployment

Use when:

- administrative access must be separated;
- data services must stay internal;
- observability is centralized.

Shape:

- dedicated admin boundary;
- internal PostgreSQL and Redis;
- runtime in protected ingress segment;
- Prometheus / Grafana on internal monitoring network.

## 4. Lab / Validation Architecture

Use when:

- validating failover;
- benchmarking;
- testing upgrades and migrations.

Shape:

- `deploy/compose/ha-lab`

This is the recommended rehearsal environment before production promotion.
