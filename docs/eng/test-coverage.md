# WAF UI Test Coverage — Reference Document

**Version:** 1.4.7
**Date:** 2026-06-29
**Status:** 158 tab tests + full runtime WAF e2e — all green

This document serves as proof of WAF functional coverage per UI tab.
If something regresses, run one of the commands below — green tests mean the system works as expected.

---

## Running the Tests

```bash
# All tab tests (fast, ~1 second)
go test ./compiler/internal/compiler/ ./control-plane/internal/easysiteprofiles/ -count=1

# Per tab
go test ./compiler/internal/compiler/ -run "TestFront_" -v
go test ./compiler/internal/compiler/ -run "TestUpstream_" -v
go test ./compiler/internal/compiler/ -run "TestHeaders_" -v
go test ./compiler/internal/compiler/ -run "TestTraffic_" -v
go test ./control-plane/internal/easysiteprofiles/ -run "TestBanEscalation_" -v
go test ./compiler/internal/compiler/ -run "TestAntibot_" -v
go test ./compiler/internal/compiler/ -run "TestGeo_" -v
go test ./compiler/internal/compiler/ -run "TestModsec_" -v
go test ./compiler/internal/compiler/ -run "TestWebSocket_" -v
go test ./compiler/internal/compiler/ -run "TestVirtualPatches_" -v
go test ./compiler/internal/compiler/ -run "TestErrorPages_" -v

# Full runtime-stack WAF reference
E2E_FILTER=TestE2EBehavioral sh scripts/run-e2e-tests.sh

# Full suite
go test ./...
```

---

## Results by Tab

### Tab 1 — Front (19 tests)

File: `compiler/internal/compiler/tab01_front_test.go`

| Test | What it verifies |
|------|-----------------|
| TestFront_HSTS_FullDirective | HSTS with includeSubDomains and preload is present in config |
| TestFront_HSTS_MaxAgeOnly | HSTS with max-age only, no subdomains/preload |
| TestFront_HSTS_Disabled_NoHeader | When HSTS=off the directive is not generated |
| TestFront_HSTS_DefaultMaxAge_WhenZero | When MaxAge=0 a default value is substituted |
| TestFront_AllowedMethods_LimitedSet | limit_except contains only the permitted methods |
| TestFront_AllowedMethods_DefaultWhenEmpty | Empty list produces the default method set |
| TestFront_AllowedMethods_BlocksOtherMethods | Disallowed methods return 405 |
| TestFront_MaxClientSize_IsSet | client_max_body_size is set in config |
| TestFront_HttpStrictParsing_Enabled | ignore_invalid_headers on + underscores_in_headers off |
| TestFront_HttpStrictParsing_Disabled | When disabled — opposite directive values |
| TestFront_MTLS_Required_DirectivesPresent | ssl_verify_client required + ssl_client_certificate |
| TestFront_MTLS_Optional_DirectivesPresent | ssl_verify_client optional for soft verification |
| TestFront_MTLS_Disabled_NoDirectives | mTLS=off — no ssl_verify_client in config |
| TestFront_MTLS_PassHeaders_Enabled | X-Client-Verify and X-Client-DN are forwarded upstream |
| TestFront_MTLS_Validation_NoCA | Error when mTLS is enabled without a CA certificate |
| TestFront_MTLS_Validation_NegativeDepth | Error when certificate chain depth is negative |
| TestFront_SecurityMode_Block_ModSecOn | SecurityMode=block creates modsecurity/easy/&lt;id&gt;.conf artifact |
| TestFront_SecurityMode_Disabled_NoModSec | SecurityMode=disabled does not create a modsec artifact |
| TestFront_SecurityMode_Monitor_NoModSecArtifact | SecurityMode=monitor does not create a modsec artifact |

---

### Tab 2 — Upstream (20 tests)

File: `compiler/internal/compiler/tab02_upstream_test.go`

