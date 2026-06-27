import { escapeHtml, formatDate, statusBadge } from "../ui.js";
import { formatCertificateExpiryByLanguage, normalizeArray } from "./sites.routing-merge.js";
import {
  normalizeAntibotChallengeRules,
  normalizeCustomLimitRules,
  normalizeServiceProfile,
  normalizeStringArray,
} from "./sites.normalize.js";
import { renderAntibotExclusionRulesEditor as renderAntibotExclusionRulesEditorModule } from "./sites.antibot-exclusion-editors.js";
import {
  normalizeAuthExclusionRules,
  normalizeAuthMode,
  normalizeAuthOrder,
  normalizeAuthServiceTokens,
  renderAuthExclusionRulesEditor,
  renderAuthServiceTokensEditor,
} from "./sites.auth-extended-editors.js";
import {
  renderAuthSessionTtlOptions,
  renderAuthUsersEditor,
} from "./sites.auth-geo.js";
import { renderAntibotHelpModal, renderAuthHelpModal } from "./sites.auth-help-modals.js";
import {
  renderTrafficBadBehaviorHelpModal,
  renderTrafficBlacklistHelpModal,
  renderTrafficAllowlistHelpModal,
  renderTrafficDnsblHelpModal,
  renderTrafficLimitsHelpModal,
  renderUpstreamHeadersHelpModal,
} from "./sites.frame-help-modals.js";
import {
  renderAntibotChapterHelpModal,
  renderBlockingChapterHelpModal,
  renderGeoChapterHelpModal,
  renderHeadersChapterHelpModal,
  renderHttpChapterHelpModal,
  renderModsecChapterHelpModal,
  renderUpstreamChapterHelpModal,
  renderWebSocketChapterHelpModal,
  renderVirtualPatchesChapterHelpModal,
  renderUpstreamMtlsChapterHelpModal,
  renderFrontMainHelpModal,
  renderFrontMtlsHelpModal,
} from "./sites.chapter-help-modals.js";
import {
  formatBanDurationSeconds,
  normalizeBanEscalationStages,
} from "./sites.traffic-helpers.js";
import {
  SETTINGS_SEARCH_INDEX,
  getQuickListTemplates,
} from "./sites.geo-lists.js";
import {
  renderCountryEditor,
  renderListEditor,
  renderStatusCodesEditor,
} from "./sites.list-renderers.js";
import { renderDetailViewRuntime } from "./sites.detail-render-view.js";
import { renderVirtualPatchesEditor as renderVirtualPatchesEditorModule } from "./sites.virtual-patches-editor.js";
import {
  renderModeTabs as renderModeTabsModule,
  renderRawEditor as renderRawEditorModule,
  renderWizardNav as renderWizardNavModule,
} from "./sites.detail-shell.js";
import { renderListView as renderListViewModule } from "./sites.list-view.js";
import { draftToEnvText, toEnvKey } from "./sites.stable-resources.js";

