import { hydrateSiteDraftPart2, draftToEasyProfilePart2, validateDraftPart2 } from "./sites.draft-profile-part2.js";

export function siteDraftFromData(site, upstream, tlsConfig, defaultSiteDraft) {
  const base = {
    id: site?.id || "",
    primary_host: site?.primary_host || "",
    enabled: Boolean(site?.enabled ?? true),
    tls_enabled: Boolean(tlsConfig),
    tls_self_signed: false,
    certificate_id: tlsConfig?.certificate_id || (site?.id ? `${site.id}-tls` : ""),
    security_mode: "block",
    upstream_id: upstream?.id || (site?.id ? `${site.id}-upstream` : ""),
    upstream_host: upstream?.host || "ui",
    upstream_port: upstream?.port || 80,
    upstream_scheme: upstream?.scheme || "http"
  };
  return { ...defaultSiteDraft(), ...base };
}

export function applyEasyProfileToDraft(draft, profile, deps) {
  if (!profile || typeof profile !== "object") {
    return draft;
  }
  const front = profile.front_service || {};
  const upstream = profile.upstream_routing || {};
  const httpBehavior = profile.http_behavior || {};
  const httpHeaders = profile.http_headers || {};
  const security = profile.security_behavior_and_limits || {};
  const antibot = profile.security_antibot || {};
  const authBasic = profile.security_auth_basic || {};
  const country = profile.security_country_policy || {};
  const apiPositive = profile.security_api_positive || {};
  const modsecurity = profile.security_modsecurity || {};
  return {
    ...draft,
    primary_host: front.server_name || draft.primary_host,
    security_mode: front.security_mode || draft.security_mode,
    service_profile: deps.normalizeServiceProfile(front.profile || draft.service_profile),
    adaptive_model_enabled: Boolean(front.adaptive_model_enabled ?? draft.adaptive_model_enabled),
    auto_lets_encrypt: Boolean(front.auto_lets_encrypt ?? draft.auto_lets_encrypt),
    use_lets_encrypt_staging: Boolean(front.use_lets_encrypt_staging ?? draft.use_lets_encrypt_staging),
    use_lets_encrypt_wildcard: Boolean(front.use_lets_encrypt_wildcard ?? draft.use_lets_encrypt_wildcard),
    certificate_authority_server: front.certificate_authority_server || draft.certificate_authority_server,
    acme_account_email: deps.normalizeEmail(front.acme_account_email || draft.acme_account_email),
    use_reverse_proxy: Boolean(upstream.use_reverse_proxy ?? draft.use_reverse_proxy),
    reverse_proxy_host: deps.resolveReverseProxyHost(draft, upstream.reverse_proxy_host) || draft.reverse_proxy_host,
    reverse_proxy_url: upstream.reverse_proxy_url || draft.reverse_proxy_url,
    reverse_proxy_custom_host: upstream.reverse_proxy_custom_host || draft.reverse_proxy_custom_host,
    reverse_proxy_ssl_sni: Boolean(upstream.reverse_proxy_ssl_sni ?? draft.reverse_proxy_ssl_sni),
    reverse_proxy_ssl_sni_name: upstream.reverse_proxy_ssl_sni_name || draft.reverse_proxy_ssl_sni_name,
    reverse_proxy_websocket: Boolean(upstream.reverse_proxy_websocket ?? draft.reverse_proxy_websocket),
    reverse_proxy_keepalive: Boolean(upstream.reverse_proxy_keepalive ?? draft.reverse_proxy_keepalive),
    pass_host_header: !(Boolean(upstream.disable_host_header ?? !draft.pass_host_header)),
    send_x_forwarded_for: !(Boolean(upstream.disable_x_forwarded_for ?? !draft.send_x_forwarded_for)),
    send_x_forwarded_proto: !(Boolean(upstream.disable_x_forwarded_proto ?? !draft.send_x_forwarded_proto)),
    send_x_real_ip: Boolean(upstream.enable_x_real_ip ?? draft.send_x_real_ip),
    allowed_methods: deps.normalizeStringArray(httpBehavior.allowed_methods).length ? deps.normalizeStringArray(httpBehavior.allowed_methods) : draft.allowed_methods,
    max_client_size: httpBehavior.max_client_size || draft.max_client_size,
    http2: Boolean(httpBehavior.http2 ?? draft.http2),
    http3: Boolean(httpBehavior.http3 ?? draft.http3),
    ssl_protocols: deps.normalizeStringArray(httpBehavior.ssl_protocols).length ? deps.normalizeStringArray(httpBehavior.ssl_protocols) : draft.ssl_protocols,
    cookie_flags: httpHeaders.cookie_flags || draft.cookie_flags,
    content_security_policy: httpHeaders.content_security_policy || draft.content_security_policy,
    permissions_policy: deps.normalizeStringArray(httpHeaders.permissions_policy),
    keep_upstream_headers: deps.normalizeStringArray(httpHeaders.keep_upstream_headers).length ? deps.normalizeStringArray(httpHeaders.keep_upstream_headers) : draft.keep_upstream_headers,
    referrer_policy: httpHeaders.referrer_policy || draft.referrer_policy,
    hsts_enabled: Boolean(httpHeaders.hsts_enabled ?? draft.hsts_enabled),
    hsts_max_age_seconds: Number(httpHeaders.hsts_max_age_seconds ?? draft.hsts_max_age_seconds),
    hsts_include_subdomains: Boolean(httpHeaders.hsts_include_subdomains ?? draft.hsts_include_subdomains),
    hsts_preload: Boolean(httpHeaders.hsts_preload ?? draft.hsts_preload),
    use_cors: Boolean(httpHeaders.use_cors ?? draft.use_cors),
    cors_allowed_origins: deps.normalizeStringArray(httpHeaders.cors_allowed_origins).length ? deps.normalizeStringArray(httpHeaders.cors_allowed_origins) : draft.cors_allowed_origins,
    use_bad_behavior: Boolean(security.use_bad_behavior ?? draft.use_bad_behavior),
    bad_behavior_status_codes: deps.normalizeArray(security.bad_behavior_status_codes).map((item) => Number(item)).filter((item) => Number.isInteger(item)),
    bad_behavior_ban_time_seconds: Number(security.bad_behavior_ban_time_seconds ?? draft.bad_behavior_ban_time_seconds),
    bad_behavior_threshold: Number(security.bad_behavior_threshold ?? draft.bad_behavior_threshold),
    bad_behavior_count_time_seconds: Number(security.bad_behavior_count_time_seconds ?? draft.bad_behavior_count_time_seconds),
    ban_escalation_enabled: Boolean(security.ban_escalation_enabled ?? draft.ban_escalation_enabled),
    ban_escalation_scope: deps.BAN_SCOPE_VALUES.includes(String(security.ban_escalation_scope || "").trim().toLowerCase())
      ? String(security.ban_escalation_scope || "").trim().toLowerCase()
      : draft.ban_escalation_scope,
    ban_escalation_stages_seconds: deps.normalizeBanEscalationStages(
      security.ban_escalation_stages_seconds,
      Number(security.bad_behavior_ban_time_seconds ?? draft.bad_behavior_ban_time_seconds)
    ),
    use_exceptions: Boolean(security.use_exceptions ?? draft.use_exceptions),
    exceptions_ip: deps.normalizeStringArray(security.exceptions_ip),
    use_blacklist: Boolean(security.use_blacklist ?? draft.use_blacklist),
    use_dnsbl: Boolean(security.use_dnsbl ?? draft.use_dnsbl),
    blacklist_ip: deps.normalizeStringArray(security.blacklist_ip),
    blacklist_rdns: deps.normalizeStringArray(security.blacklist_rdns),
    blacklist_asn: deps.normalizeStringArray(security.blacklist_asn),
    blacklist_user_agent: deps.normalizeStringArray(security.blacklist_user_agent),
    blacklist_uri: deps.normalizeStringArray(security.blacklist_uri),
    blacklist_ip_urls: deps.normalizeStringArray(security.blacklist_ip_urls),
    blacklist_rdns_urls: deps.normalizeStringArray(security.blacklist_rdns_urls),
    blacklist_asn_urls: deps.normalizeStringArray(security.blacklist_asn_urls),
    blacklist_user_agent_urls: deps.normalizeStringArray(security.blacklist_user_agent_urls),
    blacklist_uri_urls: deps.normalizeStringArray(security.blacklist_uri_urls),
    use_limit_conn: Boolean(security.use_limit_conn ?? draft.use_limit_conn),
    limit_conn_max_http1: Number(security.limit_conn_max_http1 ?? draft.limit_conn_max_http1),
    limit_conn_max_http2: Number(security.limit_conn_max_http2 ?? draft.limit_conn_max_http2),
    limit_conn_max_http3: Number(security.limit_conn_max_http3 ?? draft.limit_conn_max_http3),
    use_limit_req: Boolean(security.use_limit_req ?? draft.use_limit_req),
    limit_req_url: security.limit_req_url || draft.limit_req_url,
    limit_req_rate: security.limit_req_rate || draft.limit_req_rate,
    custom_limit_rules: deps.normalizeCustomLimitRules(security.custom_limit_rules),
    antibot_challenge: antibot.antibot_challenge || draft.antibot_challenge,
    antibot_uri: antibot.antibot_uri || draft.antibot_uri,
    antibot_scanner_auto_ban_enabled: Boolean(antibot.scanner_auto_ban_enabled ?? draft.antibot_scanner_auto_ban_enabled),
    antibot_recaptcha_score: Number(antibot.antibot_recaptcha_score ?? draft.antibot_recaptcha_score),
    antibot_recaptcha_sitekey: antibot.antibot_recaptcha_sitekey || draft.antibot_recaptcha_sitekey,
    antibot_recaptcha_secret: antibot.antibot_recaptcha_secret || draft.antibot_recaptcha_secret,
    antibot_hcaptcha_sitekey: antibot.antibot_hcaptcha_sitekey || draft.antibot_hcaptcha_sitekey,
    antibot_hcaptcha_secret: antibot.antibot_hcaptcha_secret || draft.antibot_hcaptcha_secret,
    antibot_turnstile_sitekey: antibot.antibot_turnstile_sitekey || draft.antibot_turnstile_sitekey,
    antibot_turnstile_secret: antibot.antibot_turnstile_secret || draft.antibot_turnstile_secret,
    antibot_exclusion_rules: deps.normalizeAntibotExclusionRules(antibot.exclusion_rules || draft.antibot_exclusion_rules),
    challenge_escalation_enabled: Boolean(antibot.challenge_escalation_enabled ?? draft.challenge_escalation_enabled),
    challenge_escalation_mode: String(antibot.challenge_escalation_mode || draft.challenge_escalation_mode).trim().toLowerCase() || "javascript",
    antibot_challenge_rules: deps.normalizeAntibotChallengeRules(antibot.challenge_rules || draft.antibot_challenge_rules),
    use_auth_basic: Boolean(authBasic.use_auth_basic ?? draft.use_auth_basic),
    auth_mode: deps.normalizeAuthMode(authBasic.auth_mode || draft.auth_mode),
    auth_order: deps.normalizeAuthOrder(authBasic.auth_order || draft.auth_order),
    auth_basic_location: authBasic.auth_basic_location || draft.auth_basic_location,
    auth_basic_user: authBasic.auth_basic_user || draft.auth_basic_user,
    auth_basic_password: authBasic.auth_basic_password || draft.auth_basic_password,
    auth_basic_text: authBasic.auth_basic_text || draft.auth_basic_text,
    auth_basic_users: deps.normalizeAuthBasicUsers(authBasic.users),
    auth_exclusion_rules: deps.normalizeAuthExclusionRules(authBasic.exclusion_rules || draft.auth_exclusion_rules),
    auth_service_tokens: deps.normalizeAuthServiceTokens(authBasic.service_tokens || draft.auth_service_tokens),
    auth_basic_session_inactivity_minutes: deps.normalizeAuthSessionTTLMinutes(
      authBasic.session_inactivity_minutes ?? draft.auth_basic_session_inactivity_minutes
    ),
    blacklist_country: deps.normalizeStringArray(country.blacklist_country),
    whitelist_country: deps.normalizeStringArray(country.whitelist_country),
    api_positive_security_enabled: Boolean(apiPositive.use_api_positive_security ?? draft.api_positive_security_enabled),
    api_positive_openapi_schema_ref: String(apiPositive.openapi_schema_ref || draft.api_positive_openapi_schema_ref),
    api_positive_enforcement_mode: String(apiPositive.enforcement_mode || draft.api_positive_enforcement_mode).trim().toLowerCase() || "monitor",
    api_positive_default_action: String(apiPositive.default_action || draft.api_positive_default_action).trim().toLowerCase() || "allow",
    api_positive_endpoint_policies: deps.normalizeAPIPositiveEndpointPolicies(apiPositive.endpoint_policies || draft.api_positive_endpoint_policies),
    use_modsecurity: Boolean(modsecurity.use_modsecurity ?? draft.use_modsecurity),
    use_modsecurity_crs_plugins: Boolean(modsecurity.use_modsecurity_crs_plugins ?? draft.use_modsecurity_crs_plugins),
    use_modsecurity_custom_configuration: Boolean(modsecurity.use_modsecurity_custom_configuration ?? draft.use_modsecurity_custom_configuration),
    modsecurity_crs_version: String(modsecurity.modsecurity_crs_version || draft.modsecurity_crs_version),
    modsecurity_crs_plugins: deps.normalizeStringArray(modsecurity.modsecurity_crs_plugins),
    modsecurity_custom_path: modsecurity.custom_configuration?.path || draft.modsecurity_custom_path,
    modsecurity_custom_content: modsecurity.custom_configuration?.content || draft.modsecurity_custom_content
  };
}


export async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null, deps) {
  return hydrateSiteDraftPart2(ctx, site, upstream, tlsConfig, accessPolicy, { ...deps, siteDraftFromData, applyEasyProfileToDraft });
}

export function draftToEasyProfile(draft, deps) {
  return draftToEasyProfilePart2(draft, deps);
}

export function validateDraft(draft, ctx, deps) {
  return validateDraftPart2(draft, ctx, deps);
}
