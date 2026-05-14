import { escapeHtml } from "../ui.js";

let geoRuntime = {
  GEO_SELECTOR_LABELS: {},
  CONTINENT_VALUES: [],
  COUNTRY_GROUP_VALUES: [],
  QUICK_LIST_TEMPLATES: {},
  BAD_BEHAVIOR_STATUS_OPTIONS: []
};

export function configureSitesGeoListEditorsRuntime(runtime) {
  geoRuntime = {
    ...geoRuntime,
    ...(runtime && typeof runtime === "object" ? runtime : {})
  };
}

function normalizeArray(value) {
  return Array.isArray(value) ? value : [];
}

function regionDisplayName(code) {
  const normalized = String(code || "").trim().toUpperCase();
  if (!normalized) {
    return "";
  }
  if (geoRuntime.GEO_SELECTOR_LABELS[normalized]) {
    return geoRuntime.GEO_SELECTOR_LABELS[normalized];
  }
  try {
    const display = new Intl.DisplayNames(["en"], { type: "region" });
    return display.of(normalized) || normalized;
  } catch (error) {
    return normalized;
  }
}
function isCountryCode(value) {
  const normalized = String(value || "").trim().toUpperCase();
  return /^[A-Z]{2}$/.test(normalized) && !Object.prototype.hasOwnProperty.call(geoRuntime.GEO_SELECTOR_LABELS, normalized);
}

function countryFlagEmoji(code) {
  const normalized = String(code || "").trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(normalized)) {
    return "";
  }
  const base = 0x1f1e6;
  const first = normalized.charCodeAt(0) - 65;
  const second = normalized.charCodeAt(1) - 65;
  if (first < 0 || first > 25 || second < 0 || second > 25) {
    return "";
  }
  return String.fromCodePoint(base + first, base + second);
}

function regionDisplayLabel(code) {
  const normalized = String(code || "").trim().toUpperCase();
  const name = regionDisplayName(normalized);
  if (!name) {
    return normalized;
  }
  if (!normalized) {
    return name;
  }
  if (isCountryCode(normalized)) {
    const flag = countryFlagEmoji(normalized);
    return flag ? `${name} (${flag})` : name;
  }
  return `${name} (${normalized})`;
}

function normalizeStringArray(value) {
  return normalizeArray(value)
    .map((item) => String(item || "").trim())
    .filter(Boolean);
}

function buildGeoCatalogFallback() {
  const countries = [];
  for (let first = 65; first <= 90; first += 1) {
    for (let second = 65; second <= 90; second += 1) {
      const code = String.fromCharCode(first) + String.fromCharCode(second);
      if (regionDisplayName(code) !== code) {
        countries.push(code);
      }
    }
  }
  return [...geoRuntime.CONTINENT_VALUES, ...geoRuntime.COUNTRY_GROUP_VALUES, ...countries];
}

function normalizeGeoCatalogPayload(payload) {
  const continents = normalizeStringArray(payload?.continents).map((value) => value.toUpperCase());
  const groups = normalizeStringArray(payload?.groups).map((value) => value.toUpperCase());
  const countries = normalizeStringArray(payload?.countries)
    .map((value) => value.toUpperCase())
    .filter((value) => /^[A-Z]{2}$/.test(value) && regionDisplayName(value) !== value);
  const merged = Array.from(new Set([...continents, ...groups, ...countries]));
  return merged.length ? merged : buildGeoCatalogFallback();
}

function getQuickListTemplates(field) {
  return Array.isArray(geoRuntime.QUICK_LIST_TEMPLATES[field]) ? geoRuntime.QUICK_LIST_TEMPLATES[field] : [];
}
const LIST_FIELD_SET = new Set([
  "allowed_methods",
  "ssl_protocols",
  "permissions_policy",
  "keep_upstream_headers",
  "cors_allowed_origins",
  "access_allowlist",
  "exceptions_ip",
  "access_denylist",
  "blacklist_ip",
  "blacklist_rdns",
  "blacklist_asn",
  "blacklist_user_agent",
  "blacklist_uri",
  "blacklist_ip_urls",
  "blacklist_rdns_urls",
  "blacklist_asn_urls",
  "blacklist_user_agent_urls",
  "blacklist_uri_urls",
  "blacklist_country",
  "whitelist_country",
  "modsecurity_crs_plugins"
]);

