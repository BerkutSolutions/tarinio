import { escapeHtml, setError } from "../ui.js";

const GRID = 20;
const MIN_WIDTH = 220;
const MIN_HEIGHT = 140;
const OBSERVATION_WINDOW_MS = 24 * 60 * 60 * 1000;
let layoutState = null;
const visibleWidgetsByScope = new Map();
const DASHBOARD_LAYOUT_STORAGE_KEY = "waf.dashboard.layout.v1";
const DASHBOARD_WIDGETS_STORAGE_KEY = "waf.dashboard.widgets.v1";

const WIDGETS = [
  { id: "services-up", titleKey: "dashboard.widget.servicesUp", width: 240, height: 180, x: 20, y: 20 },
  { id: "services-down", titleKey: "dashboard.widget.servicesDown", width: 240, height: 180, x: 280, y: 20 },
  { id: "requests-day", titleKey: "dashboard.widget.requestsDay", width: 300, height: 200, x: 540, y: 20 },
  { id: "attacks-day", titleKey: "dashboard.widget.attacksDay", width: 240, height: 200, x: 860, y: 20 },
  { id: "blocked-attacks", titleKey: "dashboard.widget.blockedAttacks", width: 240, height: 200, x: 1120, y: 20 },
  { id: "unique-attackers", titleKey: "dashboard.widget.uniqueAttackers", width: 300, height: 240, x: 1380, y: 20 },
  { id: "requests-series", titleKey: "dashboard.widget.requestsSeries", width: 1240, height: 340, x: 20, y: 280 },
  { id: "popular-errors", titleKey: "dashboard.widget.popularErrors", width: 360, height: 300, x: 20, y: 640 },
  { id: "top-ips", titleKey: "dashboard.widget.topIPs", width: 360, height: 300, x: 400, y: 640 },
  { id: "top-countries", titleKey: "dashboard.widget.topCountries", width: 360, height: 300, x: 780, y: 640 },
  { id: "top-urls", titleKey: "dashboard.widget.topURLs", width: 360, height: 300, x: 1160, y: 640 },
  { id: "memory", titleKey: "dashboard.widget.memory", width: 330, height: 260, x: 20, y: 960 },
  { id: "cpu", titleKey: "dashboard.widget.cpu", width: 330, height: 260, x: 370, y: 960 },
  { id: "containers-health", titleKey: "dashboard.widget.containersHealth", width: 360, height: 600, x: 1700, y: 20 }
];

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function snap(value) {
  return Math.round(value / GRID) * GRID;
}

function normalizeLayout(raw) {
  const map = new Map(Array.isArray(raw) ? raw.map((item) => [String(item?.id || ""), item]) : []);
  return WIDGETS.map((widget) => {
    const saved = map.get(widget.id) || {};
    return {
      id: widget.id,
      x: snap(Number.isFinite(saved.x) ? saved.x : widget.x),
      y: snap(Number.isFinite(saved.y) ? saved.y : widget.y),
      width: snap(clamp(Number.isFinite(saved.width) ? saved.width : widget.width, MIN_WIDTH, 1800)),
      height: snap(clamp(Number.isFinite(saved.height) ? saved.height : widget.height, MIN_HEIGHT, 900))
    };
  });
}

function loadLayout() {
  if (!layoutState) {
    let parsed = [];
    try {
      parsed = JSON.parse(window.localStorage.getItem(DASHBOARD_LAYOUT_STORAGE_KEY) || "[]");
    } catch (_error) {
      parsed = [];
    }
    layoutState = normalizeLayout(parsed);
  }
  return normalizeLayout(layoutState);
}

function saveLayout(layout) {
  layoutState = normalizeLayout(layout);
  try {
    window.localStorage.setItem(DASHBOARD_LAYOUT_STORAGE_KEY, JSON.stringify(layoutState));
  } catch (_error) {
    // ignore persistence errors
  }
}

function loadVisibleWidgetIDs(scopeID = "") {
  const fallback = WIDGETS.map((widget) => widget.id);
  const scope = String(scopeID || "").trim().toLowerCase();
  let parsed = visibleWidgetsByScope.get(scope);
  if (!Array.isArray(parsed)) {
    try {
      const allScopes = JSON.parse(window.localStorage.getItem(DASHBOARD_WIDGETS_STORAGE_KEY) || "{}");
      parsed = Array.isArray(allScopes?.[scope]) ? allScopes[scope] : null;
    } catch (_error) {
      parsed = null;
    }
  }
  if (!Array.isArray(parsed) || !parsed.length) {
    return fallback;
  }
  const allowed = new Set(fallback);
  const unique = [];
  parsed.forEach((id) => {
    const token = String(id || "");
    if (!allowed.has(token) || unique.includes(token)) {
      return;
    }
    unique.push(token);
  });
  return unique.length ? unique : fallback;
}

function saveVisibleWidgetIDs(ids, scopeID = "") {
  const scope = String(scopeID || "").trim().toLowerCase();
  const next = Array.isArray(ids) ? ids.slice() : [];
  visibleWidgetsByScope.set(scope, next);
  try {
    const raw = window.localStorage.getItem(DASHBOARD_WIDGETS_STORAGE_KEY) || "{}";
    const allScopes = JSON.parse(raw);
    allScopes[scope] = next;
    window.localStorage.setItem(DASHBOARD_WIDGETS_STORAGE_KEY, JSON.stringify(allScopes));
  } catch (_error) {
    // ignore persistence errors
  }
}

function formatNumber(value) {
  return Number(value || 0).toLocaleString();
}

function formatPercent(value) {
  return `${Number(value || 0).toFixed(1)}%`;
}

function formatBytes(bytes) {
  const value = Number(bytes || 0);
  if (!value) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = value;
  let unit = 0;
  while (n >= 1024 && unit < units.length - 1) {
    n /= 1024;
    unit++;
  }
  return `${n.toFixed(unit > 1 ? 1 : 0)} ${units[unit]}`;
}

function normalizeContainerState(value) {
  return String(value || "").trim().toLowerCase();
}

function normalizeContainerStatus(value) {
  return String(value || "").trim().toLowerCase();
}

function getContainerStatusTone(item) {
  const status = normalizeContainerStatus(item?.status);
  const state = normalizeContainerState(item?.state);
  if (status.includes("unhealthy") || state === "exited" || state === "dead") {
    return "danger";
  }
  if (status.includes("health: starting") || status.includes("starting") || status.includes("restarting") || state === "created" || state === "paused") {
    return "warning";
  }
  if (status.includes("healthy") || state === "running" || status.startsWith("up")) {
    return "success";
  }
  return "warning";
}

function formatContainerStatusLabel(value) {
  return String(value || "")
    .replace(/\s*\((healthy|unhealthy|health:\s*starting)\)\s*/ig, " ")
    .replace(/\s{2,}/g, " ")
    .trim();
}

function formatUptimeLocalized(seconds, ctx) {
  const total = Math.max(0, Math.floor(Number(seconds || 0)));
  if (!Number.isFinite(total) || total <= 0) {
    return "0m";
  }
  const minute = 60;
  const hour = 60 * minute;
  const day = 24 * hour;
  const month = 30 * day;
  const months = Math.floor(total / month);
  let rest = total % month;
  const days = Math.floor(rest / day);
  rest %= day;
  const hours = Math.floor(rest / hour);
  rest %= hour;
  const minutes = Math.floor(rest / minute);
  const chunks = [];
  if (months > 0) chunks.push(`${months}${ctx.t("dashboard.containers.time.monthShort")}`);
  if (days > 0) chunks.push(`${days}${ctx.t("dashboard.containers.time.dayShort")}`);
  if (hours > 0) chunks.push(`${hours}${ctx.t("dashboard.containers.time.hourShort")}`);
  if (minutes > 0 || !chunks.length) chunks.push(`${minutes}${ctx.t("dashboard.containers.time.minuteShort")}`);
  return chunks.join(" ");
}

function parseISOTime(value) {
  const ts = Date.parse(String(value || ""));
  return Number.isNaN(ts) ? 0 : ts;
}

function parseStatus(value) {
  const n = Number(value || 0);
  return Number.isFinite(n) ? Math.trunc(n) : 0;
}

function normalizeSiteID(value) {
  return String(value || "").trim().toLowerCase().replace(/_/g, "-");
}

