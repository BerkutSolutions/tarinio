import { escapeHtml, formatDate, setError, setLoading } from "../ui.js";

const ICON_PLUS = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v14h-2zM5 11h14v2H5z"/></svg>';
const ICON_UNLOCK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v2h-2V7a3 3 0 1 0-6 0v2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-9a2 2 0 0 1 2-2h2V7a5 5 0 0 1 5-5Z"/></svg>';
const MANUAL_BAN_TIMERS = new Map();
const ALL_SERVICES_SITE_ID = "__all__";

const EXTEND_DURATIONS = [
  { key: "1h", seconds: 60 * 60, labelKey: "bans.extend.duration.1h" },
  { key: "3h", seconds: 3 * 60 * 60, labelKey: "bans.extend.duration.3h" },
  { key: "6h", seconds: 6 * 60 * 60, labelKey: "bans.extend.duration.6h" },
  { key: "12h", seconds: 12 * 60 * 60, labelKey: "bans.extend.duration.12h" },
  { key: "1d", seconds: 24 * 60 * 60, labelKey: "bans.extend.duration.1d" },
  { key: "1w", seconds: 7 * 24 * 60 * 60, labelKey: "bans.extend.duration.1w" },
  { key: "1mo", seconds: 30 * 24 * 60 * 60, labelKey: "bans.extend.duration.1mo" },
  { key: "forever", seconds: 0, labelKey: "bans.extend.duration.forever" }
];

function normalizeIP(value) {
  return String(value || "").trim();
}

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

function buildPageButtons(totalPages, currentPage, dataAttr) {
  const pages = [];
  for (let page = 1; page <= Math.min(10, totalPages); page += 1) {
    pages.push(`<button type="button" class="btn ghost btn-sm${page === currentPage ? " active" : ""}" ${dataAttr}="${page}">${page}</button>`);
  }
  if (totalPages > 10) {
    pages.push(`<span class="muted">...</span>`);
    pages.push(`<button type="button" class="btn ghost btn-sm${totalPages === currentPage ? " active" : ""}" ${dataAttr}="${totalPages}">${totalPages}</button>`);
  }
  return pages.join("");
}

async function tryGetJSON(path) {
  try {
    const response = await fetch(path, {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" }
    });
    if (!response.ok) {
      return null;
    }
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  } catch (_error) {
    return null;
  }
}

function mergeByID(primary, secondary, idField = "id") {
  const map = new Map();
  for (const item of normalizeList(primary)) {
    const id = String(item?.[idField] || "").trim();
    if (!id) {
      continue;
    }
    map.set(id, { ...item, _origin: "primary" });
  }
  for (const item of normalizeList(secondary)) {
    const id = String(item?.[idField] || "").trim();
    if (!id || map.has(id)) {
      continue;
    }
    map.set(id, { ...item, _origin: "secondary" });
  }
  return Array.from(map.values());
}

function unwrapList(payload, keys = []) {
  if (Array.isArray(payload)) {
    return payload;
  }
  for (const key of keys) {
    if (Array.isArray(payload?.[key])) {
      return payload[key];
    }
  }
  return [];
}

function asDate(value) {
  const stamp = Date.parse(String(value || ""));
  return Number.isNaN(stamp) ? null : new Date(stamp);
}

function normalizeSiteToken(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+/, "")
    .replace(/-+$/, "");
}

function buildSiteAliasMap(sites) {
  const map = new Map();
  for (const site of normalizeList(sites)) {
    const canonical = String(site?.id || "").trim();
    if (!canonical) {
      continue;
    }
    const lower = canonical.toLowerCase();
    const token = normalizeSiteToken(canonical);
    map.set(lower, canonical);
    if (token) {
      map.set(token, canonical);
      map.set(token.replace(/-/g, "_"), canonical);
    }
  }
  return map;
}

function buildHostSiteMap(sites) {
  const map = new Map();
  for (const site of normalizeList(sites)) {
    const id = String(site?.id || "").trim();
    const host = String(site?.primary_host || "").trim().toLowerCase();
    if (!id) {
      continue;
    }
    map.set(id.toLowerCase(), id);
    if (host) {
      map.set(host, id);
    }
  }
  return map;
}

function resolveCanonicalSiteID(siteID, aliasMap, hostMap, hostHint = "") {
  const raw = String(siteID || "").trim();
  if (raw) {
    const lower = raw.toLowerCase();
    const direct = aliasMap.get(lower) || aliasMap.get(normalizeSiteToken(raw)) || hostMap.get(lower);
    if (direct) {
      return direct;
    }
    if (raw !== "-" && raw !== "unknown") {
      return raw;
    }
  }
  const host = String(hostHint || "").trim().toLowerCase();
  if (host) {
    return hostMap.get(host) || "";
  }
  return "";
}

function siteCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_${slug || "site"}`;
}

function siteEscalationCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_escalation_${slug || "site"}`;
}

