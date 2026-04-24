# Support and Lifecycle Policy

This page belongs to the current documentation branch.

TARINIO `3.0.0` uses explicit support channels so enterprise operations are not based on implicit assumptions.

## Release Channels

- `Current`: latest release, full functional and security fixes.
- `Stable`: previous minor line, bug and security fixes only.
- `LTS`: designated long-lived line for conservative production programs.

## Support Windows

For `3.0.0`, the support windows are:

- `Current`: from April 23, 2026 until the next minor release.
- `Stable` (`latest-1`): 180 days after the next minor release is published.
- `LTS 2.0`: until April 30, 2027 (security and critical resilience fixes only).

## SLA Targets

Initial response targets:

- `P1` (control-plane outage or active risk of losing protected traffic): within 1 hour.
- `P2` (critical security-function degradation without full outage): within 4 hours.
- `P3` (non-critical functional defect): within 1 business day.
- `P4` (advisory and improvement requests): within 3 business days.

## In-Scope Support

- incident analysis for documented deployment topologies;
- safe-upgrade and rollback guidance;
- migration and post-upgrade validation support;
- verification guidance for release artifacts and evidence bundles.

## Out-Of-Scope Support

- undocumented topologies and manual runtime mutations outside revision workflows;
- arbitrary leapfrog upgrades across multiple lines without staged validation;
- obsolete releases outside declared support windows.

## Supported Operational Profile

Support assumes:

- documented Docker / Docker Compose deployment patterns;
- documented upgrade flow (`install-aio` or equivalent documented sequence);
- backup before upgrade;
- post-upgrade smoke validation (`/healthz`, `/core-docs/api/app/meta`, login, compile/apply).

## Operator Responsibilities

Operators must:

- maintain and rehearse a rollback point;
- keep change records with product version and active revision;
- archive `release-manifest.json`, `signature.json`, `sbom.cdx.json`, and `provenance.json` for each promoted build;
- run restore validation at least once per month.

