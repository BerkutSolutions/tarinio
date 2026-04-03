# Logging and Reporting Model

Status: Finalized for Stage 0 task `Define logging and reporting model`
Date: 2026-03-31

## Purpose

This document defines the MVP model for:
- access logs
- security events
- control-plane events
- report aggregates

It keeps logging and reporting small enough for MVP while preserving:
- operational visibility
- security visibility
- rollout traceability
- basic UI reporting needs

## Core Rule

The MVP has three distinct data flows:
- `access logs` from runtime traffic handling
- `security events` from WAF, rate limiting, and access-control enforcement
- `control-plane events` from jobs, revisions, and audit activity

These flows must not be mixed into one undifferentiated storage stream.

## Flow 1: Access Logs

## Purpose

Access logs are high-volume runtime traffic records.
They are used for traffic counting and limited aggregate reporting, not as the primary event model.

## Source

Source: `runtime`

Specifically:
- NGINX access logging for accepted and proxied requests
- request outcomes needed for request and status aggregation

## Format

MVP format:
- structured line-based runtime log format
- one record per request
- fields sufficient for aggregation such as timestamp, site/host, client IP, method, path, status, bytes, upstream status, and latency

## Storage

Primary storage:
- runtime log files on file-backed storage defined by deployment topology

Optional derived storage:
- aggregate counters or report tables in PostgreSQL created by the control plane

## Who Reads It

Readers:
- control plane ingestion/reporting path
- operator reports in UI through control-plane API

The UI does not read raw runtime log files directly.

## Retention

MVP retention:
- raw access logs kept for short operational retention only
- aggregate report data kept longer than raw log files

Practical MVP rule:
- raw access logs are file-retained for short local history
- PostgreSQL stores derived aggregates, not full raw access log streams

## What Does Not Go to PostgreSQL

Full raw per-request access logs do not go to PostgreSQL for MVP.

Reason:
- they are high-volume
- they would make the main database grow too quickly
- MVP reporting needs aggregates, not full request-level searchable history

## What Is Aggregated

From access logs, the control plane may derive:
- requests count
- requests by site
- requests by status class
- top IPs by request volume

## Flow 2: Security Events

## Purpose

Security events capture actionable enforcement outcomes that matter to operators.

These include:
- WAF blocks or detections
- rate-limit hits
- access policy denies
- ban/unban-related enforcement outcomes when applicable in MVP

## Source

Source: `runtime`

Specifically:
- ModSecurity outputs
- runtime-enforced access control results
- runtime-enforced rate-limit results

## Format

MVP format has two layers:
- raw runtime log/output format at the runtime side
- normalized `Event` records at the control-plane side

Normalized `Event` fields should include at least:
- type
- severity
- site reference
- source component
- occurred_at
- summary
- related rule id when available
- related client IP when available
- details payload for bounded structured context

## Storage

Primary operational storage for MVP:
- normalized security `Event` records in PostgreSQL

Raw runtime-side security logs may still exist as files, but PostgreSQL stores only normalized events and bounded context, not unlimited raw log streams.

## Who Reads It

Readers:
- control-plane API
- UI events and reports views
- internal job/reporting logic

## Retention

MVP retention:
- normalized security events retained in PostgreSQL for practical operator history
- raw runtime security logs retained only as short local file history if enabled

## How Runtime Logs Become Event

Runtime-originated security outputs are transformed into control-plane `Event` records through an ingestion path:
1. runtime writes security-related outputs
2. control-plane ingestion logic or worker reads the relevant runtime outputs
3. ingestion normalizes them into bounded `Event` records
4. normalized `Event` records are stored in PostgreSQL

Normalization rule:
- one actionable security occurrence may become one `Event`
- ingestion must extract only fields needed for product workflows and UI
- ingestion must not dump unbounded raw runtime payloads into PostgreSQL

## What Is Aggregated

From security events, the control plane may derive:
- blocked requests count
- top blocking IPs
- top triggered rules
- blocked requests by site
- blocked requests by reason category

## Flow 3: Control-Plane Events

## Purpose

Control-plane events track management and rollout lifecycle.

They include:
- `Job` lifecycle
- `Revision` lifecycle
- `AuditEvent`

## Source

Source: `control-plane API` and `worker`