| Test | What it verifies |
|------|-----------------|
| TestUpstream_PassHostHeader_WithCustomHost | proxy_set_header Host is generated only when both PassHostHeader and CustomHost are set |
| TestUpstream_PassHostHeader_NoCustomHost_NoOverride | Without CustomHost there is no proxy_set_header Host |
| TestUpstream_CustomHost_IsSet | Custom host value is placed in the directive |
| TestUpstream_CustomHost_Empty_NoOverride | Empty CustomHost — no Host override |
| TestUpstream_SSLSNI_Enabled | proxy_ssl_server_name on + proxy_ssl_name for SNI |
| TestUpstream_SSLSNI_Disabled | When SNI is off — those directives are absent |
| TestUpstream_Websocket_Enabled | Upgrade and Connection "" for WebSocket proxying |
| TestUpstream_Websocket_Disabled_NoUpgrade | Without WebSocket there is no Upgrade header |
| TestUpstream_Keepalive_Enabled_ConnectionEmpty | keepalive 32 + Connection "" in the connection pool |
| TestUpstream_Keepalive_Disabled_NoConnectionEmpty | When keepalive is off there is no Connection "" |
| TestUpstream_HealthCheck_Enabled | proxy_next_upstream + keepalive in sites template |
| TestUpstream_HealthCheck_Disabled_NoNextUpstream | Without HealthCheck there is no proxy_next_upstream |
| TestUpstream_XForwardedFor_Enabled | X-Forwarded-For is forwarded upstream |
| TestUpstream_XForwardedFor_Disabled_Cleared | X-Forwarded-For is cleared when disabled |
| TestUpstream_XForwardedProto_Enabled | X-Forwarded-Proto is forwarded upstream |
| TestUpstream_XRealIP_Enabled | X-Real-IP is forwarded upstream |
| TestUpstream_MTLS_DirectivesPresent | proxy_ssl_certificate + proxy_ssl_certificate_key + proxy_ssl_trusted_certificate |
| TestUpstream_MTLS_Disabled_NoProxySSLCert | Without upstream mTLS there is no proxy_ssl_certificate |
| TestUpstream_MTLS_Validation_NoCert | Error when upstream mTLS has no certificate |
| TestUpstream_MTLS_Validation_NoKey | Error when upstream mTLS has no key |

---

### Tab 3 — HTTP Headers (18 tests)

File: `compiler/internal/compiler/tab03_headers_test.go`

| Test | What it verifies |
|------|-----------------|
| TestHeaders_ReferrerPolicy_Set | add_header Referrer-Policy is set |
| TestHeaders_ReferrerPolicy_Empty_NoHeader | Empty value — directive is absent |
| TestHeaders_CSP_Set | add_header Content-Security-Policy is set |
| TestHeaders_CSP_Empty_NoHeader | Empty value — directive is absent |
| TestHeaders_PermissionsPolicy_Set | add_header Permissions-Policy is set |
| TestHeaders_PermissionsPolicy_Empty_NoHeader | Empty value — directive is absent |
| TestHeaders_CORS_Enabled_AllowOrigin | Access-Control-Allow-Origin is added when CORS=on |
| TestHeaders_CORS_Disabled_NoAllowOrigin | Without CORS there are no CORS headers |
| TestHeaders_CORS_MultipleOrigins | Multiple allowed origins in the header |
| TestHeaders_CookieFlags_Set | proxy_cookie_flags is set for cookies |
| TestHeaders_CookieFlags_Empty_NoDirective | Empty flags — no directive |
| TestHeaders_CookieFlags_Secure | Secure flag is passed in proxy_cookie_flags |
| TestHeaders_KeepUpstreamHeaders_Single | proxy_pass_header for a single upstream header |
| TestHeaders_KeepUpstreamHeaders_Multiple | proxy_pass_header for multiple upstream headers |
| TestHeaders_KeepUpstreamHeaders_Empty_NoDirective | Empty list — no proxy_pass_header |
| TestHeaders_HSTS_FullDirective | Strict-Transport-Security with full parameter set |
| TestHeaders_HSTS_Disabled_NoSTS | Without HSTS there is no Strict-Transport-Security |
| TestHeaders_AllSecurityHeaders_Together | All security headers together in a single config |

---

### Tab 4 — Traffic (24 tests)

File: `compiler/internal/compiler/tab04_traffic_test.go`

