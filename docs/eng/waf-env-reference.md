# WAF Env Parameter Reference

This page belongs to the current documentation branch.

This document describes all WAF service fields in env format used by the raw service editor and `.env` export/import in the `Services` section.

## How To Read This Document

- Every key starts with the `WAF_SITE_` prefix.
- Arrays and structured values are passed as JSON: `[]`, `["GET","POST"]`, `[400,401,403]`, `[{"path":"/login","rate":"20r/s"}]`.
- Boolean values use `true` or `false`.
- If a field is omitted in raw mode, the product applies the current default value.
- Examples below use safe non-production values and do not contain real secrets.

## Service Identity And Publishing

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ID` | Internal service identifier used in APIs, revisions, and linked resources. | Lowercase string, usually derived from host. | None, required at creation time. | `app-example-com` |
| `WAF_SITE_PRIMARY_HOST` | Main public hostname of the protected service. | FQDN or `localhost` for local development. | None, required at creation time. | `app.example.com` |
| `WAF_SITE_ENABLED` | Enables or disables the service without deleting it. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SECURITY_MODE` | Base WAF reaction mode. | `transparent`, `monitor`, `block` | `block` | `block` |

## TLS And Certificates

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_TLS_ENABLED` | Enables HTTPS for the service. | `true`, `false` | `true` | `true` |
| `WAF_SITE_TLS_SELF_SIGNED` | Marks use of the self-signed development fallback. | `true`, `false` | `false` | `false` |
| `WAF_SITE_CERTIFICATE_ID` | Certificate identifier bound to the service. | String | `<site_id>-tls` | `app-example-com-tls` |
| `WAF_SITE_AUTO_LETS_ENCRYPT` | Automatically issues a certificate through ACME. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_LETS_ENCRYPT_STAGING` | Uses the ACME staging endpoint instead of production. | `true`, `false` | `false` | `false` |
| `WAF_SITE_USE_LETS_ENCRYPT_WILDCARD` | Requests a wildcard certificate when the flow supports it. | `true`, `false` | `false` | `false` |
| `WAF_SITE_CERTIFICATE_AUTHORITY_SERVER` | Certificate authority provider used by managed issuance. | `letsencrypt`, `zerossl`, `custom`, `import` | `letsencrypt` | `letsencrypt` |
| `WAF_SITE_ACME_ACCOUNT_EMAIL` | ACME account email for issuance and renewal. | Valid email | Empty | `ops@example.com` |

## Upstream And Reverse Proxy

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_UPSTREAM_ID` | Upstream resource identifier. | String | `<site_id>-upstream` | `app-example-com-upstream` |
| `WAF_SITE_UPSTREAM_HOST` | Host or IP of the upstream application. | FQDN, container name, IPv4, IPv6 | `ui` | `10.0.10.15` |
| `WAF_SITE_UPSTREAM_PORT` | Upstream application port. | `1..65535` | `80` | `8080` |
| `WAF_SITE_UPSTREAM_SCHEME` | Upstream protocol. | `http`, `https` | `http` | `http` |
| `WAF_SITE_USE_REVERSE_PROXY` | Enables reverse proxying to upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_REVERSE_PROXY_HOST` | Full upstream address used by the easy profile. | URL | Derived from `scheme://host:port` | `http://10.0.10.15:8080` |
| `WAF_SITE_REVERSE_PROXY_URL` | Base upstream path. | Path starting with `/` | `/` | `/` |
| `WAF_SITE_REVERSE_PROXY_CUSTOM_HOST` | Custom `Host` header passed to upstream if needed. | FQDN or empty | Empty | `backend.internal.example` |
| `WAF_SITE_REVERSE_PROXY_SSL_SNI` | Enables SNI for HTTPS upstream connections. | `true`, `false` | `false` | `true` |
| `WAF_SITE_REVERSE_PROXY_SSL_SNI_NAME` | SNI name used for TLS upstream connections. | FQDN | Empty | `backend.internal.example` |
| `WAF_SITE_REVERSE_PROXY_WEBSOCKET` | Allows websocket upgrades to upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_REVERSE_PROXY_KEEPALIVE` | Enables upstream keepalive connections. | `true`, `false` | `true` | `true` |
| `WAF_SITE_PASS_HOST_HEADER` | Passes the original `Host` header to upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_FORWARDED_FOR` | Adds `X-Forwarded-For`. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_FORWARDED_PROTO` | Adds `X-Forwarded-Proto`. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_REAL_IP` | Adds `X-Real-IP`. | `true`, `false` | `false` | `false` |

