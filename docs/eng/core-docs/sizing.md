# Sizing Guide

This page belongs to the current documentation branch.

## Purpose

This guide helps operators estimate CPU, memory, and storage needs for TARINIO deployments.

## Small Deployment

Typical shape:

- a few protected services;
- moderate traffic;
- single-node control-plane;
- bundled PostgreSQL.

Starting point:

- `4 vCPU`
- `8 GB RAM`
- fast SSD-backed storage

## Medium Deployment

Typical shape:

- multiple sites and upstreams;
- sustained production traffic;
- separate observability stack;
- stronger operational requirements.

Starting point:

- `8 vCPU`
- `16 GB RAM`
- separate PostgreSQL host or well-sized dedicated volume

## High Availability / Enterprise-Style Deployment

Typical shape:

- two control-plane nodes;
- shared PostgreSQL and Redis;
- observability enabled;
- controlled rolling upgrades and DR planning.

Starting point:

- `2 x control-plane nodes`
- dedicated PostgreSQL
- dedicated Redis
- capacity reserved for Prometheus / Grafana

## Storage Considerations

Plan storage for:

- PostgreSQL state;
- certificate materials;
- runtime artifacts;
- request and event retention;
- backup copies.

## What Drives Resource Growth

The main multipliers are:

- number of protected services;
- request volume;
- event and request log retention;
- CRS usage and policy complexity;
- observability retention;
- High Availability and benchmark workloads.

## Practical Recommendation

Start with conservative headroom, then validate with:

- [Benchmarks](core-docs/benchmarks.md)
- [Observability](core-docs/observability.md)
- your own preproduction traffic profile.