| Test | What it verifies |
|------|-----------------|
| TestTraffic_BlacklistIP_Single | deny &lt;IP&gt; is generated in config |
| TestTraffic_BlacklistIP_Multiple | Multiple IPs in the blacklist |
| TestTraffic_BlacklistIP_Empty_NoDeny | Empty list — no deny directives |
| TestTraffic_BlacklistUA_Single | User-Agent in blacklist → guard + return 403 |
| TestTraffic_BlacklistUA_BlockReturns403 | Blocked UA returns 403 |
| TestTraffic_BlacklistUA_ExceptionGuard | UA exceptions bypass via exception guard |
| TestTraffic_BlacklistURI_Single | URI in blacklist → waf guard |
| TestTraffic_BlacklistURI_Returns403 | Blocked URI returns 403 |
| TestTraffic_BlacklistURI_Multiple | Multiple URIs in the blacklist |
| TestTraffic_ExceptionsURI_Single | One URI exception generates exception guard |
| TestTraffic_ExceptionsURI_Multiple | Multiple URI exceptions |
| TestTraffic_ExceptionsURI_Empty_NoExtraGuard | Empty exceptions — no extra guard directives |
| TestTraffic_ExceptionsURI_BypassesBlacklistIP | URI exception bypasses IP block |
| TestTraffic_BlacklistCountry_Single | waf_country_guard for a blocked country |
| TestTraffic_BlacklistCountry_Multiple | Multiple countries in the blacklist |
| TestTraffic_WhitelistCountry_Single | Whitelist mode: !~ for allowed countries |
| TestTraffic_WhitelistCountry_BlocksOthers | Whitelist blocks all countries except allowed ones |
| TestTraffic_LimitConn_Enabled | l4guard/config.json contains conn_limit |
| TestTraffic_LimitConn_Disabled_NoDirective | Without LimitConn there is no conn_limit in l4guard |
| TestTraffic_LimitReq_Enabled | l4guard/config.json contains rate_per_second |
| TestTraffic_LimitReq_Disabled_NoDirective | Without LimitReq there is no rate_per_second in l4guard |
| TestTraffic_BadBehavior_Enabled | Bad behavior detection is enabled |
| TestTraffic_BadBehavior_BanReturns429 | Ban for bad behavior returns 429 |
| TestTraffic_BadBehavior_EscalationReturns403 | Ban escalation returns 403 |

---

### Tab 5 — Ban Escalation (16 tests)

File: `control-plane/internal/easysiteprofiles/tab05_ban_escalation_test.go`
Package: `control-plane/internal/easysiteprofiles`

| Test | What it verifies |
|------|-----------------|
| TestBanEscalation_Normalize_ScopeDefault | Empty scope → default "all_sites" |
| TestBanEscalation_Normalize_ScopeUpperCase | Uppercase scope is normalized to lowercase |
| TestBanEscalation_Normalize_CurrentSite | Scope "current_site" normalizes correctly |
| TestBanEscalation_Normalize_StagesDefault | Empty stages → default [300, 86400, 0] |
| TestBanEscalation_Normalize_StagesDeduped | Duplicate zeros in stages are deduplicated (one kept) |
| TestBanEscalation_Validate_InvalidScope | Invalid scope → validation error |
| TestBanEscalation_Validate_AllSitesScope_Valid | Scope "all_sites" passes validation |
| TestBanEscalation_Validate_CurrentSiteScope_Valid | Scope "current_site" passes validation |
| TestBanEscalation_Validate_EmptyStages_WhenEnabled | Empty stages when enabled=true → error |
| TestBanEscalation_Validate_TooManyStages | More than 12 stages → error |
| TestBanEscalation_Validate_NegativeStage | Negative stage value → error |
| TestBanEscalation_Validate_PermanentNotLast | Zero (permanent) ban not last → error |
| TestBanEscalation_Validate_PermanentAsLastStage_Valid | Zero ban as last stage → valid |
| TestBanEscalation_Validate_SingleFiniteStage_Valid | Single finite stage without zero → valid |
| TestBanEscalation_Validate_MaxStages_Valid | Exactly 12 stages → valid |
| TestBanEscalation_Validate_Disabled_NoStagesRequired | When enabled=false empty stages do not cause an error |

---

### Tab 6 — Antibot (14 tests)

File: `compiler/internal/compiler/tab06_antibot_test.go`

