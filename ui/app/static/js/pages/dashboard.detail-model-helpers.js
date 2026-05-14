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

const TARINIO_ADMIN_EXACT_PATHS = new Set([
  "/",
  "/login",
  "/login/2fa",
  "/challenge",
  "/challenge/verify"
]);

const TARINIO_ADMIN_PREFIX_PATHS = [
  "/static/",
  "/api/app/",
  "/api/auth/",
  "/api/dashboard/",
  "/api/reports/",
  "/api/sites",
  "/api/upstreams",
  "/api/certificates",
  "/api/tls-configs",
  "/api/easy-site-profiles",
  "/api/access-policies",
  "/api/requests",
  "/api/revisions",
  "/api/events",
  "/api/bans",
  "/api/jobs",
  "/api/settings",
  "/api/administration"
];

const TARINIO_ADMIN_SEGMENT_PREFIXES = [
  "/dashboard",
  "/sites",
  "/services",
  "/anti-ddos",
  "/tls",
  "/requests",
  "/revisions",
  "/events",
  "/bans",
  "/jobs",
  "/administration",
  "/activity",
  "/settings",
  "/about",
  "/profile",
  "/healthcheck",
  "/onboarding"
];

function isInternalManagementHost(host) {
  const value = String(host || "").trim().toLowerCase();
  return !value || value === "localhost" || value === "127.0.0.1" || value === "::1" || value === "control-plane" || value === "ui";
}

function isInternalManagementPath(path) {
  return path.startsWith("/api/") ||
    path.startsWith("/static/") ||
    path.startsWith("/dashboard") ||
    path.startsWith("/healthz") ||
    path.startsWith("/readyz") ||
    path.startsWith("/login") ||
    path.startsWith("/logout") ||
    path.startsWith("/setup") ||
    path.startsWith("/onboarding") ||
    path.startsWith("/favicon") ||
    path.startsWith("/manifest") ||
    path.startsWith("/site.webmanifest");
}

function isTarinioAdminAppPath(path) {
  const normalized = String(path || "").trim().toLowerCase();
  if (!normalized) {
    return false;
  }
  if (TARINIO_ADMIN_EXACT_PATHS.has(normalized)) {
    return true;
  }
  if (TARINIO_ADMIN_PREFIX_PATHS.some((prefix) => normalized.startsWith(prefix))) {
    return true;
  }
  return TARINIO_ADMIN_SEGMENT_PREFIXES.some((prefix) => normalized === prefix || normalized.startsWith(`${prefix}/`));
}

function shouldSkipInternalRequest(uri, siteID, host = "") {
  if (shouldSkipInternalSite(siteID)) {
    return true;
  }
  const path = String(uri || "").trim().toLowerCase();
  if (!path) {
    return false;
  }
  if (isTarinioAdminAppPath(path)) {
    return true;
  }
  if (!isInternalManagementPath(path)) {
    return false;
  }
  return isInternalManagementHost(host) || !String(siteID || "").trim();
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

export {
  parseISOTime,
  parseStatus,
  shouldSkipInternalRequest,
  resolveSiteLabel,
  addToMap,
  addToNestedMap,
  topCounts,
  normalizeCountryCode,
  isSecurityEvent,
  isBlockedSecurityEvent,
  ensureIPDetail
};