import { confirmAction, escapeHtml, formatDate, setError, setLoading } from "../ui.js";
import {
  asDate,
  buildHostSiteMap,
  buildIPSetBySite,
  buildPageButtons,
  buildSiteAliasMap,
  clearRateLimitCookies,
  formatRemaining,
  mergeByID,
  normalizeIP,
  normalizeList,
  resolveCanonicalSiteID,
  tryGetJSON,
  unwrapList
} from "./bans.helpers.base.js";
import {
  buildManualBanRows,
  clearDismissedBanRow,
  dismissBanRow,
  isDismissedBanRow,
  isStartupSelfTestSite,
  loadManualBanTimers,
  manualBanTimerRowKey,
  mergeBanRows,
  processExpiredManualBanTimers,
  removeManualBanTimer,
  saveManualBanTimers,
  upsertManualBanTimer
} from "./bans.helpers.timers.js";
import {
  buildAutoBanRows,
  buildLatestUnbanMarkers,
  countryFlagEmoji,
  parsePositiveNumber,
  renderModules,
  renderReasonList
} from "./bans.helpers.events.js";
import {
  createModalControllers,
  createPaginationMetaGetter,
  loadSiteBanDurations,
  renderCreateSiteOptions,
  renderDetail
} from "./bans.page-helpers.js";
import { createRenderRows } from "./bans.page-rows.js";
import { bindBansPageActions } from "./bans.page-actions.js";

const ICON_PLUS = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v14h-2zM5 11h14v2H5z"/></svg>';
const ICON_UNLOCK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v2h-2V7a3 3 0 1 0-6 0v2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-9a2 2 0 0 1 2-2h2V7a5 5 0 0 1 5-5Z"/></svg>';
const ALL_SERVICES_SITE_ID = "__all__";
const EXTEND_DURATIONS = [
  { key: "1h", seconds: 60 * 60, labelKey: "bans.extend.duration.1h" },
  { key: "3h", seconds: 3 * 60 * 60, labelKey: "bans.extend.duration.3h" },
  { key: "6h", seconds: 6 * 60 * 60, labelKey: "bans.extend.duration.6h" },
  { key: "12h", seconds: 12 * 60 * 60, labelKey: "bans.extend.duration.12h" },
  { key: "1d", seconds: 24 * 60 * 60, labelKey: "bans.extend.duration.1d" },
  { key: "1w", seconds: 7 * 24 * 60 * 60, labelKey: "bans.extend.duration.1w" },
  { key: "1mo", seconds: 30 * 24 * 60 * 60, labelKey: "bans.extend.duration.1mo" },
  { key: "forever", seconds: 0, labelKey: "bans.extend.duration.forever" }
];

