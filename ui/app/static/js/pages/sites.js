import { confirmAction, escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";
import {
  certificateDaysLeft,
  findEasyProfile,
  formatCertificateExpiryByLanguage,
  go,
  mergeByID,
  mergeProfilesBySite,
  normalizeArray,
  normalizeSiteID,
  notifyExpiringCertificates,
  routeBase,
  routeInfo,
  tryGetJSON,
  unwrapList,
} from "./sites.routing-merge.js";
import {
  applyServiceProfilePresetForMissingFields,
  applyServiceProfilePresetToDraft,
  formatServiceProfile,
  normalizeAPIPositiveEndpointPolicies,
  normalizeAntibotChallengeRules,
  normalizeCustomLimitRules,
  normalizeHost,
  normalizeServiceProfile,
  normalizeStringArray,
  parseIntListInput,
  parseListInput,
} from "./sites.normalize.js";
import {
  normalizeAuthBasicUsers,
  normalizeAuthSessionTTLMinutes,
  renderAuthSessionTtlOptions,
  renderAuthUsersEditor,
  syncAuthPasswordToggle,
} from "./sites.auth-geo.js";
import {
  BAN_SCOPE_VALUES,
  buildReverseProxyHostFromUpstream,
  computeUpstreamID,
  formatBanDurationSeconds,
  isValidEmail,
  normalizeBanEscalationStages,
  normalizeEmail,
  normalizeReverseProxyHost,
  parseBanDurationSeconds,
  resolvePublicServiceURL,
  resolveReverseProxyHost,
} from "./sites.traffic-helpers.js";
import {
  BAD_BEHAVIOR_STATUS_OPTIONS,
  LIST_FIELD_SET,
  SETTINGS_SEARCH_INDEX,
  buildGeoCatalogFallback,
  getQuickListTemplates,
  normalizeGeoCatalogPayload,
  regionDisplayLabel,
  regionDisplayName,
} from "./sites.geo-lists.js";
import {
  renderCountryEditor,
  renderListEditor,
  renderStatusCodesEditor,
} from "./sites.list-renderers.js";
import {
  defaultSiteDraft,
  draftToEasyProfile,
} from "./sites.draft-core.js";
import {
  hydrateSiteDraft as hydrateSiteDraftModule,
} from "./sites.profile-hydration.js";
import {
  downloadBlob,
  downloadJSON,
  downloadText,
  draftToEnvText as draftToEnvTextModule,
  envToDraft,
  importServicesFiles as importServicesFilesModule,
  requirePermissions,
  toEnvKey,
} from "./sites.import-pipeline.js";
import {
  isAutoApplyFailureError,
  putWithPostFallback,
  shouldUpsertBaseResources,
  upsertAccessPolicy,
} from "./sites.access-upsert.js";
import {
  isAlreadyExistsError,
  resolveACMEAccountEmail,
  upsertSiteResources as upsertSiteResourcesModule,
} from "./sites.resource-pipeline.js";
import { validateDraft as validateDraftModule } from "./sites.validation.js";
import {
  deleteServiceWithResources as deleteServiceWithResourcesModule,
  ensureControlPlaneAccessManagementMethods,
  exportSelectedServicesEnv as exportSelectedServicesEnvModule,
  importServicesFiles as importServicesFilesLifecycle,
} from "./sites.service-lifecycle.js";
import {
  renderModeTabs as renderModeTabsModule,
  renderRawEditor as renderRawEditorModule,
  renderWizardNav as renderWizardNavModule,
} from "./sites.detail-shell.js";
import {
  bindList as bindListModule,
  renderListView as renderListViewModule,
} from "./sites.list-view.js";
import {
  getDraftFromForm,
  highlightSelector as highlightSelectorModule,
  normalizeAutoSiteID,
  parseRawDraftFromContainer,
  syncDerivedFieldsFromID as syncDerivedFieldsFromIDModule,
  syncStateDraftFromForm as syncStateDraftFromFormModule,
  toggleCertificateImportActions as toggleCertificateImportActionsModule,
} from "./sites.detail-draft.js";
import { bindDetailListEditors } from "./sites.detail-list-bindings.js";
import { bindDetailSubmitDelete } from "./sites.detail-submit-delete.js";
import { bindDetailCore } from "./sites.detail-core-bindings.js";
import {
  bindDetailBulkDelete,
  bindDetailCertificateActions,
} from "./sites.detail-certs-bulk.js";

let pendingImportedDraft = null;
let sitesErrorLoggingInstalled = false;

function formatSitesError(error) {
  if (!error) {
    return "unknown error";
  }
  if (error instanceof Error) {
    const message = String(error.message || error.name || "error");
    const stack = String(error.stack || "").trim();
    return stack ? `${message}\n${stack}` : message;
  }
  return String(error);
}

function logSitesError(stage, error, details = {}) {
  const payload = {
    stage,
    details,
    error: formatSitesError(error),
    href: window.location.href,
    at: new Date().toISOString(),
  };
  console.error("[sites-page]", payload);
  return payload;
}

function renderSitesErrorAlert(ctx, payload) {
  const detailsText = `${payload.stage}\n${payload.error}`;
  return `
    <div class="alert">
      <div>${escapeHtml(ctx.t("sites.error.load"))}</div>
      <pre class="waf-code" style="margin-top:8px;white-space:pre-wrap;">${escapeHtml(detailsText)}</pre>
    </div>
  `;
}

function installSitesGlobalErrorLogging() {
  if (sitesErrorLoggingInstalled) {
    return;
  }
  sitesErrorLoggingInstalled = true;
  window.addEventListener("error", (event) => {
    logSitesError("window.error", event?.error || event?.message || event, {
      filename: event?.filename || "",
      lineno: Number(event?.lineno || 0),
      colno: Number(event?.colno || 0),
    });
  });
  window.addEventListener("unhandledrejection", (event) => {
    logSitesError("window.unhandledrejection", event?.reason || event, {});
  });
}

/*
SITES-01 legacy block moved to:
- sites.routing-merge.js
- sites.normalize.js
- sites.auth-geo.js
*/

function renderCustomLimitRulesEditor(rules, ctx) {
  const safeRules = normalizeCustomLimitRules(rules);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.traffic.customLimitRules"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-custom-limit-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}">
            <input data-custom-limit-rate="${index}" placeholder="20r/s" value="${escapeHtml(rule.rate)}">
            <button class="btn ghost btn-sm" type="button" data-custom-limit-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-custom-limit-add>${escapeHtml(ctx.t("sites.easy.traffic.addCustomLimit"))}</button>
      </div>
    </div>`;
}

function renderAntibotChallengeRulesEditor(rules, ctx) {
  const safeRules = normalizeAntibotChallengeRules(rules);
  const modes = ["cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"];
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.antibot.challengeRulesByUrl"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-antibot-rule-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}">
            <select data-antibot-rule-challenge="${index}">
              ${modes.map((mode) => `<option value="${mode}"${rule.challenge === mode ? " selected" : ""}>${mode}</option>`).join("")}
            </select>
            <button class="btn ghost btn-sm" type="button" data-antibot-rule-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-antibot-rule-add>${escapeHtml(ctx.t("sites.easy.antibot.addChallengeRule"))}</button>
      </div>
    </div>`;
}


/* SITES-02 legacy block moved to:
- sites.traffic-helpers.js
- sites.geo-lists.js
- sites.list-renderers.js
*/

/* SITES-03 legacy block moved to:
- sites.draft-core.js
- sites.profile-hydration.js
*/

async function hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy = null) {
  return hydrateSiteDraftModule(ctx, site, upstream, tlsConfig, accessPolicy);
}

