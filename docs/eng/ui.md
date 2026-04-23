# Interface and Operator Workflows

This page belongs to the current documentation branch.

This document describes the real TARINIO administrative interface for version `2.0.11`: login, onboarding, healthcheck, the sidebar sections, and the practical workflows behind each screen.

## Interface Map

The protected application area exposes these sections:

- `Dashboard`
- `Sites`
- `Anti-DDoS`
- `OWASP CRS`
- `TLS`
- `Requests`
- `Revisions`
- `Events`
- `Bans`
- `Administration`
- `Activity`
- `Settings`
- `Profile`

Separate routes also exist for:

- `/login`
- `/login/2fa`
- `/onboarding/user-creation`
- `/onboarding/site-tls`
- `/onboarding/confirm`
- `/healthcheck`

## Login And First-Time Setup

### `/login`

The login screen supports:

- username and password login;
- passkey login;
- enterprise `OIDC` login when enabled;
- automatic redirect to `2FA` when the account requires a second factor;
- redirect to onboarding when the system is not bootstrapped yet.

### `/login/2fa`

The second-factor screen supports:

- TOTP code entry;
- passkey completion as the second factor;
- returning to the main login screen.

### `/onboarding/*`

Initial setup is split into three stages:

1. Create the first administrator through `POST /api/auth/bootstrap`.
2. Create the first site and its TLS setup.
3. Confirm the configuration and transition into the protected UI.

Onboarding can immediately:

- create a site;
- create an upstream;
- issue a self-signed or ACME certificate;
- bind the certificate to the site;
- authorize the new administrator after completion.

## Healthcheck

The `/healthcheck` route works as a pre-entry system validation screen.

It checks:

- whether the session is valid;
- UI/runtime contract compatibility;
- availability of the APIs required by the application tabs;
- users/roles/zero-trust health through the administration probe;
- module compatibility status;
- container and runtime issues.

The check covers the following UI areas:

- `dashboard`
- `sites`
- `revisions`
- `anti-ddos`
- `owasp-crs`
- `tls`
- `requests`
- `events`
- `bans`
- `administration`
- `activity`
- `settings`
- `profile`

When a module supports self-healing, healthcheck can trigger `POST /api/app/compat/fix`.

## Dashboard

The dashboard is the primary daily monitoring view.

It provides:

- aggregate traffic and attack statistics;
- drill-down by site, URL, IP, country, and error code;
- blocked attack summaries;
- CPU and memory widgets;
- container health overview;
- direct container log viewing in a modal;
- widget visibility selection.

Use it for:

- understanding what is happening right now;
- validating the effects of a policy change;
- starting incident triage before jumping into `Requests`, `Events`, or `Bans`.

## Sites

The `/services` section is the main configuration workbench.

The list view supports:

- browsing sites in a table;
- opening a site by clicking its row;
- creating a new site;
- import;
- export.

Supported import/export formats:

- JSON configuration snapshots;
- ENV-oriented export;
- certificate archive import in TLS-related scenarios.

### Site Editor Wizard

The site editor is split into tabs.

#### `Front`

This tab defines:

- `server_name` and `service_id`;
- whether the site is enabled;
- the security mode;
- whether TLS is enabled;
- the certificate source strategy;
- self-signed, imported certificate, or ACME-based mode;
- Let’s Encrypt staging and wildcard options.

#### `Upstream`

This tab configures:

- upstream scheme, host, and port;
- reverse proxy mode;
- reverse proxy URL and host behavior;
- keepalive and websocket support;
- `Host`, `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Real-IP` forwarding;
- SNI for reverse proxy mode.

#### `HTTP`

This tab configures:

- maximum request body size;
- `HTTP/2`;
- `HTTP/3`;
- allowed methods.

#### `Headers`

This tab configures:

- cookie flags;
- `Referrer-Policy`;
- `Content-Security-Policy`;
- `Permissions-Policy`;
- CORS.

#### `Traffic`

This tab configures:

