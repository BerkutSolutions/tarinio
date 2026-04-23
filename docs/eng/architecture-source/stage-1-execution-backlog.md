# Stage 1 Execution Backlog

Status: Finalized for Stage 0 task `Convert Stage 1 into execution backlog`
Date: 2026-03-31

## Purpose

This document converts the frozen Stage 1 MVP scope into an execution-ready backlog.

Mandatory implementation order:
1. config compiler
2. validation and apply pipeline
3. runtime wiring
4. control-plane API
5. UI

## Ordered Backlog

### 1. Config Compiler

`S1-01` Create module directory structure
- Depends on: Stage 0 complete
- DoD: stable directories exist for runtime, compiler, control plane, worker, UI, deploy, and docs

`S1-02` Add revision manifest schema and bundle metadata contract
- Depends on: `S1-01`
- DoD: manifest contract exists for revision id, version, checksum, timestamps, and artifact list

`S1-03` Add compiler template asset layout
- Depends on: `S1-01`, `S1-02`
- DoD: template tree exists for nginx, site includes, ModSecurity, CRS overrides, TLS refs, and error assets

`S1-04` Implement compiler mapping for Site and Upstream
- Depends on: `S1-03`
- DoD: compiler renders listeners, host routing, and upstream targets from `Site` and `Upstream`

`S1-05` Implement compiler mapping for TLSConfig and Certificate references
- Depends on: `S1-03`
- DoD: compiler renders TLS listeners, redirects, and cert/key refs from `TLSConfig` and `Certificate`

`S1-06` Implement compiler mapping for WAFPolicy
- Depends on: `S1-03`
- DoD: compiler renders ModSecurity enablement, mode, CRS wiring, and basic custom rule includes

`S1-07` Implement compiler mapping for AccessPolicy and RateLimitPolicy
- Depends on: `S1-03`
- DoD: compiler renders allow/deny, trusted proxy handling, and rate-limit directives

`S1-08` Implement revision bundle assembly
- Depends on: `S1-04`, `S1-05`, `S1-06`, `S1-07`
- DoD: compiler outputs one complete revision bundle plus manifest from the accepted domain model

### 2. Validation and Apply Pipeline

`S1-09` Add revision persistence and lifecycle states
- Depends on: `S1-08`
- DoD: revision records support created, validated, active, failed, and rolled-back states

`S1-10` Implement bundle integrity and structure validation
- Depends on: `S1-08`, `S1-09`
- DoD: invalid or incomplete bundles fail before activation

`S1-11` Implement runtime syntax validation runner
- Depends on: `S1-10`
- DoD: generated config can be syntax-tested before activation and results are attached to the revision

`S1-12` Implement candidate staging mechanism
- Depends on: `S1-11`
- DoD: validated bundles are staged without mutating the active revision

`S1-13` Implement atomic activate switch
- Depends on: `S1-12`
- DoD: active revision can be switched atomically from one full bundle to another

`S1-14` Implement reload and post-apply health-check execution
- Depends on: `S1-13`
- DoD: reload and health-check outcomes are recorded for each apply attempt

`S1-15` Implement automatic rollback to last known good revision
- Depends on: `S1-14`
- DoD: failed apply attempts automatically restore the most recent successful revision

### 3. Runtime Wiring

`S1-16` Build NGINX runtime image with ModSecurity and CRS
- Depends on: `S1-15`
- DoD: runtime image exists and is compatible with the accepted bundle contract

`S1-17` Align runtime filesystem layout with revision bundle contract
- Depends on: `S1-16`
- DoD: runtime can read active bundles and certificate refs without direct domain-model reads

`S1-18` Add runtime health and readiness endpoints
- Depends on: `S1-16`, `S1-17`
- DoD: runtime exposes health signals required by ADR-003

`S1-19` Wire runtime support for custom error pages and rate-limit includes
- Depends on: `S1-17`
- DoD: compiled error assets and rate-limit directives execute without manual runtime edits

### 4. Control-Plane API and Worker

`S1-20` Bootstrap control-plane service skeleton
- Depends on: `S1-15`, `S1-01`
- DoD: service starts with config loading, routing shell, and storage wiring points

`S1-21` Add database schema baseline
- Depends on: `S1-20`
- DoD: baseline schema exists for accepted control-plane entities including `Revision`, `Job`, `Event`, `User`, `Role`, and `AuditEvent`

`S1-22` Add Redis coordination baseline
- Depends on: `S1-20`
- DoD: API and worker can use Redis for queues, locks, and ephemeral coordination

`S1-23` Implement Site CRUD API
- Depends on: `S1-21`
- DoD: `Site` lifecycle works through validated API endpoints backed by persistence

`S1-24` Implement Upstream CRUD API
- Depends on: `S1-21`, `S1-23`
- DoD: `Upstream` can be managed and linked to sites through API contracts

