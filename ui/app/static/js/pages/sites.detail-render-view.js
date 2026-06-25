import { renderDetailViewRuntimeTail } from "./sites.detail-render-view-part2.js";

export function renderDetailViewRuntime(state, ctx, deps) {
  const { SETTINGS_SEARCH_INDEX, escapeHtml, renderModeTabs, renderRawEditor, renderWizardNav, normalizeServiceProfile, renderListEditor, getQuickListTemplates, normalizeStringArray, renderStatusCodesEditor, renderCustomLimitRulesEditor, renderAntibotExclusionRulesEditor, normalizeBanEscalationStages, formatBanDurationSeconds, renderAntibotChallengeRulesEditor, renderAuthSessionTtlOptions, renderAuthUsersEditor, renderCountryEditor } = deps;
  const draft = state.draft;
  const isNew = state.route.mode === "create";
  const titleKey = isNew ? "sites.editor.newTitle" : "sites.editor.editTitle";
  const subtitleKey = isNew ? "sites.editor.newSubtitle" : "sites.editor.editSubtitle";
  const searchQuery = state.settingsSearch.trim().toLowerCase();
  const searchMatches = searchQuery
    ? SETTINGS_SEARCH_INDEX.filter((item) => {
      const label = String(ctx.t(item.labelKey) || "").toLowerCase();
      const id = String(item.id || "").toLowerCase();
      const selector = String(item.selector || "").toLowerCase();
      return label.includes(searchQuery) || id.includes(searchQuery) || selector.includes(searchQuery);
    }).slice(0, 10)
    : [];
  return `
    <div class="waf-page-stack">
      <section class="waf-card waf-service-shell-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t(titleKey))}</h3>
            <div class="muted">${escapeHtml(ctx.t(subtitleKey, { site: draft.primary_host || draft.id || ctx.t("sites.editor.newSiteLabel") }))}</div>
          </div>
          <div class="waf-actions">
            <button class="btn ghost btn-sm" type="button" id="service-back">${escapeHtml(ctx.t("common.back"))}</button>
            ${!isNew ? `<button class="btn ghost btn-sm" type="button" id="service-delete">${escapeHtml(ctx.t("common.delete"))}</button>` : ""}
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          ${renderModeTabs(state.editorMode, ctx)}
          ${state.editorMode === "raw" ? "" : `
            <div class="waf-field waf-service-settings-search">
              <label for="service-settings-search">${escapeHtml(ctx.t("sites.search.title"))}</label>
              <input id="service-settings-search" value="${escapeHtml(state.settingsSearch)}" placeholder="${escapeHtml(ctx.t("sites.search.placeholder"))}">
              ${searchQuery ? `
                <div class="waf-service-settings-search-dropdown">
                  ${searchMatches.length ? searchMatches.map((item) => `
                    <button type="button" class="waf-service-settings-search-item" data-settings-result="${escapeHtml(item.id)}" data-settings-tab="${escapeHtml(item.tab)}" data-settings-selector="${escapeHtml(item.selector)}">
                      ${escapeHtml(ctx.t(item.labelKey))}
                    </button>
                  `).join("") : `<div class="waf-note">${escapeHtml(ctx.t("sites.search.empty"))}</div>`}
                </div>
              ` : ""}
            </div>
          `}
        </div>
      </section>
      ${state.editorMode === "raw" ? renderRawEditor(state, ctx, isNew) : `
      <div class="waf-service-editor-layout">
        ${renderWizardNav(state.activeTab, ctx)}
        <section class="waf-card waf-service-editor-card">
          <div class="waf-card-body waf-stack">
            <div id="sites-feedback"></div>
            <form id="service-editor-form" class="waf-form waf-stack">
              <section class="waf-stack waf-service-compact-section${state.activeTab === "front" ? "" : " waf-hidden"}" data-tab-panel="front">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.wizard.front.title"))}</div>
                <div class="waf-form-grid">
                  <div class="waf-field">
                    <label for="service-host">${escapeHtml(ctx.t("sites.easy.front.serverName"))}</label>
                    <input id="service-host" value="${escapeHtml(draft.primary_host)}" placeholder="${escapeHtml(ctx.t("sites.editor.hostPlaceholder"))}">
                  </div>
                  <div class="waf-field">
                    <label for="service-id">${escapeHtml(ctx.t("sites.easy.front.serviceId"))}</label>
                    <input id="service-id" value="${escapeHtml(draft.id)}" placeholder="${escapeHtml(ctx.t("sites.editor.idPlaceholder"))}">
                  </div>
                  <div class="waf-field">
                    <label for="service-security-mode">${escapeHtml(ctx.t("sites.editor.securityMode"))}</label>
                    <select id="service-security-mode">
                      <option value="transparent"${draft.security_mode === "transparent" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.transparent"))}</option>
                      <option value="monitor"${draft.security_mode === "monitor" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.monitor"))}</option>
                      <option value="block"${draft.security_mode === "block" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.block"))}</option>
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="service-profile">${escapeHtml(ctx.t("sites.table.profile"))}</label>
                    <select id="service-profile">
                      <option value="strict"${normalizeServiceProfile(draft.service_profile) === "strict" ? " selected" : ""}>${escapeHtml(ctx.t("sites.profile.strict"))}</option>
                      <option value="balanced"${normalizeServiceProfile(draft.service_profile) === "balanced" ? " selected" : ""}>${escapeHtml(ctx.t("sites.profile.balanced"))}</option>
                      <option value="compat"${normalizeServiceProfile(draft.service_profile) === "compat" ? " selected" : ""}>${escapeHtml(ctx.t("sites.profile.compat"))}</option>
                      <option value="api"${normalizeServiceProfile(draft.service_profile) === "api" ? " selected" : ""}>${escapeHtml(ctx.t("sites.profile.api"))}</option>
                      <option value="public-edge"${normalizeServiceProfile(draft.service_profile) === "public-edge" ? " selected" : ""}>${escapeHtml(ctx.t("sites.profile.public-edge"))}</option>
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="service-ca-server">${escapeHtml(ctx.t("sites.easy.front.caServer"))}</label>
                    <select id="service-ca-server">
                      <option value="letsencrypt"${draft.certificate_authority_server === "letsencrypt" ? " selected" : ""}>letsencrypt</option>
                      <option value="zerossl"${draft.certificate_authority_server === "zerossl" ? " selected" : ""}>zerossl</option>
                      <option value="custom"${draft.certificate_authority_server === "custom" ? " selected" : ""}>custom</option>
                      <option value="import"${draft.certificate_authority_server === "import" ? " selected" : ""}>${escapeHtml(ctx.t("sites.tls.importOption"))}</option>
                    </select>
                  </div>
                  <label class="waf-checkbox">
                    <input id="service-enabled" type="checkbox"${draft.enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.serviceEnabled"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-adaptive-model-enabled" type="checkbox"${draft.adaptive_model_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.adaptiveModelEnabled"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-auto-lets-encrypt" type="checkbox"${draft.auto_lets_encrypt ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.autoLetsEncrypt"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-lets-encrypt-staging" type="checkbox"${draft.use_lets_encrypt_staging ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.letsEncryptStaging"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-lets-encrypt-wildcard" type="checkbox"${draft.use_lets_encrypt_wildcard ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.wildcardCertificates"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-tls-enabled" type="checkbox"${draft.tls_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.tlsEnabled"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-tls-self-signed" type="checkbox"${draft.tls_self_signed ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.tlsSelfSigned"))}</span>
                  </label>
                  <div class="waf-field waf-field-cert-id">
                    <label for="service-certificate-id">${escapeHtml(ctx.t("sites.tls.certificateId"))}</label>
                    <input id="service-certificate-id" value="${escapeHtml(draft.certificate_id)}" placeholder="${escapeHtml(ctx.t("sites.tls.certificatePlaceholder"))}">
                    <div class="waf-field" id="service-certificate-picker"${draft.certificate_authority_server === "import" ? "" : " style=\"display:none\""}>
                      <label for="service-import-certificate-search">${escapeHtml(ctx.t("sites.tls.importSelect"))}</label>
                      <input id="service-import-certificate-search" list="service-import-certificate-list" placeholder="${escapeHtml(ctx.t("sites.tls.importSelectPlaceholder"))}">
                      <datalist id="service-import-certificate-list">
                        ${state.certificates.map((certificate) => {
                          const id = String(certificate?.id || "").trim();
                          if (!id) {
                            return "";
                          }
                          const commonName = String(certificate?.common_name || "").trim();
                          const status = String(certificate?.status || "unknown").trim();
                          const label = commonName ? `${id} (${commonName}, ${status})` : `${id} (${status})`;
                          return `<option value="${escapeHtml(id)}" label="${escapeHtml(label)}"></option>`;
                        }).join("")}
                      </datalist>
                    </div>
                    <div class="waf-actions" id="service-certificate-import-actions"${draft.certificate_authority_server === "import" ? "" : " style=\"display:none\""}>
                      <button class="btn ghost btn-sm" type="button" id="service-certificate-import">${escapeHtml(ctx.t("sites.tls.importButton"))}</button>
                      <button class="btn ghost btn-sm" type="button" id="service-certificate-export">${escapeHtml(ctx.t("sites.tls.exportButton"))}</button>
                    </div>
                    <input id="service-certificate-archive-file" type="file" accept=".zip,application/zip,application/x-zip-compressed" class="waf-hidden">
                  </div>
                </div>
              </section>

              <section class="waf-stack${state.activeTab === "upstream" ? "" : " waf-hidden"}" data-tab-panel="upstream">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.wizard.upstream.title"))}</div>
                <input id="service-upstream-id" type="hidden" value="${escapeHtml(draft.upstream_id)}">
                <div class="waf-form-grid three waf-upstream-toggle-row">
                  <label class="waf-checkbox">
                    <input id="service-use-reverse-proxy" type="checkbox"${draft.use_reverse_proxy ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.useReverseProxy"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-keepalive" type="checkbox"${draft.reverse_proxy_keepalive ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.keepalive"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-websocket" type="checkbox"${draft.reverse_proxy_websocket ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.websocket"))}</span>
                  </label>
                </div>
                <div class="waf-upstream-layout">
                  <div class="waf-form-grid three">
                    <div class="waf-field">
                      <label for="service-reverse-proxy-custom-host">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyCustomHost"))}</label>
                      <input id="service-reverse-proxy-custom-host" value="${escapeHtml(draft.reverse_proxy_custom_host)}"${draft.pass_host_header ? "" : " disabled"}>
                    </div>
                    <div class="waf-field">
                    <label for="service-reverse-proxy-host">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyHost"))}</label>
                    <input id="service-reverse-proxy-host" value="${escapeHtml(draft.reverse_proxy_host)}">
                    </div>
                    <div class="waf-field">
                    <label for="service-reverse-proxy-url">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyUrl"))}</label>
                    <input id="service-reverse-proxy-url" value="${escapeHtml(draft.reverse_proxy_url)}">
                    </div>
                  </div>
                  <div class="waf-upstream-target-row">
                    <div class="waf-field waf-field-compact-xs">
                      <label for="service-upstream-scheme">${escapeHtml(ctx.t("sites.upstream.field.scheme"))}</label>
                      <select id="service-upstream-scheme">
                        <option value="http"${draft.upstream_scheme === "http" ? " selected" : ""}>http</option>
                        <option value="https"${draft.upstream_scheme === "https" ? " selected" : ""}>https</option>
                      </select>
                    </div>
                    <div class="waf-field waf-field-compact-md">
                      <label for="service-upstream-host">${escapeHtml(ctx.t("sites.upstream.field.host"))}</label>
                      <input id="service-upstream-host" value="${escapeHtml(draft.upstream_host)}">
                    </div>
                    <div class="waf-field waf-field-compact-xs">
                      <label for="service-upstream-port">${escapeHtml(ctx.t("sites.upstream.field.port"))}</label>
                      <input id="service-upstream-port" type="number" min="1" max="65535" value="${escapeHtml(String(draft.upstream_port))}">
                    </div>
                  </div>
                </div>
                <div class="waf-subframe waf-upstream-headers-frame">
                  <div class="waf-list-title-sm">${escapeHtml(ctx.t("sites.easy.upstream.headerForwardingTitle"))}</div>
                  <div class="waf-note">${escapeHtml(ctx.t("sites.easy.upstream.headerForwardingHint"))}</div>
                  <div class="waf-form-grid two">
                    <label class="waf-checkbox">
                      <input id="service-pass-host-header" type="checkbox"${draft.pass_host_header ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.passHostHeader"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-forwarded-for" type="checkbox"${draft.send_x_forwarded_for ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXForwardedFor"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-forwarded-proto" type="checkbox"${draft.send_x_forwarded_proto ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXForwardedProto"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-real-ip" type="checkbox"${draft.send_x_real_ip ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXRealIp"))}</span>
                    </label>
                  </div>
                </div>
                <div class="waf-form-grid two waf-upstream-sni-row">
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-ssl-sni" type="checkbox"${draft.reverse_proxy_ssl_sni ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.useSslSni"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-reverse-proxy-ssl-sni-name">${escapeHtml(ctx.t("sites.easy.upstream.sslSniName"))}</label>
                    <input id="service-reverse-proxy-ssl-sni-name" value="${escapeHtml(draft.reverse_proxy_ssl_sni_name)}">
                  </div>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "http" ? "" : " waf-hidden"}" data-tab-panel="http">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.http.title"))}</div>
                <div class="waf-form-grid">
                  ${renderListEditor("allowed_methods", ctx.t("sites.easy.http.allowedMethods"), draft.allowed_methods, "GET", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <div class="waf-field">
                    <label for="service-max-client-size">${escapeHtml(ctx.t("sites.easy.http.maxBodySize"))}</label>
                    <input id="service-max-client-size" value="${escapeHtml(draft.max_client_size)}">
                  </div>
                  ${renderListEditor("ssl_protocols", ctx.t("sites.easy.http.sslProtocols"), draft.ssl_protocols, "TLSv1.3", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <label class="waf-checkbox">
                    <input id="service-http2" type="checkbox"${draft.http2 ? " checked" : ""}>
                    <span>HTTP2</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-http3" type="checkbox"${draft.http3 ? " checked" : ""}>
                    <span>HTTP3</span>
                  </label>
                </div>
              </section>

${renderDetailViewRuntimeTail(state, ctx, deps, draft, isNew)}
  `;
}
