# Threat Model

Release baseline for this revision: `1.3.5`.

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
- High Availability coordination integrity in `Redis`;
- mTLS certificates (client CAs and keys) stored in `Vault`.

## Main Trust Boundaries

Important boundaries are:

- public client -> runtime ingress;
- runtime -> upstream applications (optionally with mutual mTLS authentication);
- operator -> control-plane;
- control-plane -> `PostgreSQL` / `Redis` / runtime API;
- runtime and control-plane -> `OpenSearch` / `ClickHouse` request backends, when enabled;
- control-plane -> `Vault` (mTLS certificates and logging secret resolution).

## Threat Classes TARINIO Helps Address

- common web attack traffic inspected through WAF/CRS;
- abusive request patterns and brute-force-like traffic;
- L4/L7 hostile traffic patterns controlled by Anti-DDoS features;
- unsafe configuration drift through revision-based change flow;
- credential stuffing attacks detected via 401 responses on configured auth paths;
- bot abuse via antibot-fail signal and JA3/JA4 TLS fingerprint correlation;
- known malicious TLS clients via JA3 blacklist (Shodan/Masscan/Metasploit/Cobalt Strike);
- known vulnerability exploitation via Virtual Patching (ModSecurity SecRule);
- WebSocket-layer attacks (frame inspection, pattern blocking, rate limiting);
- HTTP Request Smuggling / CL+TE desync via strict HTTP parsing;
- upstream channel compromise via mTLS (incoming and outgoing).

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
- mTLS certificate lifecycle management (CA rotation, client key rotation via Vault);
- validating data and synchronization outcomes after upgrades and incidents.

## Non-Goals

TARINIO is not, by itself:

- a SIEM;
- a full IAM or directory service;
- an EDR product;
- a replacement for secure application development;
- a global data-mesh/orchestrator for arbitrary external stores.