Specifically:
- job execution state transitions
- revision render/validate/apply/rollback lifecycle
- operator-triggered administrative actions

## Format

MVP format:
- structured control-plane records stored as first-class entities

This flow uses:
- `Job`
- `Revision`
- `AuditEvent`
- related `Event` records for lifecycle signaling where needed

## Storage

Primary storage:
- PostgreSQL

## Who Reads It

Readers:
- control-plane API
- UI job, rollout, certificate, and audit views
- internal operational logic

## Retention

MVP retention:
- keep these records in PostgreSQL because they are low-volume and operationally important

## Boundary: Raw vs Normalized vs Aggregated

The MVP uses three storage levels:

1. raw runtime logs
2. normalized control-plane events
3. aggregate report data

## Raw Runtime Logs

Stored as files:
- access logs
- raw runtime security outputs

Used for:
- short retention troubleshooting
- ingestion into normalized events or aggregates

Not used as the primary UI data model.

## Normalized Control-Plane Records

Stored in PostgreSQL:
- security `Event`
- `Job`
- `Revision`
- `AuditEvent`
- bounded certificate status data

Used for:
- UI operational visibility
- filtering by recent important events
- audit and apply history

## Aggregate Report Data

Stored in PostgreSQL as report-oriented aggregates.

Used for:
- dashboard/report summaries
- top-N views
- trend-lite reporting for MVP

## What Must Not Go to PostgreSQL

The following must not be stored as full raw streams in PostgreSQL for MVP:
- full per-request access logs
- unbounded raw ModSecurity payload logs
- unlimited raw runtime debug output
- full-text copies of all runtime log files

Only normalized and bounded operationally useful records should enter PostgreSQL.

## Minimal Reports for MVP

The MVP report set includes:
- `requests count`
- `blocked requests`
- `top IPs`
- `top rules`
- `cert status`
- `apply/revision status`

## Report Definitions

`requests count`
- source: access-log aggregates
- shape: total requests over selected recent window, optionally by site

`blocked requests`
- source: normalized security events
- shape: total blocked requests over selected recent window, optionally by site

`top IPs`
- source: access-log aggregates and/or security-event aggregates
- shape: top client IPs by request volume or block volume

`top rules`
- source: normalized security events
- shape: top WAF rule ids or categories by hit/block count

`cert status`
- source: certificate metadata plus job/event outcomes
- shape: active, expiring, failed issuance, failed renewal

`apply/revision status`
- source: `Revision`, `Job`, and related control-plane events
- shape: latest apply attempts, active revision, failed revisions, rollback outcomes

## UI Data Needs

The UI needs:
- recent security event list
- recent apply/revision status list
- certificate status summary
- request and blocked-request summary counts
- top IPs summary
- top rules summary
- audit visibility for critical actions

The UI does not need:
- direct raw access-log browsing as an MVP requirement
- full-text search across runtime logs
- external SIEM-style correlation

## Required API Endpoints

The control plane will need MVP endpoints such as:
- `GET /api/reports/traffic-summary`
- `GET /api/reports/security-summary`
- `GET /api/reports/top-ips`
- `GET /api/reports/top-rules`
- `GET /api/reports/cert-status`
- `GET /api/reports/revisions`
- `GET /api/events`
- `GET /api/jobs`
- `GET /api/audit`

Exact route naming may change, but the API contract must cover these data shapes.

## Link to Deployment Topology

Per the single-node deployment topology:
- runtime writes raw logs to runtime-accessible file-backed storage
- control-plane ingestion reads from runtime outputs or runtime-managed log locations
- normalized events and aggregates are written into PostgreSQL
- worker may participate in ingestion and aggregation jobs under control-plane ownership

## Ingestion Path in MVP

The expected MVP path is:
1. runtime emits access logs and security outputs
2. control plane or worker reads those runtime outputs from local/internal paths
3. access logs are aggregated into report data
4. security outputs are normalized into `Event`
5. control-plane lifecycle changes are written directly as `Job`, `Revision`, `AuditEvent`, and related `Event`
6. UI reads only through control-plane API

## Resulting Rule

For MVP:
- raw runtime logs stay mostly in files
- PostgreSQL stores normalized events and bounded aggregates
- reports are aggregate-first, not raw-log-first
- UI reads control-plane data only
- no SIEM, no external log pipeline, and no full-text runtime log platform are introduced
