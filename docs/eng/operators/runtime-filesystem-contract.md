# Runtime Filesystem Contract

Status: Stage 1 runtime contract for the shared runtime root used by single-node and HA control-plane deployments.

## Purpose

This document defines the contract between:

- compiled revision bundles;
- staged candidates;
- active revision selection;
- runtime-visible paths under `/etc/waf/...`.

It does not add new runtime logic. It formalizes the ownership boundaries already assumed by the system.

## Single Source Of Truth

The only authoritative source for active revision selection is:

- `<runtime-root>/active/current.json`

In the runtime image this is typically:

- `/var/lib/waf/active/current.json`

Runtime must use this file to determine:

- which revision is active;
- which candidate bundle should be exposed as current.

Runtime must not infer the active revision from:

- `/etc/waf/current`;
- symlink targets;
- guessed directory names.

`/etc/waf/current` is a derived representation only.

## Canonical Staging And Activation Paths

Main paths:

- staged bundle root: `<runtime-root>/candidates/<revision-id>/`
- active pointer: `<runtime-root>/active/current.json`

Example:

- `/var/lib/waf/candidates/rev-001/`
- `/var/lib/waf/active/current.json`

`current.json` stores:

- `revision_id`
- `candidate_path`

## Paths Inside A Bundle

Inside a staged bundle, stable relative paths must be used:

- `manifest.json`
- `nginx/nginx.conf`
- `nginx/conf.d/*.conf`
- `nginx/access/*.conf`
- `nginx/ratelimits/*.conf`
- `nginx/sites/*.conf`
- `modsecurity/modsecurity.conf`
- `modsecurity/sites/*.conf`
- `modsecurity/crs-setup.conf`
- `modsecurity/crs-overrides/*.conf`
- `tls/*.conf`
- `errors/...`
- `ddos-model/config.json`
- `l4guard/config.json`

These paths must remain identical:

- in compiler output;
- in staged candidates;
- in the bundle referenced by `current.json`.

## Runtime-Visible Canonical Paths

Runtime exposes the selected bundle through:

- `/etc/waf/current`

This is a symlink or equivalent derived projection to `candidate_path`.

Runtime also provides stable paths such as:

- `/etc/waf/nginx/nginx.conf`
- `/etc/waf/nginx/conf.d/`
- `/etc/waf/nginx/access/`
- `/etc/waf/nginx/ratelimits/`
- `/etc/waf/nginx/sites/`
- `/etc/waf/modsecurity/modsecurity.conf`
- `/etc/waf/modsecurity/sites/`
- `/etc/waf/modsecurity/crs-setup.conf`
- `/etc/waf/modsecurity/crs-overrides/`
- `/etc/waf/tls/`
- `/etc/waf/errors/`
- `/etc/waf/l4guard/config.json`
- `/etc/waf/ddos-model/config.json`

These runtime-visible paths must reflect the selected bundle contents one-to-one.

## OWASP CRS Rule

Installed OWASP CRS content is not stored inside the compiled bundle.

Runtime exposes it separately at:

- `/etc/waf/modsecurity/coreruleset/`

This directory:

- is provided by runtime;
- remains outside bundle ownership;
- can be referenced by the bundle, but must not be overwritten by the bundle.

## Ownership Boundaries

- the compiler owns only bundle-relative artifact paths;
- staging owns materialization into `candidates/<revision-id>/`;
- activation owns only `active/current.json`;
- runtime owns only the derived `/etc/waf/...` view.

Runtime does not own:

- revision selection policy;
- revision history;
- revision metadata beyond the active pointer;
- control-plane state.

## Final Rule

For the shared runtime-root contract used by both the single-node baseline and the HA control-plane topology:

- `active/current.json` is authoritative;
- `/etc/waf/current` is derived;
- relative artifact paths must not change between compile, stage, and runtime;
- the same artifact must have the same relative path at every stage.
