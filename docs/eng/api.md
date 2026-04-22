# API

This page belongs to the current documentation branch.

This document describes the current control-plane HTTP API for version `2.0.6`. The catalog is aligned with the routes registered in `control-plane/internal/httpserver/server.go`.

## General Rules

- Application responses are JSON unless the endpoint is a download/export path.
- Primary authentication is session-based through cookies.
- Authorization is enforced server-side on every endpoint.
- Access control is permission-driven through RBAC.
- The UI uses this same API; if an endpoint is listed here, it is part of the real product surface.

## System Routes

### `GET /healthz`

System health endpoint.

It checks:

- revision store availability;
- revision catalog API health;
- component-level state required by the control-plane.

### `GET /api/setup/status`

First-run setup status.

Used by:

- login;
- onboarding;
- entry-guard logic before opening the protected UI.

### `GET /api/app/meta`

Application metadata:

- version;
- product name;
- repository URL;
- update-related metadata when update checks are enabled.

### `POST /api/app/ping`

Background session validation endpoint used by the UI.

### `GET /api/app/compat`

Compatibility report for application/runtime modules.

### `POST /api/app/compat/fix`

Attempts to fix a detected compatibility issue.

## Runtime Settings

### `GET /api/settings/runtime`

Reads runtime settings for the `Settings -> General` tab.

Used for:

- deployment mode;
- update checks;
- current version metadata.

### `PUT /api/settings/runtime`

Updates runtime settings.

The UI currently uses it for:

- `update_checks_enabled`
- retention values for `logs`, `activity`, `events`, and `bans`

### `POST /api/settings/runtime/check-updates`

Runs update checks.

Supports both manual and background usage.

### `GET /api/settings/runtime/storage-indexes?storage_indexes_limit=N&storage_indexes_offset=N`

Reads storage indexes for the `Settings -> Storage` tab.

### `DELETE /api/settings/runtime/storage-indexes?date=YYYY-MM-DD`

Deletes a storage index for a specific date.

## OWASP CRS

### `GET /api/owasp-crs/status`

Returns the status of the installed CRS release.

### `POST /api/owasp-crs/check-updates`

Performs a dry-run update availability check.

### `POST /api/owasp-crs/update`

Triggers an OWASP CRS update.

Also accepts the hourly auto-update toggle path used by the UI.

## Auth And Account

### Bootstrap And Login

- `POST /api/auth/bootstrap`
- `POST /api/auth/login`
- `POST /api/auth/login/2fa`
- `GET /api/auth/providers`
- `GET /api/auth/oidc/start`
- `GET /api/auth/oidc/callback`
- `POST /api/auth/logout`
- `GET /api/auth/me`

### 2FA

- `GET /api/auth/2fa/status`
- `POST /api/auth/2fa/setup`
- `POST /api/auth/2fa/enable`
- `POST /api/auth/2fa/disable`

### Password

- `POST /api/auth/change-password`

### Passkeys

- `POST /api/auth/passkeys/login/begin`
- `POST /api/auth/passkeys/login/finish`
- `POST /api/auth/login/2fa/passkey/begin`
- `POST /api/auth/login/2fa/passkey/finish`
- `GET /api/auth/passkeys`
- `POST /api/auth/passkeys/register/begin`
- `POST /api/auth/passkeys/register/finish`
- `PUT /api/auth/passkeys/{id}/rename`
- `DELETE /api/auth/passkeys/{id}`

## Configuration Resources

### Sites

- `GET /api/sites`
- `POST /api/sites`
- `GET /api/sites/{id}`
- `PUT /api/sites/{id}`
- `DELETE /api/sites/{id}`
- `POST /api/sites/{id}/ban`
- `POST /api/sites/{id}/unban`

### Upstreams

- `GET /api/upstreams`
- `POST /api/upstreams`
- `GET /api/upstreams/{id}`
- `PUT /api/upstreams/{id}`
- `DELETE /api/upstreams/{id}`

### Certificates

- `GET /api/certificates`
- `POST /api/certificates`
- `GET /api/certificates/{id}`
- `PUT /api/certificates/{id}`
- `DELETE /api/certificates/{id}`

### TLS Configs

- `GET /api/tls-configs`
- `POST /api/tls-configs`
- `GET /api/tls-configs/{siteID}`
- `PUT /api/tls-configs/{siteID}`
- `DELETE /api/tls-configs/{siteID}`

### TLS Auto Renew

- `GET /api/tls/auto-renew`
- `PUT /api/tls/auto-renew`

### Certificate Material Operations

- `POST /api/certificate-materials/upload`
- `POST /api/certificate-materials/import-archive`
- `POST /api/certificate-materials/export`
- `GET /api/certificate-materials/export/{certificateID}`

### ACME

- `POST /api/certificates/acme/issue`
- `POST /api/certificates/acme/renew/{certificateID}`
- `POST /api/certificates/self-signed/issue`

## Policies

### WAF Policies

- `GET /api/waf-policies`
- `POST /api/waf-policies`
- `GET /api/waf-policies/{id}`
- `PUT /api/waf-policies/{id}`
- `DELETE /api/waf-policies/{id}`

