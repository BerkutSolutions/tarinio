# Standalone Product Boundary

Status: Finalized for Stage 0 task `Define standalone product boundary`

## Purpose

This document fixes the product boundary for the standalone self-hosted WAF MVP.
It defines exactly what belongs to the product, how responsibilities are split,
and where future SCC integration may connect without creating an MVP dependency.

This document is normative for Stage 0 and must be used as the basis for:
- ADR-001 runtime/control-plane split
- domain model definition
- deployment topology
- config compiler contract
- rollout and rollback design

## Product Scope

The product is a standalone self-hosted web application firewall that:
- terminates inbound HTTP/HTTPS traffic
- proxies requests to protected upstream applications
- applies WAF inspection through ModSecurity and OWASP CRS
- applies basic access controls and rate limiting
- manages site, policy, certificate, job, event, user, role, and audit data
- provides a local admin UI and API for operators

The product is not an SCC module and does not require SCC to boot, configure,
run, store data, authenticate operators, or apply runtime configuration.

## Core Boundary

The MVP is split into three primary product zones:
- `data plane`: runtime request processing and enforcement
- `control plane`: persistent state, policy management, config compilation, and deployment orchestration
- `UI`: operator-facing interface that reads and mutates state only through the control-plane API

The MVP may also include a worker/job runner inside the control-plane boundary,
but it is not a separate product zone. It exists to execute asynchronous
control-plane work such as certificate renewal and config apply jobs.

## Data Plane

## What runtime does

The data plane is the runtime gateway built on NGINX with ModSecurity and OWASP CRS.
It is responsible for:
- accepting inbound HTTP and HTTPS traffic
- terminating TLS for configured sites
- selecting the target site by host and route rules defined in compiled config
- proxying requests to upstream applications
- enforcing generated WAF behavior through ModSecurity and CRS
- enforcing generated access rules such as allow/deny and rate limits
- serving configured error responses
- exposing runtime health signals needed for safe reload/apply checks
- emitting access logs and WAF/security event outputs produced by request handling

## What runtime does not do

The data plane does not:
- store the source-of-truth site or policy model
- provide operator CRUD APIs for sites, policies, users, or certificates
- decide business policy on its own
- compile high-level product objects into config
- own rollout history or job history as the system of record
- define UI behavior
- depend on SCC services

## Runtime inputs

The data plane consumes only deployed runtime artifacts, specifically:
- generated NGINX configuration
- generated ModSecurity configuration
- generated CRS enablement and overrides
- referenced TLS certificate and key material
- generated access control and rate-limit directives
- generated custom error-page assets when enabled

The data plane must treat these artifacts as read-only deployed input.

## Runtime outputs

The data plane produces operational output only:
- proxied traffic behavior
- runtime process status
- access logs
- WAF/security logs and events
- reload success or failure signal

These outputs may be collected by the control plane, but they are not the data
plane's persistent ownership domain.

## Control Plane

## What control plane does

The control plane is the management system of the product. It is responsible for:
- storing all source-of-truth product state in persistent storage
- exposing authenticated and authorized APIs for operator actions
- managing sites, upstreams, routing definitions, TLS metadata, WAF policies,
  access policies, jobs, events, users, roles, and audit records
- validating operator input before it affects runtime behavior
- compiling stored product state into revisioned runtime bundles
- orchestrating config validation, activation, reload, health-check, and rollback
- scheduling and running background jobs such as certificate renewal and cleanup
- recording revision history, apply results, and audit trail
- ingesting, normalizing, and storing runtime-originated events needed by the product

## What control plane does not do

The control plane does not:
- sit inline on the request path for normal traffic forwarding
- inspect or proxy every request itself
- replace NGINX, ModSecurity, or CRS with a custom enforcement engine
- require SCC for authentication, authorization, audit, or storage in MVP
- allow the UI to bypass API contracts and write directly to storage or runtime files

## Control-plane ownership

The control plane is the sole owner of:
- the source-of-truth domain model
- persisted operator intent
- configuration revision records
- rollout and rollback state
- certificate lifecycle state
- user, role, session, and audit state
- normalized event and job state stored by the product

No other product zone may mutate these records directly.

## UI

## What UI does

The UI is an admin client for operators. It is responsible for:
- displaying current control-plane state
- collecting operator input for configuration changes
- invoking control-plane APIs to create, update, delete, and trigger actions
- showing rollout status, job status, events, certificate health, and audit history
- reflecting backend validation errors and permission outcomes

## What UI does not do

