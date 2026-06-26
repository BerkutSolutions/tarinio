export function renderDetailViewRuntimeTail(state, ctx, deps, draft, isNew) {
  const { escapeHtml, renderListEditor, getQuickListTemplates, normalizeStringArray, renderStatusCodesEditor, renderCustomLimitRulesEditor, renderAntibotExclusionRulesEditor, normalizeBanEscalationStages, formatBanDurationSeconds, renderAntibotChallengeRulesEditor, renderAuthSessionTtlOptions, renderAuthUsersEditor, renderCountryEditor, renderAuthExclusionRulesEditor, renderAuthServiceTokensEditor, renderAuthHelpModal, renderAntibotHelpModal, renderTrafficBadBehaviorHelpModal, renderTrafficLimitsHelpModal, renderTrafficDnsblHelpModal, renderHeadersChapterHelpModal, renderBlockingChapterHelpModal, renderAntibotChapterHelpModal, renderGeoChapterHelpModal, renderModsecChapterHelpModal, normalizeAuthMode } = deps;
  const authMode = normalizeAuthMode(draft.auth_mode);
  return `
              <section class="waf-stack waf-service-compact-section${state.activeTab === "headers" ? "" : " waf-hidden"}" data-tab-panel="headers">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.headers.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.headers.subtitle"))}</div>
                  </div>
                  <button class="waf-help-icon-btn" type="button" id="service-headers-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.headers.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.headers.open"))}">?</button>
                </div>
                ${renderHeadersChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-form-grid">
                  <div class="waf-field">
                    <label for="service-cookie-flags">${escapeHtml(ctx.t("sites.easy.headers.cookieFlags"))}</label>
                    <input id="service-cookie-flags" value="${escapeHtml(draft.cookie_flags)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-referrer-policy">${escapeHtml(ctx.t("sites.easy.headers.referrerPolicy"))}</label>
                    <input id="service-referrer-policy" value="${escapeHtml(draft.referrer_policy)}">
                  </div>
                  <label class="waf-checkbox">
                    <input id="service-hsts-enabled" type="checkbox"${draft.hsts_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.headers.hstsEnabled"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-hsts-max-age">${escapeHtml(ctx.t("sites.easy.headers.hstsMaxAge"))}</label>
                    <input id="service-hsts-max-age" type="number" min="0" value="${escapeHtml(String(draft.hsts_max_age_seconds || 0))}">
                  </div>
                  <label class="waf-checkbox">
                    <input id="service-hsts-include-subdomains" type="checkbox"${draft.hsts_include_subdomains ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.headers.hstsIncludeSubdomains"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-hsts-preload" type="checkbox"${draft.hsts_preload ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.headers.hstsPreload"))}</span>
                  </label>
                  <div class="waf-field full">
                    <label for="service-content-security-policy">${escapeHtml(ctx.t("sites.easy.headers.contentSecurityPolicy"))}</label>
                    <textarea id="service-content-security-policy" rows="3">${escapeHtml(draft.content_security_policy)}</textarea>
                  </div>
                  ${renderListEditor("permissions_policy", ctx.t("sites.easy.headers.permissionsPolicy"), draft.permissions_policy, "geolocation=()", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  ${renderListEditor("keep_upstream_headers", ctx.t("sites.easy.headers.keepUpstreamHeaders"), draft.keep_upstream_headers, "X-Forwarded-For", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <label class="waf-checkbox">
                    <input id="service-use-cors" type="checkbox"${draft.use_cors ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.headers.useCors"))}</span>
                  </label>
                  ${renderListEditor("cors_allowed_origins", ctx.t("sites.easy.headers.allowedOrigins"), draft.cors_allowed_origins, "https://app.example.com", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "traffic" ? "" : " waf-hidden"}" data-tab-panel="traffic">
                <div>
                  <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.traffic.title"))}</div>
                  <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.traffic.subtitle"))}</div>
                </div>
                <div class="waf-traffic-layout">
                  <div class="waf-stack">
                    <div class="waf-subcard waf-stack waf-antiddos-frame">
                    <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                      <div>
                        <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior"))}</div>
                        <div class="muted">${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior.subtitle"))}</div>
                      </div>
                      <button class="waf-help-icon-btn" type="button" id="service-traffic-badbehavior-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.badBehavior.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.badBehavior.help.open"))}">?</button>
                    </div>
                    <div class="waf-form-grid">
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-bad-behavior" type="checkbox"${draft.use_bad_behavior ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBadBehavior"))}</span>
                      </label>
                      ${renderStatusCodesEditor(draft.bad_behavior_status_codes, ctx)}
                      <div class="waf-field">
                        <label for="service-bad-behavior-ban-time">${escapeHtml(ctx.t("sites.easy.traffic.banDurationSeconds"))}</label>
                        <input id="service-bad-behavior-ban-time" type="number" min="0" value="${escapeHtml(String(draft.bad_behavior_ban_time_seconds))}">
                      </div>
                      <div class="waf-field">
                        <label for="service-bad-behavior-threshold">${escapeHtml(ctx.t("sites.easy.traffic.threshold"))}</label>
                        <input id="service-bad-behavior-threshold" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_threshold))}">
                      </div>
                      <div class="waf-field">
                        <label for="service-bad-behavior-count-time">${escapeHtml(ctx.t("sites.easy.traffic.periodSeconds"))}</label>
                        <input id="service-bad-behavior-count-time" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_count_time_seconds))}">
                      </div>
                    </div>
                    </div>
                    <div class="waf-subcard waf-stack waf-antiddos-frame">
                      <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                        <div>
                          <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.limits"))}</div>
                          <div class="muted">${escapeHtml(ctx.t("sites.easy.traffic.frame.limits.subtitle"))}</div>
                        </div>
                        <button class="waf-help-icon-btn" type="button" id="service-traffic-limits-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.limits.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.limits.help.open"))}">?</button>
                      </div>
                      <div class="waf-form-grid">
                        <label class="waf-checkbox waf-field full">
                          <input id="service-use-limit-conn" type="checkbox"${draft.use_limit_conn ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitConnections"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http1">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp1Connections"))}</label>
                          <input id="service-limit-conn-max-http1" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http1))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http2">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp2Streams"))}</label>
                          <input id="service-limit-conn-max-http2" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http2))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http3">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp3Streams"))}</label>
                          <input id="service-limit-conn-max-http3" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http3))}">
                        </div>
                        <label class="waf-checkbox waf-field full">
                          <input id="service-use-limit-req" type="checkbox"${draft.use_limit_req ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitRequests"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-limit-req-url">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestUrl"))}</label>
                          <input id="service-limit-req-url" value="${escapeHtml(draft.limit_req_url)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-req-rate">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestRate"))}</label>
                          <input id="service-limit-req-rate" value="${escapeHtml(draft.limit_req_rate)}">
                        </div>
                        ${renderCustomLimitRulesEditor(draft.custom_limit_rules, ctx)}
                      </div>
                    </div>
                  </div>
                  <div class="waf-subcard waf-stack waf-antiddos-frame">
                    <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                      <div>
                        <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.dnsbl"))}</div>
                        <div class="muted">${escapeHtml(ctx.t("sites.easy.traffic.frame.dnsbl.subtitle"))}</div>
                      </div>
                      <button class="waf-help-icon-btn" type="button" id="service-traffic-dnsbl-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.dnsbl.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.dnsbl.help.open"))}">?</button>
                    </div>
                    <div class="waf-note">${escapeHtml(ctx.t("sites.lists.note"))}</div>
                    <div class="waf-form-grid">
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-blacklist" type="checkbox"${draft.use_blacklist ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBlacklisting"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-dnsbl" type="checkbox"${draft.use_dnsbl ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateDnsbl"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-allowlist" type="checkbox"${draft.use_allowlist ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateAllowlist"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-exceptions" type="checkbox"${draft.use_exceptions ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateExceptions"))}</span>
                      </label>
                      ${renderListEditor("access_denylist", ctx.t("sites.lists.denylist"), draft.access_denylist, "203.0.113.10", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("access_allowlist", ctx.t("sites.lists.allowlist"), draft.access_allowlist, "10.0.0.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("exceptions_ip", ctx.t("sites.easy.traffic.exceptions"), draft.exceptions_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${draft.use_allowlist || normalizeStringArray(draft.access_allowlist).length
                        ? ""
                        : `<div class="waf-note waf-field full">${escapeHtml(ctx.t("sites.easy.traffic.allowlistDisabledHint"))}</div>`}
                      ${renderListEditor("blacklist_ip", ctx.t("sites.easy.traffic.blacklistIp"), draft.blacklist_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_rdns", ctx.t("sites.easy.traffic.blacklistRdns"), draft.blacklist_rdns, ".shodan.io", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_asn", ctx.t("sites.easy.traffic.blacklistAsn"), draft.blacklist_asn, "AS13335", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_user_agent", ctx.t("sites.easy.traffic.blacklistUserAgent"), draft.blacklist_user_agent, "curl/*", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), presets: getQuickListTemplates("blacklist_user_agent"), selectedTemplates: state.listTemplateSelection.blacklist_user_agent, ctx })}
                      ${renderListEditor("blacklist_uri", ctx.t("sites.easy.traffic.blacklistUri"), draft.blacklist_uri, "/admin", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), presets: getQuickListTemplates("blacklist_uri"), selectedTemplates: state.listTemplateSelection.blacklist_uri, ctx })}
                      ${renderListEditor("blacklist_ip_urls", ctx.t("sites.easy.traffic.blacklistIpUrls"), draft.blacklist_ip_urls, "https://example.com/ip.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_rdns_urls", ctx.t("sites.easy.traffic.blacklistRdnsUrls"), draft.blacklist_rdns_urls, "https://example.com/rdns.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_asn_urls", ctx.t("sites.easy.traffic.blacklistAsnUrls"), draft.blacklist_asn_urls, "https://example.com/asn.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_user_agent_urls", ctx.t("sites.easy.traffic.blacklistUserAgentUrls"), draft.blacklist_user_agent_urls, "https://example.com/ua.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_uri_urls", ctx.t("sites.easy.traffic.blacklistUriUrls"), draft.blacklist_uri_urls, "https://example.com/uri.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                    </div>
                  </div>
                </div>
                ${renderTrafficBadBehaviorHelpModal(ctx)}
                ${renderTrafficLimitsHelpModal(ctx)}
                ${renderTrafficDnsblHelpModal(ctx)}
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "blocking" ? "" : " waf-hidden"}" data-tab-panel="blocking">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.blocking.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.blocking.subtitle"))}</div>
                  </div>
                  <button class="waf-help-icon-btn" type="button" id="service-blocking-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.blocking.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.blocking.open"))}">?</button>
                </div>
                ${renderBlockingChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-note">${escapeHtml(ctx.t("sites.easy.blocking.baseHint"))}</div>
                <div class="waf-form-grid">
                  <label class="waf-checkbox waf-field full">
                    <input id="service-ban-escalation-enabled" type="checkbox"${draft.ban_escalation_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.blocking.enabled"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-ban-escalation-scope">${escapeHtml(ctx.t("sites.easy.blocking.scope"))}</label>
                    <select id="service-ban-escalation-scope">
                      <option value="all_sites"${draft.ban_escalation_scope === "all_sites" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.blocking.scope.allSites"))}</option>
                      <option value="current_site"${draft.ban_escalation_scope === "current_site" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.blocking.scope.currentSite"))}</option>
                    </select>
                  </div>
                  <div class="waf-field full">
                    <label for="service-ban-stage-input">${escapeHtml(ctx.t("sites.easy.blocking.stageInput"))}</label>
                    <div class="waf-inline">
                      <input id="service-ban-stage-input" placeholder="${escapeHtml(ctx.t("sites.easy.blocking.stagePlaceholder"))}">
                      <button class="btn ghost btn-sm" type="button" data-ban-stage-add>${escapeHtml(ctx.t("sites.easy.blocking.addStage"))}</button>
                    </div>
                    <div class="waf-note">${escapeHtml(ctx.t("sites.easy.blocking.help"))}</div>
                    <div class="waf-inline">
                      ${normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds).map((seconds, index) => `
                        <span class="badge badge-neutral">
                          ${escapeHtml(`${ctx.t("sites.easy.blocking.stage")} ${index + 1}: ${seconds === 0 ? ctx.t("sites.easy.blocking.permanent") : formatBanDurationSeconds(seconds)}`)}
                          <button
                            class="waf-list-remove"
                            type="button"
                            data-ban-stage-remove="${index}">x</button>
                        </span>
                      `).join("")}
                    </div>
                  </div>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "antibot" ? "" : " waf-hidden"}" data-tab-panel="antibot">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.antibot.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.antibot.subtitle"))}</div>
                  </div>
                  <button class="waf-help-icon-btn" type="button" id="service-antibot-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.antibot.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.antibot.open"))}">?</button>
                </div>
                ${renderAntibotChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-antibot-auth-grid">
                  <section class="waf-subcard waf-antibot-editor-frame">
                    <div class="waf-card-head">
                      <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                        <h3>${escapeHtml(ctx.t("sites.easy.antibot.frameTitle"))}</h3>
                        <button class="waf-help-icon-btn" type="button" id="service-antibot-help-btn" title="${escapeHtml(ctx.t("sites.easy.antibot.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.antibot.help.open"))}">?</button>
                      </div>
                    </div>
                    <div class="waf-card-body">
                      <div class="waf-form-grid">
                        <div class="waf-field">
                          <label for="service-antibot-challenge">${escapeHtml(ctx.t("sites.easy.antibot.challenge"))}</label>
                          <select id="service-antibot-challenge">
                            ${["no", "cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].map((mode) => `<option value="${mode}"${draft.antibot_challenge === mode ? " selected" : ""}>${mode}</option>`).join("")}
                          </select>
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-uri">${escapeHtml(ctx.t("sites.easy.antibot.url"))}</label>
                          <input id="service-antibot-uri" value="${escapeHtml(draft.antibot_uri)}">
                        </div>
                        <label class="waf-checkbox waf-field full">
                          <input id="service-antibot-scanner-auto-ban-enabled" type="checkbox"${draft.antibot_scanner_auto_ban_enabled ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.antibot.scannerAutoBanEnabled"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-antibot-recaptcha-score">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaScore"))}</label>
                          <input id="service-antibot-recaptcha-score" type="number" step="0.1" min="0" max="1" value="${escapeHtml(String(draft.antibot_recaptcha_score))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-recaptcha-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaSitekey"))}</label>
                          <input id="service-antibot-recaptcha-sitekey" value="${escapeHtml(draft.antibot_recaptcha_sitekey)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-recaptcha-secret">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaSecret"))}</label>
                          <input id="service-antibot-recaptcha-secret" type="password" value="${escapeHtml(draft.antibot_recaptcha_secret)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-hcaptcha-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.hcaptchaSitekey"))}</label>
                          <input id="service-antibot-hcaptcha-sitekey" value="${escapeHtml(draft.antibot_hcaptcha_sitekey)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-hcaptcha-secret">${escapeHtml(ctx.t("sites.easy.antibot.hcaptchaSecret"))}</label>
                          <input id="service-antibot-hcaptcha-secret" type="password" value="${escapeHtml(draft.antibot_hcaptcha_secret)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-turnstile-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.turnstileSitekey"))}</label>
                          <input id="service-antibot-turnstile-sitekey" value="${escapeHtml(draft.antibot_turnstile_sitekey)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-turnstile-secret">${escapeHtml(ctx.t("sites.easy.antibot.turnstileSecret"))}</label>
                          <input id="service-antibot-turnstile-secret" type="password" value="${escapeHtml(draft.antibot_turnstile_secret)}">
                        </div>
                        ${renderAntibotExclusionRulesEditor(draft.antibot_exclusion_rules, ctx)}
                        <label class="waf-checkbox waf-field full">
                          <input id="service-antibot-escalation-enabled" type="checkbox"${draft.challenge_escalation_enabled ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.antibot.twoLayerEscalation"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-antibot-escalation-mode">${escapeHtml(ctx.t("sites.easy.antibot.escalationMode"))}</label>
                          <select id="service-antibot-escalation-mode">
                            ${["cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].map((mode) => `<option value="${mode}"${draft.challenge_escalation_mode === mode ? " selected" : ""}>${mode}</option>`).join("")}
                          </select>
                        </div>
                        ${renderAntibotChallengeRulesEditor(draft.antibot_challenge_rules, ctx)}
                      </div>
                    </div>
                  </section>
                  <section class="waf-subcard waf-auth-editor-frame">
                    <div class="waf-card-head">
                      <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                        <h3>${escapeHtml(ctx.t("sites.easy.antibot.authSectionTitle"))}</h3>
                        <button class="waf-help-icon-btn" type="button" id="service-auth-help-btn" title="${escapeHtml(ctx.t("sites.easy.auth.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.auth.help.open"))}">?</button>
                      </div>
                    </div>
                    <div class="waf-card-body">
                      <div class="waf-form-grid">
                        <label class="waf-checkbox">
                          <input id="service-use-auth-basic" type="checkbox"${draft.use_auth_basic ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.antibot.useAuthBasic"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-auth-mode">${escapeHtml(ctx.t("sites.easy.auth.mode"))}</label>
                          <select id="service-auth-mode">
                            <option value="basic"${authMode === "basic" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.auth.mode.basic"))}</option>
                            <option value="service_token"${authMode === "service_token" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.auth.mode.serviceToken"))}</option>
                            <option value="basic_or_token"${authMode === "basic_or_token" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.auth.mode.basicOrToken"))}</option>
                          </select>
                        </div>
                        <div class="waf-field">
                          <label for="service-auth-order">${escapeHtml(ctx.t("sites.easy.auth.order"))}</label>
                          <select id="service-auth-order">
                            <option value="auth_first"${draft.auth_order === "auth_first" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.auth.order.authFirst"))}</option>
                            <option value="antibot_first"${draft.auth_order === "antibot_first" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.auth.order.antibotFirst"))}</option>
                          </select>
                        </div>
                        <div class="waf-field">
                          <label for="service-auth-basic-location">${escapeHtml(ctx.t("sites.easy.antibot.authBasicLocation"))}</label>
                          <input id="service-auth-basic-location" value="${escapeHtml("sitewide")}" readonly>
                        </div>
                        <div class="waf-field">
                          <label for="service-auth-basic-text">${escapeHtml(ctx.t("sites.easy.antibot.authText"))}</label>
                          <input id="service-auth-basic-text" value="${escapeHtml(draft.auth_basic_text)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-auth-basic-session-ttl">${escapeHtml(ctx.t("sites.easy.antibot.authSessionTtl"))}</label>
                          <select id="service-auth-basic-session-ttl">
                            ${renderAuthSessionTtlOptions(draft.auth_basic_session_inactivity_minutes, ctx)}
                          </select>
                        </div>
                        <div class="waf-field full">
                          <div class="waf-note">${escapeHtml(ctx.t("sites.easy.antibot.authSessionTtlHint"))}</div>
                        </div>
                        ${renderAuthExclusionRulesEditor(draft.auth_exclusion_rules, ctx)}
                        ${authMode === "service_token" ? "" : renderAuthUsersEditor(draft.auth_basic_users, ctx)}
                        ${authMode === "basic" ? "" : renderAuthServiceTokensEditor(draft.auth_service_tokens, ctx)}
                      </div>
                    </div>
                  </section>
                </div>
                ${renderAntibotHelpModal(ctx)}
                ${renderAuthHelpModal(ctx)}
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "geo" ? "" : " waf-hidden"}" data-tab-panel="geo">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.geo.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.geo.subtitle"))}</div>
                  </div>
                  <button class="waf-help-icon-btn" type="button" id="service-geo-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.geo.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.geo.open"))}">?</button>
                </div>
                ${renderGeoChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-form-grid">
                  ${renderCountryEditor("blacklist_country", ctx.t("sites.easy.geo.countryBlacklist"), draft.blacklist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.blacklist_country, ctx })}
                  ${renderCountryEditor("whitelist_country", ctx.t("sites.easy.geo.countryWhitelist"), draft.whitelist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.whitelist_country, ctx })}
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "modsec" ? "" : " waf-hidden"}" data-tab-panel="modsec">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.modsec.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.modsec.subtitle"))}</div>
                  </div>
                  <button class="waf-help-icon-btn" type="button" id="service-modsec-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.modsec.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.modsec.open"))}">?</button>
                </div>
                ${renderModsecChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-form-grid">
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity" type="checkbox"${draft.use_modsecurity ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useModsecurity"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity-crs-plugins" type="checkbox"${draft.use_modsecurity_crs_plugins ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useCrsPlugins"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity-custom-configuration" type="checkbox"${draft.use_modsecurity_custom_configuration ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useCustomConfiguration"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-modsecurity-crs-version">${escapeHtml(ctx.t("sites.easy.modsec.crsVersion"))}</label>
                    <input id="service-modsecurity-crs-version" value="${escapeHtml(draft.modsecurity_crs_version)}">
                  </div>
                  ${renderListEditor("modsecurity_crs_plugins", ctx.t("sites.easy.modsec.crsPlugins"), draft.modsecurity_crs_plugins, "plugin-id", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <div class="waf-field${draft.use_modsecurity_custom_configuration ? "" : " waf-hidden"}">
                    <label for="service-modsecurity-custom-path">${escapeHtml(ctx.t("sites.easy.modsec.customPath"))}</label>
                    <input id="service-modsecurity-custom-path" value="${escapeHtml(draft.modsecurity_custom_path)}">
                  </div>
                  <div class="waf-field full${draft.use_modsecurity_custom_configuration ? "" : " waf-hidden"}">
                    <label for="service-modsecurity-custom-content">${escapeHtml(ctx.t("sites.easy.modsec.customContent"))}</label>
                    <textarea id="service-modsecurity-custom-content" rows="6">${escapeHtml(draft.modsecurity_custom_content)}</textarea>
                  </div>
                </div>
              </section>

              <div class="waf-actions waf-actions-between">
                <button class="btn ghost btn-sm" type="button" id="service-back-bottom">${escapeHtml(ctx.t("common.back"))}</button>
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t(isNew ? "sites.action.createSite" : "sites.action.saveSite"))}</button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>
  `;
}
