import { escapeHtml } from "../ui.js";

function mountWidgetFrame(widget, refs, deps) {
  const { boardNode, pageNode, layout } = refs;
  if (!widget || boardNode.querySelector(`.dashboard-frame[data-widget-id="${widget.id}"]`)) {
    return;
  }
  const frameNode = deps.createFrame(widget, deps.ctx);
  boardNode.appendChild(frameNode);
  deps.wireFrameInteractions(pageNode, boardNode, layout, frameNode, () => {
    refs.layoutDirty = true;
    if (typeof deps.setLayoutDirty === "function") {
      deps.setLayoutDirty();
    }
    deps.persistLayoutNow();
  });
}

function unmountWidgetFrame(widgetID, refs) {
  const frameNode = refs.boardNode.querySelector(`.dashboard-frame[data-widget-id="${widgetID}"]`);
  if (frameNode) {
    frameNode.remove();
  }
}

function renderRequestsWidget(stats, refs, deps) {
  const bodyNode = refs.boardNode.querySelector('[data-widget-body="requests-series"]');
  if (!bodyNode) {
    return;
  }
  const rows = deps.prepareSeriesRows(stats);
  const chartWidth = Math.floor(bodyNode.clientWidth || 1100) - 16;
  bodyNode.innerHTML = deps.renderRequestsSeries(rows, deps.ctx, chartWidth);
  deps.bindRequestsChartHover(bodyNode, rows, deps.ctx);
}

function renderStats(stats, refs, deps) {
  refs.latestStats = stats;
  const rendered = deps.mergeWidgetData(stats, refs.detailModel, refs.latestContainersOverview, deps.ctx);
  deps.WIDGETS.forEach((widget) => {
    const bodyNode = refs.boardNode.querySelector(`[data-widget-body="${widget.id}"]`);
    if (!bodyNode) return;
    if (widget.id === "requests-series") {
      renderRequestsWidget(stats, refs, deps);
    } else {
      const prevContainersScrollTop = widget.id === "containers-health"
        ? (bodyNode.querySelector(".dashboard-containers-list")?.scrollTop || 0)
        : 0;
      bodyNode.innerHTML = rendered[widget.id] || `<div class="dashboard-widget-content waf-empty">${escapeHtml(deps.ctx.t("common.none"))}</div>`;
      if (widget.id === "containers-health") {
        const nextList = bodyNode.querySelector(".dashboard-containers-list");
        if (nextList) {
          nextList.scrollTop = prevContainersScrollTop;
        }
      }
    }
  });

  deps.ensureDetailModel().then((computed) => {
    if (!refs.latestStats || refs.latestStats !== stats) {
      return;
    }
    const rerendered = deps.mergeWidgetData(stats, computed, refs.latestContainersOverview, deps.ctx);
    ["unique-attackers", "top-ips", "top-countries", "containers-health"].forEach((id) => {
      const bodyNode = refs.boardNode.querySelector(`[data-widget-body="${id}"]`);
      if (!bodyNode) return;
      const prevContainersScrollTop = id === "containers-health"
        ? (bodyNode.querySelector(".dashboard-containers-list")?.scrollTop || 0)
        : 0;
      bodyNode.innerHTML = rerendered[id] || `<div class="dashboard-widget-content waf-empty">${escapeHtml(deps.ctx.t("common.none"))}</div>`;
      if (id === "containers-health") {
        const nextList = bodyNode.querySelector(".dashboard-containers-list");
        if (nextList) {
          nextList.scrollTop = prevContainersScrollTop;
        }
      }
    });
  }).catch(() => {});
}

export {
  mountWidgetFrame,
  unmountWidgetFrame,
  renderRequestsWidget,
  renderStats
};
