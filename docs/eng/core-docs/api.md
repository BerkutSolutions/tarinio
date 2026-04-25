# API

This page belongs to the current documentation branch.

This document describes the current control-plane HTTP API for version `current release`. The catalog is aligned with the routes registered in `control-plane/internal/httpserver/server.go`.

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

### `GET /core-docs/api/setup/status`

First-run setup status.

Used by:

- login;
- onboarding;
- entry-guard logic before opening the protected UI.

### `GET /core-docs/api/app/meta`

Application metadata:

- version;
- product name;
- repository URL;
- update-related metadata when update checks are enabled.

### `POST /core-docs/api/app/ping`

Background session validation endpoint used by the UI.

### `GET /core-docs/api/app/compat`

Compatibility report for application/runtime modules.

### `POST /core-docs/api/app/compat/fix`

Attempts to fix a detected compatibility issue.

## Runtime Settings

### `GET /core-docs/api/settings/runtime`

Reads runtime settings for the `Settings -> General` tab.

Used for:

- deployment mode;
- update checks;
- current version metadata.

### `PUT /core-docs/api/settings/runtime`

Updates runtime settings.

The UI currently uses it for:

- `update_checks_enabled`
- retention values for `logs`, `activity`, `events`, and `bans`

### `POST /core-docs/api/settings/runtime/check-updates`

Runs update checks.

Supports both manual and background usage.

### `GET /core-docs/api/settings/runtime/storage-indexes?storage_indexes_limit=N&storage_indexes_offset=N`

Reads storage indexes for the `Settings -> Storage` tab.

### `DELETE /core-docs/api/settings/runtime/storage-indexes?date=YYYY-MM-DD`

Deletes a storage index for a specific date.

## OWASP CRS

### `GET /core-docs/api/owasp-crs/status`

Returns the status of the installed CRS release.

### `POST /core-docs/api/owasp-crs/check-updates`

Performs a dry-run update availability check.

### `POST /core-docs/api/owasp-crs/update`

Triggers an OWASP CRS update.

Also accepts the hourly auto-update toggle path used by the UI.

## Auth And Account

### Bootstrap And Login

- `POST /core-docs/api/auth/bootstrap`
- `POST /core-docs/api/auth/login`
- `POST /core-docs/api/auth/login/2fa`
- `GET /core-docs/api/auth/providers`
- `GET /core-docs/api/auth/oidc/start`
- `GET /core-docs/api/auth/oidc/callback`
- `POST /core-docs/api/auth/logout`
- `GET /core-docs/api/auth/me`

### 2FA

- `GET /core-docs/api/auth/2fa/status`
- `POST /core-docs/api/auth/2fa/setup`
- `POST /core-docs/api/auth/2fa/enable`
- `POST /core-docs/api/auth/2fa/disable`

### Password

- `POST /core-docs/api/auth/change-password`

### Passkeys

- `POST /core-docs/api/auth/passkeys/login/begin`
- `POST /core-docs/api/auth/passkeys/login/finish`
- `POST /core-docs/api/auth/login/2fa/passkey/begin`
- `POST /core-docs/api/auth/login/2fa/passkey/finish`
- `GET /core-docs/api/auth/passkeys`
- `POST /core-docs/api/auth/passkeys/register/begin`
- `POST /core-docs/api/auth/passkeys/register/finish`
- `PUT /core-docs/api/auth/passkeys/{id}/rename`
- `DELETE /core-docs/api/auth/passkeys/{id}`

## Configuration Resources

### Sites

- `GET /core-docs/api/sites`
- `POST /core-docs/api/sites`
- `GET /core-docs/api/sites/{id}`
- `PUT /core-docs/api/sites/{id}`
- `DELETE /core-docs/api/sites/{id}`
- `POST /core-docs/api/sites/{id}/ban`
- `POST /core-docs/api/sites/{id}/unban`

### Upstreams

- `GET /core-docs/api/upstreams`
- `POST /core-docs/api/upstreams`
- `GET /core-docs/api/upstreams/{id}`
- `PUT /core-docs/api/upstreams/{id}`
- `DELETE /core-docs/api/upstreams/{id}`

### Certificates

- `GET /core-docs/api/certificates`
- `POST /core-docs/api/certificates`
- `GET /core-docs/api/certificates/{id}`
- `PUT /core-docs/api/certificates/{id}`
- `DELETE /core-docs/api/certificates/{id}`

### TLS Configs

- `GET /core-docs/api/tls-configs`
- `POST /core-docs/api/tls-configs`
- `GET /core-docs/api/tls-configs/{siteID}`
- `PUT /core-docs/api/tls-configs/{siteID}`
- `DELETE /core-docs/api/tls-configs/{siteID}`

### TLS Auto Renew

- `GET /core-docs/api/tls/auto-renew`
- `PUT /core-docs/api/tls/auto-renew`

### Certificate Material Operations

- `POST /core-docs/api/certificate-materials/upload`
- `POST /core-docs/api/certificate-materials/import-archive`
- `POST /core-docs/api/certificate-materials/export`
- `GET /core-docs/api/certificate-materials/export/{certificateID}`

### ACME

- `POST /core-docs/api/certificates/acme/issue`
- `POST /core-docs/api/certificates/acme/renew/{certificateID}`
- `POST /core-docs/api/certificates/self-signed/issue`

