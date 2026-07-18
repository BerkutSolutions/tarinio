export function buildDetailDraftFromForm(container, state, deps = {}) {
  const {
    computeUpstreamID,
    normalizeServiceProfile,
    normalizeEmail,
    normalizeStringArray,
    normalizeArray,
    normalizeBanEscalationStages,
    normalizeAntibotExclusionRules,
    normalizeAuthBasicUsers,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthServiceTokens,
    normalizeAuthSessionTTLMinutes,
    normalizeAPIPositiveEndpointPolicies,
    normalizeGeoTimeWindows,
    normalizeWSBlockPatterns,
    normalizeModSecurityExclusionRules,
  } = deps;
  return {
    id: container.querySelector("#service-id").value.trim().toLowerCase(),
    primary_host: container.querySelector("#service-host").value.trim().toLowerCase(),
    enabled: container.querySelector("#service-enabled").checked,
    adaptive_model_enabled: container.querySelector("#service-adaptive-model-enabled")?.checked || false,
    tls_enabled: container.querySelector("#service-tls-enabled").checked,
    tls_self_signed: container.querySelector("#service-tls-self-signed").checked,
    certificate_id: container.querySelector("#service-certificate-id").value.trim().toLowerCase(),
    security_mode: container.querySelector("#service-security-mode").value,
    service_profile: normalizeServiceProfile(container.querySelector("#service-profile")?.value || state.draft.service_profile),
    upstream_id: computeUpstreamID(container.querySelector("#service-id").value),
    upstream_host: container.querySelector("#service-upstream-host").value.trim(),
    upstream_port: Number(container.querySelector("#service-upstream-port").value || "80"),
    upstream_scheme: container.querySelector("#service-upstream-scheme").value,
    auto_lets_encrypt: container.querySelector("#service-auto-lets-encrypt").checked,
    use_lets_encrypt_staging: container.querySelector("#service-lets-encrypt-staging").checked,
    use_lets_encrypt_wildcard: container.querySelector("#service-lets-encrypt-wildcard").checked,
    certificate_authority_server: container.querySelector("#service-ca-server").value,
    acme_account_email: normalizeEmail(state.draft.acme_account_email),
    use_reverse_proxy: container.querySelector("#service-use-reverse-proxy").checked,
    reverse_proxy_host: container.querySelector("#service-reverse-proxy-host").value.trim(),
    reverse_proxy_url: container.querySelector("#service-reverse-proxy-url").value.trim(),
    reverse_proxy_custom_host: container.querySelector("#service-reverse-proxy-custom-host").value.trim(),
    reverse_proxy_ssl_sni: container.querySelector("#service-reverse-proxy-ssl-sni").checked,
    reverse_proxy_ssl_sni_name: container.querySelector("#service-reverse-proxy-ssl-sni-name").value.trim(),
    reverse_proxy_websocket: container.querySelector("#service-reverse-proxy-websocket").checked,
    reverse_proxy_keepalive: container.querySelector("#service-reverse-proxy-keepalive").checked,
    health_check_enabled: Boolean(container.querySelector("#service-health-check-enabled")?.checked),
    health_check_path: container.querySelector("#service-health-check-path")?.value.trim() || "/health",
    health_check_interval_seconds: Number(container.querySelector("#service-health-check-interval")?.value || 10),
    health_check_fail_threshold: Number(container.querySelector("#service-health-check-fail-threshold")?.value || 3),
    pass_host_header: container.querySelector("#service-pass-host-header")?.checked ?? true,
    send_x_forwarded_for: container.querySelector("#service-send-x-forwarded-for")?.checked ?? true,
    send_x_forwarded_proto: container.querySelector("#service-send-x-forwarded-proto")?.checked ?? true,
    send_x_real_ip: container.querySelector("#service-send-x-real-ip")?.checked ?? false,
    allowed_methods: normalizeStringArray(state.draft.allowed_methods),
    max_client_size: container.querySelector("#service-max-client-size").value.trim(),
    http2: container.querySelector("#service-http2").checked,
    http3: container.querySelector("#service-http3").checked,
    http_strict_parsing: Boolean(container.querySelector("#service-http-strict-parsing")?.checked),
    ssl_protocols: normalizeStringArray(state.draft.ssl_protocols),
    cookie_flags: container.querySelector("#service-cookie-flags").value.trim(),
    content_security_policy: container.querySelector("#service-content-security-policy").value.trim(),
    permissions_policy: normalizeStringArray(state.draft.permissions_policy),
    keep_upstream_headers: normalizeStringArray(state.draft.keep_upstream_headers),
    referrer_policy: container.querySelector("#service-referrer-policy").value.trim(),
    hsts_enabled: container.querySelector("#service-hsts-enabled")?.checked ?? true,
    hsts_max_age_seconds: Number(container.querySelector("#service-hsts-max-age")?.value || "15552000"),
    hsts_include_subdomains: container.querySelector("#service-hsts-include-subdomains")?.checked ?? false,
    hsts_preload: container.querySelector("#service-hsts-preload")?.checked ?? false,
    use_cors: container.querySelector("#service-use-cors").checked,
    cors_allowed_origins: normalizeStringArray(state.draft.cors_allowed_origins),
    use_allowlist: container.querySelector("#service-use-allowlist")?.checked || false,
    use_exceptions: container.querySelector("#service-use-exceptions")?.checked || false,
    access_allowlist: normalizeStringArray(state.draft.access_allowlist),
    exceptions_ip: normalizeStringArray(state.draft.exceptions_ip),
    exceptions_uri: normalizeStringArray(state.draft.exceptions_uri),
    access_denylist: normalizeStringArray(state.draft.access_denylist),
    use_bad_behavior: container.querySelector("#service-use-bad-behavior").checked,
    bad_behavior_status_codes: normalizeArray(state.draft.bad_behavior_status_codes).map((item) => Number(item)).filter((item) => Number.isInteger(item)),
    bad_behavior_ban_time_seconds: Number(container.querySelector("#service-bad-behavior-ban-time").value || "300"),
    bad_behavior_threshold: Number(container.querySelector("#service-bad-behavior-threshold").value || "20"),
    bad_behavior_count_time_seconds: Number(container.querySelector("#service-bad-behavior-count-time").value || "30"),
    ban_escalation_enabled: container.querySelector("#service-ban-escalation-enabled")?.checked || false,
    ban_escalation_scope: container.querySelector("#service-ban-escalation-scope")?.value || "all_sites",
    ban_escalation_stages_seconds: normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, Number(container.querySelector("#service-bad-behavior-ban-time").value || "300")),
    use_blacklist: container.querySelector("#service-use-blacklist").checked,
    use_dnsbl: container.querySelector("#service-use-dnsbl").checked,
    blacklist_ip: normalizeStringArray(state.draft.blacklist_ip),
    blacklist_rdns: normalizeStringArray(state.draft.blacklist_rdns),
    blacklist_asn: normalizeStringArray(state.draft.blacklist_asn),
    blacklist_user_agent: normalizeStringArray(state.draft.blacklist_user_agent),
    blacklist_uri: normalizeStringArray(state.draft.blacklist_uri),
    blacklist_ja3: normalizeStringArray(state.draft.blacklist_ja3),
    blacklist_ip_urls: normalizeStringArray(state.draft.blacklist_ip_urls),
    blacklist_rdns_urls: normalizeStringArray(state.draft.blacklist_rdns_urls),
    blacklist_asn_urls: normalizeStringArray(state.draft.blacklist_asn_urls),
    blacklist_user_agent_urls: normalizeStringArray(state.draft.blacklist_user_agent_urls),
    blacklist_uri_urls: normalizeStringArray(state.draft.blacklist_uri_urls),
    blacklist_ja3_urls: normalizeStringArray(state.draft.blacklist_ja3_urls),
    use_limit_conn: container.querySelector("#service-use-limit-conn").checked,
    limit_conn_max_http1: Number(container.querySelector("#service-limit-conn-max-http1").value || "200"),
    limit_conn_max_http2: Number(container.querySelector("#service-limit-conn-max-http2").value || "400"),
    limit_conn_max_http3: Number(container.querySelector("#service-limit-conn-max-http3").value || "400"),
    use_limit_req: container.querySelector("#service-use-limit-req").checked,
    limit_req_url: container.querySelector("#service-limit-req-url").value.trim(),
    limit_req_rate: (() => { const v = (container.querySelector("#service-limit-req-rate")?.value || "").trim(); const unit = container.querySelector("#service-limit-req-rate-unit")?.value || "r/s"; return v ? `${v}${unit}` : ""; })(),
    custom_limit_rules: Array.from(container.querySelectorAll("[data-custom-limit-path]"))
      .map((input) => {
        const index = String(input.dataset.customLimitPath || "");
        const rateInput = container.querySelector(`[data-custom-limit-rate="${index}"]`);
        const rateUnitInput = container.querySelector(`[data-custom-limit-rate-unit="${index}"]`);
        return {
          path: String(input.value || "").trim(),
          rate: (() => { const v = String(rateInput?.value || "").trim(); const unit = rateUnitInput?.value || "r/s"; return v ? `${v}${unit}` : ""; })()
        };
      }),
    antibot_challenge: container.querySelector("#service-antibot-enabled")?.checked ? (container.querySelector("#service-antibot-challenge").value || "cookie") : "no",
    antibot_challenge_template: container.querySelector("#service-antibot-challenge-template")?.value || "v2",
    antibot_uri: container.querySelector("#service-antibot-uri").value.trim(),
    antibot_scanner_auto_ban_enabled: container.querySelector("#service-antibot-scanner-auto-ban-enabled")?.checked ?? false,
    antibot_recaptcha_score: Number(container.querySelector("#service-antibot-recaptcha-score").value || "0.7"),
    antibot_recaptcha_sitekey: container.querySelector("#service-antibot-recaptcha-sitekey").value.trim(),
    antibot_recaptcha_secret: container.querySelector("#service-antibot-recaptcha-secret").value.trim(),
    antibot_hcaptcha_sitekey: container.querySelector("#service-antibot-hcaptcha-sitekey").value.trim(),
    antibot_hcaptcha_secret: container.querySelector("#service-antibot-hcaptcha-secret").value.trim(),
    antibot_turnstile_sitekey: container.querySelector("#service-antibot-turnstile-sitekey").value.trim(),
    antibot_turnstile_secret: container.querySelector("#service-antibot-turnstile-secret").value.trim(),
    antibot_exclusion_rules: normalizeAntibotExclusionRules(
      Array.from(container.querySelectorAll("[data-antibot-exclusion-path]"))
        .map((input) => {
          const index = String(input.dataset.antibotExclusionPath || "");
          const methodsInput = container.querySelector(`[data-antibot-exclusion-methods="${index}"]`);
          return {
            path: String(input.value || "").trim(),
            methods: String(methodsInput?.value || "")
              .split(/[\s,|]+/)
              .map((item) => item.trim())
              .filter(Boolean)
          };
        })
    ),
    challenge_escalation_enabled: container.querySelector("#service-antibot-escalation-enabled")?.checked || false,
    challenge_escalation_mode: container.querySelector("#service-antibot-escalation-mode")?.value || "javascript",
    antibot_challenge_rules: Array.from(container.querySelectorAll("[data-antibot-rule-path]"))
      .map((input) => {
        const index = String(input.dataset.antibotRulePath || "");
        const modeInput = container.querySelector(`[data-antibot-rule-challenge="${index}"]`);
        return {
          path: String(input.value || "").trim(),
          challenge: String(modeInput?.value || "").trim().toLowerCase()
        };
      }),
    use_auth_basic: container.querySelector("#service-use-auth-basic").checked,
    auth_mode: normalizeAuthMode(container.querySelector("#service-auth-mode")?.value || "basic"),
    auth_order: normalizeAuthOrder(container.querySelector("#service-auth-order")?.value || "auth_first"),
    auth_basic_location: container.querySelector("#service-auth-basic-location").value.trim(),
    auth_basic_user: "",
    auth_basic_password: "",
    auth_basic_text: container.querySelector("#service-auth-basic-text").value.trim(),
    auth_basic_template: container.querySelector("#service-auth-basic-template")?.value || "v1",
    auth_basic_users: Array.from(container.querySelectorAll("[data-auth-user-username]"))
      .map((input) => {
        const index = String(input.dataset.authUserUsername || "");
        const passwordInput = container.querySelector(`[data-auth-user-password="${index}"]`);
        const enabledInput = container.querySelector(`[data-auth-user-enabled="${index}"]`);
        const currentUser = normalizeAuthBasicUsers(state.draft.auth_basic_users)[Number.parseInt(index, 10)] || {};
        return {
          username: String(input.value || "").trim(),
          password: String(passwordInput?.value || "").trim() || (passwordInput?.dataset.authUserPasswordStored === "true" ? "********" : ""),
          password_length: Number(currentUser.password_length || 0),
          enabled: Boolean(enabledInput?.checked),
          last_login_at: String(currentUser.last_login_at || "")
        };
      }),
    auth_exclusion_rules: Array.from(container.querySelectorAll("[data-auth-exclusion-path]")).map((input) => {
      const index = String(input.dataset.authExclusionPath || "");
      const methodsInput = container.querySelector(`[data-auth-exclusion-methods="${index}"]`);
      return {
        path: String(input.value || "").trim(),
        methods: String(methodsInput?.value || "").split(/[\s,|]+/).map((item) => item.trim()).filter(Boolean)
      };
    }),
    auth_service_tokens: normalizeAuthServiceTokens(
      Array.from(container.querySelectorAll("[data-auth-token-service-name]")).map((input) => {
        const index = String(input.dataset.authTokenServiceName || "");
        const tokenInput = container.querySelector(`[data-auth-token-secret="${index}"]`);
        const enabledInput = container.querySelector(`[data-auth-token-enabled="${index}"]`);
        const lastUsed = normalizeAuthServiceTokens(state.draft.auth_service_tokens)[Number.parseInt(index, 10)]?.last_used_at || "";
        return {
          service_name: String(input.value || "").trim(),
          token: String(tokenInput?.value || "").trim(),
          enabled: Boolean(enabledInput?.checked),
          last_used_at: String(lastUsed || "")
        };
      })
    ),
    auth_basic_session_inactivity_minutes: normalizeAuthSessionTTLMinutes(
      container.querySelector("#service-auth-basic-session-ttl")?.value
    ),
    blacklist_country: normalizeStringArray(state.draft.blacklist_country),
    whitelist_country: normalizeStringArray(state.draft.whitelist_country),
    show_geo_block_page: Boolean(container.querySelector("#service-show-geo-block-page")?.checked),
    use_custom_error_pages: Boolean(container.querySelector("#service-use-custom-error-pages")?.checked ?? true),
    disabled_error_pages: (() => {
      const cbs = container.querySelectorAll(".waf-ep-page-cb");
      const disabled = [];
      cbs.forEach((cb) => { if (!cb.checked) disabled.push(cb.dataset.epSlug); });
      return disabled;
    })(),
    geo_time_windows: normalizeGeoTimeWindows(state.draft.geo_time_windows),
    api_positive_security_enabled: Boolean(state.draft.api_positive_security_enabled),
    api_positive_openapi_schema_ref: String(state.draft.api_positive_openapi_schema_ref || "").trim(),
    api_positive_enforcement_mode: String(state.draft.api_positive_enforcement_mode || "monitor").trim().toLowerCase() || "monitor",
    api_positive_default_action: String(state.draft.api_positive_default_action || "allow").trim().toLowerCase() || "allow",
    api_positive_endpoint_policies: normalizeAPIPositiveEndpointPolicies(state.draft.api_positive_endpoint_policies),
    use_modsecurity: container.querySelector("#service-use-modsecurity").checked,
    use_modsecurity_crs_plugins: container.querySelector("#service-use-modsecurity-crs-plugins").checked,
    use_modsecurity_custom_configuration: container.querySelector("#service-use-modsecurity-custom-configuration").checked,
    modsecurity_crs_version: container.querySelector("#service-modsecurity-crs-version").value.trim(),
    modsecurity_crs_plugins: normalizeStringArray(state.draft.modsecurity_crs_plugins),
    modsecurity_exclusion_rules: normalizeModSecurityExclusionRules(state.draft.modsecurity_exclusion_rules),
    modsecurity_custom_path: container.querySelector("#service-modsecurity-custom-path").value.trim(),
    modsecurity_custom_content: container.querySelector("#service-modsecurity-custom-content").value,
    use_ws_inspection: Boolean(container.querySelector("#service-use-ws-inspection")?.checked),
    ws_block_patterns: normalizeWSBlockPatterns(state.draft.ws_block_patterns),
    ws_max_message_bytes: Math.max(0, Number.parseInt(container.querySelector("#service-ws-max-message-bytes")?.value || "0", 10) || 0),
    ws_rate_msg_per_sec: Math.max(0, Number.parseInt(container.querySelector("#service-ws-rate-msg-per-sec")?.value || "0", 10) || 0),
    mtls_enabled: Boolean(container.querySelector("#service-mtls-enabled")?.checked),
    mtls_optional: Boolean(container.querySelector("#service-mtls-optional")?.checked),
    mtls_verify_depth: Math.max(0, Number.parseInt(container.querySelector("#service-mtls-verify-depth")?.value || "2", 10) || 2),
    mtls_client_ca_ref: String(container.querySelector("#service-mtls-client-ca-ref")?.value || "").trim(),
    mtls_pass_headers: Boolean(container.querySelector("#service-mtls-pass-headers")?.checked),
    upstream_mtls_enabled: Boolean(container.querySelector("#service-upstream-mtls-enabled")?.checked),
    upstream_mtls_cert_ref: String(container.querySelector("#service-upstream-mtls-cert-ref")?.value || "").trim(),
    upstream_mtls_key_ref: String(container.querySelector("#service-upstream-mtls-key-ref")?.value || "").trim(),
    upstream_mtls_ca_ref: String(container.querySelector("#service-upstream-mtls-ca-ref")?.value || "").trim()
  };
}