| Test | What it verifies |
|------|-----------------|
| TestAntibot_Disabled_NoGuardVars | When antibot is disabled there are no guard variables in config |
| TestAntibot_Javascript_ChallengeVar | Challenge mode "javascript" is set in config |
| TestAntibot_Javascript_RedirectURI | AntibotURI generates redirect 302 |
| TestAntibot_Recaptcha_ChallengeVar | Challenge mode "recaptcha" is set in config |
| TestAntibot_Hcaptcha_ChallengeVar | Challenge mode "hcaptcha" is set in config |
| TestAntibot_Turnstile_ChallengeVar | Challenge mode "turnstile" (Cloudflare) is set |
| TestAntibot_ScannerAutoBan_GuardPresent | ScannerAutoBan=true adds scanner guard + return 403 |
| TestAntibot_ScannerAutoBan_Disabled_NoScannerGuard | Without ScannerAutoBan there is no scanner guard |
| TestAntibot_ExclusionRule_BypassesChallenge | ExclusionRule generates waf_antibot_exception_guard |
| TestAntibot_CookieGuard_VerifiesSession | CookieGuard checks waf_antibot_verified cookie |
| TestAntibot_ChallengeEscalation_Enabled | Challenge escalation: turnstile in config + X-WAF-Antibot-Provider |
| TestAntibot_ChallengeEscalation_WithRules | Escalation with URI-based rules |
| TestAntibot_UnverifiedRequest_RedirectOrBlock | Unverified request → redirect or block |
| TestAntibot_DebugHeader_Present | Debug header X-WAF-Antibot-Mode is present |

---

### Tab 7 — Geo-filtering / Time Windows (16 tests)

File: `compiler/internal/compiler/tab07_geo_test.go`

| Test | What it verifies |
|------|-----------------|
| TestGeo_TimeWindow_SnippetInSiteConf | "geo time-window enforcement" snippet appears in site.conf |
| TestGeo_TimeWindow_BlockAction_Returns403 | Action=block generates return 403 |
| TestGeo_TimeWindow_AllowAction_No403 | Action=allow does not generate return 403 |
| TestGeo_TimeWindow_HourRange_InSnippet | Hour range is present in server snippet |
| TestGeo_TimeWindow_DaysOfWeek_InSnippet | Days of week are present in server snippet |
| TestGeo_TimeWindow_ExceptionGuard_Bypass | Exception guard bypasses geo restriction |
| TestGeo_TimeWindow_MapArtifact_Generated | Artifact nginx/geo-timewindow/&lt;id&gt;.conf is created |
| TestGeo_TimeWindow_HttpConf_HourMap | HTTP config contains map $time_iso8601 for hours |
| TestGeo_TimeWindow_HttpConf_CountryMap | HTTP config contains country mapping ("JP" 1, "KR" 1) |
| TestGeo_TimeWindow_InvalidWindow_Ignored | Window with HoursStart >= HoursEnd is silently discarded |
| TestGeo_TimeWindow_Empty_NoSnippet | Empty GeoTimeWindows — no snippet in config |
| TestGeo_TimeWindow_MultipleWindows | Multiple windows generate indices _0_ and _1_ |
| TestGeo_Validate_InvalidAction | Action other than block/allow → validation error |
| TestGeo_Validate_InvalidCountryCode | Non-existent country code → validation error |
| TestGeo_Validate_HoursStartGEHoursEnd | HoursStart >= HoursEnd → validation error |
| TestGeo_Validate_InvalidDayOfWeek | Day of week outside 0-6 → validation error |

---

### Tab 8 — ModSecurity (9 tests)

File: `compiler/internal/compiler/tab08_modsec_test.go`

| Test | What it verifies |
|------|-----------------|
| TestModsec_Enabled_ArtifactCreated | UseModSecurity=true creates modsecurity/easy/&lt;id&gt;.conf artifact |
| TestModsec_Disabled_NoArtifact | UseModSecurity=false — no modsec artifact |
| TestModsec_RulesFileDirective_InSiteConf | modsecurity_rules_file /etc/waf/modsecurity/easy/&lt;id&gt;.conf is present |
| TestModsec_Disabled_NoRulesFileDirective | Without UseModSecurity there is no modsecurity_rules_file directive |
| TestModsec_CRSVersion_InArtifact | CRS version is reflected in the artifact content |
| TestModsec_Plugins_InArtifact | CRS plugins are listed in the artifact |
| TestModsec_CustomContent_InArtifact | Custom SecRule rules are included in the artifact |
| TestModsec_SecurityMode_Disabled_ArtifactStillCreated | UseModSecurity=true creates the artifact even when SecurityMode=disabled |
| TestModsec_SecurityMode_Disabled_UseModSecFalse_NoArtifact | UseModSecurity=false + disabled — no artifact |

---

### Tab 9 — WebSocket Inspection (10 tests)

File: `compiler/internal/compiler/tab09_websocket_test.go`

