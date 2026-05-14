export function normalizeIP(value) {
  return String(value || "").trim();
}

export function normalizeList(value) {
  return Array.isArray(value) ? value : [];
}

export function buildPageButtons(totalPages, currentPage, dataAttr) {
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

export async function tryGetJSON(path) {
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

export function mergeByID(primary, secondary, idField = "id") {
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

export function unwrapList(payload, keys = []) {
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

export function asDate(value) {
  const stamp = Date.parse(String(value || ""));
  return Number.isNaN(stamp) ? null : new Date(stamp);
}

export function normalizeSiteToken(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+/, "")
    .replace(/-+$/, "");
}

export function buildSiteAliasMap(sites) {
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

export function buildHostSiteMap(sites) {
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

export function resolveCanonicalSiteID(siteID, aliasMap, hostMap, hostHint = "") {
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

export function siteCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_${slug || "site"}`;
}

export function siteEscalationCookieName(siteID) {
  const slug = String(siteID || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
  return `waf_rate_limited_escalation_${slug || "site"}`;
}

export function clearRateLimitCookies(siteID) {
  const rateCookie = siteCookieName(siteID);
  const escalationCookie = siteEscalationCookieName(siteID);
  document.cookie = `${rateCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
  document.cookie = `${escalationCookie}=; Path=/; Max-Age=0; SameSite=Lax`;
}

export function buildIPSetBySite(accessPolicies, fieldName, resolveSiteID) {
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

export function formatRemaining(expiresAt, now, t) {
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