function clearRateLimitCookies(siteID) {
  const rateCookie = siteCookieName(siteID);
  const escalationCookie = siteEscalationCookieName(siteID);
  document.cookie = `${rateCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
  document.cookie = `${escalationCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
}

function buildIPSetBySite(accessPolicies, fieldName, resolveSiteID) {
  const out = new Map();
  for (const policy of normalizeList(accessPolicies)) {
    const siteID = resolveSiteID(String(policy?.site_id || "").trim());
    if (!siteID) {
      continue;
    }
    const set = out.get(siteID) || new Set();
    for (const value of normalizeList(policy?.[fieldName])) {
      const ip = normalizeIP(value);
      if (ip) {
        set.add(ip);
      }
    }
    out.set(siteID, set);
  }
  return out;
}

function formatRemaining(expiresAt, now, t) {
  if (expiresAt === null) {
    return t("bans.time.permanent");
  }
  const diff = expiresAt.getTime() - now.getTime();
  if (diff <= 0) {
    return t("bans.time.expired");
  }
  const totalSec = Math.floor(diff / 1000);
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

function manualBanTimerRowKey(siteID, ip) {
  return `${String(siteID || "").trim().toLowerCase()}::${normalizeIP(ip).toLowerCase()}`;
}

function isStartupSelfTestSite(siteID) {
  return String(siteID || "").trim().toLowerCase().startsWith("startup-self-test-");
}

function loadManualBanTimers() {
  return new Map(Array.from(MANUAL_BAN_TIMERS.entries()).map(([key, value]) => [key, { ...value }]));
}

function saveManualBanTimers(map) {
  MANUAL_BAN_TIMERS.clear();
  for (const [key, value] of map.entries()) {
    MANUAL_BAN_TIMERS.set(key, { ...value });
  }
}

function upsertManualBanTimer(map, siteID, ip, unbanAt, createdAt = new Date().toISOString()) {
  const key = manualBanTimerRowKey(siteID, ip);
  map.set(key, {
    siteID: String(siteID || "").trim(),
    ip: normalizeIP(ip),
    unbanAt: new Date(unbanAt).toISOString(),
    createdAt: String(createdAt || "").trim() || new Date().toISOString()
  });
}

function removeManualBanTimer(map, siteID, ip) {
  map.delete(manualBanTimerRowKey(siteID, ip));
}

async function processExpiredManualBanTimers(ctx, timers) {
  const now = Date.now();
  let changed = false;
  for (const item of Array.from(timers.values())) {
    const unbanAt = Date.parse(String(item?.unbanAt || ""));
    if (Number.isNaN(unbanAt) || unbanAt > now) {
      continue;
    }
    try {
      await postBanAction(ctx, item.siteID, "primary", "unban", item.ip);
      removeManualBanTimer(timers, item.siteID, item.ip);
      changed = true;
    } catch (_error) {
      // Keep timer entry and retry on next refresh/page open.
    }
  }
  if (changed) {
    saveManualBanTimers(timers);
  }
}

function buildManualBanRows(accessPolicies, resolveSiteID, manualBanTimers) {
  const now = Date.now();
  const rows = [];
  for (const policy of normalizeList(accessPolicies)) {
    const rawSiteID = String(policy?.site_id || "").trim();
    if (isStartupSelfTestSite(rawSiteID)) {
      continue;
    }
    const siteID = resolveSiteID(rawSiteID) || rawSiteID;
    if (!siteID) {
      continue;
    }
    const updatedAt = asDate(policy?.updated_at);
    for (const ip of normalizeList(policy?.denylist)) {
      const value = String(ip || "").trim();
      if (!value) {
        continue;
      }
      const timer = manualBanTimers.get(manualBanTimerRowKey(siteID, value));
      const timerStamp = Date.parse(String(timer?.unbanAt || ""));
      const expiresAt = Number.isNaN(timerStamp) || timerStamp <= now ? null : new Date(timerStamp);
      rows.push({
        siteID,
        ip: value,
        country: "-",
        source: "manual",
        occurredAt: updatedAt,
        expiresAt,
        modules: new Set(["manual"]),
        origin: policy?._origin || "primary"
      });
    }
  }
  return rows;
}

function mergeBanRows(manualRows, autoRows) {
  const merged = new Map();
  const allRows = [...normalizeList(autoRows), ...normalizeList(manualRows)];
  for (const row of allRows) {
    const key = `${String(row?.siteID || "").trim().toLowerCase()}::${normalizeIP(row?.ip || "").toLowerCase()}`;
    if (!key || key === "::") {
      continue;
    }
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, {
        ...row,
        modules: new Set(row?.modules || []),
        statuses: new Set(row?.statuses || []),
        reasons: new Set(row?.reasons || []),
        paths: new Set(row?.paths || []),
        hosts: new Set(row?.hosts || []),
        referers: new Set(row?.referers || []),
        userAgents: new Set(row?.userAgents || []),
        eventIDs: new Set(row?.eventIDs || [])
      });
      continue;
    }

    const existingAt = existing?.occurredAt instanceof Date ? existing.occurredAt.getTime() : 0;
    const nextAt = row?.occurredAt instanceof Date ? row.occurredAt.getTime() : 0;
    const preferRow = row.source === "manual" || nextAt >= existingAt;

    merged.set(key, {
      ...(preferRow ? existing : row),
      ...(preferRow ? row : existing),
      source: existing.source === "manual" || row.source === "manual" ? "manual" : (row.source || existing.source),
      occurredAt: preferRow ? row.occurredAt : existing.occurredAt,
      expiresAt: row.source === "manual" ? row.expiresAt : (existing.source === "manual" ? existing.expiresAt : (preferRow ? row.expiresAt : existing.expiresAt)),
      modules: new Set([...(existing.modules || []), ...(row.modules || [])]),
      statuses: new Set([...(existing.statuses || []), ...(row.statuses || [])]),
      reasons: new Set([...(existing.reasons || []), ...(row.reasons || [])]),
      paths: new Set([...(existing.paths || []), ...(row.paths || [])]),
      hosts: new Set([...(existing.hosts || []), ...(row.hosts || [])]),
      referers: new Set([...(existing.referers || []), ...(row.referers || [])]),
      userAgents: new Set([...(existing.userAgents || []), ...(row.userAgents || [])]),
      eventIDs: new Set([...(existing.eventIDs || []), ...(row.eventIDs || [])]),
      blockedCount: Math.max(Number(existing.blockedCount || 0), Number(row.blockedCount || 0)),
      latestEvent: preferRow ? (row.latestEvent || existing.latestEvent) : (existing.latestEvent || row.latestEvent),
      origin: existing.origin === "primary" || row.origin === "primary" ? "primary" : (row.origin || existing.origin)
    });
  }
  return Array.from(merged.values());
}

