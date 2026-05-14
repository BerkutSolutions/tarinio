export function normalizeHost(value) {
  return String(value || "").trim().toLowerCase();
}

export function normalizeSiteID(value) {
  return String(value || "").trim().toLowerCase();
}
const SERVICE_PROFILE_VALUES = ["strict", "balanced", "compat", "api", "public-edge"];

export function normalizeServiceProfile(value) {
  const normalized = String(value || "").trim().toLowerCase();
  return SERVICE_PROFILE_VALUES.includes(normalized) ? normalized : "balanced";
}

export function formatServiceProfile(value, ctx) {
  const normalized = normalizeServiceProfile(value);
  const key = `sites.profile.${normalized}`;
  const translated = String(ctx.t(key) || "").trim();
  return translated && translated !== key ? translated : normalized;
}

export function normalizeAPIPositiveEndpointPolicies(value, deps = {}) {
  const normalizeStringArray = deps.normalizeStringArray;
  const items = Array.isArray(value) ? value : [];
  const out = [];
  for (const item of items) {
    const path = String(item?.path || "").trim();
    if (!path) {
      continue;
    }
    out.push({
      path,
      methods: normalizeStringArray(item?.methods).map((entry) => entry.toUpperCase()),
      token_ids: normalizeStringArray(item?.token_ids),
      content_types: normalizeStringArray(item?.content_types).map((entry) => entry.toLowerCase()),
      mode: String(item?.mode || "").trim().toLowerCase()
    });
  }
  return out;
}

export function applyServiceProfilePresetToDraft(draft, profile) {
  const next = { ...draft, service_profile: normalizeServiceProfile(profile) };
  if (next.service_profile === "strict") {
    next.security_mode = "block";
    next.use_bad_behavior = true;
    next.bad_behavior_status_codes = [400, 401, 403, 404, 405, 429, 444];
    next.bad_behavior_threshold = 60;
    next.bad_behavior_count_time_seconds = 60;
    next.bad_behavior_ban_time_seconds = 900;
    next.use_limit_req = true;
    next.limit_req_url = "/";
    next.limit_req_rate = "80r/s";
    next.use_limit_conn = true;
    next.limit_conn_max_http1 = 120;
    next.limit_conn_max_http2 = 220;
    next.limit_conn_max_http3 = 220;
    next.antibot_challenge = "javascript";
  } else if (next.service_profile === "compat") {
    next.security_mode = "monitor";
    next.use_bad_behavior = false;
    next.use_limit_req = false;
    next.use_limit_conn = true;
    next.limit_conn_max_http1 = 300;
    next.limit_conn_max_http2 = 500;
    next.limit_conn_max_http3 = 500;
    next.antibot_challenge = "no";
  } else if (next.service_profile === "api") {
    next.security_mode = "block";
    next.allowed_methods = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"];
    next.use_cors = true;
    next.cors_allowed_origins = ["*"];
    next.use_limit_req = true;
    next.limit_req_url = "/api/";
    next.limit_req_rate = "200r/s";
    next.antibot_challenge = "no";
    next.api_positive_security_enabled = true;
    next.api_positive_enforcement_mode = "monitor";
    next.api_positive_default_action = "allow";
  } else if (next.service_profile === "public-edge") {
    next.security_mode = "block";
    next.use_bad_behavior = true;
    next.bad_behavior_status_codes = [400, 401, 403, 404, 405, 429, 444];
    next.bad_behavior_threshold = 80;
    next.bad_behavior_count_time_seconds = 60;
    next.bad_behavior_ban_time_seconds = 600;
    next.use_blacklist = true;
    next.use_dnsbl = true;
    next.use_limit_req = true;
    next.limit_req_url = "/";
    next.limit_req_rate = "100r/s";
    next.antibot_challenge = "javascript";
  } else {
    next.security_mode = "block";
  }
  return next;
}

export function applyServiceProfilePresetForMissingFields(draft, missingFields, deps = {}) {
  const normalizeArray = deps.normalizeArray;
  const missing = new Set(normalizeArray(missingFields).map((item) => String(item || "").trim()));
  if (!missing.size) {
    return draft;
  }
  const preset = applyServiceProfilePresetToDraft({ ...draft }, draft?.service_profile);
  const merged = { ...preset };
  for (const key of Object.keys(draft || {})) {
    if (!missing.has(key)) {
      merged[key] = draft[key];
    }
  }
  return merged;
}

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

export function normalizeBanEscalationStages(values, fallbackBase = 300, deps = {}) {
  const normalizeArray = deps.normalizeArray;
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
  // Legacy placeholder from default easy profile template.
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
