import { escapeHtml } from "../ui.js";

function bindWidgetPickerAndDetails(refs, deps) {
  const {
    container,
    pageNode,
    boardNode,
    layout,
    widgetsScopeID,
    visibleWidgetIDs,
    modal,
    liveLogs
  } = refs;
  const widgetsToggleNode = container.querySelector("#dashboard-widgets-toggle");
  const widgetsMenuNode = container.querySelector("#dashboard-widgets-menu");

  const setWidgetsMenuOpen = (open) => {
    if (!widgetsToggleNode || !widgetsMenuNode) {
      return;
    }
    widgetsMenuNode.hidden = !open;
    widgetsToggleNode.setAttribute("aria-expanded", open ? "true" : "false");
  };

  const persistVisibleWidgets = () => {
    deps.saveVisibleWidgetIDs(Array.from(visibleWidgetIDs), widgetsScopeID);
  };

  const renderWidgetsMenu = () => {
    if (!widgetsMenuNode) {
      return;
    }
    widgetsMenuNode.innerHTML = `
      <div class="dashboard-widget-picker-title">${escapeHtml(deps.ctx.t("dashboard.widgets.subtitle"))}</div>
      ${deps.WIDGETS.map((widget) => {
        const checked = visibleWidgetIDs.has(widget.id) ? "checked" : "";
        return `
          <label class="dashboard-widget-picker-row">
            <input type="checkbox" data-widget-visibility-id="${escapeHtml(widget.id)}" ${checked}>
            <span>${escapeHtml(deps.ctx.t(widget.titleKey))}</span>
          </label>
        `;
      }).join("")}
      <div class="dashboard-widget-picker-actions">
        <button type="button" class="btn primary btn-sm" id="dashboard-widgets-save">${escapeHtml(deps.ctx.t("common.save"))}</button>
      </div>
    `;
    widgetsMenuNode.querySelector("#dashboard-widgets-save")?.addEventListener("click", () => {
      persistVisibleWidgets();
      setWidgetsMenuOpen(false);
    });
  };

  widgetsToggleNode?.addEventListener("click", (event) => {
    event.preventDefault();
    const open = widgetsMenuNode?.hidden !== false;
    setWidgetsMenuOpen(open);
  });
  document.addEventListener("click", (event) => {
    if (!widgetsMenuNode || widgetsMenuNode.hidden) {
      return;
    }
    if (widgetsMenuNode.contains(event.target) || widgetsToggleNode?.contains(event.target)) {
      return;
    }
    setWidgetsMenuOpen(false);
  });
  widgetsMenuNode?.addEventListener("change", (event) => {
    const input = event.target?.closest?.("input[data-widget-visibility-id]");
    if (!input) {
      return;
    }
    const widgetID = String(input.dataset.widgetVisibilityId || "");
    if (!widgetID) {
      return;
    }
    if (!input.checked && visibleWidgetIDs.size <= 1) {
      input.checked = true;
      return;
    }
    if (input.checked) {
      visibleWidgetIDs.add(widgetID);
      const widget = deps.WIDGETS.find((item) => item.id === widgetID);
      if (widget) {
        deps.mountWidgetFrame(widget);
      }
      deps.applyAllGeometry(boardNode, layout);
      if (refs.latestStats) {
        deps.renderStats(refs.latestStats);
      }
    } else {
      visibleWidgetIDs.delete(widgetID);
      deps.unmountWidgetFrame(widgetID);
      deps.applyAllGeometry(boardNode, layout);
    }
  });
  renderWidgetsMenu();

  const openDetailFromTarget = async (target) => {
    if (pageNode.dataset.editMode === "1") {
      return;
    }
    const actionNode = target?.closest?.("[data-widget-action]");
    if (!actionNode) {
      return;
    }
    const action = String(actionNode.dataset.widgetAction || "").trim();
    if (!action || action === "requests-series") {
      return;
    }
    if (action !== "container-logs" && !refs.latestStats) {
      return;
    }
    if (action !== "container-logs") {
      liveLogs.stop();
    }
    const payload = {
      ip: String(actionNode.dataset.ip || "").trim(),
      errorCode: String(actionNode.dataset.errorCode || "").trim(),
      countryCode: String(actionNode.dataset.countryCode || "").trim(),
      url: String(actionNode.dataset.url || "").trim(),
      containerName: String(actionNode.dataset.containerName || "").trim()
    };
    if (action === "container-logs") {
      const name = payload.containerName;
      if (!name) {
        return;
      }
      modal.open({ title: `${deps.ctx.t("dashboard.containers.logs.title")} ${name}`, subtitle: deps.ctx.t("dashboard.containers.logs.subtitle"), body: `<div class="waf-empty">${escapeHtml(deps.ctx.t("common.loading"))}</div>` });
      try {
        await liveLogs.start(name);
      } catch (error) {
        modal.open({ title: `${deps.ctx.t("dashboard.containers.logs.title")} ${name}`, subtitle: deps.ctx.t("dashboard.containers.logs.subtitle"), body: `<div class="alert">${escapeHtml(error?.message || deps.ctx.t("dashboard.containers.logs.error"))}</div>` });
        liveLogs.stop();
        return;
      }
      return;
    }
    modal.open({ title: deps.ctx.t("dashboard.detail.loadingTitle"), subtitle: deps.ctx.t("dashboard.detail.loadingSubtitle"), body: `<div class="waf-empty">${escapeHtml(deps.ctx.t("common.loading"))}</div>` });
    const computed = await deps.ensureDetailModel();
    modal.open(deps.buildWidgetDetail(action, payload, refs.latestStats, computed || deps.buildDetailModel(refs.latestStats, [], []), refs.latestContainersOverview, deps.ctx));
  };

  boardNode.addEventListener("click", (event) => {
    openDetailFromTarget(event.target);
  });
  boardNode.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    const actionNode = event.target?.closest?.("[data-widget-action]");
    if (!actionNode) {
      return;
    }
    event.preventDefault();
    openDetailFromTarget(actionNode);
  });

  const modalBodyNode = container.querySelector("#dashboard-detail-content");
  modalBodyNode?.addEventListener("click", (event) => {
    openDetailFromTarget(event.target);
  });
  modalBodyNode?.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    const actionNode = event.target?.closest?.("[data-widget-action]");
    if (!actionNode) {
      return;
    }
    event.preventDefault();
    openDetailFromTarget(actionNode);
  });
}

export {
  bindWidgetPickerAndDetails
};
