# Compiler Template Asset Layout

Status: MVP layout for `S1-03`

## Purpose

This file fixes the static compiler template tree used to build one revision bundle.

It is aligned with:
- ADR-002 bundle structure
- the manifest `contents.kind` contract
- the accepted domain model

The layout is static.
It does not introduce runtime-side mutation, rule generators, or non-MVP extension points.

## Template Tree

```text
compiler/templates/
  nginx/
    nginx.conf.tmpl
    access/
      site.conf.tmpl
    conf.d/
      base.conf.tmpl
    ratelimits/
      http.conf.tmpl
      site.conf.tmpl
    sites/
      site.conf.tmpl
  modsecurity/
    modsecurity.conf.tmpl
    sites/
      site.conf.tmpl
    crs-setup.conf.tmpl
    crs-overrides/
      site-overrides.conf.tmpl
  tls/
    refs.conf.tmpl
  errors/
    403.html.tmpl
    429.html.tmpl
    50x.html.tmpl
```

## Mapping Rules

Each rendered artifact from this tree must map directly into `manifest.contents`.

Canonical rule for all compiler outputs:
- bundle paths are relative paths inside one `Revision` bundle
- staging materializes them under `candidates/<revision-id>/`
- activation records the selected candidate only in `active/current.json`
- runtime consumes `active/current.json` as the only source of truth
- runtime exposes `/etc/waf/current` only as a derived symlink to the selected candidate bundle
- runtime-facing `/etc/waf/...` paths must map 1:1 to the same bundle-relative artifact paths

## NGINX

Templates:
- `compiler/templates/nginx/nginx.conf.tmpl`
- `compiler/templates/nginx/access/site.conf.tmpl`
- `compiler/templates/nginx/conf.d/base.conf.tmpl`
- `compiler/templates/nginx/ratelimits/http.conf.tmpl`
- `compiler/templates/nginx/ratelimits/site.conf.tmpl`
- `compiler/templates/nginx/sites/site.conf.tmpl`

Manifest kind:
- `nginx_config`

Domain model sources:
- `Site`
- `Upstream`
- `TLSConfig`
- `AccessPolicy`
- `RateLimitPolicy`

Bundle output paths:
- `nginx/nginx.conf`
- `nginx/conf.d/base.conf`
- `nginx/conf.d/ratelimits.conf`
- `nginx/access/<site-id>.conf`
- `nginx/ratelimits/<site-id>.conf`
- `nginx/sites/<site-id>.conf`

Canonical runtime paths:
- `/etc/waf/nginx/nginx.conf`
- `/etc/waf/nginx/conf.d/base.conf`
- `/etc/waf/nginx/conf.d/ratelimits.conf`
- `/etc/waf/nginx/access/<site-id>.conf`
- `/etc/waf/nginx/ratelimits/<site-id>.conf`
- `/etc/waf/nginx/sites/<site-id>.conf`

## ModSecurity

Templates:
- `compiler/templates/modsecurity/modsecurity.conf.tmpl`
- `compiler/templates/modsecurity/sites/site.conf.tmpl`

Manifest kind:
- `modsecurity_config`

Domain model sources:
- `WAFPolicy`
- `Site`

Bundle output paths:
- `modsecurity/modsecurity.conf`
- `modsecurity/sites/<site-id>.conf`

Canonical runtime paths:
- `/etc/waf/modsecurity/modsecurity.conf`
- `/etc/waf/modsecurity/sites/<site-id>.conf`

## CRS Wiring and Overrides

Templates:
- `compiler/templates/modsecurity/crs-setup.conf.tmpl`
- `compiler/templates/modsecurity/crs-overrides/site-overrides.conf.tmpl`

Manifest kind:
- `crs_config`

Domain model sources:
- `WAFPolicy`

Bundle output paths:
- `modsecurity/crs-setup.conf`
- `modsecurity/crs-overrides/<site-id>.conf`

Canonical runtime paths:
- `/etc/waf/modsecurity/crs-setup.conf`
- `/etc/waf/modsecurity/crs-overrides/<site-id>.conf`
- `/etc/waf/modsecurity/coreruleset/` for the installed CRS tree exposed by runtime without modifying bundle contents

## TLS References

Templates:
- `compiler/templates/tls/refs.conf.tmpl`

Manifest kind:
- `tls_ref`

Domain model sources:
- `TLSConfig`
- `Certificate`

Bundle output paths:
- `tls/<site-id>.conf`

Canonical runtime paths:
- `/etc/waf/tls/<site-id>.conf`

## Error Pages

Templates:
- `compiler/templates/errors/403.html.tmpl`
- `compiler/templates/errors/429.html.tmpl`
- `compiler/templates/errors/50x.html.tmpl`

Manifest kind:
- `error_asset`

Domain model sources:
- `Site`
- site error-page assignment fields

Bundle output paths:
- `errors/<site-id>/403.html`
- `errors/<site-id>/429.html`
- `errors/<site-id>/50x.html`

Canonical runtime paths:
- `/etc/waf/errors/<site-id>/403.html`
- `/etc/waf/errors/<site-id>/429.html`
- `/etc/waf/errors/<site-id>/50x.html`

## Activation And Runtime Contract

Single-node MVP uses these filesystem rules:
- staged bundle root: `<runtime-root>/candidates/<revision-id>/`
- active pointer path: `<runtime-root>/active/current.json`
- `active/current.json` is the only runtime selection source of truth
- `current.json.candidate_path` must point to the selected staged bundle root
- `/etc/waf/current` is a derived symlink to that same bundle root
- runtime must not decide which revision is active by reading `/etc/waf/current`
- runtime must not infer the active revision from partial symlink state

## MVP Boundaries

This layout does not include:
- rule-generation engines
- plugin template hooks
- runtime-generated templates
- multi-node deployment metadata
- enterprise-only artifact groups

It exists only to support deterministic compiler output for MVP bundles.
