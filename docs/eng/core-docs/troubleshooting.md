# Troubleshooting Guide

This page belongs to the current documentation branch.

## Purpose

This guide maps common symptoms to likely causes and first recovery actions.

## `/healthz` Is Not Healthy

Check:

- container health;
- control-plane logs;
- PostgreSQL connectivity;
- Redis connectivity in High Availability mode.

First actions:

- verify environment variables;
- confirm PostgreSQL and Redis are reachable;
- inspect recent upgrades or migrations.

## Login Does Not Work

Check:

- bootstrap credentials or user state;
- session storage;
- cookie and HTTPS boundary;
- browser console and network responses.

First actions:

- verify the admin user exists;
- confirm session persistence;
- check trusted proxy and host configuration.

## Revision Compile Or Apply Fails

Check:

- `Revisions`;
- `Events`;
- runtime health;
- High Availability lock contention metrics;
- control-plane logs.

First actions:

- verify only one apply is in progress;
- inspect generated errors;
- re-run after fixing configuration or runtime readiness.

## Runtime Is Healthy But Traffic Is Broken

Check:

- `Sites`;
- `Upstreams`;
- TLS bindings;
- runtime logs;
- request archive or `Requests` view.

First actions:

- confirm routing to the expected upstream;
- verify certificate bindings;
- test with a simple known-good page.

## Too Many `403` Or `429`

Check:

- CRS mode and exclusions;
- rate-limit policies;
- Anti-DDoS settings;
- `Events`, `Requests`, and `Bans`.

First actions:

- identify which layer is blocking: WAF, rate-limit, or Anti-DDoS;
- temporarily relax the tightest policy;
- capture a sample of the failing request before making broad changes.

## mTLS — Client Certificate Rejected

Check:

- `mtls_enabled` and `mtls_optional` in the site profile;
- `mtls_client_ca_ref` path — the CA file must exist inside the runtime container;
- `mtls_verify_depth` — the client certificate chain must fit within the configured depth;
- `Events` and nginx logs for `ssl_verify_client` errors.

First actions:

- switch to `mtls_optional` for diagnosis without blocking;
- verify the CA file is accessible inside the runtime container;
- confirm the CA file is a valid PEM.

## JA3/JA4 — Legitimate Clients Are Blocked

Check:

- `blacklist_ja3` list in the site profile;
- `Events` — the blocked fingerprint is shown there;
- no popular browser or client fingerprint was accidentally added.

First actions:

- copy the fingerprint from the event and compare with the blacklist;
- remove the incorrect entry and recompile the revision.

## WebSocket — Connection Closes Unexpectedly

Check:

- `use_ws_inspection` and `reverse_proxy_websocket` — both must be `true`;
- `ws_block_patterns` — a pattern may match legitimate frames;
- `ws_max_message_bytes` — the limit may be too low;
- `ws_rate_msg_per_sec` — frame rate limit.

First actions:

- disable `use_ws_inspection` and verify if the problem disappears;
- if yes, look for a conflicting pattern or an overly tight frame size limit.

- identify whether the source is WAF, rate-limit, or Anti-DDoS;
- temporarily relax the smallest relevant policy;
- capture an example request before changing too much.

## Metrics Or Dashboards Are Empty

Check:

- metrics tokens;
- Prometheus targets;
- Grafana data source health;
- direct `/metrics` responses.

First actions:

- verify the token-protected endpoints manually;
- confirm scrape targets are `up`;
- inspect Prometheus configuration.

## Upgrade Finished But Something Feels Wrong

Check:

- post-upgrade smoke results;
- `/core-docs/api/app/meta`;
- current revision state;
- runtime readiness;
- migration logs.

First actions:

- run the strict smoke validation;
- compare against the previous known-good revision;
- roll back if core health is not stable.