function draftToEnvText(draft) {
  return draftToEnvTextModule(draft);
}

function validateDraft(draft, ctx) {
  return validateDraftModule(draft, ctx);
}

/* SITES-04 legacy block moved to:
- sites.import-pipeline.js
*/

function renderListView(state, ctx) {
  return renderListViewModule(state, ctx, formatCertificateExpiryByLanguage, statusBadge, formatDate);
}

function renderModeTabs(activeMode, ctx) {
  return renderModeTabsModule(activeMode, ctx);
}

function renderWizardNav(activeTab, ctx) {
  return renderWizardNavModule(activeTab, ctx);
}

function renderRawEditor(state, ctx, isNew) {
  return renderRawEditorModule(state, ctx, isNew, toEnvKey, draftToEnvText);
}

function renderDetailView(state, ctx) {
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
                  <div class="waf-field waf-http-max-size">
                    <label for="service-max-client-size">${escapeHtml(ctx.t("sites.easy.http.maxBodySize"))}</label>
                    <input id="service-max-client-size" value="${escapeHtml(draft.max_client_size)}">
                  </div>
                  ${renderListEditor("ssl_protocols", ctx.t("sites.easy.http.sslProtocols"), draft.ssl_protocols, "TLSv1.3", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <label class="waf-checkbox waf-http-version-toggle">
                    <input id="service-http2" type="checkbox"${draft.http2 ? " checked" : ""}>
                    <span>HTTP2</span>
                  </label>
                  <label class="waf-checkbox waf-http-version-toggle">
                    <input id="service-http3" type="checkbox"${draft.http3 ? " checked" : ""}>
                    <span>HTTP3</span>
                  </label>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "headers" ? "" : " waf-hidden"}" data-tab-panel="headers">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.headers.title"))}</div>
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
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.traffic.title"))}</div>
                <div class="waf-traffic-layout">
                  <div class="waf-stack">
                    <div class="waf-subcard waf-stack waf-antiddos-frame">
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior"))}</div>
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
                      <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.limits"))}</div>
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
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.dnsbl"))}</div>
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
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "blocking" ? "" : " waf-hidden"}" data-tab-panel="blocking">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.blocking.title"))}</div>
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
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.antibot.title"))}</div>
                <div class="waf-antibot-auth-grid">
                  <section class="waf-subcard waf-antibot-editor-frame">
                    <div class="waf-card-head">
                      <h3>${escapeHtml(ctx.t("sites.easy.antibot.frameTitle"))}</h3>
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
                      <h3>${escapeHtml(ctx.t("sites.easy.antibot.authSectionTitle"))}</h3>
                    </div>
                    <div class="waf-card-body">
                      <div class="waf-form-grid">
                        <label class="waf-checkbox">
                          <input id="service-use-auth-basic" type="checkbox"${draft.use_auth_basic ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.antibot.useAuthBasic"))}</span>
                        </label>
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
                        ${renderAuthUsersEditor(draft.auth_basic_users, ctx)}
                      </div>
                    </div>
                  </section>
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "geo" ? "" : " waf-hidden"}" data-tab-panel="geo">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.geo.title"))}</div>
                <div class="waf-form-grid">
                  ${renderCountryEditor("blacklist_country", ctx.t("sites.easy.geo.countryBlacklist"), draft.blacklist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.blacklist_country, ctx })}
                  ${renderCountryEditor("whitelist_country", ctx.t("sites.easy.geo.countryWhitelist"), draft.whitelist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.whitelist_country, ctx })}
                </div>
              </section>

              <section class="waf-stack waf-service-compact-section${state.activeTab === "modsec" ? "" : " waf-hidden"}" data-tab-panel="modsec">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.modsec.title"))}</div>
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
      `}
    </div>
  `;
}