function shouldSkipInternalSite(siteID) {
  const site = normalizeSiteID(siteID);
  return site === "control-plane-access" || site === "control-plane" || site === "ui";
}

function shouldSkipInternalRequest(uri, siteID) {
  if (shouldSkipInternalSite(siteID)) {
    return true;
  }
  const path = String(uri || "").trim().toLowerCase();
  if (!path) {
    return false;
  }
  return path.startsWith("/api/dashboard") ||
    path.startsWith("/dashboard") ||
    path.startsWith("/healthz") ||
    path.startsWith("/readyz") ||
    path.startsWith("/login");
}

function resolveSiteLabel(siteID, host) {
  const site = String(siteID || "").trim();
  if (site && site !== "-") {
    return site;
  }
  const hostLabel = String(host || "").trim().toLowerCase();
  if (hostLabel) {
    return hostLabel;
  }
  return "-";
}

function addToMap(map, key, delta = 1) {
  const token = String(key || "").trim();
  if (!token) {
    return;
  }
  map.set(token, Number(map.get(token) || 0) + Number(delta || 0));
}

function addToNestedMap(map, outerKey, innerKey, delta = 1) {
  const outer = String(outerKey || "").trim();
  const inner = String(innerKey || "").trim();
  if (!outer || !inner) {
    return;
  }
  let innerMap = map.get(outer);
  if (!innerMap) {
    innerMap = new Map();
    map.set(outer, innerMap);
  }
  addToMap(innerMap, inner, delta);
}

function topCounts(map, limit = 10) {
  const out = [];
  map.forEach((count, key) => {
    if (!String(key || "").trim() || Number(count || 0) <= 0) {
      return;
    }
    out.push({ key: String(key), count: Number(count) });
  });
  out.sort((a, b) => (b.count - a.count) || a.key.localeCompare(b.key));
  return limit > 0 && out.length > limit ? out.slice(0, limit) : out;
}

function normalizeCountryCode(value) {
  const raw = String(value || "").trim().replace(/^["']+|["']+$/g, "");
  const token = raw.toUpperCase();
  if (!token || token === "UNKNOWN" || token === "-" || token === "N/A") {
    return "UNK";
  }

  const bracket = token.match(/\(([A-Z]{2})\)\s*$/);
  if (bracket) {
    return bracket[1];
  }

  const exact = token.match(/\b([A-Z]{2})\b/);
  if (exact) {
    return exact[1];
  }

  const compact = token.replace(/[^A-Z]/g, "");
  if (compact.length === 2) {
    return compact;
  }
  return "UNK";
}

function countryFlag(code) {
  const token = normalizeCountryCode(code);
  if (!/^[A-Z]{2}$/.test(token)) {
    return "";
  }
  const first = 127397 + token.charCodeAt(0);
  const second = 127397 + token.charCodeAt(1);
  return String.fromCodePoint(first, second);
}

function countryName(code) {
  const token = normalizeCountryCode(code);
  if (token === "UNK") {
    return "Unknown";
  }
  try {
    const names = new Intl.DisplayNames(["ru", "en"], { type: "region" });
    return names.of(token) || token;
  } catch (_error) {
    return token;
  }
}

function blockedByStatus(status) {
  return status === 403 || status === 429 || status === 444;
}

function isSecurityEvent(item) {
  const type = String(item?.type || "");
  return type === "security_access" || type === "security_rate_limit" || type === "security_waf";
}

function isBlockedSecurityEvent(item) {
  const rawBlocked = item?.details?.blocked;
  if (typeof rawBlocked === "boolean") {
    return rawBlocked;
  }
  return blockedByStatus(parseStatus(item?.details?.status));
}

function ensureIPDetail(map, ip) {
  const key = String(ip || "").trim();
  if (!key) {
    return null;
  }
  if (!map.has(key)) {
    map.set(key, {
      ip: key,
      countryCounts: new Map(),
      requests: 0,
      attacks: 0,
      blocked: 0,
      pages: new Map(),
      methods: new Map(),
      sites: new Map(),
      errorCodes: new Map()
    });
  }
  return map.get(key);
}

function parseRequestsJSONL(text) {
  const raw = String(text || "").trim();
  if (!raw) {
    return [];
  }
  if (raw.startsWith("[")) {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed.filter((row) => row && typeof row === "object") : [];
    } catch (_error) {
      return [];
    }
  }
  const rows = [];
  for (const sourceLine of raw.split(/\r?\n/)) {
    const line = String(sourceLine || "").trim();
    if (!line) {
      continue;
    }
    try {
      const parsed = JSON.parse(line);
      if (parsed && typeof parsed === "object") {
        rows.push(parsed);
      }
    } catch (_error) {
      rows.push({ stream: "archive", ingested_at: "", raw: line, entry: {} });
    }
  }
  return rows;
}

async function fetchRequestsRows() {
  try {
    const response = await fetch("/api/requests", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "text/plain" }
    });
    if (!response.ok) {
      return [];
    }
    return parseRequestsJSONL(await response.text());
  } catch (_error) {
    return [];
  }
}

async function fetchEventsRows() {
  try {
    const response = await fetch("/api/events", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return [];
    }
    const payload = await response.json();
    if (Array.isArray(payload)) return payload;
    if (Array.isArray(payload?.events)) return payload.events;
    if (Array.isArray(payload?.items)) return payload.items;
    return [];
  } catch (_error) {
    return [];
  }
}

async function fetchContainersOverview() {
  try {
    const response = await fetch("/api/dashboard/containers/overview", {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return null;
    }
    return await response.json();
  } catch (_error) {
    return null;
  }
}

async function fetchContainerLogs(container, since = "", tail = 1000) {
  const params = new URLSearchParams();
  params.set("container", String(container || ""));
  if (since) {
    params.set("since", since);
  }
  if (tail > 0) {
    params.set("tail", String(tail));
  }
  const response = await fetch(`/api/dashboard/containers/logs?${params.toString()}`, {
    method: "GET",
    credentials: "include",
    headers: { Accept: "application/json" }
  });
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        message = String(payload.error);
      }
    } catch (_error) {
      // ignore parse errors
    }
    throw new Error(message);
  }
  return await response.json();
}

