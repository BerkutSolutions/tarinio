# Runtime Filesystem Contract

Status: Stage 1 runtime contract for `S1-17`

## Purpose

This file fixes the single-node MVP filesystem contract between:
- compiled revision bundles
- candidate staging
- active pointer selection
- runtime-visible `/etc/waf/...` paths

It does not add runtime behavior.
It only aligns already implemented paths and ownership boundaries.

## Single Source Of Truth

The only source of truth for runtime bundle selection is:
- `<runtime-root>/active/current.json`

In the runtime image this is typically:
- `/var/lib/waf/active/current.json`

The runtime must use this file to determine:
- which revision is active
- which staged candidate bundle must be exposed

The runtime must not select the active revision by reading:
- `/etc/waf/current`
- individual symlink targets
- inferred directory names

`/etc/waf/current` is derived only.

## Staging And Activation Paths

Canonical paths:
- staged bundle root: `<runtime-root>/candidates/<revision-id>/`
- active pointer: `<runtime-root>/active/current.json`

Example:
- `/var/lib/waf/candidates/rev-001/`
- `/var/lib/waf/active/current.json`

`current.json` stores:
- `revision_id`
- `candidate_path`

The `candidate_path` points to the selected staged bundle root.

## Bundle-Relative Paths

Inside one staged bundle, the canonical artifact paths are:
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

These paths are the same:
- in compiler output
- in staged bundle directories
- in the `candidate_path` bundle referenced by `current.json`

## Runtime-Visible Canonical Paths

Runtime exposes the selected bundle under:
- `/etc/waf/current`

This is a derived symlink to the selected `candidate_path`.

Runtime also exposes stable canonical paths for NGINX and ModSecurity:
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

These runtime-visible paths must map 1:1 to the same artifact paths inside the selected bundle.

## CRS Path Rule

The installed OWASP CRS is not stored in the compiled bundle.

Runtime exposes the installed CRS tree at:
- `/etc/waf/modsecurity/coreruleset/`

This path is runtime-provided and remains outside bundle ownership.
The bundle references it but does not redefine or mutate it.

## Ownership Rules

- compiler owns bundle-relative artifact paths
- staging owns materialization into `candidates/<revision-id>/`
- activation owns only `active/current.json`
- runtime owns only the derived `/etc/waf/...` view

Runtime does not own:
- bundle selection policy
- revision history
- revision metadata beyond the active pointer file

## Resulting Rule

For single-node MVP:
- `active/current.json` is authoritative
- `/etc/waf/current` is derived
- bundle-relative paths stay stable from compiler through runtime
- identical artifacts must keep identical relative paths at every stage
