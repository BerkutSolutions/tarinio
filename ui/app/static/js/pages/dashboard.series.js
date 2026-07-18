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
  const attacks = Array.isArray(stats?.attacks_series) ? stats.attacks_series : [];
  const requests = Array.isArray(stats?.requests_series) ? stats.requests_series : [];
  const hourFormatter = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", hour12: false });
  const attacksByTimestamp = new Map(attacks.map((item) => [String(item?.timestamp || ""), Number(item?.count || 0)]));
  const source = requests.length ? requests : attacks;
  return source.map((item) => ({
    label: (() => {
      const ts = String(item?.timestamp || "");
      if (!ts) {
        return String(item?.label || "");
      }
      const dt = new Date(ts);
      return Number.isNaN(dt.getTime()) ? String(item?.label || "") : hourFormatter.format(dt);
    })(),
    timestamp: String(item?.timestamp || ""),
    requests: Number(item?.count || 0),
    attacks: attacksByTimestamp.get(String(item?.timestamp || "")) || 0
  }));
}

function renderRequestsSeries(rows, ctx, chartWidthPx) {
  if (!rows.length) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.series.empty"))}</div>`;
  }
  const maxRequests = Math.max(1, ...rows.map((row) => row.requests));
  // This is the middle tick of the Y axis, not the arithmetic mean of the
  // observation window. A mean is misleading for sparse traffic and makes
  // the scale read "2.25" below a peak of 54; the midpoint must be 27.
  const midpointRequests = maxRequests / 2;
  const maxY = Math.max(maxRequests, ...rows.map((row) => row.attacks));
  const width = Math.max(320, Math.floor(chartWidthPx || 1100));
  const height = 230;
  const pad = { left: 40, right: 20, top: 18, bottom: 34 };
  const chartWidth = width - pad.left - pad.right;
  const chartHeight = height - pad.top - pad.bottom;

  const toPoints = (field) => rows.map((row, index) => {
    const x = pad.left + (index / Math.max(rows.length - 1, 1)) * chartWidth;
    const y = pad.top + (1 - (row[field] / maxY)) * chartHeight;
    return { x, y, ...row };
  });
  const toPath = (points) => points.reduce((path, point, index) => {
    if (index === 0) return `M ${point.x.toFixed(2)} ${point.y.toFixed(2)}`;
    const previous = points[index - 1];
    const controlX = ((previous.x + point.x) / 2).toFixed(2);
    return `${path} C ${controlX} ${previous.y.toFixed(2)}, ${controlX} ${point.y.toFixed(2)}, ${point.x.toFixed(2)} ${point.y.toFixed(2)}`;
  }, "");
  const requestPath = toPath(toPoints("requests"));
  const attackPath = toPath(toPoints("attacks"));
  const gridLines = 4;
  const xLabelStep = Math.max(1, Math.ceil((rows.length - 1) / 4));

  return `
    <div class="dashboard-widget-content dashboard-series" data-requests-chart="true">
      <div class="dashboard-series-legend">
        <span><i class="dot dot-requests"></i>${escapeHtml(ctx.t("dashboard.series.requests"))}</span>
        <span><i class="dot dot-attacks"></i>${escapeHtml(ctx.t("dashboard.series.attacks"))}</span>
      </div>
      <div class="dashboard-series-canvas">
        <svg viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" class="dashboard-series-svg">
          ${Array.from({ length: gridLines }, (_, index) => {
            const y = pad.top + ((index + 1) / gridLines) * chartHeight;
            return `<line x1="${pad.left}" y1="${y}" x2="${width - pad.right}" y2="${y}" stroke="rgba(148, 163, 184, 0.16)" stroke-width="1" stroke-dasharray="3 7"></line>`;
          }).join("")}
          <text x="${pad.left - 8}" y="${pad.top + 4}" text-anchor="end" font-size="11" fill="rgba(203, 213, 225, 0.68)">${escapeHtml(formatNumber(maxRequests))}</text>
          <text x="${pad.left - 8}" y="${pad.top + chartHeight / 2 + 4}" text-anchor="end" font-size="11" fill="rgba(203, 213, 225, 0.68)">${escapeHtml(formatNumber(midpointRequests))}</text>
          <path d="${requestPath}" fill="none" stroke="#2895ff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></path>
          <path d="${attackPath}" fill="none" stroke="#f17322" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></path>
          ${rows.map((row, index) => {
            if (index % xLabelStep !== 0 && index !== rows.length - 1) return "";
            const x = pad.left + (index / Math.max(rows.length - 1, 1)) * chartWidth;
            return `<text x="${x}" y="${height - 8}" text-anchor="middle" font-size="11" fill="rgba(203, 213, 225, 0.68)">${escapeHtml(row.label)}</text>`;
          }).join("")}
          <line data-chart-cursor="true" x1="0" x2="0" y1="${pad.top}" y2="${pad.top + chartHeight}" stroke="rgba(148, 163, 184, 0.8)" stroke-width="1" stroke-dasharray="3 5" hidden></line>
          <circle data-chart-point="true" cx="0" cy="0" r="4" fill="#12181f" stroke="#f17322" stroke-width="2" hidden></circle>
          <rect x="${pad.left}" y="${pad.top}" width="${chartWidth}" height="${chartHeight}" fill="transparent" data-chart-overlay="true"></rect>
        </svg>
        <div class="dashboard-chart-tooltip" data-chart-tooltip="true" hidden></div>
      </div>
    </div>
  `;
}

