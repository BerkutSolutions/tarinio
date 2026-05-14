import { escapeHtml } from "../ui.js";

function createLiveLogsController(container, modalBodyNode, ctx, fetchContainerLogs) {
  let liveLogsTimer = null;
  let liveLogsState = null;

  const stop = () => {
    if (liveLogsTimer) {
      clearInterval(liveLogsTimer);
      liveLogsTimer = null;
    }
    liveLogsState = null;
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
    if (!liveLogsState || !liveLogsState.container) {
      return;
    }
    const payload = await fetchContainerLogs(liveLogsState.container, liveLogsState.since || "", liveLogsState.since ? 0 : 1500);
    const rows = Array.isArray(payload?.lines) ? payload.lines : [];
    if (rows.length) {
      rows.forEach((row) => {
        const key = `${String(row?.timestamp || "")}|${String(row?.message || "")}`;
        if (liveLogsState.seen?.has(key)) {
          return;
        }
        liveLogsState.seen?.add(key);
        liveLogsState.lines.push(row);
      });
      if (liveLogsState.lines.length > 8000) {
        liveLogsState.lines = liveLogsState.lines.slice(liveLogsState.lines.length - 8000);
        if (liveLogsState.seen) {
          liveLogsState.seen = new Set(liveLogsState.lines.map((line) => `${String(line?.timestamp || "")}|${String(line?.message || "")}`));
        }
      }
      const lastTs = rows[rows.length - 1]?.timestamp;
      if (lastTs) {
        liveLogsState.since = lastTs;
      }
    }
    updateModalBody();
  };

  const start = async (containerName) => {
    liveLogsState = { container: containerName, since: "", lines: [], seen: new Set() };
    await poll();
    liveLogsTimer = window.setInterval(async () => {
      const modalNode = container.querySelector("#dashboard-detail-modal");
      if (!modalNode || modalNode.classList.contains("waf-hidden")) {
        stop();
        return;
      }
      try {
        await poll();
      } catch (_error) {
        // silent live retry
      }
    }, 2000);
  };

  return {
    stop,
    start
  };
}

export {
  createLiveLogsController
};
