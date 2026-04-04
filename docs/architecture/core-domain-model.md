# Core Domain Model

Status: Finalized for Stage 0 task `Define core domain model`
Date: 2026-03-31

## Purpose

This document defines the minimal control-plane domain model required for the
standalone WAF MVP.

The model is intentionally small. It includes only the entities needed to:
- manage protected sites
- define runtime behavior
- compile runtime configuration
- operate certificates, jobs, events, users, roles, and audit

This is a control-plane model only.
It is not a runtime state model.

## Modeling Rules

- every entity in this document is owned by the control plane
- runtime process state is not part of the source-of-truth domain model
- runtime files are generated artifacts, not domain entities
- entities must influence runtime only through compiler output where applicable
- no plugin, cluster, fleet, or enterprise-only entities are included
- every config apply must target a specific `Revision`
- runtime must never consume domain entities directly; it consumes compiled artifacts only

## Site

## Responsibility

`Site` is the top-level protected application definition.
It tells the control plane what hostname and routing surface must be exposed by the WAF.

Owner: control plane

## Key Fields

- `id`
- `name`
- `enabled`
- `primary_host`
- `aliases[]`
- `listen_http`
- `listen_https`
- `default_upstream_id`
- `tls_config_id`
- `waf_policy_id`
- `access_policy_id`
- `rate_limit_policy_id`
- `error_page_profile`
- `created_at`
- `updated_at`

## Relationships

- one `Site` references one default `Upstream`
- one `Site` references zero or one `TLSConfig`
- one `Site` references zero or one `WAFPolicy`
- one `Site` references zero or one `AccessPolicy`
- one `Site` references zero or one `RateLimitPolicy`
- one `Site` is referenced by many `Event`
- one `Site` is referenced by many `Job`
- one `Site` is referenced by many `AuditEvent`

## Runtime Impact

`Site` affects compiler output for:
- NGINX `server` blocks
- host matching
- HTTP/HTTPS listener enablement
- upstream routing selection
- policy attachment points
- custom error page wiring

## Upstream

## Responsibility

`Upstream` describes where protected traffic is proxied after the runtime accepts a request.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `name`
- `scheme`
- `host`
- `port`
- `base_path`
- `pass_host_header`
- `connect_timeout_seconds`
- `read_timeout_seconds`
- `send_timeout_seconds`
- `created_at`
- `updated_at`

## Relationships

- many `Upstream` belong to one `Site`
- one `Upstream` may be selected as the site's default upstream
- one `Upstream` may be referenced by `AuditEvent`

## Runtime Impact

`Upstream` affects compiler output for:
- NGINX upstream target definitions
- proxy pass target construction
- proxy timeout directives
- host-header forwarding behavior

## TLSConfig

## Responsibility

`TLSConfig` defines how a site uses TLS and which certificate is attached to it.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `mode`
- `certificate_id`
- `redirect_http_to_https`
- `hsts_enabled`
- `hsts_max_age_seconds`
- `created_at`
- `updated_at`

## Relationships

- one `TLSConfig` belongs to one `Site`
- one `TLSConfig` references zero or one `Certificate`
- one `TLSConfig` may be referenced by `AuditEvent`

## Runtime Impact

`TLSConfig` affects compiler output for:
- TLS listener enablement
- certificate/key file references
- redirect behavior from HTTP to HTTPS
- TLS-related response headers such as HSTS when enabled

## Certificate

## Responsibility

`Certificate` stores certificate lifecycle metadata and runtime file references for a site certificate.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `source`
- `status`
- `common_name`
- `sans[]`
- `not_before`
- `not_after`
- `storage_ref`
- `private_key_ref`
- `last_renewed_at`
- `last_error`
- `created_at`
- `updated_at`

## Relationships

- one `Certificate` may be referenced by one or more `TLSConfig`
- one `Certificate` may be referenced by many `Job`
- one `Certificate` may be referenced by many `Event`
- one `Certificate` may be referenced by many `AuditEvent`

## Runtime Impact

`Certificate` affects compiler output indirectly through `TLSConfig` by providing:
- certificate file reference
- private key file reference

