export function normalizeEmail(value) {
  return String(value || "").trim().toLowerCase();
}

export const BAN_SCOPE_VALUES = ["current_site", "all_sites"];

export function parseBanDurationSeconds(value) {
  const raw = String(value || "").trim().toLowerCase();
  if (!raw) {
    return null;
  }
  if (/^\d+$/.test(raw)) {
    const seconds = Number.parseInt(raw, 10);
    return Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
  }
  const match = raw.match(/^(\d+)\s*(s|m|h|d)$/);
  if (!match) {
    return null;
  }
  const num = Number.parseInt(match[1], 10);
  if (!Number.isFinite(num) || num < 0) {
    return null;
  }
  const unit = match[2];
  if (unit === "s") {
    return num;
  }
  if (unit === "m") {
    return num * 60;
  }
  if (unit === "h") {
    return num * 3600;
  }
  if (unit === "d") {
    return num * 86400;
  }
  return null;
}

export function formatBanDurationSeconds(seconds) {
  const value = Number.parseInt(String(seconds), 10);
  if (!Number.isFinite(value) || value < 0) {
    return "-";
  }
  if (value === 0) {
    return "0";
  }
  if (value % 86400 === 0) {
    return `${value / 86400}d`;
  }
  if (value % 3600 === 0) {
    return `${value / 3600}h`;
  }
  if (value % 60 === 0) {
    return `${value / 60}m`;
  }
  return `${value}s`;
}

import { normalizeArray } from "./sites.routing-merge.js";

export function normalizeBanEscalationStages(values, fallbackBase = 300) {
  const out = [];
  for (const raw of normalizeArray(values)) {
    const value = Number.parseInt(String(raw), 10);
    if (!Number.isFinite(value) || value < 0) {
      continue;
    }
    out.push(value);
    if (value === 0) {
      break;
    }
  }
  if (!out.length) {
    const base = Number.parseInt(String(fallbackBase), 10);
    const normalizedBase = Number.isFinite(base) && base >= 0 ? base : 300;
    return [normalizedBase, 86400, 0];
  }
  return out;
}

export function normalizeReverseProxyHost(value) {
  const normalized = String(value || "").trim();
  const lower = normalized.toLowerCase();
  if (!lower) {
    return "";
  }
  if (lower === "http://upstream-server:8080") {
    return "";
  }
  return normalized;
}

export function buildReverseProxyHostFromUpstream(upstreamScheme, upstreamHost, upstreamPort) {
  const host = String(upstreamHost || "").trim();
  if (!host) {
    return "";
  }
  const scheme = String(upstreamScheme || "http").trim().toLowerCase() === "https" ? "https" : "http";
  const port = Number(upstreamPort);
  if (Number.isInteger(port) && port > 0) {
    return `${scheme}://${host}:${port}`;
  }
  return `${scheme}://${host}`;
}

export function resolveReverseProxyHost(draft, explicitValue = "") {
  const manual = normalizeReverseProxyHost(explicitValue || draft?.reverse_proxy_host);
  if (manual) {
    return manual;
  }
  return buildReverseProxyHostFromUpstream(draft?.upstream_scheme, draft?.upstream_host, draft?.upstream_port);
}

export function isValidEmail(value) {
  const normalized = normalizeEmail(value);
  if (!normalized) {
    return false;
  }
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized);
}

export function resolvePublicServiceURL(site, tlsState) {
  const host = String(site?.primary_host || site?.id || "").trim();
  if (!host) {
    return "";
  }
  if (/^https?:\/\//i.test(host)) {
    return host;
  }
  const scheme = tlsState === "managed" || tlsState === "detected" ? "https" : "http";
  return `${scheme}://${host}`;
}

export function computeUpstreamID(siteID) {
  const normalized = String(siteID || "").trim().toLowerCase();
  return normalized ? `${normalized}-upstream` : "";
}
