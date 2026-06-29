import { normalizeArray } from "./sites.routing-merge.js";
import {
  normalizeAPIPositiveEndpointPolicies,
  normalizeAntibotExclusionRules,
  normalizeAntibotChallengeRules,
  normalizeCustomLimitRules,
  normalizeServiceProfile,
  normalizeStringArray,
  normalizeGeoTimeWindows,
} from "./sites.normalize.js";
import { normalizeAuthBasicUsers, normalizeAuthSessionTTLMinutes } from "./sites.auth-geo.js";
import {
  BAN_SCOPE_VALUES,
  normalizeBanEscalationStages,
  normalizeEmail,
  resolveReverseProxyHost,
} from "./sites.traffic-helpers.js";

export function defaultSiteDraft() {
  return {
    id: "", primary_host: "", enabled: true, tls_enabled: true, tls_self_signed: false, certificate_id: "", security_mode: "block",
    service_profile: "balanced", adaptive_model_enabled: false, upstream_id: "", upstream_host: "ui", upstream_port: 80, upstream_scheme: "http",
    auto_lets_encrypt: true, use_lets_encrypt_staging: false, use_lets_encrypt_wildcard: false, certificate_authority_server: "letsencrypt",
    acme_account_email: "", use_reverse_proxy: true, reverse_proxy_host: "http://upstream-server:8080", reverse_proxy_url: "/",
    reverse_proxy_custom_host: "", reverse_proxy_ssl_sni: false, reverse_proxy_ssl_sni_name: "", reverse_proxy_websocket: true, reverse_proxy_keepalive: true,
    health_check_enabled: false, health_check_path: "/health", health_check_interval_seconds: 10, health_check_fail_threshold: 3,
    pass_host_header: true, send_x_forwarded_for: true, send_x_forwarded_proto: true, send_x_real_ip: false, allowed_methods: ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"],
    max_client_size: "100m", http2: true, http3: false, ssl_protocols: ["TLSv1.2", "TLSv1.3"], http_strict_parsing: false, cookie_flags: "* SameSite=Lax",
    content_security_policy: "", permissions_policy: [], keep_upstream_headers: ["*"], referrer_policy: "no-referrer-when-downgrade",
    hsts_enabled: true, hsts_max_age_seconds: 15552000, hsts_include_subdomains: false, hsts_preload: false, use_cors: false, cors_allowed_origins: ["*"],
    use_allowlist: false, use_exceptions: false, access_allowlist: [], exceptions_ip: [], exceptions_uri: [], access_denylist: [], use_bad_behavior: true, bad_behavior_status_codes: [400, 401, 405, 444],
    bad_behavior_ban_time_seconds: 300, bad_behavior_threshold: 120, bad_behavior_count_time_seconds: 120, ban_escalation_enabled: false, ban_escalation_scope: "all_sites",
    ban_escalation_stages_seconds: [300, 86400, 0], use_blacklist: false, use_dnsbl: false, blacklist_ip: [], blacklist_rdns: [], blacklist_asn: [], blacklist_user_agent: [],
    blacklist_uri: [], blacklist_ip_urls: [], blacklist_rdns_urls: [], blacklist_asn_urls: [], blacklist_user_agent_urls: [], blacklist_uri_urls: [], blacklist_ja3: [], blacklist_ja3_urls: [], use_limit_conn: true,
    limit_conn_max_http1: 200, limit_conn_max_http2: 400, limit_conn_max_http3: 400, use_limit_req: true, limit_req_url: "/", limit_req_rate: "120r/s", custom_limit_rules: [],
    antibot_challenge: "no", antibot_challenge_template: "v1", antibot_uri: "/challenge", antibot_scanner_auto_ban_enabled: true, antibot_recaptcha_score: 0.7, antibot_recaptcha_sitekey: "", antibot_recaptcha_secret: "",
    antibot_hcaptcha_sitekey: "", antibot_hcaptcha_secret: "", antibot_turnstile_sitekey: "", antibot_turnstile_secret: "", antibot_exclusion_rules: [], challenge_escalation_enabled: false,
    challenge_escalation_mode: "javascript", antibot_challenge_rules: [], use_auth_basic: false, auth_basic_location: "sitewide", auth_basic_user: "changeme", auth_basic_password: "",
    auth_basic_text: "Restricted area", auth_basic_users: [{ username: "changeme", password: "", enabled: true, last_login_at: "" }], auth_basic_session_inactivity_minutes: 60,
    blacklist_country: [], whitelist_country: [], geo_time_windows: [], api_positive_security_enabled: false, api_positive_openapi_schema_ref: "", api_positive_enforcement_mode: "monitor",
    api_positive_default_action: "allow", api_positive_endpoint_policies: [], use_modsecurity: true, use_modsecurity_crs_plugins: true, use_modsecurity_custom_configuration: false,
    modsecurity_crs_version: "4", modsecurity_crs_plugins: [], modsecurity_custom_path: "modsec/anomaly_score.conf", modsecurity_custom_content: "",
    use_ws_inspection: false, ws_block_patterns: [], ws_max_message_bytes: 0, ws_rate_msg_per_sec: 0,
    mtls_enabled: false, mtls_optional: false, mtls_verify_depth: 2, mtls_client_ca_ref: "", mtls_pass_headers: false,
    upstream_mtls_enabled: false, upstream_mtls_cert_ref: "", upstream_mtls_key_ref: "", upstream_mtls_ca_ref: "",
    use_custom_error_pages: true,
    disabled_error_pages: [],
  };
}