| Test | What it verifies |
|------|-----------------|
| TestWebSocket_Inspection_SnippetInSiteConf | When UseWSInspection=true the pattern appears in site.conf (Lua block) |
| TestWebSocket_Inspection_Disabled_NoSnippet | UseWSInspection=false — no WS snippet in config |
| TestWebSocket_BlockPattern_InSnippet | WSBlockPatterns are present in the generated snippet |
| TestWebSocket_MaxMessageBytes_InSnippet | WSMaxMessageBytes is present in snippet |
| TestWebSocket_RateMsgPerSec_InSnippet | WSRateMsgPerSec is present in snippet |
| TestWebSocket_NoPatterns_NoSnippet | No patterns and no limits → empty snippet |
| TestWebSocket_Validate_InvalidPattern | Invalid regex → error with index reference |
| TestWebSocket_Validate_ValidPatterns | Valid patterns pass validation without error |
| TestWebSocket_Normalize_DedupPatterns | Duplicate patterns are removed during normalization |
| TestWebSocket_Normalize_EmptyRemoved | Empty strings are removed during normalization |

---

### Tab 10 — Virtual Patches (10 tests)

File: `compiler/internal/compiler/tab10_virtualpatches_test.go`

| Test | What it verifies |
|------|-----------------|
| TestVirtualPatches_Block_URI_Rule | SecRule REQUEST_URI with deny,status:403 for target=uri |
| TestVirtualPatches_Block_Body_Rule | SecRule REQUEST_BODY with deny,status:403 for target=body |
| TestVirtualPatches_Block_Header_Rule | SecRule REQUEST_HEADERS with deny,status:403 for target=header |
| TestVirtualPatches_Monitor_URI_Rule | Monitor action → pass without deny for target=uri |
| TestVirtualPatches_Monitor_Body_Rule | Monitor action → pass without deny for target=body |
| TestVirtualPatches_ID_InRuleMsg | Patch ID is present in the SecRule msg |
| TestVirtualPatches_Multiple_Rules | Multiple patches — all patterns present in artifact |
| TestVirtualPatches_Empty_NoRules | Nil patches — no virtual patch SecRules |
| TestVirtualPatches_Integration_InModsecArtifact | Patch is present in modsecurity/easy/&lt;id&gt;.conf artifact |
| TestVirtualPatches_SecurityMode_Disabled_ArtifactStillCreated | UseModSecurity=true creates artifact even when SecurityMode=disabled |

---

### Tab 11 — Custom Error Pages (2 tests)

File: `compiler/internal/compiler/tab11_errorpages_test.go`

| Test | What it verifies |
|------|-----------------|
| TestErrorPages_Enabled_HasProxyInterceptAndErrorPages | `UseCustomErrorPages=true` enables `proxy_intercept_errors on`, generates `error_page`, and emits site-scoped `/__waf_errors/<site>/...` paths |
| TestErrorPages_Disabled_NoProxyIntercept | `UseCustomErrorPages=false` does not add `proxy_intercept_errors on` and does not generate `error_page` directives |

---

## Runtime e2e — WAF Behavioral Reference (31 scenarios)

File: `ui/tests/e2e_behavioral_test.go`

This suite runs in the full Docker Compose stack (`control-plane`, `runtime`, `postgres`, `vault`, `upstream-echo`) and verifies real WAF behavior on HTTP requests, not just artifact generation.

| Scenario group | What it proves |
|----------------|----------------|
| Blacklist IP / User-Agent / URI | Requests are actually blocked with 403; disabling rules restores pass-through |
| RateLimit and custom route limits | Burst traffic receives 429; route-specific limits trigger on the target URI only |
| Antibot and cookie flow | A new client receives 302 to challenge, verify sets the cookie, then upstream returns 200 |
| SecurityMode transparent / monitor | Passive modes do not leave active deny/ModSecurity/auth/antibot blocking paths enabled |
| Custom error pages | Branded 403 page is enabled, while `disabled_error_pages` falls back to the short standard page |
| Scanner auto-ban | `sqlmap`/`nikto` signatures are blocked with 403 independently of the main antibot challenge |
| ModSecurity CRS | SQL injection and XSS are blocked with 403; legitimate requests pass |
| Geo policy | Blacklist/whitelist country configs apply and 451 is available for geo-block responses |
| Basic Auth gate | Anonymous requests get 302 to `/auth`; verify endpoint returns 204 and sets auth cookie |
| Response headers | CORS, CSP, Referrer-Policy, Permissions-Policy and HSTS are present in real responses |
| Exceptions URI | URI exceptions bypass blacklist guard |
| Virtual patches | Per-site URI SecRule blocks the request with 403 |
| Antibot exclusions/rules/escalation | Exclusions skip challenge, per-path rules change challenge, two-layer escalation routes to stage1 |
| Cookie flags / upstream headers / strict parsing / WS / geo time windows / JA3 | Configs apply without nginx reload errors and preserve runtime invariants |