const SETTINGS_SEARCH_INDEX = [
  { id: "primary_host", tab: "front", selector: "#service-host", labelKey: "sites.easy.front.serverName" },
  { id: "service_id", tab: "front", selector: "#service-id", labelKey: "sites.easy.front.serviceId" },
  { id: "security_mode", tab: "front", selector: "#service-security-mode", labelKey: "sites.editor.securityMode" },
  { id: "service_profile", tab: "front", selector: "#service-profile", labelKey: "sites.table.profile" },
  { id: "certificate_authority_server", tab: "front", selector: "#service-ca-server", labelKey: "sites.easy.front.caServer" },
  { id: "enabled", tab: "front", selector: "#service-enabled", labelKey: "sites.easy.front.serviceEnabled" },
  { id: "tls_enabled", tab: "front", selector: "#service-tls-enabled", labelKey: "sites.easy.front.tlsEnabled" },
  { id: "certificate_id", tab: "front", selector: "#service-certificate-id", labelKey: "sites.tls.certificateId" },
  { id: "upstream_host", tab: "upstream", selector: "#service-upstream-host", labelKey: "sites.upstream.field.host" },
  { id: "upstream_port", tab: "upstream", selector: "#service-upstream-port", labelKey: "sites.upstream.field.port" },
  { id: "use_reverse_proxy", tab: "upstream", selector: "#service-use-reverse-proxy", labelKey: "sites.easy.upstream.useReverseProxy" },
  { id: "pass_host_header", tab: "upstream", selector: "#service-pass-host-header", labelKey: "sites.easy.upstream.passHostHeader" },
  { id: "send_x_forwarded_for", tab: "upstream", selector: "#service-send-x-forwarded-for", labelKey: "sites.easy.upstream.sendXForwardedFor" },
  { id: "send_x_forwarded_proto", tab: "upstream", selector: "#service-send-x-forwarded-proto", labelKey: "sites.easy.upstream.sendXForwardedProto" },
  { id: "send_x_real_ip", tab: "upstream", selector: "#service-send-x-real-ip", labelKey: "sites.easy.upstream.sendXRealIp" },
  { id: "allowed_methods", tab: "http", selector: "#list-input-allowed_methods", labelKey: "sites.easy.http.allowedMethods" },
  { id: "max_client_size", tab: "http", selector: "#service-max-client-size", labelKey: "sites.easy.http.maxBodySize" },
  { id: "hsts_enabled", tab: "headers", selector: "#service-hsts-enabled", labelKey: "sites.easy.headers.hstsEnabled" },
  { id: "hsts_max_age_seconds", tab: "headers", selector: "#service-hsts-max-age", labelKey: "sites.easy.headers.hstsMaxAge" },
  { id: "content_security_policy", tab: "headers", selector: "#service-content-security-policy", labelKey: "sites.easy.headers.contentSecurityPolicy" },
  { id: "permissions_policy", tab: "headers", selector: "#list-input-permissions_policy", labelKey: "sites.easy.headers.permissionsPolicy" },
  { id: "access_allowlist", tab: "traffic", selector: "#list-input-access_allowlist", labelKey: "sites.lists.allowlist" },
  { id: "exceptions_ip", tab: "traffic", selector: "#list-input-exceptions_ip", labelKey: "sites.easy.traffic.exceptions" },
  { id: "access_denylist", tab: "traffic", selector: "#list-input-access_denylist", labelKey: "sites.lists.denylist" },
  { id: "use_blacklist", tab: "traffic", selector: "#service-use-blacklist", labelKey: "sites.easy.traffic.activateBlacklisting" },
  { id: "use_limit_req", tab: "traffic", selector: "#service-use-limit-req", labelKey: "sites.easy.traffic.activateLimitRequests" },
  { id: "ban_escalation_enabled", tab: "blocking", selector: "#service-ban-escalation-enabled", labelKey: "sites.easy.blocking.enabled" },
  { id: "ban_escalation_scope", tab: "blocking", selector: "#service-ban-escalation-scope", labelKey: "sites.easy.blocking.scope" },
  { id: "antibot_challenge", tab: "antibot", selector: "#service-antibot-challenge", labelKey: "sites.easy.antibot.challenge" },
  { id: "blacklist_country", tab: "geo", selector: "#country-search-blacklist_country", labelKey: "sites.easy.geo.countryBlacklist" },
  { id: "whitelist_country", tab: "geo", selector: "#country-search-whitelist_country", labelKey: "sites.easy.geo.countryWhitelist" },
  { id: "use_modsecurity", tab: "modsec", selector: "#service-use-modsecurity", labelKey: "sites.easy.modsec.useModsecurity" }
];