export function siteDraftFromData(site, upstream, tlsConfig) {
  const base = {
    id: site?.id || "", primary_host: site?.primary_host || "", enabled: Boolean(site?.enabled ?? true), tls_enabled: Boolean(tlsConfig), tls_self_signed: false,
    certificate_id: tlsConfig?.certificate_id || (site?.id ? `${site.id}-tls` : ""), security_mode: "block",
    upstream_id: upstream?.id || (site?.id ? `${site.id}-upstream` : ""), upstream_host: upstream?.host || "ui", upstream_port: upstream?.port || 80, upstream_scheme: upstream?.scheme || "http"
  };
  return { ...defaultSiteDraft(), ...base };
}

export function draftToEasyProfile(draft) {
  const siteID = String(draft.id || "").trim().toLowerCase();
  const primaryHost = String(draft.primary_host || "").trim().toLowerCase();
  const reverseProxyHost = resolveReverseProxyHost(draft, draft.reverse_proxy_host);
  const reverseProxyURL = String(draft.reverse_proxy_url || "").trim();
  const limitReqURL = String(draft.limit_req_url || "").trim();
  const limitReqRateRaw = String(draft.limit_req_rate || "").trim().toLowerCase().replace(/\s+/g, "");
  const limitReqRate = /^\d+r\/s$/.test(limitReqRateRaw) ? limitReqRateRaw : "100r/s";
  const authBasicText = String(draft.auth_basic_text || "").trim() || "Restricted area";
  const authUsers = normalizeAuthBasicUsers(draft.auth_basic_users);
  const firstUser = authUsers[0] || { username: "", password: "" };
  const authSessionTTLMinutes = normalizeAuthSessionTTLMinutes(draft.auth_basic_session_inactivity_minutes);
  const customPath = String(draft.modsecurity_custom_path || "").trim() || "modsec/anomaly_score.conf";
  const securityMode = ["transparent", "monitor", "block"].includes(String(draft.security_mode || "").trim().toLowerCase()) ? String(draft.security_mode || "").trim().toLowerCase() : "block";
  const banEscalationScope = BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase()) ? String(draft.ban_escalation_scope || "").trim().toLowerCase() : "all_sites";
  const banEscalationStages = normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds);

  return {
    site_id: siteID,
    front_service: {
      server_name: primaryHost, security_mode: securityMode, profile: normalizeServiceProfile(draft.service_profile), adaptive_model_enabled: Boolean(draft.adaptive_model_enabled),
      auto_lets_encrypt: draft.auto_lets_encrypt, use_lets_encrypt_staging: draft.use_lets_encrypt_staging, use_lets_encrypt_wildcard: draft.use_lets_encrypt_wildcard,
      certificate_authority_server: draft.certificate_authority_server, acme_account_email: normalizeEmail(draft.acme_account_email)
    },
    upstream_routing: {
      use_reverse_proxy: draft.use_reverse_proxy, reverse_proxy_host: reverseProxyHost, reverse_proxy_url: reverseProxyURL.startsWith("/") ? reverseProxyURL : "/",
      reverse_proxy_custom_host: draft.reverse_proxy_custom_host, reverse_proxy_ssl_sni: draft.reverse_proxy_ssl_sni, reverse_proxy_ssl_sni_name: draft.reverse_proxy_ssl_sni_name,
      reverse_proxy_websocket: draft.reverse_proxy_websocket, reverse_proxy_keepalive: draft.reverse_proxy_keepalive, disable_host_header: !draft.pass_host_header,
      disable_x_forwarded_for: !draft.send_x_forwarded_for, disable_x_forwarded_proto: !draft.send_x_forwarded_proto, enable_x_real_ip: draft.send_x_real_ip
    },
    http_behavior: { allowed_methods: draft.allowed_methods, max_client_size: draft.max_client_size, http2: draft.http2, http3: draft.http3, ssl_protocols: draft.ssl_protocols, http_strict_parsing: Boolean(draft.http_strict_parsing) },
    http_headers: {
      cookie_flags: draft.cookie_flags, content_security_policy: draft.content_security_policy, permissions_policy: draft.permissions_policy, keep_upstream_headers: draft.keep_upstream_headers,
      referrer_policy: draft.referrer_policy, hsts_enabled: Boolean(draft.hsts_enabled), hsts_max_age_seconds: Number(draft.hsts_max_age_seconds || 0),
      hsts_include_subdomains: Boolean(draft.hsts_include_subdomains), hsts_preload: Boolean(draft.hsts_preload), use_cors: draft.use_cors, cors_allowed_origins: draft.cors_allowed_origins
    },
    security_behavior_and_limits: {
      use_bad_behavior: draft.use_bad_behavior, bad_behavior_status_codes: draft.bad_behavior_status_codes, bad_behavior_ban_time_seconds: draft.bad_behavior_ban_time_seconds,
      bad_behavior_threshold: draft.bad_behavior_threshold, bad_behavior_count_time_seconds: draft.bad_behavior_count_time_seconds, ban_escalation_enabled: draft.ban_escalation_enabled,
      ban_escalation_scope: banEscalationScope, ban_escalation_stages_seconds: banEscalationStages, use_exceptions: draft.use_exceptions, exceptions_ip: draft.exceptions_ip, exceptions_uri: draft.exceptions_uri,
      use_blacklist: draft.use_blacklist, use_dnsbl: draft.use_dnsbl, blacklist_ip: draft.blacklist_ip, blacklist_rdns: draft.blacklist_rdns, blacklist_asn: draft.blacklist_asn,
      blacklist_user_agent: draft.blacklist_user_agent, blacklist_uri: draft.blacklist_uri, blacklist_ip_urls: draft.blacklist_ip_urls, blacklist_rdns_urls: draft.blacklist_rdns_urls,
      blacklist_asn_urls: draft.blacklist_asn_urls, blacklist_user_agent_urls: draft.blacklist_user_agent_urls, blacklist_uri_urls: draft.blacklist_uri_urls, use_limit_conn: draft.use_limit_conn,
      limit_conn_max_http1: draft.limit_conn_max_http1, limit_conn_max_http2: draft.limit_conn_max_http2, limit_conn_max_http3: draft.limit_conn_max_http3, use_limit_req: draft.use_limit_req,
      limit_req_url: limitReqURL.startsWith("/") ? limitReqURL : "/", limit_req_rate: limitReqRate, custom_limit_rules: normalizeCustomLimitRules(draft.custom_limit_rules)
    },
    security_antibot: {
      antibot_challenge: draft.antibot_challenge, antibot_challenge_template: draft.antibot_challenge_template || "v1", antibot_uri: draft.antibot_uri, scanner_auto_ban_enabled: Boolean(draft.antibot_scanner_auto_ban_enabled),
      antibot_recaptcha_score: draft.antibot_recaptcha_score, antibot_recaptcha_sitekey: draft.antibot_recaptcha_sitekey, antibot_recaptcha_secret: draft.antibot_recaptcha_secret,
      antibot_hcaptcha_sitekey: draft.antibot_hcaptcha_sitekey, antibot_hcaptcha_secret: draft.antibot_hcaptcha_secret, antibot_turnstile_sitekey: draft.antibot_turnstile_sitekey,
      antibot_turnstile_secret: draft.antibot_turnstile_secret, exclusion_rules: normalizeAntibotExclusionRules(draft.antibot_exclusion_rules), challenge_escalation_enabled: Boolean(draft.challenge_escalation_enabled),
      challenge_escalation_mode: String(draft.challenge_escalation_mode || "javascript").trim().toLowerCase() || "javascript", challenge_rules: normalizeAntibotChallengeRules(draft.antibot_challenge_rules)
    },
    security_auth_basic: { use_auth_basic: draft.use_auth_basic, auth_basic_location: "sitewide", auth_basic_user: firstUser.username, auth_basic_password: firstUser.password, auth_basic_text: authBasicText, users: authUsers, session_inactivity_minutes: authSessionTTLMinutes },
    security_country_policy: { blacklist_country: draft.blacklist_country, whitelist_country: draft.whitelist_country, show_geo_block_page: Boolean(draft.show_geo_block_page), geo_time_windows: normalizeGeoTimeWindows(draft.geo_time_windows) },
    security_api_positive: {
      use_api_positive_security: Boolean(draft.api_positive_security_enabled), openapi_schema_ref: String(draft.api_positive_openapi_schema_ref || "").trim(),
      enforcement_mode: String(draft.api_positive_enforcement_mode || "monitor").trim().toLowerCase() || "monitor",
      default_action: String(draft.api_positive_default_action || "allow").trim().toLowerCase() || "allow", endpoint_policies: normalizeAPIPositiveEndpointPolicies(draft.api_positive_endpoint_policies)
    },
    security_modsecurity: {
      use_modsecurity: draft.use_modsecurity, use_modsecurity_crs_plugins: draft.use_modsecurity_crs_plugins, use_modsecurity_custom_configuration: draft.use_modsecurity_custom_configuration,
      modsecurity_crs_version: draft.modsecurity_crs_version, modsecurity_crs_plugins: draft.modsecurity_crs_plugins, custom_configuration: { path: customPath, content: draft.modsecurity_custom_content }
    },
    use_custom_error_pages: Boolean(draft.use_custom_error_pages ?? true),
    disabled_error_pages: Array.isArray(draft.disabled_error_pages) ? draft.disabled_error_pages : [],
  };
}
