# Troubleshooting Guide

This page belongs to the current documentation branch.

## Purpose

This guide maps common symptoms to likely causes and first recovery actions.

## `/healthz` Is Not Healthy

Check:

- container health;
- control-plane logs;
- PostgreSQL connectivity;
- Redis connectivity in HA mode.

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
- HA lock contention metrics;
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
- `/api/app/meta`;
- current revision state;
- runtime readiness;
- migration logs.

First actions:

- run the strict smoke validation;
- compare against the previous known-good revision;
- roll back if core health is not stable.
