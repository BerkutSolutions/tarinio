# OWASP CRS Operations

This document explains how TARINIO manages OWASP CRS versions and site-level CRS behavior.

## Scope

The `OWASP CRS` section manages:

- installed CRS release visibility;
- dry-run update checks;
- operator-triggered updates;
- optional hourly auto-update checks.

It is complementary to site-level ModSecurity behavior configured in `Sites`.

## Update Model

Runtime behavior:

- on first runtime start, TARINIO can pull the latest OWASP CRS release;
- later changes are operator-driven from the `OWASP CRS` page or from the scheduled auto-update mode;
- CRS is not expected to change silently just because the page was opened.

Optional scheduled mode:

- the operator can enable `hourly auto-update` from the `OWASP CRS` page;
- when enabled, runtime checks roughly once per hour and updates to a newer release when available;
- when disabled, runtime keeps the currently active CRS version.

## Runtime/API Controls

Control-plane endpoints:

- `GET /core-docs/api/owasp-crs/status`
- `POST /core-docs/api/owasp-crs/check-updates`
- `POST /core-docs/api/owasp-crs/update`

Notes:

- `check-updates` is non-destructive;
- `update` applies the latest release and triggers the runtime-side update path;
- `update` can also toggle `enable_hourly_auto_update`.

## Site-Level CRS Behavior

Easy profile controls relevant to CRS:

- `use_modsecurity`
- `use_modsecurity_crs_plugins`
- `use_modsecurity_custom_configuration`
- custom configuration path/content

Defaults:

- CRS is used by default when ModSecurity is enabled for Easy profiles;
- custom ModSecurity configuration is optional and injected only when explicitly enabled.

Security modes:

- `block` -> `SecRuleEngine On`
- `monitor` -> `SecRuleEngine DetectionOnly`
- `transparent` -> `SecRuleEngine Off`

## Safe Operational Flow

1. Check current CRS status from the `OWASP CRS` page.
2. Run a dry-run update check.
3. Schedule the update in a controlled window when needed.
4. Revalidate behavior after the update through `Requests`, `Events`, and site traffic checks.
5. If the new ruleset causes unacceptable behavior, use revision-based configuration rollback where applicable and isolate site-side exceptions carefully.

## Smoke Validation

Use:

- `deploy/compose/default/test-xss.ps1`

Capabilities:

- sends several XSS probes through query, body, and header vectors;
- supports container mode when host TLS tooling is problematic;
- verifies that CRS rules are loaded;
- helps confirm that blocking behavior still works after updates.

## Related Documents

- `docs/eng/core-docs/ui.md`
- `docs/eng/operators/waf-tuning-guide.md`
- `docs/eng/core-docs/runbook.md`

