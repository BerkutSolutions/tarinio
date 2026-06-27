import { escapeHtml } from "../ui.js";
import { normalizeArray } from "./sites.routing-merge.js";

export function renderModeTabs(activeMode, ctx) {
  const mode = String(activeMode || "easy").trim().toLowerCase() === "raw" ? "raw" : "easy";
  return `
    <div class="waf-mode-switch">
      <button class="btn ${mode === "easy" ? "primary" : "ghost"} btn-sm" type="button" data-mode-tab="easy">${escapeHtml(ctx.t("sites.mode.easy"))}</button>
      <button class="btn ghost btn-sm" type="button" disabled>${escapeHtml(ctx.t("sites.mode.advanced"))}</button>
      <button class="btn ${mode === "raw" ? "primary" : "ghost"} btn-sm" type="button" data-mode-tab="raw">${escapeHtml(ctx.t("sites.mode.raw"))}</button>
    </div>
    ${mode === "raw" ? "" : `<div class="waf-note">${escapeHtml(ctx.t("sites.mode.note"))}</div>`}
  `;
}

export function renderWizardNav(activeTab, ctx) {
  const items = [
    { id: "front", title: ctx.t("sites.wizard.front.title"), subtitle: ctx.t("sites.wizard.front.subtitle") },
    { id: "upstream", title: ctx.t("sites.wizard.upstream.title"), subtitle: ctx.t("sites.wizard.upstream.subtitle") },
    { id: "http", title: ctx.t("sites.easy.tab.http.title"), subtitle: ctx.t("sites.easy.tab.http.subtitle") },
    { id: "headers", title: ctx.t("sites.easy.tab.headers.title"), subtitle: ctx.t("sites.easy.tab.headers.subtitle") },
    { id: "traffic", title: ctx.t("sites.easy.tab.traffic.title"), subtitle: ctx.t("sites.easy.tab.traffic.subtitle") },
    { id: "blocking", title: ctx.t("sites.easy.tab.blocking.title"), subtitle: ctx.t("sites.easy.tab.blocking.subtitle") },
    { id: "antibot", title: ctx.t("sites.easy.tab.antibot.title"), subtitle: ctx.t("sites.easy.tab.antibot.subtitle") },
    { id: "geo", title: ctx.t("sites.easy.tab.geo.title"), subtitle: ctx.t("sites.easy.tab.geo.subtitle") },
    { id: "modsec", title: ctx.t("sites.easy.tab.modsec.title"), subtitle: ctx.t("sites.easy.tab.modsec.subtitle") },
    { id: "websocket", title: ctx.t("sites.easy.tab.websocket.title"), subtitle: ctx.t("sites.easy.tab.websocket.subtitle") },
    { id: "virtualpatches", title: ctx.t("sites.easy.tab.virtualpatches.title"), subtitle: ctx.t("sites.easy.tab.virtualpatches.subtitle") }
  ];
  return `
    <aside class="waf-service-wizard-nav" role="tablist" aria-label="${escapeHtml(ctx.t("sites.wizard.aria"))}">
      ${items.map((item, index) => `
        <button
          class="waf-service-wizard-item${activeTab === item.id ? " is-active" : ""}"
          type="button"
          role="tab"
          aria-selected="${activeTab === item.id ? "true" : "false"}"
          data-wizard-tab="${item.id}">
          <div class="waf-service-step-index">${index + 1}</div>
          <div class="waf-service-wizard-copy">
            <div class="waf-service-wizard-title">${escapeHtml(item.title)}</div>
            <div class="waf-service-wizard-subtitle">${escapeHtml(item.subtitle)}</div>
          </div>
        </button>
      `).join("")}
    </aside>
  `;
}

export function renderRawEditor(state, ctx, isNew, toEnvKey, draftToEnvText) {
  const missingFields = normalizeArray(state.rawMissingFields);
  return `
    <section class="waf-card waf-service-editor-card">
      <div class="waf-card-body waf-stack">
        <div id="sites-feedback"></div>
        <form id="service-editor-form" class="waf-form waf-stack">
          <div class="waf-list-title">${escapeHtml(ctx.t("sites.raw.title"))}</div>
          <div class="waf-note">${escapeHtml(ctx.t("sites.raw.description"))}</div>
          ${missingFields.length ? `
            <div class="waf-empty">
              <strong>${escapeHtml(ctx.t("sites.raw.missingFieldsTitle"))}</strong>
              <pre class="waf-code">${escapeHtml(missingFields.map((field) => toEnvKey(field)).join("\n"))}</pre>
            </div>
          ` : ""}
          <div class="waf-field full">
            <label for="service-raw-env">${escapeHtml(ctx.t("sites.raw.label"))}</label>
            <textarea id="service-raw-env" rows="32" class="waf-code">${escapeHtml(state.rawEnvText || draftToEnvText(state.draft))}</textarea>
          </div>
          <div class="waf-actions waf-actions-between">
            <button class="btn ghost btn-sm" type="button" id="service-back-bottom">${escapeHtml(ctx.t("common.back"))}</button>
            <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t(isNew ? "sites.action.createSite" : "sites.action.saveSite"))}</button>
          </div>
        </form>
      </div>
    </section>
  `;
}
