import { escapeHtml, setError } from "../ui.js";
import { buildDetailModel } from "./dashboard.detail-model.js";
import { normalizeCountryCode } from "./dashboard.detail-model-helpers.js";
import { fetchRequestsRows, fetchEventsRows, fetchContainersOverview, fetchContainerLogs } from "./dashboard.data-fetch.js";
import {
  WIDGETS,
  normalizeLayout,
  loadLayout,
  saveLayout,
  loadVisibleWidgetIDs,
  saveVisibleWidgetIDs,
  formatNumber,
  formatPercent,
  formatBytes,
  getContainerStatusTone,
  formatContainerStatusLabel,
  formatUptimeLocalized
} from "./dashboard.layout-core.js";
import { applyAllGeometry } from "./dashboard.layout-geometry.js";
import { createFrame, wireFrameInteractions } from "./dashboard.frame.js";
import { renderSystemMemory, renderSystemCPU, prepareSeriesRows, renderRequestsSeries, bindRequestsChartHover } from "./dashboard.series.js";
import { renderCountryBadge, mergeWidgetData as mergeWidgetDataT4 } from "./dashboard.widgets.js";
import { buildWidgetDetail as buildWidgetDetailT4 } from "./dashboard.detail-builder.js";
import { createModalState } from "./dashboard.modal.js";
import { createLiveLogsController } from "./dashboard.live-logs.js";
import { mountWidgetFrame, unmountWidgetFrame, renderRequestsWidget, renderStats } from "./dashboard.page-renderers.js";
import { bindWidgetPickerAndDetails } from "./dashboard.page-interactions.js";
import { bindDashboardControls, bindDashboardLifecycle, loadDashboardStats } from "./dashboard.page-lifecycle.js";
import {
  DASHBOARD_CONTRACT_MARKER_CONTAINERS_OVERVIEW,
  DASHBOARD_CONTRACT_MARKER_WIDGET_SERVICES_UP,
  DASHBOARD_CONTRACT_MARKERS_WIDGETS,
  DASHBOARD_CONTRACT_MARKERS_WIDGET_IDS,
  DASHBOARD_CONTRACT_MARKERS_FRAME
} from "./dashboard.contract-markers.js";
const DASHBOARD_CONTRACT_MARKER_STATS = "/api/dashboard/stats";
/* dashboard contract markers:
 "/api/dashboard/containers/overview"
 "dashboard.widget.servicesUp" "dashboard.widget.servicesDown" "dashboard.widget.requestsDay" "dashboard.widget.attacksDay"
 "dashboard.widget.blockedAttacks" "dashboard.widget.uniqueAttackers" "dashboard.widget.requestsSeries"
 "dashboard.widget.popularErrors" "dashboard.widget.topIPs" "dashboard.widget.topCountries" "dashboard.widget.topURLs"
 "dashboard.widget.memory" "dashboard.widget.cpu" "dashboard.widget.containersHealth"
 "services-up" "services-down" "requests-day" "attacks-day" "blocked-attacks" "unique-attackers"
 "requests-series" "popular-errors" "top-ips" "top-countries" "top-urls" "memory" "cpu" "containers-health"
 "frame-resize-handle" "resize-se" "resize-e" "resize-s" "resize-w" "resize-n" "dashboard-frame-header"
*/

function mergeWidgetData(stats, detailModel, containersOverview, ctx) {
  return mergeWidgetDataT4(stats, detailModel, containersOverview, ctx, {
    formatNumber,
    normalizeCountryCode,
    formatPercent,
    formatUptimeLocalized,
    getContainerStatusTone,
    formatContainerStatusLabel,
    renderSystemMemory,
    renderSystemCPU
  });
}

function buildWidgetDetail(action, payload, stats, detailModel, containersOverview, ctx) {
  const extracted = buildWidgetDetailT4(action, payload, stats, detailModel, containersOverview, ctx, {
    formatNumber,
    formatPercent,
    formatBytes,
    normalizeCountryCode,
    renderCountryBadge: (code) => renderCountryBadge(code, { normalizeCountryCode }),
    formatUptimeLocalized,
    formatContainerStatusLabel,
    getContainerStatusTone
  });
  return extracted;
}

