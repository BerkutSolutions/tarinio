# Hardening Guide

This page belongs to the current documentation branch.

## Goal

This guide defines the recommended production hardening baseline beyond the minimal deployment instructions.

## Administrative Boundary

Recommended baseline:

- restrict control-plane access to trusted networks;
- publish admin endpoints behind HTTPS;
- avoid direct broad internet exposure of the admin UI;
- require named operator accounts and second factors.

## Secret Management

Recommended baseline:

- keep secrets outside public repositories;
- rotate bootstrap secrets before production use;
- treat certificate exports and API tokens as sensitive material;
- use a protected secret store when possible.

## Network Segmentation

Recommended baseline:

- separate public ingress from administrative access;
- restrict PostgreSQL and Redis to internal networks only;
- use host firewall rules together with TARINIO protections;
- expose only the ports required for your topology.

## Data Services

Recommended baseline:

- persistent storage for PostgreSQL and runtime artifacts;
- controlled backup access;
- backup retention with at least one off-host copy;
- restore drills, not only backup creation.

## Change Safety

Recommended baseline:

- all meaningful policy changes go through compile/apply;
- keep rollback-ready revisions;
- use batch-safe changes for large provisioning;
- validate after every security-sensitive rollout.

## Observability

Recommended baseline:

- enable metrics and dashboards;
- monitor revision failures, runtime readiness, and lock contention;
- keep enough retention to analyze incidents and releases.