`S1-25` Implement TLS config and certificate metadata API
- Depends on: `S1-21`, `S1-23`
- DoD: `TLSConfig` and `Certificate` metadata can be created, updated, and queried

`S1-26` Implement manual certificate upload flow
- Depends on: `S1-25`
- DoD: manual cert upload stores refs correctly and can feed revision compilation

`S1-27` Implement Let's Encrypt issuance and renewal flow
- Depends on: `S1-25`, `S1-22`
- DoD: issuance and renewal jobs update certificate state and operator-visible outcomes

`S1-28` Implement WAFPolicy CRUD API
- Depends on: `S1-21`, `S1-23`, `S1-08`
- DoD: WAF policy changes persist and compile into revision output

`S1-29` Implement AccessPolicy CRUD API
- Depends on: `S1-21`, `S1-23`, `S1-08`
- DoD: access policy changes persist and compile into revision output

`S1-30` Implement manual ban and unban API
- Depends on: `S1-29`
- DoD: manual ban/unban changes are stored in control-plane state and reach runtime only through compilation

`S1-31` Implement RateLimitPolicy CRUD API
- Depends on: `S1-21`, `S1-23`, `S1-08`
- DoD: rate-limit policy changes persist and compile into revision output

`S1-32` Implement revision compile request API
- Depends on: `S1-23`, `S1-24`, `S1-25`, `S1-28`, `S1-29`, `S1-31`
- DoD: control plane can create a new revision bundle from persisted domain state

`S1-33` Implement apply job runner and revision lifecycle orchestration
- Depends on: `S1-22`, `S1-32`, `S1-15`
- DoD: apply runs through `Job` records and the accepted ADR-003 sequence

`S1-34` Implement Event ingestion for security and apply outcomes
- Depends on: `S1-33`, `S1-18`
- DoD: runtime security outputs and rollout outcomes appear as normalized `Event` records

`S1-35` Implement event retention and reporting backend
- Depends on: `S1-34`
- DoD: backend serves agreed MVP report summaries without storing raw access-log streams in PostgreSQL

`S1-36` Implement auth baseline with RBAC and TOTP 2FA
- Depends on: `S1-21`
- DoD: control-plane endpoints are protected by local auth, role checks, and TOTP-based 2FA

`S1-37` Implement audit logging for critical actions
- Depends on: `S1-23`, `S1-25`, `S1-28`, `S1-29`, `S1-31`, `S1-33`, `S1-36`
- DoD: critical control-plane actions produce traceable `AuditEvent` records

### 5. UI

`S1-38` Build MVP admin shell
- Depends on: `S1-36`, `S1-37`
- DoD: UI shell exposes the fixed MVP navigation from the accepted UI IA

`S1-39` Build Sites screens
- Depends on: `S1-38`, `S1-23`, `S1-24`, `S1-32`
- DoD: operators can list, create, edit, and inspect sites and upstream linkage

`S1-40` Build Policies screens
- Depends on: `S1-38`, `S1-28`, `S1-29`, `S1-30`, `S1-31`
- DoD: operators can manage WAF, access, and rate-limit settings through the UI

`S1-41` Build TLS and certificate screens
- Depends on: `S1-38`, `S1-25`, `S1-26`, `S1-27`
- DoD: operators can inspect cert status and manage site TLS through the UI

`S1-42` Build Events screen
- Depends on: `S1-38`, `S1-34`
- DoD: UI shows recent normalized events with basic filters

`S1-43` Build Dashboard screen
- Depends on: `S1-38`, `S1-35`
- DoD: dashboard shows request, block, top IP, top rule, cert, and revision summaries

`S1-44` Build Jobs screen
- Depends on: `S1-38`, `S1-33`
- DoD: UI shows recent jobs, statuses, timestamps, and linked revision or certificate context

`S1-45` Build Administration screens
- Depends on: `S1-38`, `S1-36`, `S1-37`
- DoD: operators can manage users and roles and inspect audit history

### 6. Deploy, Docs, and Validation

`S1-46` Create single-node Docker Compose environment
- Depends on: `S1-18`, `S1-20`, `S1-22`, `S1-33`
- DoD: full MVP stack starts locally with the accepted topology and internal networking

`S1-47` Write operator quickstart
- Depends on: `S1-39`, `S1-41`, `S1-46`
- DoD: a new operator can deploy the stack and protect a first site using the docs

`S1-48` Write MVP WAF tuning guide
- Depends on: `S1-40`, `S1-42`
- DoD: operators have a practical guide for MVP WAF tuning and false-positive handling

`S1-49` Execute Stage 1 end-to-end validation
- Depends on: `S1-45`, `S1-46`, `S1-47`, `S1-48`
- DoD: validated scenarios cover routing, TLS, WAF blocking, rate limiting, jobs, reports, and rollback

## First Implementation Task

Start with `S1-01 Create module directory structure`.