export async function renderDashboard(container, ctx) {
  const titleNode = document.getElementById("page-title");
  const descNode = document.getElementById("page-desc");
  if (titleNode) titleNode.textContent = "";
  if (descNode) descNode.textContent = "";

  const layout = loadLayout();
  container.innerHTML = `
    <div id="dashboard-page">
      <div class="dashboard-toolbar">
        <div class="dashboard-actions">
          <button class="btn ghost btn-sm" id="dashboard-layout-reset" type="button">${escapeHtml(ctx.t("dashboard.action.resetLayout"))}</button>
          <button class="btn ghost btn-sm" id="dashboard-edit-toggle" type="button">${escapeHtml(ctx.t("dashboard.action.editLayout"))}</button>
          <div class="dashboard-widget-picker">
            <button class="btn ghost btn-sm" id="dashboard-widgets-toggle" type="button" aria-expanded="false">${escapeHtml(ctx.t("dashboard.action.widgets"))}</button>
            <div class="dashboard-widget-picker-menu" id="dashboard-widgets-menu" hidden></div>
          </div>
        </div>
      </div>
      <div class="dashboard-board" id="dashboard-board"></div>
      <div class="waf-modal waf-hidden" id="dashboard-detail-modal" role="dialog" aria-modal="true" aria-labelledby="dashboard-detail-title" tabindex="-1">
        <button class="waf-modal-overlay" type="button" data-dashboard-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
        <div class="waf-modal-card">
          <div class="waf-card-head">
            <div>
              <h3 id="dashboard-detail-title">${escapeHtml(ctx.t("dashboard.detail.title"))}</h3>
              <div class="muted" id="dashboard-detail-subtitle">${escapeHtml(ctx.t("dashboard.detail.subtitle"))}</div>
            </div>
            <button class="btn ghost btn-sm" type="button" data-dashboard-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
          </div>
          <div class="waf-card-body" id="dashboard-detail-content"><div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div></div>
        </div>
      </div>
    </div>
  `;

  const pageNode = container.querySelector("#dashboard-page");
  const boardNode = container.querySelector("#dashboard-board");
  pageNode.dataset.editMode = "0";

  const pageState = {
    latestStats: null,
    latestContainersOverview: null,
    detailModel: null,
    detailModelGeneratedAt: "",
    detailModelPromise: null
  };
  let layoutDirty = false;
  let requestsChartRenderRAF = 0;
  let containersOverviewFailCount = 0;
  let containersOverviewNextRetryAt = 0;
  const widgetsScopeID = String(ctx?.currentUser?.username || ctx?.currentUser?.id || "").trim().toLowerCase();
  const visibleWidgetIDs = new Set(loadVisibleWidgetIDs(widgetsScopeID));

  const persistLayoutNow = () => {
    if (!layoutDirty) {
      return;
    }
    saveLayout(layout);
    layoutDirty = false;
  };

  const ensureDetailModel = async () => {
    if (!pageState.latestStats) {
      return null;
    }
    const generatedAt = String(pageState.latestStats?.generated_at || "");
    if (pageState.detailModel && pageState.detailModelGeneratedAt === generatedAt) {
      return pageState.detailModel;
    }
    if (pageState.detailModelPromise) {
      return pageState.detailModelPromise;
    }
    pageState.detailModelPromise = Promise.all([fetchRequestsRows(), fetchEventsRows()])
      .then(([requestsRows, eventsRows]) => {
        pageState.detailModel = buildDetailModel(pageState.latestStats, requestsRows, eventsRows);
        pageState.detailModelGeneratedAt = generatedAt;
        return pageState.detailModel;
      })
      .catch(() => {
        pageState.detailModel = buildDetailModel(pageState.latestStats, [], []);
        pageState.detailModelGeneratedAt = generatedAt;
        return pageState.detailModel;
      })
      .finally(() => {
        pageState.detailModelPromise = null;
      });
    return pageState.detailModelPromise;
  };

  const modal = createModalState(container, ctx);
  const modalBodyNode = container.querySelector("#dashboard-detail-content");
  const liveLogs = createLiveLogsController(container, modalBodyNode, ctx, fetchContainerLogs);

  container.querySelectorAll("[data-dashboard-detail-close='true']").forEach((node) => {
    node.addEventListener("click", liveLogs.stop);
  });
  container.querySelector("#dashboard-detail-modal")?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      liveLogs.stop();
    }
  });

  const refs = {
    container,
    pageNode,
    boardNode,
    layout,
    widgetsScopeID,
    visibleWidgetIDs,
    modal,
    liveLogs,
    layoutDirty: false,
    latestStats: pageState.latestStats,
    latestContainersOverview: pageState.latestContainersOverview,
    detailModel: pageState.detailModel
  };
  const mountFrame = (widget) => mountWidgetFrame(widget, refs, {
    boardNode,
    pageNode,
    layout,
    ctx,
    createFrame,
    wireFrameInteractions,
    persistLayoutNow,
    setLayoutDirty: () => {
      layoutDirty = true;
    }
  });
  const unmountFrame = (widgetID) => unmountWidgetFrame(widgetID, refs);
  const renderRequests = (stats) => renderRequestsWidget(stats, refs, { boardNode, prepareSeriesRows, renderRequestsSeries, bindRequestsChartHover, ctx });
  const renderAllStats = (stats) => {
    refs.detailModel = pageState.detailModel;
    refs.latestContainersOverview = pageState.latestContainersOverview;
    renderStats(stats, refs, {
      boardNode,
      WIDGETS,
      ctx,
      mergeWidgetData,
      ensureDetailModel,
      prepareSeriesRows,
      renderRequestsSeries,
      bindRequestsChartHover
    });
    pageState.latestStats = refs.latestStats;
  };

  WIDGETS.filter((widget) => visibleWidgetIDs.has(widget.id)).forEach((widget) => {
    mountFrame(widget);
  });
  applyAllGeometry(boardNode, layout);

  bindWidgetPickerAndDetails(refs, {
    ctx,
    WIDGETS,
    saveVisibleWidgetIDs,
    applyAllGeometry,
    mountWidgetFrame: mountFrame,
    unmountWidgetFrame: unmountFrame,
    renderStats: renderAllStats,
    ensureDetailModel,
    buildWidgetDetail,
    buildDetailModel
  });

  const load = (silent = false) => loadDashboardStats(silent, {
    ctx,
    boardNode,
    layout,
    pageState,
    refs,
    fetchContainersOverview,
    renderAllStats,
    applyAllGeometry,
    setError,
    containersOverviewFailCountRef: {
      get value() {
        return containersOverviewFailCount;
      },
      set value(next) {
        containersOverviewFailCount = next;
      }
    },
    containersOverviewNextRetryAtRef: {
      get value() {
        return containersOverviewNextRetryAt;
      },
      set value(next) {
        containersOverviewNextRetryAt = next;
      }
    }
  });

  bindDashboardControls({
    container,
    pageNode,
    pageState,
    ctx,
    layout,
    normalizeLayout,
    saveLayout,
    applyAllGeometry,
    boardNode,
    renderAllStats,
    persistLayoutNow,
    setLayoutDirty: (next) => {
      layoutDirty = !!next;
    }
  });

  bindDashboardLifecycle({
    container,
    pageNode,
    pageState,
    boardNode,
    layout,
    applyAllGeometry,
    renderRequests,
    persistLayoutNow,
    requestsChartRenderRAFRef: {
      get value() {
        return requestsChartRenderRAF;
      },
      set value(next) {
        requestsChartRenderRAF = next;
      }
    },
    loadDashboardStats: load
  });

  await load(false);
  return () => {
    liveLogs.stop();
    persistLayoutNow();
    if (container.__dashboardAutoRefreshTimer) {
      clearInterval(container.__dashboardAutoRefreshTimer);
      container.__dashboardAutoRefreshTimer = null;
    }
    if (container.__dashboardResizeHandler) {
      window.removeEventListener("resize", container.__dashboardResizeHandler);
      container.__dashboardResizeHandler = null;
    }
    if (container.__dashboardPageHideHandler) {
      window.removeEventListener("pagehide", container.__dashboardPageHideHandler);
      container.__dashboardPageHideHandler = null;
    }
    if (container.__dashboardRequestsResizeObserver) {
      container.__dashboardRequestsResizeObserver.disconnect();
      container.__dashboardRequestsResizeObserver = null;
    }
    if (requestsChartRenderRAF) {
      window.cancelAnimationFrame(requestsChartRenderRAF);
      requestsChartRenderRAF = 0;
    }
  };
}

