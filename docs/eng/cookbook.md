# Operations Cookbook

This page belongs to the current documentation branch.

## Purpose

This cookbook groups common operator tasks into short, repeatable sequences.

## Add A New Site

1. Create the site.
2. Create the upstream.
3. Bind TLS if needed.
4. Compile a revision.
5. Apply the revision.
6. Validate through `/healthcheck` and `Requests`.

## Roll Out A Large Batch Safely

1. Persist changes with `X-WAF-Auto-Apply-Disabled: true`.
2. Review the intended state.
3. Compile once.
4. Apply once.
5. Validate through `Dashboard`, `Events`, and metrics.

## Tighten CRS Carefully

1. Record the current revision.
2. Change the CRS-related policy.
3. Compile and apply.
4. Watch for false positives in `Requests` and `Events`.
5. Add exclusions only where justified.

## Investigate A Blocked Request

1. Find the request in `Requests`.
2. Correlate it with `Events`.
3. Determine whether WAF, rate-limit, or Anti-DDoS blocked it.
4. Decide whether to tune the policy or keep the block.

## Perform A Rolling Upgrade

1. Back up before upgrade.
2. Validate HA health.
3. Upgrade one control-plane node at a time.
4. Probe API availability during the upgrade.
5. Run strict post-upgrade smoke validation.
