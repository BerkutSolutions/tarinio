# Let's Encrypt DNS-01 Operations

This document describes the DNS-01 certificate issuance flow and the operational checks around it.

## Scope

DNS-01:

- validates domain ownership through DNS TXT records;
- affects certificate issuance and renewal lifecycle;
- does not update OWASP CRS or WAF behavior by itself.

## Typical Flow

1. Configure DNS provider credentials such as a Cloudflare API token.
2. Start certificate issuance from the TLS workflow.
3. Verify that TXT challenge records are created and have propagated.
4. Confirm that certificate materials are issued and bound to the target site.
5. Apply the resulting runtime configuration and validate the HTTPS endpoint.

## Runtime And Control-Plane Expectations

- runtime must have access to the challenge and certificate material paths it needs;
- the certificate should appear in control-plane state;
- the site should reference it through TLS config;
- after apply, the host should present the expected certificate chain.

## Recommended Sequence With Other Security Operations

1. Validate DNS-01 certificate health.
2. Validate site TLS binding and host routing.
3. Perform OWASP CRS checks or updates separately from the `OWASP CRS` page.
4. Run HTTP/TLS smoke validation after certificate issuance or renewal.

## Operational Notes

- treat DNS provider tokens as secrets;
- document resolver, propagation, and provider-specific requirements;
- validate renewal before the expiration window becomes critical;
- keep a known-good fallback path for certificate import when ACME automation is not sufficient.

## Related Documents

- `docs/eng/core-docs/ui.md`
- `docs/eng/core-docs/runbook.md`
- `docs/eng/core-docs/security.md`

