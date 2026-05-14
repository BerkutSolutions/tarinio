import { asDate, normalizeIP, normalizeList } from "./bans.helpers.base.js";

const MANUAL_BAN_TIMERS = new Map();
const DISMISSED_BAN_ROWS = new Set();

export function manualBanTimerRowKey(siteID, ip) {
  return `${String(siteID || "").trim().toLowerCase()}::${normalizeIP(ip).toLowerCase()}`;
}

export function dismissBanRow(siteID, ip) {
  DISMISSED_BAN_ROWS.add(manualBanTimerRowKey(siteID, ip));
}

export function clearDismissedBanRow(siteID, ip) {
  DISMISSED_BAN_ROWS.delete(manualBanTimerRowKey(siteID, ip));
}

export function isDismissedBanRow(siteID, ip) {
  return DISMISSED_BAN_ROWS.has(manualBanTimerRowKey(siteID, ip));
}

export function isStartupSelfTestSite(siteID) {
  return String(siteID || "").trim().toLowerCase().startsWith("startup-self-test-");
}

export function loadManualBanTimers() {
  return new Map(Array.from(MANUAL_BAN_TIMERS.entries()).map(([key, value]) => [key, { ...value }]));
}

export function saveManualBanTimers(map) {
  MANUAL_BAN_TIMERS.clear();
  for (const [key, value] of map.entries()) {
    MANUAL_BAN_TIMERS.set(key, { ...value });
  }
}

export function upsertManualBanTimer(map, siteID, ip, unbanAt, createdAt = new Date().toISOString()) {
  const key = manualBanTimerRowKey(siteID, ip);
  map.set(key, {
    siteID: String(siteID || "").trim(),
    ip: normalizeIP(ip),
    unbanAt: new Date(unbanAt).toISOString(),
    createdAt: String(createdAt || "").trim() || new Date().toISOString()
  });
}

export function removeManualBanTimer(map, siteID, ip) {
  map.delete(manualBanTimerRowKey(siteID, ip));
}

export async function processExpiredManualBanTimers(ctx, timers, postBanAction) {
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

export function buildManualBanRows(accessPolicies, resolveSiteID, manualBanTimers) {
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

export function mergeBanRows(manualRows, autoRows) {
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
