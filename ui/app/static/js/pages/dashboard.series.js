import { escapeHtml } from "../ui.js";
import { clamp, formatNumber, formatPercent, formatBytes } from "./dashboard.layout-core.js";

function renderSystemMemory(stats, ctx) {
  const system = stats?.system || {};
  const usedPercent = Number(system.memory_used_percent || 0);
  return `
    <button type="button" class="dashboard-widget-content dashboard-system dashboard-system-clickable" data-widget-action="memory">
      <div class="dashboard-system-main">${escapeHtml(formatPercent(usedPercent))}</div>
      <div class="dashboard-progress"><span style="width:${escapeHtml(String(clamp(usedPercent, 0, 100)))}%"></span></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.memoryUsed"))}</span><strong>${escapeHtml(formatBytes(system.memory_used_bytes))}</strong></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.memoryFree"))}</span><strong>${escapeHtml(formatBytes(system.memory_free_bytes))}</strong></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.memoryTotal"))}</span><strong>${escapeHtml(formatBytes(system.memory_total_bytes))}</strong></div>
    </button>
  `;
}

function renderSystemCPU(stats, ctx) {
  const system = stats?.system || {};
  const load = clamp(Number(system.cpu_load_percent || 0), 0, 100);
  return `
    <button type="button" class="dashboard-widget-content dashboard-system dashboard-system-clickable" data-widget-action="cpu">
      <div class="dashboard-system-main">${escapeHtml(formatPercent(load))}</div>
      <div class="dashboard-progress"><span style="width:${escapeHtml(String(load))}%"></span></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.cpuCores"))}</span><strong>${escapeHtml(formatNumber(system.cpu_cores || 0))}</strong></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.goroutines"))}</span><strong>${escapeHtml(formatNumber(system.goroutines || 0))}</strong></div>
      <div class="dashboard-system-row"><span>${escapeHtml(ctx.t("dashboard.system.heap"))}</span><strong>${escapeHtml(formatNumber(system.control_plane_heap_mb || 0))} MB</strong></div>
    </button>
  `;
}

function prepareSeriesRows(stats) {
  const requests = Array.isArray(stats?.requests_series) ? stats.requests_series : [];
  const blockedRaw = Array.isArray(stats?.blocked_series) ? stats.blocked_series : [];
  const blockedMap = new Map(blockedRaw.map((item) => [String(item?.timestamp || ""), Number(item?.count || 0)]));
  const hourFormatter = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", hour12: false });
  return requests.map((item) => ({
    label: (() => {
      const ts = String(item?.timestamp || "");
      if (!ts) {
        return String(item?.label || "");
      }
      const dt = new Date(ts);
      return Number.isNaN(dt.getTime()) ? String(item?.label || "") : hourFormatter.format(dt);
    })(),
    requests: Number(item?.count || 0),
    blocked: blockedMap.get(String(item?.timestamp || "")) || 0
  }));
}