function renderListEditor(field, label, values, placeholder = "", options = {}) {
  const safeValues = normalizeStringArray(values);
  const fullWidth = options.full !== false;
  const emptyLabel = options.emptyLabel || "No values yet";
  const fieldClass = fullWidth ? "waf-field full" : "waf-field";
  const presets = Array.isArray(options.presets) ? options.presets : [];
  const selectedTemplates = normalizeStringArray(options.selectedTemplates);
  const ctx = options.ctx || null;
  const quickTemplateLabel = ctx?.t ? ctx.t("sites.easy.listTemplates.quick") : "Quick templates";
  const selectedTemplateLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: selectedTemplates.length }) : `Selected: ${selectedTemplates.length}`;
  const selectedValueLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: safeValues.length }) : `Selected: ${safeValues.length}`;
  const selectedItemsLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: selectedTemplates.length + safeValues.length }) : `Selected: ${selectedTemplates.length + safeValues.length}`;
  const selectedTemplateSet = new Set(selectedTemplates);
  const availablePresets = presets.filter((preset) => {
    const presetID = String(preset?.id || "").trim();
    return presetID && !selectedTemplateSet.has(presetID);
  });
  const presetByID = new Map(presets.map((preset) => [String(preset?.id || "").trim(), preset]));
  const resolvePresetLabel = (preset) => String(
    ctx?.t && preset?.labelKey ? ctx.t(preset.labelKey) : (preset?.label || preset?.id || "")
  );
  return `
    <div class="${fieldClass}">
      <label>${escapeHtml(label)}</label>
      <div class="waf-inline">
        <input id="list-input-${escapeHtml(field)}" placeholder="${escapeHtml(placeholder)}">
        <button class="btn ghost btn-sm" type="button" data-list-add="${escapeHtml(field)}">+</button>
      </div>
      ${presets.length ? `
        <div class="waf-template-picker">
          <details class="waf-status-dropdown">
            <summary>${escapeHtml(`${quickTemplateLabel} (${selectedTemplateLabel})`)}</summary>
            <div class="waf-status-options">
              ${availablePresets.map((preset) => `
                <button
                  class="btn ghost btn-sm waf-template-option"
                  type="button"
                  data-list-template-apply="${escapeHtml(field)}"
                  data-list-template-id="${escapeHtml(String(preset.id || ""))}">${escapeHtml(resolvePresetLabel(preset))}</button>
              `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
            </div>
          </details>
          <details class="waf-status-dropdown waf-list-selected-dropdown">
            <summary>${escapeHtml(`${ctx?.t ? ctx.t("sites.easy.listTemplates.add") : "Added"} (${selectedItemsLabel})`)}</summary>
            <div class="waf-status-options waf-list-selected-options">
              ${selectedTemplates.map((presetID) => {
                const preset = presetByID.get(presetID);
                if (!preset) {
                  return "";
                }
                return `
                  <div class="waf-list-selected-item">
                    <span class="waf-list-selected-value">${escapeHtml(resolvePresetLabel(preset))}</span>
                    <button
                      class="waf-list-remove"
                      type="button"
                      data-list-template-remove="${escapeHtml(field)}"
                      data-list-template-id="${escapeHtml(presetID)}">x</button>
                  </div>
                `;
              }).join("")}
              ${safeValues.map((value, index) => `
                <div class="waf-list-selected-item">
                  <span class="waf-list-selected-value">${escapeHtml(value)}</span>
                  <button
                    class="waf-list-remove"
                    type="button"
                    data-list-remove="${escapeHtml(field)}"
                    data-list-index="${index}">x</button>
                </div>
              `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
            </div>
            <div class="waf-note">${escapeHtml(selectedValueLabel)}</div>
          </details>
        </div>
      ` : ""}
      ${presets.length ? "" : `
        <div class="waf-inline">
          ${safeValues.map((value, index) => `
            <span class="badge badge-neutral">
              ${escapeHtml(value)}
              <button
                class="waf-list-remove"
                type="button"
                data-list-remove="${escapeHtml(field)}"
                data-list-index="${index}">x</button>
            </span>
          `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
        </div>
      `}
    </div>
  `;
}

function renderCountryEditor(field, label, values, catalog, options = {}) {
  const safeValues = normalizeStringArray(values);
  const catalogOptions = normalizeStringArray(catalog).length ? normalizeStringArray(catalog) : buildGeoCatalogFallback();
  const fullWidth = options.full !== false;
  const emptyLabel = options.emptyLabel || "No values yet";
  const fieldClass = fullWidth ? "waf-field full" : "waf-field";
  const search = String(options.search || "").trim().toLowerCase();
  const selected = new Set(safeValues);
  const ctx = options.ctx || null;
  const selectedCountLabel = ctx?.t ? ctx.t("sites.easy.selectedCount", { count: safeValues.length }) : `Selected: ${safeValues.length}`;
  const searchPlaceholder = ctx?.t ? ctx.t("sites.easy.geo.searchPlaceholder") : "Search country or code";
  const filteredOptions = catalogOptions.filter((value) => {
    if (!search) {
      return true;
    }
    const haystack = `${value} ${regionDisplayLabel(value)} ${regionDisplayName(value)}`.toLowerCase();
    return haystack.includes(search);
  });
  const visibleOptions = filteredOptions.length ? filteredOptions : catalogOptions.filter((value) => selected.has(value));
  return `
    <div class="${fieldClass}">
      <label>${escapeHtml(label)}</label>
      <details class="waf-status-dropdown waf-country-picker" open>
        <summary>${escapeHtml(selectedCountLabel)}</summary>
        <input id="country-search-${escapeHtml(field)}" class="waf-country-search" placeholder="${escapeHtml(searchPlaceholder)}" value="${escapeHtml(search)}">
        <div class="waf-status-options waf-country-options">
          ${visibleOptions.map((value) => `
            <label class="waf-checkbox waf-status-option waf-country-option">
              <input type="checkbox" data-country-toggle="${escapeHtml(field)}" data-country-value="${escapeHtml(value)}"${selected.has(value) ? " checked" : ""}>
              <span>${escapeHtml(value)}</span>
              <span class="waf-note waf-country-option-name">${escapeHtml(regionDisplayLabel(value))}</span>
            </label>
          `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
        </div>
      </details>
      <div class="waf-inline">
        ${safeValues.map((value, index) => `
          <span class="badge badge-neutral">
            ${escapeHtml(regionDisplayLabel(value))}
            <button
              class="waf-list-remove"
              type="button"
              data-list-remove="${escapeHtml(field)}"
              data-list-index="${index}">x</button>
          </span>
        `).join("") || `<span class="waf-note">${escapeHtml(emptyLabel)}</span>`}
      </div>
    </div>
  `;
}

function renderStatusCodesEditor(selectedCodes, ctx) {
  const selected = new Set(normalizeArray(selectedCodes).map((item) => Number(item)).filter((item) => Number.isInteger(item)));
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.badStatusCodes"))}</label>
      <details class="waf-status-dropdown">
        <summary>${escapeHtml(ctx.t("sites.easy.selectedCount", { count: selected.size }))}</summary>
        <div class="waf-note">${escapeHtml(ctx.t("sites.easy.badStatusCodesHint"))}</div>
        <div class="waf-status-options">
          ${geoRuntime.BAD_BEHAVIOR_STATUS_OPTIONS.map(([code, text]) => `
            <label class="waf-checkbox waf-status-option">
              <input type="checkbox" data-bad-code="${code}"${selected.has(code) ? " checked" : ""}>
              <span>${escapeHtml(`${code} ${text}`)}</span>
            </label>
          `).join("")}
        </div>
      </details>
    </div>
  `;
}

export {
  regionDisplayName,
  isCountryCode,
  countryFlagEmoji,
  regionDisplayLabel,
  buildGeoCatalogFallback,
  normalizeGeoCatalogPayload,
  getQuickListTemplates,
  LIST_FIELD_SET,
  SETTINGS_SEARCH_INDEX,
  renderListEditor,
  renderCountryEditor,
  renderStatusCodesEditor
};
