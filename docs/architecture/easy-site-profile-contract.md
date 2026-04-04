# Easy Site Profile Contract (T2-01)

Status: Completed contract definition for `T2-01` (extended)  
Date: 2026-04-01

## Purpose

This document freezes the user-friendly Easy mode contract for site create/edit.

It defines:
- grouped tabs and field intent
- payload shape and value types
- defaults and validation
- mapping to current backend/compiler model
- BunkerWeb variable equivalence as internal reference only

## UX Rules

- Easy mode must show human-readable labels and descriptions.
- Internal technical variable names (for example `REVERSE_PROXY_SSL_SNI`) are forbidden in Easy UI labels.
- Each editable field must have backend persistence and compile/apply effect.
- No UI-only toggles are allowed.

## Scope (Current Batch)

Included groups:
1. Web Service Front
2. Upstream Routing
3. HTTP Behavior
4. HTTP Headers
5. Security Behavior and Limits
6. Security Antibot
7. Security Auth Basic
8. Security Country Policy
9. Security ModSecurity

## Canonical Easy Payload (API Contract Target)

```json
{
  "site_id": "site-a",
  "front_service": {
    "server_name": "www.example.com",
    "security_mode": "block",
    "auto_lets_encrypt": true,
    "use_lets_encrypt_staging": false,
    "use_lets_encrypt_wildcard": false,
    "certificate_authority_server": "letsencrypt"
  },
  "upstream_routing": {
    "use_reverse_proxy": true,
    "reverse_proxy_host": "http://upstream-server:8080",
    "reverse_proxy_url": "/",
    "reverse_proxy_custom_host": "",
    "reverse_proxy_ssl_sni": false,
    "reverse_proxy_ssl_sni_name": "",
    "reverse_proxy_websocket": true,
    "reverse_proxy_keepalive": true
  },
  "http_behavior": {
    "allowed_methods": ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"],
    "max_client_size": "100m",
    "http2": true,
    "http3": false,
    "ssl_protocols": ["TLSv1.2", "TLSv1.3"]
  },
  "http_headers": {
    "cookie_flags": "* SameSite=Lax",
    "content_security_policy": "",
    "permissions_policy": "",
    "keep_upstream_headers": ["*"],
    "referrer_policy": "no-referrer-when-downgrade",
    "use_cors": false,
    "cors_allowed_origins": ["*"]
  },
  "security_behavior_and_limits": {
    "use_bad_behavior": true,
    "bad_behavior_status_codes": [400, 401, 403, 404, 405, 429, 444],
    "bad_behavior_ban_time_seconds": 3600,
    "bad_behavior_threshold": 30,
    "bad_behavior_count_time_seconds": 60,
    "use_blacklist": false,
    "use_dnsbl": false,
    "blacklist_ip": [],
    "blacklist_rdns": [],
    "blacklist_asn": [],
    "blacklist_user_agent": [],
    "blacklist_uri": [],
    "blacklist_ip_urls": [],
    "blacklist_rdns_urls": [],
    "blacklist_asn_urls": [],
    "blacklist_user_agent_urls": [],
    "blacklist_uri_urls": [],
    "use_limit_conn": true,
    "limit_conn_max_http1": 200,
    "limit_conn_max_http2": 400,
    "limit_conn_max_http3": 400,
    "use_limit_req": true,
    "limit_req_url": "/",
    "limit_req_rate": "100r/s"
  },
  "security_antibot": {
    "antibot_challenge": "no",
    "antibot_uri": "/challenge",
    "antibot_recaptcha_score": 0.7,
    "antibot_recaptcha_sitekey": "",
    "antibot_recaptcha_secret": "",
    "antibot_hcaptcha_sitekey": "",
    "antibot_hcaptcha_secret": "",
    "antibot_turnstile_sitekey": "",
    "antibot_turnstile_secret": ""
  },
  "security_auth_basic": {
    "use_auth_basic": false,
    "auth_basic_location": "sitewide",
    "auth_basic_user": "changeme",
    "auth_basic_password": "",
    "auth_basic_text": "Restricted area"
  },
  "security_country_policy": {
    "blacklist_country": [],
    "whitelist_country": []
  },
  "security_modsecurity": {
    "use_modsecurity": true,
    "use_modsecurity_crs_plugins": false,
    "modsecurity_crs_version": "4",
    "modsecurity_crs_plugins": [],
    "custom_configuration": {
      "path": "modsec/anomaly_score.conf",
      "content": ""
    }
  }
}
```