## Policies

### WAF Policies

- `GET /core-docs/api/waf-policies`
- `POST /core-docs/api/waf-policies`
- `GET /core-docs/api/waf-policies/{id}`
- `PUT /core-docs/api/waf-policies/{id}`
- `DELETE /core-docs/api/waf-policies/{id}`

### Access Policies

- `GET /core-docs/api/access-policies`
- `POST /core-docs/api/access-policies`
- `POST /core-docs/api/access-policies/upsert`
- `PUT /core-docs/api/access-policies/upsert`
- `GET /core-docs/api/access-policies/{id}`
- `PUT /core-docs/api/access-policies/{id}`
- `DELETE /core-docs/api/access-policies/{id}`

### Rate-Limit Policies

- `GET /core-docs/api/rate-limit-policies`
- `POST /core-docs/api/rate-limit-policies`
- `GET /core-docs/api/rate-limit-policies/{id}`
- `PUT /core-docs/api/rate-limit-policies/{id}`
- `DELETE /core-docs/api/rate-limit-policies/{id}`

### Easy Site Profiles

- `GET /core-docs/api/easy-site-profiles/{siteID}`
- `PUT /core-docs/api/easy-site-profiles/{siteID}`
- `POST /core-docs/api/easy-site-profiles/{siteID}`
- `GET /core-docs/api/easy-site-profiles/catalog/countries`

### Anti-DDoS

- `GET /core-docs/api/anti-ddos/settings`
- `POST /core-docs/api/anti-ddos/settings`
- `PUT /core-docs/api/anti-ddos/settings`

## Observability And Reporting

### Requests And Events

- `GET /core-docs/api/events`
- `GET /core-docs/api/requests`

### Dashboard

- `GET /core-docs/api/dashboard/stats`
- `GET /core-docs/api/dashboard/containers/overview`
- `GET /core-docs/api/dashboard/containers/logs`
- `GET /core-docs/api/dashboard/containers/issues`

### Reports

- `GET /core-docs/api/reports/revisions`

### Audit

- `GET /core-docs/api/audit`

## Revisions

### `GET /core-docs/api/revisions`

The new aggregated revision catalog in `current release`.

It powers the `Revisions` UI and returns:

- services;
- revisions;
- summary counters;
- status and rollout timeline data.

### `POST /core-docs/api/revisions/compile`

Compiles a new revision.

### `POST /core-docs/api/revisions/{revisionID}/apply`

Applies the selected revision.

Required permission:

- `revisions.write`

### `POST /core-docs/api/revisions/{revisionID}/approve`

Approves a revision that is blocked by the configured approval workflow.

Required permission:

- `revisions.approve`

### `DELETE /core-docs/api/revisions/{revisionID}`

Deletes an inactive revision.

Related snapshot/job data is removed together with the inactive revision. Active revisions are not deletable.

Required permission:

- `revisions.write`

### `DELETE /core-docs/api/revisions/statuses`

Clears the revision status timeline.

Important:

- clears timeline/status counters;
- does not erase the pinned last successful or failed apply result stored on the revision itself.

## Administration

### `GET /core-docs/api/administration/users`

Lists users for the administration table.

### `POST /core-docs/api/administration/users`

Creates a user with an explicit role set.

### `GET /core-docs/api/administration/users/{id}`

Reads one user for modal inspection.

### `PUT /core-docs/api/administration/users/{id}`

Updates a user, including status, roles, and optional password rotation.

### `GET /core-docs/api/administration/roles`

Lists roles and returns the known permission catalog for the role editor.

### `POST /core-docs/api/administration/roles`

Creates a role.

### `GET /core-docs/api/administration/roles/{id}`

Reads one role.

### `PUT /core-docs/api/administration/roles/{id}`

Updates a role. The built-in `admin` role is normalized back to full access.

### `GET /core-docs/api/administration/zero-trust/health`

Zero-trust health probe for users, roles, default role presence, and admin permission completeness.

### `GET /core-docs/api/administration/scripts`

Returns the administrative script catalog.

### `POST /core-docs/api/administration/scripts/{scriptID}/run`

Runs a registered administrative script with input values.

### `GET /core-docs/api/administration/scripts/runs/{runID}/download`

Downloads the archived execution result.

### Enterprise Administration

- `GET /core-docs/api/administration/enterprise`
- `PUT /core-docs/api/administration/enterprise`
- `POST /core-docs/api/administration/enterprise/scim-tokens`
- `DELETE /core-docs/api/administration/enterprise/scim-tokens/{id}`
- `GET /core-docs/api/administration/support-bundle`

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
- `Revisions`: `revisions`, `revisions/{id}/apply`, `revisions/{id}/approve`, `revisions/statuses`
- `Events`: `events`, `sites`
- `Bans`: `sites/{id}/ban`, `sites/{id}/unban`, `events`, `access-policies`
- `Administration`: `administration/users*`, `administration/roles*`, `administration/zero-trust/health`, `administration/scripts*`
- `Activity`: `audit`
- `Settings`: `settings/runtime`, `app/meta`
- `Profile`: `auth/me`, `auth/change-password`, `auth/2fa/*`, `auth/passkeys/*`


