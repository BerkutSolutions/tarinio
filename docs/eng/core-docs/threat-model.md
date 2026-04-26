# Threat Model

This page belongs to the current documentation branch.

Release baseline for this revision: `3.0.6`.

## Purpose

This document explains what TARINIO is designed to help defend, what boundaries exist, and where operator responsibility remains.

## Protected Assets

TARINIO primarily protects:

- application ingress;
- policy-controlled request handling;
- controlled release of traffic protection settings;
- operator visibility into security and rollout events.

In current release, protected operational assets also include:

- revision integrity and rollout safety;
- control-plane state integrity in `PostgreSQL`;
- request/event visibility integrity across `OpenSearch` and optional `ClickHouse`;
- High Availability coordination integrity in `Redis`.

## Main Trust Boundaries

Important boundaries are:

- public client -> runtime ingress;
- runtime -> upstream applications;
- operator -> control-plane;
- control-plane -> `PostgreSQL` / `Redis` / runtime API;
- runtime and control-plane -> `OpenSearch` / `ClickHouse` request backends, when enabled.

## Threat Classes TARINIO Helps Address

- common web attack traffic inspected through WAF/CRS;
- abusive request patterns and brute-force-like traffic;
- L4/L7 hostile traffic patterns controlled by Anti-DDoS features;
- unsafe configuration drift through revision-based change flow.

In current release, TARINIO also helps reduce:

- split-brain risks for compile/apply in High Availability mode through distributed locking;
- integrity drift risks during controlled request-history migration/synchronization flows.

## Shared Responsibility

TARINIO does not remove the need for:

- secure application code;
- network segmentation;
- secret management;
- backup and disaster recovery;
- host and container hardening.

Operators remain responsible for:

- lifecycle and retention tuning of `OpenSearch` and `ClickHouse`;
- backup and recovery drills for every enabled backend (`PostgreSQL`, `Redis`, `OpenSearch`, `ClickHouse`);
- validating data and synchronization outcomes after upgrades and incidents.

## Non-Goals

TARINIO is not, by itself:

- a SIEM;
- a full IAM or directory service;
- an EDR product;
- a replacement for secure application development;
- a global data-mesh/orchestrator for arbitrary external stores.