## Default Profile

Default values must avoid accidental operator lockout or user bans.

Required default baseline:
- connection limits: high and permissive (`200/400/400`)
- request rate: high and permissive (`100r/s` + non-zero burst in compiler mapping)
- bad behavior ban: enabled but with moderate threshold (`30` over `60s`)
- blacklist: disabled by default
- antibot: disabled (`no`) by default
- auth basic: disabled by default
- country allow/deny lists: empty by default
- modsecurity: enabled by default, CRS plugins disabled by default

Env-based limits are bootstrap fallback only; source of truth is persisted Easy profile values compiled into revision artifacts.

Implemented for `T2-08`:
- compiler emits `l4guard/config.json` per revision bundle from Easy limiter settings
- runtime l4guard reads file config first (`/etc/waf/l4guard/config.json`)
- environment values (`WAF_L4_GUARD_*`) override file values only when explicitly provided

## First-Init Template Behavior

On first onboarding site initialization:
- control-plane creates site/upstream resources;
- onboarding persists an Easy profile template before first compile/apply;
- template is based on the default profile and patched with actual first-init values:
  - `front_service.server_name` from created site host
  - `upstream_routing.reverse_proxy_host` from created upstream target

This guarantees first apply uses an explicit persisted profile instead of implicit placeholders.
All values remain editable through the same Easy profile API/UI flow after bootstrap.

## Bad Behavior Status Code Catalog

Easy mode must support full selectable catalog from the provided requirements list (47 explicit codes):

`400,401,402,403,404,405,406,407,408,409,410,411,412,413,414,415,416,417,418,421,422,423,424,425,426,428,429,431,444,451,500,501,502,503,504,505,507,508,510,511,520,521,522,523,524,525,526`

Default selected codes:

`400,401,403,404,405,429,444`

## Validation Contract

- `site_id`: lowercase slug, required.
- `server_name`: required hostname/FQDN.
- `security_mode`: one of `block|monitor|transparent` (UI wording may differ, backend value must be canonical).
- `reverse_proxy_host`: required URI when `use_reverse_proxy=true`.
- `reverse_proxy_url`: must start with `/`.
- `allowed_methods`: non-empty subset of `GET|POST|HEAD|OPTIONS|PUT|DELETE|PATCH`.
- `certificate_authority_server`: enum from backend registry (initial set at least `letsencrypt`).
- `max_client_size`: nginx size syntax (`k|m|g`).
- `ssl_protocols`: subset of supported runtime list.
- `bad_behavior_status_codes`: subset of supported 41-code catalog.
- list fields (`blacklist_*`, header arrays): normalized trim + dedupe.
- `permissions_policy`: list-style editable entries with trim + dedupe normalization.
- limits (`limit_conn_*`, thresholds, times): positive integers.
- `limit_req_rate`: format `N r/s` or canonical no-space `Nr/s` normalized to `Nr/s`.
- `antibot_challenge`: one of `no|cookie|javascript|captcha|recaptcha|hcaptcha|turnstile|mcaptcha`.
- `antibot_uri`: required absolute path when antibot mode is not `no`.
- `antibot_recaptcha_score`: float range `0.0..1.0`.
- `auth_basic_location`: enum (`sitewide` initially, extensible).
- `auth_basic_user`: required when auth basic is enabled.
- `auth_basic_password`: write-only in API responses (masked/omitted on reads).
- `blacklist_country` and `whitelist_country`: values from geo registry (countries, groups, continents), dedupe and conflict validation.
- `modsecurity_crs_plugins`: list of plugin identifiers, trim + dedupe.
- `custom_configuration.path`: fixed/validated safe relative path under allowed modsecurity custom config directory.