- allowlist and denylist;
- exception IP lists;
- bad behavior ban controls;
- connection limits for HTTP/1, HTTP/2, and HTTP/3;
- limit request behavior by URL;
- DNSBL;
- quick-list presets for typical signatures and URI groups.

#### `Blocking`

This tab configures:

- ban escalation;
- escalation scope;
- ban duration escalation stages.

#### `Antibot`

This tab supports:

- choosing a challenge mechanism;
- antibot URI;
- reCAPTCHA;
- hCaptcha;
- Cloudflare Turnstile;
- basic HTTP authentication as a protective barrier.

#### `Geo`

This tab configures:

- country whitelist;
- country blacklist;
- country search via the catalog.

#### `ModSecurity`

This tab configures:

- whether ModSecurity is enabled;
- CRS plugin usage;
- custom ModSecurity configuration;
- CRS version;
- custom ruleset path and inline content.

### Additional `Sites` Features

- In-editor settings search helps operators jump directly to a field.
- Certificate import and export are available inside the site card.
- Saving a site synchronizes site, upstream, TLS config, access policy, and easy site profile data.
- The section uses `easy-site-profiles` as the high-level application contract for site behavior.

## Anti-DDoS

This section manages global Anti-DDoS settings through `/api/anti-ddos/settings`.

It includes:

- an L4 settings block;
- an L7 settings block;
- model parameters;
- a model log table;
- a log detail modal;
- a help block that explains the model behavior.

Use it to:

- create or update global protection settings;
- inspect Anti-DDoS-related events;
- understand how the adaptive model classifies and escalates traffic.

## OWASP CRS

This section manages the OWASP CRS release state.

It supports:

- viewing the current release status;
- dry-run update checks;
- triggering an update;
- enabling hourly automatic update checks.

Use it to:

- verify whether the current CRS bundle is outdated;
- update the runtime rule set in a controlled way;
- assess CRS changes before a compile/apply window.

## TLS

This section covers the certificate lifecycle end to end.

### `Certificates`

Supports:

- creating and updating certificate metadata;
- listing certificates;
- multi-select;
- deletion;
- material archive import;
- export of selected materials.

### `Bindings`

Supports:

- binding `site_id -> certificate_id`;
- updating an existing binding;
- deleting a TLS config.

### `Auto Renew`

Supports:

- enabling automatic renewal;
- configuring `renew_before_days`.

### `Upload`

Supports:

- uploading a PEM certificate and private key;
- creating a certificate record together with its materials.

### `ACME`

Supports:

- issuing a certificate;
- issuing and binding in one step;
- manual renewal;
- CA selection: `letsencrypt`, `zerossl`, `custom`;
- `http-01` and `dns-01`;
- DNS provider env values;
- DNS resolvers;
- ZeroSSL EAB values;
- staging and wildcard options.

## Requests

This section is designed for line-by-line request log analysis.

It provides:

- filtering by site, method, HTTP status, and time;
- free-text search;
- pagination;
- sortable columns;
- a request detail modal;
- raw entry rendering.

Typical use cases:

- finding a specific `request_id`;
- checking which URI started returning `403` after a policy change;
- inspecting `user-agent`, referer, and upstream information for a suspicious request.

## Revisions

This is the core change-management screen in `2.0.11`.

It includes:

- the main service grid;
- a rollout status sidebar;
- a revision event timeline;
- a modal for the selected service revisions;
- a status detail modal.

It supports:

- browsing revisions by service;
- inspecting active, pending, and failed revisions;
- applying a revision;
- deleting an inactive revision;
- viewing the last apply result and last event;
- clearing the status timeline without losing the revision’s pinned last apply result.

The screen is powered by the aggregated revision catalog `GET /api/revisions`.

## Events

This section is used for security event review.

It supports:

- event tables;
- filtering and search;
- event detail modals;
- site correlation.

It is the main screen for answering which protection layer triggered and why.

## Bans

This section consolidates block-related information from events and access policy state.

It supports:

- active and expired ban review;
- manual IP bans by site;
- extending bans using standard duration presets;
- manual unban;
- detail inspection by IP, country, module, and event context;
- synchronization with allowlist and denylist context.