In the latest green run: 30 scenarios PASS, the mTLS upload scenario SKIP because no test CA file is present in the environment; the overall `TestE2EBehavioral` completed with `PASS`.

---

## Summary Table

| Tab | File | Tests | Package |
|-----|------|-------|---------|
| 1 — Front | tab01_front_test.go | 19 | compiler/internal/compiler |
| 2 — Upstream | tab02_upstream_test.go | 20 | compiler/internal/compiler |
| 3 — Headers | tab03_headers_test.go | 18 | compiler/internal/compiler |
| 4 — Traffic | tab04_traffic_test.go | 24 | compiler/internal/compiler |
| 5 — Ban Escalation | tab05_ban_escalation_test.go | 16 | control-plane/internal/easysiteprofiles |
| 6 — Antibot | tab06_antibot_test.go | 14 | compiler/internal/compiler |
| 7 — Geo | tab07_geo_test.go | 16 | compiler/internal/compiler |
| 8 — ModSecurity | tab08_modsec_test.go | 9 | compiler/internal/compiler |
| 9 — WebSocket | tab09_websocket_test.go | 10 | compiler/internal/compiler |
| 10 — Virtual Patches | tab10_virtualpatches_test.go | 10 | compiler/internal/compiler |
| 11 — Error Pages | tab11_errorpages_test.go | 2 | compiler/internal/compiler |
| **Total tab tests** | | **158** | |
| Runtime WAF e2e | e2e_behavioral_test.go | 31 scenarios | ui/tests |

---

## Architectural Findings from Tests

The following facts are confirmed by tests — not by documentation, but by actual compiler behavior:

**Upstream Host override:**
`PassHostHeader` without `CustomHost` does not generate `proxy_set_header Host`. Both fields are required simultaneously (template: `{{- if and .PassHostHeader .ReverseProxyCustomHost }}`).

**HealthCheck and Keepalive:**
`proxy_next_upstream` and `keepalive` directives are generated only in `sites/site.conf.tmpl` (non-easy template), not in the easy template. Tested via `nginxSiteData`.

**Traffic limiting:**
`LimitConn` and `LimitReq` go into `l4guard/config.json`, not the nginx config. `conn_limit = max(200, LimitConnMaxHTTP1)` — takes the maximum, minimum is 200.

**Geo time windows:**
The server snippet contains `$waf_geo_tw_*` variables; country codes only appear in the HTTP map artifact `nginx/geo-timewindow/&lt;id&gt;.conf`. Invalid windows (HoursStart >= HoursEnd) are silently discarded.

**ModSecurity:**
`UseModSecurity=true` creates the `modsecurity/easy/&lt;id&gt;.conf` artifact regardless of `SecurityMode`. Setting `SecurityMode=disabled` without `UseModSecurity=false` does not suppress the artifact.

**ChallengeEscalation:**
Escalation replaces the base challenge with the escalation mode. The string `"javascript"` disappears from config when escalation is active with mode `"turnstile"`.

**WebSocket inspection:**
The WS snippet is a Lua block, not nginx directives. The snippet is not generated when the pattern list is empty and limits are zero, even if `UseWSInspection=true`.

**Runtime WAF:**
The behavioral e2e is the reference for actual WAF behavior: it checks runtime nginx responses, cookies, redirects, status codes and applied revisions. Tab tests prove artifact structure; runtime e2e proves that those artifacts work together.

---

## Release Integration

The preflight in `scripts/release.ps1` runs tab tests, `go test ./...`, and full `TestE2EBehavioral` before publishing. Any single test failure aborts the release with exit code 1.

Project rule (`.work/PROMT.md`): every new UI feature requires a corresponding test in `tab0X_*_test.go` or `easysiteprofiles/tab0X_*_test.go` in the same PR.