function parsePositiveNumber(value, fallback = 0) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return fallback;
  }
  return parsed;
}

function parseSecurityStatus(value) {
  const parsed = Number.parseInt(String(value || ""), 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

function normalizeCountryCode(value) {
  const raw = String(value || "").trim().replace(/^["']+|["']+$/g, "");
  const token = raw.toUpperCase();
  if (!token || token === "UNKNOWN" || token === "-" || token === "N/A") {
    return "";
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
  return compact.length === 2 ? compact : "";
}

function countryFlagEmoji(value) {
  const code = normalizeCountryCode(value);
  if (!code) {
    return "-";
  }
  const base = 0x1f1e6;
  const first = code.charCodeAt(0) - 65;
  const second = code.charCodeAt(1) - 65;
  if (first < 0 || first > 25 || second < 0 || second > 25) {
    return "-";
  }
  return String.fromCodePoint(base + first, base + second);
}

function deriveModuleFromEvent(item) {
  const type = String(item?.type || "").trim().toLowerCase();
  const status = parseSecurityStatus(item?.details?.status);
  if (type === "security_rate_limit" || status === 429) {
    return "limits";
  }
  if (type === "security_access" || status === 403 || status === 444) {
    return "access";
  }
  if (type === "security_waf") {
    return "bad_behavior";
  }
  return "unknown";
}

function shouldTreatAsAutoBanEvent(item) {
  const type = String(item?.type || "").trim().toLowerCase();
  if (type === "security_rate_limit" || type === "security_access") {
    return true;
  }
  const status = parseSecurityStatus(item?.details?.status);
  return status === 403 || status === 429 || status === 444;
}

function renderReasonList(values) {
  const list = Array.from(values || []);
  if (!list.length) {
    return "-";
  }
  return list.join(", ");
}

function buildAutoBanRows(events, resolveSiteID, siteBanDurationByID) {
  const grouped = new Map();
  for (const item of normalizeList(events)) {
    if (!shouldTreatAsAutoBanEvent(item)) {
      continue;
    }
    const details = item?.details && typeof item.details === "object" ? item.details : {};
    const ip = normalizeIP(details.client_ip || details.ip);
    if (!ip) {
      continue;
    }
    const rawSiteID = String(item?.site_id || "").trim();
    if (isStartupSelfTestSite(rawSiteID)) {
      continue;
    }
    const siteID = resolveSiteID(rawSiteID, details.host);
    if (!siteID) {
      continue;
    }
    const occurredAt = asDate(item?.occurred_at);
    if (!occurredAt) {
      continue;
    }
    const durationSec = parsePositiveNumber(siteBanDurationByID.get(siteID), 300);
    const key = `${siteID}::${ip}`;
    const current = grouped.get(key) || {
      siteID,
      ip,
      country: normalizeCountryCode(details.country || details.client_country || ""),
      source: "auto",
      occurredAt,
      expiresAt: durationSec === 0 ? null : new Date(occurredAt.getTime() + (durationSec * 1000)),
      modules: new Set(),
      statuses: new Set(),
      reasons: new Set(),
      paths: new Set(),
      hosts: new Set(),
      referers: new Set(),
      userAgents: new Set(),
      eventIDs: new Set(),
      blockedCount: 0,
      latestEvent: item,
      origin: "primary"
    };

    if (!current.occurredAt || occurredAt.getTime() >= current.occurredAt.getTime()) {
      current.occurredAt = occurredAt;
      current.latestEvent = item;
      current.country = normalizeCountryCode(details.country || details.client_country || current.country || "");
      current.expiresAt = durationSec === 0 ? null : new Date(occurredAt.getTime() + (durationSec * 1000));
    }

    const moduleID = deriveModuleFromEvent(item);
    if (moduleID) {
      current.modules.add(moduleID);
    }
    const status = parseSecurityStatus(details.status);
    if (status > 0) {
      current.statuses.add(String(status));
    }
    const reason = String(item?.summary || "").trim();
    if (reason) {
      current.reasons.add(reason);
    }
    const path = String(details.path || details.uri || "").trim();
    if (path) {
      current.paths.add(path);
    }
    const host = String(details.host || "").trim();
    if (host) {
      current.hosts.add(host);
    }
    const referer = String(details.referer || "").trim();
    if (referer) {
      current.referers.add(referer);
    }
    const ua = String(details.user_agent || "").trim();
    if (ua) {
      current.userAgents.add(ua);
    }
    const eventID = String(item?.id || "").trim();
    if (eventID) {
      current.eventIDs.add(eventID);
    }
    current.blockedCount += 1;
    grouped.set(key, current);
  }
  return Array.from(grouped.values());
}

function renderModules(modules, t) {
  return Array.from(modules)
    .map((item) => t(`bans.module.${item}`))
    .join(", ");
}

async function postBanAction(ctx, siteID, origin, action, ip) {
  const base = origin === "secondary" ? "/api-app/sites/" : "/api/sites/";
  const path = `${base}${encodeURIComponent(siteID)}/${action}`;
  if (origin === "secondary") {
    const response = await fetch(path, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ ip })
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `HTTP ${response.status}`);
    }
    return;
  }
  await ctx.api.post(path, { ip });
}

async function addExceptionForSite(ctx, siteID, ip) {
  const policies = normalizeList(await ctx.api.get("/api/access-policies"));
  const existing = policies.find((item) => String(item?.site_id || "").trim() === String(siteID || "").trim()) || null;
  const allowlist = new Set(normalizeList(existing?.allowlist).map((item) => normalizeIP(item)).filter(Boolean));
  allowlist.add(normalizeIP(ip));
  const denylist = normalizeList(existing?.denylist).map((item) => normalizeIP(item)).filter(Boolean);
  const payload = {
    id: String(existing?.id || `${siteID}-access`),
    site_id: String(siteID || "").trim(),
    enabled: true,
    allowlist: Array.from(allowlist),
    denylist
  };
  await ctx.api.post("/api/access-policies/upsert", payload);
}

export async function renderBans(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.bans"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("bans.subtitle"))}</div>
        </div>
        <div class="waf-actions">
          <button class="btn" id="bans-create" type="button">${escapeHtml(ctx.t("bans.action.ban"))}</button>
          <button class="btn ghost btn-sm" id="bans-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
        </div>
      </div>
      <div class="waf-card-body waf-stack">
        <div id="bans-status"></div>
        <div id="bans-list"></div>
      </div>
    </section>
    <div class="waf-modal waf-hidden" id="bans-create-modal" role="dialog" aria-modal="true" aria-labelledby="bans-create-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-create-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-create-title">${escapeHtml(ctx.t("bans.create.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.create.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-create-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-create-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-create-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <select id="bans-create-site"></select>
            </div>
            <div class="waf-field">
              <label for="bans-create-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-create-ip" type="text" placeholder="203.0.113.10">
            </div>
            <div class="waf-field">
              <label for="bans-create-duration">${escapeHtml(ctx.t("bans.create.duration"))}</label>
              <select id="bans-create-duration">
                ${EXTEND_DURATIONS.map((item) => `<option value="${escapeHtml(String(item.seconds))}">${escapeHtml(ctx.t(item.labelKey))}</option>`).join("")}
              </select>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn" id="bans-create-submit" type="button">${escapeHtml(ctx.t("bans.action.ban"))}</button>
            <button class="btn ghost" type="button" data-bans-create-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-detail-modal" role="dialog" aria-modal="true" aria-labelledby="bans-detail-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-detail-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-detail-title">${escapeHtml(ctx.t("events.detail.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("events.detail.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-detail-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body" id="bans-detail-content">
          <div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-extend-modal" role="dialog" aria-modal="true" aria-labelledby="bans-extend-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-extend-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-extend-title">${escapeHtml(ctx.t("bans.extend.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.extend.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-extend-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-extend-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-extend-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <input id="bans-extend-site" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-extend-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-extend-ip" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-extend-duration">${escapeHtml(ctx.t("bans.extend.duration"))}</label>
              <select id="bans-extend-duration">
                ${EXTEND_DURATIONS.map((item) => `<option value="${escapeHtml(String(item.seconds))}">${escapeHtml(ctx.t(item.labelKey))}</option>`).join("")}
              </select>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn success" id="bans-extend-submit" type="button">${escapeHtml(ctx.t("bans.action.extend"))}</button>
            <button class="btn ghost" type="button" data-bans-extend-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="waf-modal waf-hidden" id="bans-unban-modal" role="dialog" aria-modal="true" aria-labelledby="bans-unban-title" tabindex="-1">
      <button class="waf-modal-overlay" type="button" data-bans-unban-close="true" aria-label="${escapeHtml(ctx.t("ui.close"))}"></button>
      <div class="waf-modal-card">
        <div class="waf-card-head">
          <div>
            <h3 id="bans-unban-title">${escapeHtml(ctx.t("bans.unban.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("bans.unban.subtitle"))}</div>
          </div>
          <button class="btn ghost btn-sm" type="button" data-bans-unban-close="true">${escapeHtml(ctx.t("ui.close"))}</button>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="bans-unban-status"></div>
          <div class="waf-form-grid three">
            <div class="waf-field">
              <label for="bans-unban-site">${escapeHtml(ctx.t("bans.col.site"))}</label>
              <input id="bans-unban-site" type="text" readonly>
            </div>
            <div class="waf-field">
              <label for="bans-unban-ip">${escapeHtml(ctx.t("bans.col.ip"))}</label>
              <input id="bans-unban-ip" type="text" readonly>
            </div>
          </div>
          <div class="waf-actions">
            <button class="btn danger" id="bans-unban-submit" type="button">${escapeHtml(ctx.t("bans.action.unban"))}</button>
            <button class="btn ghost" type="button" data-bans-unban-close="true">${escapeHtml(ctx.t("common.cancel"))}</button>
          </div>
        </div>
      </div>
    </div>
  `;

  const statusNode = container.querySelector("#bans-status");
  const listNode = container.querySelector("#bans-list");
  const createModalNode = container.querySelector("#bans-create-modal");
  const createStatusNode = container.querySelector("#bans-create-status");
  const createSiteNode = container.querySelector("#bans-create-site");
  const createIPNode = container.querySelector("#bans-create-ip");
  const createDurationNode = container.querySelector("#bans-create-duration");
  const createSubmitNode = container.querySelector("#bans-create-submit");
  const detailModalNode = container.querySelector("#bans-detail-modal");
  const detailContentNode = container.querySelector("#bans-detail-content");
  const extendModalNode = container.querySelector("#bans-extend-modal");
  const extendStatusNode = container.querySelector("#bans-extend-status");
  const extendSiteNode = container.querySelector("#bans-extend-site");
  const extendIPNode = container.querySelector("#bans-extend-ip");
  const extendDurationNode = container.querySelector("#bans-extend-duration");
  const extendSubmitNode = container.querySelector("#bans-extend-submit");
  const unbanModalNode = container.querySelector("#bans-unban-modal");
  const unbanStatusNode = container.querySelector("#bans-unban-status");
  const unbanSiteNode = container.querySelector("#bans-unban-site");
  const unbanIPNode = container.querySelector("#bans-unban-ip");
  const unbanSubmitNode = container.querySelector("#bans-unban-submit");
  let latestSiteIDs = [];
  let extendDraft = null;
  let unbanDraft = null;
  const pagingState = {
    pageSize: 10,
    page: 1
  };

  const getPaginationMeta = (total) => {
    const totalPages = Math.max(1, Math.ceil(total / pagingState.pageSize));
    if (pagingState.page > totalPages) {
      pagingState.page = totalPages;
    }
    if (pagingState.page < 1) {
      pagingState.page = 1;
    }
    const start = total === 0 ? 0 : (pagingState.page - 1) * pagingState.pageSize;
    const end = Math.min(start + pagingState.pageSize, total);
    return { totalPages, start, end };
  };

  const closeDetail = () => {
    detailModalNode?.classList.add("waf-hidden");
  };
  const openCreateModal = () => {
    createStatusNode.innerHTML = "";
    if (createSiteNode && latestSiteIDs.length && !createSiteNode.value) {
      createSiteNode.value = latestSiteIDs[0];
    }
    if (createDurationNode && !createDurationNode.value) {
      createDurationNode.value = String(EXTEND_DURATIONS[0]?.seconds || 0);
    }
    createModalNode?.classList.remove("waf-hidden");
    createIPNode?.focus();
  };
  const closeCreateModal = () => {
    createModalNode?.classList.add("waf-hidden");
  };
  const closeExtendModal = () => {
    extendDraft = null;
    if (extendStatusNode) {
      extendStatusNode.innerHTML = "";
    }
    extendModalNode?.classList.add("waf-hidden");
  };
  const openExtendModal = (row, siteLabel) => {
    if (!row || !extendModalNode) {
      return;
    }
    extendDraft = { row };
    if (extendSiteNode) {
      extendSiteNode.value = String(siteLabel || row.siteID || "").trim();
    }
    if (extendIPNode) {
      extendIPNode.value = String(row.ip || "").trim();
    }
    if (extendDurationNode) {
      const current = row.expiresAt ? Math.max(0, Math.round((row.expiresAt.getTime() - Date.now()) / 1000)) : 0;
      const closest = EXTEND_DURATIONS.find((item) => item.seconds === 0 ? current <= 0 : current <= item.seconds) || EXTEND_DURATIONS[0];
      extendDurationNode.value = String(closest.seconds);
    }
    extendStatusNode.innerHTML = "";
    extendModalNode.classList.remove("waf-hidden");
    extendDurationNode?.focus();
  };
  const closeUnbanModal = () => {
    unbanDraft = null;
    if (unbanStatusNode) {
      unbanStatusNode.innerHTML = "";
    }
    unbanModalNode?.classList.add("waf-hidden");
  };
  const openUnbanModal = (row, siteLabel) => {
    if (!row || !unbanModalNode) {
      return;
    }
    unbanDraft = { row };
    if (unbanSiteNode) {
      unbanSiteNode.value = String(siteLabel || row.siteID || "").trim();
    }
    if (unbanIPNode) {
      unbanIPNode.value = String(row.ip || "").trim();
    }
    if (unbanStatusNode) {
      unbanStatusNode.innerHTML = "";
    }
    unbanModalNode.classList.remove("waf-hidden");
    unbanSubmitNode?.focus();
  };

  detailModalNode?.querySelectorAll("[data-bans-detail-close]").forEach((node) => {
    node.addEventListener("click", closeDetail);
  });
  detailModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeDetail();
    }
  });
  createModalNode?.querySelectorAll("[data-bans-create-close]").forEach((node) => {
    node.addEventListener("click", closeCreateModal);
  });
  createModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeCreateModal();
    }
  });
  extendModalNode?.querySelectorAll("[data-bans-extend-close]").forEach((node) => {
    node.addEventListener("click", closeExtendModal);
  });
  extendModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeExtendModal();
    }
  });
  unbanModalNode?.querySelectorAll("[data-bans-unban-close]").forEach((node) => {
    node.addEventListener("click", closeUnbanModal);
  });
  unbanModalNode?.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeUnbanModal();
    }
  });

  const renderCreateSiteOptions = (sites) => {
    if (!createSiteNode) {
      return;
    }
    const selected = String(createSiteNode.value || "").trim();
    const options = normalizeList(sites)
      .map((site) => {
        const id = String(site?.id || "").trim();
        if (!id) {
          return null;
        }
        if (isStartupSelfTestSite(id)) {
          return null;
        }
        const label = String(site?.primary_host || id).trim() || id;
        return { id, label };
      })
      .filter(Boolean)
      .sort((left, right) => left.label.localeCompare(right.label, undefined, { sensitivity: "base" }));
    latestSiteIDs = options.map((item) => item.id);
    createSiteNode.innerHTML = [
      `<option value="${ALL_SERVICES_SITE_ID}">${escapeHtml(ctx.t("bans.site.allServices"))}</option>`,
      ...options.map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.label)}</option>`)
    ].join("");
    if (selected && latestSiteIDs.includes(selected)) {
      createSiteNode.value = selected;
      return;
    }
    createSiteNode.value = ALL_SERVICES_SITE_ID;
  };

  const renderDetail = (row, siteLabel) => {
    if (!detailContentNode || !row) {
      return;
    }
    const detailRows = [
      ["events.detail.field.time", row.occurredAt ? row.occurredAt.toISOString() : "-"],
      ["events.detail.field.site", siteLabel || row.siteID || "-"],
      ["bans.col.ip", row.ip || "-"],
      ["bans.col.country", countryFlagEmoji(row.country)],
      ["bans.col.module", renderModules(row.modules || new Set(), ctx.t) || "-"],
      ["events.detail.field.summary", renderReasonList(row.reasons || new Set())],
      ["bans.col.left", row.expiresAt ? formatRemaining(row.expiresAt, new Date(), ctx.t) : ctx.t("bans.time.permanent")]
    ];
    if (row.blockedCount > 0) {
      detailRows.push(["dashboard.detail.blocked", String(row.blockedCount)]);
    }
    if (row.statuses?.size) {
      detailRows.push(["events.col.status", renderReasonList(row.statuses)]);
    }
    const latestRaw = row.latestEvent?.details && typeof row.latestEvent.details === "object"
      ? row.latestEvent.details
      : {};
    const eventMeta = {
      event_ids: Array.from(row.eventIDs || []),
      paths: Array.from(row.paths || []),
      hosts: Array.from(row.hosts || []),
      referers: Array.from(row.referers || []),
      user_agents: Array.from(row.userAgents || []),
      latest_event: latestRaw
    };
    detailContentNode.innerHTML = `
      <div class="waf-table-wrap">
        <table class="waf-table waf-detail-table">
          <tbody>
            ${detailRows.map(([labelKey, value]) => `
              <tr>
                <th>${escapeHtml(ctx.t(labelKey) === labelKey ? labelKey : ctx.t(labelKey))}</th>
                <td><pre class="waf-code">${escapeHtml(String(value || "-"))}</pre></td>
              </tr>
            `).join("")}
            <tr>
              <th>${escapeHtml(ctx.t("events.detail.field.details"))}</th>
              <td><pre class="waf-code">${escapeHtml(JSON.stringify(eventMeta, null, 2))}</pre></td>
            </tr>
          </tbody>
        </table>
      </div>
    `;
    detailModalNode?.classList.remove("waf-hidden");
  };

  const loadSiteBanDurations = async (sites, resolveSiteID) => {
    const entries = await Promise.all(
      normalizeList(sites).map(async (site) => {
        const candidate = String(site?.id || "").trim();
        if (!candidate) {
          return null;
        }
        try {
          const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(candidate)}`);
          const whitelist = normalizeList(profile?.security_country_policy?.whitelist_country)
            .map((item) => String(item || "").trim())
            .filter(Boolean);
          if (whitelist.length > 0) {
            return [resolveSiteID(candidate), 0];
          }
          const duration = parsePositiveNumber(profile?.security_behavior_and_limits?.bad_behavior_ban_time_seconds, 300);
          return [resolveSiteID(candidate), duration];
        } catch (_error) {
          return [resolveSiteID(candidate), 300];
        }
      })
    );
    const out = new Map();
    for (const item of entries) {
      if (!item || !item[0]) {
        continue;
      }
      out.set(item[0], item[1]);
    }
    return out;
  };

  const renderRows = async () => {
    setLoading(statusNode, ctx.t("bans.loading"));
    try {
      const [sitesResponse, accessResponse, eventsResponse, sitesSecondary, accessSecondary, eventsSecondary] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/access-policies"),
        ctx.api.get("/api/events"),
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/access-policies"),
        tryGetJSON("/api-app/events")
      ]);

      const sites = mergeByID(sitesResponse, unwrapList(sitesSecondary, ["sites"]), "id");
      // /api/access-policies returns an array in current backend contract.
      // Keep backward compatibility with older wrapped payloads too.
      const accessPolicies = mergeByID(
        unwrapList(accessResponse, ["access_policies"]),
        unwrapList(accessSecondary, ["access_policies"]),
        "id"
      );
      const events = mergeByID(unwrapList(eventsResponse, ["events", "items"]), unwrapList(eventsSecondary, ["events", "items"]), "id");

      const siteByID = new Map(sites.map((site) => [String(site?.id || ""), site]));
      renderCreateSiteOptions(sites);
      const siteAliasMap = buildSiteAliasMap(sites);
      const hostSiteMap = buildHostSiteMap(sites);
      const canonicalSiteID = (value, hostHint = "") => resolveCanonicalSiteID(value, siteAliasMap, hostSiteMap, hostHint);
      const allowlistBySite = buildIPSetBySite(accessPolicies, "allowlist", canonicalSiteID);
      const siteBanDurationByID = await loadSiteBanDurations(sites, canonicalSiteID);
      const manualBanTimers = loadManualBanTimers();
      await processExpiredManualBanTimers(ctx, manualBanTimers);
      const manualRows = buildManualBanRows(accessPolicies, canonicalSiteID, manualBanTimers);
      const autoRows = buildAutoBanRows(events, canonicalSiteID, siteBanDurationByID);
      const rows = mergeBanRows(manualRows, autoRows).sort((a, b) => {
        const left = a.occurredAt ? a.occurredAt.getTime() : 0;
        const right = b.occurredAt ? b.occurredAt.getTime() : 0;
        return right - left;
      });

      if (rows.length === 0) {
        statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.empty"))}</div>`;
        listNode.innerHTML = "";
        return;
      }

      statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.total"))}: ${rows.length}</div>`;

      const now = new Date();
      const meta = getPaginationMeta(rows.length);
      const pageRows = rows.slice(meta.start, meta.end);

      listNode.innerHTML = `
        <div class="waf-table-wrap">
          <table class="waf-table">
            <thead>
              <tr>
                <th>${escapeHtml(ctx.t("bans.col.ip"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.country"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.date"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.left"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.site"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.module"))}</th>
                <th>${escapeHtml(ctx.t("bans.col.actions"))}</th>
              </tr>
            </thead>
            <tbody>
              ${pageRows.map((row, index) => {
                const site = siteByID.get(row.siteID);
                const siteLabel = site?.primary_host || row.siteID;
                const allowlisted = (allowlistBySite.get(row.siteID) || new Set()).has(row.ip);
                return `
                  <tr class="waf-table-row-clickable" data-ban-row="${meta.start + index}" tabindex="0" role="button">
                    <td>${escapeHtml(row.ip)}</td>
                    <td>${escapeHtml(countryFlagEmoji(row.country))}</td>
                    <td>${escapeHtml(row.occurredAt ? formatDate(row.occurredAt.toISOString()) : "-")}</td>
                    <td>${escapeHtml(formatRemaining(row.expiresAt, now, ctx.t))}</td>
                    <td>${escapeHtml(siteLabel)}</td>
                    <td>${escapeHtml(renderModules(row.modules, ctx.t))}${allowlisted ? " (allowlist)" : ""}</td>
                    <td>
                      <div class="waf-ban-actions">
                        <button type="button" class="btn success btn-sm waf-ban-btn" data-action="extend" data-row="${meta.start + index}" title="${escapeHtml(ctx.t("bans.action.extend"))}">
                          <span class="waf-ban-btn-icon">${ICON_PLUS}</span>
                          <span>${escapeHtml(ctx.t("bans.action.extend"))}</span>
                        </button>
                        <button type="button" class="btn danger btn-sm waf-ban-btn" data-action="unban" data-row="${meta.start + index}" title="${escapeHtml(ctx.t("bans.action.unban"))}">
                          <span class="waf-ban-btn-icon">${ICON_UNLOCK}</span>
                          <span>${escapeHtml(ctx.t("bans.action.unban"))}</span>
                        </button>
                      </div>
                    </td>
                  </tr>
                `;
              }).join("")}
            </tbody>
          </table>
        </div>
        <div class="waf-pager">
          <div class="waf-inline">
            <label for="bans-page-size">${escapeHtml(ctx.t("activity.filter.pageSize"))}</label>
            <select id="bans-page-size">
              <option value="10"${pagingState.pageSize === 10 ? " selected" : ""}>10</option>
              <option value="25"${pagingState.pageSize === 25 ? " selected" : ""}>25</option>
              <option value="50"${pagingState.pageSize === 50 ? " selected" : ""}>50</option>
              <option value="100"${pagingState.pageSize === 100 ? " selected" : ""}>100</option>
            </select>
          </div>
          <div class="waf-actions">
            ${buildPageButtons(meta.totalPages, pagingState.page, "data-bans-page")}
          </div>
        </div>
      `;

      listNode.querySelector("#bans-page-size")?.addEventListener("change", async (event) => {
        const nextSize = Number.parseInt(String(event.target?.value || "10"), 10);
        if (!Number.isFinite(nextSize) || nextSize <= 0) {
          return;
        }
        pagingState.pageSize = nextSize;
        pagingState.page = 1;
        await renderRows();
      });

      listNode.querySelectorAll("[data-bans-page]").forEach((button) => {
        button.addEventListener("click", async () => {
          const nextPage = Number.parseInt(String(button.dataset.bansPage || "1"), 10);
          if (!Number.isFinite(nextPage) || nextPage < 1) {
            return;
          }
          pagingState.page = nextPage;
          await renderRows();
        });
      });

      listNode.querySelectorAll(".waf-ban-btn").forEach((button) => {
        button.addEventListener("click", async () => {
          button.blur();
          const action = String(button.dataset.action || "");
          const rowIndex = Number.parseInt(String(button.dataset.row || "-1"), 10);
          const row = rows[rowIndex];
          if (!row || !row.siteID || !row.ip) {
            return;
          }
          const ip = normalizeIP(row.ip);
          const isAllowlisted = (allowlistBySite.get(row.siteID) || new Set()).has(ip);
          if (isAllowlisted) {
            ctx.notify("IP is in allowlist for this site; no ban actions applied.");
            return;
          }
          if (action === "unban") {
            const site = siteByID.get(row.siteID);
            openUnbanModal(row, site?.primary_host || row.siteID);
            return;
          }

          const site = siteByID.get(row.siteID);
          openExtendModal(row, site?.primary_host || row.siteID);
        });
      });

      listNode.querySelectorAll("[data-ban-row]").forEach((rowNode) => {
        const open = () => {
          const rowIndex = Number.parseInt(String(rowNode.getAttribute("data-ban-row") || "-1"), 10);
          const row = rows[rowIndex];
          if (!row) {
            return;
          }
          const site = siteByID.get(row.siteID);
          const siteLabel = site?.primary_host || row.siteID;
          renderDetail(row, siteLabel);
        };
        rowNode.addEventListener("click", (event) => {
          if (event.target instanceof HTMLElement && event.target.closest(".waf-ban-btn")) {
            return;
          }
          open();
        });
        rowNode.addEventListener("keydown", (event) => {
          if (event.key !== "Enter" && event.key !== " ") {
            return;
          }
          event.preventDefault();
          open();
        });
      });
    } catch (_error) {
      setError(statusNode, ctx.t("bans.error.load"));
      listNode.innerHTML = "";
    }
  };

  container.querySelector("#bans-refresh")?.addEventListener("click", () => {
    renderRows();
  });
  container.querySelector("#bans-create")?.addEventListener("click", () => {
    openCreateModal();
  });
  createSubmitNode?.addEventListener("click", async () => {
    const siteID = String(createSiteNode?.value || "").trim();
    const ip = normalizeIP(createIPNode?.value || "");
    const durationSec = Number.parseInt(String(createDurationNode?.value || "0"), 10);
    if (!siteID || !ip) {
      setError(createStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    if (!Number.isFinite(durationSec) || durationSec < 0) {
      setError(createStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    try {
      createSubmitNode.disabled = true;
      const targets = siteID === ALL_SERVICES_SITE_ID ? [...latestSiteIDs] : [siteID];
      if (!targets.length) {
        setError(createStatusNode, ctx.t("bans.error.createValidation"));
        return;
      }
      await Promise.all(targets.map((targetSiteID) => postBanAction(ctx, targetSiteID, "primary", "ban", ip)));
      const timers = loadManualBanTimers();
      for (const targetSiteID of targets) {
        if (durationSec > 0) {
          upsertManualBanTimer(timers, targetSiteID, ip, new Date(Date.now() + (durationSec * 1000)));
        } else {
          removeManualBanTimer(timers, targetSiteID, ip);
        }
      }
      saveManualBanTimers(timers);
      createIPNode.value = "";
      createDurationNode.value = String(EXTEND_DURATIONS[0]?.seconds || 0);
      closeCreateModal();
      ctx.notify(ctx.t("toast.ipBanned"));
      await renderRows();
    } catch (error) {
      setError(createStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      createSubmitNode.disabled = false;
    }
  });

  extendSubmitNode?.addEventListener("click", async () => {
    const row = extendDraft?.row;
    if (!row || !row.siteID || !row.ip) {
      setError(extendStatusNode, ctx.t("bans.error.action"));
      return;
    }
    const durationSec = Number.parseInt(String(extendDurationNode?.value || "0"), 10);
    if (!Number.isFinite(durationSec) || durationSec < 0) {
      setError(extendStatusNode, ctx.t("bans.error.createValidation"));
      return;
    }
    const ip = normalizeIP(row.ip);
    try {
      extendSubmitNode.disabled = true;
      await postBanAction(ctx, row.siteID, "primary", "ban", ip);
      const timers = loadManualBanTimers();
      if (durationSec > 0) {
        const currentTimer = timers.get(manualBanTimerRowKey(row.siteID, ip));
        const currentUnbanAt = Date.parse(String(currentTimer?.unbanAt || row.expiresAt?.toISOString?.() || ""));
        const startFrom = Number.isNaN(currentUnbanAt) ? Date.now() : Math.max(Date.now(), currentUnbanAt);
        upsertManualBanTimer(timers, row.siteID, ip, new Date(startFrom + (durationSec * 1000)));
      } else {
        removeManualBanTimer(timers, row.siteID, ip);
      }
      saveManualBanTimers(timers);
      closeExtendModal();
      ctx.notify(ctx.t("toast.ipBanned"));
      await renderRows();
    } catch (error) {
      setError(extendStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      extendSubmitNode.disabled = false;
    }
  });

  unbanSubmitNode?.addEventListener("click", async () => {
    const row = unbanDraft?.row;
    if (!row || !row.siteID || !row.ip) {
      setError(unbanStatusNode, ctx.t("bans.error.action"));
      return;
    }
    const ip = normalizeIP(row.ip);
    try {
      unbanSubmitNode.disabled = true;
      if (row.source === "manual") {
        await postBanAction(ctx, row.siteID, "primary", "unban", ip);
        const timers = loadManualBanTimers();
        removeManualBanTimer(timers, row.siteID, ip);
        saveManualBanTimers(timers);
      } else {
        await addExceptionForSite(ctx, row.siteID, ip);
      }
      clearRateLimitCookies(row.siteID);
      closeUnbanModal();
      ctx.notify(ctx.t("toast.ipUnbanned"));
      await renderRows();
    } catch (error) {
      setError(unbanStatusNode, error?.message || ctx.t("bans.error.action"));
    } finally {
      unbanSubmitNode.disabled = false;
    }
  });

  await renderRows();
}
