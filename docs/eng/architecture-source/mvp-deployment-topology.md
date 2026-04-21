# MVP Deployment Topology

Status: Finalized for Stage 0 task `Define MVP deployment topology`
Date: 2026-03-31

## Purpose

This document defines the single-node deployment topology for the standalone WAF MVP.

It fixes:
- the required runtime components
- storage and artifact boundaries
- network trust boundaries
- the allowed dependency graph between components

This topology is for MVP only.
It does not define multi-node, fleet, SCC-dependent, or Kubernetes-first deployment.

## Topology Summary

The MVP deployment consists of:
- `runtime`
- `control-plane API`
- `worker`
- `PostgreSQL`
- `Redis`
- persistent volumes for product state and deployed artifacts
- internal network boundaries between management and runtime components

The topology is single-node:
- one runtime instance
- one API instance
- one worker instance
- one PostgreSQL instance
- one Redis instance

## High-Level Ownership

- `runtime` owns traffic enforcement only
- `control-plane API` owns source-of-truth state and rollout orchestration
- `worker` executes background control-plane work but does not become a second control plane
- `PostgreSQL` stores persistent product state
- `Redis` stores ephemeral coordination and short-lived runtime-adjacent operational state
- volumes store deployed artifacts and file-backed assets needed by runtime or control-plane workflows

## Component: Runtime

## Responsibility

`runtime` is the data plane based on NGINX + ModSecurity + OWASP CRS.

It is responsible for:
- accepting inbound HTTP/HTTPS traffic
- terminating TLS
- enforcing compiled WAF, access, and rate-limit behavior
- proxying traffic to upstream applications
- exposing runtime health signals
- emitting access and runtime security outputs

## What It Stores

`runtime` stores no source-of-truth product model.

It may access:
- active compiled bundle artifacts
- mounted certificate and key files or references
- runtime logs
- transient process state

These are deployed or transient artifacts, not canonical product state.

## Inputs

`runtime` may receive:
- inbound client traffic
- active compiled revision artifacts
- reload instructions from the control plane
- health-check probes from the control plane

## Outputs

`runtime` may emit:
- proxied traffic behavior
- runtime health/readiness signals
- access logs
- security and apply-related runtime signals

## Allowed Dependencies

`runtime` may depend on:
- deployed artifact volumes
- mounted certificate material or references
- internal control-plane calls required for health/apply orchestration
- protected upstream services on the application side

`runtime` must not depend on:
- PostgreSQL as source-of-truth input
- direct reads of domain entities
- SCC services

## Component: Control-Plane API

## Responsibility

`control-plane API` is the primary management component.

It is responsible for:
- owning the domain model and source-of-truth state
- exposing admin APIs
- authenticating and authorizing operators
- validating user input
- invoking compiler and rollout logic
- creating and updating `Revision`, `Job`, `Event`, and `AuditEvent` state
- selecting the active revision and controlling rollback decisions

## What It Stores

The API owns persistent application state through storage services:
- sites
- upstreams
- TLS metadata
- certificate metadata
- WAF, access, and rate-limit policies
- revisions
- jobs
- events
- users, roles, sessions, and audit records

It may also access:
- bundle storage locations
- certificate storage references

## Inputs

`control-plane API` receives:
- operator requests from UI or API clients
- worker updates and job results
- runtime health and apply feedback
- storage reads from PostgreSQL and Redis

## Outputs

`control-plane API` emits:
- API responses
- compiler requests
- rollout/apply instructions to runtime
- job scheduling requests to worker
- persisted state to PostgreSQL
- ephemeral coordination data to Redis

## Allowed Dependencies

`control-plane API` may depend on:
- PostgreSQL
- Redis
- deployed artifact volume access for revision management
- certificate storage references
- internal connectivity to runtime for apply and health-check
- internal connectivity to worker for background execution

`control-plane API` must not depend on:
- SCC as required identity or storage backend
- runtime files as source of truth

## Component: Worker

## Responsibility

`worker` executes asynchronous tasks on behalf of the control plane.

It is responsible for:
- running queued background jobs
- certificate issuance and renewal work
- cleanup tasks
- compile/apply jobs when executed asynchronously
- reporting job outcomes back to control-plane state

## What It Stores

`worker` does not own persistent product state.

It may use:
- temporary in-process execution state
- Redis-backed job coordination
- access to bundle and certificate storage paths during task execution

All durable results must be written back through control-plane-owned state.

## Inputs

`worker` receives:
- job dispatch or queue signals
- control-plane task parameters
- storage reads needed to execute a job

## Outputs

`worker` emits:
- job status updates
- event-producing failures or success summaries
- artifact generation side effects when performing compiler/apply-related work
- certificate lifecycle results

## Allowed Dependencies

`worker` may depend on:
- PostgreSQL for reading and updating job-related control-plane state
- Redis for queueing, locks, and ephemeral coordination
- bundle storage paths
- certificate storage paths
- internal runtime connectivity only when executing control-plane-authorized operational actions

`worker` must not:
- become an independent API surface
- own source-of-truth state
- make rollout decisions outside control-plane rules
- become a second control plane with divergent logic

## Component: PostgreSQL

## Responsibility

`PostgreSQL` is the primary persistent data store for the control plane.

