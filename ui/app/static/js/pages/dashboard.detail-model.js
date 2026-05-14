import {
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
} from "./dashboard.detail-model-helpers.js";

const OBSERVATION_WINDOW_MS = 24 * 60 * 60 * 1000;

export function buildDetailModel(stats, requestRows, eventRows) {
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
  const fallbackAttacksBySite = new Map();
  const fallbackBlockedBySite = new Map();
  const fallbackAttacksByURL = new Map();
  const fallbackAttacksByCountry = new Map();
  const fallbackIPDetails = new Map();
  let eventAttackCount = 0;

  (Array.isArray(requestRows) ? requestRows : []).forEach((row) => {
    const entry = row?.entry && typeof row.entry === "object" ? row.entry : {};
    const when = parseISOTime(entry.timestamp || row?.ingested_at);
    if (!when || when < cutoff) {
      return;
    }
    const site = resolveSiteLabel(entry.site, entry.host);
    const uri = String(entry.uri || "-").trim() || "-";
    if (shouldSkipInternalRequest(uri, entry.site, entry.host)) {
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

    if (status === 403 || status === 429 || status === 444) {
      addToMap(fallbackAttacksBySite, site, 1);
      addToMap(fallbackBlockedBySite, site, 1);
      addToMap(fallbackAttacksByURL, uri, 1);
      addToMap(fallbackAttacksByCountry, requestCountry, 1);
      if (ip) {
        const fallbackDetail = ensureIPDetail(fallbackIPDetails, ip);
        if (fallbackDetail) {
          fallbackDetail.attacks += 1;
          fallbackDetail.blocked += 1;
          addToMap(fallbackDetail.pages, uri, 1);
          addToMap(fallbackDetail.methods, method, 1);
          addToMap(fallbackDetail.sites, site, 1);
          addToMap(fallbackDetail.countryCounts, requestCountry, 1);
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
    const path = String(details.path || details.uri || "-").trim() || "-";
    if (shouldSkipInternalRequest(path, item?.site_id, details?.host)) {
      return;
    }
    const rawBlocked = details.blocked;
    if (typeof rawBlocked === "boolean" && !rawBlocked) {
      return;
    }
    eventAttackCount += 1;

    const ip = String(details.client_ip || details.ip || "").trim();
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

  if (eventAttackCount === 0) {
    fallbackAttacksBySite.forEach((count, key) => addToMap(attacksBySite, key, count));
    fallbackBlockedBySite.forEach((count, key) => addToMap(blockedBySite, key, count));
    fallbackAttacksByURL.forEach((count, key) => addToMap(attacksByURL, key, count));
    fallbackAttacksByCountry.forEach((count, key) => addToMap(attacksByCountry, key, count));
    fallbackIPDetails.forEach((fallbackDetail, ip) => {
      const detail = ensureIPDetail(ipDetails, ip);
      if (!detail) {
        return;
      }
      detail.attacks += fallbackDetail.attacks;
      detail.blocked += fallbackDetail.blocked;
      fallbackDetail.pages.forEach((count, key) => addToMap(detail.pages, key, count));
      fallbackDetail.methods.forEach((count, key) => addToMap(detail.methods, key, count));
      fallbackDetail.sites.forEach((count, key) => addToMap(detail.sites, key, count));
      fallbackDetail.countryCounts.forEach((count, key) => addToMap(detail.countryCounts, key, count));
    });
  }

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