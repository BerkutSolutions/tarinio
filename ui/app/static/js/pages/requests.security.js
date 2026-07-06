export const REQUEST_ROW_TYPE_REQUEST = "request";
export const REQUEST_ROW_TYPE_SECURITY = "security";
export const LEGACY_REQUESTS_ROW_TYPE_SUPPORT_FIELD = "legacy_row_type_support";

const SECURITY_REASON_LABELS = {
  modsecurity: "requests.securityReason.modsecurity",
  waf: "requests.securityReason.modsecurity",
  sqli: "requests.securityReason.sqli",
  xss: "requests.securityReason.xss",
  burst: "requests.securityReason.burst",
  rate_limit: "requests.securityReason.rateLimit",
  ratelimit: "requests.securityReason.rateLimit",
  threat_intel: "requests.securityReason.threatIntel",
  intel: "requests.securityReason.threatIntel",
  reputation: "requests.securityReason.threatIntel",
  scanner: "requests.securityReason.scanner",
  bot: "requests.securityReason.scanner",
  geo: "requests.securityReason.geo",
  country: "requests.securityReason.geo",
  auth: "requests.securityReason.auth",
  challenge: "requests.securityReason.challenge",
  access_blocked: "requests.securityReason.accessBlocked",
};

function normalizeSecurityToken(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function requestRowTypeLabelKey(rowType) {
  return rowType === REQUEST_ROW_TYPE_SECURITY ? "requests.type.security" : "requests.type.request";
}

export function requestRowTypeSortValue(row) {
  return row?.rowType === REQUEST_ROW_TYPE_SECURITY ? "0-security" : "1-request";
}

export function firstNonEmptyString(...values) {
  for (const value of values) {
    const text = String(value || "").trim();
    if (text) {
      return text;
    }
  }
  return "";
}

export function coerceObject(value) {
  return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

export function inferLegacyRequestRowType(row) {
  const explicit = normalizeSecurityToken(row?.row_type || row?.rowType);
  if (explicit === REQUEST_ROW_TYPE_SECURITY) {
    return REQUEST_ROW_TYPE_SECURITY;
  }
  if (explicit === REQUEST_ROW_TYPE_REQUEST) {
    return REQUEST_ROW_TYPE_REQUEST;
  }
  const stream = normalizeSecurityToken(row?.stream);
  const type = normalizeSecurityToken(row?.type);
  const source = normalizeSecurityToken(row?.source_component);
  if (
    stream.startsWith("security") ||
    type.startsWith("security") ||
    type.includes("modsecurity") ||
    source.startsWith("security")
  ) {
    return REQUEST_ROW_TYPE_SECURITY;
  }
  return REQUEST_ROW_TYPE_REQUEST;
}

export function buildRequestEntry(row) {
  const sourceEntry = coerceObject(row?.entry);
  const details = coerceObject(row?.details);
  return {
    ...details,
    ...sourceEntry,
    request_id: firstNonEmptyString(sourceEntry.request_id, details.request_id, sourceEntry.event_id, row?.id),
    timestamp: firstNonEmptyString(sourceEntry.timestamp, details.timestamp, row?.occurred_at, row?.ingested_at),
    method: firstNonEmptyString(sourceEntry.method, details.method),
    uri: firstNonEmptyString(sourceEntry.uri, details.uri, details.path),
    status: sourceEntry.status ?? details.status ?? "",
    client_ip: firstNonEmptyString(sourceEntry.client_ip, details.client_ip),
    upstream_addr: firstNonEmptyString(sourceEntry.upstream_addr, details.upstream_addr, row?.source_component),
    referer: firstNonEmptyString(sourceEntry.referer, details.referer),
    user_agent: firstNonEmptyString(sourceEntry.user_agent, details.user_agent),
    host: firstNonEmptyString(sourceEntry.host, details.host),
    site: firstNonEmptyString(sourceEntry.site, details.site, row?.site_id),
  };
}

export function normalizedSecurityEventType(row) {
  const topLevel = normalizeSecurityToken(row?.event_type);
  if (topLevel) {
    return topLevel;
  }
  const details = coerceObject(row?.details);
  return normalizeSecurityToken(details.event_type || row?.type);
}

export function normalizeSecurityReason(row) {
  const topLevelReason = String(row?.security_reason || "").trim();
  if (topLevelReason) {
    return topLevelReason;
  }
  const details = coerceObject(row?.details);
  const eventType = normalizedSecurityEventType(row);
  if (eventType) {
    return eventType;
  }
  return firstNonEmptyString(row?.summary, row?.type, details.intel_label, details.feed, details.path);
}

export function securityReasonLabelKey(reason) {
  const normalized = normalizeSecurityToken(reason);
  if (!normalized) {
    return "";
  }
  for (const [token, labelKey] of Object.entries(SECURITY_REASON_LABELS)) {
    if (normalized === token || normalized.startsWith(token + "_") || normalized.includes("_" + token + "_")) {
      return labelKey;
    }
  }
  return "";
}

export function buildSecurityDetailSummary(row, ctx = null) {
  if (row?.rowType !== REQUEST_ROW_TYPE_SECURITY) {
    return null;
  }
  const reason = normalizeSecurityReason(row);
  const topLevelReason = String(row?.security_reason || "").trim();
  const reasonLabelKey = securityReasonLabelKey(reason);
  const reasonLabel = reasonLabelKey && ctx?.t ? String(ctx.t(reasonLabelKey) || "") : reason;
  return {
    typeLabel: ctx?.t ? String(ctx.t(requestRowTypeLabelKey(row?.rowType)) || "") : REQUEST_ROW_TYPE_SECURITY,
    reasonLabel: reasonLabel || reason,
    reasonRaw: topLevelReason || reason,
    normalizedEventType: normalizedSecurityEventType(row),
    legacyCompatibility: Boolean(row?.[LEGACY_REQUESTS_ROW_TYPE_SUPPORT_FIELD]),
    legacyCompatibilityText: ctx?.t ? String(ctx.t("requests.detail.legacyCompatibilityEnabled") || "") : "legacy row_type compatibility enabled",
  };
}

export function buildRequestDetailsMeta(row, ctx = null) {
  const details = coerceObject(row?.details);
  const meta = [];
  const push = (labelKey, value) => {
    const text = String(value ?? "").trim();
    if (text) {
      meta.push([labelKey, text]);
    }
  };
  const securitySummary = buildSecurityDetailSummary(row, ctx);

  push("requests.detail.summary", row?.summary);
  if (securitySummary?.reasonLabel) {
    push("requests.detail.securityReason", securitySummary.reasonLabel);
  }
  if (securitySummary?.normalizedEventType) {
    const eventTypeLabelKey = securityReasonLabelKey(securitySummary.normalizedEventType);
    const eventTypeLabel = eventTypeLabelKey && ctx?.t
      ? String(ctx.t(eventTypeLabelKey) || securitySummary.normalizedEventType)
      : securitySummary.normalizedEventType;
    push("requests.detail.eventType", eventTypeLabel);
  } else {
    push("requests.detail.eventType", row?.event_type || row?.type);
  }
  push("requests.detail.source", row?.source_component);
  push("requests.detail.siteId", row?.site_id);
  push("requests.detail.country", details.country);
  push("requests.detail.city", details.city);
  push("requests.detail.pathRequestsSec", details.path_requests_sec);
  push("requests.detail.requestsSecond", details.requests_second);
  push("requests.detail.blocked", typeof details.blocked === "boolean" ? (details.blocked ? "true" : "false") : details.blocked);
  push("requests.detail.feed", details.feed);
  push("requests.detail.intelLabel", details.intel_label);
  push("requests.detail.intelScore", details.intel_score);
  if (securitySummary?.legacyCompatibility) {
    push("requests.detail.legacyCompatibility", securitySummary.legacyCompatibilityText);
  }
  return meta;
}

export function buildSecurityBadge(row, ctx, escapeHtml) {
  if (row?.rowType !== REQUEST_ROW_TYPE_SECURITY) {
    return "";
  }
  const reason = normalizeSecurityReason(row);
  const labelKey = securityReasonLabelKey(reason);
  const label = labelKey ? ctx.t(labelKey) : reason;
  if (!label) {
    return `<span class="waf-badge waf-badge-warning">${escapeHtml(ctx.t("requests.securityBadge"))}</span>`;
  }
  const title = reason && label !== reason ? `${label}: ${reason}` : label;
  return `<span class="waf-badge waf-badge-warning" title="${escapeHtml(title)}">${escapeHtml(label)}</span>`;
}
