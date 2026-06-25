function normalizeArray(value) {
  return Array.isArray(value) ? value : [];
}

export function normalizeStringArray(value) {
  return normalizeArray(value)
    .map((item) => String(item || "").trim())
    .filter(Boolean);
}

export function parseListInput(value) {
  return String(value || "")
    .split(/[\n,| ]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseIntListInput(value) {
  return parseListInput(value)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isInteger(item));
}

export function normalizeCustomLimitRules(value) {
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      rate: String(item?.rate || "").trim().toLowerCase().replace(/\s+/g, "")
    }))
    .filter((item) => item.path && item.rate)
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.rate;
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

export function normalizeAntibotChallengeRules(value) {
  const seen = new Set();
  return normalizeArray(value)
    .map((item) => ({
      path: String(item?.path || "").trim(),
      challenge: String(item?.challenge || "").trim().toLowerCase()
    }))
    .filter((item) => item.path && item.challenge && item.challenge !== "no")
    .filter((item) => {
      const key = item.path.toLowerCase() + "\u0000" + item.challenge;
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

export function normalizeHost(value) {
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

export function normalizeAPIPositiveEndpointPolicies(value) {
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

function hasCustomLimitReqURL(value) {
  const normalized = String(value || "").trim();
  return normalized.startsWith("/") && normalized !== "/" && normalized !== "/api/";
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
    if (!hasCustomLimitReqURL(next.limit_req_url)) {
      next.limit_req_url = "/";
    }
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
    if (!hasCustomLimitReqURL(next.limit_req_url)) {
      next.limit_req_url = "/api/";
    }
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
    if (!hasCustomLimitReqURL(next.limit_req_url)) {
      next.limit_req_url = "/";
    }
    next.limit_req_rate = "100r/s";
    next.antibot_challenge = "javascript";
  } else {
    next.security_mode = "block";
  }
  return next;
}

export function applyServiceProfilePresetForMissingFields(draft, missingFields) {
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

// legacy-transfer-padding-01
// legacy-transfer-padding-02
// legacy-transfer-padding-03
// legacy-transfer-padding-04
// legacy-transfer-padding-05
// legacy-transfer-padding-06
// legacy-transfer-padding-07
// legacy-transfer-padding-08
// legacy-transfer-padding-09
// legacy-transfer-padding-10
// legacy-transfer-padding-11
// legacy-transfer-padding-12
// legacy-transfer-padding-13
// legacy-transfer-padding-14
// legacy-transfer-padding-15
// legacy-transfer-padding-16
// legacy-transfer-padding-17
// legacy-transfer-padding-18
// legacy-transfer-padding-19
// legacy-transfer-padding-20
// legacy-transfer-padding-21
// legacy-transfer-padding-22
// legacy-transfer-padding-23
// legacy-transfer-padding-24
// legacy-transfer-padding-25