The UI does not:
- own or persist the source-of-truth configuration model
- compile runtime config
- write NGINX, ModSecurity, CRS, or certificate files
- apply runtime changes directly
- enforce security by hiding controls instead of relying on backend checks
- define backend behavior by implication

The UI is a client, not a control-plane substitute.

## Configuration Ownership and Lifecycle

## Who owns configuration

The control plane owns configuration in source form.

Source form means product-level objects such as:
- Site
- Upstream
- TLSConfig / certificate assignment metadata
- WAFPolicy
- AccessPolicy
- rate-limit settings
- custom error-page assignments

This source form is the only operator-editable configuration model.

The data plane does not own source configuration.
The UI does not own source configuration.

## Who compiles configuration

The control plane compiles configuration.

Compilation means transforming persisted product objects into a revisioned runtime
bundle that contains:
- concrete NGINX config files
- concrete ModSecurity config files
- CRS includes and overrides
- runtime-facing site routing layout
- access-control and rate-limit directives
- references to the certificate material to be mounted or deployed

Compilation is a backend responsibility and must not be implemented in the UI or runtime.

## Who applies configuration

The control plane applies configuration.

Apply means:
- selecting a compiled revision
- placing the revision into the active runtime location
- triggering runtime validation and reload flow
- checking post-reload health
- recording success or failure
- initiating rollback when activation is not healthy

The runtime executes the loaded configuration after apply, but it does not decide
what revision becomes active.

## Operational Hand-off

The boundary between control plane and data plane is a deployment hand-off:
- control plane produces a validated revision bundle
- control plane activates that bundle for the runtime
- runtime loads the bundle and enforces it on traffic

The runtime must not reinterpret operator intent beyond what is expressed in the
compiled bundle.

## Allowed Cross-Boundary Flows

Allowed flows in MVP are:
- `UI -> Control Plane API`: operator reads state and requests changes
- `Control Plane -> Storage`: persist source-of-truth data and revision metadata
- `Control Plane -> Compiler`: generate runtime bundle from persisted state
- `Control Plane -> Runtime`: validate, activate, reload, health-check, rollback
- `Runtime -> Control Plane`: send status, logs, and event data required by the product
- `Runtime -> Upstream Apps`: proxy protected application traffic

Disallowed flows in MVP are:
- `UI -> Runtime` direct mutation
- `UI -> Storage` direct writes
- `Runtime -> Storage` direct ownership of product model
- `Runtime -> SCC` required dependency
- `UI -> SCC` required dependency
- `Control Plane -> SCC` required dependency for core product workflows

## Storage Boundary

Persistent product data belongs to the control plane and is stored in MVP backing services.

Persistent data includes:
- sites and upstream definitions
- policies and policy settings
- certificate metadata and issuance state
- jobs, events, and audit records
- users, roles, permissions, sessions, and 2FA state
- active and historical config revision metadata

Runtime-generated deployed files may exist on disk or in mounted volumes, but
they are deployment artifacts derived from control-plane state, not the primary
source of truth.

## SCC Future Boundary

SCC integration is a future boundary only.

In MVP:
- SCC is not required to install the product
- SCC is not required to authenticate operators
- SCC is not required to store configuration
- SCC is not required to collect audit or event data
- SCC is not required to trigger rollouts

Future SCC integration may later consume or exchange:
- identity and trust context
- audit exports
- security event forwarding
- deep links or administrative navigation hand-offs
- fleet or centralized management hooks in a later stage

These future seams must remain optional and must not alter the standalone
runtime/control-plane/UI split defined in this document.

## Explicit Non-Goals for MVP

The MVP explicitly does not include:
- multi-node cluster management
- centralized fleet orchestration
- remote agent architecture
- SCC-dependent login or SCC-dependent authorization
- SCC as a required storage, audit, or event backend
- a custom traffic-processing engine instead of NGINX
- a custom WAF engine instead of ModSecurity + CRS
- direct operator editing of raw NGINX or ModSecurity config as the main workflow
- plugin frameworks or generalized extension systems
- enterprise policy distribution across multiple runtimes
- active-active config consensus between multiple control-plane nodes
- UI-first behavior that defines backend contracts after the fact
- runtime or UI implementation before compiler and boundary decisions are fixed

## Concrete MVP Boundary Summary

For MVP, the product boundary is:
- one standalone product with its own runtime, control plane, storage, and admin UI
- one source-of-truth configuration owner: the control plane
- one configuration compiler owner: the control plane
- one configuration apply owner: the control plane
- one traffic enforcement owner: the data plane runtime
- one operator interaction surface: the UI through the control-plane API

Any implementation decision that violates this ownership model is outside the
approved Stage 0 architecture.
