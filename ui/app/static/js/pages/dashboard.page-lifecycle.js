function loadDashboardStats(silent, deps) {
  return deps.ctx.api.get("/api/dashboard/stats")
    .then(async (stats) => {
      if (Date.now() >= deps.containersOverviewNextRetryAtRef.value) {
        const containersOverview = await deps.fetchContainersOverview();
        if (containersOverview) {
          deps.pageState.latestContainersOverview = containersOverview;
          deps.refs.latestContainersOverview = containersOverview;
          deps.containersOverviewFailCountRef.value = 0;
          deps.containersOverviewNextRetryAtRef.value = 0;
        } else {
          deps.pageState.latestContainersOverview = null;
          deps.refs.latestContainersOverview = null;
          deps.containersOverviewFailCountRef.value += 1;
          const delay = Math.min(120000, 5000 * (2 ** Math.max(0, deps.containersOverviewFailCountRef.value - 1)));
          deps.containersOverviewNextRetryAtRef.value = Date.now() + delay;
        }
      }
      deps.renderAllStats(stats);
      if (!silent) {
        deps.applyAllGeometry(deps.boardNode, deps.layout);
      }
    })
    .catch((error) => {
      if (!silent) {
        deps.setError(deps.boardNode, error?.message || deps.ctx.t("dashboard.error"));
      }
    });
}

function bindDashboardControls(deps) {
  deps.container.querySelector("#dashboard-edit-toggle")?.addEventListener("click", () => {
    const wasEdit = deps.pageNode.dataset.editMode === "1";
    deps.pageNode.dataset.editMode = wasEdit ? "0" : "1";
    deps.container.querySelector("#dashboard-edit-toggle").textContent = deps.pageNode.dataset.editMode === "1" ? deps.ctx.t("dashboard.action.doneEdit") : deps.ctx.t("dashboard.action.editLayout");
    deps.pageNode.classList.toggle("dashboard-edit-mode", deps.pageNode.dataset.editMode === "1");
    if (wasEdit) {
      deps.persistLayoutNow();
    }
  });

  deps.container.querySelector("#dashboard-layout-reset")?.addEventListener("click", () => {
    const defaults = deps.normalizeLayout([]);
    deps.layout.splice(0, deps.layout.length, ...defaults);
    deps.saveLayout(deps.layout);
    deps.setLayoutDirty(false);
    deps.applyAllGeometry(deps.boardNode, deps.layout);
    if (deps.pageState.latestStats) {
      deps.renderAllStats(deps.pageState.latestStats);
    }
  });
}

function bindDashboardLifecycle(deps) {
  if (deps.container.__dashboardResizeHandler) {
    window.removeEventListener("resize", deps.container.__dashboardResizeHandler);
  }
  deps.container.__dashboardResizeHandler = () => {
    deps.applyAllGeometry(deps.boardNode, deps.layout);
    if (deps.pageState.latestStats) {
      deps.renderRequests(deps.pageState.latestStats);
    }
  };
  window.addEventListener("resize", deps.container.__dashboardResizeHandler);

  if (deps.container.__dashboardPageHideHandler) {
    window.removeEventListener("pagehide", deps.container.__dashboardPageHideHandler);
  }
  deps.container.__dashboardPageHideHandler = () => {
    deps.persistLayoutNow();
  };
  window.addEventListener("pagehide", deps.container.__dashboardPageHideHandler);

  if (deps.container.__dashboardRequestsResizeObserver) {
    deps.container.__dashboardRequestsResizeObserver.disconnect();
  }
  if (typeof ResizeObserver !== "undefined") {
    const requestsBody = deps.boardNode.querySelector('[data-widget-body="requests-series"]');
    if (requestsBody) {
      deps.container.__dashboardRequestsResizeObserver = new ResizeObserver(() => {
        if (!deps.pageState.latestStats) {
          return;
        }
        if (deps.requestsChartRenderRAFRef.value) {
          window.cancelAnimationFrame(deps.requestsChartRenderRAFRef.value);
        }
        deps.requestsChartRenderRAFRef.value = window.requestAnimationFrame(() => {
          deps.renderRequests(deps.pageState.latestStats);
          deps.requestsChartRenderRAFRef.value = 0;
        });
      });
      deps.container.__dashboardRequestsResizeObserver.observe(requestsBody);
    }
  }

  if (deps.container.__dashboardAutoRefreshTimer) {
    clearInterval(deps.container.__dashboardAutoRefreshTimer);
  }
  deps.container.__dashboardAutoRefreshTimer = window.setInterval(() => {
    if (!document.body.contains(deps.pageNode)) {
      clearInterval(deps.container.__dashboardAutoRefreshTimer);
      deps.container.__dashboardAutoRefreshTimer = null;
      return;
    }
    deps.loadDashboardStats(true);
  }, 5000);
}

export {
  bindDashboardControls,
  bindDashboardLifecycle,
  loadDashboardStats
};