function bindRequestsChartHover(bodyNode, rows, ctx) {
  const overlay = bodyNode.querySelector("[data-chart-overlay='true']");
  const tooltip = bodyNode.querySelector("[data-chart-tooltip='true']");
  const svg = bodyNode.querySelector(".dashboard-series-svg");
  const cursor = bodyNode.querySelector("[data-chart-cursor='true']");
  const point = bodyNode.querySelector("[data-chart-point='true']");
  if (!overlay || !tooltip || !svg || !rows.length) {
    return;
  }
  const viewBox = String(svg.getAttribute("viewBox") || "").split(/\s+/).map((part) => Number(part));
  const width = Number.isFinite(viewBox[2]) ? viewBox[2] : 1100;
  const padLeft = 40;
  const padRight = 20;
  const chartWidth = width - padLeft - padRight;
  const nearest = (x) => clamp(Math.round(((x - padLeft) / chartWidth) * (rows.length - 1)), 0, rows.length - 1);
  const show = (event) => {
    const rect = svg.getBoundingClientRect();
    const localX = ((event.clientX - rect.left) / Math.max(rect.width, 1)) * width;
    const row = rows[nearest(localX)];
    const timestamp = new Date(row.timestamp);
    const formatted = Number.isNaN(timestamp.getTime())
      ? row.label
      : new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit", hour12: false }).format(timestamp);
    tooltip.textContent = `${formatted}\n${ctx.t("dashboard.series.requests")}: ${formatNumber(row.requests)}\n${ctx.t("dashboard.series.attacks")}: ${formatNumber(row.attacks)}`;
    tooltip.hidden = false;
    const pointX = padLeft + (nearest(localX) / Math.max(rows.length - 1, 1)) * chartWidth;
    const pointY = 18 + (1 - (row.attacks / Math.max(1, ...rows.flatMap((item) => [item.requests, item.attacks])))) * (Number(viewBox[3]) - 52);
    if (cursor) {
      cursor.setAttribute("x1", String(pointX));
      cursor.setAttribute("x2", String(pointX));
      cursor.hidden = false;
    }
    if (point) {
      point.setAttribute("cx", String(pointX));
      point.setAttribute("cy", String(pointY));
      point.hidden = false;
    }
    const left = clamp(event.clientX - rect.left + 12, 8, Math.max(8, rect.width - (tooltip.offsetWidth || 180) - 8));
    const top = clamp(event.clientY - rect.top - 36, 8, Math.max(8, rect.height - 28));
    tooltip.style.left = `${Math.round(left)}px`;
    tooltip.style.top = `${Math.round(top)}px`;
  };
  overlay.addEventListener("mousemove", show);
  overlay.addEventListener("mouseenter", show);
  overlay.addEventListener("mouseleave", () => {
    tooltip.hidden = true;
    if (cursor) cursor.hidden = true;
    if (point) point.hidden = true;
  });
}

export {
  renderSystemMemory,
  renderSystemCPU,
  prepareSeriesRows,
  renderRequestsSeries,
  bindRequestsChartHover
};