Use it to:

- isolate an attack source quickly;
- understand what is already blocked automatically;
- preserve operational context when applying manual bans.

## Administration

This section is now a real operator workspace rather than a placeholder.

It includes:

- a user table with account status, role bindings, and last login data;
- left-click row inspection through a read-only modal;
- explicit edit actions for accounts and roles;
- creation of new users and roles in-place from the same screen;
- a role editor with grouped permission selection;
- an `Enterprise` panel for `OIDC`, `SCIM`, approval policy, and support evidence;
- the existing catalog of administrative scripts and downloadable run archives.

Operational rules:

- the built-in `admin` account always keeps full access;
- base role templates now include `auditor`, `manager`, and `soc`;
- access to UI sections is enforced by server-side zero-trust permissions, not only by frontend visibility.
- enterprise group mappings are configured from the same screen and apply to external identities coming from `OIDC` and `SCIM`.

## Activity

The `/activity` section exposes administrative audit data through `GET /api/audit`.

It supports:

- action, site, and period filters;
- paginated browsing;
- an operator activity timeline;
- analysis of who changed what and when.

It is the main screen for investigating operator-side changes.

## Settings

The settings section is split into tabs:

- `General`
- `Storage`
- `Security`
- `Logging`
- `Secrets`
- `About`

### `General`

Allows operators to:

- switch the UI language;
- enable or disable update checks;
- trigger manual update checks;
- see deployment mode and current version.

### `Storage`

Allows operators to:

- configure retention for request logs;
- configure retention for activity, events, and bans;
- review storage indexes;
- delete individual storage index entries.

### `Security`

Allows operators to:

- enable login brute-force protection with configurable thresholds;
- configure max failed attempts, window size, and block duration for login throttling;
- control whether insecure Vault TLS mode can be used at all.

### `Logging`

Allows operators to:

- configure hot/cold logging backends and retention windows;
- tune routing between hot and cold streams;
- manage OpenSearch and ClickHouse connection settings.

### `Secrets`

Allows operators to:

- select a secret provider (`encrypted_file` or Vault);
- configure Vault endpoint, mount, and path prefix;
- provide a Vault token without exposing existing secrets in plain text.

### `About`

Displays:

- product branding;
- version information;
- project and organization links.

Access model:

- non-admin users always retain access to `About`;
- `General` and `Storage` require dedicated permissions and are no longer implicitly available to every authenticated account.

## Profile

The profile section combines personal preferences and account protection settings.

It supports:

- viewing the profile;
- changing the password;
- reviewing and configuring `2FA`;
- enabling `TOTP`;
- disabling `2FA` with confirmation;
- registering passkeys;
- renaming passkeys;
- deleting passkeys;
- language selection;
- UI timezone preferences.

## Shared Navigation And Application Mechanics

Across the application, TARINIO uses:

- client-side navigation without full page reloads;
- a profile hover card with roles and permission counts;
- a notification center;
- background session pings;
- dedicated handling for `401`, `403`, and `429`;
- `ru/en` i18n.

## Recommended Operational Workflows

### New Site

1. Create the site in `Sites`.
2. Configure upstream behavior and baseline HTTP/TLS settings.
3. Issue or bind a certificate in `TLS` when needed.
4. Compile a revision.
5. Apply the revision.
6. Validate behavior through `Dashboard`, `Requests`, and `Events`.

### Protection Change

1. Update site, Anti-DDoS, or ModSecurity settings.
2. Validate the intended impact.
3. Run compile/apply.
4. Watch `Dashboard`, `Events`, and `Bans`.
5. Roll back through `Revisions` if the change degrades traffic.

### Incident Investigation

1. Use `Dashboard` for the overall picture.
2. Use `Events` for the exact detection reason.
3. Use `Requests` for request-level detail.
4. Use `Bans` for block decisions and manual intervention.
5. Use `Activity` to understand whether the incident overlaps with a recent administrative change.
6. Export a signed support bundle from `Administration -> Enterprise` when evidence needs to leave the system.

