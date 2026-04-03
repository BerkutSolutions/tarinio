# Control-Plane Schema Baseline

This directory contains SQL migrations for the standalone WAF control plane.

`0001_schema_baseline.sql` defines the minimal Stage 1 baseline for:
- `revisions`
- `jobs`
- `events`
- `users`
- `roles`
- `audit_events`

Scope rules for this baseline:
- DDL only
- no ORM
- no business logic
- no runtime dependency
- only minimal foreign keys required by the accepted domain model
