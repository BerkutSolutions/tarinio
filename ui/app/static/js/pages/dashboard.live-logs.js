import { escapeHtml } from "../ui.js";

function createLiveLogsController(container, modalBodyNode, ctx, fetchContainerLogs) {
  let liveLogsTimer = null;
  let liveLogsState = null;
  let visibilityHandler = null;
  let pollPromise = null;

  const clearTimer = () => {
    if (liveLogsTimer) {
      window.clearInterval(liveLogsTimer);
      liveLogsTimer = null;
    }
  };

  const isVisible = () => typeof document === "undefined" || document.visibilityState !== "hidden";

  const isModalOpen = () => {
    const modalNode = container.querySelector("#dashboard-detail-modal");
    return Boolean(modalNode) && !modalNode.classList.contains("waf-hidden");
  };

  const stop = () => {
    clearTimer();
    if (visibilityHandler && typeof document !== "undefined") {
      document.removeEventListener("visibilitychange", visibilityHandler);
      visibilityHandler = null;
    }
    liveLogsState = null;
    pollPromise = null;
  };

  const renderBody = () => {
    const state = liveLogsState;
    if (!state) {
      return `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.containers.logs.empty"))}</div>`;
    }
    return `
      <div class="dashboard-container-logs-wrap">
        <div class="dashboard-container-logs-meta">
          <span class="badge badge-neutral">${escapeHtml(ctx.t("dashboard.containers.container"))}: ${escapeHtml(state.container)}</span>
          <span class="badge badge-neutral">${escapeHtml(ctx.t("dashboard.containers.logs.lines"))}: ${escapeHtml(String(state.lines.length))}</span>
        </div>
        <pre class="waf-code dashboard-container-logs-pre">${escapeHtml(state.lines.map((line) => `${line.timestamp ? `[${line.timestamp}] ` : ""}${line.message}`).join("\n"))}</pre>
      </div>
    `;
  };

  const updateModalBody = () => {
    if (!modalBodyNode) {
      return;
    }
    const prevPreNode = modalBodyNode.querySelector(".dashboard-container-logs-pre");
    const prevScrollTop = prevPreNode ? prevPreNode.scrollTop : 0;
    const wasPinnedToBottom = prevPreNode
      ? (prevPreNode.scrollHeight - prevPreNode.scrollTop - prevPreNode.clientHeight) < 24
      : true;
    modalBodyNode.innerHTML = renderBody();
    const nextPreNode = modalBodyNode.querySelector(".dashboard-container-logs-pre");
    if (!nextPreNode) {
      return;
    }
    if (wasPinnedToBottom) {
      nextPreNode.scrollTop = nextPreNode.scrollHeight;
      return;
    }
    nextPreNode.scrollTop = prevScrollTop;
  };

  const poll = async () => {
    if (pollPromise) {
      return pollPromise;
    }
    pollPromise = (async () => {
      const state = liveLogsState;
      if (!state || !state.container) {
        return;
      }
      const payload = await fetchContainerLogs(state.container, state.since || "", state.since ? 0 : 1500);
      if (state !== liveLogsState) {
        return;
      }
      const rows = Array.isArray(payload?.lines) ? payload.lines : [];
      if (rows.length) {
        rows.forEach((row) => {
          const key = `${String(row?.timestamp || "")}|${String(row?.message || "")}`;
          if (state.seen?.has(key)) {
            return;
          }
          state.seen?.add(key);
          state.lines.push(row);
        });
        if (state.lines.length > 8000) {
          state.lines = state.lines.slice(state.lines.length - 8000);
          if (state.seen) {
            state.seen = new Set(state.lines.map((line) => `${String(line?.timestamp || "")}|${String(line?.message || "")}`));
          }
        }
        const lastTs = rows[rows.length - 1]?.timestamp;
        if (lastTs) {
          state.since = lastTs;
        }
      }
      updateModalBody();
    })();
    try {
      await pollPromise;
    } finally {
      pollPromise = null;
    }
  };

  const startPolling = () => {
    clearTimer();
    if (!liveLogsState || !isModalOpen() || !isVisible()) {
      return;
    }
    liveLogsTimer = window.setInterval(async () => {
      if (!isModalOpen()) {
        stop();
        return;
      }
      if (!isVisible()) {
        clearTimer();
        return;
      }
      try {
        await poll();
      } catch (_error) {
        // silent live retry
      }
    }, 2000);
  };

  const ensureVisibilityListener = () => {
    if (visibilityHandler || typeof document === "undefined") {
      return;
    }
    visibilityHandler = () => {
      if (!liveLogsState) {
        clearTimer();
        return;
      }
      if (!isVisible()) {
        clearTimer();
        return;
      }
      if (!isModalOpen()) {
        stop();
        return;
      }
      poll().catch(() => {});
      startPolling();
    };
    document.addEventListener("visibilitychange", visibilityHandler);
  };

  const start = async (containerName) => {
    stop();
    liveLogsState = { container: containerName, since: "", lines: [], seen: new Set() };
    ensureVisibilityListener();
    if (isVisible()) {
      await poll();
    }
    startPolling();
  };

  return {
    stop,
    start
  };
}

export {
  createLiveLogsController
};