function renderRequestsSeries(rows, ctx, chartWidthPx) {
  if (!rows.length) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.series.empty"))}</div>`;
  }
  const maxY = Math.max(1, ...rows.map((row) => Math.max(row.requests, row.blocked)));
  const width = Math.max(320, Math.floor(chartWidthPx || 1100));
  const height = 260;
  const pad = { left: 56, right: 20, top: 16, bottom: 36 };
  const chartWidth = width - pad.left - pad.right;
  const chartHeight = height - pad.top - pad.bottom;

  const pointsReq = rows.map((row, index) => {
    const x = pad.left + (index / Math.max(rows.length - 1, 1)) * chartWidth;
    const y = pad.top + (1 - (row.requests / maxY)) * chartHeight;
    return { x, y, ...row };
  });
  const pointsBlocked = rows.map((row, index) => {
    const x = pad.left + (index / Math.max(rows.length - 1, 1)) * chartWidth;
    const y = pad.top + (1 - (row.blocked / maxY)) * chartHeight;
    return { x, y, ...row };
  });
  const reqPolyline = pointsReq.map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(" ");
  const blockedPolyline = pointsBlocked.map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(" ");
  const yTicks = 5;
  const yStep = Math.max(1, Math.ceil(maxY / yTicks));
  const yValues = Array.from({ length: yTicks + 1 }, (_, i) => i * yStep);
  const xLabelStep = Math.max(1, Math.ceil(rows.length / 8));

  return `
    <div class="dashboard-widget-content dashboard-series" data-requests-chart="true">
      <div class="dashboard-series-canvas">
        <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" class="dashboard-series-svg">
          ${yValues.map((value) => {
            const y = pad.top + (1 - (value / Math.max(yValues[yValues.length - 1], 1))) * chartHeight;
            return `
              <line x1="${pad.left}" y1="${y}" x2="${width - pad.right}" y2="${y}" stroke="rgba(255,255,255,0.08)" stroke-width="1"></line>
              <text x="${pad.left - 8}" y="${y + 4}" text-anchor="end" font-size="12" fill="rgba(255,255,255,0.7)">${value}</text>
            `;
          }).join("")}
          <polyline points="${blockedPolyline}" fill="none" stroke="#ff5d6e" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></polyline>
          <polyline points="${reqPolyline}" fill="none" stroke="#58a6ff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></polyline>
          <rect x="${pad.left}" y="${pad.top}" width="${chartWidth}" height="${chartHeight}" fill="transparent" data-chart-overlay="true"></rect>
        </svg>
        <div class="dashboard-chart-tooltip" data-chart-tooltip="true" hidden></div>
      </div>
      <div class="dashboard-series-axis">
        ${rows.map((row, index) => (index % xLabelStep === 0 ? `<span>${escapeHtml(row.label)}</span>` : "<span></span>")).join("")}
      </div>
      <div class="dashboard-series-legend">
        <span><i class="dot dot-requests"></i>${escapeHtml(ctx.t("dashboard.series.requests"))}</span>
        <span><i class="dot dot-blocked"></i>${escapeHtml(ctx.t("dashboard.series.blocked"))}</span>
      </div>
    </div>
  `;
}

function bindRequestsChartHover(bodyNode, rows, ctx) {
  const overlay = bodyNode.querySelector("[data-chart-overlay='true']");
  const tooltip = bodyNode.querySelector("[data-chart-tooltip='true']");
  const svg = bodyNode.querySelector(".dashboard-series-svg");
  if (!overlay || !tooltip || !svg || !rows.length) {
    return;
  }
  const viewBox = String(svg.getAttribute("viewBox") || "").split(/\s+/).map((part) => Number(part));
  const width = Number.isFinite(viewBox[2]) ? viewBox[2] : 1100;
  const padLeft = 56;
  const padRight = 20;
  const chartWidth = width - padLeft - padRight;
  const nearest = (x) => clamp(Math.round(((x - padLeft) / chartWidth) * (rows.length - 1)), 0, rows.length - 1);
  const show = (event) => {
    const rect = svg.getBoundingClientRect();
    const localX = ((event.clientX - rect.left) / Math.max(rect.width, 1)) * width;
    const row = rows[nearest(localX)];
    tooltip.textContent = `${row.label}: ${ctx.t("dashboard.series.requests")} ${formatNumber(row.requests)}, ${ctx.t("dashboard.series.blocked")} ${formatNumber(row.blocked)}`;
    tooltip.hidden = false;
    const left = clamp(event.clientX - rect.left + 12, 8, Math.max(8, rect.width - (tooltip.offsetWidth || 180) - 8));
    const top = clamp(event.clientY - rect.top - 36, 8, Math.max(8, rect.height - 28));
    tooltip.style.left = `${Math.round(left)}px`;
    tooltip.style.top = `${Math.round(top)}px`;
  };
  overlay.addEventListener("mousemove", show);
  overlay.addEventListener("mouseenter", show);
  overlay.addEventListener("mouseleave", () => {
    tooltip.hidden = true;
  });
}

export {
  renderSystemMemory,
  renderSystemCPU,
  prepareSeriesRows,
  renderRequestsSeries,
  bindRequestsChartHover
};