## HTTP Behavior And Base Limits

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ALLOWED_METHODS` | Allowed HTTP methods. | JSON string array | `["GET","POST","HEAD","OPTIONS","PUT","PATCH","DELETE"]` | `["GET","POST","OPTIONS"]` |
| `WAF_SITE_MAX_CLIENT_SIZE` | Maximum request body size. | nginx size value like `10m`, `100m`, `1g` | `100m` | `50m` |
| `WAF_SITE_HTTP2` | Enables HTTP/2 on the frontend. | `true`, `false` | `true` | `true` |
| `WAF_SITE_HTTP3` | Enables HTTP/3 on the frontend. | `true`, `false` | `false` | `false` |
| `WAF_SITE_SSL_PROTOCOLS` | Allowed TLS protocol versions. | JSON string array | `["TLSv1.2","TLSv1.3"]` | `["TLSv1.3"]` |

## HTTP Headers And Browser Security

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_COOKIE_FLAGS` | Cookie flags template normalized by WAF. | String | `* SameSite=Lax` | `* SameSite=Strict; Secure` |
| `WAF_SITE_CONTENT_SECURITY_POLICY` | `Content-Security-Policy` header value. | String or empty | Empty | `default-src 'self'; img-src 'self' data:` |
| `WAF_SITE_PERMISSIONS_POLICY` | `Permissions-Policy` directives. | JSON string array | `[]` | `["camera=()","geolocation=()"]` |
| `WAF_SITE_KEEP_UPSTREAM_HEADERS` | Upstream headers to preserve without filtering. | JSON string array | `["*"]` | `["X-Request-Id","X-Upstream-Time"]` |
| `WAF_SITE_REFERRER_POLICY` | `Referrer-Policy` header value. | String | `no-referrer-when-downgrade` | `strict-origin-when-cross-origin` |
| `WAF_SITE_USE_CORS` | Enables CORS responses on the frontend. | `true`, `false` | `false` | `true` |
| `WAF_SITE_CORS_ALLOWED_ORIGINS` | Allowed CORS origins. | JSON string array | `["*"]` | `["https://app.example.com","https://admin.example.com"]` |

## Allowlist, Exceptions, And Denylist

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_ALLOWLIST` | Enables service-level allowlist access mode. | `true`, `false` | `false` | `false` |
| `WAF_SITE_USE_EXCEPTIONS` | Enables exception entries for selected IPs. | `true`, `false` | `false` | `true` |
| `WAF_SITE_ACCESS_ALLOWLIST` | IP/CIDR entries always allowed. | JSON string array | `[]` | `["203.0.113.10","198.51.100.0/24"]` |
| `WAF_SITE_EXCEPTIONS_IP` | IP/CIDR entries excluded from part of the protection logic. | JSON string array | `[]` | `["203.0.113.15"]` |
| `WAF_SITE_ACCESS_DENYLIST` | IP/CIDR entries always denied. | JSON string array | `[]` | `["198.51.100.23"]` |

## Bad Behavior And Ban Escalation

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_BAD_BEHAVIOR` | Enables bad behavior counting based on HTTP responses. | `true`, `false` | `true` | `true` |
| `WAF_SITE_BAD_BEHAVIOR_STATUS_CODES` | Status codes counted as bad behavior incidents. | JSON number array | `[400,401,405,444]` | `[400,401,403,429,444]` |
| `WAF_SITE_BAD_BEHAVIOR_BAN_TIME_SECONDS` | Initial ban duration in seconds. | Integer `>= 0` | `300` | `600` |
| `WAF_SITE_BAD_BEHAVIOR_THRESHOLD` | Trigger threshold for bad behavior. | Integer `>= 0` | `120` | `60` |
| `WAF_SITE_BAD_BEHAVIOR_COUNT_TIME_SECONDS` | Counting window for bad behavior. | Integer `>= 0` | `120` | `60` |
| `WAF_SITE_BAN_ESCALATION_ENABLED` | Enables staged ban escalation. | `true`, `false` | `false` | `true` |
| `WAF_SITE_BAN_ESCALATION_SCOPE` | Scope where escalation applies. | `current_site`, `all_sites` | `all_sites` | `all_sites` |
| `WAF_SITE_BAN_ESCALATION_STAGES_SECONDS` | Escalation stage durations. `0` means permanent and must be last. | JSON number array | `[300,86400,0]` | `[600,3600,86400,0]` |