export async function renderBans(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.bans"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("bans.subtitle"))}</div>
        </div>
        <div class="waf-actions">
          <button class="btn" id="bans-create" type="button">${escapeHtml(ctx.t("bans.action.ban"))}</button>
          <button class="btn ghost btn-sm" id="bans-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
      </div>
      <div class="waf-card-body waf-stack">
        <div id="bans-status"></div>
        <div id="bans-list"></div>
      </div>
    </section>
    <div class="waf-modal waf-hidden" id="bans-create-modal" role="dialog" aria-modal="true" aria-labelledby="bans-create-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-create-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-create-title">${escapeHtml(ctx.t("bans.create.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.create.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-create-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-create-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-create-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <select id="bans-create-site"></select>
            </div>
            <div class="waf-field">
              <label for="bans-create-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-create-ip" type="text" placeholder="203.0.113.10">
            </div>
            <div class="waf-field">
              <label for="bans-create-duration">${escapeHtml(ctx.t("bans.create.duration"))}</label>
              <select id="bans-create-duration">
                ${EXTEND_DURATIONS.map((item) => `<option value="${escapeHtml(String(item.seconds))}">${escapeHtml(ctx.t(item.labelKey))}</option>`).join("")}
              </select>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn" id="bans-create-submit" type="button">${escapeHtml(ctx.t("bans.action.ban"))}</button>
            <button class="btn ghost" type="button" data-bans-create-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-detail-modal" role="dialog" aria-modal="true" aria-labelledby="bans-detail-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-detail-title">${escapeHtml(ctx.t("events.detail.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("events.detail.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body" id="bans-detail-content">
          <div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-extend-modal" role="dialog" aria-modal="true" aria-labelledby="bans-extend-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-extend-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-extend-title">${escapeHtml(ctx.t("bans.extend.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.extend.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-extend-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-extend-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-extend-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <input id="bans-extend-site" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-extend-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-extend-ip" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-extend-duration">${escapeHtml(ctx.t("bans.extend.duration"))}</label>
              <select id="bans-extend-duration">
                ${EXTEND_DURATIONS.map((item) => `<option value="${escapeHtml(String(item.seconds))}">${escapeHtml(ctx.t(item.labelKey))}</option>`).join("")}
              </select>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn success" id="bans-extend-submit" type="button">${escapeHtml(ctx.t("bans.action.extend"))}</button>
            <button class="btn ghost" type="button" data-bans-extend-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-unban-modal" role="dialog" aria-modal="true" aria-labelledby="bans-unban-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-unban-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-unban-title">${escapeHtml(ctx.t("bans.unban.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.unban.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-unban-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-unban-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-unban-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <input id="bans-unban-site" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-unban-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-unban-ip" type="text" readonly>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn danger" id="bans-unban-submit" type="button">${escapeHtml(ctx.t("bans.action.unban"))}</button>
            <button class="btn ghost" type="button" data-bans-unban-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
  `;

  const statusNode = container.querySelector("#bans-status");
  const listNode = container.querySelector("#bans-list");
  const createModalNode = container.querySelector("#bans-create-modal");
  const createStatusNode = container.querySelector("#bans-create-status");
  const createSiteNode = container.querySelector("#bans-create-site");
  const createIPNode = container.querySelector("#bans-create-ip");
  const createDurationNode = container.querySelector("#bans-create-duration");
  const createSubmitNode = container.querySelector("#bans-create-submit");
  const detailModalNode = container.querySelector("#bans-detail-modal");
  const detailContentNode = container.querySelector("#bans-detail-content");
  const extendModalNode = container.querySelector("#bans-extend-modal");
  const extendStatusNode = container.querySelector("#bans-extend-status");
  const extendSiteNode = container.querySelector("#bans-extend-site");
  const extendIPNode = container.querySelector("#bans-extend-ip");
  const extendDurationNode = container.querySelector("#bans-extend-duration");
  const extendSubmitNode = container.querySelector("#bans-extend-submit");
  const unbanModalNode = container.querySelector("#bans-unban-modal");
  const unbanStatusNode = container.querySelector("#bans-unban-status");
  const unbanSiteNode = container.querySelector("#bans-unban-site");
  const unbanIPNode = container.querySelector("#bans-unban-ip");
  const unbanSubmitNode = container.querySelector("#bans-unban-submit");
  const latestSiteIDsRef = { current: [] };
  const pagingState = {
    pageSize: 10,
    page: 1
  };
  const getPaginationMeta = createPaginationMetaGetter(pagingState);
  const modalControllers = createModalControllers({
    detailModalNode,
    createModalNode,
    createStatusNode,
    createSiteNode,
    createDurationNode,
    createIPNode,
    extendModalNode,
    extendStatusNode,
    extendSiteNode,
    extendIPNode,
    extendDurationNode,
    unbanModalNode,
    unbanStatusNode,
    unbanSiteNode,
    unbanIPNode,
    unbanSubmitNode,
    latestSiteIDsRef,
    extendDurations: EXTEND_DURATIONS
  });

  const renderRows = createRenderRows({
    ctx,
    statusNode,
    listNode,
    pagingState,
    getPaginationMeta,
    renderCreateSiteOptions: (sites) => renderCreateSiteOptions({
      createSiteNode,
      sites,
      isStartupSelfTestSite,
      latestSiteIDsRef,
      t: ctx.t,
      allServicesSiteID: ALL_SERVICES_SITE_ID
    }),
    loadSiteBanDurations: (sites, resolveSiteID) => loadSiteBanDurations({
      sites,
      resolveSiteID,
      api: ctx.api
    }),
    renderDetail: (row, siteLabel) => renderDetail({
      detailContentNode,
      detailModalNode,
      row,
      siteLabel,
      t: ctx.t,
      formatRemaining
    }),
    openExtendModal: modalControllers.openExtendModal,
    resolveCanonicalSiteID,
    tryGetJSON,
    confirmAction,
    iconPlus: ICON_PLUS,
    iconUnlock: ICON_UNLOCK
  });
  bindBansPageActions({
    container,
    ctx,
    allServicesSiteID: ALL_SERVICES_SITE_ID,
    latestSiteIDsRef,
    createSiteNode,
    createIPNode,
    createDurationNode,
    createStatusNode,
    createSubmitNode,
    extendDurationNode,
    extendStatusNode,
    extendSubmitNode,
    unbanStatusNode,
    unbanSubmitNode,
    extendDurations: EXTEND_DURATIONS,
    modalControllers,
    renderRows
  });

  await renderRows();
}
