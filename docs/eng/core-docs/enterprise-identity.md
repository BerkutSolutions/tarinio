# Enterprise Identity

This page belongs to the current documentation branch.

This document describes the real enterprise identity model in TARINIO `3.0.2`.

## What Is Implemented

TARINIO `3.0.2` provides:

- OpenID Connect (`OIDC`) login for interactive operator access
- `SCIM 2.0` user provisioning with bearer-token authentication
- external group to local role mapping
- approval-gated revision rollout

The identity surface is implemented by the real control-plane routes and UI, not by documentation-only placeholders.

## Important Boundary: LDAP / AD

TARINIO `3.0.2` does **not** expose a standalone direct `LDAP` password-login endpoint.

Instead, the product supports `LDAP` / `Active Directory` backed identity in the practical enterprise way:

- directory users authenticate through an external `OIDC` identity provider
- directory users can also be provisioned through a `SCIM` bridge
- directory groups are mapped to TARINIO roles through external group mappings

In other words, TARINIO supports `LDAP/AD group mapping` when those groups are projected into:

- the configured `OIDC` groups claim
- the `SCIM` user/group payload

This keeps the control-plane focused on policy, revision, and audit responsibilities instead of embedding a separate LDAP client stack into the runtime login path.

## OIDC Login

Interactive enterprise login is exposed through:

- `GET /core-docs/api/auth/providers`
- `GET /core-docs/api/auth/oidc/start`
- `GET /core-docs/api/auth/oidc/callback`

### OIDC Flow

1. The login page discovers whether enterprise SSO is enabled through `GET /core-docs/api/auth/providers`.
2. When enabled, the UI shows the enterprise SSO button.
3. `GET /core-docs/api/auth/oidc/start` creates a short-lived login challenge and redirects the browser to the provider authorization endpoint.
4. `GET /core-docs/api/auth/oidc/callback` exchanges the authorization code, validates the `id_token`, checks issuer/audience/nonce, and creates a TARINIO session.

### OIDC Validation

The control-plane validates:

- issuer
- audience (`client_id`)
- signing key from JWKS
- `nonce`
- verified email, when that policy is enabled
- allowed email domains, when configured

### OIDC Group Mapping

OIDC configuration stores:

- default role IDs
- external group to role mappings
- the groups claim name

On login, TARINIO resolves the effective role set from:

- configured default roles
- mapped roles for every matching external group

The resulting roles are written into the TARINIO user record and session.

## SCIM Provisioning

Provisioning endpoints are exposed through:

- `GET /scim/v2/ServiceProviderConfig`
- `GET /scim/v2/Users`
- `POST /scim/v2/Users`
- `GET /scim/v2/Users/{id}`
- `PUT /scim/v2/Users/{id}`
- `PATCH /scim/v2/Users/{id}`
- `DELETE /scim/v2/Users/{id}`
- `GET /scim/v2/Groups`

### Authentication

SCIM uses per-token bearer authentication.

Tokens are created and revoked from:

- `POST /core-docs/api/administration/enterprise/scim-tokens`
- `DELETE /core-docs/api/administration/enterprise/scim-tokens/{id}`

The raw token value is shown only at creation time. Later reads expose only metadata such as:

- display name
- prefix
- creation time
- last used time

### SCIM User Behavior

Provisioned users are stored as TARINIO users with:

- `auth_source=scim`
- external identity metadata
- external groups
- last sync timestamp

SCIM provisioning updates the local TARINIO user instead of creating parallel shadow records for the same external identity.

## Approval Workflow

Revision approval policy is managed through `Administration -> Enterprise` and enforced by:

- `POST /core-docs/api/revisions/compile`
- `POST /core-docs/api/revisions/{revisionID}/approve`
- `POST /core-docs/api/revisions/{revisionID}/apply`

### Behavior

When approval policy is enabled:

- a newly compiled revision enters `pending_approval`
- the revision stores the required approval count
- eligible reviewers can approve it
- apply is blocked until the approval threshold is met

The approval record stores:

- approving user ID
- username
- comment
- approval time

### Self-Approval

Self-approval can be disabled. In that mode, the same user who compiled the revision cannot be counted as an approver.

## User Record Model

Enterprise identity extends the local TARINIO user model with:

- `auth_source`
- `external_id`
- `external_groups`
- `last_synced_at`

This allows the control-plane to keep:

- local bootstrap and local-password users
- OIDC users
- SCIM users

inside one auditable user catalog without losing source attribution.

## Administration UI

`Administration -> Enterprise` is the live control surface for:

- enabling OIDC login
- entering issuer/client settings
- configuring default roles
- configuring external group mappings
- enabling SCIM
- minting and revoking SCIM tokens
- configuring approval policy
- downloading signed support bundles

## Recommended Enterprise Layout

For enterprise directory integration:

1. Keep TARINIO local bootstrap only for emergency access.
2. Use an `OIDC` provider backed by `LDAP/AD`.
3. Project directory groups into the OIDC groups claim.
4. Mirror the same group vocabulary into `SCIM` if automated provisioning is required.
5. Map external groups to TARINIO roles in the enterprise settings.

## Related Documents

- [API](core-docs/api.md)
- [Security](core-docs/security.md)
- [Evidence And Releases](core-docs/evidence-and-releases.md)
- [UI](core-docs/ui.md)


