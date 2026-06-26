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
  renderTrafficDnsblHelpModal,
  renderTrafficLimitsHelpModal,
  renderUpstreamHeadersHelpModal,
} from "./sites.frame-help-modals.js";
import {
  renderAntibotChapterHelpModal,
  renderBlockingChapterHelpModal,
  renderFrontChapterHelpModal,
  renderGeoChapterHelpModal,
  renderHeadersChapterHelpModal,
  renderHttpChapterHelpModal,
  renderModsecChapterHelpModal,
  renderUpstreamChapterHelpModal,
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
import {
  renderModeTabs as renderModeTabsModule,
  renderRawEditor as renderRawEditorModule,
  renderWizardNav as renderWizardNavModule,
} from "./sites.detail-shell.js";
import { renderListView as renderListViewModule } from "./sites.list-view.js";
import { draftToEnvText, toEnvKey } from "./sites.stable-resources.js";

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

function renderAntibotExclusionRulesEditor(rules, ctx) {
  return renderAntibotExclusionRulesEditorModule(rules, ctx, { escapeHtml, normalizeArray });
}

const renderAuthHelpModalSafe = (ctx) => renderAuthHelpModal(ctx, { escapeHtml });
const renderAntibotHelpModalSafe = (ctx) => renderAntibotHelpModal(ctx, { escapeHtml });
const renderTrafficBadBehaviorHelpModalSafe = (ctx) => renderTrafficBadBehaviorHelpModal(ctx, escapeHtml);
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
    renderTrafficLimitsHelpModal: renderTrafficLimitsHelpModalSafe,
    renderTrafficDnsblHelpModal: renderTrafficDnsblHelpModalSafe,
    renderUpstreamHeadersHelpModal: renderUpstreamHeadersHelpModalSafe,
    renderFrontChapterHelpModal,
    renderUpstreamChapterHelpModal,
    renderHttpChapterHelpModal,
    renderHeadersChapterHelpModal,
    renderBlockingChapterHelpModal,
    renderAntibotChapterHelpModal,
    renderGeoChapterHelpModal,
    renderModsecChapterHelpModal,
    normalizeAuthMode,
    normalizeAuthOrder,
    normalizeAuthExclusionRules,
    normalizeAuthServiceTokens,
  });
}
