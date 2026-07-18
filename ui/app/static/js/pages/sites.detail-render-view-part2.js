export function renderDetailViewRuntimeTail(state, ctx, deps, draft, isNew) {
  const { escapeHtml, renderListEditor, getQuickListTemplates, normalizeStringArray, renderStatusCodesEditor, renderCustomLimitRulesEditor, renderAntibotExclusionRulesEditor, renderModSecurityExclusionRulesEditor, normalizeBanEscalationStages, formatBanDurationSeconds, renderAntibotChallengeRulesEditor, renderAuthSessionTtlOptions, renderAuthUsersEditor, renderCountryEditor, renderAuthExclusionRulesEditor, renderAuthServiceTokensEditor, renderAuthHelpModal, renderAntibotHelpModal, renderTrafficBadBehaviorHelpModal, renderTrafficBlacklistHelpModal, renderTrafficAllowlistHelpModal, renderTrafficLimitsHelpModal, renderTrafficDnsblHelpModal, renderHeadersChapterHelpModal, renderBlockingChapterHelpModal, renderAntibotChapterHelpModal, renderGeoChapterHelpModal, renderModsecChapterHelpModal, renderGeoTimeWindowsEditor, normalizeAuthMode, renderVirtualPatchesEditor, renderWebSocketChapterHelpModal, renderVirtualPatchesChapterHelpModal, renderErrorPagesTab } = deps;
  const authMode = normalizeAuthMode(draft.auth_mode);
  return `

              <section class="waf-stack waf-service-compact-section${state.activeTab === "traffic" ? "" : " waf-hidden"}" data-tab-panel="traffic">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.traffic.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.traffic.subtitle"))}</div>
                  </div>
                </div>
                <div class="waf-antibot-auth-grid">
                  <div class="waf-stack">
                    <section class="waf-subcard waf-stack">
                      <div class="waf-card-head">
                        <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                          <div>
                            <h3>${escapeHtml(ctx.t("sites.easy.traffic.frame.allowlists"))}</h3>
                            <div class="muted" style="font-size:12px;margin-top:2px;">${escapeHtml(ctx.t("sites.easy.traffic.frame.allowlists.subtitle"))}</div>
                          </div>
                          <button class="waf-help-icon-btn" type="button" id="service-traffic-allowlist-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.allowlist.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.allowlist.help.open"))}">?</button>
                        </div>
                      </div>
                      ${renderTrafficAllowlistHelpModal(ctx, escapeHtml)}
                      <div class="waf-card-body">
                        <div class="waf-form-grid">
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-allowlist" type="checkbox"${draft.use_allowlist ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateAllowlist"))}</span>
                          </label>
                          ${renderListEditor("access_allowlist", ctx.t("sites.lists.allowlist"), draft.access_allowlist, "10.0.0.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), inputFilter: "cidr", disabled: !draft.use_allowlist })}
                          ${renderListEditor("exceptions_ip", ctx.t("sites.easy.traffic.exceptions"), draft.exceptions_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), inputFilter: "cidr", disabled: !draft.use_allowlist })}
                          ${renderListEditor("exceptions_uri", ctx.t("sites.easy.traffic.exceptionsUri"), draft.exceptions_uri, "/healthz", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), inputFilter: "uri", disabled: !draft.use_allowlist })}
                        </div>
                      </div>
                    </section>
                    <section class="waf-subcard waf-stack" style="margin-top:12px;">
                      <div class="waf-card-head">
                        <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                          <div>
                            <h3>${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior"))}</h3>
                            <div class="muted" style="font-size:12px;margin-top:2px;">${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior.subtitle"))}</div>
                          </div>
                          <button class="waf-help-icon-btn" type="button" id="service-traffic-badbehavior-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.badBehavior.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.badBehavior.help.open"))}">?</button>
                        </div>
                      </div>
                      <div class="waf-card-body">
                        <div class="waf-form-grid">
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-bad-behavior" type="checkbox"${draft.use_bad_behavior ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBadBehavior"))}</span>
                          </label>
                          ${renderStatusCodesEditor(draft.bad_behavior_status_codes, ctx, !draft.use_bad_behavior)}
                          <div class="waf-field">
                            <label for="service-bad-behavior-ban-time">${escapeHtml(ctx.t("sites.easy.traffic.banDurationSeconds"))}</label>
                            <input id="service-bad-behavior-ban-time" type="number" min="0" value="${escapeHtml(String(draft.bad_behavior_ban_time_seconds))}"${!draft.use_bad_behavior ? " disabled" : ""}>
                          </div>
                          <div class="waf-field">
                            <label for="service-bad-behavior-threshold">${escapeHtml(ctx.t("sites.easy.traffic.threshold"))}</label>
                            <input id="service-bad-behavior-threshold" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_threshold))}"${!draft.use_bad_behavior ? " disabled" : ""}>
                          </div>
                          <div class="waf-field">
                            <label for="service-bad-behavior-count-time">${escapeHtml(ctx.t("sites.easy.traffic.periodSeconds"))}</label>
                            <input id="service-bad-behavior-count-time" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_count_time_seconds))}"${!draft.use_bad_behavior ? " disabled" : ""}>
                          </div>
                        </div>
                      </div>
                    </section>
                    <section class="waf-subcard waf-stack" style="margin-top:12px;">
                      <div class="waf-card-head">
                        <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                          <div>
                            <h3>${escapeHtml(ctx.t("sites.easy.traffic.frame.limits"))}</h3>
                            <div class="muted" style="font-size:12px;margin-top:2px;">${escapeHtml(ctx.t("sites.easy.traffic.frame.limits.subtitle"))}</div>
                          </div>
                          <button class="waf-help-icon-btn" type="button" id="service-traffic-limits-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.limits.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.limits.help.open"))}">?</button>
                        </div>
                      </div>
                      <div class="waf-card-body">
                        <div class="waf-form-grid">
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-limit-conn" type="checkbox"${draft.use_limit_conn ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitConnections"))}</span>
                          </label>
                          <div class="waf-field">
                            <label for="service-limit-conn-max-http1">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp1Connections"))}</label>
                            <input id="service-limit-conn-max-http1" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http1))}"${!draft.use_limit_conn ? " disabled" : ""}>
                          </div>
                          <div class="waf-field">
                            <label for="service-limit-conn-max-http2">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp2Streams"))}</label>
                            <input id="service-limit-conn-max-http2" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http2))}"${!draft.use_limit_conn ? " disabled" : ""}>
                          </div>
                          <div class="waf-field">
                            <label for="service-limit-conn-max-http3">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp3Streams"))}</label>
                            <input id="service-limit-conn-max-http3" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http3))}"${!draft.use_limit_conn ? " disabled" : ""}>
                          </div>
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-limit-req" type="checkbox"${draft.use_limit_req ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitRequests"))}</span>
                          </label>
                          <div class="waf-field">
                            <label for="service-limit-req-url">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestUrl"))}</label>
                            <input id="service-limit-req-url" value="${escapeHtml(draft.limit_req_url)}"${!draft.use_limit_req ? " disabled" : ""}>
                          </div>
                          <div class="waf-field">
                            <label for="service-limit-req-rate">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestRate"))}</label>
                            <div class="waf-rate-wrap">
                              <input id="service-limit-req-rate" type="number" min="1" inputmode="numeric" value="${escapeHtml(String(draft.limit_req_rate || "").replace(/r\/s$|r\/m$/i, "").trim())}"${!draft.use_limit_req ? " disabled" : ""}>
                              <select id="service-limit-req-rate-unit"${!draft.use_limit_req ? " disabled" : ""}>
                                <option value="r/s"${!String(draft.limit_req_rate || "").endsWith("r/m") ? " selected" : ""}>r/s</option>
                                <option value="r/m"${String(draft.limit_req_rate || "").endsWith("r/m") ? " selected" : ""}>r/m</option>
                              </select>
                            </div>
                          </div>
                          ${renderCustomLimitRulesEditor(draft.custom_limit_rules, ctx, { escapeHtml }, !draft.use_limit_req)}
                        </div>
                      </div>
                    </section>
                  </div>
                  <div class="waf-stack">
                    <section class="waf-subcard waf-stack">
                      <div class="waf-card-head">
                        <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                          <div>
                            <h3>${escapeHtml(ctx.t("sites.easy.traffic.frame.blacklists"))}</h3>
                            <div class="muted" style="font-size:12px;margin-top:2px;">${escapeHtml(ctx.t("sites.easy.traffic.frame.blacklists.subtitle"))}</div>
                          </div>
                          <button class="waf-help-icon-btn" type="button" id="service-traffic-blacklist-help-btn" title="${escapeHtml(ctx.t("sites.easy.traffic.blacklist.help.open"))}" aria-label="${escapeHtml(ctx.t("sites.easy.traffic.blacklist.help.open"))}">?</button>
                        </div>
                      </div>
                      <div class="waf-card-body">
                        <div class="waf-form-grid">
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-blacklist" type="checkbox"${draft.use_blacklist ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBlacklisting"))}</span>
                          </label>
                          ${renderListEditor("blacklist_ip", ctx.t("sites.easy.traffic.blacklistIp"), draft.blacklist_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), inputFilter: "cidr", disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_rdns", ctx.t("sites.easy.traffic.blacklistRdns"), draft.blacklist_rdns, ".shodan.io", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_asn", ctx.t("sites.easy.traffic.blacklistAsn"), draft.blacklist_asn, "AS13335", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_user_agent", ctx.t("sites.easy.traffic.blacklistUserAgent"), draft.blacklist_user_agent, "curl/*", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist, presets: getQuickListTemplates("blacklist_user_agent"), selectedTemplates: state.listTemplateSelection.blacklist_user_agent, ctx })}
                          ${renderListEditor("blacklist_uri", ctx.t("sites.easy.traffic.blacklistUri"), draft.blacklist_uri, "/admin", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist, presets: getQuickListTemplates("blacklist_uri"), selectedTemplates: state.listTemplateSelection.blacklist_uri, ctx })}
                          ${/* TODO(ja3): blacklist_ja3 field removed — nginx 1.22.1+debian modsecurity incompatible with JA3 module. Re-enable when JA3/JA4 support is added. Field kept in data model. */""}
                          ${renderListEditor("blacklist_ip_urls", ctx.t("sites.easy.traffic.blacklistIpUrls"), draft.blacklist_ip_urls, "https://example.com/ip.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_rdns_urls", ctx.t("sites.easy.traffic.blacklistRdnsUrls"), draft.blacklist_rdns_urls, "https://example.com/rdns.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_asn_urls", ctx.t("sites.easy.traffic.blacklistAsnUrls"), draft.blacklist_asn_urls, "https://example.com/asn.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_user_agent_urls", ctx.t("sites.easy.traffic.blacklistUserAgentUrls"), draft.blacklist_user_agent_urls, "https://example.com/ua.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${renderListEditor("blacklist_uri_urls", ctx.t("sites.easy.traffic.blacklistUriUrls"), draft.blacklist_uri_urls, "https://example.com/uri.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), disabled: !draft.use_blacklist })}
                          ${/* TODO(ja3): blacklist_ja3_urls removed — same reason as blacklist_ja3. */""}
                          <label class="waf-checkbox waf-field full">
                            <input id="service-use-dnsbl" type="checkbox"${draft.use_dnsbl ? " checked" : ""}>
                            <span>${escapeHtml(ctx.t("sites.easy.traffic.activateDnsbl"))}</span>
                          </label>
                          ${renderListEditor("access_denylist", ctx.t("sites.lists.denylist"), draft.access_denylist, "203.0.113.10", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), inputFilter: "cidr", disabled: !draft.use_dnsbl })}
                        </div>
                      </div>
                    </section>
                  </div>
                </div>
                ${renderTrafficBadBehaviorHelpModal(ctx)}
                ${renderTrafficLimitsHelpModal(ctx)}
                ${renderTrafficBlacklistHelpModal(ctx)}
                ${renderTrafficAllowlistHelpModal(ctx, escapeHtml)}
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
                      <label class="waf-checkbox waf-field full" style="margin-bottom:12px;">
                        <input id="service-antibot-enabled" type="checkbox"${draft.antibot_challenge !== "no" ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.antibot.enabled"))}</span>
                      </label>
                      <div class="waf-field full${draft.antibot_challenge === "no" ? " waf-disabled" : ""}" id="antibot-template-row" style="display:flex;align-items:flex-end;gap:8px;margin-bottom:20px;">
                        <div style="flex:1;min-width:0;">
                          <label for="service-antibot-challenge-template">${escapeHtml(ctx.t("sites.easy.antibot.challengeTemplate"))}</label>
                          <select id="service-antibot-challenge-template"${draft.antibot_challenge === "no" ? " disabled" : ""}>
                            ${[
                              {val:"v1", key:"sites.easy.antibot.template.v1"},
                              {val:"v2", key:"sites.easy.antibot.template.v2"},
                              {val:"v3", key:"sites.easy.antibot.template.v3"},
                              {val:"v4", key:"sites.easy.antibot.template.v4"},
                              {val:"v5", key:"sites.easy.antibot.template.v5"},
                            ].map((t) => `<option value="${t.val}"${(draft.antibot_challenge_template || "v1") === t.val ? " selected" : ""}>${escapeHtml(ctx.t(t.key))}</option>`).join("")}
                          </select>
                        </div>
                        <button type="button" class="btn ghost btn-sm" id="antibot-template-preview-btn" style="align-self:flex-end;margin-bottom:3px;white-space:nowrap;flex-shrink:0;"${draft.antibot_challenge === "no" ? " disabled" : ""}>${escapeHtml(ctx.t("sites.easy.antibot.previewTemplate"))}</button>
                      </div>
                      <div id="antibot-body-wrap"${draft.antibot_challenge === "no" ? ' class="waf-disabled"' : ""}>
                        <div class="waf-form-grid">
                        <div class="waf-field">
                          <label for="service-antibot-challenge">${escapeHtml(ctx.t("sites.easy.antibot.challenge"))}</label>
                          <select id="service-antibot-challenge"${draft.antibot_challenge === "no" ? " disabled" : ""}>
                            ${["cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].map((mode) => `<option value="${mode}"${draft.antibot_challenge === mode ? " selected" : ""}>${mode}</option>`).join("")}
                          </select>
                        </div>
                        <div class="waf-field">
                          <label for="service-antibot-uri">${escapeHtml(ctx.t("sites.easy.antibot.url"))}</label>
                          <input id="service-antibot-uri" value="${escapeHtml(draft.antibot_uri)}"${draft.antibot_challenge === "no" ? " disabled" : ""}>
                        </div>
                        <label class="waf-checkbox waf-field full">
                          <input id="service-antibot-scanner-auto-ban-enabled" type="checkbox"${draft.antibot_scanner_auto_ban_enabled ? " checked" : ""}${draft.antibot_challenge === "no" ? " disabled" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.antibot.scannerAutoBanEnabled"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-antibot-recaptcha-score">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaScore"))}</label>
                          <input id="service-antibot-recaptcha-score" type="number" step="0.1" min="0" max="1" value="${escapeHtml(String(draft.antibot_recaptcha_score))}"${draft.antibot_challenge === "no" ? " disabled" : ""}>
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
                        <div class="waf-field full" style="display:flex;align-items:flex-end;gap:8px;">
                          <div style="flex:1;min-width:0;">
                            <label for="service-auth-basic-template">${escapeHtml(ctx.t("sites.easy.auth.template"))}</label>
                            <select id="service-auth-basic-template">
                              ${[1,2,3,4,5,6,7,8,9].map((number) => `<option value="v${number}"${(draft.auth_basic_template || "v1") === `v${number}` ? " selected" : ""}>${escapeHtml(ctx.t(`sites.easy.auth.template.v${number}`))}</option>`).join("")}
                            </select>
                          </div>
                          <button type="button" class="btn ghost btn-sm" id="auth-basic-template-preview-btn" style="align-self:flex-end;margin-bottom:3px;white-space:nowrap;flex-shrink:0;">${escapeHtml(ctx.t("sites.easy.auth.previewTemplate"))}</button>
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
                  <label class="waf-checkbox full">
                    <input id="service-show-geo-block-page" type="checkbox"${draft.show_geo_block_page ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.geo.showGeoBlockPage"))}</span>
                  </label>
                  ${renderCountryEditor("blacklist_country", ctx.t("sites.easy.geo.countryBlacklist"), draft.blacklist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.blacklist_country, ctx })}
                  ${renderCountryEditor("whitelist_country", ctx.t("sites.easy.geo.countryWhitelist"), draft.whitelist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.whitelist_country, ctx })}
                  ${renderGeoTimeWindowsEditor(draft.geo_time_windows, state.geoCatalog, ctx)}
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
                  ${renderModSecurityExclusionRulesEditor(draft.modsecurity_exclusion_rules, ctx)}
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "websocket" ? "" : " waf-hidden"}" data-tab-panel="websocket">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.websocket.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.websocket.subtitle"))}</div>
                  </div>
                </div>
                ${renderWebSocketChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-antibot-auth-grid">
                  <section class="waf-subcard">
                    <div class="waf-card-head">
                      <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                        <h3>${escapeHtml(ctx.t("sites.easy.ws.frameTitle"))}</h3>
                        <button class="waf-help-icon-btn" type="button" id="service-websocket-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.websocket.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.websocket.open"))}">?</button>
                      </div>
                    </div>
                    <div class="waf-card-body">
                      <div class="waf-form-grid">
                        <label class="waf-checkbox full">
                          <input id="service-use-ws-inspection" type="checkbox"${draft.use_ws_inspection ? " checked" : ""}${!draft.reverse_proxy_websocket ? " disabled" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.ws.useInspection"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-ws-max-message-bytes">${escapeHtml(ctx.t("sites.easy.ws.maxMessageBytes"))}</label>
                          <input id="service-ws-max-message-bytes" type="number" min="0" value="${escapeHtml(String(draft.ws_max_message_bytes || 0))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-ws-rate-msg-per-sec">${escapeHtml(ctx.t("sites.easy.ws.rateMsgPerSec"))}</label>
                          <input id="service-ws-rate-msg-per-sec" type="number" min="0" value="${escapeHtml(String(draft.ws_rate_msg_per_sec || 0))}">
                        </div>
                        ${renderListEditor("ws_block_patterns", ctx.t("sites.easy.ws.blockPatterns"), draft.ws_block_patterns, ctx.t("sites.easy.ws.blockPatternPlaceholder"), { full: true, emptyLabel: ctx.t("sites.easy.noValues") })}
                      </div>
                    </div>
                  </section>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "virtualpatches" ? "" : " waf-hidden"}" data-tab-panel="virtualpatches">
                <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                  <div>
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.virtualpatches.title"))}</div>
                    <div class="muted">${escapeHtml(ctx.t("sites.easy.tab.virtualpatches.subtitle"))}</div>
                  </div>
                </div>
                ${renderVirtualPatchesChapterHelpModal(ctx, escapeHtml)}
                <div class="waf-antibot-auth-grid">
                  <section class="waf-subcard">
                    <div class="waf-card-head">
                      <div class="waf-inline" style="justify-content:space-between;align-items:center;width:100%;">
                        <h3>${escapeHtml(ctx.t("sites.easy.virtualpatches.frameTitle"))}</h3>
                        <button class="waf-help-icon-btn" type="button" id="service-virtualpatches-chapter-help-btn" title="${escapeHtml(ctx.t("sites.help.virtualpatches.open"))}" aria-label="${escapeHtml(ctx.t("sites.help.virtualpatches.open"))}">?</button>
                      </div>
                    </div>
                    <div class="waf-card-body">
                      <p class="muted" style="margin:0 0 12px;">${escapeHtml(ctx.t("sites.easy.virtualpatches.hint"))}</p>
                      ${renderVirtualPatchesEditor(state, ctx, escapeHtml)}
                    </div>
                  </section>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "errorpages" ? "" : " waf-hidden"}" data-tab-panel="errorpages">
                ${renderErrorPagesTab(draft, ctx)}
              </section>

              <div class="waf-actions waf-actions-between">
                <button class="btn ghost btn-sm" type="button" id="service-back-bottom">${escapeHtml(ctx.t("common.back"))}</button>
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t(isNew ? "sites.action.createSite" : "sites.action.saveSite"))}</button>
              </div>
          </div>
        </section>
      </div>
    </div>
  `;
}
