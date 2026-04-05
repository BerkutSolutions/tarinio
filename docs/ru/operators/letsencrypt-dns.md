# Let's Encrypt DNS-01 Operations

Date: `2026-04-04`

This document describes DNS-01 certificate issuance flow and operational checks.

## 1. Scope

Let's Encrypt DNS-01:
- validates domain ownership through DNS TXT records;
- affects certificate issuance/renewal lifecycle;
- does not change or update OWASP CRS by itself.

## 2. Typical flow

1. Configure DNS provider credentials (for example Cloudflare API token).
2. Start certificate issuance from UI/certificate workflow.
3. Ensure TXT challenge records are created and propagated.
4. Confirm certificate material is issued and bound to target site.
5. Apply revision and validate HTTPS endpoint.

## 3. Operational checks

- runtime has access to control-plane challenge/material directories;
- certificate appears in control-plane state and is referenced by TLS config;
- target host serves expected certificate chain after apply.

## 4. Recommended sequence with WAF operations

1. Validate DNS-01 certificate health.
2. Validate site TLS binding and host routing.
3. Perform OWASP CRS dry-run/update separately from the `OWASP CRS` page.
4. Run XSS smoke checks after CRS update.