function renderContainersHealthWidget(overview, ctx) {
  if (!overview || !Array.isArray(overview?.containers)) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("dashboard.containers.empty"))}</div>`;
  }
  const containers = overview.containers
    .slice()
    .sort((left, right) => String(left?.name || "").localeCompare(String(right?.name || ""), undefined, { sensitivity: "base" }))
    .slice(0, 12);
  const uptimeText = formatUptimeLocalized(overview?.host_uptime_seconds || 0, ctx);
  return `
    <div class="dashboard-widget-content dashboard-containers-widget" data-widget-action="containers-health">
      <div class="dashboard-containers-uptime-inline">${escapeHtml(ctx.t("dashboard.containers.uptime"))}: ${escapeHtml(uptimeText)}</div>
      <div class="dashboard-containers-system-row">
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(formatPercent(overview?.total_cpu_percent || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.cpu"))}</div>
        </div>
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(formatPercent(overview?.avg_memory_percent || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.memory"))}</div>
        </div>
        <div class="mini-metric mini-metric-running">
          <div class="mini-metric-value">${escapeHtml(String(overview?.running_containers || 0))} / ${escapeHtml(String(overview?.total_containers || 0))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t("dashboard.containers.running"))}</div>
        </div>
      </div>
      <div class="dashboard-containers-list-wrap">
        <div class="dashboard-containers-list-title">${escapeHtml(ctx.t("dashboard.containers.container"))}</div>
        <div class="dashboard-list dashboard-containers-list">
          ${containers.map((item) => `
            <button type="button" class="dashboard-list-row clickable container-status-${escapeHtml(getContainerStatusTone(item))}" data-widget-action="container-logs" data-container-name="${escapeHtml(String(item?.name || ""))}">
              <div class="dashboard-list-label">
                <strong>${escapeHtml(String(item?.name || "-"))}</strong>
                <div class="muted">${escapeHtml(formatContainerStatusLabel(item?.status || "-"))}</div>
              </div>
              <div class="dashboard-list-meta">
                <span class="badge badge-neutral">CPU ${escapeHtml(formatPercent(item?.cpu_percent || 0))}</span>
                <span class="badge badge-neutral">MEM ${escapeHtml(formatPercent(item?.memory_percent || 0))}</span>
                <span class="badge badge-neutral">${escapeHtml(String(item?.network_in_text || "0 B"))} / ${escapeHtml(String(item?.network_out_text || "0 B"))}</span>
              </div>
            </button>
          `).join("")}
        </div>
      </div>
    </div>
  `;
}

function buildDetailModel(stats, requestRows, eventRows) {
  const generatedAt = parseISOTime(stats?.generated_at) || Date.now();
  const cutoff = generatedAt - OBSERVATION_WINDOW_MS;

  const requestsBySite = new Map();
  const requestsByURL = new Map();
  const requestsByMethod = new Map();
  const requestsByIP = new Map();
  const attacksBySite = new Map();
  const blockedBySite = new Map();
  const attacksByURL = new Map();
  const attacksByCountry = new Map();
  const errorsByCodeAndSite = new Map();
  const ipDetails = new Map();

  (Array.isArray(requestRows) ? requestRows : []).forEach((row) => {
    const entry = row?.entry && typeof row.entry === "object" ? row.entry : {};
    const when = parseISOTime(entry.timestamp || row?.ingested_at);
    if (!when || when < cutoff) {
      return;
    }
    const site = resolveSiteLabel(entry.site, entry.host);
    const uri = String(entry.uri || "-").trim() || "-";
    if (shouldSkipInternalRequest(uri, site)) {
      return;
    }
    const method = String(entry.method || "-").trim() || "-";
    const ip = String(entry.client_ip || "").trim();
    const status = parseStatus(entry.status);
    const requestCountry = normalizeCountryCode(entry.country || entry.client_country || entry.country_code);

    addToMap(requestsBySite, site, 1);
    addToMap(requestsByURL, uri, 1);
    addToMap(requestsByMethod, method, 1);

    if (ip) {
      addToMap(requestsByIP, ip, 1);
      const detail = ensureIPDetail(ipDetails, ip);
      if (detail) {
        detail.requests += 1;
        addToMap(detail.pages, uri, 1);
        addToMap(detail.methods, method, 1);
        addToMap(detail.sites, site, 1);
        if (requestCountry !== "UNK") {
          addToMap(detail.countryCounts, requestCountry, 1);
        }
      }
    }

    if (status >= 400 && status <= 599) {
      const code = String(status);
      addToNestedMap(errorsByCodeAndSite, code, site, 1);
      if (ip) {
        const detail = ensureIPDetail(ipDetails, ip);
        if (detail) {
          addToMap(detail.errorCodes, code, 1);
        }
      }
    }
  });

  (Array.isArray(eventRows) ? eventRows : []).forEach((item) => {
    if (!isSecurityEvent(item)) {
      return;
    }
    const when = parseISOTime(item?.occurred_at);
    if (!when || when < cutoff) {
      return;
    }
    const details = item?.details && typeof item.details === "object" ? item.details : {};
    const site = resolveSiteLabel(item?.site_id, details?.host);
    if (shouldSkipInternalSite(site)) {
      return;
    }
    const rawBlocked = details.blocked;
    if (typeof rawBlocked === "boolean" && !rawBlocked) {
      return;
    }

    const ip = String(details.client_ip || details.ip || "").trim();
    const path = String(details.path || details.uri || "-").trim() || "-";
    const status = parseStatus(details.status);
    const countryCode = normalizeCountryCode(details.country || details.client_country || details.country_code || details.geo_country);

    addToMap(attacksBySite, site, 1);
    addToMap(attacksByURL, path, 1);
    addToMap(attacksByCountry, countryCode, 1);
    if (isBlockedSecurityEvent(item)) {
      addToMap(blockedBySite, site, 1);
    }
    if (status >= 400 && status <= 599) {
      addToNestedMap(errorsByCodeAndSite, String(status), site, 1);
    }

    if (ip) {
      const detail = ensureIPDetail(ipDetails, ip);
      if (detail) {
        detail.attacks += 1;
        if (isBlockedSecurityEvent(item)) {
          detail.blocked += 1;
        }
        addToMap(detail.pages, path, 1);
        addToMap(detail.sites, site, 1);
        addToMap(detail.countryCounts, countryCode, 1);
      }
    }
  });

  const errorsByCode = [];
  const errorsByCodeSites = new Map();
  errorsByCodeAndSite.forEach((siteMap, code) => {
    const sites = topCounts(siteMap, 20);
    const total = sites.reduce((acc, item) => acc + Number(item.count || 0), 0);
    errorsByCode.push({ key: code, count: total });
    errorsByCodeSites.set(code, sites);
  });
  errorsByCode.sort((a, b) => (b.count - a.count) || a.key.localeCompare(b.key));

  const ipCountryByIP = new Map();
  const ipDetailsSummary = [];
  ipDetails.forEach((detail) => {
    const country = topCounts(detail.countryCounts, 1)[0]?.key || "UNK";
    ipCountryByIP.set(detail.ip, country);
    ipDetailsSummary.push({
      ip: detail.ip,
      countryCode: country,
      requests: detail.requests,
      attacks: detail.attacks,
      blocked: detail.blocked,
      pages: topCounts(detail.pages, 12),
      methods: topCounts(detail.methods, 8),
      sites: topCounts(detail.sites, 12),
      errorCodes: topCounts(detail.errorCodes, 8)
    });
  });
  ipDetailsSummary.sort((a, b) => (b.attacks - a.attacks) || (b.requests - a.requests) || a.ip.localeCompare(b.ip));

  return {
    requestsBySite: topCounts(requestsBySite, 20),
    requestsByURL: topCounts(requestsByURL, 20),
    requestsByMethod: topCounts(requestsByMethod, 10),
    requestsByIP: topCounts(requestsByIP, 20),
    attacksBySite: topCounts(attacksBySite, 20),
    blockedBySite: topCounts(blockedBySite, 20),
    attacksByURL: topCounts(attacksByURL, 20),
    attacksByCountry: topCounts(attacksByCountry, 20),
    errorsByCode,
    errorsByCodeSites,
    ipDetailsByIP: new Map(ipDetailsSummary.map((item) => [item.ip, item])),
    ipDetailsSummary,
    ipCountryByIP
  };
}

function rectsOverlap(a, b) {
  return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
}

function getLayoutItem(layout, id) {
  return layout.find((item) => item.id === id);
}

function resolveOverlaps(layout, movedID) {
  let changed = true;
  let guard = 0;
  while (changed && guard < 200) {
    changed = false;
    guard++;
    for (let i = 0; i < layout.length; i++) {
      for (let j = i + 1; j < layout.length; j++) {
        const a = layout[i];
        const b = layout[j];
        if (!rectsOverlap(a, b)) continue;
        let target = b;
        let fixed = a;
        if (a.id === movedID && b.id !== movedID) {
          target = b;
          fixed = a;
        } else if (b.id === movedID && a.id !== movedID) {
          target = a;
          fixed = b;
        } else if (a.y > b.y) {
          target = a;
          fixed = b;
        }
        target.y = snap(Math.max(target.y, fixed.y + fixed.height + GRID));
        changed = true;
      }
    }
  }
}

function constrainItem(item, boardWidth) {
  item.width = snap(clamp(item.width, MIN_WIDTH, Math.max(MIN_WIDTH, boardWidth - GRID)));
  item.height = snap(clamp(item.height, MIN_HEIGHT, 900));
  item.x = snap(Math.max(0, item.x));
  item.y = snap(Math.max(0, item.y));
  if (item.x + item.width > boardWidth) {
    item.x = snap(Math.max(0, boardWidth - item.width));
  }
}

function constrainDraggedItem(item, boardWidth, mode) {
  item.width = snap(clamp(item.width, MIN_WIDTH, Math.max(MIN_WIDTH, boardWidth - GRID)));
  item.height = snap(clamp(item.height, MIN_HEIGHT, 900));
  item.x = snap(Math.max(0, item.x));
  item.y = snap(Math.max(0, item.y));
  if (item.x + item.width <= boardWidth) {
    return;
  }
  if (mode.includes("e") && !mode.includes("w")) {
    item.width = snap(clamp(boardWidth - item.x, MIN_WIDTH, Math.max(MIN_WIDTH, boardWidth - GRID)));
    return;
  }
  item.x = snap(Math.max(0, boardWidth - item.width));
}

function applyFrameGeometry(boardNode, layout, frameNode) {
  const item = getLayoutItem(layout, String(frameNode.dataset.widgetId || ""));
  if (!item) {
    return;
  }
  const mobile = window.matchMedia("(max-width: 900px)").matches;
  if (mobile) {
    frameNode.style.left = "";
    frameNode.style.top = "";
    frameNode.style.width = "";
    frameNode.style.height = "";
  } else {
    frameNode.style.left = `${item.x}px`;
    frameNode.style.top = `${item.y}px`;
    frameNode.style.width = `${item.width}px`;
    frameNode.style.height = `${item.height}px`;
  }
}

function applyAllGeometry(boardNode, layout) {
  const frameNodes = Array.from(boardNode.querySelectorAll(".dashboard-frame"));
  const visibleIDs = new Set(frameNodes.map((node) => String(node.dataset.widgetId || "")));
  frameNodes.forEach((frameNode) => {
    applyFrameGeometry(boardNode, layout, frameNode);
  });
  const visibleLayout = layout.filter((entry) => visibleIDs.has(entry.id));
  const maxBottom = visibleLayout.reduce((acc, current) => Math.max(acc, current.y + current.height), 0);
  boardNode.style.minHeight = `${Math.max(560, maxBottom + GRID)}px`;
}

function renderMetric(value, label, tone = "success", widgetAction = "") {
  const clickable = widgetAction ? "clickable" : "";
  const actionAttr = widgetAction ? ` data-widget-action="${escapeHtml(widgetAction)}"` : "";
  return `
    <button type="button" class="dashboard-widget-content dashboard-stat dashboard-metric tone-${escapeHtml(tone)} ${clickable}"${actionAttr}>
      <div class="dashboard-stat-value">${escapeHtml(formatNumber(value))}</div>
      <div class="dashboard-stat-label">${escapeHtml(label)}</div>
    </button>
  `;
}

function renderCountryBadge(code) {
  const normalized = normalizeCountryCode(code);
  if (normalized === "UNK") {
    return `<span class="dashboard-ip-country">${escapeHtml(countryName(normalized))}</span>`;
  }
  const flag = countryFlag(normalized);
  const name = countryName(normalized);
  return `<span class="dashboard-ip-country">${escapeHtml(name)}${flag ? ` (${escapeHtml(flag)})` : ""}</span>`;
}

function renderTopList(items, emptyText, options = {}) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="dashboard-widget-content waf-empty">${escapeHtml(emptyText)}</div>`;
  }
  const renderLabel = typeof options.renderLabel === "function" ? options.renderLabel : (item) => escapeHtml(String(item?.key || "-"));
  const renderMetaRight = typeof options.renderMetaRight === "function" ? options.renderMetaRight : () => "";
  const rowAttrs = typeof options.rowAttrs === "function" ? options.rowAttrs : () => "";
  const containerAttr = options.containerAction ? ` data-widget-action="${escapeHtml(options.containerAction)}"` : "";
  return `
    <div class="dashboard-widget-content dashboard-scroll-area"${containerAttr}>
      <div class="dashboard-list">
        ${items.map((item, index) => {
          const attrs = rowAttrs(item, index);
          return `
            <button type="button" class="dashboard-list-row ${attrs ? "clickable" : ""}" ${attrs}>
              <div class="dashboard-list-label">${renderLabel(item)}</div>
              <div class="dashboard-list-meta">
                <span class="badge badge-neutral">${escapeHtml(formatNumber(item?.count || 0))}</span>
                ${renderMetaRight(item)}
              </div>
            </button>
          `;
        }).join("")}
      </div>
    </div>
  `;
}

function renderIPTopList(items, emptyText, widgetAction = "") {
  return renderTopList(items, emptyText, {
    containerAction: widgetAction,
    renderLabel: (item) => {
      const ip = String(item?.key || "-").trim() || "-";
      return `<span class="dashboard-ip-main">${escapeHtml(ip)}</span><div class="dashboard-ip-country-block">${renderCountryBadge(item?.countryCode || item?.key)}</div>`;
    },
    rowAttrs: (item) => {
      const ip = String(item?.key || "").trim();
      if (!ip) {
        return widgetAction ? `data-widget-action="${escapeHtml(widgetAction)}"` : "";
      }
      return `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"`;
    }
  });
}

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

function mergeWidgetData(stats, detailModel, containersOverview, ctx) {
  const statsTopIPs = Array.isArray(stats?.top_attacker_ips) ? stats.top_attacker_ips : [];
  const statsTopCountries = Array.isArray(stats?.top_attacker_countries) ? stats.top_attacker_countries : [];
  const fallbackTopCountries = Array.isArray(detailModel?.attacksByCountry) ? detailModel.attacksByCountry : [];
  const topCountryItems = (statsTopCountries.length ? statsTopCountries : fallbackTopCountries).map((item) => ({
    key: item?.key,
    count: item?.count,
    countryCode: item?.key
  }));
  const topIPItems = statsTopIPs.map((item) => {
    const key = String(item?.key || "").trim();
    return {
      key,
      count: Number(item?.count || 0),
      countryCode: detailModel?.ipCountryByIP?.get?.(key) || "UNK"
    };
  });

  return {
    "services-up": renderMetric(stats?.services_up, ctx.t("dashboard.value.servicesUp"), "success", "services-up"),
    "services-down": renderMetric(stats?.services_down, ctx.t("dashboard.value.servicesDown"), "danger", "services-down"),
    "requests-day": renderMetric(stats?.requests_day, ctx.t("dashboard.value.requestsDay"), "success", "requests-day"),
    "attacks-day": renderMetric(stats?.attacks_day, ctx.t("dashboard.value.attacksDay"), "warning", "attacks-day"),
    "blocked-attacks": renderMetric(stats?.blocked_attacks_day, ctx.t("dashboard.value.blockedAttacksDay"), "danger", "blocked-attacks"),
    "unique-attackers": renderIPTopList(topIPItems.length ? topIPItems : (detailModel?.ipDetailsSummary || []).slice(0, 10).map((item) => ({
      key: item.ip,
      count: Math.max(item.attacks, item.requests),
      countryCode: item.countryCode
    })), ctx.t("dashboard.empty.topIPs"), "unique-attackers"),
    "popular-errors": renderTopList(stats?.popular_errors || [], ctx.t("dashboard.empty.popularErrors"), {
      containerAction: "popular-errors",
      rowAttrs: (item) => {
        const code = String(item?.key || "").trim();
        return code ? `data-widget-action="error-detail" data-error-code="${escapeHtml(code)}"` : "";
      }
    }),
    "top-ips": renderIPTopList(topIPItems, ctx.t("dashboard.empty.topIPs"), "top-ips"),
    "top-countries": renderTopList(topCountryItems, ctx.t("dashboard.empty.topCountries"), {
      containerAction: "top-countries",
      renderLabel: (item) => renderCountryBadge(item?.countryCode || item?.key),
      rowAttrs: (item) => `data-widget-action="country-detail" data-country-code="${escapeHtml(normalizeCountryCode(item?.countryCode || item?.key))}"`
    }),
    "top-urls": renderTopList(stats?.most_attacked_urls || [], ctx.t("dashboard.empty.topURLs"), {
      containerAction: "top-urls",
      rowAttrs: (item) => {
        const url = String(item?.key || "").trim();
        return url ? `data-widget-action="url-detail" data-url="${escapeHtml(url)}"` : "";
      }
    }),
    memory: renderSystemMemory(stats, ctx),
    cpu: renderSystemCPU(stats, ctx),
    "containers-health": renderContainersHealthWidget(containersOverview, ctx)
  };
}

function createFrame(widget, ctx) {
  const frameNode = document.createElement("section");
  frameNode.className = "waf-card dashboard-frame";
  frameNode.dataset.widgetId = widget.id;
  frameNode.innerHTML = `
    <div class="waf-card-head dashboard-frame-header"><h3>${escapeHtml(ctx.t(widget.titleKey))}</h3></div>
    <div class="waf-card-body" data-widget-body="${escapeHtml(widget.id)}"><div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div></div>
    <div class="frame-resize-handle resize-se" data-resize-dir="se" title="${escapeHtml(ctx.t("dashboard.action.resize"))}"></div>
    <div class="frame-resize-handle resize-e" data-resize-dir="e"></div>
    <div class="frame-resize-handle resize-s" data-resize-dir="s"></div>
    <div class="frame-resize-handle resize-w" data-resize-dir="w"></div>
    <div class="frame-resize-handle resize-n" data-resize-dir="n"></div>
  `;
  return frameNode;
}

function wireFrameInteractions(pageNode, boardNode, layout, frameNode, onLayoutMutated) {
  let dragState = null;
  const handlePointerDown = (event) => {
    if (pageNode.dataset.editMode !== "1" || window.matchMedia("(max-width: 900px)").matches) {
      return;
    }
    const item = getLayoutItem(layout, String(frameNode.dataset.widgetId || ""));
    if (!item) {
      return;
    }
    const resizeHandle = event.target.closest("[data-resize-dir]");
    const header = event.target.closest(".dashboard-frame-header");
    if (!resizeHandle && !header) {
      return;
    }
    event.preventDefault();
    dragState = {
      id: item.id,
      pointerID: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      baseX: item.x,
      baseY: item.y,
      baseWidth: item.width,
      baseHeight: item.height,
      changed: false,
      mode: resizeHandle ? `resize-${String(resizeHandle.dataset.resizeDir || "se")}` : "move"
    };
    frameNode.classList.add("dragging");
    frameNode.setPointerCapture(event.pointerId);
  };

  const handlePointerMove = (event) => {
    if (!dragState || dragState.pointerID !== event.pointerId) {
      return;
    }
    const item = getLayoutItem(layout, dragState.id);
    if (!item) {
      return;
    }
    const boardWidth = Math.max(600, boardNode.clientWidth);
    const dx = event.clientX - dragState.startX;
    const dy = event.clientY - dragState.startY;
    if (dragState.mode === "move") {
      item.x = snap(dragState.baseX + dx);
      item.y = snap(dragState.baseY + dy);
    } else {
      let x = dragState.baseX;
      let y = dragState.baseY;
      let width = dragState.baseWidth;
      let height = dragState.baseHeight;
      if (dragState.mode.includes("e")) width = snap(dragState.baseWidth + dx);
      if (dragState.mode.includes("s")) height = snap(dragState.baseHeight + dy);
      if (dragState.mode.includes("w")) {
        x = snap(dragState.baseX + dx);
        width = snap(dragState.baseWidth - dx);
      }
      if (dragState.mode.includes("n")) {
        y = snap(dragState.baseY + dy);
        height = snap(dragState.baseHeight - dy);
      }
      item.x = x;
      item.y = y;
      item.width = width;
      item.height = height;
    }
    dragState.changed = true;
    constrainDraggedItem(item, boardWidth, dragState.mode);
    applyAllGeometry(boardNode, layout);
  };

  const handlePointerEnd = (event) => {
    if (!dragState || dragState.pointerID !== event.pointerId) {
      return;
    }
    frameNode.classList.remove("dragging");
    frameNode.releasePointerCapture(event.pointerId);
    const boardWidth = Math.max(600, boardNode.clientWidth);
    layout.forEach((entry) => constrainItem(entry, boardWidth));
    resolveOverlaps(layout, dragState.id);
    applyAllGeometry(boardNode, layout);
    if (dragState.changed) onLayoutMutated?.();
    dragState = null;
  };

  frameNode.addEventListener("pointerdown", handlePointerDown);
  frameNode.addEventListener("pointermove", handlePointerMove);
  frameNode.addEventListener("pointerup", handlePointerEnd);
  frameNode.addEventListener("pointercancel", handlePointerEnd);
}

function renderSummaryMetrics(items, ctx) {
  if (!Array.isArray(items) || !items.length) {
    return "";
  }
  const displayValue = (value) => {
    if (typeof value === "string") {
      return value;
    }
    return formatNumber(value);
  };
  return `
    <div class="dashboard-mini-metrics">
      ${items.map((item) => `
        <div class="mini-metric">
          <div class="mini-metric-value">${escapeHtml(displayValue(item.value))}</div>
          <div class="mini-metric-label">${escapeHtml(ctx.t(item.labelKey))}</div>
        </div>
      `).join("")}
    </div>
  `;
}

function renderDetailTable(items, ctx, labelTitle, countTitle, options = {}) {
  const list = Array.isArray(items) ? items : [];
  if (!list.length) {
    return `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
  }
  const labelFormatter = typeof options.labelFormatter === "function"
    ? options.labelFormatter
    : (item) => escapeHtml(String(item?.key || "-"));
  const countFormatter = typeof options.countFormatter === "function"
    ? options.countFormatter
    : (item) => escapeHtml(formatNumber(item?.count || 0));
  const attrs = typeof options.rowAttrs === "function" ? options.rowAttrs : () => "";
  return `
    <div class="waf-table-wrap">
      <table class="waf-table">
        <thead><tr><th>${escapeHtml(labelTitle)}</th><th>${escapeHtml(countTitle)}</th></tr></thead>
        <tbody>
          ${list.map((item, index) => {
            const rowAttrs = attrs(item, index);
            const extraClass = rowAttrs ? "waf-table-row-clickable" : "";
            const roleAttrs = rowAttrs ? ' tabindex="0" role="button"' : "";
            return `<tr class="${extraClass}" ${rowAttrs}${roleAttrs}><td>${labelFormatter(item)}</td><td>${countFormatter(item)}</td></tr>`;
          }).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderDetailSection(title, content) {
  return `
    <section class="dashboard-detail-section">
      <h4 class="dashboard-detail-section-title">${escapeHtml(title)}</h4>
      ${content}
    </section>
  `;
}

function buildWidgetDetail(action, payload, stats, detailModel, containersOverview, ctx) {
  const services = Array.isArray(stats?.services) ? stats.services : [];
  const attackBySiteMap = new Map((detailModel?.attacksBySite || []).map((item) => [item.key, item.count]));

  if (action === "services-up" || action === "services-down") {
    const isUp = action === "services-up";
    const rows = services.filter((item) => Boolean(item?.up) === isUp).map((item) => ({
      key: `${item?.name || "-"} (${item?.up ? "UP" : "DOWN"}, ${item?.checked_at || "-"})`,
      count: 1
    }));
    return {
      title: ctx.t(isUp ? "dashboard.widget.servicesUp" : "dashboard.widget.servicesDown"),
      subtitle: ctx.t("dashboard.detail.componentsSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: isUp ? "dashboard.value.servicesUp" : "dashboard.value.servicesDown", value: rows.length },
        { labelKey: "dashboard.detail.totalComponents", value: services.length }
      ], ctx) + renderDetailTable(rows, ctx, ctx.t("dashboard.detail.component"), ctx.t("dashboard.detail.state"))
    };
  }

  if (action === "requests-day") {
    return {
      title: ctx.t("dashboard.widget.requestsDay"),
      subtitle: ctx.t("dashboard.detail.requestsSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.value.requestsDay", value: stats?.requests_day || 0 },
        { labelKey: "dashboard.detail.uniqueIPs", value: detailModel?.requestsByIP?.length || 0 }
      ], ctx) +
      renderDetailTable(detailModel?.requestsBySite || [], ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests")) +
      renderDetailTable(detailModel?.requestsByURL || [], ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.requests"))
    };
  }

  if (action === "attacks-day") {
    return {
      title: ctx.t("dashboard.widget.attacksDay"),
      subtitle: ctx.t("dashboard.detail.attacksSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.value.attacksDay", value: stats?.attacks_day || 0 },
        { labelKey: "dashboard.value.blockedAttacksDay", value: stats?.blocked_attacks_day || 0 }
      ], ctx) +
      renderDetailTable(detailModel?.attacksBySite || [], ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.attacks")) +
      renderDetailTable(detailModel?.attacksByURL || [], ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.attacks"))
    };
  }

  if (action === "blocked-attacks") {
    const rows = (detailModel?.blockedBySite || []).map((item) => ({
      key: item.key,
      count: item.count,
      total: attackBySiteMap.get(item.key) || 0
    }));
    const blockedIPs = (detailModel?.ipDetailsSummary || [])
      .filter((item) => Number(item?.blocked || 0) > 0)
      .slice(0, 20)
      .map((item) => ({
        key: item.ip,
        count: item.blocked,
        countryCode: item.countryCode
      }));
    return {
      title: ctx.t("dashboard.widget.blockedAttacks"),
      subtitle: ctx.t("dashboard.detail.blockedSubtitle"),
      body: renderDetailTable(rows, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.blocked"), {
        labelFormatter: (item) => {
          const pct = item.total > 0 ? `${((item.count * 100) / item.total).toFixed(1)}%` : "0%";
          return `${escapeHtml(item.key)} <span class="muted">(${escapeHtml(pct)})</span>`;
        }
      }) +
      renderDetailTable(blockedIPs, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.blocked"), {
        labelFormatter: (item) => `${escapeHtml(String(item?.key || "-"))} ${renderCountryBadge(item?.countryCode)}`,
        rowAttrs: (item) => {
          const ip = String(item?.key || "").trim();
          return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
        }
      })
    };
  }

  if (action === "popular-errors") {
    return {
      title: ctx.t("dashboard.widget.popularErrors"),
      subtitle: ctx.t("dashboard.detail.errorsSubtitle"),
      body: renderDetailTable(detailModel?.errorsByCode || stats?.popular_errors || [], ctx, ctx.t("dashboard.detail.errorCode"), ctx.t("dashboard.detail.requests"), {
        rowAttrs: (item) => {
          const code = String(item?.key || "").trim();
          return code ? `data-widget-action="error-detail" data-error-code="${escapeHtml(code)}"` : "";
        }
      })
    };
  }

  if (action === "error-detail") {
    const code = String(payload?.errorCode || "").trim();
    return {
      title: `${ctx.t("dashboard.detail.errorCode")} ${code || "-"}`,
      subtitle: ctx.t("dashboard.detail.errorsBySiteSubtitle"),
      body: renderDetailTable(detailModel?.errorsByCodeSites?.get?.(code) || [], ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests"))
    };
  }

  if (action === "top-ips" || action === "unique-attackers") {
    const items = (detailModel?.ipDetailsSummary || []).slice(0, 20).map((item) => ({
      key: item.ip,
      count: Math.max(item.attacks, item.requests),
      countryCode: item.countryCode
    }));
    return {
      title: ctx.t("dashboard.widget.topIPs"),
      subtitle: ctx.t("dashboard.detail.topIPsSubtitle"),
      body: renderDetailTable(items, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.requests"), {
        labelFormatter: (item) => `${escapeHtml(String(item?.key || "-"))} ${renderCountryBadge(item?.countryCode)}`,
        rowAttrs: (item) => {
          const ip = String(item?.key || "").trim();
          return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
        }
      })
    };
  }

  if (action === "ip-detail") {
    const ip = String(payload?.ip || "").trim();
    const detail = detailModel?.ipDetailsByIP?.get?.(ip);
    if (!detail) {
      return { title: `${ctx.t("dashboard.detail.ip")} ${ip || "-"}`, subtitle: ctx.t("dashboard.detail.ipSubtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>` };
    }
    return {
      title: `${ctx.t("dashboard.detail.ip")} ${detail.ip}`,
      subtitle: ctx.t("dashboard.detail.ipSubtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.detail.requests", value: detail.requests },
        { labelKey: "dashboard.detail.attacks", value: detail.attacks },
        { labelKey: "dashboard.detail.blocked", value: detail.blocked }
      ], ctx) +
      `<div class="dashboard-ip-country-block">${renderCountryBadge(detail.countryCode)}</div>` +
      renderDetailTable(detail.sites, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests")) +
      renderDetailTable(detail.pages, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.requests")) +
      renderDetailTable(detail.methods, ctx, ctx.t("dashboard.detail.method"), ctx.t("dashboard.detail.requests"))
    };
  }

  if (action === "top-countries" || action === "country-detail") {
    const code = normalizeCountryCode(payload?.countryCode || "");
    const rows = code && code !== "UNK" ? (detailModel?.attacksByCountry || []).filter((item) => normalizeCountryCode(item.key) === code) : (detailModel?.attacksByCountry || []);
    return {
      title: code && code !== "UNK" ? `${ctx.t("dashboard.detail.country")} ${countryName(code)}${countryFlag(code) ? ` (${countryFlag(code)})` : ""}` : ctx.t("dashboard.widget.topCountries"),
      subtitle: ctx.t("dashboard.detail.countrySubtitle"),
      body: renderDetailTable(rows, ctx, ctx.t("dashboard.detail.country"), ctx.t("dashboard.detail.attacks"), { labelFormatter: (item) => renderCountryBadge(item.key) })
    };
  }

  if (action === "top-urls" || action === "url-detail") {
    const targetURL = String(payload?.url || "").trim();
    const rows = targetURL ? (detailModel?.attacksByURL || []).filter((item) => item.key === targetURL) : (detailModel?.attacksByURL || []);
    return {
      title: targetURL ? `${ctx.t("dashboard.detail.page")}: ${targetURL}` : ctx.t("dashboard.widget.topURLs"),
      subtitle: ctx.t("dashboard.detail.urlSubtitle"),
      body: renderDetailTable(rows, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.attacks"))
    };
  }

  if (action === "memory" || action === "cpu") {
    const system = stats?.system || {};
    const metrics = action === "cpu"
      ? [
        { labelKey: "dashboard.detail.cpuLoad", value: formatPercent(system.cpu_load_percent || 0) },
        { labelKey: "dashboard.system.cpuCores", value: system.cpu_cores || 0 },
        { labelKey: "dashboard.system.goroutines", value: system.goroutines || 0 }
      ]
      : [
        { labelKey: "dashboard.detail.memoryUsedBytes", value: formatBytes(system.memory_used_bytes || 0) },
        { labelKey: "dashboard.detail.memoryFreeBytes", value: formatBytes(system.memory_free_bytes || 0) },
        { labelKey: "dashboard.detail.memoryTotalBytes", value: formatBytes(system.memory_total_bytes || 0) }
      ];
    const loadSections = [];
    const requestsBySite = detailModel?.requestsBySite || [];
    const requestsByURL = detailModel?.requestsByURL || [];
    const requestsByIP = detailModel?.requestsByIP || [];
    const attacksBySite = detailModel?.attacksBySite || [];

    if (requestsBySite.length) {
      loadSections.push(renderDetailSection(
        ctx.t("dashboard.detail.loadBySites"),
        renderDetailTable(requestsBySite, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.requests"))
      ));
    }
    if (requestsByURL.length) {
      loadSections.push(renderDetailSection(
        ctx.t("dashboard.detail.loadByPages"),
        renderDetailTable(requestsByURL, ctx, ctx.t("dashboard.detail.page"), ctx.t("dashboard.detail.requests"))
      ));
    }
    if (requestsByIP.length) {
      loadSections.push(renderDetailSection(
        ctx.t("dashboard.detail.loadByIPs"),
        renderDetailTable(requestsByIP, ctx, ctx.t("dashboard.detail.ip"), ctx.t("dashboard.detail.requests"), {
          labelFormatter: (item) => {
            const ip = String(item?.key || "-");
            const countryCode = detailModel?.ipCountryByIP?.get?.(ip) || "UNK";
            return `${escapeHtml(ip)} ${renderCountryBadge(countryCode)}`;
          },
          rowAttrs: (item) => {
            const ip = String(item?.key || "").trim();
            return ip ? `data-widget-action="ip-detail" data-ip="${escapeHtml(ip)}"` : "";
          }
        })
      ));
    }
    if (attacksBySite.length) {
      loadSections.push(renderDetailSection(
        ctx.t("dashboard.detail.loadByAttackSites"),
        renderDetailTable(attacksBySite, ctx, ctx.t("dashboard.detail.site"), ctx.t("dashboard.detail.attacks"))
      ));
    }
    return {
      title: ctx.t(action === "cpu" ? "dashboard.widget.cpu" : "dashboard.widget.memory"),
      subtitle: ctx.t("dashboard.detail.loadSubtitle"),
      body: renderSummaryMetrics(metrics, ctx) +
      (loadSections.length
        ? loadSections.join("")
        : `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.detail.loadSourcesEmpty"))}</div>`)
    };
  }

  if (action === "containers-health") {
    const overview = containersOverview;
    if (!overview || !Array.isArray(overview?.containers)) {
      return {
        title: ctx.t("dashboard.widget.containersHealth"),
        subtitle: ctx.t("dashboard.containers.subtitle"),
        body: `<div class="waf-empty">${escapeHtml(ctx.t("dashboard.containers.empty"))}</div>`
      };
    }
    const rows = overview.containers
      .slice()
      .sort((left, right) => String(left?.name || "").localeCompare(String(right?.name || ""), undefined, { sensitivity: "base" }))
      .map((item) => ({
      ...item,
      key: item?.name || "-",
      count: item?.cpu_percent || 0
      }));
    return {
      title: ctx.t("dashboard.widget.containersHealth"),
      subtitle: ctx.t("dashboard.containers.subtitle"),
      body: renderSummaryMetrics([
        { labelKey: "dashboard.containers.uptime", value: formatUptimeLocalized(overview?.host_uptime_seconds || 0, ctx) },
        { labelKey: "dashboard.containers.cpu", value: formatPercent(overview?.total_cpu_percent || 0) },
        { labelKey: "dashboard.containers.memory", value: formatPercent(overview?.avg_memory_percent || 0) },
        { labelKey: "dashboard.containers.network", value: `${overview?.total_network_in_text || "0 B"} / ${overview?.total_network_out_text || "0 B"}` }
      ], ctx) + renderDetailTable(rows, ctx, ctx.t("dashboard.containers.container"), ctx.t("dashboard.containers.cpu"), {
        labelFormatter: (item) => `
          <div><strong>${escapeHtml(String(item?.name || "-"))}</strong></div>
          <div class="muted">${escapeHtml(formatContainerStatusLabel(item?.status || "-"))}</div>
          <div class="muted">MEM ${escapeHtml(formatPercent(item?.memory_percent || 0))} | NET ${escapeHtml(String(item?.network_in_text || "0 B"))} / ${escapeHtml(String(item?.network_out_text || "0 B"))}</div>
        `,
        rowAttrs: (item) => {
          const name = String(item?.name || "").trim();
          if (!name) {
            return "";
          }
          return `data-status-tone="${escapeHtml(getContainerStatusTone(item))}" data-widget-action="container-logs" data-container-name="${escapeHtml(name)}"`;
        },
        countFormatter: (item) => `CPU ${formatPercent(item?.cpu_percent || 0)}`
      })
    };
  }

  return { title: ctx.t("dashboard.detail.title"), subtitle: ctx.t("dashboard.detail.subtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>` };
}

function createModalState(container, ctx) {
  const modalNode = container.querySelector("#dashboard-detail-modal");
  const modalTitleNode = container.querySelector("#dashboard-detail-title");
  const modalSubtitleNode = container.querySelector("#dashboard-detail-subtitle");
  const modalBodyNode = container.querySelector("#dashboard-detail-content");

  const close = () => {
    modalNode?.classList.add("waf-hidden");
  };

  modalNode?.querySelectorAll("[data-dashboard-detail-close='true']").forEach((node) => {
    node.addEventListener("click", close);
  });
  modalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      close();
    }
  });

  return {
    open(detail) {
      if (!modalNode || !modalTitleNode || !modalSubtitleNode || !modalBodyNode) {
        return;
      }
      modalTitleNode.textContent = String(detail?.title || ctx.t("dashboard.detail.title"));
      modalSubtitleNode.textContent = String(detail?.subtitle || "");
      modalBodyNode.innerHTML = detail?.body || `<div class="waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
      modalNode.classList.remove("waf-hidden");
      modalNode.focus();
    },
    close
  };
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

  let latestStats = null;
  let latestContainersOverview = null;
  let detailModel = null;
  let detailModelGeneratedAt = "";
  let detailModelPromise = null;
  let layoutDirty = false;
  let requestsChartRenderRAF = 0;
  let liveLogsTimer = null;
  let liveLogsState = null;
  let containersOverviewFailCount = 0;
  let containersOverviewNextRetryAt = 0;
  const widgetsScopeID = String(ctx?.currentUser?.username || ctx?.currentUser?.id || "").trim().toLowerCase();
  const visibleWidgetIDs = new Set(loadVisibleWidgetIDs(widgetsScopeID));

  const ensureDetailModel = async () => {
    if (!latestStats) {
      return null;
    }
    const generatedAt = String(latestStats?.generated_at || "");
    if (detailModel && detailModelGeneratedAt === generatedAt) {
      return detailModel;
    }
    if (detailModelPromise) {
      return detailModelPromise;
    }
    detailModelPromise = Promise.all([fetchRequestsRows(), fetchEventsRows()])
      .then(([requestsRows, eventsRows]) => {
        detailModel = buildDetailModel(latestStats, requestsRows, eventsRows);
        detailModelGeneratedAt = generatedAt;
        return detailModel;
      })
      .catch(() => {
        detailModel = buildDetailModel(latestStats, [], []);
        detailModelGeneratedAt = generatedAt;
        return detailModel;
      })
      .finally(() => {
        detailModelPromise = null;
      });
    return detailModelPromise;
  };

  const modal = createModalState(container, ctx);
  const modalBodyNode = container.querySelector("#dashboard-detail-content");

  const stopLiveLogs = () => {
    if (liveLogsTimer) {
      clearInterval(liveLogsTimer);
      liveLogsTimer = null;
    }
    liveLogsState = null;
  };

  container.querySelectorAll("[data-dashboard-detail-close='true']").forEach((node) => {
    node.addEventListener("click", stopLiveLogs);
  });
  container.querySelector("#dashboard-detail-modal")?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      stopLiveLogs();
    }
  });

  const renderLogsBody = () => {
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

  const updateLiveLogsModalBody = () => {
    if (!modalBodyNode) {
      return;
    }
    const prevPreNode = modalBodyNode.querySelector(".dashboard-container-logs-pre");
    const prevScrollTop = prevPreNode ? prevPreNode.scrollTop : 0;
    const wasPinnedToBottom = prevPreNode
      ? (prevPreNode.scrollHeight - prevPreNode.scrollTop - prevPreNode.clientHeight) < 24
      : true;
    modalBodyNode.innerHTML = renderLogsBody();
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

  const pollContainerLogs = async () => {
    if (!liveLogsState || !liveLogsState.container) {
      return;
    }
    const payload = await fetchContainerLogs(liveLogsState.container, liveLogsState.since || "", liveLogsState.since ? 0 : 1500);
    const rows = Array.isArray(payload?.lines) ? payload.lines : [];
    if (rows.length) {
      liveLogsState.lines.push(...rows);
      if (liveLogsState.lines.length > 8000) {
        liveLogsState.lines = liveLogsState.lines.slice(liveLogsState.lines.length - 8000);
      }
      const lastTs = rows[rows.length - 1]?.timestamp;
      if (lastTs) {
        liveLogsState.since = lastTs;
      }
    }
    updateLiveLogsModalBody();
  };

  const mountWidgetFrame = (widget) => {
    if (!widget || boardNode.querySelector(`.dashboard-frame[data-widget-id="${widget.id}"]`)) {
      return;
    }
    const frameNode = createFrame(widget, ctx);
    boardNode.appendChild(frameNode);
    wireFrameInteractions(pageNode, boardNode, layout, frameNode, () => {
      layoutDirty = true;
    });
  };

  const unmountWidgetFrame = (widgetID) => {
    const frameNode = boardNode.querySelector(`.dashboard-frame[data-widget-id="${widgetID}"]`);
    if (frameNode) {
      frameNode.remove();
    }
  };

  WIDGETS.filter((widget) => visibleWidgetIDs.has(widget.id)).forEach((widget) => {
    mountWidgetFrame(widget);
  });
  applyAllGeometry(boardNode, layout);

  const renderRequestsWidget = (stats) => {
    const bodyNode = boardNode.querySelector('[data-widget-body="requests-series"]');
    if (!bodyNode) {
      return;
    }
    const rows = prepareSeriesRows(stats);
    const chartWidth = Math.floor(bodyNode.clientWidth || 1100) - 16;
    bodyNode.innerHTML = renderRequestsSeries(rows, ctx, chartWidth);
    bindRequestsChartHover(bodyNode, rows, ctx);
  };

  const renderStats = (stats) => {
    latestStats = stats;
    const rendered = mergeWidgetData(stats, detailModel, latestContainersOverview, ctx);
    WIDGETS.forEach((widget) => {
      const bodyNode = boardNode.querySelector(`[data-widget-body="${widget.id}"]`);
      if (!bodyNode) return;
      if (widget.id === "requests-series") {
        renderRequestsWidget(stats);
      } else {
        const prevContainersScrollTop = widget.id === "containers-health"
          ? (bodyNode.querySelector(".dashboard-containers-list")?.scrollTop || 0)
          : 0;
        bodyNode.innerHTML = rendered[widget.id] || `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
        if (widget.id === "containers-health") {
          const nextList = bodyNode.querySelector(".dashboard-containers-list");
          if (nextList) {
            nextList.scrollTop = prevContainersScrollTop;
          }
        }
      }
    });

    ensureDetailModel().then((computed) => {
      if (!latestStats || latestStats !== stats) {
        return;
      }
      const rerendered = mergeWidgetData(stats, computed, latestContainersOverview, ctx);
      ["unique-attackers", "top-ips", "top-countries", "containers-health"].forEach((id) => {
        const bodyNode = boardNode.querySelector(`[data-widget-body="${id}"]`);
        if (!bodyNode) return;
        const prevContainersScrollTop = id === "containers-health"
          ? (bodyNode.querySelector(".dashboard-containers-list")?.scrollTop || 0)
          : 0;
        bodyNode.innerHTML = rerendered[id] || `<div class="dashboard-widget-content waf-empty">${escapeHtml(ctx.t("common.none"))}</div>`;
        if (id === "containers-health") {
          const nextList = bodyNode.querySelector(".dashboard-containers-list");
          if (nextList) {
            nextList.scrollTop = prevContainersScrollTop;
          }
        }
      });
    }).catch(() => {});
  };

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
    saveVisibleWidgetIDs(Array.from(visibleWidgetIDs), widgetsScopeID);
  };

  const renderWidgetsMenu = () => {
    if (!widgetsMenuNode) {
      return;
    }
    widgetsMenuNode.innerHTML = `
      <div class="dashboard-widget-picker-title">${escapeHtml(ctx.t("dashboard.widgets.subtitle"))}</div>
      ${WIDGETS.map((widget) => {
        const checked = visibleWidgetIDs.has(widget.id) ? "checked" : "";
        return `
          <label class="dashboard-widget-picker-row">
            <input type="checkbox" data-widget-visibility-id="${escapeHtml(widget.id)}" ${checked}>
            <span>${escapeHtml(ctx.t(widget.titleKey))}</span>
          </label>
        `;
      }).join("")}
      <div class="dashboard-widget-picker-actions">
        <button type="button" class="btn primary btn-sm" id="dashboard-widgets-save">${escapeHtml(ctx.t("common.save"))}</button>
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
      const widget = WIDGETS.find((item) => item.id === widgetID);
      if (widget) {
        mountWidgetFrame(widget);
      }
      applyAllGeometry(boardNode, layout);
      if (latestStats) {
        renderStats(latestStats);
      }
    } else {
      visibleWidgetIDs.delete(widgetID);
      unmountWidgetFrame(widgetID);
      applyAllGeometry(boardNode, layout);
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
    if (action !== "container-logs" && !latestStats) {
      return;
    }
    if (action !== "container-logs") {
      stopLiveLogs();
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
      modal.open({ title: `${ctx.t("dashboard.containers.logs.title")} ${name}`, subtitle: ctx.t("dashboard.containers.logs.subtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>` });
      liveLogsState = { container: name, since: "", lines: [] };
      try {
        await pollContainerLogs();
      } catch (error) {
        modal.open({ title: `${ctx.t("dashboard.containers.logs.title")} ${name}`, subtitle: ctx.t("dashboard.containers.logs.subtitle"), body: `<div class="alert">${escapeHtml(error?.message || ctx.t("dashboard.containers.logs.error"))}</div>` });
        stopLiveLogs();
        return;
      }
      liveLogsTimer = window.setInterval(async () => {
        const modalNode = container.querySelector("#dashboard-detail-modal");
        if (!modalNode || modalNode.classList.contains("waf-hidden")) {
          stopLiveLogs();
          return;
        }
        try {
          await pollContainerLogs();
        } catch (_error) {
          // silent live retry
        }
      }, 2000);
      return;
    }
    modal.open({ title: ctx.t("dashboard.detail.loadingTitle"), subtitle: ctx.t("dashboard.detail.loadingSubtitle"), body: `<div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>` });
    const computed = await ensureDetailModel();
    modal.open(buildWidgetDetail(action, payload, latestStats, computed || buildDetailModel(latestStats, [], []), latestContainersOverview, ctx));
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

  const load = async (silent = false) => {
    try {
      const stats = await ctx.api.get("/api/dashboard/stats");
      if (Date.now() >= containersOverviewNextRetryAt) {
        const containersOverview = await fetchContainersOverview();
        if (containersOverview) {
          latestContainersOverview = containersOverview;
          containersOverviewFailCount = 0;
          containersOverviewNextRetryAt = 0;
        } else {
          latestContainersOverview = null;
          containersOverviewFailCount += 1;
          const delay = Math.min(120000, 5000 * (2 ** Math.max(0, containersOverviewFailCount - 1)));
          containersOverviewNextRetryAt = Date.now() + delay;
        }
      }
      renderStats(stats);
      if (!silent) {
        applyAllGeometry(boardNode, layout);
      }
    } catch (error) {
      if (!silent) {
        setError(boardNode, error?.message || ctx.t("dashboard.error"));
      }
    }
  };

  container.querySelector("#dashboard-edit-toggle")?.addEventListener("click", () => {
    const wasEdit = pageNode.dataset.editMode === "1";
    pageNode.dataset.editMode = wasEdit ? "0" : "1";
    container.querySelector("#dashboard-edit-toggle").textContent = pageNode.dataset.editMode === "1" ? ctx.t("dashboard.action.doneEdit") : ctx.t("dashboard.action.editLayout");
    pageNode.classList.toggle("dashboard-edit-mode", pageNode.dataset.editMode === "1");
    if (wasEdit && layoutDirty) {
      saveLayout(layout);
      layoutDirty = false;
    }
  });

  container.querySelector("#dashboard-layout-reset")?.addEventListener("click", () => {
    const defaults = normalizeLayout([]);
    layout.splice(0, layout.length, ...defaults);
    saveLayout(layout);
    layoutDirty = false;
    applyAllGeometry(boardNode, layout);
    if (latestStats) {
      renderStats(latestStats);
    }
  });

  if (container.__dashboardResizeHandler) {
    window.removeEventListener("resize", container.__dashboardResizeHandler);
  }
  container.__dashboardResizeHandler = () => {
    applyAllGeometry(boardNode, layout);
    if (latestStats) {
      renderRequestsWidget(latestStats);
    }
  };
  window.addEventListener("resize", container.__dashboardResizeHandler);

  if (container.__dashboardRequestsResizeObserver) {
    container.__dashboardRequestsResizeObserver.disconnect();
  }
  if (typeof ResizeObserver !== "undefined") {
    const requestsBody = boardNode.querySelector('[data-widget-body="requests-series"]');
    if (requestsBody) {
      container.__dashboardRequestsResizeObserver = new ResizeObserver(() => {
        if (!latestStats) {
          return;
        }
        if (requestsChartRenderRAF) {
          window.cancelAnimationFrame(requestsChartRenderRAF);
        }
        requestsChartRenderRAF = window.requestAnimationFrame(() => {
          renderRequestsWidget(latestStats);
          requestsChartRenderRAF = 0;
        });
      });
      container.__dashboardRequestsResizeObserver.observe(requestsBody);
    }
  }

  if (container.__dashboardAutoRefreshTimer) {
    clearInterval(container.__dashboardAutoRefreshTimer);
  }
  container.__dashboardAutoRefreshTimer = window.setInterval(() => {
    if (!document.body.contains(pageNode)) {
      clearInterval(container.__dashboardAutoRefreshTimer);
      container.__dashboardAutoRefreshTimer = null;
      return;
    }
    load(true);
  }, 5000);

  await load(false);
}