## Easy UI Interaction Contract

To keep Easy mode understandable and not technical:

- `Allowed methods`, `Keep upstream headers`, `Permissions-Policy`, `Core Rule Set Plugins`, blacklist/whitelist list fields:
  - row-based editable list
  - `+` button adds a new row
  - `x` button removes a row
- `Bad status codes`:
  - searchable dropdown with checkboxes
  - multiple selection
  - selected count badge
- `certificate_authority_server`, country lists:
  - dropdown/select controls, not free-text technical input
- secret fields (`auth_basic_password`, antibot provider secrets):
  - masked in UI
  - never echoed back in plain text from API

## Mapping to Current Backend and Compiler

Current Stage 1 entities already available:
- `sites.Site`
- `upstreams.Upstream`
- `tlsconfigs.TLSConfig` + certificate flow
- `wafpolicies.WAFPolicy`
- `accesspolicies.AccessPolicy`
- `ratelimitpolicies.RateLimitPolicy`

Contract mapping rules for this batch:
- `front_service.server_name` -> `sites.primary_host`
- `upstream_routing.reverse_proxy_host` -> `upstreams.(scheme, host, port, base path)`
- `security_behavior_and_limits.limit_req_*` -> `ratelimitpolicies`
- `security_behavior_and_limits.blacklist_ip` (initial MVP bridge) -> `accesspolicies.denylist`
- missing sections (`http_headers`, `http_behavior`, full bad-behavior details, dnsbl/asn/ua/uri lists, conn limit family, antibot, auth-basic, country policy, modsecurity extensions) require domain extension in `T2-02` and compiler mapping in `T2-04`

## Internal BunkerWeb Equivalence (Reference Only)

These names are internal reference mapping only; they must not be rendered in Easy labels.

- Front service:
  - `server_name` -> `SERVER_NAME`
  - `security_mode` -> `SECURITY_MODE`
  - `auto_lets_encrypt` -> `AUTO_LETS_ENCRYPT`
  - `use_lets_encrypt_staging` -> `USE_LETS_ENCRYPT_STAGING`
  - `use_lets_encrypt_wildcard` -> `USE_LETS_ENCRYPT_WILDCARD`
  - `certificate_authority_server` -> `LETS_ENCRYPT_SERVER`
- Upstream:
  - `use_reverse_proxy` -> `USE_REVERSE_PROXY`
  - `reverse_proxy_host` -> `REVERSE_PROXY_HOST`
  - `reverse_proxy_url` -> `REVERSE_PROXY_URL`
  - `reverse_proxy_custom_host` -> `REVERSE_PROXY_CUSTOM_HOST`
  - `reverse_proxy_ssl_sni` -> `REVERSE_PROXY_SSL_SNI`
  - `reverse_proxy_ssl_sni_name` -> `REVERSE_PROXY_SSL_SNI_NAME`
  - `reverse_proxy_websocket` -> `REVERSE_PROXY_WS`
  - `reverse_proxy_keepalive` -> `REVERSE_PROXY_KEEPALIVE`
- HTTP General:
  - `allowed_methods` -> `ALLOWED_METHODS`
  - `max_client_size` -> `MAX_CLIENT_SIZE`
  - `http2` -> `HTTP2`
  - `http3` -> `HTTP3`
  - `ssl_protocols` -> `SSL_PROTOCOLS`
