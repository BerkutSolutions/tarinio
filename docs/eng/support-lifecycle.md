# Support and Lifecycle Policy

This page belongs to the current documentation branch.

## Release Philosophy

TARINIO follows a forward-moving release model:

- the newest release becomes the supported product baseline;
- upgrades are expected to go through `scripts/install-aio.sh`;
- upgrades are protected by backups, post-upgrade checks, and data-safe migration paths;
- the project does not maintain a long tail of concurrent legacy release lines.

## What This Means Operationally

For operators, the practical rule is:

- run the newest validated release you are prepared to operate;
- do not plan for long-term production stay on outdated builds;
- treat upgrade readiness and rollback safety as part of normal operations.

## Supported Deployment Assumption

The support baseline assumes:

- deployment through the documented Docker / Docker Compose flows;
- upgrade through the AIO installer or equivalent documented sequence;
- backups before upgrade;
- no manual mutation of runtime artifacts outside the product workflows.

## Upgrade Safety Baseline

TARINIO `2.0.2` expects upgrades to be safe because:

- the AIO installer takes lightweight backups before upgrade;
- PostgreSQL-backed state migrations are versioned;
- legacy state migration is non-destructive;
- post-upgrade smoke validation can be enforced;
- HA control-plane upgrades can be validated through the rolling upgrade helpers.

## Operator Responsibilities

Operators are expected to:

- keep deployment artifacts aligned with the current release;
- run the documented backup and smoke checks;
- verify resource headroom before upgrading;
- maintain a known-good rollback point.

## What Is Not Promised

This policy does not promise:

- indefinite parallel support for old releases;
- arbitrary upgrade paths across many skipped versions without operator validation;
- support for undocumented deployment layouts.

## Recommended Enterprise Practice

For enterprise-style usage:

- promote through lab -> preprod -> prod;
- keep the HA lab as a rehearsal environment;
- treat upgrade validation as mandatory change control;
- archive release notes and benchmark results with each promotion.