/* SITES-05 legacy block moved to:
- sites.access-upsert.js
*/

/* SITES-06 legacy block moved to:
- sites.resource-pipeline.js
*/

async function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig, options = {}) {
  return upsertSiteResourcesModule(
    draft,
    ctx,
    resolveACMEAccountEmail,
    existingSite,
    existingUpstream,
    existingTLSConfig,
    options
  );
}

async function deleteServiceWithResources(siteID, ctx, snapshot = null) {
  return deleteServiceWithResourcesModule(siteID, ctx, isAutoApplyFailureError, snapshot);
}

async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, accessBySite, selectedSiteIDs) {
  return exportSelectedServicesEnvModule(
    ctx,
    sites,
    upstreamsBySite,
    tlsBySite,
    accessBySite,
    selectedSiteIDs,
    hydrateSiteDraft,
    downloadText,
    draftToEnvText
  );
}

async function importServicesFiles(files, ctx) {
  return importServicesFilesLifecycle(
    files,
    ctx,
    importServicesFilesModule,
    validateDraft,
    upsertSiteResources,
    putWithPostFallback
  );
}

export async function renderSites(container, ctx) {
  installSitesGlobalErrorLogging();
  const route = routeInfo();
  const state = {
    route,
    sites: [],
    upstreams: [],
    tlsConfigs: [],
    certificates: [],
    accessPolicies: [],
    easyProfiles: [],
    geoCatalog: buildGeoCatalogFallback(),
    search: "",
    sort: "updated-desc",
    editorMode: "easy",
    activeTab: "front",
    settingsSearch: "",
    settingsMatches: [],
    highlightedSelector: "",
    rawEnvText: "",
    rawMissingFields: [],
    filteredSites: [],
    selectedSiteIDs: new Set(),
    upstreamsBySite: new Map(),
    tlsBySite: new Map(),
    certificateBySiteID: new Map(),
    certificateByHost: new Map(),
    accessBySite: new Map(),
    easyProfilesBySite: new Map(),
    pendingImportedDraftRef: {
      set(imported) {
        pendingImportedDraft = imported
          ? {
              draft: imported.draft,
              missingFields: imported.missingFields,
              rawEnvText: imported.rawEnvText,
            }
          : null;
      },
    },
    countryFilters: {
      blacklist_country: "",
      whitelist_country: ""
    },
    listTemplateSelection: {
      blacklist_user_agent: [],
      blacklist_uri: []
    },
    draft: defaultSiteDraft()
  };

  const rebuildIndexes = () => {
    state.upstreamsBySite = new Map();
    state.tlsBySite = new Map();
    state.certificateBySiteID = new Map();
    state.certificateByHost = new Map();
    state.accessBySite = new Map();
    state.easyProfilesBySite = new Map();
    for (const upstream of state.upstreams) {
      const items = state.upstreamsBySite.get(upstream.site_id) || [];
      items.push(upstream);
      state.upstreamsBySite.set(upstream.site_id, items);
    }
    for (const tlsConfig of state.tlsConfigs) {
      state.tlsBySite.set(tlsConfig.site_id, tlsConfig);
    }
    for (const certificate of state.certificates) {
      const certificateID = String(certificate?.id || "").trim().toLowerCase();
      if (certificateID.endsWith("-tls")) {
        const relatedSiteID = certificateID.slice(0, -4);
        if (relatedSiteID && !state.certificateBySiteID.has(relatedSiteID)) {
          state.certificateBySiteID.set(relatedSiteID, certificate);
        }
      }
      const host = normalizeHost(certificate?.common_name);
      if (host && !state.certificateByHost.has(host)) {
        state.certificateByHost.set(host, certificate);
      }
    }
    for (const accessPolicy of state.accessPolicies) {
      const siteID = normalizeSiteID(accessPolicy?.site_id);
      if (!siteID || state.accessBySite.has(siteID)) {
        continue;
      }
      state.accessBySite.set(siteID, accessPolicy);
    }
    for (const easyProfile of state.easyProfiles) {
      const siteID = normalizeSiteID(easyProfile?.site_id);
      if (!siteID || state.easyProfilesBySite.has(siteID)) {
        continue;
      }
      state.easyProfilesBySite.set(siteID, easyProfile);
    }
  };

  const applyFilters = () => {
    const search = state.search.trim().toLowerCase();
    const sites = state.sites.filter((site) => {
      if (!search) {
        return true;
      }
      const profile = state.easyProfilesBySite.get(normalizeSiteID(site.id));
      const profileValue = normalizeServiceProfile(profile?.front_service?.profile);
      return `${site.id} ${site.primary_host} ${profileValue}`.toLowerCase().includes(search);
    });
    sites.sort((left, right) => {
      if (state.sort === "name-asc") {
        return String(left.primary_host || left.id).localeCompare(String(right.primary_host || right.id));
      }
      if (state.sort === "name-desc") {
        return String(right.primary_host || right.id).localeCompare(String(left.primary_host || left.id));
      }
      if (state.sort === "created-desc") {
        return String(right.created_at || "").localeCompare(String(left.created_at || ""));
      }
      return String(right.updated_at || right.created_at || "").localeCompare(String(left.updated_at || left.created_at || ""));
    });
    state.filteredSites = sites;
  };

  const syncDraftFromRoute = async () => {
    if (state.route.mode === "list") {
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    if (state.route.mode === "create") {
      if (pendingImportedDraft && pendingImportedDraft.draft) {
        state.draft = ensureControlPlaneAccessManagementMethods({ ...pendingImportedDraft.draft });
        state.rawEnvText = String(pendingImportedDraft.rawEnvText || draftToEnvText(state.draft));
        state.rawMissingFields = normalizeArray(pendingImportedDraft.missingFields);
        pendingImportedDraft = null;
      } else {
        state.draft = defaultSiteDraft();
        state.rawEnvText = draftToEnvText(state.draft);
        state.rawMissingFields = [];
      }
      state.editorMode = "easy";
      state.listTemplateSelection.blacklist_user_agent = [];
      state.listTemplateSelection.blacklist_uri = [];
      state.activeTab = "front";
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    const site = state.sites.find((item) => item.id === state.route.siteID);
    const upstream = state.upstreamsBySite.get(state.route.siteID)?.[0] || null;
    const tlsConfig = state.tlsBySite.get(state.route.siteID) || null;
    const accessPolicy = state.accessBySite.get(normalizeSiteID(state.route.siteID)) || null;
    state.draft = await hydrateSiteDraft(ctx, site, upstream, tlsConfig, accessPolicy);
    state.listTemplateSelection.blacklist_user_agent = [];
    state.listTemplateSelection.blacklist_uri = [];
    state.rawEnvText = draftToEnvText(state.draft);
    state.rawMissingFields = [];
    state.editorMode = "easy";
    state.activeTab = "front";
    state.settingsSearch = "";
    state.settingsMatches = [];
    state.highlightedSelector = "";
  };

  const render = () => {
    try {
      applyFilters();
      container.innerHTML = state.route.mode === "list" ? renderListView(state, ctx) : renderDetailView(state, ctx);
      bind();
    } catch (error) {
      const payload = logSitesError("render", error, {
        routeMode: state.route?.mode || "",
        siteID: state.route?.siteID || "",
      });
      container.innerHTML = renderSitesErrorAlert(ctx, payload);
    }
  };

  const load = async () => {
    try {
      setLoading(container, ctx.t("sites.loading"));
      const [sitesResponse, upstreamsResponse, tlsConfigsResponse, certificatesResponse, accessPoliciesResponse, easyProfilesResponse, geoCatalogResponse] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/upstreams"),
        ctx.api.get("/api/tls-configs"),
        ctx.api.get("/api/certificates").catch(() => []),
        ctx.api.get("/api/access-policies").catch(() => []),
        ctx.api.get("/api/easy-site-profiles").catch(() => []),
        ctx.api.get("/api/easy-site-profiles/catalog/countries").catch(() => null)
      ]);
      const [secondarySites, secondaryUpstreams, secondaryTLSConfigs, secondaryCertificates, secondaryEasyProfiles] = await Promise.all([
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/upstreams"),
        tryGetJSON("/api-app/tls-configs"),
        tryGetJSON("/api-app/certificates"),
        tryGetJSON("/api-app/easy-site-profiles")
      ]);
      state.sites = mergeByID(sitesResponse, unwrapList(secondarySites, ["sites"]));
      state.upstreams = mergeByID(upstreamsResponse, unwrapList(secondaryUpstreams, ["upstreams"]));
      state.tlsConfigs = mergeByID(tlsConfigsResponse, unwrapList(secondaryTLSConfigs, ["tls_configs", "tlsConfigs"]));
      state.certificates = mergeByID(certificatesResponse, unwrapList(secondaryCertificates, ["certificates"]));
      notifyExpiringCertificates(ctx, state.certificates);
      state.accessPolicies = normalizeArray(accessPoliciesResponse);
      state.easyProfiles = mergeProfilesBySite(easyProfilesResponse, secondaryEasyProfiles);
      state.selectedSiteIDs = new Set(Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => site.id === id)));
      state.geoCatalog = normalizeGeoCatalogPayload(geoCatalogResponse);
      rebuildIndexes();
      await syncDraftFromRoute();
      render();
    } catch (error) {
      const payload = logSitesError("load", error, {
        routeMode: state.route?.mode || "",
        siteID: state.route?.siteID || "",
      });
      container.innerHTML = renderSitesErrorAlert(ctx, payload);
    }
  };

  const bindList = () => {
    bindListModule(container, state, ctx, {
      go,
      load,
      downloadJSON,
      exportSelectedServicesEnv,
      importServicesFiles,
      toEnvKey,
      putWithPostFallback,
      normalizeSiteID,
      render,
    });
  };

  const bindDetail = () => {
    const feedback = container.querySelector("#sites-feedback");
    const parseRawDraft = () => parseRawDraftFromContainer(container, state, {
      envToDraft,
      normalizeArray,
      applyServiceProfilePresetForMissingFields,
    });
    const getDraft = () => getDraftFromForm(container, state, {
      normalizeServiceProfile,
      computeUpstreamID,
      normalizeEmail,
      normalizeStringArray,
      normalizeArray,
      normalizeBanEscalationStages,
      normalizeAuthBasicUsers,
      normalizeAuthSessionTTLMinutes,
      normalizeAPIPositiveEndpointPolicies,
    });
    const syncStateDraftFromForm = () => syncStateDraftFromFormModule(state, getDraft, {
      normalizeArray,
      BAN_SCOPE_VALUES,
      normalizeBanEscalationStages,
      normalizeAuthBasicUsers,
      normalizeAuthSessionTTLMinutes,
    });
    const syncDerivedFieldsFromID = (idInput, certificateInput, upstreamInput) => syncDerivedFieldsFromIDModule(
      idInput,
      certificateInput,
      upstreamInput,
      computeUpstreamID
    );

    bindDetailCore(container, state, ctx, {
      go,
      render,
      getDraft,
      parseRawDraft,
      syncStateDraftFromForm,
      draftToEnvText,
      ensureControlPlaneAccessManagementMethods,
      normalizeAutoSiteID,
      syncDerivedFieldsFromID,
      normalizeServiceProfile,
      applyServiceProfilePresetToDraft,
      toggleCertificateImportActions: toggleCertificateImportActionsModule,
      highlightSelector: highlightSelectorModule,
    });

    bindDetailBulkDelete(container, state, ctx, {
      load,
      deleteServiceWithResources,
    });
    bindDetailCertificateActions(container, ctx, {
      load,
      downloadBlob,
    });

    bindDetailListEditors(container, state, {
      LIST_FIELD_SET,
      getQuickListTemplates,
      normalizeStringArray,
      normalizeCustomLimitRules,
      normalizeAntibotChallengeRules,
      normalizeAuthBasicUsers,
      syncAuthPasswordToggle,
      normalizeArray,
      parseBanDurationSeconds,
      normalizeBanEscalationStages,
      setError,
      render,
      syncStateDraftFromForm,
      ctx,
      feedback,
    });

    bindDetailSubmitDelete(container, state, ctx, {
      parseRawDraft,
      getDraft,
      syncStateDraftFromForm,
      ensureControlPlaneAccessManagementMethods,
      validateDraft,
      shouldUpsertBaseResources,
      upsertSiteResources,
      upsertAccessPolicy,
      putWithPostFallback,
      draftToEasyProfile,
      go,
      deleteServiceWithResources,
    });

    if (state.highlightedSelector) {
      window.setTimeout(() => highlightSelectorModule(container, state.highlightedSelector), 30);
    }
  };

  const bind = () => {
    if (state.route.mode === "list") {
      bindList();
      return;
    }
    bindDetail();
  };

  await load();
}