- HTTP Headers:
  - `cookie_flags` -> `COOKIE_FLAGS`
  - `content_security_policy` -> `CONTENT_SECURITY_POLICY`
  - `permissions_policy` -> `PERMISSIONS_POLICY`
  - `keep_upstream_headers` -> `KEEP_UPSTREAM_HEADERS`
  - `referrer_policy` -> `REFERRER_POLICY`
  - `use_cors` -> `USE_CORS`
  - `cors_allowed_origins` -> `CORS_ALLOW_ORIGIN`
- Security behavior and limits:
  - `use_bad_behavior` -> `USE_BAD_BEHAVIOR`
  - `bad_behavior_status_codes` -> `BAD_BEHAVIOR_STATUS_CODES`
  - `bad_behavior_ban_time_seconds` -> `BAD_BEHAVIOR_BAN_TIME`
  - `bad_behavior_threshold` -> `BAD_BEHAVIOR_THRESHOLD`
  - `bad_behavior_count_time_seconds` -> `BAD_BEHAVIOR_COUNT_TIME`
  - `use_blacklist` -> `USE_BLACKLIST`
  - `use_dnsbl` -> `USE_DNSBL`
  - `blacklist_*` -> `BLACKLIST_*`
  - `use_limit_conn` -> `USE_LIMIT_CONN`
  - `limit_conn_max_http1` -> `LIMIT_CONN_MAX_HTTP1`
  - `limit_conn_max_http2` -> `LIMIT_CONN_MAX_HTTP2`
  - `limit_conn_max_http3` -> `LIMIT_CONN_MAX_HTTP3`
  - `use_limit_req` -> `USE_LIMIT_REQ`
  - `limit_req_url` -> `LIMIT_REQ_URL`
  - `limit_req_rate` -> `LIMIT_REQ_RATE`
- Security antibot:
  - `antibot_challenge` -> `USE_ANTIBOT`
  - `antibot_uri` -> `ANTIBOT_URI`
  - `antibot_recaptcha_score` -> `ANTIBOT_RECAPTCHA_SCORE`
  - `antibot_recaptcha_sitekey` -> `ANTIBOT_RECAPTCHA_SITEKEY`
  - `antibot_recaptcha_secret` -> `ANTIBOT_RECAPTCHA_SECRET`
  - `antibot_hcaptcha_sitekey` -> `ANTIBOT_HCAPTCHA_SITEKEY`
  - `antibot_hcaptcha_secret` -> `ANTIBOT_HCAPTCHA_SECRET`
  - `antibot_turnstile_sitekey` -> `ANTIBOT_TURNSTILE_SITEKEY`
  - `antibot_turnstile_secret` -> `ANTIBOT_TURNSTILE_SECRET`
- Security auth basic:
  - `use_auth_basic` -> `USE_AUTH_BASIC`
  - `auth_basic_location` -> `AUTH_BASIC_LOCATION`
  - `auth_basic_user` -> `AUTH_BASIC_USER`
  - `auth_basic_password` -> `AUTH_BASIC_PASSWORD`
  - `auth_basic_text` -> `AUTH_BASIC_TEXT`
- Security country policy:
  - `blacklist_country` -> `BLACKLIST_COUNTRY`
  - `whitelist_country` -> `WHITELIST_COUNTRY`
- Security modsecurity:
  - `use_modsecurity` -> `USE_MODSECURITY`
  - `use_modsecurity_crs_plugins` -> `USE_MODSECURITY_CRS_PLUGINS`
  - `modsecurity_crs_version` -> `MODSECURITY_CRS_VERSION`
  - `modsecurity_crs_plugins` -> `MODSECURITY_CRS_PLUGINS`
  - `custom_configuration.path/content` -> custom modsecurity include content (compiler-managed material)

## Delivery Gate for T2-01

`T2-01` is considered complete only when:
- this contract is accepted as source for `T2-02`, `T2-03`, `T2-04`
- next tasks implement backend-first (no UI-first contract drift)
- UI labels stay human-readable and technical identifiers remain internal

