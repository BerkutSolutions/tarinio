# Release Notes Policy

This page belongs to the current documentation branch.

## Purpose

Release notes should help operators understand not only what changed, but what they must do next.

## Recommended Sections For Every Release

- `Changed`
- `Fixed`
- `Operational Impact`
- `Upgrade Notes`
- `Rollback Notes`
- `Documentation Updates`

## What Must Be Called Out Explicitly

- storage migrations;
- required environment variable changes;
- compose or image changes;
- High Availability behavior changes;
- benchmark or resource-profile changes;
- new operational checks.

## Upgrade Expectation

Because TARINIO follows a forward-moving release model, release notes should always tell operators:

- whether the AIO path is sufficient;
- whether post-upgrade smoke validation should be stricter than usual;
- whether special rollback attention is needed.

