# Security Profiles

TARINIO `1.3.5` includes 5 built-in service security profiles:

1. `strict`
2. `balanced`
3. `compat`
4. `api`
5. `public-edge`

A profile is a baseline preset. Operators can keep the profile and add per-service overrides.

## How Presets Work

- The selected profile writes `front_service.profile` and applies a baseline bundle.
- `balanced` is the default baseline from `DefaultProfile(...)`.
- Other profiles are overlays on top of that baseline.
- Manual edits in UI/API are preserved after save.

## Profile Comparison

### `strict`

Use when false positives are acceptable and aggressive protection is preferred.

Key preset values:

- `front_service.security_mode = block`
- `security_behavior_and_limits.use_bad_behavior = true`
- `bad_behavior_status_codes = [400,401,403,404,405,429,444]`
- `bad_behavior_threshold = 60`
- `bad_behavior_count_time_seconds = 60`
- `bad_behavior_ban_time_seconds = 900`
- `use_limit_req = true`, `limit_req_url = /`, `limit_req_rate = 80r/s`
- `use_limit_conn = true`, `http1/http2/http3 = 120/220/220`
- `security_antibot.antibot_challenge = javascript`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

### `balanced`

Default profile for general production web traffic.

Baseline defaults (can be environment-tuned via `WAF_DEFAULT_*`):

- `front_service.security_mode = block`
- `use_bad_behavior = true`
- default bad-behavior status set (focus on abuse signals)
- `use_limit_req = true`, default `limit_req_rate = 120r/s`
- `use_limit_conn = true`, defaults `200/400/400`
- `security_antibot.antibot_challenge = no`
- `security_api_positive.use_api_positive_security = false`
- `security_modsecurity.use_modsecurity = true`

For management-site IDs, baseline limits are relaxed and scoped to `/api/` to reduce admin UI false positives.

### `compat`

Use for legacy apps where compatibility and low friction are more important than strict blocking.

Key preset values:

- `front_service.security_mode = monitor`
- `use_bad_behavior = false`
- `use_limit_req = false`
- `use_limit_conn = true`, `http1/http2/http3 = 300/500/500`
- `security_antibot.antibot_challenge = no`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

### `api`

Use for API-first services.

Key preset values:

- `front_service.security_mode = block`
- `http_behavior.allowed_methods = GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD`
- `http_headers.use_cors = true`
- `http_headers.cors_allowed_origins = ["*"]`
- `use_limit_req = true`, `limit_req_url = /api/`, `limit_req_rate = 200r/s`
- `security_antibot.antibot_challenge = no`
- `security_api_positive.use_api_positive_security = true`
- `security_api_positive.enforcement_mode = monitor`
- `security_api_positive.default_action = allow`

### `public-edge`

Use for internet-facing edge entry points with elevated bot/abuse pressure.

Key preset values:

- `front_service.security_mode = block`
- `use_bad_behavior = true`
- `bad_behavior_status_codes = [400,401,403,404,405,429,444]`
- `bad_behavior_threshold = 80`
- `bad_behavior_count_time_seconds = 60`
- `bad_behavior_ban_time_seconds = 600`
- `use_blacklist = true`
- `use_dnsbl = true`
- `use_limit_req = true`, `limit_req_url = /`, `limit_req_rate = 100r/s`
- `security_antibot.antibot_challenge = javascript`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

## When To Choose Which

- Choose `strict` for hardened public portals with predictable client behavior.
- Choose `balanced` for most web workloads.
- Choose `compat` for fragile legacy apps during migration/hardening.
- Choose `api` for REST/gRPC gateway style traffic with API schema controls.
- Choose `public-edge` for high-risk edge endpoints (marketing/public landing/login edge).

## Additional Profile Settings (v1.3.x)

Beyond baseline preset values, all profiles support the following extended setting groups:

### Incoming mTLS
- `front_service.mtls_enabled` — require a client certificate.
- `front_service.mtls_optional` — optional mode (pass status without blocking).
- `front_service.mtls_verify_depth` — chain verification depth (default `1`).
- `front_service.mtls_client_ca_ref` — path to the client CA file.
- `front_service.mtls_pass_headers` — forward `X-Client-Verify` / `X-Client-DN` headers to upstream.

### Upstream mTLS (Outgoing)
- `upstream_routing.upstream_mtls_enabled` — present a client certificate when proxying.
- `upstream_routing.upstream_mtls_cert_ref` — WAF client certificate path.
- `upstream_routing.upstream_mtls_key_ref` — WAF client key path.
- `upstream_routing.upstream_mtls_ca_ref` — CA for upstream server verification.

### JA3/JA4 Fingerprinting
- `security_behavior_and_limits.blacklist_ja3` — JA3 fingerprint block list.
- `security_behavior_and_limits.blacklist_ja3_urls` — URL sources for JA3 blacklists.

### Virtual Patching
- `virtual_patches[]` — temporary blocking rules with TTL: `match_uri`, `match_header`, `match_body`, `action`, `expires_at`.

### Geo/Time Windows
- `geo_time_windows[]` — GeoIP and schedule restrictions: `countries`, `weekdays`, `time_from`, `time_to`, `action`.

### WebSocket Inspection
- `security_websocket.use_ws_inspection` — enable WS frame inspection.
- `security_websocket.ws_block_patterns` — regex patterns for frame blocking.
- `security_websocket.ws_max_message_bytes` — frame size limit.
- `security_websocket.ws_rate_msg_per_sec` — frame rate limit.

## Operator Notes

- Profile selection does not lock configuration; all fields remain editable.
- `security_auth_basic.users[]` and `session_inactivity_minutes` are independent of selected profile and can be tuned per service.
- In RAW editor and API payloads, profile is stored as `front_service.profile`.
