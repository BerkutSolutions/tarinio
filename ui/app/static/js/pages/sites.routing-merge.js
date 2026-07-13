const TLS_EXPIRY_ALERT_STORAGE_KEY = "waf_tls_expiry_alerted";

export function routeBase() {
  return "/services";
}

export function routeInfo() {
  const path = (window.location.pathname || routeBase()).replace(/\/+$/, "") || routeBase();
  if (path === routeBase()) {
    return { mode: "list", siteID: "" };
  }
  if (path === `${routeBase()}/new`) {
    return { mode: "create", siteID: "" };
  }
  if (path.startsWith(`${routeBase()}/`)) {
    return { mode: "detail", siteID: decodeURIComponent(path.slice(`${routeBase()}/`.length)) };
  }
  return { mode: "list", siteID: "" };
}

export function go(path) {
  window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

export function formatCertificateExpiryByLanguage(value, language) {
  const date = new Date(String(value || ""));
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = String(date.getFullYear());
  const locale = String(language || "").trim().toLowerCase();
  if (locale === "ru") {
    return `${day}.${month}.${year}`;
  }
  return `${day}-${month}-${year}`;
}

export function certificateDaysLeft(value) {
  const date = new Date(String(value || ""));
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  const millisLeft = date.getTime() - Date.now();
  return Math.ceil(millisLeft / (24 * 60 * 60 * 1000));
}

function readExpiryAlertState() {
  try {
    const raw = window.localStorage.getItem(TLS_EXPIRY_ALERT_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch (_error) {
    return {};
  }
}

function writeExpiryAlertState(state) {
  try {
    window.localStorage.setItem(TLS_EXPIRY_ALERT_STORAGE_KEY, JSON.stringify(state));
  } catch (_error) {
    // ignore storage write errors
  }
}

export function notifyExpiringCertificates(ctx, certificates) {
  const language = String(ctx.getLanguage?.() || "en").trim().toLowerCase();
  const previous = readExpiryAlertState();
  const next = {};
  for (const certificate of normalizeArray(certificates)) {
    const id = String(certificate?.id || "").trim().toLowerCase();
    const notAfterRaw = String(certificate?.not_after || "").trim();
    if (!id || !notAfterRaw) {
      continue;
    }
    const key = `${id}|${notAfterRaw}`;
    const daysLeft = certificateDaysLeft(notAfterRaw);
    const shouldAlert = typeof daysLeft === "number" && daysLeft < 30;
    if (!shouldAlert) {
      continue;
    }
    next[key] = true;
    if (previous[key]) {
      continue;
    }
    const title = String(certificate?.common_name || certificate?.id || "certificate").trim();
    const expiresAt = formatCertificateExpiryByLanguage(notAfterRaw, language);
    ctx.notify(
      ctx.t("sites.toast.tlsExpiringSoon", {
        value: title,
        count: String(Math.max(0, daysLeft)),
        date: expiresAt,
      }),
      "error",
      { sticky: true }
    );
  }
  writeExpiryAlertState(next);
}

export function normalizeArray(value) {
  return Array.isArray(value) ? value : [];
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
    if (!text) {
      return null;
    }
    return JSON.parse(text);
  } catch (error) {
    return null;
  }
}

export function mergeBySiteID(primary, secondary) {
  const map = new Map();
  for (const item of normalizeArray(primary)) {
    const id = normalizeSiteID(String(item?.site_id || "").trim());
    if (!id) continue;
    map.set(id, { ...item, _origin: "primary" });
  }
  for (const item of normalizeArray(secondary)) {
    const id = normalizeSiteID(String(item?.site_id || "").trim());
    if (!id || map.has(id)) continue;
    map.set(id, { ...item, _origin: "secondary" });
  }
  return Array.from(map.values());
}

export function mergeByID(primary, secondary) {
  const map = new Map();
  for (const item of normalizeArray(primary)) {
    const id = String(item?.id || "").trim();
    const normalizedID = normalizeSiteID(id);
    if (!normalizedID) {
      continue;
    }
    map.set(normalizedID, { ...item, _origin: "primary" });
  }
  for (const item of normalizeArray(secondary)) {
    const id = String(item?.id || "").trim();
    const normalizedID = normalizeSiteID(id);
    if (!normalizedID || map.has(normalizedID)) {
      continue;
    }
    map.set(normalizedID, { ...item, _origin: "secondary" });
  }
  return Array.from(map.values());
}

export function normalizeSiteID(value) {
  return String(value || "").trim().toLowerCase();
}

export function mergeProfilesBySite(primary, secondaryPayload) {
  const map = new Map();
  for (const item of normalizeArray(primary)) {
    const siteID = normalizeSiteID(item?.site_id);
    if (!siteID) {
      continue;
    }
    map.set(siteID, item);
  }
  for (const item of unwrapList(secondaryPayload, ["easy_site_profiles"])) {
    const siteID = normalizeSiteID(item?.site_id);
    if (!siteID || map.has(siteID)) {
      continue;
    }
    map.set(siteID, item);
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

export function findEasyProfile(payload, siteID) {
  const profiles = unwrapList(payload, ["easy_site_profiles"]);
  const target = String(siteID || "").trim().toLowerCase();
  return profiles.find((item) => String(item?.site_id || "").trim().toLowerCase() === target) || null;
}
