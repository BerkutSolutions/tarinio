export async function hydrateSiteDraftPart2(ctx, site, upstream, tlsConfig, accessPolicy = null, deps) {
  let draft = site ? deps.siteDraftFromData(site, upstream, tlsConfig, deps.defaultSiteDraft) : deps.defaultSiteDraft();
  if (site?.id) {
    try {
      const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`);
      draft = applyEasyProfileToDraft(draft, profile, deps);
    } catch (error) {
      const secondaryDump = await deps.tryGetJSON("/api-app/easy-site-profiles");
      const secondaryProfile = deps.findEasyProfile(secondaryDump, site.id);
      if (secondaryProfile) {
        draft = applyEasyProfileToDraft(draft, secondaryProfile, deps);
      } else if (error?.status !== 404) {
        console.warn("failed to load easy-site-profile", error);
      }
    }
  }
  draft.access_allowlist = deps.normalizeStringArray(accessPolicy?.allowlist);
  draft.access_denylist = deps.normalizeStringArray(accessPolicy?.denylist);
  draft.use_allowlist = draft.access_allowlist.length > 0;
  if (!deps.normalizeStringArray(draft.exceptions_ip).length) {
    draft.exceptions_ip = [...draft.access_allowlist];
  }
  if (!draft.use_exceptions) {
    draft.use_exceptions = deps.normalizeStringArray(draft.exceptions_ip).length > 0;
  }
  return draft;
}

export function draftToEasyProfilePart2(draft, deps) {
  const siteID = String(draft.id || "").trim().toLowerCase();
  const primaryHost = String(draft.primary_host || "").trim().toLowerCase();
  const reverseProxyHost = deps.resolveReverseProxyHost(draft, draft.reverse_proxy_host);
  const reverseProxyURL = String(draft.reverse_proxy_url || "").trim();
  const limitReqURL = String(draft.limit_req_url || "").trim();
  const limitReqRateRaw = String(draft.limit_req_rate || "").trim().toLowerCase().replace(/\s+/g, "");
  const limitReqRate = /^\d+r\/s$/.test(limitReqRateRaw) ? limitReqRateRaw : "100r/s";
  const authBasicLocation = "sitewide";
  const authBasicText = String(draft.auth_basic_text || "").trim() || "Restricted area";
  const authUsers = deps.normalizeAuthBasicUsers(draft.auth_basic_users);
  const authMode = deps.normalizeAuthMode(draft.auth_mode);
  const authOrder = deps.normalizeAuthOrder(draft.auth_order);
  const authExclusionRules = deps.normalizeAuthExclusionRules(draft.auth_exclusion_rules);
  const authServiceTokens = deps.normalizeAuthServiceTokens(draft.auth_service_tokens);
  const firstUser = authUsers[0] || { username: "", password: "" };
  const authSessionTTLMinutes = deps.normalizeAuthSessionTTLMinutes(draft.auth_basic_session_inactivity_minutes);
  const customPath = String(draft.modsecurity_custom_path || "").trim() || "modsec/anomaly_score.conf";
  const securityMode = ["transparent", "monitor", "block"].includes(String(draft.security_mode || "").trim().toLowerCase())
    ? String(draft.security_mode || "").trim().toLowerCase()
    : "block";
  const banEscalationScope = deps.BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase())
    ? String(draft.ban_escalation_scope || "").trim().toLowerCase()
    : "all_sites";
  const banEscalationStages = deps.normalizeBanEscalationStages(
    draft.ban_escalation_stages_seconds,
    draft.bad_behavior_ban_time_seconds
  );

  return {
    site_id: siteID,
    front_service: {
      server_name: primaryHost,
      security_mode: securityMode,
      profile: deps.normalizeServiceProfile(draft.service_profile),
      adaptive_model_enabled: Boolean(draft.adaptive_model_enabled),
      auto_lets_encrypt: draft.auto_lets_encrypt,
      use_lets_encrypt_staging: draft.use_lets_encrypt_staging,
      use_lets_encrypt_wildcard: draft.use_lets_encrypt_wildcard,
      certificate_authority_server: draft.certificate_authority_server,
      acme_account_email: deps.normalizeEmail(draft.acme_account_email),
      mtls_enabled: Boolean(draft.mtls_enabled),
      mtls_optional: Boolean(draft.mtls_optional),
      mtls_verify_depth: Math.max(0, Number.parseInt(String(draft.mtls_verify_depth || "2"), 10) || 2),
      mtls_client_ca_ref: String(draft.mtls_client_ca_ref || "").trim(),
      mtls_pass_headers: Boolean(draft.mtls_pass_headers)
    },
    upstream_routing: {
      use_reverse_proxy: draft.use_reverse_proxy,
      reverse_proxy_host: reverseProxyHost,
      reverse_proxy_url: reverseProxyURL.startsWith("/") ? reverseProxyURL : "/",
      reverse_proxy_custom_host: draft.reverse_proxy_custom_host,
      reverse_proxy_ssl_sni: draft.reverse_proxy_ssl_sni,
      reverse_proxy_ssl_sni_name: draft.reverse_proxy_ssl_sni_name,
      reverse_proxy_websocket: draft.reverse_proxy_websocket,
      reverse_proxy_keepalive: draft.reverse_proxy_keepalive,
      disable_host_header: !draft.pass_host_header,
      disable_x_forwarded_for: !draft.send_x_forwarded_for,
      disable_x_forwarded_proto: !draft.send_x_forwarded_proto,
      enable_x_real_ip: draft.send_x_real_ip,
      health_check_enabled: Boolean(draft.health_check_enabled),
      health_check_path: String(draft.health_check_path || "/health").trim(),
      health_check_interval_seconds: Number(draft.health_check_interval_seconds || 10),
      health_check_fail_threshold: Number(draft.health_check_fail_threshold || 3),
      upstream_mtls_enabled: Boolean(draft.upstream_mtls_enabled),
      upstream_mtls_cert_ref: String(draft.upstream_mtls_cert_ref || "").trim(),
      upstream_mtls_key_ref: String(draft.upstream_mtls_key_ref || "").trim(),
      upstream_mtls_ca_ref: String(draft.upstream_mtls_ca_ref || "").trim()
    },
    http_behavior: {
      allowed_methods: draft.allowed_methods,
      max_client_size: draft.max_client_size,
      http2: draft.http2,
      http3: draft.http3,
      ssl_protocols: draft.ssl_protocols
    },
    http_headers: {
      cookie_flags: draft.cookie_flags,
      content_security_policy: draft.content_security_policy,
      permissions_policy: draft.permissions_policy,
      keep_upstream_headers: draft.keep_upstream_headers,
      referrer_policy: draft.referrer_policy,
      hsts_enabled: Boolean(draft.hsts_enabled),
      hsts_max_age_seconds: Number(draft.hsts_max_age_seconds || 0),
      hsts_include_subdomains: Boolean(draft.hsts_include_subdomains),
      hsts_preload: Boolean(draft.hsts_preload),
      use_cors: draft.use_cors,
      cors_allowed_origins: draft.cors_allowed_origins
    },
    security_behavior_and_limits: {
      use_bad_behavior: draft.use_bad_behavior,
      bad_behavior_status_codes: draft.bad_behavior_status_codes,
      bad_behavior_ban_time_seconds: draft.bad_behavior_ban_time_seconds,
      bad_behavior_threshold: draft.bad_behavior_threshold,
      bad_behavior_count_time_seconds: draft.bad_behavior_count_time_seconds,
      ban_escalation_enabled: draft.ban_escalation_enabled,
      ban_escalation_scope: banEscalationScope,
      ban_escalation_stages_seconds: banEscalationStages,
      use_exceptions: draft.use_exceptions,
      exceptions_ip: draft.exceptions_ip,
      exceptions_uri: draft.exceptions_uri,
      use_blacklist: draft.use_blacklist,
      use_dnsbl: draft.use_dnsbl,
      blacklist_ip: draft.blacklist_ip,
      blacklist_rdns: draft.blacklist_rdns,
      blacklist_asn: draft.blacklist_asn,
      blacklist_user_agent: draft.blacklist_user_agent,
      blacklist_uri: draft.blacklist_uri,
      blacklist_ja3: draft.blacklist_ja3,
      blacklist_ip_urls: draft.blacklist_ip_urls,
      blacklist_rdns_urls: draft.blacklist_rdns_urls,
      blacklist_asn_urls: draft.blacklist_asn_urls,
      blacklist_user_agent_urls: draft.blacklist_user_agent_urls,
      blacklist_uri_urls: draft.blacklist_uri_urls,
      blacklist_ja3_urls: draft.blacklist_ja3_urls,
      use_limit_conn: draft.use_limit_conn,
      limit_conn_max_http1: draft.limit_conn_max_http1,
      limit_conn_max_http2: draft.limit_conn_max_http2,
      limit_conn_max_http3: draft.limit_conn_max_http3,
      use_limit_req: draft.use_limit_req,
      limit_req_url: limitReqURL.startsWith("/") ? limitReqURL : "/",
      limit_req_rate: limitReqRate,
      custom_limit_rules: deps.normalizeCustomLimitRules(draft.custom_limit_rules)
    },
    security_antibot: {
      antibot_challenge: draft.antibot_challenge,
      antibot_uri: draft.antibot_uri,
      scanner_auto_ban_enabled: Boolean(draft.antibot_scanner_auto_ban_enabled),
      antibot_recaptcha_score: draft.antibot_recaptcha_score,
      antibot_recaptcha_sitekey: draft.antibot_recaptcha_sitekey,
      antibot_recaptcha_secret: draft.antibot_recaptcha_secret,
      antibot_hcaptcha_sitekey: draft.antibot_hcaptcha_sitekey,
      antibot_hcaptcha_secret: draft.antibot_hcaptcha_secret,
      antibot_turnstile_sitekey: draft.antibot_turnstile_sitekey,
      antibot_turnstile_secret: draft.antibot_turnstile_secret,
      exclusion_rules: deps.normalizeAntibotExclusionRules(draft.antibot_exclusion_rules),
      challenge_escalation_enabled: Boolean(draft.challenge_escalation_enabled),
      challenge_escalation_mode: String(draft.challenge_escalation_mode || "javascript").trim().toLowerCase() || "javascript",
      challenge_rules: deps.normalizeAntibotChallengeRules(draft.antibot_challenge_rules)
    },
    security_auth_basic: {
      use_auth_basic: draft.use_auth_basic,
      auth_mode: authMode,
      auth_order: authOrder,
      auth_basic_location: authBasicLocation,
      auth_basic_user: firstUser.username,
      auth_basic_password: firstUser.password,
      auth_basic_text: authBasicText,
      users: authUsers,
      exclusion_rules: authExclusionRules,
      service_tokens: authServiceTokens,
      session_inactivity_minutes: authSessionTTLMinutes
    },
    security_country_policy: {
      blacklist_country: draft.blacklist_country,
      whitelist_country: draft.whitelist_country,
      geo_time_windows: deps.normalizeGeoTimeWindows(draft.geo_time_windows)
    },
    security_api_positive: {
      use_api_positive_security: Boolean(draft.api_positive_security_enabled),
      openapi_schema_ref: String(draft.api_positive_openapi_schema_ref || "").trim(),
      enforcement_mode: String(draft.api_positive_enforcement_mode || "monitor").trim().toLowerCase() || "monitor",
      default_action: String(draft.api_positive_default_action || "allow").trim().toLowerCase() || "allow",
      endpoint_policies: deps.normalizeAPIPositiveEndpointPolicies(draft.api_positive_endpoint_policies)
    },
    security_modsecurity: {
      use_modsecurity: draft.use_modsecurity,
      use_modsecurity_crs_plugins: draft.use_modsecurity_crs_plugins,
      use_modsecurity_custom_configuration: draft.use_modsecurity_custom_configuration,
      modsecurity_crs_version: draft.modsecurity_crs_version,
      modsecurity_crs_plugins: draft.modsecurity_crs_plugins,
      custom_configuration: {
        path: customPath,
        content: draft.modsecurity_custom_content
      }
    },
    security_websocket: {
      use_ws_inspection: Boolean(draft.use_ws_inspection),
      ws_block_patterns: deps.normalizeWSBlockPatterns(draft.ws_block_patterns),
      ws_max_message_bytes: Math.max(0, Number.parseInt(String(draft.ws_max_message_bytes || "0"), 10) || 0),
      ws_rate_msg_per_sec: Math.max(0, Number.parseInt(String(draft.ws_rate_msg_per_sec || "0"), 10) || 0)
    }
  };
}

export function validateDraftPart2(draft, ctx, deps) {
  if (!draft.id.trim()) {
    return ctx.t("sites.validation.siteIdRequired");
  }
  if (!draft.primary_host.trim()) {
    return ctx.t("sites.validation.primaryHostRequired");
  }
  if (!draft.upstream_id.trim()) {
    return ctx.t("sites.validation.upstreamIdRequired");
  }
  if (!draft.upstream_host.trim()) {
    return ctx.t("sites.validation.upstreamHostRequired");
  }
  if (!Number.isInteger(draft.upstream_port) || draft.upstream_port < 1 || draft.upstream_port > 65535) {
    return ctx.t("sites.validation.portRange");
  }
  if (!deps.normalizeStringArray(draft.allowed_methods).length) {
    return ctx.t("sites.validation.allowedMethodsRequired");
  }
  if (draft.use_bad_behavior && !deps.normalizeArray(draft.bad_behavior_status_codes).length) {
    return ctx.t("sites.validation.badBehaviorStatusCodesRequired");
  }
  if (draft.use_bad_behavior && (!Number.isFinite(draft.bad_behavior_ban_time_seconds) || draft.bad_behavior_ban_time_seconds < 0)) {
    return ctx.t("sites.validation.badBehaviorBanDuration");
  }
  if (draft.ban_escalation_enabled) {
    if (!deps.BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase())) {
      return ctx.t("sites.validation.banEscalationScope");
    }
    const stages = deps.normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds);
    if (!stages.length) {
      return ctx.t("sites.validation.banEscalationStagesRequired");
    }
    if (stages.length > 12) {
      return ctx.t("sites.validation.banEscalationStagesLimit");
    }
    for (let i = 0; i < stages.length; i += 1) {
      const value = stages[i];
      if (!Number.isFinite(value) || value < 0) {
        return ctx.t("sites.validation.banEscalationStageValue");
      }
      if (value === 0 && i !== stages.length - 1) {
        return ctx.t("sites.validation.banEscalationPermanentLast");
      }
    }
  }
  if (draft.use_limit_req && !String(draft.limit_req_rate || "").trim()) {
    return ctx.t("sites.validation.limitReqRateRequired");
  }
  if (draft.use_limit_req && !/^\d+r\/s$/i.test(String(draft.limit_req_rate || "").trim().replace(/\s+/g, ""))) {
    return ctx.t("sites.validation.limitReqRateFormat");
  }
  if (deps.normalizeCustomLimitRules(draft.custom_limit_rules).length > 32) {
    return ctx.t("sites.validation.customLimitRulesLimit");
  }
  if (draft.antibot_challenge === "no" && deps.normalizeAntibotExclusionRules(draft.antibot_exclusion_rules).length) {
    return ctx.t("sites.validation.antibotExclusionsRequireEnabled");
  }
  if (draft.hsts_enabled && (!Number.isFinite(draft.hsts_max_age_seconds) || Number(draft.hsts_max_age_seconds) <= 0)) {
    return ctx.t("sites.validation.hstsMaxAgePositive");
  }
  if (draft.hsts_preload && !draft.hsts_enabled) {
    return ctx.t("sites.validation.hstsPreloadNeedsEnabled");
  }
  if (draft.hsts_include_subdomains && !draft.hsts_enabled) {
    return ctx.t("sites.validation.hstsPreloadNeedsEnabled");
  }
  if (draft.hsts_preload && !draft.hsts_include_subdomains) {
    return ctx.t("sites.validation.hstsPreloadNeedsIncludeSubdomains");
  }
  if (draft.hsts_preload && Number(draft.hsts_max_age_seconds || 0) < 31536000) {
    return ctx.t("sites.validation.hstsPreloadNeedsMaxAge");
  }
  for (const rule of deps.normalizeCustomLimitRules(draft.custom_limit_rules)) {
    if (!rule.path.startsWith("/")) {
      return ctx.t("sites.validation.customLimitPathFormat");
    }
    if (!/^\d+r\/s$/i.test(rule.rate)) {
      return ctx.t("sites.validation.customLimitRateFormat");
    }
  }
  if (deps.normalizeAntibotExclusionRules(draft.antibot_exclusion_rules).length > 32) {
    return ctx.t("sites.validation.antibotExclusionRulesLimit");
  }
  for (const rule of deps.normalizeAntibotExclusionRules(draft.antibot_exclusion_rules)) {
    if (!rule.path.startsWith("/")) {
      return ctx.t("sites.validation.antibotExclusionPathFormat");
    }
    const methods = Array.isArray(rule.methods) ? rule.methods : [];
    if (!methods.length) {
      return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
    }
    if (methods.includes("*") && methods.length !== 1) {
      return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
    }
    for (const method of methods) {
      if (!["*", "GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE", "PATCH"].includes(method)) {
        return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
      }
    }
  }
  if (deps.normalizeAntibotChallengeRules(draft.antibot_challenge_rules).length > 32) {
    return "Too many antibot challenge rules (max 32).";
  }
  for (const rule of deps.normalizeAntibotChallengeRules(draft.antibot_challenge_rules)) {
    if (!rule.path.startsWith("/")) {
      return "Antibot challenge rule path must start with /.";
    }
    if (!["cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].includes(rule.challenge)) {
      return "Antibot challenge rule mode is invalid.";
    }
  }
  if (draft.use_auth_basic) {
    const users = deps.normalizeAuthBasicUsers(draft.auth_basic_users);
    const enabledUsers = users.filter((item) => item.enabled);
    const authMode = deps.normalizeAuthMode(draft.auth_mode);
    const enabledTokens = deps.normalizeAuthServiceTokens(draft.auth_service_tokens).filter((item) => item.enabled && String(item.token || "").trim());
    if (authMode !== "service_token" && !enabledUsers.length) {
      return ctx.t("sites.validation.authBasicUserRequired");
    }
    if (authMode !== "basic" && !enabledTokens.length) {
      return ctx.t("sites.validation.authServiceTokenRequired");
    }
    if (authMode !== "service_token") {
      for (const user of enabledUsers) {
        if (!String(user.password || "").trim()) {
          return ctx.t("sites.validation.authBasicPasswordRequired");
        }
      }
    }
    for (const token of deps.normalizeAuthServiceTokens(draft.auth_service_tokens)) {
      if (token.enabled && !String(token.token || "").trim()) {
        return ctx.t("sites.validation.authServiceTokenSecretRequired");
      }
    }
  }
  if (deps.normalizeAuthExclusionRules(draft.auth_exclusion_rules).length > 32) {
    return ctx.t("sites.validation.authExclusionRulesLimit");
  }
  for (const rule of deps.normalizeAuthExclusionRules(draft.auth_exclusion_rules)) {
    if (!rule.path.startsWith("/")) {
      return ctx.t("sites.validation.authExclusionPathFormat");
    }
    const methods = Array.isArray(rule.methods) ? rule.methods : [];
    if (!methods.length) {
      return ctx.t("sites.validation.authExclusionMethodsInvalid");
    }
    if (methods.includes("*") && methods.length !== 1) {
      return ctx.t("sites.validation.authExclusionMethodsInvalid");
    }
    for (const method of methods) {
      if (!["*", "GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE", "PATCH"].includes(method)) {
        return ctx.t("sites.validation.authExclusionMethodsInvalid");
      }
    }
  }
  if (draft.use_modsecurity_custom_configuration && !String(draft.modsecurity_custom_path || "").trim()) {
    return ctx.t("sites.validation.modsecCustomPathRequired");
  }
  return "";
}
