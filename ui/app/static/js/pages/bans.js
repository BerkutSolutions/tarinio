import { confirmAction, escapeHtml, formatDate, setError, setLoading } from "../ui.js";

const AUTO_BAN_EVENT_TYPES = new Set(["security_rate_limit", "security_access", "security_waf"]);
const AUTO_BAN_FALLBACK_SECONDS = 300;
const DAY_BAN_SECONDS = 24 * 60 * 60;
const ESCALATION_STORAGE_KEY = "waf_ban_escalation_v1";

const ICON_PLUS = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M11 5h2v14h-2zM5 11h14v2H5z"/></svg>';
const ICON_UNLOCK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a5 5 0 0 1 5 5v2h-2V7a3 3 0 1 0-6 0v2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-9a2 2 0 0 1 2-2h2V7a5 5 0 0 1 5-5Z"/></svg>';

function loadEscalationState() {
  try {
    const parsed = JSON.parse(window.localStorage.getItem(ESCALATION_STORAGE_KEY) || "{}");
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch (_error) {
    return {};
  }
}

function saveEscalationState(state) {
  window.localStorage.setItem(ESCALATION_STORAGE_KEY, JSON.stringify(state || {}));
}

function getEscalationRecord(state, ip) {
  const key = String(ip || "").trim();
  if (!key) {
    return null;
  }
  if (!state[key] || typeof state[key] !== "object") {
    state[key] = {
      level: 0,
      last_unban_ms: 0,
      last_promoted_token: "",
      last_enforced_level: 0
    };
  }
  return state[key];
}

function normalizeIP(value) {
  return String(value || "").trim();
}

function normalizeBanStages(values) {
  const out = [];
  for (const raw of normalizeList(values)) {
    const value = Number.parseInt(String(raw), 10);
    if (!Number.isFinite(value) || value < 0) {
      continue;
    }
    out.push(value);
    if (value === 0) {
      break;
    }
  }
  return out;
}

function buildBanPolicy(profile) {
  const sec = profile?.security_behavior_and_limits || {};
  const base = Number.parseInt(String(sec?.bad_behavior_ban_time_seconds ?? AUTO_BAN_FALLBACK_SECONDS), 10);
  const baseSeconds = Number.isFinite(base) ? Math.max(0, base) : AUTO_BAN_FALLBACK_SECONDS;
  const enabled = Boolean(sec?.ban_escalation_enabled);
  const scopeRaw = String(sec?.ban_escalation_scope || "").trim().toLowerCase();
  const scope = enabled && scopeRaw === "current_site" ? "current_site" : "all_sites";
  const customStages = normalizeBanStages(sec?.ban_escalation_stages_seconds);
  const stages = enabled && customStages.length
    ? customStages
    : [baseSeconds, DAY_BAN_SECONDS, 0];
  return {
    enabled,
    scope,
    stages,
    baseSeconds
  };
}

function getPolicyForSite(siteID, policyBySite) {
  const key = String(siteID || "").trim();
  if (!key) {
    return { enabled: false, scope: "all_sites", stages: [AUTO_BAN_FALLBACK_SECONDS, DAY_BAN_SECONDS, 0], baseSeconds: AUTO_BAN_FALLBACK_SECONDS };
  }
  return policyBySite.get(key) || { enabled: false, scope: "all_sites", stages: [AUTO_BAN_FALLBACK_SECONDS, DAY_BAN_SECONDS, 0], baseSeconds: AUTO_BAN_FALLBACK_SECONDS };
}

function normalizeList(value) {
  return Array.isArray(value) ? value : [];
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
  } catch (error) {
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

function clearRateLimitCookie(siteID) {
  const cookie = siteCookieName(siteID);
  document.cookie = `${cookie}=; Path=/; Max-Age=0; SameSite=Lax`;
}

function parseIP(details) {
  if (!details || typeof details !== "object") {
    return "";
  }
  return String(details.client_ip || details.ip || "").trim();
}

function parseCountry(details) {
  if (!details || typeof details !== "object") {
    return "-";
  }
  return String(details.country || details.client_country || details.geo_country || details.country_code || "").trim() || "-";
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

function moduleByEvent(item) {
  const type = String(item?.type || "").trim().toLowerCase();
  if (type === "security_rate_limit") {
    return "limits";
  }
  if (type === "security_waf") {
    return "ddos";
  }
  if (type === "security_access") {
    const status = Number(item?.details?.status || 0);
    if (status === 429) {
      return "bad_behavior";
    }
    if (status === 403 || status === 444) {
      return "headers_access";
    }
    return "access";
  }
  return "unknown";
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

function buildAutoBanRows(events, policyBySite, resolveSiteID, allowlistBySite, escalationState) {
  const now = new Date();
  const byKey = new Map();
  for (const item of normalizeList(events)) {
    const type = String(item?.type || "").trim().toLowerCase();
    if (!AUTO_BAN_EVENT_TYPES.has(type)) {
      continue;
    }
    const details = item?.details && typeof item.details === "object" ? item.details : {};
    const siteID = resolveSiteID(String(item?.site_id || "").trim(), String(details.host || "").trim());
    if (!siteID) {
      continue;
    }
    const ip = parseIP(details);
    if (!ip) {
      continue;
    }
    if ((allowlistBySite.get(siteID) || new Set()).has(ip)) {
      continue;
    }
    const occurredAt = asDate(item?.occurred_at);
    if (!occurredAt) {
      continue;
    }
    const key = `${siteID}|${ip}`;
    const moduleID = moduleByEvent(item);
    const policy = getPolicyForSite(siteID, policyBySite);
    const rec = getEscalationRecord(escalationState, ip);
    const level = Math.max(0, Math.min(Number(rec?.level || 0), Math.max(0, policy.stages.length - 1)));
    const banSeconds = Number(policy.stages[level] ?? policy.baseSeconds ?? AUTO_BAN_FALLBACK_SECONDS);
    const expiresAt = banSeconds === 0 ? null : new Date(occurredAt.getTime() + banSeconds * 1000);

    if (expiresAt && expiresAt.getTime() <= now.getTime()) {
      continue;
    }

    const existing = byKey.get(key);
    if (!existing || occurredAt.getTime() > existing.occurredAt.getTime()) {
      byKey.set(key, {
        siteID,
        ip,
        country: parseCountry(details),
        source: "auto",
        occurredAt,
        expiresAt,
        policyScope: policy.scope,
        banStages: policy.stages,
        modules: new Set([moduleID]),
        origin: item?._origin || "primary"
      });
      continue;
    }
    existing.modules.add(moduleID);
  }
  return Array.from(byKey.values());
}

function buildManualBanRows(accessPolicies, resolveSiteID) {
  const rows = [];
  for (const policy of normalizeList(accessPolicies)) {
    const siteID = resolveSiteID(String(policy?.site_id || "").trim());
    if (!siteID) {
      continue;
    }
    const updatedAt = asDate(policy?.updated_at);
    for (const ip of normalizeList(policy?.denylist)) {
      const value = String(ip || "").trim();
      if (!value) {
        continue;
      }
      rows.push({
        siteID,
        ip: value,
        country: "-",
        source: "manual",
        occurredAt: updatedAt,
        expiresAt: null,
        modules: new Set(["manual"]),
        origin: policy?._origin || "primary"
      });
    }
  }
  return rows;
}

function mergeRows(autoRows, manualRows) {
  const merged = new Map();
  for (const item of [...autoRows, ...manualRows]) {
    const key = `${item.origin}|${item.siteID}|${item.ip}`;
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, item);
      continue;
    }
    if (item.source === "manual") {
      merged.set(key, { ...existing, ...item, source: "manual", expiresAt: null, modules: new Set([...existing.modules, ...item.modules]) });
      continue;
    }
    if (!existing.occurredAt || (item.occurredAt && item.occurredAt.getTime() > existing.occurredAt.getTime())) {
      merged.set(key, { ...existing, ...item, modules: new Set([...existing.modules, ...item.modules]) });
      continue;
    }
    existing.modules = new Set([...existing.modules, ...item.modules]);
  }
  return Array.from(merged.values()).sort((a, b) => {
    const left = a.occurredAt ? a.occurredAt.getTime() : 0;
    const right = b.occurredAt ? b.occurredAt.getTime() : 0;
    return right - left;
  });
}

function renderModules(modules, t) {
  return Array.from(modules)
    .map((item) => t(`bans.module.${item}`))
    .join(", ");
}

function applyEscalationLevel(rows, escalationState, policyBySite) {
  let changed = false;
  for (const row of rows) {
    if (row.source !== "auto" || !row.ip) {
      continue;
    }
    const rec = getEscalationRecord(escalationState, row.ip);
    if (!rec) {
      continue;
    }
    const occurredMs = Number(row?.occurredAt?.getTime?.() || 0);
    const lastUnban = Number(rec.last_unban_ms || 0);
    if (occurredMs > 0 && lastUnban > 0 && occurredMs > lastUnban) {
      const promoteToken = `${row.siteID}|${occurredMs}`;
      if (rec.last_promoted_token !== promoteToken) {
        const policy = getPolicyForSite(row.siteID, policyBySite);
        rec.level = Math.min(Math.max(0, policy.stages.length - 1), Number(rec.level || 0) + 1);
        rec.last_promoted_token = promoteToken;
        changed = true;
      }
    }
    row.escalationLevel = Number(rec.level || 0);
    const policy = getPolicyForSite(row.siteID, policyBySite);
    const cappedLevel = Math.max(0, Math.min(row.escalationLevel, Math.max(0, policy.stages.length - 1)));
    row.escalationLevel = cappedLevel;
    const stageSeconds = Number(policy.stages[cappedLevel] ?? policy.baseSeconds ?? AUTO_BAN_FALLBACK_SECONDS);
    if (stageSeconds === 0) {
      row.expiresAt = null;
      row.modules.add("manual");
    } else {
      const startMs = occurredMs || Date.now();
      row.expiresAt = new Date(startMs + stageSeconds * 1000);
    }
  }
  return changed;
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

async function applyBanAllSites(ctx, sites, ip, action, allowlistBySite) {
  for (const site of normalizeList(sites)) {
    const siteID = String(site?.id || "").trim();
    if (!siteID) {
      continue;
    }
    if ((allowlistBySite.get(siteID) || new Set()).has(ip)) {
      continue;
    }
    await postBanAction(ctx, siteID, String(site?._origin || "primary"), action, ip);
  }
}

async function enforceEscalation(rows, sites, escalationState, ctx, allowlistBySite, denylistBySite, policyBySite) {
  let applied = false;
  for (const row of rows) {
    if (row.source !== "auto") {
      continue;
    }
    const ip = normalizeIP(row.ip);
    if (!ip) {
      continue;
    }
    if ((allowlistBySite.get(row.siteID) || new Set()).has(ip)) {
      continue;
    }
    if ((denylistBySite.get(row.siteID) || new Set()).has(ip)) {
      continue;
    }
    const rec = getEscalationRecord(escalationState, ip);
    if (!rec) {
      continue;
    }
    const policy = getPolicyForSite(row.siteID, policyBySite);
    const level = Number(rec.level || 0);
    if (level <= Number(rec.last_enforced_level || 0)) {
      continue;
    }
    const stageSeconds = Number(policy.stages[Math.max(0, Math.min(level, policy.stages.length - 1))] ?? policy.baseSeconds ?? AUTO_BAN_FALLBACK_SECONDS);
    if (stageSeconds !== 0) {
      // Finite stages are represented in ban timers; hard denylist ban is only for permanent stage.
      rec.last_enforced_level = level;
      applied = true;
      continue;
    }
    if (policy.scope === "all_sites") {
      await applyBanAllSites(ctx, sites, ip, "ban", allowlistBySite);
      rec.last_enforced_level = level;
      applied = true;
      continue;
    }
    await postBanAction(ctx, row.siteID, row.origin, "ban", ip);
    rec.last_enforced_level = level;
    applied = true;
  }
  return applied;
}

export async function renderBans(container, ctx) {
  container.innerHTML = `
    <section class="waf-card">
      <div class="waf-card-head">
        <div>
          <h3>${escapeHtml(ctx.t("app.bans"))}</h3>
          <div class="muted">${escapeHtml(ctx.t("bans.subtitle"))}</div>
        </div>
        <button class="btn ghost btn-sm" id="bans-refresh" type="button">${escapeHtml(ctx.t("common.refresh"))}</button>
      </div>
      <div class="waf-card-body waf-stack">
        <div id="bans-status"></div>
        <div id="bans-list"></div>
      </div>
    </section>
  `;

  const statusNode = container.querySelector("#bans-status");
  const listNode = container.querySelector("#bans-list");

  const loadBanPoliciesBySite = async (siteIDs) => {
    const out = new Map();
    const secondaryProfileDump = await tryGetJSON("/api-app/easy-site-profiles");
    const secondaryProfiles = unwrapList(secondaryProfileDump, ["easy_site_profiles"]);
    const secondaryBySite = new Map(secondaryProfiles.map((item) => [String(item?.site_id || "").trim(), item]));
    await Promise.all(siteIDs.map(async (siteID) => {
      try {
        const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`);
        out.set(siteID, buildBanPolicy(profile));
      } catch (error) {
        const secondary = secondaryBySite.get(siteID);
        out.set(siteID, buildBanPolicy(secondary || null));
      }
    }));
    return out;
  };

  const renderRows = async (skipAutoEnforce = false) => {
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
      const accessPolicies = mergeByID(accessResponse?.access_policies, unwrapList(accessSecondary, ["access_policies"]), "id");
      const events = [
        ...normalizeList(eventsResponse?.events).map((item) => ({ ...item, _origin: "primary" })),
        ...unwrapList(eventsSecondary, ["events"]).map((item) => ({ ...item, _origin: "secondary" }))
      ];

      const siteByID = new Map(sites.map((site) => [String(site?.id || ""), site]));
      const siteAliasMap = buildSiteAliasMap(sites);
      const hostSiteMap = buildHostSiteMap(sites);
      const canonicalSiteID = (value, hostHint = "") => resolveCanonicalSiteID(value, siteAliasMap, hostSiteMap, hostHint);
      const allowlistBySite = buildIPSetBySite(accessPolicies, "allowlist", canonicalSiteID);
      const denylistBySite = buildIPSetBySite(accessPolicies, "denylist", canonicalSiteID);
      const policyBySite = await loadBanPoliciesBySite(
        Array.from(new Set([
          ...sites.map((site) => canonicalSiteID(site?.id)).filter(Boolean),
          ...events.map((item) => canonicalSiteID(item?.site_id, item?.details?.host)).filter(Boolean),
          ...accessPolicies.map((item) => canonicalSiteID(item?.site_id)).filter(Boolean)
        ]))
      );

      const escalationState = loadEscalationState();
      const rows = mergeRows(
        buildAutoBanRows(events, policyBySite, canonicalSiteID, allowlistBySite, escalationState),
        buildManualBanRows(accessPolicies, canonicalSiteID)
      );
      const escalationChanged = applyEscalationLevel(rows, escalationState, policyBySite);
      if (escalationChanged) {
        saveEscalationState(escalationState);
      }
      if (!skipAutoEnforce) {
        const enforced = await enforceEscalation(rows, sites, escalationState, ctx, allowlistBySite, denylistBySite, policyBySite);
        if (enforced) {
          saveEscalationState(escalationState);
          await renderRows(true);
          return;
        }
      }
      if (rows.length === 0) {
        statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.empty"))}</div>`;
        listNode.innerHTML = "";
        return;
      }

      statusNode.innerHTML = `<div class="waf-empty">${escapeHtml(ctx.t("bans.total"))}: ${rows.length}</div>`;

      const now = new Date();
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
              ${rows.map((row, index) => {
                const site = siteByID.get(row.siteID);
                const siteLabel = site?.primary_host || row.siteID;
                const stageSeconds = Number((row.banStages || [])[Math.max(0, Number(row.escalationLevel || 0))] ?? -1);
                const escalationLabel = stageSeconds === 0
                  ? (row.policyScope === "all_sites" ? " [GLOBAL PERM]" : " [SITE PERM]")
                  : Number(row.escalationLevel || 0) > 0
                    ? ` [L${Number(row.escalationLevel || 0) + 1}]`
                    : "";
                return `
                  <tr>
                    <td>${escapeHtml(row.ip)}</td>
                    <td>${escapeHtml(row.country)}</td>
                    <td>${escapeHtml(row.occurredAt ? formatDate(row.occurredAt.toISOString()) : "-")}</td>
                    <td>${escapeHtml(formatRemaining(row.expiresAt, now, ctx.t))}</td>
                    <td>${escapeHtml(`${siteLabel}${escalationLabel}`)}</td>
                    <td>${escapeHtml(renderModules(row.modules, ctx.t))}${(allowlistBySite.get(row.siteID) || new Set()).has(row.ip) ? " (allowlist)" : ""}</td>
                    <td>
                      <div class="waf-ban-actions">
                        <button type="button" class="btn success btn-sm waf-ban-btn" data-action="extend" data-row="${index}" title="${escapeHtml(ctx.t("bans.action.extend"))}">
                          <span class="waf-ban-btn-icon">${ICON_PLUS}</span>
                          <span>${escapeHtml(ctx.t("bans.action.extend"))}</span>
                        </button>
                        <button type="button" class="btn danger btn-sm waf-ban-btn" data-action="unban" data-row="${index}" title="${escapeHtml(ctx.t("bans.action.unban"))}">
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
      `;

      listNode.querySelectorAll(".waf-ban-btn").forEach((button) => {
        button.addEventListener("click", async () => {
          const action = String(button.dataset.action || "");
          const rowIndex = Number.parseInt(String(button.dataset.row || "-1"), 10);
          const row = rows[rowIndex];
          if (!row || !row.siteID || !row.ip) {
            return;
          }
          const ip = normalizeIP(row.ip);
          const escalationState = loadEscalationState();
          const rec = getEscalationRecord(escalationState, ip);
          const policy = getPolicyForSite(row.siteID, policyBySite);
          const currentLevel = Math.max(0, Math.min(Number(rec?.level || 0), Math.max(0, policy.stages.length - 1)));
          const currentStageSeconds = Number(policy.stages[currentLevel] ?? policy.baseSeconds ?? AUTO_BAN_FALLBACK_SECONDS);
          const isAllowlisted = (allowlistBySite.get(row.siteID) || new Set()).has(ip);
          if (isAllowlisted) {
            ctx.notify("IP is in allowlist for this site; no ban actions applied.");
            return;
          }

          if (action === "unban") {
            if (!confirmAction(ctx.t("bans.confirm.unban", { ip: row.ip, site: row.siteID }))) {
              return;
            }
            try {
              button.disabled = true;
              if (currentStageSeconds === 0 && policy.scope === "all_sites") {
                await applyBanAllSites(ctx, sites, ip, "unban", allowlistBySite);
              } else {
                await postBanAction(ctx, row.siteID, row.origin, "unban", ip);
              }
              rec.last_unban_ms = Date.now();
              rec.last_enforced_level = 0;
              saveEscalationState(escalationState);
              clearRateLimitCookie(row.siteID);
              ctx.notify(ctx.t("toast.ipUnbanned"));
              await renderRows();
            } catch (error) {
              setError(statusNode, error?.message || ctx.t("bans.error.action"));
            } finally {
              button.disabled = false;
            }
            return;
          }

          if (!confirmAction(ctx.t("bans.confirm.extend", { ip: row.ip, site: row.siteID }))) {
            return;
          }
          try {
            button.disabled = true;
            const level = currentLevel;
            if (currentStageSeconds === 0 && policy.scope === "all_sites") {
              await applyBanAllSites(ctx, sites, ip, "ban", allowlistBySite);
              rec.last_enforced_level = level;
            } else {
              if (currentStageSeconds === 0) {
                await postBanAction(ctx, row.siteID, row.origin, "ban", ip);
              }
              rec.last_enforced_level = level;
            }
            saveEscalationState(escalationState);
            ctx.notify(ctx.t("toast.ipBanned"));
            await renderRows();
          } catch (error) {
            setError(statusNode, error?.message || ctx.t("bans.error.action"));
          } finally {
            button.disabled = false;
          }
        });
      });
    } catch (error) {
      setError(statusNode, ctx.t("bans.error.load"));
      listNode.innerHTML = "";
    }
  };

  container.querySelector("#bans-refresh")?.addEventListener("click", () => {
    renderRows();
  });

  await renderRows();
}
