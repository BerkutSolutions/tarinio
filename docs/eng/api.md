# API (EN)

Documentation baseline: `1.0.19`

## Release notes

### 1.0.19 (2026-04-04)

- Fixed Easy site profile save flow for reverse proxy: when `upstream_routing.reverse_proxy_host` is empty in UI, it is now derived from upstream scheme/host/port.
- Eliminated false `400 Bad Request` on service save when reverse proxy is enabled and upstream target fields are already filled.

### 1.0.10 (2026-04-03)

- TARINIO branding and a single version source via `meta.go`.
- Extended `/api/app/meta` (version/name/links).

## General rules

- All responses are JSON.
- Authentication is session-based (cookie) after login.
- Authorization is enforced server-side for every endpoint (zero-trust).

## System endpoints

- `GET /healthz` — liveness.
- `GET /api/setup/status` — first-run setup status.
- `GET /api/app/meta` — application version and metadata.

## Settings

- `GET /api/settings/runtime`
- `PUT /api/settings/runtime`
- `POST /api/settings/runtime/check-updates`
- OWASP CRS:
  - `GET /api/owasp-crs/status`
  - `POST /api/owasp-crs/check-updates`
  - `POST /api/owasp-crs/update`

## Auth

- `POST /api/auth/bootstrap`
- `POST /api/auth/login`
- `POST /api/auth/login/2fa`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/auth/change-password`
- `GET /api/auth/2fa/status`
- `POST /api/auth/2fa/setup`
- `POST /api/auth/2fa/enable`
- `POST /api/auth/2fa/disable`
- Passkeys:
  - `POST /api/auth/passkeys/login/begin`
  - `POST /api/auth/passkeys/login/finish`
  - `POST /api/auth/login/2fa/passkey/begin`
  - `POST /api/auth/login/2fa/passkey/finish`
  - `GET /api/auth/passkeys`
  - `POST /api/auth/passkeys/register/begin`
  - `POST /api/auth/passkeys/register/finish`
  - `PUT /api/auth/passkeys/{id}/rename`
  - `DELETE /api/auth/passkeys/{id}`

## Configuration (core resources)

- Sites: `/api/sites` and `/api/sites/{id}`
- Upstreams: `/api/upstreams` and `/api/upstreams/{id}`
- Certificates: `/api/certificates` and `/api/certificates/{id}`
- TLS configs: `/api/tls-configs` and `/api/tls-configs/{id}`
- Upload material: `POST /api/certificate-materials/upload`
- ACME:
  - `POST /api/certificates/acme/issue`
  - `POST /api/certificates/acme/renew/{certificateID}`

## Policies

- WAF policies: `/api/waf-policies` and `/api/waf-policies/{id}`
- Access policies: `/api/access-policies` and `/api/access-policies/{id}`
- Rate-limit policies: `/api/rate-limit-policies` and `/api/rate-limit-policies/{id}`
- Easy site profiles: `/api/easy-site-profiles/{siteID}`
- Catalog: `GET /api/easy-site-profiles/catalog/countries`
- Anti-DDoS: `/api/anti-ddos/settings`

## Observability / reports

- Events: `GET /api/events`
- Requests: `GET /api/requests`
- Revisions report: `GET /api/reports/revisions`
- Dashboard stats: `GET /api/dashboard/stats`
- Audit: `GET /api/audit`

## Revisions

- Compile: `POST /api/revisions/compile`
- Apply: `POST /api/revisions/{revisionID}/apply`





