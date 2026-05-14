const GRID = 20;
const MIN_WIDTH = 220;
const MIN_HEIGHT = 140;
let layoutState = null;
const visibleWidgetsByScope = new Map();
const DASHBOARD_LAYOUT_STORAGE_KEY = "waf.dashboard.layout.v1";
const DASHBOARD_WIDGETS_STORAGE_KEY = "waf.dashboard.widgets.v1";

const WIDGETS = [
  { id: "services-up", titleKey: "dashboard.widget.servicesUp", width: 280, height: 200, x: 20, y: 20 },
  { id: "services-down", titleKey: "dashboard.widget.servicesDown", width: 280, height: 200, x: 320, y: 20 },
  { id: "requests-day", titleKey: "dashboard.widget.requestsDay", width: 340, height: 220, x: 620, y: 20 },
  { id: "attacks-day", titleKey: "dashboard.widget.attacksDay", width: 280, height: 220, x: 980, y: 20 },
  { id: "blocked-attacks", titleKey: "dashboard.widget.blockedAttacks", width: 280, height: 220, x: 1280, y: 20 },
  { id: "unique-attackers", titleKey: "dashboard.widget.uniqueAttackers", width: 300, height: 240, x: 1280, y: 260 },
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

function localizeContainerDuration(raw, ctx) {
  let text = String(raw || "").trim();
  if (!text) {
    return "";
  }
  text = text
    .replace(/\babout an hour\b/ig, `~1${ctx.t("dashboard.containers.time.hourShort")}`)
    .replace(/\bless than a second\b/ig, `<1${ctx.t("dashboard.containers.time.secondShort")}`)
    .replace(/\b(\d+)\s+seconds?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.secondShort")}`)
    .replace(/\b(\d+)\s+minutes?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.minuteShort")}`)
    .replace(/\b(\d+)\s+hours?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.hourShort")}`)
    .replace(/\b(\d+)\s+days?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.dayShort")}`)
    .replace(/\b(\d+)\s+weeks?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.weekShort")}`)
    .replace(/\b(\d+)\s+months?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.monthShort")}`)
    .replace(/\b(\d+)\s+years?\b/ig, (_m, n) => `${n}${ctx.t("dashboard.containers.time.yearShort")}`)
    .replace(/\s+ago\b/ig, ` ${ctx.t("dashboard.containers.status.ago")}`)
    .replace(/\s{2,}/g, " ")
    .trim();
  return text;
}

function formatContainerStatusLabel(value, ctx) {
  const cleaned = String(value || "")
    .replace(/\s*\((healthy|unhealthy|health:\s*starting)\)\s*/ig, " ")
    .replace(/\s{2,}/g, " ")
    .trim();
  if (!cleaned) {
    return "-";
  }
  const normalized = cleaned.toLowerCase();
  if (normalized === "created") {
    return ctx.t("dashboard.containers.status.created");
  }
  if (normalized === "paused") {
    return ctx.t("dashboard.containers.status.paused");
  }
  if (normalized === "dead") {
    return ctx.t("dashboard.containers.status.dead");
  }
  if (normalized === "removal in progress") {
    return ctx.t("dashboard.containers.status.removalInProgress");
  }
  if (normalized.startsWith("up ")) {
    return `${ctx.t("dashboard.containers.status.up")} ${localizeContainerDuration(cleaned.slice(3), ctx)}`.trim();
  }
  let match = cleaned.match(/^exited\s+\(([^)]+)\)\s*(.+)$/i);
  if (match) {
    return `${ctx.t("dashboard.containers.status.exited")} (${match[1]}) ${localizeContainerDuration(match[2], ctx)}`.trim();
  }
  match = cleaned.match(/^restarting\s+\(([^)]+)\)\s*(.+)$/i);
  if (match) {
    return `${ctx.t("dashboard.containers.status.restarting")} (${match[1]}) ${localizeContainerDuration(match[2], ctx)}`.trim();
  }
  return localizeContainerDuration(cleaned, ctx);
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

export {
  GRID,
  MIN_WIDTH,
  MIN_HEIGHT,
  WIDGETS,
  clamp,
  snap,
  normalizeLayout,
  loadLayout,
  saveLayout,
  loadVisibleWidgetIDs,
  saveVisibleWidgetIDs,
  formatNumber,
  formatPercent,
  formatBytes,
  normalizeContainerState,
  normalizeContainerStatus,
  getContainerStatusTone,
  localizeContainerDuration,
  formatContainerStatusLabel,
  formatUptimeLocalized,
  rectsOverlap,
  getLayoutItem,
  resolveOverlaps,
  constrainItem,
  constrainDraggedItem,
};
