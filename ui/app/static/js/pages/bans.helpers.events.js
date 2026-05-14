import { asDate, normalizeIP, normalizeList } from "./bans.helpers.base.js";
import { isStartupSelfTestSite } from "./bans.helpers.timers.js";

export function parsePositiveNumber(value, fallback = 0) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return fallback;
  }
  return parsed;
}

export function parseSecurityStatus(value) {
  const parsed = Number.parseInt(String(value || ""), 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function normalizeCountryCode(value) {
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

export function countryFlagEmoji(value) {
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

export function deriveModuleFromEvent(item) {
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

export function shouldTreatAsAutoBanEvent(item) {
  const type = String(item?.type || "").trim().toLowerCase();
  const summary = String(item?.summary || "").trim().toLowerCase();
  const details = item?.details && typeof item.details === "object" ? item.details : {};
  if (summary.includes("not blocked")) {
    return false;
  }
  if (details?.blocked === true) {
    return true;
  }
  if (summary.includes("access blocked") || summary.includes("rate limit triggered")) {
    return true;
  }
  if (type === "security_access") {
    return true;
  }
  const status = parseSecurityStatus(details?.status);
  return status === 403 || status === 429 || status === 444;
}

export function renderReasonList(values) {
  const list = Array.from(values || []);
  if (!list.length) {
    return "-";
  }
  return list.join(", ");
}

export function buildAutoBanRows(events, resolveSiteID, siteBanDurationByID, unbanMarkersBySite) {
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
    const latestUnbanAt = unbanMarkersBySite.get(key);
    if (latestUnbanAt instanceof Date && latestUnbanAt.getTime() >= occurredAt.getTime()) {
      continue;
    }
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

export function buildLatestUnbanMarkers(auditItems, resolveSiteID) {
  const out = new Map();
  for (const item of normalizeList(auditItems)) {
    const action = String(item?.action || "").trim().toLowerCase();
    const status = String(item?.status || "").trim().toLowerCase();
    if (action !== "accesspolicy.unban" || status !== "succeeded") {
      continue;
    }
    const details = item?.details_json && typeof item.details_json === "object"
      ? item.details_json
      : (item?.details && typeof item.details === "object" ? item.details : {});
    const ip = normalizeIP(details?.ip);
    if (!ip) {
      continue;
    }
    const rawSiteID = String(item?.site_id || item?.resource_id || "").trim();
    const siteID = resolveSiteID(rawSiteID);
    if (!siteID) {
      continue;
    }
    const occurredAt = asDate(item?.occurred_at);
    if (!occurredAt) {
      continue;
    }
    const key = `${siteID}::${ip}`;
    const existing = out.get(key);
    if (!(existing instanceof Date) || occurredAt.getTime() >= existing.getTime()) {
      out.set(key, occurredAt);
    }
  }
  return out;
}

export function renderModules(modules, t) {
  return Array.from(modules)
    .map((item) => t(`bans.module.${item}`))
    .join(", ");
}

export async function postBanAction(ctx, siteID, origin, action, ip) {
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