Certificate issuance state itself is control-plane state, not runtime state.

## WAFPolicy

## Responsibility

`WAFPolicy` defines whether WAF inspection is enabled for a site and how ModSecurity/CRS are wired.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `enabled`
- `mode`
- `crs_enabled`
- `anomaly_threshold`
- `body_inspection_enabled`
- `max_request_body_kb`
- `custom_rules_text`
- `created_at`
- `updated_at`

## Relationships

- one `WAFPolicy` belongs to one `Site`
- one `WAFPolicy` may be referenced by many `Event`
- one `WAFPolicy` may be referenced by many `AuditEvent`

## Runtime Impact

`WAFPolicy` affects compiler output for:
- ModSecurity on/off state
- detection vs blocking mode
- CRS include enablement
- CRS threshold tuning
- generated custom rule includes
- request-body inspection settings

## AccessPolicy

## Responsibility

`AccessPolicy` defines simple site-level access control rules for MVP.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `default_action`
- `allow_cidrs[]`
- `deny_cidrs[]`
- `trusted_proxy_cidrs[]`
- `created_at`
- `updated_at`

## Relationships

- one `AccessPolicy` belongs to one `Site`
- one `AccessPolicy` may be referenced by many `Event`
- one `AccessPolicy` may be referenced by many `AuditEvent`

## Runtime Impact

`AccessPolicy` affects compiler output for:
- IP allow rules
- IP deny rules
- trusted proxy handling needed for client IP extraction in MVP scope

## RateLimitPolicy

## Responsibility

`RateLimitPolicy` defines basic per-site request throttling behavior for MVP.

Owner: control plane

## Key Fields

- `id`
- `site_id`
- `enabled`
- `requests`
- `window_seconds`
- `burst`
- `action`
- `status_code`
- `created_at`
- `updated_at`

## Relationships

- one `RateLimitPolicy` belongs to one `Site`
- one `RateLimitPolicy` may be referenced by many `Event`
- one `RateLimitPolicy` may be referenced by many `AuditEvent`

## Runtime Impact

`RateLimitPolicy` affects compiler output for:
- NGINX rate-limit zone settings
- per-site rate-limit enforcement directives
- rejection status behavior such as `429`

## Event

## Responsibility

`Event` stores normalized operational and security events needed by the product.

Owner: control plane

## Key Fields

- `id`
- `type`
- `severity`
- `site_id`
- `source_component`
- `occurred_at`
- `summary`
- `details_json`
- `related_revision`
- `related_job_id`
- `related_certificate_id`
- `related_rule_id`

## Relationships

- many `Event` may belong to one `Site`
- many `Event` may reference one `Revision`
- many `Event` may reference one `Job`
- many `Event` may reference one `Certificate`

## Runtime Impact

`Event` does not directly affect runtime config.
It is derived control-plane state used for visibility, troubleshooting, and reporting.

## Job

## Responsibility

`Job` tracks asynchronous work executed by the control plane.

Owner: control plane

## Key Fields

- `id`
- `type`
- `status`
- `site_id`
- `certificate_id`
- `requested_by_user_id`
- `scheduled_at`
- `started_at`
- `finished_at`
- `result_summary`
- `result_details_json`
- `target_revision`

## Relationships

- many `Job` may belong to one `Site`
- many `Job` may target one `Revision`
- many `Job` may reference one `Certificate`
- many `Job` may be requested by one `User`
- one `Job` may emit many `Event`
- one `Job` may be referenced by many `AuditEvent`

## Runtime Impact

`Job` does not compile directly into runtime config.
It drives control-plane processes such as:
- certificate issuance or renewal
- config compile/apply runs
- cleanup or maintenance actions

## Revision

## Responsibility

`Revision` represents one compiled runtime configuration snapshot produced by the
control plane compiler and tracked through validation, apply, activation, and rollback history.

Owner: control plane

## Key Fields

- `id`
- `version`
- `status`
- `checksum`
- `created_at`
- `validated_at`
- `applied_at`
- `failed_at`
- `rolled_back_at`

## Relationships