## Blacklists, DNSBL, And Signature Lists

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_BLACKLIST` | Enables signature-based blacklists for the service. | `true`, `false` | `false` | `true` |
| `WAF_SITE_USE_DNSBL` | Enables DNSBL checks. | `true`, `false` | `false` | `true` |
| `WAF_SITE_BLACKLIST_IP` | Local IP/CIDR blacklist. | JSON string array | `[]` | `["203.0.113.0/24"]` |
| `WAF_SITE_BLACKLIST_RDNS` | Reverse-DNS blacklist. | JSON string or pattern array | `[]` | `["scanner.example.net"]` |
| `WAF_SITE_BLACKLIST_ASN` | ASN blacklist. | JSON string or number array | `[]` | `["AS12345"]` |
| `WAF_SITE_BLACKLIST_USER_AGENT` | User-Agent blacklist or regex list. | JSON string array | `[]` | `["curl/.*","sqlmap","HeadlessChrome"]` |
| `WAF_SITE_BLACKLIST_URI` | URI blacklist or regex list. | JSON string array | `[]` | `["/wp-admin","/\\.env","/phpmyadmin"]` |
| `WAF_SITE_BLACKLIST_IP_URLS` | External sources for IP blacklists. | JSON URL array | `[]` | `["https://lists.example.net/ip-deny.txt"]` |
| `WAF_SITE_BLACKLIST_RDNS_URLS` | External sources for RDNS blacklists. | JSON URL array | `[]` | `["https://lists.example.net/rdns-deny.txt"]` |
| `WAF_SITE_BLACKLIST_ASN_URLS` | External sources for ASN blacklists. | JSON URL array | `[]` | `["https://lists.example.net/asn-deny.txt"]` |
| `WAF_SITE_BLACKLIST_USER_AGENT_URLS` | External sources for User-Agent blacklists. | JSON URL array | `[]` | `["https://lists.example.net/ua-deny.txt"]` |
| `WAF_SITE_BLACKLIST_URI_URLS` | External sources for URI blacklists. | JSON URL array | `[]` | `["https://lists.example.net/uri-deny.txt"]` |

## Connection Limiting And Request Limiting

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_LIMIT_CONN` | Enables concurrent connection limiting. | `true`, `false` | `true` | `true` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP1` | Concurrent HTTP/1.x connection limit. | Integer `>= 0` | `200` | `80` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP2` | Concurrent HTTP/2 connection or stream limit in the product model. | Integer `>= 0` | `400` | `200` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP3` | Concurrent HTTP/3 connection or stream limit in the product model. | Integer `>= 0` | `400` | `200` |
| `WAF_SITE_USE_LIMIT_REQ` | Enables request rate limiting. | `true`, `false` | `true` | `true` |
| `WAF_SITE_LIMIT_REQ_URL` | Base path for the global rate limit. | Path starting with `/` | `/` | `/` |
| `WAF_SITE_LIMIT_REQ_RATE` | Global rate-limit value in `Nr/s` format. | String like `20r/s` | `120r/s` | `20r/s` |
| `WAF_SITE_CUSTOM_LIMIT_RULES` | Route-specific rate-limit rules. | JSON array of `{ "path": "/...", "rate": "Nr/s" }` | `[]` | `[{"path":"/login","rate":"10r/s"},{"path":"/api/2/envelope/","rate":"30r/s"}]` |

## Anti-Bot And Challenge

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ANTIBOT_CHALLENGE` | Anti-bot challenge type. | `no`, `js`, `recaptcha`, `hcaptcha`, `turnstile` | `no` | `turnstile` |
| `WAF_SITE_ANTIBOT_URI` | Challenge page path. | Path | `/challenge` | `/challenge` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SCORE` | reCAPTCHA score threshold. | Number `0..1` | `0.7` | `0.8` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SITEKEY` | reCAPTCHA site key. | String or empty | Empty | `recaptcha-site-key` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SECRET` | reCAPTCHA secret. | String or empty | Empty | `recaptcha-secret` |
| `WAF_SITE_ANTIBOT_HCAPTCHA_SITEKEY` | hCaptcha site key. | String or empty | Empty | `hcaptcha-site-key` |
| `WAF_SITE_ANTIBOT_HCAPTCHA_SECRET` | hCaptcha secret. | String or empty | Empty | `hcaptcha-secret` |
| `WAF_SITE_ANTIBOT_TURNSTILE_SITEKEY` | Cloudflare Turnstile site key. | String or empty | Empty | `turnstile-site-key` |
| `WAF_SITE_ANTIBOT_TURNSTILE_SECRET` | Cloudflare Turnstile secret. | String or empty | Empty | `turnstile-secret` |

## HTTP Basic Auth

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_AUTH_BASIC` | Enables HTTP Basic Auth on the service. | `true`, `false` | `false` | `false` |
| `WAF_SITE_AUTH_BASIC_LOCATION` | Where Basic Auth applies. | Usually `sitewide` | `sitewide` | `sitewide` |
| `WAF_SITE_AUTH_BASIC_USER` | Basic Auth username. | String | `changeme` | `ops` |
| `WAF_SITE_AUTH_BASIC_PASSWORD` | Basic Auth password. | String or empty | Empty | `strong-password` |
| `WAF_SITE_AUTH_BASIC_TEXT` | Realm text for Basic Auth. | String | `Restricted area` | `Private admin area` |

