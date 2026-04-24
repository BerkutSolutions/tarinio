# OWASP CRS Operations

Date: `2026-04-04`

This document explains how TARINIO manages OWASP CRS versions and service-level CRS behavior.

## 1. Update model

Runtime behavior:
- On first runtime start, TARINIO performs a one-time automatic pull of the latest OWASP CRS release from GitHub.
- After first start, CRS is not silently auto-updated from page open.
- Further updates are operator-driven from UI `OWASP CRS` page (manual button).

Optional scheduled mode:
- Operator can enable `hourly auto-update` on the `OWASP CRS` page.
- When enabled, runtime checks once per hour and updates automatically if a newer release exists.
- When disabled, runtime keeps the current active CRS version.

## 2. Runtime/API controls

Control-plane endpoints:
- `GET /core-docs/api/owasp-crs/status`
- `POST /core-docs/api/owasp-crs/check-updates` (`dry_run` support)
- `POST /core-docs/api/owasp-crs/update`

Notes:
- `check-updates` is non-destructive and returns latest metadata.
- `update` applies latest release and triggers runtime reload.
- `update` accepts `enable_hourly_auto_update` to toggle periodic updates.

## 3. Service-level behavior

Easy profile controls:
- `use_modsecurity`
- `use_modsecurity_crs_plugins`
- `use_modsecurity_custom_configuration`
- `custom_configuration.path/content`

Defaults:
- CRS is enabled by default for Easy profiles when ModSecurity is enabled.
- Custom configuration is optional and injected only when checkbox is enabled.

Security modes:
- `block` -> `SecRuleEngine On`
- `monitor` -> `SecRuleEngine DetectionOnly`
- `transparent` -> `SecRuleEngine Off`

## 4. Smoke validation

Use:
- `deploy/compose/default/test-xss.ps1`

Capabilities:
- sends 5 XSS probes (query/body/header vectors),
- supports container mode (avoids host TLS `schannel` issues),
- verifies CRS rules are loaded from runtime logs (`local > 0`),
- checks blocked ratio threshold.

