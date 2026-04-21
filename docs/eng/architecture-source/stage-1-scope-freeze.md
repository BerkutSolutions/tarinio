# Stage 1 Scope Freeze

Status: Finalized for Stage 0 task `Freeze Stage 1 scope`
Date: 2026-03-31

## Purpose

This document freezes the Stage 1 MVP scope for the standalone WAF.

Its purpose is to:
- define exactly what is included in MVP
- define exactly what is excluded from MVP
- prevent Stage 1 scope creep
- provide the acceptance boundary for Stage 1 implementation and validation

This scope freeze is binding for Stage 1 planning and execution.

## Scope Rule

Stage 1 must implement a usable single-node standalone WAF MVP.

Stage 1 must not expand into:
- enterprise management
- speculative extensibility
- broader security platform features
- advanced analytics or investigation tooling

## In Scope

The following are in scope for Stage 1 MVP.

## Traffic Handling

- reverse proxy for protected upstream applications
- host-based routing
- simple path handling required by MVP routing behavior

## TLS

- manual certificate upload
- site certificate assignment
- Let's Encrypt HTTP-01 issuance
- automatic certificate renewal jobs
- site TLS enablement and HTTP-to-HTTPS redirect

## WAF and Protection

- ModSecurity integration
- OWASP CRS enablement per site
- per-site WAF enable/disable
- detection/blocking mode selection
- basic custom rules within controlled MVP limits
- basic per-site rate limiting
- site-level IP allow and deny rules
- manual ban and unban support

## Configuration and Runtime Operations

- compiler-generated runtime configuration
- revision-based config apply
- validation before activation
- safe reload
- rollback to last known good revision

## Events, Logs, and Reporting

- normalized security events
- control-plane events for jobs, revisions, and audit
- aggregated reporting from runtime logs
- short-retention raw runtime log files for operational use
- basic reports for:
  - requests count
  - blocked requests
  - top IPs
  - top rules
  - certificate status
  - apply/revision status

## Jobs

- config apply jobs
- certificate issuance jobs
- certificate renewal jobs
- required cleanup and background operational jobs needed by MVP

## Security and Administration

- local auth model
- RBAC baseline
- TOTP-based 2FA baseline
- audit logging for critical admin and operational actions

## UI

- MVP UI structure already fixed in the UI information architecture document
- Dashboard
- Sites
- Policies
- TLS & Certificates
- Events
- Access Control
- Jobs
- Administration

## Deployment Model

- single-node deployment only
- runtime + control-plane API + worker + PostgreSQL + Redis
- local persistent volumes for bundles, certificates, and required runtime artifacts

## Out of Scope

The following are explicitly out of scope for Stage 1 MVP.

## Extensibility and Platform Expansion

- plugin system
- plugin marketplace
- generalized extension framework
- custom policy DSL

## Enterprise and Centralized Management

- enterprise-only features
- cluster deployment
- fleet management
- remote agent model
- centralized multi-runtime control
- SCC-required integration

## Advanced Security Features

- advanced anti-bot system
- country-based rules
- advanced DDoS feature set beyond MVP rate limiting
- advanced policy packs beyond basic MVP settings
- upstream mTLS

## Advanced Data and Investigation

- SIEM integration
- full log search across raw runtime logs
- full-text search platform
- external log pipelines
- advanced analytics
- deep historical investigation workspace
- complex trend analytics beyond basic reports

## Product Surface Expansion

- Pro sections
- enterprise UI pages
- plugin pages
- multi-tenant architecture
- high-scale design work beyond single-node MVP needs

## Explicit Anti-Goals

The project intentionally does not aim for the following in Stage 1, even if they seem attractive:
- feature parity with BunkerWeb
- turning the product into a generic security platform
- exposing raw runtime config as the main operator workflow
- building future-proof plugin abstractions before core workflows work
- adding enterprise management before the standalone product is solid
- expanding reports into a full analytics product
- making UI screens drive backend contracts
- optimizing for scale before single-node operability is proven

## What Counts as a Ready MVP

Stage 1 is considered a ready MVP when the product can do all of the following on a single node:
- create and manage protected sites
- route traffic to upstream applications
- terminate TLS with manual or Let's Encrypt certificates
- renew certificates automatically through jobs
- enable WAF protection with ModSecurity and CRS
- apply basic custom rules
- apply rate limiting
- apply IP allow/deny and manual bans
- safely compile, validate, apply, reload, and roll back runtime configuration
- expose security and operational events in the UI
- show the basic reports defined for MVP
- support local administrative login with RBAC, 2FA, and audit

## Stage 1 Completion Criteria

Stage 1 is complete only when:
- all in-scope capabilities above are implemented
- compiler, validation, apply, and rollback flow work end-to-end
- runtime is driven only by compiled revisions
- UI sections reflect the fixed MVP information architecture
- deployment works in the single-node topology
- events, jobs, certificates, and revision status are visible through the control plane
- MVP reports are available through API and UI
- end-to-end validation confirms:
  - routing works
  - TLS works
  - WAF blocking works
  - rate limiting works
  - apply and rollback work
  - certificate renewal works
  - audit and core operational visibility work

## Link to Domain Model

The Stage 1 scope is bounded by the already accepted domain model.

Stage 1 uses:
- `Site`
- `Upstream`
- `TLSConfig`
- `Certificate`
- `WAFPolicy`
- `AccessPolicy`
- `RateLimitPolicy`
- `Revision`
- `Event`
- `Job`
- `User`
- `Role`
- `AuditEvent`

No Stage 1 feature may require introducing enterprise-only or plugin-oriented entities.

## Link to UI Information Architecture

Stage 1 UI implementation must stay inside the fixed MVP UI IA:
- Dashboard
- Sites
- Policies
- TLS & Certificates
- Events
- Access Control
- Jobs
- Administration

No Stage 1 work may add enterprise navigation, plugin navigation, or extra product areas outside that structure.

## Link to Deployment Topology

Stage 1 implementation is constrained by the fixed MVP topology:
- single-node only
- one runtime
- one control-plane API
- one worker
- one PostgreSQL
- one Redis
- local/internal artifact and certificate storage

No Stage 1 feature may assume:
- multi-node coordination
- fleet rollout
- SCC dependency
- Kubernetes-first deployment

## Resulting Rule

If a proposed Stage 1 feature is not clearly inside `In Scope`, it is out of scope until explicitly approved.

If a proposed change pushes the product toward enterprise management, plugin systems, advanced analytics, or non-MVP complexity, it must be rejected during Stage 1 execution.