- one `Revision` is produced from a set of control-plane entities anchored by `Site`
- one `Revision` may be targeted by many `Job`
- one `Revision` may be referenced by many `Event`
- one `Revision` may be referenced by many `AuditEvent`

## Runtime Impact

`Revision` affects runtime indirectly by being the only unit that may be:
- validated
- applied
- activated
- rolled back

Runtime receives only the compiled artifact set for the selected `Revision`.
Runtime never reads the domain model directly.

## User

## Responsibility

`User` represents a local administrative identity for the standalone product.

Owner: control plane

## Key Fields

- `id`
- `username`
- `email`
- `password_hash`
- `is_active`
- `totp_enabled`
- `last_login_at`
- `created_at`
- `updated_at`

## Relationships

- many `User` may have many `Role`
- one `User` may request many `Job`
- one `User` may produce many `AuditEvent`

## Runtime Impact

`User` does not affect runtime config directly.
It affects who is allowed to mutate control-plane state.

## Role

## Responsibility

`Role` groups permissions for administrative actions in the control plane.

Owner: control plane

## Key Fields

- `id`
- `name`
- `permissions[]`
- `created_at`
- `updated_at`

## Relationships

- many `Role` may be assigned to many `User`
- one `Role` may be referenced by many `AuditEvent`

## Runtime Impact

`Role` does not affect runtime config directly.
It affects authorization for control-plane API and UI actions.

## AuditEvent

## Responsibility

`AuditEvent` records traceable administrative actions and important system changes.

Owner: control plane

## Key Fields

- `id`
- `actor_user_id`
- `action`
- `resource_type`
- `resource_id`
- `site_id`
- `job_id`
- `status`
- `occurred_at`
- `summary`
- `details_json`

## Relationships

- many `AuditEvent` may reference one `User`
- many `AuditEvent` may reference one `Site`
- many `AuditEvent` may reference one `Job`
- many `AuditEvent` may reference one `Revision`
- many `AuditEvent` may reference one domain entity by `resource_type` and `resource_id`

## Runtime Impact

`AuditEvent` does not affect runtime config directly.
It preserves accountability for changes that may later change compiler output or rollout state.

## Control-Plane State vs Runtime State

The following are control-plane state:
- all entities defined in this document
- compiler inputs
- revision history
- revision metadata
- job history
- event history
- audit history
- certificate lifecycle metadata

The following are not control-plane domain entities:
- active NGINX worker process state
- in-memory rate-limit counters
- live ModSecurity transaction state
- open connections
- active TLS sessions
- generated config files on disk
- runtime health probe results as transient process state

Transient runtime state may be observed and normalized into `Event`, but it must
not replace the control-plane domain model.

## Minimal Relationship Summary

- `Site` is the anchor entity for runtime-affecting configuration
- `Upstream`, `TLSConfig`, `WAFPolicy`, `AccessPolicy`, and `RateLimitPolicy` attach runtime behavior to `Site`
- `Certificate` supports `TLSConfig`
- `Revision` is the rollback and apply-history unit for compiled runtime output
- `Job`, `Event`, and `AuditEvent` capture operational lifecycle around changes
- `User` and `Role` govern who may mutate control-plane state

## Domain Model to Runtime Mapping

The runtime path is:
1. control-plane domain model stores operator intent
2. compiler reads `Site` plus attached policy and TLS entities
3. compiler renders concrete artifacts and stores them as a `Revision`
4. control plane validates and applies the generated revision
5. runtime enforces only the compiled result

Concrete mapping in MVP:
- `Site` + `Upstream` -> server blocks, host routing, proxy targets
- `Site` + `TLSConfig` + `Certificate` -> TLS listeners, cert/key references, redirects
- `Site` + `WAFPolicy` -> ModSecurity directives, CRS includes, custom rule includes
- `Site` + `AccessPolicy` -> allow/deny directives, trusted proxy handling
- `Site` + `RateLimitPolicy` -> limit zone and enforcement directives
- `Revision` -> versioned compiled bundle used for validation, apply, activation, and rollback
- `Job`, `Event`, `User`, `Role`, `AuditEvent` -> no direct runtime config, but govern and trace the lifecycle of changes


