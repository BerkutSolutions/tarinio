# Security Benchmark Pack

This section defines the reproducible security validation pack for TARINIO `3.0.2`.

## Purpose

- validate protection quality against realistic attack traffic;
- measure false positives on normal traffic;
- prove rollout safety before production promotion.

## Scenario Matrix

1. Human baseline traffic (normal browser/API behavior).
2. Scanner and recon traffic (`/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`).
3. Brute-force and auth abuse (`/login`, token endpoints).
4. L7 flood and burst patterns.
5. Mixed production-like traffic with legitimate peaks.

## Core Metrics

- `false_positive_rate` by scenario and profile;
- precision/recall where labels are available;
- latency impact (`p95`, `p99`) versus baseline;
- `cpu` and `memory` under load;
- challenge completion and bypass stability for anti-bot.

## Evidence Artifacts

- scenario manifest and replay inputs;
- raw metric exports and summary reports;
- revision/result correlation for compile/apply checks;
- signed release evidence (`release-manifest`, `signature`, `sbom`, `provenance`).

## Pass/Fail Criteria

- `false_positive_rate` stays within the approved profile baseline;
- labeled attack scenarios do not regress recall against the previous release;
- `p95`/`p99` latency regression stays within the approved SLO window;
- `cpu`/`memory` stay inside the benchmark environment budget.

## 3.0.2 Linkage

This benchmark pack is the validation base for:

- security profiles;
- API positive security;
- dual-layer anti-bot challenge controls;
- Sentinel explainability improvements;
- threat-intel and geo-context extensions.