### Access Policies

- `GET /api/access-policies`
- `POST /api/access-policies`
- `POST /api/access-policies/upsert`
- `PUT /api/access-policies/upsert`
- `GET /api/access-policies/{id}`
- `PUT /api/access-policies/{id}`
- `DELETE /api/access-policies/{id}`

### Rate-Limit Policies

- `GET /api/rate-limit-policies`
- `POST /api/rate-limit-policies`
- `GET /api/rate-limit-policies/{id}`
- `PUT /api/rate-limit-policies/{id}`
- `DELETE /api/rate-limit-policies/{id}`

### Easy Site Profiles

- `GET /api/easy-site-profiles/{siteID}`
- `PUT /api/easy-site-profiles/{siteID}`
- `POST /api/easy-site-profiles/{siteID}`
- `GET /api/easy-site-profiles/catalog/countries`

### Anti-DDoS

- `GET /api/anti-ddos/settings`
- `POST /api/anti-ddos/settings`
- `PUT /api/anti-ddos/settings`

## Observability And Reporting

### Requests And Events

- `GET /api/events`
- `GET /api/requests`

### Dashboard

- `GET /api/dashboard/stats`
- `GET /api/dashboard/containers/overview`
- `GET /api/dashboard/containers/logs`
- `GET /api/dashboard/containers/issues`

### Reports

- `GET /api/reports/revisions`

### Audit

- `GET /api/audit`

## Revisions

### `GET /api/revisions`

The new aggregated revision catalog in `2.0.6`.

It powers the `Revisions` UI and returns:

- services;
- revisions;
- summary counters;
- status and rollout timeline data.

### `POST /api/revisions/compile`

Compiles a new revision.

### `POST /api/revisions/{revisionID}/apply`

Applies the selected revision.

### `POST /api/revisions/{revisionID}/approve`

Approves a revision that is blocked by the configured approval workflow.

### `DELETE /api/revisions/{revisionID}`

Deletes an inactive revision.

Related snapshot/job data is removed together with the inactive revision. Active revisions are not deletable.

### `DELETE /api/revisions/statuses`

Clears the revision status timeline.

Important:

- clears timeline/status counters;
- does not erase the pinned last successful or failed apply result stored on the revision itself.

## Administration

### `GET /api/administration/users`

Lists users for the administration table.

### `POST /api/administration/users`

Creates a user with an explicit role set.

### `GET /api/administration/users/{id}`

Reads one user for modal inspection.

### `PUT /api/administration/users/{id}`

Updates a user, including status, roles, and optional password rotation.

### `GET /api/administration/roles`

Lists roles and returns the known permission catalog for the role editor.

### `POST /api/administration/roles`

Creates a role.

### `GET /api/administration/roles/{id}`

Reads one role.

### `PUT /api/administration/roles/{id}`

Updates a role. The built-in `admin` role is normalized back to full access.

### `GET /api/administration/zero-trust/health`

Zero-trust health probe for users, roles, default role presence, and admin permission completeness.

### `GET /api/administration/scripts`

Returns the administrative script catalog.

### `POST /api/administration/scripts/{scriptID}/run`

Runs a registered administrative script with input values.

### `GET /api/administration/scripts/runs/{runID}/download`

Downloads the archived execution result.

### Enterprise Administration

- `GET /api/administration/enterprise`
- `PUT /api/administration/enterprise`
- `POST /api/administration/enterprise/scim-tokens`
- `DELETE /api/administration/enterprise/scim-tokens/{id}`
- `GET /api/administration/support-bundle`

### SCIM

- `GET /scim/v2/ServiceProviderConfig`
- `GET /scim/v2/Users`
- `POST /scim/v2/Users`
- `GET /scim/v2/Users/{id}`
- `PUT /scim/v2/Users/{id}`
- `PATCH /scim/v2/Users/{id}`
- `DELETE /scim/v2/Users/{id}`
- `GET /scim/v2/Groups`

## API-To-UI Mapping

- `Dashboard`: `dashboard/*`, `events`, `requests`
- `Sites`: `sites`, `upstreams`, `tls-configs`, `certificates`, `access-policies`, `easy-site-profiles`
- `Anti-DDoS`: `anti-ddos/settings`, `events`
- `OWASP CRS`: `owasp-crs/*`
- `TLS`: `certificates`, `tls-configs`, `tls/auto-renew`, `certificate-materials/*`, `certificates/acme/*`
- `Requests`: `requests`, `settings/runtime`, `sites`
- `Revisions`: `revisions`, `revisions/{id}/apply`, `revisions/statuses`
- `Events`: `events`, `sites`
- `Bans`: `sites/{id}/ban`, `sites/{id}/unban`, `events`, `access-policies`
- `Administration`: `administration/users*`, `administration/roles*`, `administration/zero-trust/health`, `administration/scripts*`
- `Activity`: `audit`
- `Settings`: `settings/runtime`, `app/meta`
- `Profile`: `auth/me`, `auth/change-password`, `auth/2fa/*`, `auth/passkeys/*`
