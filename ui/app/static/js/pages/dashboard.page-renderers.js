import { escapeHtml } from "../ui.js";

function captureScrollState(bodyNode) {
  const selectors = [".dashboard-scroll-area", ".dashboard-containers-list"];
  return selectors.flatMap((selector) => Array.from(bodyNode.querySelectorAll(selector)).map((node, index) => ({
    selector,
    index,
    top: node.scrollTop || 0,
    left: node.scrollLeft || 0
  })));
}

function restoreScrollState(bodyNode, items) {
  items.forEach((item) => {
    const nodes = bodyNode.querySelectorAll(item.selector);
    const node = nodes[item.index] || null;
    if (!node) {
      return;
    }
    node.scrollTop = item.top;
    node.scrollLeft = item.left;
  });
}

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
      const scrollState = captureScrollState(bodyNode);
      bodyNode.innerHTML = rendered[widget.id] || `<div class="dashboard-widget-content waf-empty">${escapeHtml(deps.ctx.t("common.none"))}</div>`;
      restoreScrollState(bodyNode, scrollState);
    }
  });
}

export {
  mountWidgetFrame,
  unmountWidgetFrame,
  renderRequestsWidget,
  renderStats
};