function renderCustomLimitRulesEditor(rules, ctx, deps, disabled = false) {
  const safeRules = normalizeCustomLimitRules(rules, { normalizeArray });
  const dis = disabled ? " disabled" : "";
  return `
    <div class="waf-field full${disabled ? " waf-disabled" : ""}">
      <label>${escapeHtml(ctx.t("sites.easy.traffic.customLimitRules"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-custom-limit-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}"${dis}>
            <div class="waf-custom-limit-rate-wrap">
              <input data-custom-limit-rate="${index}" type="number" min="1" inputmode="numeric" placeholder="20" value="${escapeHtml(String(rule.rate || "").replace(/r\/s$|r\/m$/i, "").trim())}"${dis}>
              <select data-custom-limit-rate-unit="${index}"${dis}>
                <option value="r/s"${!String(rule.rate || "").endsWith("r/m") ? " selected" : ""}>r/s</option>
                <option value="r/m"${String(rule.rate || "").endsWith("r/m") ? " selected" : ""}>r/m</option>
              </select>
            </div>
            <button class="btn ghost btn-sm" type="button" data-custom-limit-remove="${index}"${dis}>x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-custom-limit-add${dis}>${escapeHtml(ctx.t("sites.easy.traffic.addCustomLimit"))}</button>
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

function renderGeoTimeWindowsEditor(windows, geoCatalog, ctx) {
  const safeWindows = Array.isArray(windows) ? windows : [];
  const days = [
    ctx.t("sites.easy.geo.timeWindow.sun"),
    ctx.t("sites.easy.geo.timeWindow.mon"),
    ctx.t("sites.easy.geo.timeWindow.tue"),
    ctx.t("sites.easy.geo.timeWindow.wed"),
    ctx.t("sites.easy.geo.timeWindow.thu"),
    ctx.t("sites.easy.geo.timeWindow.fri"),
    ctx.t("sites.easy.geo.timeWindow.sat"),
  ];
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.geo.timeWindows"))}</label>
      <div class="waf-stack">
        ${safeWindows.map((w, index) => `
          <div class="waf-inline waf-custom-limit-row" style="flex-wrap:wrap;gap:4px;">
            <input data-geo-tw-countries="${index}" placeholder="RU,CN" value="${escapeHtml((w.countries || []).join(","))}" style="width:120px;" title="${escapeHtml(ctx.t("sites.easy.geo.timeWindow.countries"))}">
            <select data-geo-tw-action="${index}">
              <option value="block"${w.action === "block" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.geo.timeWindow.actionBlock"))}</option>
              <option value="allow"${w.action === "allow" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.geo.timeWindow.actionAllow"))}</option>
            </select>
            <input data-geo-tw-hours-start="${index}" type="number" min="0" max="23" value="${Number(w.hours_start) || 0}" style="width:56px;" title="${escapeHtml(ctx.t("sites.easy.geo.timeWindow.hoursStart"))}">
            <span>–</span>
            <input data-geo-tw-hours-end="${index}" type="number" min="0" max="23" value="${Number(w.hours_end) || 0}" style="width:56px;" title="${escapeHtml(ctx.t("sites.easy.geo.timeWindow.hoursEnd"))}">
            <span class="muted" style="font-size:0.85em;">UTC</span>
            <span>${escapeHtml(ctx.t("sites.easy.geo.timeWindow.days"))}</span>
            ${days.map((d, di) => `<label class="waf-checkbox" style="font-size:0.85em;"><input type="checkbox" data-geo-tw-day="${index}-${di}"${(w.days_of_week || []).includes(di) ? " checked" : ""}><span>${escapeHtml(d)}</span></label>`).join("")}
            <button class="btn ghost btn-sm" type="button" data-geo-tw-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeWindows.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-geo-tw-add>${escapeHtml(ctx.t("sites.easy.geo.timeWindow.add"))}</button>
      </div>
    </div>`;
}

function renderAntibotExclusionRulesEditor(rules, ctx) {
  return renderAntibotExclusionRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

const renderAuthHelpModalSafe = (ctx) => renderAuthHelpModal(ctx, { escapeHtml });
const renderAntibotHelpModalSafe = (ctx) => renderAntibotHelpModal(ctx, { escapeHtml });
const renderTrafficBadBehaviorHelpModalSafe = (ctx) => renderTrafficBadBehaviorHelpModal(ctx, escapeHtml);
const renderTrafficBlacklistHelpModalSafe = (ctx) => renderTrafficBlacklistHelpModal(ctx, escapeHtml);
const renderTrafficAllowlistHelpModalSafe = (ctx) => renderTrafficAllowlistHelpModal(ctx, escapeHtml);
const renderTrafficLimitsHelpModalSafe = (ctx) => renderTrafficLimitsHelpModal(ctx, escapeHtml);
const renderTrafficDnsblHelpModalSafe = (ctx) => renderTrafficDnsblHelpModal(ctx, escapeHtml);
const renderUpstreamHeadersHelpModalSafe = (ctx) => renderUpstreamHeadersHelpModal(ctx, escapeHtml);

export function renderListView(state, ctx) {
  return renderListViewModule(state, ctx, formatCertificateExpiryByLanguage, statusBadge, formatDate);
}

export function renderModeTabs(activeMode, ctx) {
  return renderModeTabsModule(activeMode, ctx);
}

export function renderWizardNav(activeTab, ctx) {
  return renderWizardNavModule(activeTab, ctx);
}

export function renderRawEditor(state, ctx, isNew) {
  return renderRawEditorModule(state, ctx, isNew, toEnvKey, draftToEnvText);
}

export function renderDetailView(state, ctx) {
  return renderDetailViewRuntime(state, ctx, {
    SETTINGS_SEARCH_INDEX,
    escapeHtml,
    renderModeTabs,
    renderRawEditor,
    renderWizardNav,
    normalizeServiceProfile,
    renderListEditor,
    getQuickListTemplates,
    normalizeStringArray,
    renderStatusCodesEditor,
    renderCustomLimitRulesEditor,
    renderAntibotExclusionRulesEditor,
    normalizeBanEscalationStages,
    formatBanDurationSeconds,
    renderAntibotChallengeRulesEditor,
    renderAuthSessionTtlOptions,
    renderAuthUsersEditor,
    renderCountryEditor,
    renderAuthExclusionRulesEditor: (rules, childCtx) =>
      renderAuthExclusionRulesEditor(rules, childCtx, { escapeHtml, normalizeArray }),
    renderAuthServiceTokensEditor: (tokens, childCtx) =>
      renderAuthServiceTokensEditor(tokens, childCtx, { escapeHtml, normalizeArray }),
    renderAuthHelpModal: renderAuthHelpModalSafe,
    renderAntibotHelpModal: renderAntibotHelpModalSafe,
    renderTrafficBadBehaviorHelpModal: renderTrafficBadBehaviorHelpModalSafe,
    renderTrafficBlacklistHelpModal: renderTrafficBlacklistHelpModalSafe,
    renderTrafficAllowlistHelpModal: renderTrafficAllowlistHelpModalSafe,
    renderTrafficLimitsHelpModal: renderTrafficLimitsHelpModalSafe,
    renderTrafficDnsblHelpModal: renderTrafficDnsblHelpModalSafe,
    renderUpstreamHeadersHelpModal: renderUpstreamHeadersHelpModalSafe,
    renderUpstreamChapterHelpModal,
    renderHttpChapterHelpModal,
    renderHeadersChapterHelpModal,
    renderBlockingChapterHelpModal,
    renderAntibotChapterHelpModal,
    renderGeoChapterHelpModal,
    renderModsecChapterHelpModal,
    renderWebSocketChapterHelpModal,
    renderVirtualPatchesChapterHelpModal,
    renderUpstreamMtlsChapterHelpModal,
    renderFrontMainHelpModal,
    renderFrontMtlsHelpModal,
    renderGeoTimeWindowsEditor,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthExclusionRules,
    normalizeAuthServiceTokens,
    renderVirtualPatchesEditor: (vpState, vpCtx) => renderVirtualPatchesEditorModule(vpState, vpCtx),
  });
}