## What It Stores

Persistent state includes:
- domain entities
- revisions and revision metadata
- job records
- events
- audit events
- users, roles, auth-related state, and sessions as designed for MVP

## Inputs

`PostgreSQL` receives writes from:
- control-plane API
- worker acting on behalf of the control plane

## Outputs

`PostgreSQL` serves reads to:
- control-plane API
- worker

## Allowed Dependencies

`PostgreSQL` should be reachable only from internal management components.

## Component: Redis

## Responsibility

`Redis` is the MVP store for ephemeral coordination and short-lived operational state.

## What It Stores

`Redis` may store:
- job queue state
- locks
- short-lived coordination flags
- short-lived rate-limit support data if needed by the chosen runtime/control-plane design

`Redis` is not the source of truth for the product model.

## Inputs

`Redis` receives writes from:
- control-plane API
- worker

## Outputs

`Redis` serves reads to:
- control-plane API
- worker

## Allowed Dependencies

`Redis` should be reachable only from internal management components.

## Volumes and Storage Boundaries

The MVP requires persistent volumes for:
- compiled revision bundles
- certificate/key material or stable file references
- runtime log destinations where file-backed retention is needed

## Compiled Bundles

Compiled bundles live in a control-plane-managed deployment artifact volume.

That volume stores:
- revision bundle directories
- bundle manifests
- active revision pointer or equivalent activation reference

Ownership rule:
- control plane owns bundle creation, retention, selection, and activation
- runtime consumes only the active bundle

## Certificate and Key References

Certificate and key references live in control-plane-managed certificate storage.

That storage contains:
- file-backed certificate material when locally managed
- stable filesystem references used by compiled runtime configuration
- metadata linkage back to control-plane `Certificate` records

Ownership rule:
- control plane owns certificate metadata and assignment
- runtime only consumes mounted file references required for TLS

## Persistent State vs Deployed Artifacts

Persistent state is:
- PostgreSQL data
- control-plane metadata about revisions, jobs, events, audit, users, and certificates

Deployed artifacts are:
- compiled revision bundles
- active runtime config files
- mounted certificate and key files
- generated error-page assets

Boundary rule:
- persistent state is the source of truth
- deployed artifacts are derived outputs used by runtime
- deleting deployed artifacts must not redefine business truth
- runtime must never be treated as the canonical state holder

## Worker Interaction Model

`worker` interacts with other components as follows:

- with `control-plane API`:
  - receives work definitions
  - updates job progress and results
  - follows control-plane rules for apply and rollback

- with `PostgreSQL`:
  - reads job inputs and entity state needed for execution
  - writes back job results, revision updates, events, and certificate outcomes through control-plane-owned schemas

- with `Redis`:
  - uses queue, lease, and lock semantics for background execution

- with `runtime`:
  - may invoke operational actions only when authorized by control-plane workflow
  - must not invent runtime changes outside a `Revision`-based flow

The worker is an executor, not an authority boundary.

## Network Trust Boundaries

The MVP has two main trust zones:
- external traffic zone
- internal management zone

## External Exposure in MVP

Externally exposed components:
- `runtime` for inbound HTTP/HTTPS traffic
- `control-plane API` only if administrative access is intentionally exposed for operators

Default preference for MVP:
- runtime is externally reachable
- control-plane API should be exposed only to trusted admin networks or local/private access paths

`PostgreSQL`, `Redis`, and `worker` must not be exposed externally.

## Internal-Only Connections

Internal-only connections in MVP:
- `control-plane API -> PostgreSQL`
- `control-plane API -> Redis`
- `control-plane API -> runtime`
- `worker -> PostgreSQL`
- `worker -> Redis`
- `worker -> runtime` only for control-plane-authorized operational tasks

These connections must stay inside the single-node private network boundary.

## Trust Boundary Rules

- client traffic enters through `runtime`
- admin traffic enters through `control-plane API`
- source-of-truth storage is reachable only from management components
- runtime is less trusted for state ownership than the control plane
- no component other than the control plane may be treated as authoritative for product state

## Allowed Dependency Graph

Allowed MVP dependency graph:
- `UI/Admin Client -> Control-Plane API`
- `Control-Plane API -> PostgreSQL`
- `Control-Plane API -> Redis`
- `Control-Plane API -> bundle/cert volumes`
- `Control-Plane API -> runtime`
- `Worker -> PostgreSQL`
- `Worker -> Redis`
- `Worker -> bundle/cert volumes`
- `Worker -> runtime` for operational execution only
- `Runtime -> bundle/cert volumes`
- `Runtime -> upstream applications`

Disallowed MVP dependencies:
- `Runtime -> PostgreSQL` for product model reads
- `Runtime -> Redis` as source-of-truth product control path
- `Runtime -> SCC`
- `Worker -> SCC`
- `API -> SCC` as required runtime dependency

## Resulting Topology Rule

For single-node MVP:
- control-plane API owns state and rollout decisions
- worker executes background tasks but does not become a second control plane
- runtime enforces only deployed artifacts and never owns source of truth
- PostgreSQL stores persistent truth
- Redis stores ephemeral coordination only
- externally exposed surface is minimized to runtime and, when needed, restricted admin API access