## Geo Policies

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_BLACKLIST_COUNTRY` | Countries or regional groups denied by policy. | JSON array of country and group codes | `[]` | `["RU","CN","APAC"]` |
| `WAF_SITE_WHITELIST_COUNTRY` | Countries or regional groups explicitly allowed. | JSON array of country and group codes | `[]` | `["DE","FR","PL"]` |

## ModSecurity And CRS

| Parameter | Purpose | Allowed values | Default | Example |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_MODSECURITY` | Enables ModSecurity for the service. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_MODSECURITY_CRS_PLUGINS` | Enables CRS plugins on top of the base CRS. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_MODSECURITY_CUSTOM_CONFIGURATION` | Enables a custom ModSecurity config for the service. | `true`, `false` | `false` | `false` |
| `WAF_SITE_MODSECURITY_CRS_VERSION` | CRS version to use. | Product-supported string or number | `4` | `4` |
| `WAF_SITE_MODSECURITY_CRS_PLUGINS` | CRS plugin IDs. | JSON string array | `[]` | `["plugin-php","plugin-wordpress"]` |
| `WAF_SITE_MODSECURITY_CUSTOM_PATH` | Path to the custom ModSecurity config inside the runtime bundle. | String | `modsec/anomaly_score.conf` | `modsec/custom-rules.conf` |
| `WAF_SITE_MODSECURITY_CUSTOM_CONTENT` | Inline content of the custom ModSecurity config. | Multiline string or empty | Empty | `SecRuleEngine On` |

## Practical Recommendations

- For export and raw editing, prefer storing the full `.env` rather than only changed fields. That makes environment-to-environment transfers safer.
- `WAF_SITE_CUSTOM_LIMIT_RULES`, blacklist arrays, and geo lists should remain valid JSON. Do not mix JSON syntax with comma-separated free text in the same field.
- If the service uses HTTPS to upstream, enable `WAF_SITE_REVERSE_PROXY_SSL_SNI` and set `WAF_SITE_REVERSE_PROXY_SSL_SNI_NAME` when the upstream expects a specific `server_name`.
- Secrets such as `*_SECRET`, `WAF_SITE_AUTH_BASIC_PASSWORD`, and ACME credentials are better injected from secure secret storage instead of being shared in plain files.

## Minimal Example

```env
WAF_SITE_ID=app-example-com
WAF_SITE_PRIMARY_HOST=app.example.com
WAF_SITE_ENABLED=true
WAF_SITE_TLS_ENABLED=true
WAF_SITE_CERTIFICATE_ID=app-example-com-tls
WAF_SITE_SECURITY_MODE=block
WAF_SITE_UPSTREAM_ID=app-example-com-upstream
WAF_SITE_UPSTREAM_HOST=10.0.10.15
WAF_SITE_UPSTREAM_PORT=8080
WAF_SITE_UPSTREAM_SCHEME=http
WAF_SITE_USE_REVERSE_PROXY=true
WAF_SITE_REVERSE_PROXY_HOST=http://10.0.10.15:8080
WAF_SITE_REVERSE_PROXY_URL=/
WAF_SITE_ALLOWED_METHODS=["GET","POST","HEAD","OPTIONS"]
WAF_SITE_USE_LIMIT_REQ=true
WAF_SITE_LIMIT_REQ_URL=/
WAF_SITE_LIMIT_REQ_RATE=20r/s
WAF_SITE_CUSTOM_LIMIT_RULES=[{"path":"/login","rate":"10r/s"},{"path":"/api/","rate":"40r/s"}]
WAF_SITE_USE_MODSECURITY=true
WAF_SITE_MODSECURITY_CRS_VERSION=4
```
