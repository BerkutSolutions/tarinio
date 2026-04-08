import { confirmAction, escapeHtml, formatDate, setError, setLoading, statusBadge } from "../ui.js";

function routeBase() {
  return "/services";
}

function routeInfo() {
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

function go(path) {
  window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

function normalizeArray(value) {
  return Array.isArray(value) ? value : [];
}

async function tryGetJSON(path) {
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

function mergeByID(primary, secondary) {
  const map = new Map();
  for (const item of normalizeArray(primary)) {
    const id = String(item?.id || "");
    if (!id) {
      continue;
    }
    map.set(id, { ...item, _origin: "primary" });
  }
  for (const item of normalizeArray(secondary)) {
    const id = String(item?.id || "");
    if (!id || map.has(id)) {
      continue;
    }
    map.set(id, { ...item, _origin: "secondary" });
  }
  return Array.from(map.values());
}

function unwrapList(payload, keys = []) {
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

function findEasyProfile(payload, siteID) {
  const profiles = unwrapList(payload, ["easy_site_profiles"]);
  const target = String(siteID || "").trim().toLowerCase();
  return profiles.find((item) => String(item?.site_id || "").trim().toLowerCase() === target) || null;
}

function normalizeStringArray(value) {
  return normalizeArray(value)
    .map((item) => String(item || "").trim())
    .filter(Boolean);
}

function parseListInput(value) {
  return String(value || "")
    .split(/[\n,| ]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseIntListInput(value) {
  return parseListInput(value)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isInteger(item));
}

function normalizeCustomLimitRules(value) {
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

function renderCustomLimitRulesEditor(rules, ctx) {
  const safeRules = normalizeCustomLimitRules(rules);
  return `
    <div class="waf-field full">
      <label>${escapeHtml(ctx.t("sites.easy.traffic.customLimitRules"))}</label>
      <div class="waf-stack">
        ${safeRules.map((rule, index) => `
          <div class="waf-inline waf-custom-limit-row">
            <input data-custom-limit-path="${index}" placeholder="/login" value="${escapeHtml(rule.path)}">
            <input data-custom-limit-rate="${index}" placeholder="20r/s" value="${escapeHtml(rule.rate)}">
            <button class="btn ghost btn-sm" type="button" data-custom-limit-remove="${index}">x</button>
          </div>
        `).join("")}
        ${safeRules.length ? "" : `<span class="waf-note">${escapeHtml(ctx.t("sites.easy.noValues"))}</span>`}
        <button class="btn ghost btn-sm" type="button" data-custom-limit-add>${escapeHtml(ctx.t("sites.easy.traffic.addCustomLimit"))}</button>
      </div>
    </div>`;
}

function normalizeHost(value) {
  return String(value || "").trim().toLowerCase();
}

function normalizeSiteID(value) {
  return String(value || "").trim().toLowerCase();
}

function normalizeEmail(value) {
  return String(value || "").trim().toLowerCase();
}

const BAN_SCOPE_VALUES = ["current_site", "all_sites"];

function parseBanDurationSeconds(value) {
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

function formatBanDurationSeconds(seconds) {
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

function normalizeBanEscalationStages(values, fallbackBase = 300) {
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

function normalizeReverseProxyHost(value) {
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

function buildReverseProxyHostFromUpstream(upstreamScheme, upstreamHost, upstreamPort) {
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

function resolveReverseProxyHost(draft, explicitValue = "") {
  const manual = normalizeReverseProxyHost(explicitValue || draft?.reverse_proxy_host);
  if (manual) {
    return manual;
  }
  return buildReverseProxyHostFromUpstream(draft?.upstream_scheme, draft?.upstream_host, draft?.upstream_port);
}

function isValidEmail(value) {
  const normalized = normalizeEmail(value);
  if (!normalized) {
    return false;
  }
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized);
}

function resolvePublicServiceURL(site, tlsState) {
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

function computeUpstreamID(siteID) {
  const normalized = String(siteID || "").trim().toLowerCase();
  return normalized ? `${normalized}-upstream` : "";
}

const BAD_BEHAVIOR_STATUS_OPTIONS = [
  [400, "Bad Request"],
  [401, "Unauthorized"],
  [402, "Payment Required"],
  [403, "Forbidden"],
  [404, "Not Found"],
  [405, "Method Not Allowed"],
  [406, "Not Acceptable"],
  [407, "Proxy Authentication Required"],
  [408, "Request Timeout"],
  [409, "Conflict"],
  [410, "Gone"],
  [411, "Length Required"],
  [412, "Precondition Failed"],
  [413, "Payload Too Large"],
  [414, "URI Too Long"],
  [415, "Unsupported Media Type"],
  [416, "Range Not Satisfiable"],
  [417, "Expectation Failed"],
  [418, "I'm a teapot"],
  [421, "Misdirected Request"],
  [422, "Unprocessable Entity"],
  [423, "Locked"],
  [424, "Failed Dependency"],
  [425, "Too Early"],
  [426, "Upgrade Required"],
  [428, "Precondition Required"],
  [429, "Too Many Requests"],
  [431, "Request Header Fields Too Large"],
  [444, "No Response (nginx)"],
  [451, "Unavailable For Legal Reasons"],
  [500, "Internal Server Error"],
  [501, "Not Implemented"],
  [502, "Bad Gateway"],
  [503, "Service Unavailable"],
  [504, "Gateway Timeout"],
  [505, "HTTP Version Not Supported"],
  [507, "Insufficient Storage"],
  [508, "Loop Detected"],
  [510, "Not Extended"],
  [511, "Network Authentication Required"],
  [520, "Unknown Error (Cloudflare)"],
  [521, "Web Server Is Down (Cloudflare)"],
  [522, "Connection Timed Out (Cloudflare)"],
  [523, "Origin Is Unreachable (Cloudflare)"],
  [524, "A Timeout Occurred (Cloudflare)"],
  [525, "SSL Handshake Failed (Cloudflare)"],
  [526, "Invalid SSL Certificate (Cloudflare)"]
];

const CONTINENT_VALUES = ["AF", "AN", "AS", "EU", "NA", "OC", "SA"];
const COUNTRY_GROUP_VALUES = ["APAC", "EMEA", "LATAM", "DACH", "CIS", "GCC", "NORAM"];
const GEO_SELECTOR_LABELS = {
  AF: "Africa",
  AN: "Antarctica",
  AS: "Asia",
  EU: "Europe",
  NA: "North America",
  OC: "Oceania",
  SA: "South America",
  APAC: "Asia-Pacific",
  EMEA: "Europe, Middle East and Africa",
  LATAM: "Latin America",
  DACH: "DACH",
  CIS: "CIS",
  GCC: "Gulf Cooperation Council",
  NORAM: "North America"
};
const QUICK_LIST_TEMPLATES = {
  blacklist_user_agent: [
    {
      id: "scanner_uas",
      label: "Aggressive scanners",
      items: ["sqlmap", "nikto", "nmap", "masscan", "zgrab", "gobuster", "dirbuster", "wpscan", "acunetix", "nessus"]
    },
    {
      id: "cli_clients",
      label: "CLI and scripted clients",
      items: ["curl/.*", "python-requests", "python-httpx", "aiohttp", "Go-http-client", "libwww-perl"]
    },
    {
      id: "headless_tools",
      label: "Headless automation",
      items: ["HeadlessChrome", "PhantomJS", "selenium", "playwright"]
    }
  ],
  blacklist_uri: [
    {
      id: "common_probe_paths",
      label: "Common probe paths",
      items: ["/\\.env", "/\\.git", "/\\.svn", "/server-status", "/actuator", "/manager/html", "/cgi-bin", "/boaform", "/phpinfo"]
    },
    {
      id: "wordpress_probes",
      label: "WordPress probes",
      items: ["/wp-admin", "/wp-login\\.php", "/xmlrpc\\.php", "/wp-content", "/wp-includes"]
    },
    {
      id: "admin_panels",
      label: "Admin panels and PHP tooling",
      items: ["/phpmyadmin", "/pma", "/adminer", "/vendor/phpunit"]
    }
  ]
};

function regionDisplayName(code) {
  const normalized = String(code || "").trim().toUpperCase();
  if (!normalized) {
    return "";
  }
  if (GEO_SELECTOR_LABELS[normalized]) {
    return GEO_SELECTOR_LABELS[normalized];
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
  return /^[A-Z]{2}$/.test(normalized) && !Object.prototype.hasOwnProperty.call(GEO_SELECTOR_LABELS, normalized);
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
  return [...CONTINENT_VALUES, ...COUNTRY_GROUP_VALUES, ...countries];
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
  return Array.isArray(QUICK_LIST_TEMPLATES[field]) ? QUICK_LIST_TEMPLATES[field] : [];
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
  const selectedPreset = String(options.selectedPreset || "");
  return `
    <div class="${fieldClass}">
      <label>${escapeHtml(label)}</label>
      <div class="waf-inline">
        <input id="list-input-${escapeHtml(field)}" placeholder="${escapeHtml(placeholder)}">
        <button class="btn ghost btn-sm" type="button" data-list-add="${escapeHtml(field)}">+</button>
      </div>
      ${presets.length ? `
        <div class="waf-preset-row">
          <select id="list-template-${escapeHtml(field)}">
            <option value="">Quick templates</option>
            ${presets.map((preset) => `
              <option value="${escapeHtml(preset.id)}"${preset.id === selectedPreset ? " selected" : ""}>${escapeHtml(preset.label)}</option>
            `).join("")}
          </select>
          <button class="btn ghost btn-sm" type="button" data-list-template-add="${escapeHtml(field)}">Add template</button>
        </div>
      ` : ""}
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
        <summary>${escapeHtml(`Selected: ${safeValues.length}`)}</summary>
        <input id="country-search-${escapeHtml(field)}" class="waf-country-search" placeholder="Search country or code" value="${escapeHtml(search)}">
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
          ${BAD_BEHAVIOR_STATUS_OPTIONS.map(([code, text]) => `
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

function defaultSiteDraft() {
  return {
    id: "",
    primary_host: "",
    enabled: true,
    tls_enabled: true,
    tls_self_signed: false,
    certificate_id: "",
    security_mode: "block",
    upstream_id: "",
    upstream_host: "ui",
    upstream_port: 80,
    upstream_scheme: "http",
    auto_lets_encrypt: true,
    use_lets_encrypt_staging: false,
    use_lets_encrypt_wildcard: false,
    certificate_authority_server: "letsencrypt",
    acme_account_email: "",
    use_reverse_proxy: true,
    reverse_proxy_host: "http://upstream-server:8080",
    reverse_proxy_url: "/",
    reverse_proxy_custom_host: "",
    reverse_proxy_ssl_sni: false,
    reverse_proxy_ssl_sni_name: "",
    reverse_proxy_websocket: true,
    reverse_proxy_keepalive: true,
    pass_host_header: true,
    send_x_forwarded_for: true,
    send_x_forwarded_proto: true,
    send_x_real_ip: false,
    allowed_methods: ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"],
    max_client_size: "100m",
    http2: true,
    http3: false,
    ssl_protocols: ["TLSv1.2", "TLSv1.3"],
    cookie_flags: "* SameSite=Lax",
    content_security_policy: "",
    permissions_policy: [],
    keep_upstream_headers: ["*"],
    referrer_policy: "no-referrer-when-downgrade",
    use_cors: false,
    cors_allowed_origins: ["*"],
    use_allowlist: false,
    use_exceptions: false,
    access_allowlist: [],
    exceptions_ip: [],
    access_denylist: [],
    use_bad_behavior: true,
    bad_behavior_status_codes: [400, 401, 405, 444],
    bad_behavior_ban_time_seconds: 300,
    bad_behavior_threshold: 120,
    bad_behavior_count_time_seconds: 120,
    ban_escalation_enabled: false,
    ban_escalation_scope: "all_sites",
    ban_escalation_stages_seconds: [300, 86400, 0],
    use_blacklist: false,
    use_dnsbl: false,
    blacklist_ip: [],
    blacklist_rdns: [],
    blacklist_asn: [],
    blacklist_user_agent: [],
    blacklist_uri: [],
    blacklist_ip_urls: [],
    blacklist_rdns_urls: [],
    blacklist_asn_urls: [],
    blacklist_user_agent_urls: [],
    blacklist_uri_urls: [],
    use_limit_conn: true,
    limit_conn_max_http1: 200,
    limit_conn_max_http2: 400,
    limit_conn_max_http3: 400,
    use_limit_req: true,
    limit_req_url: "/",
    limit_req_rate: "120r/s",
    custom_limit_rules: [],
    antibot_challenge: "no",
    antibot_uri: "/challenge",
    antibot_recaptcha_score: 0.7,
    antibot_recaptcha_sitekey: "",
    antibot_recaptcha_secret: "",
    antibot_hcaptcha_sitekey: "",
    antibot_hcaptcha_secret: "",
    antibot_turnstile_sitekey: "",
    antibot_turnstile_secret: "",
    use_auth_basic: false,
    auth_basic_location: "sitewide",
    auth_basic_user: "changeme",
    auth_basic_password: "",
    auth_basic_text: "Restricted area",
    blacklist_country: [],
    whitelist_country: [],
    use_modsecurity: true,
    use_modsecurity_crs_plugins: true,
    use_modsecurity_custom_configuration: false,
    modsecurity_crs_version: "4",
    modsecurity_crs_plugins: [],
    modsecurity_custom_path: "modsec/anomaly_score.conf",
    modsecurity_custom_content: ""
  };
}

function siteDraftFromData(site, upstream, tlsConfig) {
  const base = {
    id: site?.id || "",
    primary_host: site?.primary_host || "",
    enabled: Boolean(site?.enabled ?? true),
    tls_enabled: Boolean(tlsConfig),
    tls_self_signed: false,
    certificate_id: tlsConfig?.certificate_id || (site?.id ? `${site.id}-tls` : ""),
    security_mode: "block",
    upstream_id: upstream?.id || (site?.id ? `${site.id}-upstream` : ""),
    upstream_host: upstream?.host || "ui",
    upstream_port: upstream?.port || 80,
    upstream_scheme: upstream?.scheme || "http"
  };
  return { ...defaultSiteDraft(), ...base };
}

function applyEasyProfileToDraft(draft, profile) {
  if (!profile || typeof profile !== "object") {
    return draft;
  }
  const front = profile.front_service || {};
  const upstream = profile.upstream_routing || {};
  const httpBehavior = profile.http_behavior || {};
  const httpHeaders = profile.http_headers || {};
  const security = profile.security_behavior_and_limits || {};
  const antibot = profile.security_antibot || {};
  const authBasic = profile.security_auth_basic || {};
  const country = profile.security_country_policy || {};
  const modsecurity = profile.security_modsecurity || {};
  return {
    ...draft,
    primary_host: front.server_name || draft.primary_host,
    security_mode: front.security_mode || draft.security_mode,
    auto_lets_encrypt: Boolean(front.auto_lets_encrypt ?? draft.auto_lets_encrypt),
    use_lets_encrypt_staging: Boolean(front.use_lets_encrypt_staging ?? draft.use_lets_encrypt_staging),
    use_lets_encrypt_wildcard: Boolean(front.use_lets_encrypt_wildcard ?? draft.use_lets_encrypt_wildcard),
    certificate_authority_server: front.certificate_authority_server || draft.certificate_authority_server,
    acme_account_email: normalizeEmail(front.acme_account_email || draft.acme_account_email),
    use_reverse_proxy: Boolean(upstream.use_reverse_proxy ?? draft.use_reverse_proxy),
    reverse_proxy_host: resolveReverseProxyHost(draft, upstream.reverse_proxy_host) || draft.reverse_proxy_host,
    reverse_proxy_url: upstream.reverse_proxy_url || draft.reverse_proxy_url,
    reverse_proxy_custom_host: upstream.reverse_proxy_custom_host || draft.reverse_proxy_custom_host,
    reverse_proxy_ssl_sni: Boolean(upstream.reverse_proxy_ssl_sni ?? draft.reverse_proxy_ssl_sni),
    reverse_proxy_ssl_sni_name: upstream.reverse_proxy_ssl_sni_name || draft.reverse_proxy_ssl_sni_name,
    reverse_proxy_websocket: Boolean(upstream.reverse_proxy_websocket ?? draft.reverse_proxy_websocket),
    reverse_proxy_keepalive: Boolean(upstream.reverse_proxy_keepalive ?? draft.reverse_proxy_keepalive),
    pass_host_header: !(Boolean(upstream.disable_host_header ?? !draft.pass_host_header)),
    send_x_forwarded_for: !(Boolean(upstream.disable_x_forwarded_for ?? !draft.send_x_forwarded_for)),
    send_x_forwarded_proto: !(Boolean(upstream.disable_x_forwarded_proto ?? !draft.send_x_forwarded_proto)),
    send_x_real_ip: Boolean(upstream.enable_x_real_ip ?? draft.send_x_real_ip),
    allowed_methods: normalizeStringArray(httpBehavior.allowed_methods).length ? normalizeStringArray(httpBehavior.allowed_methods) : draft.allowed_methods,
    max_client_size: httpBehavior.max_client_size || draft.max_client_size,
    http2: Boolean(httpBehavior.http2 ?? draft.http2),
    http3: Boolean(httpBehavior.http3 ?? draft.http3),
    ssl_protocols: normalizeStringArray(httpBehavior.ssl_protocols).length ? normalizeStringArray(httpBehavior.ssl_protocols) : draft.ssl_protocols,
    cookie_flags: httpHeaders.cookie_flags || draft.cookie_flags,
    content_security_policy: httpHeaders.content_security_policy || draft.content_security_policy,
    permissions_policy: normalizeStringArray(httpHeaders.permissions_policy),
    keep_upstream_headers: normalizeStringArray(httpHeaders.keep_upstream_headers).length ? normalizeStringArray(httpHeaders.keep_upstream_headers) : draft.keep_upstream_headers,
    referrer_policy: httpHeaders.referrer_policy || draft.referrer_policy,
    use_cors: Boolean(httpHeaders.use_cors ?? draft.use_cors),
    cors_allowed_origins: normalizeStringArray(httpHeaders.cors_allowed_origins).length ? normalizeStringArray(httpHeaders.cors_allowed_origins) : draft.cors_allowed_origins,
    use_bad_behavior: Boolean(security.use_bad_behavior ?? draft.use_bad_behavior),
    bad_behavior_status_codes: normalizeArray(security.bad_behavior_status_codes).map((item) => Number(item)).filter((item) => Number.isInteger(item)),
    bad_behavior_ban_time_seconds: Number(security.bad_behavior_ban_time_seconds ?? draft.bad_behavior_ban_time_seconds),
    bad_behavior_threshold: Number(security.bad_behavior_threshold ?? draft.bad_behavior_threshold),
    bad_behavior_count_time_seconds: Number(security.bad_behavior_count_time_seconds ?? draft.bad_behavior_count_time_seconds),
    ban_escalation_enabled: Boolean(security.ban_escalation_enabled ?? draft.ban_escalation_enabled),
    ban_escalation_scope: BAN_SCOPE_VALUES.includes(String(security.ban_escalation_scope || "").trim().toLowerCase())
      ? String(security.ban_escalation_scope || "").trim().toLowerCase()
      : draft.ban_escalation_scope,
    ban_escalation_stages_seconds: normalizeBanEscalationStages(
      security.ban_escalation_stages_seconds,
      Number(security.bad_behavior_ban_time_seconds ?? draft.bad_behavior_ban_time_seconds)
    ),
    use_exceptions: Boolean(security.use_exceptions ?? draft.use_exceptions),
    exceptions_ip: normalizeStringArray(security.exceptions_ip),
    use_blacklist: Boolean(security.use_blacklist ?? draft.use_blacklist),
    use_dnsbl: Boolean(security.use_dnsbl ?? draft.use_dnsbl),
    blacklist_ip: normalizeStringArray(security.blacklist_ip),
    blacklist_rdns: normalizeStringArray(security.blacklist_rdns),
    blacklist_asn: normalizeStringArray(security.blacklist_asn),
    blacklist_user_agent: normalizeStringArray(security.blacklist_user_agent),
    blacklist_uri: normalizeStringArray(security.blacklist_uri),
    blacklist_ip_urls: normalizeStringArray(security.blacklist_ip_urls),
    blacklist_rdns_urls: normalizeStringArray(security.blacklist_rdns_urls),
    blacklist_asn_urls: normalizeStringArray(security.blacklist_asn_urls),
    blacklist_user_agent_urls: normalizeStringArray(security.blacklist_user_agent_urls),
    blacklist_uri_urls: normalizeStringArray(security.blacklist_uri_urls),
    use_limit_conn: Boolean(security.use_limit_conn ?? draft.use_limit_conn),
    limit_conn_max_http1: Number(security.limit_conn_max_http1 ?? draft.limit_conn_max_http1),
    limit_conn_max_http2: Number(security.limit_conn_max_http2 ?? draft.limit_conn_max_http2),
    limit_conn_max_http3: Number(security.limit_conn_max_http3 ?? draft.limit_conn_max_http3),
    use_limit_req: Boolean(security.use_limit_req ?? draft.use_limit_req),
    limit_req_url: security.limit_req_url || draft.limit_req_url,
    limit_req_rate: security.limit_req_rate || draft.limit_req_rate,
    custom_limit_rules: normalizeCustomLimitRules(security.custom_limit_rules),
    antibot_challenge: antibot.antibot_challenge || draft.antibot_challenge,
    antibot_uri: antibot.antibot_uri || draft.antibot_uri,
    antibot_recaptcha_score: Number(antibot.antibot_recaptcha_score ?? draft.antibot_recaptcha_score),
    antibot_recaptcha_sitekey: antibot.antibot_recaptcha_sitekey || draft.antibot_recaptcha_sitekey,
    antibot_recaptcha_secret: antibot.antibot_recaptcha_secret || draft.antibot_recaptcha_secret,
    antibot_hcaptcha_sitekey: antibot.antibot_hcaptcha_sitekey || draft.antibot_hcaptcha_sitekey,
    antibot_hcaptcha_secret: antibot.antibot_hcaptcha_secret || draft.antibot_hcaptcha_secret,
    antibot_turnstile_sitekey: antibot.antibot_turnstile_sitekey || draft.antibot_turnstile_sitekey,
    antibot_turnstile_secret: antibot.antibot_turnstile_secret || draft.antibot_turnstile_secret,
    use_auth_basic: Boolean(authBasic.use_auth_basic ?? draft.use_auth_basic),
    auth_basic_location: authBasic.auth_basic_location || draft.auth_basic_location,
    auth_basic_user: authBasic.auth_basic_user || draft.auth_basic_user,
    auth_basic_password: authBasic.auth_basic_password || draft.auth_basic_password,
    auth_basic_text: authBasic.auth_basic_text || draft.auth_basic_text,
    blacklist_country: normalizeStringArray(country.blacklist_country),
    whitelist_country: normalizeStringArray(country.whitelist_country),
    use_modsecurity: Boolean(modsecurity.use_modsecurity ?? draft.use_modsecurity),
    use_modsecurity_crs_plugins: Boolean(modsecurity.use_modsecurity_crs_plugins ?? draft.use_modsecurity_crs_plugins),
    use_modsecurity_custom_configuration: Boolean(modsecurity.use_modsecurity_custom_configuration ?? draft.use_modsecurity_custom_configuration),
    modsecurity_crs_version: String(modsecurity.modsecurity_crs_version || draft.modsecurity_crs_version),
    modsecurity_crs_plugins: normalizeStringArray(modsecurity.modsecurity_crs_plugins),
    modsecurity_custom_path: modsecurity.custom_configuration?.path || draft.modsecurity_custom_path,
    modsecurity_custom_content: modsecurity.custom_configuration?.content || draft.modsecurity_custom_content
  };
}

function draftToEasyProfile(draft) {
  const siteID = String(draft.id || "").trim().toLowerCase();
  const primaryHost = String(draft.primary_host || "").trim().toLowerCase();
  const reverseProxyHost = resolveReverseProxyHost(draft, draft.reverse_proxy_host);
  const reverseProxyURL = String(draft.reverse_proxy_url || "").trim();
  const limitReqURL = String(draft.limit_req_url || "").trim();
  const limitReqRateRaw = String(draft.limit_req_rate || "").trim().toLowerCase().replace(/\s+/g, "");
  const limitReqRate = /^\d+r\/s$/.test(limitReqRateRaw) ? limitReqRateRaw : "100r/s";
  const authBasicLocation = "sitewide";
  const authBasicText = String(draft.auth_basic_text || "").trim() || "Restricted area";
  const customPath = String(draft.modsecurity_custom_path || "").trim() || "modsec/anomaly_score.conf";
  const securityMode = ["transparent", "monitor", "block"].includes(String(draft.security_mode || "").trim().toLowerCase())
    ? String(draft.security_mode || "").trim().toLowerCase()
    : "block";
  const banEscalationScope = BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase())
    ? String(draft.ban_escalation_scope || "").trim().toLowerCase()
    : "all_sites";
  const banEscalationStages = normalizeBanEscalationStages(
    draft.ban_escalation_stages_seconds,
    draft.bad_behavior_ban_time_seconds
  );

  return {
    site_id: siteID,
    front_service: {
      server_name: primaryHost,
      security_mode: securityMode,
      auto_lets_encrypt: draft.auto_lets_encrypt,
      use_lets_encrypt_staging: draft.use_lets_encrypt_staging,
      use_lets_encrypt_wildcard: draft.use_lets_encrypt_wildcard,
      certificate_authority_server: draft.certificate_authority_server,
      acme_account_email: normalizeEmail(draft.acme_account_email)
    },
    upstream_routing: {
      use_reverse_proxy: draft.use_reverse_proxy,
      reverse_proxy_host: reverseProxyHost,
      reverse_proxy_url: reverseProxyURL.startsWith("/") ? reverseProxyURL : "/",
      reverse_proxy_custom_host: draft.reverse_proxy_custom_host,
      reverse_proxy_ssl_sni: draft.reverse_proxy_ssl_sni,
      reverse_proxy_ssl_sni_name: draft.reverse_proxy_ssl_sni_name,
      reverse_proxy_websocket: draft.reverse_proxy_websocket,
      reverse_proxy_keepalive: draft.reverse_proxy_keepalive,
      disable_host_header: !draft.pass_host_header,
      disable_x_forwarded_for: !draft.send_x_forwarded_for,
      disable_x_forwarded_proto: !draft.send_x_forwarded_proto,
      enable_x_real_ip: draft.send_x_real_ip
    },
    http_behavior: {
      allowed_methods: draft.allowed_methods,
      max_client_size: draft.max_client_size,
      http2: draft.http2,
      http3: draft.http3,
      ssl_protocols: draft.ssl_protocols
    },
    http_headers: {
      cookie_flags: draft.cookie_flags,
      content_security_policy: draft.content_security_policy,
      permissions_policy: draft.permissions_policy,
      keep_upstream_headers: draft.keep_upstream_headers,
      referrer_policy: draft.referrer_policy,
      use_cors: draft.use_cors,
      cors_allowed_origins: draft.cors_allowed_origins
    },
    security_behavior_and_limits: {
      use_bad_behavior: draft.use_bad_behavior,
      bad_behavior_status_codes: draft.bad_behavior_status_codes,
      bad_behavior_ban_time_seconds: draft.bad_behavior_ban_time_seconds,
      bad_behavior_threshold: draft.bad_behavior_threshold,
      bad_behavior_count_time_seconds: draft.bad_behavior_count_time_seconds,
      ban_escalation_enabled: draft.ban_escalation_enabled,
      ban_escalation_scope: banEscalationScope,
      ban_escalation_stages_seconds: banEscalationStages,
      use_exceptions: draft.use_exceptions,
      exceptions_ip: draft.exceptions_ip,
      use_blacklist: draft.use_blacklist,
      use_dnsbl: draft.use_dnsbl,
      blacklist_ip: draft.blacklist_ip,
      blacklist_rdns: draft.blacklist_rdns,
      blacklist_asn: draft.blacklist_asn,
      blacklist_user_agent: draft.blacklist_user_agent,
      blacklist_uri: draft.blacklist_uri,
      blacklist_ip_urls: draft.blacklist_ip_urls,
      blacklist_rdns_urls: draft.blacklist_rdns_urls,
      blacklist_asn_urls: draft.blacklist_asn_urls,
      blacklist_user_agent_urls: draft.blacklist_user_agent_urls,
      blacklist_uri_urls: draft.blacklist_uri_urls,
      use_limit_conn: draft.use_limit_conn,
      limit_conn_max_http1: draft.limit_conn_max_http1,
      limit_conn_max_http2: draft.limit_conn_max_http2,
      limit_conn_max_http3: draft.limit_conn_max_http3,
      use_limit_req: draft.use_limit_req,
      limit_req_url: limitReqURL.startsWith("/") ? limitReqURL : "/",
      limit_req_rate: limitReqRate,
      custom_limit_rules: normalizeCustomLimitRules(draft.custom_limit_rules)
    },
    security_antibot: {
      antibot_challenge: draft.antibot_challenge,
      antibot_uri: draft.antibot_uri,
      antibot_recaptcha_score: draft.antibot_recaptcha_score,
      antibot_recaptcha_sitekey: draft.antibot_recaptcha_sitekey,
      antibot_recaptcha_secret: draft.antibot_recaptcha_secret,
      antibot_hcaptcha_sitekey: draft.antibot_hcaptcha_sitekey,
      antibot_hcaptcha_secret: draft.antibot_hcaptcha_secret,
      antibot_turnstile_sitekey: draft.antibot_turnstile_sitekey,
      antibot_turnstile_secret: draft.antibot_turnstile_secret
    },
    security_auth_basic: {
      use_auth_basic: draft.use_auth_basic,
      auth_basic_location: authBasicLocation,
      auth_basic_user: draft.auth_basic_user,
      auth_basic_password: draft.auth_basic_password,
      auth_basic_text: authBasicText
    },
    security_country_policy: {
      blacklist_country: draft.blacklist_country,
      whitelist_country: draft.whitelist_country
    },
    security_modsecurity: {
      use_modsecurity: draft.use_modsecurity,
      use_modsecurity_crs_plugins: draft.use_modsecurity_crs_plugins,
      use_modsecurity_custom_configuration: draft.use_modsecurity_custom_configuration,
      modsecurity_crs_version: draft.modsecurity_crs_version,
      modsecurity_crs_plugins: draft.modsecurity_crs_plugins,
      custom_configuration: {
        path: customPath,
        content: draft.modsecurity_custom_content
      }
    }
  };
}

function validateDraft(draft, ctx) {
  if (!draft.id.trim()) {
    return ctx.t("sites.validation.siteIdRequired");
  }
  if (!draft.primary_host.trim()) {
    return ctx.t("sites.validation.primaryHostRequired");
  }
  if (!draft.upstream_id.trim()) {
    return ctx.t("sites.validation.upstreamIdRequired");
  }
  if (!draft.upstream_host.trim()) {
    return ctx.t("sites.validation.upstreamHostRequired");
  }
  if (!Number.isInteger(draft.upstream_port) || draft.upstream_port < 1 || draft.upstream_port > 65535) {
    return ctx.t("sites.validation.portRange");
  }
  if (!normalizeStringArray(draft.allowed_methods).length) {
    return ctx.t("sites.validation.allowedMethodsRequired");
  }
  if (draft.use_bad_behavior && !normalizeArray(draft.bad_behavior_status_codes).length) {
    return ctx.t("sites.validation.badBehaviorStatusCodesRequired");
  }
  if (draft.use_bad_behavior && (!Number.isFinite(draft.bad_behavior_ban_time_seconds) || draft.bad_behavior_ban_time_seconds < 0)) {
    return ctx.t("sites.validation.badBehaviorBanDuration");
  }
  if (draft.ban_escalation_enabled) {
    if (!BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase())) {
      return ctx.t("sites.validation.banEscalationScope");
    }
    const stages = normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds);
    if (!stages.length) {
      return ctx.t("sites.validation.banEscalationStagesRequired");
    }
    if (stages.length > 12) {
      return ctx.t("sites.validation.banEscalationStagesLimit");
    }
    for (let i = 0; i < stages.length; i += 1) {
      const value = stages[i];
      if (!Number.isFinite(value) || value < 0) {
        return ctx.t("sites.validation.banEscalationStageValue");
      }
      if (value === 0 && i !== stages.length - 1) {
        return ctx.t("sites.validation.banEscalationPermanentLast");
      }
    }
  }
  if (draft.use_limit_req && !String(draft.limit_req_rate || "").trim()) {
    return ctx.t("sites.validation.limitReqRateRequired");
  }
  if (draft.use_limit_req && !/^\d+r\/s$/i.test(String(draft.limit_req_rate || "").trim().replace(/\s+/g, ""))) {
    return ctx.t("sites.validation.limitReqRateFormat");
  }
  if (normalizeCustomLimitRules(draft.custom_limit_rules).length > 32) {
    return ctx.t("sites.validation.customLimitRulesLimit");
  }
  for (const rule of normalizeCustomLimitRules(draft.custom_limit_rules)) {
    if (!rule.path.startsWith("/")) {
      return ctx.t("sites.validation.customLimitPathFormat");
    }
    if (!/^\d+r\/s$/i.test(rule.rate)) {
      return ctx.t("sites.validation.customLimitRateFormat");
    }
  }
  if (draft.use_auth_basic && !String(draft.auth_basic_user || "").trim()) {
    return ctx.t("sites.validation.authBasicUserRequired");
  }
  if (draft.use_modsecurity_custom_configuration && !String(draft.modsecurity_custom_path || "").trim()) {
    return ctx.t("sites.validation.modsecCustomPathRequired");
  }
  return "";
}

function downloadJSON(filename, payload) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function downloadText(filename, content, type = "text/plain;charset=utf-8") {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function downloadBlob(filename, blob) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function toEnvKey(field) {
  return `WAF_SITE_${String(field || "").toUpperCase()}`;
}

function normalizeEnvPrimitive(value) {
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? String(value) : "0";
  }
  return String(value ?? "");
}

function draftToEnvText(draft) {
  const baseDraft = defaultSiteDraft();
  const lines = [];
  for (const field of Object.keys(baseDraft)) {
    const value = draft?.[field];
    const rendered = Array.isArray(value) ? JSON.stringify(value) : normalizeEnvPrimitive(value);
    lines.push(`${toEnvKey(field)}=${rendered}`);
  }
  return `${lines.join("\n")}\n`;
}

function parseBooleanEnv(value) {
  const normalized = String(value || "").trim().toLowerCase();
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  throw new Error(`invalid boolean value: ${value}`);
}

function parseNumberEnv(value) {
  const num = Number(String(value || "").trim());
  if (!Number.isFinite(num)) {
    throw new Error(`invalid number value: ${value}`);
  }
  return num;
}

function parseArrayEnv(value) {
  const trimmed = String(value || "").trim();
  if (!trimmed) {
    return [];
  }
  if (trimmed.startsWith("[")) {
    const parsed = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) {
      throw new Error(`invalid array value: ${value}`);
    }
    return parsed;
  }
  return trimmed
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}


function userPermissionsSet(ctx) {
  const items = Array.isArray(ctx?.currentUser?.permissions) ? ctx.currentUser.permissions : [];
  return new Set(items.map((item) => String(item || "").trim().toLowerCase()).filter(Boolean));
}

function requirePermissions(ctx, requiredPermissions, errorKey) {
  const required = Array.isArray(requiredPermissions) ? requiredPermissions : [];
  if (!required.length) {
    return;
  }
  const granted = userPermissionsSet(ctx);
  const missing = required
    .map((item) => String(item || "").trim().toLowerCase())
    .filter((item) => item && !granted.has(item));
  if (!missing.length) {
    return;
  }
  const permissions = missing.join(", ");
  throw new Error(ctx.t(errorKey, { permissions }));
}


function envToDraft(text) {
  const baseDraft = defaultSiteDraft();
  const knownFields = new Set(Object.keys(baseDraft));
  const envToField = new Map(Object.keys(baseDraft).map((field) => [toEnvKey(field), field]));
  const parsed = {};
  const presentFields = new Set();

  const lines = String(text || "").split(/\r?\n/);
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) {
      continue;
    }
    const eq = line.indexOf("=");
    if (eq <= 0) {
      throw new Error(`invalid env line: ${line}`);
    }
    const key = line.slice(0, eq).trim();
    const rawValue = line.slice(eq + 1);
    const field = envToField.get(key);
    if (!field || !knownFields.has(field)) {
      throw new Error(`unknown env parameter: ${key}`);
    }
    const template = baseDraft[field];
    if (Array.isArray(template)) {
      parsed[field] = parseArrayEnv(rawValue);
    } else if (typeof template === "boolean") {
      parsed[field] = parseBooleanEnv(rawValue);
    } else if (typeof template === "number") {
      parsed[field] = parseNumberEnv(rawValue);
    } else {
      parsed[field] = String(rawValue ?? "");
    }
    presentFields.add(field);
  }

  const missingFields = Object.keys(baseDraft).filter((field) => !presentFields.has(field));
  return {
    draft: { ...baseDraft, ...parsed },
    missingFields
  };
}

function buildImportPayloadFromDraft(draft) {
  const normalized = ensureControlPlaneAccessManagementMethods({ ...draft });
  const site = {
    id: normalized.id.trim().toLowerCase(),
    primary_host: normalized.primary_host.trim().toLowerCase(),
    enabled: Boolean(normalized.enabled)
  };
  const upstream = {
    id: computeUpstreamID(site.id),
    site_id: site.id,
    host: normalized.upstream_host.trim(),
    port: Number(normalized.upstream_port || 80),
    scheme: String(normalized.upstream_scheme || "http").toLowerCase()
  };
  const tls = normalized.tls_enabled
    ? {
        site_id: site.id,
        certificate_id: (normalized.certificate_id.trim() || `${site.id}-tls`).toLowerCase()
      }
    : null;
  return {
    draft: { ...normalized, id: site.id, upstream_id: upstream.id },
    site,
    upstream,
    tls,
    easyProfile: draftToEasyProfile({ ...normalized, id: site.id, upstream_id: upstream.id })
  };
}

function diffObjects(previous, next, path = "") {
  const left = previous && typeof previous === "object" ? previous : {};
  const right = next && typeof next === "object" ? next : {};
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);
  const lines = [];
  for (const key of keys) {
    const currentPath = path ? `${path}.${key}` : key;
    const l = left[key];
    const r = right[key];
    const bothObjects = l && r && typeof l === "object" && typeof r === "object" && !Array.isArray(l) && !Array.isArray(r);
    if (bothObjects) {
      lines.push(...diffObjects(l, r, currentPath));
      continue;
    }
    if (JSON.stringify(l) !== JSON.stringify(r)) {
      lines.push(`${currentPath}: ${JSON.stringify(l)} -> ${JSON.stringify(r)}`);
    }
  }
  return lines;
}

async function applyImportPayload(ctx, payload) {
  const { draft, site, upstream, tls, easyProfile } = payload;
  const existingSite = await ctx.api.get(`/api/sites/${encodeURIComponent(site.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
  const existingUpstream = await ctx.api.get(`/api/upstreams/${encodeURIComponent(upstream.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
  const existingTLSConfig = await ctx.api.get(`/api/tls-configs/${encodeURIComponent(site.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));
  const existingEasy = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`).catch((error) => (error?.status === 404 ? null : Promise.reject(error)));

  await upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig);
  await putWithPostFallback(ctx, `/api/easy-site-profiles/${encodeURIComponent(site.id)}`, easyProfile);

  const diffLines = [];
  if (existingSite) {
    diffLines.push(...diffObjects(existingSite, site, "site"));
  }
  if (existingUpstream) {
    diffLines.push(...diffObjects(existingUpstream, upstream, "upstream"));
  }
  if (tls && existingTLSConfig) {
    diffLines.push(...diffObjects(existingTLSConfig, tls, "tls"));
  }
  if (existingEasy) {
    diffLines.push(...diffObjects(existingEasy, easyProfile, "easy"));
  }
  return {
    updatedExisting: Boolean(existingSite),
    diffLines
  };
}

async function importServicesJSON(file, ctx) {
  requirePermissions(ctx, ["sites.write", "upstreams.write"], "sites.error.importJsonPermissions");
  const payload = JSON.parse(await file.text());
  const sites = normalizeArray(payload.sites);
  const upstreams = normalizeArray(payload.upstreams);
  for (const site of sites) {
    try {
      await ctx.api.post("/api/sites", site);
    } catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) {
        await ctx.api.put(`/api/sites/${encodeURIComponent(site.id)}`, site);
      } else {
        throw error;
      }
    }
  }
  for (const upstream of upstreams) {
    try {
      await ctx.api.post("/api/upstreams", upstream);
    } catch (error) {
      const message = String(error?.payload?.error || "");
      if (message.includes("already exists")) {
        await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstream.id)}`, upstream);
      } else {
        throw error;
      }
    }
  }
  return { imported: sites.length };
}

function renderListView(state, ctx) {
  return `
    <div class="waf-page-stack">
      <section class="waf-card waf-services-card">
        <div class="waf-card-head waf-services-toolbar">
          <div>
            <h3>${escapeHtml(ctx.t("sites.list.title"))}</h3>
            <div class="muted">${escapeHtml(ctx.t("sites.list.subtitle"))}</div>
          </div>
          <div class="waf-actions">
            <button class="btn primary btn-sm" type="button" id="services-create">${escapeHtml(ctx.t("sites.action.createSite"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-import">${escapeHtml(ctx.t("sites.action.import"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-export">${escapeHtml(ctx.t("sites.action.export"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-delete-selected">${escapeHtml(ctx.t("sites.action.deleteSelected"))}</button>
            <button class="btn ghost btn-sm" type="button" id="services-refresh">${escapeHtml(ctx.t("common.refresh"))}</button>
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          <div id="sites-feedback"></div>
          <div class="waf-services-filters">
            <div class="waf-field">
              <label for="services-search">${escapeHtml(ctx.t("sites.filters.search"))}</label>
              <input id="services-search" value="${escapeHtml(state.search)}" placeholder="${escapeHtml(ctx.t("sites.filters.searchPlaceholder"))}">
            </div>
            <div class="waf-field">
              <label for="services-sort">${escapeHtml(ctx.t("sites.filters.sort"))}</label>
              <select id="services-sort">
                <option value="updated-desc"${state.sort === "updated-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.updatedDesc"))}</option>
                <option value="name-asc"${state.sort === "name-asc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.nameAsc"))}</option>
                <option value="name-desc"${state.sort === "name-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.nameDesc"))}</option>
                <option value="created-desc"${state.sort === "created-desc" ? " selected" : ""}>${escapeHtml(ctx.t("sites.sort.createdDesc"))}</option>
              </select>
            </div>
          </div>
          <div class="waf-table-wrap">
            <table class="waf-table waf-services-table">
              <thead>
                <tr>
                  <th class="waf-check-col">
                    <input type="checkbox" id="services-select-all"${state.filteredSites.length && state.filteredSites.every((site) => state.selectedSiteIDs.has(site.id)) ? " checked" : ""}>
                  </th>
                  <th>${escapeHtml(ctx.t("sites.table.name"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.upstream"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.tls"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.status"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.updated"))}</th>
                  <th>${escapeHtml(ctx.t("sites.table.actions"))}</th>
                </tr>
              </thead>
              <tbody>
                ${state.filteredSites.length ? state.filteredSites.map((site) => {
                  const upstream = state.upstreamsBySite.get(site.id)?.[0] || null;
                  const tls = state.tlsBySite.get(site.id);
                  const tlsState = tls
                    ? "managed"
                    : (state.certificateBySiteID.get(site.id) || state.certificateByHost.get(normalizeHost(site.primary_host)) ? "detected" : "missing");
                  const serviceURL = resolvePublicServiceURL(site, tlsState);
                  return `
                    <tr class="waf-table-row-clickable" data-open-site-edit="${escapeHtml(site.id)}">
                      <td class="waf-check-col">
                        <input type="checkbox" data-select-site="${escapeHtml(site.id)}"${state.selectedSiteIDs.has(site.id) ? " checked" : ""}>
                      </td>
                      <td>
                        <button class="waf-link-button" type="button" data-open-service="${escapeHtml(serviceURL)}" title="${escapeHtml(ctx.t("sites.action.openService"))}">${escapeHtml(site.primary_host || site.id)}</button>
                      </td>
                      <td>${upstream ? `${escapeHtml(upstream.host)}:${escapeHtml(String(upstream.port))}` : escapeHtml(ctx.t("common.notSet"))}</td>
                      <td>
                        ${tlsState === "managed"
                          ? `<span class="badge badge-success">${escapeHtml(ctx.t("sites.state.tlsManaged"))}</span>`
                          : tlsState === "detected"
                            ? `<span class="badge badge-warning">${escapeHtml(ctx.t("sites.state.tlsDetected"))}</span>`
                            : `<span class="badge badge-neutral">${escapeHtml(ctx.t("sites.state.tlsMissing"))}</span>`}
                      </td>
                      <td>${statusBadge(site.enabled ? "active" : "failed")}</td>
                      <td>${escapeHtml(formatDate(site.updated_at || site.created_at))}</td>
                      <td>
                        <div class="waf-actions">
                          <button class="btn ghost btn-sm" type="button" data-open-site="${escapeHtml(site.id)}">${escapeHtml(ctx.t("common.edit"))}</button>
                        </div>
                      </td>
                    </tr>
                  `;
                }).join("") : `
                  <tr>
                    <td colspan="7">
                      <div class="waf-empty">${escapeHtml(ctx.t("sites.empty.sites"))}</div>
                    </td>
                  </tr>
                `}
              </tbody>
            </table>
          </div>
          <input id="services-import-file" type="file" accept=".json,.env,application/json,text/plain" multiple class="waf-hidden">
        </div>
      </section>
    </div>
  `;
}

function renderModeTabs(ctx) {
  return `
    <div class="waf-mode-switch">
      <button class="btn primary btn-sm" type="button">${escapeHtml(ctx.t("sites.mode.easy"))}</button>
      <button class="btn ghost btn-sm" type="button" disabled>${escapeHtml(ctx.t("sites.mode.advanced"))}</button>
      <button class="btn ghost btn-sm" type="button" disabled>${escapeHtml(ctx.t("sites.mode.raw"))}</button>
    </div>
    <div class="waf-note">${escapeHtml(ctx.t("sites.mode.note"))}</div>
  `;
}

function renderWizardNav(activeTab, ctx) {
  const items = [
    { id: "front", title: ctx.t("sites.wizard.front.title"), subtitle: ctx.t("sites.wizard.front.subtitle") },
    { id: "upstream", title: ctx.t("sites.wizard.upstream.title"), subtitle: ctx.t("sites.wizard.upstream.subtitle") },
    { id: "http", title: ctx.t("sites.easy.tab.http.title"), subtitle: ctx.t("sites.easy.tab.http.subtitle") },
    { id: "headers", title: ctx.t("sites.easy.tab.headers.title"), subtitle: ctx.t("sites.easy.tab.headers.subtitle") },
    { id: "traffic", title: ctx.t("sites.easy.tab.traffic.title"), subtitle: ctx.t("sites.easy.tab.traffic.subtitle") },
    { id: "blocking", title: ctx.t("sites.easy.tab.blocking.title"), subtitle: ctx.t("sites.easy.tab.blocking.subtitle") },
    { id: "antibot", title: ctx.t("sites.easy.tab.antibot.title"), subtitle: ctx.t("sites.easy.tab.antibot.subtitle") },
    { id: "geo", title: ctx.t("sites.easy.tab.geo.title"), subtitle: ctx.t("sites.easy.tab.geo.subtitle") },
    { id: "modsec", title: ctx.t("sites.easy.tab.modsec.title"), subtitle: ctx.t("sites.easy.tab.modsec.subtitle") }
  ];
  return `
    <aside class="waf-service-wizard-nav" role="tablist" aria-label="${escapeHtml(ctx.t("sites.wizard.aria"))}">
      ${items.map((item, index) => `
        <button
          class="waf-service-wizard-item${activeTab === item.id ? " is-active" : ""}"
          type="button"
          role="tab"
          aria-selected="${activeTab === item.id ? "true" : "false"}"
          data-wizard-tab="${item.id}">
          <div class="waf-service-step-index">${index + 1}</div>
          <div>
            <div class="waf-list-title">${escapeHtml(item.title)}</div>
            <div class="waf-note">${escapeHtml(item.subtitle)}</div>
          </div>
        </button>
      `).join("")}
    </aside>
  `;
}

function renderDetailView(state, ctx) {
  const draft = state.draft;
  const isNew = state.route.mode === "create";
  const titleKey = isNew ? "sites.editor.newTitle" : "sites.editor.editTitle";
  const subtitleKey = isNew ? "sites.editor.newSubtitle" : "sites.editor.editSubtitle";
  const searchQuery = state.settingsSearch.trim().toLowerCase();
  const searchMatches = searchQuery
    ? SETTINGS_SEARCH_INDEX.filter((item) => {
      const label = String(ctx.t(item.labelKey) || "").toLowerCase();
      const id = String(item.id || "").toLowerCase();
      const selector = String(item.selector || "").toLowerCase();
      return label.includes(searchQuery) || id.includes(searchQuery) || selector.includes(searchQuery);
    }).slice(0, 10)
    : [];
  return `
    <div class="waf-page-stack">
      <section class="waf-card waf-service-shell-card">
        <div class="waf-card-head">
          <div>
            <h3>${escapeHtml(ctx.t(titleKey))}</h3>
            <div class="muted">${escapeHtml(ctx.t(subtitleKey, { site: draft.primary_host || draft.id || ctx.t("sites.editor.newSiteLabel") }))}</div>
          </div>
          <div class="waf-actions">
            <button class="btn ghost btn-sm" type="button" id="service-back">${escapeHtml(ctx.t("common.back"))}</button>
            ${!isNew ? `<button class="btn ghost btn-sm" type="button" id="service-delete">${escapeHtml(ctx.t("common.delete"))}</button>` : ""}
          </div>
        </div>
        <div class="waf-card-body waf-stack">
          ${renderModeTabs(ctx)}
          <div class="waf-field waf-service-settings-search">
            <label for="service-settings-search">${escapeHtml(ctx.t("sites.search.title"))}</label>
            <input id="service-settings-search" value="${escapeHtml(state.settingsSearch)}" placeholder="${escapeHtml(ctx.t("sites.search.placeholder"))}">
            ${searchQuery ? `
              <div class="waf-service-settings-search-dropdown">
                ${searchMatches.length ? searchMatches.map((item) => `
                  <button type="button" class="waf-service-settings-search-item" data-settings-result="${escapeHtml(item.id)}" data-settings-tab="${escapeHtml(item.tab)}" data-settings-selector="${escapeHtml(item.selector)}">
                    ${escapeHtml(ctx.t(item.labelKey))}
                  </button>
                `).join("") : `<div class="waf-note">${escapeHtml(ctx.t("sites.search.empty"))}</div>`}
              </div>
            ` : ""}
          </div>
        </div>
      </section>
      <div class="waf-service-editor-layout">
        ${renderWizardNav(state.activeTab, ctx)}
        <section class="waf-card waf-service-editor-card">
          <div class="waf-card-body waf-stack">
            <div id="sites-feedback"></div>
            <form id="service-editor-form" class="waf-form waf-stack">
              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "front" ? "" : " waf-hidden"}" data-tab-panel="front">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.wizard.front.title"))}</div>
                <div class="waf-form-grid">
                  <div class="waf-field">
                    <label for="service-host">${escapeHtml(ctx.t("sites.easy.front.serverName"))}</label>
                    <input id="service-host" value="${escapeHtml(draft.primary_host)}" placeholder="${escapeHtml(ctx.t("sites.editor.hostPlaceholder"))}">
                  </div>
                  <div class="waf-field">
                    <label for="service-id">${escapeHtml(ctx.t("sites.easy.front.serviceId"))}</label>
                    <input id="service-id" value="${escapeHtml(draft.id)}" placeholder="${escapeHtml(ctx.t("sites.editor.idPlaceholder"))}">
                  </div>
                  <div class="waf-field">
                    <label for="service-security-mode">${escapeHtml(ctx.t("sites.editor.securityMode"))}</label>
                    <select id="service-security-mode">
                      <option value="transparent"${draft.security_mode === "transparent" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.transparent"))}</option>
                      <option value="monitor"${draft.security_mode === "monitor" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.monitor"))}</option>
                      <option value="block"${draft.security_mode === "block" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.security.block"))}</option>
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="service-ca-server">${escapeHtml(ctx.t("sites.easy.front.caServer"))}</label>
                    <select id="service-ca-server">
                      <option value="letsencrypt"${draft.certificate_authority_server === "letsencrypt" ? " selected" : ""}>letsencrypt</option>
                      <option value="zerossl"${draft.certificate_authority_server === "zerossl" ? " selected" : ""}>zerossl</option>
                      <option value="custom"${draft.certificate_authority_server === "custom" ? " selected" : ""}>custom</option>
                      <option value="import"${draft.certificate_authority_server === "import" ? " selected" : ""}>${escapeHtml(ctx.t("sites.tls.importOption"))}</option>
                    </select>
                  </div>
                  <label class="waf-checkbox">
                    <input id="service-enabled" type="checkbox"${draft.enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.serviceEnabled"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-auto-lets-encrypt" type="checkbox"${draft.auto_lets_encrypt ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.autoLetsEncrypt"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-lets-encrypt-staging" type="checkbox"${draft.use_lets_encrypt_staging ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.letsEncryptStaging"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-lets-encrypt-wildcard" type="checkbox"${draft.use_lets_encrypt_wildcard ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.wildcardCertificates"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-tls-enabled" type="checkbox"${draft.tls_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.tlsEnabled"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-tls-self-signed" type="checkbox"${draft.tls_self_signed ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.front.tlsSelfSigned"))}</span>
                  </label>
                  <div class="waf-field waf-field-cert-id">
                    <label for="service-certificate-id">${escapeHtml(ctx.t("sites.tls.certificateId"))}</label>
                    <input id="service-certificate-id" value="${escapeHtml(draft.certificate_id)}" placeholder="${escapeHtml(ctx.t("sites.tls.certificatePlaceholder"))}">
                    <div class="waf-field" id="service-certificate-picker"${draft.certificate_authority_server === "import" ? "" : " style=\"display:none\""}>
                      <label for="service-import-certificate-search">${escapeHtml(ctx.t("sites.tls.importSelect"))}</label>
                      <input id="service-import-certificate-search" list="service-import-certificate-list" placeholder="${escapeHtml(ctx.t("sites.tls.importSelectPlaceholder"))}">
                      <datalist id="service-import-certificate-list">
                        ${state.certificates.map((certificate) => {
                          const id = String(certificate?.id || "").trim();
                          if (!id) {
                            return "";
                          }
                          const commonName = String(certificate?.common_name || "").trim();
                          const status = String(certificate?.status || "unknown").trim();
                          const label = commonName ? `${id} (${commonName}, ${status})` : `${id} (${status})`;
                          return `<option value="${escapeHtml(id)}" label="${escapeHtml(label)}"></option>`;
                        }).join("")}
                      </datalist>
                    </div>
                    <div class="waf-actions" id="service-certificate-import-actions"${draft.certificate_authority_server === "import" ? "" : " style=\"display:none\""}>
                      <button class="btn ghost btn-sm" type="button" id="service-certificate-import">${escapeHtml(ctx.t("sites.tls.importButton"))}</button>
                      <button class="btn ghost btn-sm" type="button" id="service-certificate-export">${escapeHtml(ctx.t("sites.tls.exportButton"))}</button>
                    </div>
                    <input id="service-certificate-archive-file" type="file" accept=".zip,application/zip,application/x-zip-compressed" class="waf-hidden">
                  </div>
                </div>
              </section>

              <section class="waf-subcard waf-stack${state.activeTab === "upstream" ? "" : " waf-hidden"}" data-tab-panel="upstream">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.wizard.upstream.title"))}</div>
                <input id="service-upstream-id" type="hidden" value="${escapeHtml(draft.upstream_id)}">
                <div class="waf-form-grid three waf-upstream-toggle-row">
                  <label class="waf-checkbox">
                    <input id="service-use-reverse-proxy" type="checkbox"${draft.use_reverse_proxy ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.useReverseProxy"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-keepalive" type="checkbox"${draft.reverse_proxy_keepalive ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.keepalive"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-websocket" type="checkbox"${draft.reverse_proxy_websocket ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.websocket"))}</span>
                  </label>
                </div>
                <div class="waf-upstream-layout">
                  <div class="waf-form-grid three">
                    <div class="waf-field">
                      <label for="service-reverse-proxy-custom-host">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyCustomHost"))}</label>
                      <input id="service-reverse-proxy-custom-host" value="${escapeHtml(draft.reverse_proxy_custom_host)}"${draft.pass_host_header ? "" : " disabled"}>
                    </div>
                    <div class="waf-field">
                    <label for="service-reverse-proxy-host">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyHost"))}</label>
                    <input id="service-reverse-proxy-host" value="${escapeHtml(draft.reverse_proxy_host)}">
                    </div>
                    <div class="waf-field">
                    <label for="service-reverse-proxy-url">${escapeHtml(ctx.t("sites.easy.upstream.reverseProxyUrl"))}</label>
                    <input id="service-reverse-proxy-url" value="${escapeHtml(draft.reverse_proxy_url)}">
                    </div>
                  </div>
                  <div class="waf-upstream-target-row">
                    <div class="waf-field waf-field-compact-xs">
                      <label for="service-upstream-scheme">${escapeHtml(ctx.t("sites.upstream.field.scheme"))}</label>
                      <select id="service-upstream-scheme">
                        <option value="http"${draft.upstream_scheme === "http" ? " selected" : ""}>http</option>
                        <option value="https"${draft.upstream_scheme === "https" ? " selected" : ""}>https</option>
                      </select>
                    </div>
                    <div class="waf-field waf-field-compact-md">
                      <label for="service-upstream-host">${escapeHtml(ctx.t("sites.upstream.field.host"))}</label>
                      <input id="service-upstream-host" value="${escapeHtml(draft.upstream_host)}">
                    </div>
                    <div class="waf-field waf-field-compact-xs">
                      <label for="service-upstream-port">${escapeHtml(ctx.t("sites.upstream.field.port"))}</label>
                      <input id="service-upstream-port" type="number" min="1" max="65535" value="${escapeHtml(String(draft.upstream_port))}">
                    </div>
                  </div>
                </div>
                <div class="waf-subframe waf-upstream-headers-frame">
                  <div class="waf-list-title-sm">${escapeHtml(ctx.t("sites.easy.upstream.headerForwardingTitle"))}</div>
                  <div class="waf-note">${escapeHtml(ctx.t("sites.easy.upstream.headerForwardingHint"))}</div>
                  <div class="waf-form-grid two">
                    <label class="waf-checkbox">
                      <input id="service-pass-host-header" type="checkbox"${draft.pass_host_header ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.passHostHeader"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-forwarded-for" type="checkbox"${draft.send_x_forwarded_for ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXForwardedFor"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-forwarded-proto" type="checkbox"${draft.send_x_forwarded_proto ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXForwardedProto"))}</span>
                    </label>
                    <label class="waf-checkbox">
                      <input id="service-send-x-real-ip" type="checkbox"${draft.send_x_real_ip ? " checked" : ""}>
                      <span>${escapeHtml(ctx.t("sites.easy.upstream.sendXRealIp"))}</span>
                    </label>
                  </div>
                </div>
                <div class="waf-form-grid two waf-upstream-sni-row">
                  <label class="waf-checkbox">
                    <input id="service-reverse-proxy-ssl-sni" type="checkbox"${draft.reverse_proxy_ssl_sni ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.upstream.useSslSni"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-reverse-proxy-ssl-sni-name">${escapeHtml(ctx.t("sites.easy.upstream.sslSniName"))}</label>
                    <input id="service-reverse-proxy-ssl-sni-name" value="${escapeHtml(draft.reverse_proxy_ssl_sni_name)}">
                  </div>
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "http" ? "" : " waf-hidden"}" data-tab-panel="http">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.http.title"))}</div>
                <div class="waf-form-grid">
                  ${renderListEditor("allowed_methods", ctx.t("sites.easy.http.allowedMethods"), draft.allowed_methods, "GET", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <div class="waf-field">
                    <label for="service-max-client-size">${escapeHtml(ctx.t("sites.easy.http.maxBodySize"))}</label>
                    <input id="service-max-client-size" value="${escapeHtml(draft.max_client_size)}">
                  </div>
                  ${renderListEditor("ssl_protocols", ctx.t("sites.easy.http.sslProtocols"), draft.ssl_protocols, "TLSv1.3", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <label class="waf-checkbox">
                    <input id="service-http2" type="checkbox"${draft.http2 ? " checked" : ""}>
                    <span>HTTP2</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-http3" type="checkbox"${draft.http3 ? " checked" : ""}>
                    <span>HTTP3</span>
                  </label>
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "headers" ? "" : " waf-hidden"}" data-tab-panel="headers">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.headers.title"))}</div>
                <div class="waf-form-grid">
                  <div class="waf-field">
                    <label for="service-cookie-flags">${escapeHtml(ctx.t("sites.easy.headers.cookieFlags"))}</label>
                    <input id="service-cookie-flags" value="${escapeHtml(draft.cookie_flags)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-referrer-policy">${escapeHtml(ctx.t("sites.easy.headers.referrerPolicy"))}</label>
                    <input id="service-referrer-policy" value="${escapeHtml(draft.referrer_policy)}">
                  </div>
                  <div class="waf-field full">
                    <label for="service-content-security-policy">${escapeHtml(ctx.t("sites.easy.headers.contentSecurityPolicy"))}</label>
                    <textarea id="service-content-security-policy" rows="3">${escapeHtml(draft.content_security_policy)}</textarea>
                  </div>
                  ${renderListEditor("permissions_policy", ctx.t("sites.easy.headers.permissionsPolicy"), draft.permissions_policy, "geolocation=()", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  ${renderListEditor("keep_upstream_headers", ctx.t("sites.easy.headers.keepUpstreamHeaders"), draft.keep_upstream_headers, "X-Forwarded-For", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <label class="waf-checkbox">
                    <input id="service-use-cors" type="checkbox"${draft.use_cors ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.headers.useCors"))}</span>
                  </label>
                  ${renderListEditor("cors_allowed_origins", ctx.t("sites.easy.headers.allowedOrigins"), draft.cors_allowed_origins, "https://app.example.com", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "traffic" ? "" : " waf-hidden"}" data-tab-panel="traffic">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.traffic.title"))}</div>
                <div class="waf-traffic-layout">
                  <div class="waf-stack">
                    <div class="waf-subcard waf-stack waf-antiddos-frame">
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.badBehavior"))}</div>
                    <div class="waf-form-grid">
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-bad-behavior" type="checkbox"${draft.use_bad_behavior ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBadBehavior"))}</span>
                      </label>
                      ${renderStatusCodesEditor(draft.bad_behavior_status_codes, ctx)}
                      <div class="waf-field">
                        <label for="service-bad-behavior-ban-time">${escapeHtml(ctx.t("sites.easy.traffic.banDurationSeconds"))}</label>
                        <input id="service-bad-behavior-ban-time" type="number" min="0" value="${escapeHtml(String(draft.bad_behavior_ban_time_seconds))}">
                      </div>
                      <div class="waf-field">
                        <label for="service-bad-behavior-threshold">${escapeHtml(ctx.t("sites.easy.traffic.threshold"))}</label>
                        <input id="service-bad-behavior-threshold" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_threshold))}">
                      </div>
                      <div class="waf-field">
                        <label for="service-bad-behavior-count-time">${escapeHtml(ctx.t("sites.easy.traffic.periodSeconds"))}</label>
                        <input id="service-bad-behavior-count-time" type="number" min="1" value="${escapeHtml(String(draft.bad_behavior_count_time_seconds))}">
                      </div>
                    </div>
                    </div>
                    <div class="waf-subcard waf-stack waf-antiddos-frame">
                      <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.limits"))}</div>
                      <div class="waf-form-grid">
                        <label class="waf-checkbox waf-field full">
                          <input id="service-use-limit-conn" type="checkbox"${draft.use_limit_conn ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitConnections"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http1">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp1Connections"))}</label>
                          <input id="service-limit-conn-max-http1" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http1))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http2">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp2Streams"))}</label>
                          <input id="service-limit-conn-max-http2" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http2))}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-conn-max-http3">${escapeHtml(ctx.t("sites.easy.traffic.maxHttp3Streams"))}</label>
                          <input id="service-limit-conn-max-http3" type="number" min="1" value="${escapeHtml(String(draft.limit_conn_max_http3))}">
                        </div>
                        <label class="waf-checkbox waf-field full">
                          <input id="service-use-limit-req" type="checkbox"${draft.use_limit_req ? " checked" : ""}>
                          <span>${escapeHtml(ctx.t("sites.easy.traffic.activateLimitRequests"))}</span>
                        </label>
                        <div class="waf-field">
                          <label for="service-limit-req-url">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestUrl"))}</label>
                          <input id="service-limit-req-url" value="${escapeHtml(draft.limit_req_url)}">
                        </div>
                        <div class="waf-field">
                          <label for="service-limit-req-rate">${escapeHtml(ctx.t("sites.easy.traffic.limitRequestRate"))}</label>
                          <input id="service-limit-req-rate" value="${escapeHtml(draft.limit_req_rate)}">
                        </div>
                        ${renderCustomLimitRulesEditor(draft.custom_limit_rules, ctx)}
                      </div>
                    </div>
                  </div>
                  <div class="waf-subcard waf-stack waf-antiddos-frame">
                    <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.traffic.frame.dnsbl"))}</div>
                    <div class="waf-note">${escapeHtml(ctx.t("sites.lists.note"))}</div>
                    <div class="waf-form-grid">
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-blacklist" type="checkbox"${draft.use_blacklist ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateBlacklisting"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-dnsbl" type="checkbox"${draft.use_dnsbl ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateDnsbl"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-allowlist" type="checkbox"${draft.use_allowlist ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateAllowlist"))}</span>
                      </label>
                      <label class="waf-checkbox waf-field full">
                        <input id="service-use-exceptions" type="checkbox"${draft.use_exceptions ? " checked" : ""}>
                        <span>${escapeHtml(ctx.t("sites.easy.traffic.activateExceptions"))}</span>
                      </label>
                      ${renderListEditor("access_denylist", ctx.t("sites.lists.denylist"), draft.access_denylist, "203.0.113.10", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("access_allowlist", ctx.t("sites.lists.allowlist"), draft.access_allowlist, "10.0.0.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("exceptions_ip", ctx.t("sites.easy.traffic.exceptions"), draft.exceptions_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${draft.use_allowlist || normalizeStringArray(draft.access_allowlist).length
                        ? ""
                        : `<div class="waf-note waf-field full">${escapeHtml(ctx.t("sites.easy.traffic.allowlistDisabledHint"))}</div>`}
                      ${renderListEditor("blacklist_ip", ctx.t("sites.easy.traffic.blacklistIp"), draft.blacklist_ip, "203.0.113.0/24", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_rdns", ctx.t("sites.easy.traffic.blacklistRdns"), draft.blacklist_rdns, ".shodan.io", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_asn", ctx.t("sites.easy.traffic.blacklistAsn"), draft.blacklist_asn, "AS13335", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_user_agent", ctx.t("sites.easy.traffic.blacklistUserAgent"), draft.blacklist_user_agent, "curl/*", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), presets: getQuickListTemplates("blacklist_user_agent"), selectedPreset: state.listTemplateSelection.blacklist_user_agent })}
                      ${renderListEditor("blacklist_uri", ctx.t("sites.easy.traffic.blacklistUri"), draft.blacklist_uri, "/admin", { full: false, emptyLabel: ctx.t("sites.easy.noValues"), presets: getQuickListTemplates("blacklist_uri"), selectedPreset: state.listTemplateSelection.blacklist_uri })}
                      ${renderListEditor("blacklist_ip_urls", ctx.t("sites.easy.traffic.blacklistIpUrls"), draft.blacklist_ip_urls, "https://example.com/ip.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_rdns_urls", ctx.t("sites.easy.traffic.blacklistRdnsUrls"), draft.blacklist_rdns_urls, "https://example.com/rdns.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_asn_urls", ctx.t("sites.easy.traffic.blacklistAsnUrls"), draft.blacklist_asn_urls, "https://example.com/asn.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_user_agent_urls", ctx.t("sites.easy.traffic.blacklistUserAgentUrls"), draft.blacklist_user_agent_urls, "https://example.com/ua.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                      ${renderListEditor("blacklist_uri_urls", ctx.t("sites.easy.traffic.blacklistUriUrls"), draft.blacklist_uri_urls, "https://example.com/uri.txt", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                    </div>
                  </div>
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "blocking" ? "" : " waf-hidden"}" data-tab-panel="blocking">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.blocking.title"))}</div>
                <div class="waf-note">${escapeHtml(ctx.t("sites.easy.blocking.baseHint"))}</div>
                <div class="waf-form-grid">
                  <label class="waf-checkbox waf-field full">
                    <input id="service-ban-escalation-enabled" type="checkbox"${draft.ban_escalation_enabled ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.blocking.enabled"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-ban-escalation-scope">${escapeHtml(ctx.t("sites.easy.blocking.scope"))}</label>
                    <select id="service-ban-escalation-scope">
                      <option value="all_sites"${draft.ban_escalation_scope === "all_sites" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.blocking.scope.allSites"))}</option>
                      <option value="current_site"${draft.ban_escalation_scope === "current_site" ? " selected" : ""}>${escapeHtml(ctx.t("sites.easy.blocking.scope.currentSite"))}</option>
                    </select>
                  </div>
                  <div class="waf-field full">
                    <label for="service-ban-stage-input">${escapeHtml(ctx.t("sites.easy.blocking.stageInput"))}</label>
                    <div class="waf-inline">
                      <input id="service-ban-stage-input" placeholder="${escapeHtml(ctx.t("sites.easy.blocking.stagePlaceholder"))}">
                      <button class="btn ghost btn-sm" type="button" data-ban-stage-add>${escapeHtml(ctx.t("sites.easy.blocking.addStage"))}</button>
                    </div>
                    <div class="waf-note">${escapeHtml(ctx.t("sites.easy.blocking.help"))}</div>
                    <div class="waf-inline">
                      ${normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds).map((seconds, index) => `
                        <span class="badge badge-neutral">
                          ${escapeHtml(`${ctx.t("sites.easy.blocking.stage")} ${index + 1}: ${seconds === 0 ? ctx.t("sites.easy.blocking.permanent") : formatBanDurationSeconds(seconds)}`)}
                          <button
                            class="waf-list-remove"
                            type="button"
                            data-ban-stage-remove="${index}">x</button>
                        </span>
                      `).join("")}
                    </div>
                  </div>
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "antibot" ? "" : " waf-hidden"}" data-tab-panel="antibot">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.antibot.title"))}</div>
                <div class="waf-form-grid">
                  <div class="waf-field">
                    <label for="service-antibot-challenge">${escapeHtml(ctx.t("sites.easy.antibot.challenge"))}</label>
                    <select id="service-antibot-challenge">
                      ${["no", "cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].map((mode) => `<option value="${mode}"${draft.antibot_challenge === mode ? " selected" : ""}>${mode}</option>`).join("")}
                    </select>
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-uri">${escapeHtml(ctx.t("sites.easy.antibot.url"))}</label>
                    <input id="service-antibot-uri" value="${escapeHtml(draft.antibot_uri)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-recaptcha-score">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaScore"))}</label>
                    <input id="service-antibot-recaptcha-score" type="number" step="0.1" min="0" max="1" value="${escapeHtml(String(draft.antibot_recaptcha_score))}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-recaptcha-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaSitekey"))}</label>
                    <input id="service-antibot-recaptcha-sitekey" value="${escapeHtml(draft.antibot_recaptcha_sitekey)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-recaptcha-secret">${escapeHtml(ctx.t("sites.easy.antibot.recaptchaSecret"))}</label>
                    <input id="service-antibot-recaptcha-secret" type="password" value="${escapeHtml(draft.antibot_recaptcha_secret)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-hcaptcha-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.hcaptchaSitekey"))}</label>
                    <input id="service-antibot-hcaptcha-sitekey" value="${escapeHtml(draft.antibot_hcaptcha_sitekey)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-hcaptcha-secret">${escapeHtml(ctx.t("sites.easy.antibot.hcaptchaSecret"))}</label>
                    <input id="service-antibot-hcaptcha-secret" type="password" value="${escapeHtml(draft.antibot_hcaptcha_secret)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-turnstile-sitekey">${escapeHtml(ctx.t("sites.easy.antibot.turnstileSitekey"))}</label>
                    <input id="service-antibot-turnstile-sitekey" value="${escapeHtml(draft.antibot_turnstile_sitekey)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-antibot-turnstile-secret">${escapeHtml(ctx.t("sites.easy.antibot.turnstileSecret"))}</label>
                    <input id="service-antibot-turnstile-secret" type="password" value="${escapeHtml(draft.antibot_turnstile_secret)}">
                  </div>
                  <label class="waf-checkbox">
                    <input id="service-use-auth-basic" type="checkbox"${draft.use_auth_basic ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.antibot.useAuthBasic"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-auth-basic-location">${escapeHtml(ctx.t("sites.easy.antibot.authBasicLocation"))}</label>
                    <input id="service-auth-basic-location" value="${escapeHtml("sitewide")}" readonly>
                  </div>
                  <div class="waf-field">
                    <label for="service-auth-basic-user">${escapeHtml(ctx.t("sites.easy.antibot.authBasicUsername"))}</label>
                    <input id="service-auth-basic-user" value="${escapeHtml(draft.auth_basic_user)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-auth-basic-password">${escapeHtml(ctx.t("sites.easy.antibot.authBasicPassword"))}</label>
                    <input id="service-auth-basic-password" type="password" value="${escapeHtml(draft.auth_basic_password)}">
                  </div>
                  <div class="waf-field">
                    <label for="service-auth-basic-text">${escapeHtml(ctx.t("sites.easy.antibot.authText"))}</label>
                    <input id="service-auth-basic-text" value="${escapeHtml(draft.auth_basic_text)}">
                  </div>
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "geo" ? "" : " waf-hidden"}" data-tab-panel="geo">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.geo.title"))}</div>
                <div class="waf-form-grid">
                  ${renderCountryEditor("blacklist_country", ctx.t("sites.easy.geo.countryBlacklist"), draft.blacklist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.blacklist_country })}
                  ${renderCountryEditor("whitelist_country", ctx.t("sites.easy.geo.countryWhitelist"), draft.whitelist_country, state.geoCatalog, { full: false, emptyLabel: ctx.t("sites.easy.noValues"), search: state.countryFilters.whitelist_country })}
                </div>
              </section>

              <section class="waf-subcard waf-stack waf-service-compact-section${state.activeTab === "modsec" ? "" : " waf-hidden"}" data-tab-panel="modsec">
                <div class="waf-list-title">${escapeHtml(ctx.t("sites.easy.tab.modsec.title"))}</div>
                <div class="waf-form-grid">
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity" type="checkbox"${draft.use_modsecurity ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useModsecurity"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity-crs-plugins" type="checkbox"${draft.use_modsecurity_crs_plugins ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useCrsPlugins"))}</span>
                  </label>
                  <label class="waf-checkbox">
                    <input id="service-use-modsecurity-custom-configuration" type="checkbox"${draft.use_modsecurity_custom_configuration ? " checked" : ""}>
                    <span>${escapeHtml(ctx.t("sites.easy.modsec.useCustomConfiguration"))}</span>
                  </label>
                  <div class="waf-field">
                    <label for="service-modsecurity-crs-version">${escapeHtml(ctx.t("sites.easy.modsec.crsVersion"))}</label>
                    <input id="service-modsecurity-crs-version" value="${escapeHtml(draft.modsecurity_crs_version)}">
                  </div>
                  ${renderListEditor("modsecurity_crs_plugins", ctx.t("sites.easy.modsec.crsPlugins"), draft.modsecurity_crs_plugins, "plugin-id", { full: false, emptyLabel: ctx.t("sites.easy.noValues") })}
                  <div class="waf-field${draft.use_modsecurity_custom_configuration ? "" : " waf-hidden"}">
                    <label for="service-modsecurity-custom-path">${escapeHtml(ctx.t("sites.easy.modsec.customPath"))}</label>
                    <input id="service-modsecurity-custom-path" value="${escapeHtml(draft.modsecurity_custom_path)}">
                  </div>
                  <div class="waf-field full${draft.use_modsecurity_custom_configuration ? "" : " waf-hidden"}">
                    <label for="service-modsecurity-custom-content">${escapeHtml(ctx.t("sites.easy.modsec.customContent"))}</label>
                    <textarea id="service-modsecurity-custom-content" rows="6">${escapeHtml(draft.modsecurity_custom_content)}</textarea>
                  </div>
                </div>
              </section>

              <div class="waf-actions waf-actions-between">
                <button class="btn ghost btn-sm" type="button" id="service-back-bottom">${escapeHtml(ctx.t("common.back"))}</button>
                <button class="btn primary btn-sm" type="submit">${escapeHtml(ctx.t(isNew ? "sites.action.createSite" : "sites.action.saveSite"))}</button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>
  `;
}

async function upsertAccessPolicy(draft, ctx, existingAccessPolicy) {
  const siteID = normalizeSiteID(draft.id);
  if (!siteID) {
    return;
  }
  const allowlistSources = [];
  if (draft.use_allowlist) {
    allowlistSources.push(...normalizeStringArray(draft.access_allowlist));
  }
  if (draft.use_exceptions) {
    allowlistSources.push(...normalizeStringArray(draft.exceptions_ip));
  }
  const allowlist = Array.from(new Set(allowlistSources));
  const denylist = normalizeStringArray(draft.access_denylist);
  if (!allowlist.length && !denylist.length && !existingAccessPolicy) {
    return;
  }
  const payload = {
    id: String(existingAccessPolicy?.id || `${siteID}-access`),
    site_id: siteID,
    enabled: true,
    allowlist,
    denylist
  };
  const resolvePolicyForSite = async () => {
    const accessPolicies = normalizeArray(await ctx.api.get("/api/access-policies"));
    return accessPolicies.find((item) => normalizeSiteID(item?.site_id) === siteID) || null;
  };
  const normalizeListForCompare = (values) => normalizeStringArray(values).slice().sort();
  const matchesPayload = (policy) => {
    if (!policy) {
      return false;
    }
    if (normalizeSiteID(policy.site_id) !== siteID) {
      return false;
    }
    const policyAllow = normalizeListForCompare(policy.allowlist);
    const policyDeny = normalizeListForCompare(policy.denylist);
    const expectedAllow = normalizeListForCompare(payload.allowlist);
    const expectedDeny = normalizeListForCompare(payload.denylist);
    return JSON.stringify(policyAllow) === JSON.stringify(expectedAllow) &&
      JSON.stringify(policyDeny) === JSON.stringify(expectedDeny);
  };
  if (!allowlist.length && !denylist.length && existingAccessPolicy) {
    try {
      await ctx.api.delete(`/api/access-policies/${encodeURIComponent(payload.id)}`);
    } catch (error) {
      if (error?.status !== 404 && !isAutoApplyFailureError(error)) {
        throw error;
      }
      const policyForSite = await resolvePolicyForSite();
      if (policyForSite) {
        throw error;
      }
    }
    return;
  }
  try {
    await ctx.api.post("/api/access-policies/upsert", payload);
  } catch (error) {
    if (isAutoApplyFailureError(error)) {
      const policyForSite = await resolvePolicyForSite();
      if (matchesPayload(policyForSite)) {
        return;
      }
    }
    if (error?.status === 404 || error?.status === 405) {
      // Backward compatibility with older backend versions without upsert endpoint.
      if (existingAccessPolicy) {
        try {
          await ctx.api.put(`/api/access-policies/${encodeURIComponent(payload.id)}`, payload);
          return;
        } catch (putError) {
          if (isAutoApplyFailureError(putError)) {
            const policyForSite = await resolvePolicyForSite();
            if (matchesPayload(policyForSite)) {
              return;
            }
          }
          if (putError?.status !== 404) {
            throw putError;
          }
        }
      }
      try {
        await ctx.api.post("/api/access-policies", payload);
      } catch (postError) {
        if (isAutoApplyFailureError(postError)) {
          const policyForSite = await resolvePolicyForSite();
          if (matchesPayload(policyForSite)) {
            return;
          }
        }
        throw postError;
      }
      return;
    }
    const message = String(error?.message || "").toLowerCase();
    if (error?.status === 400 && message.includes("already exists")) {
      const policyForSite = await resolvePolicyForSite();
      if (policyForSite?.id) {
        const upsertPayload = { ...payload, id: String(policyForSite.id) };
        try {
          await ctx.api.put(`/api/access-policies/${encodeURIComponent(String(policyForSite.id))}`, upsertPayload);
        } catch (putError) {
          if (isAutoApplyFailureError(putError)) {
            const persistedPolicy = await resolvePolicyForSite();
            if (matchesPayload(persistedPolicy)) {
              return;
            }
          }
          throw putError;
        }
        return;
      }
    }
    throw error;
  }
}

async function resolveACMEAccountEmail(draft, ctx) {
  const fromDraft = normalizeEmail(draft?.acme_account_email);
  if (isValidEmail(fromDraft)) {
    return fromDraft;
  }

  const siteID = String(draft?.id || "").trim().toLowerCase();
  if (siteID) {
    try {
      const ownProfile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(siteID)}`);
      const ownEmail = normalizeEmail(ownProfile?.front_service?.acme_account_email);
      if (isValidEmail(ownEmail)) {
        return ownEmail;
      }
    } catch (error) {
      if (error?.status !== 404) {
        console.warn("failed to read own easy profile for acme email", error);
      }
    }
  }

  try {
    const sites = await ctx.api.get("/api/sites");
    for (const site of normalizeArray(sites)) {
      const candidateSiteID = String(site?.id || "").trim().toLowerCase();
      if (!candidateSiteID || candidateSiteID === siteID) {
        continue;
      }
      try {
        const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(candidateSiteID)}`);
        const email = normalizeEmail(profile?.front_service?.acme_account_email);
        if (isValidEmail(email)) {
          return email;
        }
      } catch (error) {
        if (error?.status !== 404) {
          console.warn("failed to read easy profile for acme email", error);
        }
      }
    }
  } catch (error) {
    console.warn("failed to list sites for acme email fallback", error);
  }

  try {
    const me = await ctx.api.get("/api/auth/me");
    const sessionEmail = normalizeEmail(me?.email || me?.user?.email);
    if (isValidEmail(sessionEmail)) {
      return sessionEmail;
    }
  } catch (error) {
    if (error?.status !== 401 && error?.status !== 403 && error?.status !== 404) {
      console.warn("failed to resolve auth/me email fallback", error);
    }
  }

  return "";
}

function isAutoApplyFailureError(error) {
  const message = String(error?.message || "").toLowerCase();
  return message.includes("apply failed") ||
    message.includes("reload failed") ||
    message.includes("health-check failed") ||
    message.includes("default upstream is required") ||
    message.includes("revision apply") ||
    message.includes("limit_req");
}

function isAlreadyExistsError(error) {
  const message = String(error?.message || "").toLowerCase();
  return (error?.status === 400 || error?.status === 409) && message.includes("already exists");
}

async function upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig) {
  const sitePayload = {
    id: draft.id.trim().toLowerCase(),
    primary_host: draft.primary_host.trim().toLowerCase(),
    enabled: draft.enabled
  };
  const upstreamPayload = {
    id: draft.upstream_id.trim().toLowerCase(),
    site_id: sitePayload.id,
    host: draft.upstream_host.trim(),
    port: draft.upstream_port,
    scheme: draft.upstream_scheme
  };
  const cleanupActions = [];
  const runCleanup = async () => {
    for (let index = cleanupActions.length - 1; index >= 0; index -= 1) {
      try {
        await cleanupActions[index]();
      } catch (_error) {
        // best-effort cleanup; keep original save error
      }
    }
  };
  const rollbackable = (fn) => cleanupActions.push(fn);
  const shouldKeepStateOnApplyError = (error) => {
    // Auto-apply failures may return after resources were persisted in control-plane state.
    // Do not rollback saved entities in this case, otherwise TLS bindings can be silently removed.
    return isAutoApplyFailureError(error);
  };
  const isNotFound = (error) => error?.status === 404;
  const deleteIgnoreNotFound = async (path) => {
    try {
      await ctx.api.delete(path);
    } catch (error) {
      if (!isNotFound(error)) {
        throw error;
      }
    }
  };
  const siteExists = async (siteID) => {
    const sites = await ctx.api.get("/api/sites");
    return Array.isArray(sites) && sites.some((site) => normalizeSiteID(site?.id) === normalizeSiteID(siteID));
  };
  const upstreamExists = async (upstreamID) => {
    const upstreams = await ctx.api.get("/api/upstreams");
    return Array.isArray(upstreams) && upstreams.some((upstream) => normalizeSiteID(upstream?.id) === normalizeSiteID(upstreamID));
  };
  const tlsConfigMatches = async (siteID, certificateID) => {
    const tlsConfigs = await ctx.api.get("/api/tls-configs");
    if (!Array.isArray(tlsConfigs)) {
      return false;
    }
    return tlsConfigs.some((item) =>
      normalizeSiteID(item?.site_id) === normalizeSiteID(siteID) &&
      String(item?.certificate_id || "").trim().toLowerCase() === String(certificateID || "").trim().toLowerCase());
  };
  const certificateExists = async (certificateID) => {
    const certificates = await ctx.api.get("/api/certificates");
    return Array.isArray(certificates) && certificates.some((certificate) => String(certificate?.id || "").toLowerCase() === String(certificateID || "").toLowerCase());
  };

  try {
    if (existingSite) {
      try {
        await ctx.api.put(`/api/sites/${encodeURIComponent(sitePayload.id)}`, sitePayload);
      } catch (error) {
        if (error?.status === 404) {
          try {
            await ctx.api.post("/api/sites", sitePayload);
          } catch (postError) {
            if (!isAutoApplyFailureError(postError)) {
              throw postError;
            }
            const persisted = await siteExists(sitePayload.id);
            if (!persisted) {
              throw postError;
            }
          }
        } else if (isAutoApplyFailureError(error)) {
          const persisted = await siteExists(sitePayload.id);
          if (!persisted) {
            throw error;
          }
        } else {
          throw error;
        }
      }
    } else {
      let createdSite = false;
      try {
        await ctx.api.post("/api/sites", sitePayload);
        createdSite = true;
      } catch (error) {
        if (isAlreadyExistsError(error)) {
          await ctx.api.put(`/api/sites/${encodeURIComponent(sitePayload.id)}`, sitePayload);
          createdSite = false;
        } else if (!isAutoApplyFailureError(error)) {
          throw error;
        } else {
          createdSite = await siteExists(sitePayload.id);
          if (!createdSite) {
            throw error;
          }
        }
      }
      if (createdSite) {
        rollbackable(async () => deleteIgnoreNotFound(`/api/sites/${encodeURIComponent(sitePayload.id)}`));
      }
    }

    if (existingUpstream) {
      try {
        await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`, upstreamPayload);
      } catch (error) {
        if (error?.status === 404) {
          try {
            await ctx.api.post("/api/upstreams", upstreamPayload);
          } catch (postError) {
            if (!isAutoApplyFailureError(postError)) {
              throw postError;
            }
            const persisted = await upstreamExists(upstreamPayload.id);
            if (!persisted) {
              throw postError;
            }
          }
        } else if (isAutoApplyFailureError(error)) {
          const persisted = await upstreamExists(upstreamPayload.id);
          if (!persisted) {
            throw error;
          }
        } else {
          throw error;
        }
      }
    } else {
      let createdUpstream = false;
      try {
        await ctx.api.post("/api/upstreams", upstreamPayload);
        createdUpstream = true;
      } catch (error) {
        if (isAlreadyExistsError(error)) {
          await ctx.api.put(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`, upstreamPayload);
          createdUpstream = false;
        } else if (isAutoApplyFailureError(error)) {
          createdUpstream = await upstreamExists(upstreamPayload.id);
          if (!createdUpstream) {
            throw error;
          }
        } else {
          throw error;
        }
      }
      if (createdUpstream) {
        rollbackable(async () => deleteIgnoreNotFound(`/api/upstreams/${encodeURIComponent(upstreamPayload.id)}`));
      }
    }

    if (draft.tls_enabled) {
      const certificateID = (draft.certificate_id.trim() || `${sitePayload.id}-tls`).toLowerCase();
      const hadCertificateBefore = await certificateExists(certificateID);
      const importMode = String(draft.certificate_authority_server || "").trim().toLowerCase() === "import";
      const isCurrentTLSCertificate = normalizeSiteID(existingTLSConfig?.site_id) === normalizeSiteID(sitePayload.id) &&
        String(existingTLSConfig?.certificate_id || "").trim().toLowerCase() === certificateID;
      if (importMode && !hadCertificateBefore) {
        throw new Error(ctx.t("sites.tls.importRequired"));
      }
      if (!importMode && !hadCertificateBefore && !isCurrentTLSCertificate) {
        const issueEndpoint = draft.tls_self_signed ? "/api/certificates/self-signed/issue" : "/api/certificates/acme/issue";
        const acmeAccountEmail = draft.tls_self_signed ? "" : await resolveACMEAccountEmail(draft, ctx);
        if (!draft.tls_self_signed && acmeAccountEmail) {
          draft.acme_account_email = acmeAccountEmail;
        }
        try {
          await ctx.api.post(issueEndpoint, {
            certificate_id: certificateID,
            common_name: sitePayload.primary_host,
            san_list: [],
            certificate_authority_server: draft.certificate_authority_server,
            use_lets_encrypt_staging: Boolean(draft.use_lets_encrypt_staging),
            account_email: acmeAccountEmail
          });
        } catch (error) {
          if (!isAlreadyExistsError(error)) {
            throw error;
          }
        }
      }
      if (!hadCertificateBefore && !importMode) {
        rollbackable(async () => deleteIgnoreNotFound(`/api/certificates/${encodeURIComponent(certificateID)}`));
      }
      const tlsPayload = { site_id: sitePayload.id, certificate_id: certificateID };
      if (existingTLSConfig) {
        try {
          await ctx.api.put(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`, tlsPayload);
        } catch (error) {
          if (error?.status === 404) {
            try {
              await ctx.api.post("/api/tls-configs", tlsPayload);
            } catch (postError) {
              if (!isAutoApplyFailureError(postError)) {
                throw postError;
              }
              const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
              if (!persisted) {
                throw postError;
              }
            }
          } else if (isAutoApplyFailureError(error)) {
            const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
            if (!persisted) {
              throw error;
            }
          } else {
            throw error;
          }
        }
      } else {
        try {
          await ctx.api.post("/api/tls-configs", tlsPayload);
        } catch (error) {
          const message = String(error?.message || "").toLowerCase();
          const hasConflict = error?.status === 409 || message.includes("already exists");
          if (hasConflict) {
            await ctx.api.put(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`, tlsPayload);
          } else if (isAutoApplyFailureError(error)) {
            const persisted = await tlsConfigMatches(sitePayload.id, certificateID);
            if (!persisted) {
              throw error;
            }
          } else {
            throw error;
          }
        }
        rollbackable(async () => deleteIgnoreNotFound(`/api/tls-configs/${encodeURIComponent(sitePayload.id)}`));
      }
    }
  } catch (error) {
    if (!shouldKeepStateOnApplyError(error)) {
      await runCleanup();
    }
    throw error;
  }

}

async function deleteServiceWithResources(siteID, ctx, snapshot = null) {
  const normalizedSiteID = normalizeSiteID(siteID);
  if (!normalizedSiteID) {
    return;
  }
  const normalizeIDValue = (value) => String(value || "").trim().toLowerCase();
  const isNotFound = (error) => error?.status === 404;
  const includesByID = (items, id) => normalizeArray(items).some((item) => normalizeIDValue(item?.id) === normalizeIDValue(id));
  const includesBySiteID = (items, id) => normalizeArray(items).some((item) => normalizeIDValue(item?.site_id) === normalizeIDValue(id));
  const deleteIgnoreSafe = async (path, verifyDeleted = null) => {
    try {
      await ctx.api.delete(path);
    } catch (error) {
      if (isNotFound(error)) {
        return;
      }
      if (isAutoApplyFailureError(error) && typeof verifyDeleted === "function") {
        const deleted = await verifyDeleted();
        if (deleted) {
          return;
        }
      }
      if (!isAutoApplyFailureError(error)) {
        throw error;
      }
    }
  };
  const sites = Array.isArray(snapshot?.sites) ? snapshot.sites : await ctx.api.get("/api/sites").catch(() => []);
  const upstreams = Array.isArray(snapshot?.upstreams)
    ? snapshot.upstreams
    : await ctx.api.get("/api/upstreams").catch(() => []);
  const tlsConfigs = Array.isArray(snapshot?.tlsConfigs)
    ? snapshot.tlsConfigs
    : await ctx.api.get("/api/tls-configs").catch(() => []);
  const easyProfiles = Array.isArray(snapshot?.easyProfiles)
    ? snapshot.easyProfiles
    : await ctx.api.get("/api/easy-site-profiles").catch(() => []);
  const upstreamsForSite = normalizeArray(upstreams).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID);
  const hasSite = includesByID(sites, normalizedSiteID);
  const hasTLSConfig = includesBySiteID(tlsConfigs, normalizedSiteID);
  const hasEasyProfile = includesBySiteID(easyProfiles, normalizedSiteID);
  if (hasTLSConfig) {
    await deleteIgnoreSafe(`/api/tls-configs/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/tls-configs").catch(() => []);
      return !includesBySiteID(latest, normalizedSiteID);
    });
  }
  const easyAccessPolicyID = `easy-${normalizedSiteID}-access`;
  const hasEasyAccessPolicy = includesByID(snapshot?.accessPolicies, easyAccessPolicyID);
  if (hasEasyAccessPolicy) {
    await deleteIgnoreSafe(`/api/access-policies/${encodeURIComponent(easyAccessPolicyID)}`, async () => {
      const latest = await ctx.api.get("/api/access-policies").catch(() => []);
      return !includesByID(latest, easyAccessPolicyID);
    });
  }
  if (hasEasyProfile) {
    await deleteIgnoreSafe(`/api/easy-site-profiles/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/easy-site-profiles").catch(() => []);
      return !includesBySiteID(latest, normalizedSiteID);
    });
  }
  const [wafPolicies, ratePolicies, accessPolicies] = await Promise.all([
    Array.isArray(snapshot?.wafPolicies) ? snapshot.wafPolicies : ctx.api.get("/api/waf-policies").catch(() => []),
    Array.isArray(snapshot?.ratePolicies) ? snapshot.ratePolicies : ctx.api.get("/api/rate-limit-policies").catch(() => []),
    Array.isArray(snapshot?.accessPolicies) ? snapshot.accessPolicies : ctx.api.get("/api/access-policies").catch(() => [])
  ]);
  for (const policy of normalizeArray(wafPolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/waf-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/waf-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of normalizeArray(ratePolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/rate-limit-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/rate-limit-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  for (const policy of normalizeArray(accessPolicies).filter((item) => normalizeSiteID(item?.site_id) === normalizedSiteID)) {
    const policyID = String(policy?.id || "").trim();
    if (!policyID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/access-policies/${encodeURIComponent(policyID)}`, async () => {
      const latest = await ctx.api.get("/api/access-policies").catch(() => []);
      return !includesByID(latest, policyID);
    });
  }
  if (hasSite) {
    await deleteIgnoreSafe(`/api/sites/${encodeURIComponent(normalizedSiteID)}`, async () => {
      const latest = await ctx.api.get("/api/sites").catch(() => []);
      return !includesByID(latest, normalizedSiteID);
    });
  }
  for (const upstream of upstreamsForSite) {
    const upstreamID = String(upstream?.id || "").trim();
    if (!upstreamID) {
      continue;
    }
    await deleteIgnoreSafe(`/api/upstreams/${encodeURIComponent(upstreamID)}`, async () => {
      const latest = await ctx.api.get("/api/upstreams").catch(() => []);
      return !includesByID(latest, upstreamID);
    });
  }
}

async function putWithPostFallback(ctx, path, payload, options = {}) {
  const tolerateAutoApplyError = Boolean(options?.tolerateAutoApplyError);
  const verifyPersisted = typeof options?.verifyPersisted === "function" ? options.verifyPersisted : null;
  try {
    await ctx.api.post(path, payload);
    return;
  } catch (postError) {
    if (tolerateAutoApplyError && isAutoApplyFailureError(postError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) {
        return;
      }
    }
    if (postError?.status !== 404 && postError?.status !== 405) {
      throw postError;
    }
  }
  try {
    await ctx.api.put(path, payload);
  } catch (putError) {
    if (tolerateAutoApplyError && isAutoApplyFailureError(putError) && verifyPersisted) {
      const persisted = await verifyPersisted();
      if (persisted) {
        return;
      }
    }
    throw putError;
  }
}

function ensureControlPlaneAccessManagementMethods(draft) {
  const siteID = String(draft?.id || "").trim().toLowerCase();
  if (siteID !== "control-plane-access") {
    return draft;
  }
  const required = ["GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"];
  const methods = normalizeStringArray(draft.allowed_methods).map((item) => item.toUpperCase());
  const merged = [...methods];
  for (const method of required) {
    if (!merged.includes(method)) {
      merged.push(method);
    }
  }
  return { ...draft, allowed_methods: merged };
}

function shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig) {
  if (!existingSite) {
    return true;
  }
  if (String(existingSite?._origin || "") === "secondary") {
    return true;
  }
  if (existingUpstream && String(existingUpstream?._origin || "") === "secondary") {
    return true;
  }
  if (existingTLSConfig && String(existingTLSConfig?._origin || "") === "secondary") {
    return true;
  }
  const siteID = draft.id.trim().toLowerCase();
  const siteHost = draft.primary_host.trim().toLowerCase();
  if (String(existingSite.id || "").toLowerCase() !== siteID) {
    return true;
  }
  if (String(existingSite.primary_host || "").toLowerCase() !== siteHost) {
    return true;
  }
  if (Boolean(existingSite.enabled) !== Boolean(draft.enabled)) {
    return true;
  }

  const upstreamID = draft.upstream_id.trim().toLowerCase();
  if (!existingUpstream || String(existingUpstream.id || "").toLowerCase() !== upstreamID) {
    return true;
  }
  if (String(existingUpstream.site_id || "").toLowerCase() !== siteID) {
    return true;
  }
  if (String(existingUpstream.host || "") !== String(draft.upstream_host || "").trim()) {
    return true;
  }
  if (Number(existingUpstream.port || 0) !== Number(draft.upstream_port || 0)) {
    return true;
  }
  if (String(existingUpstream.scheme || "").toLowerCase() !== String(draft.upstream_scheme || "").toLowerCase()) {
    return true;
  }

  if (draft.tls_enabled) {
    if (!existingTLSConfig) {
      return true;
    }
    const certificateID = (draft.certificate_id.trim() || `${siteID}-tls`).toLowerCase();
    if (String(existingTLSConfig.site_id || "").toLowerCase() !== siteID) {
      return true;
    }
    if (String(existingTLSConfig.certificate_id || "").toLowerCase() !== certificateID) {
      return true;
    }
  }
  return false;
}

async function exportSelectedServicesEnv(ctx, sites, upstreamsBySite, tlsBySite, selectedSiteIDs) {
  const targets = sites.filter((site) => selectedSiteIDs.has(site.id));
  for (const site of targets) {
    const upstream = upstreamsBySite.get(site.id)?.[0] || null;
    const tlsConfig = tlsBySite.get(site.id) || null;
    let draft = siteDraftFromData(site, upstream, tlsConfig);
    try {
      const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`);
      draft = applyEasyProfileToDraft(draft, profile);
    } catch (error) {
      const secondaryDump = await tryGetJSON("/api-app/easy-site-profiles");
      const secondaryProfile = findEasyProfile(secondaryDump, site.id);
      if (secondaryProfile) {
        draft = applyEasyProfileToDraft(draft, secondaryProfile);
      }
    }
    downloadText(`${site.id}.env`, draftToEnvText(draft));
  }
  return targets.length;
}

async function importServicesFiles(files, ctx) {
  const results = [];
  for (const file of files) {
    const name = String(file?.name || "").toLowerCase();
    if (name.endsWith(".json")) {
      const jsonResult = await importServicesJSON(file, ctx);
      results.push({ file: file.name, kind: "json", ...jsonResult });
      continue;
    }
    if (name.endsWith(".env")) {
      requirePermissions(ctx, ["certificates.write", "tls.write"], "sites.error.importEnvPermissions");
      const text = await file.text();
      const { draft, missingFields } = envToDraft(text);
      const validationError = validateDraft(draft, ctx);
      if (validationError) {
        throw new Error(`${file.name}: ${validationError}`);
      }
      const payload = buildImportPayloadFromDraft(draft);
      const applied = await applyImportPayload(ctx, payload);
      results.push({
        file: file.name,
        kind: "env",
        siteID: payload.site.id,
        missingFields,
        ...applied
      });
      continue;
    }
    throw new Error(`${file.name}: unsupported extension`);
  }
  return results;
}

export async function renderSites(container, ctx) {
  const route = routeInfo();
  const state = {
    route,
    sites: [],
    upstreams: [],
    tlsConfigs: [],
    certificates: [],
    accessPolicies: [],
    geoCatalog: buildGeoCatalogFallback(),
    search: "",
    sort: "updated-desc",
    activeTab: "front",
    settingsSearch: "",
    settingsMatches: [],
    highlightedSelector: "",
    filteredSites: [],
    selectedSiteIDs: new Set(),
    upstreamsBySite: new Map(),
    tlsBySite: new Map(),
    certificateBySiteID: new Map(),
    certificateByHost: new Map(),
    accessBySite: new Map(),
    countryFilters: {
      blacklist_country: "",
      whitelist_country: ""
    },
    listTemplateSelection: {
      blacklist_user_agent: "",
      blacklist_uri: ""
    },
    draft: defaultSiteDraft()
  };

  const rebuildIndexes = () => {
    state.upstreamsBySite = new Map();
    state.tlsBySite = new Map();
    state.certificateBySiteID = new Map();
    state.certificateByHost = new Map();
    state.accessBySite = new Map();
    for (const upstream of state.upstreams) {
      const items = state.upstreamsBySite.get(upstream.site_id) || [];
      items.push(upstream);
      state.upstreamsBySite.set(upstream.site_id, items);
    }
    for (const tlsConfig of state.tlsConfigs) {
      state.tlsBySite.set(tlsConfig.site_id, tlsConfig);
    }
    for (const certificate of state.certificates) {
      const certificateID = String(certificate?.id || "").trim().toLowerCase();
      if (certificateID.endsWith("-tls")) {
        const relatedSiteID = certificateID.slice(0, -4);
        if (relatedSiteID && !state.certificateBySiteID.has(relatedSiteID)) {
          state.certificateBySiteID.set(relatedSiteID, certificate);
        }
      }
      const host = normalizeHost(certificate?.common_name);
      if (host && !state.certificateByHost.has(host)) {
        state.certificateByHost.set(host, certificate);
      }
    }
    for (const accessPolicy of state.accessPolicies) {
      const siteID = normalizeSiteID(accessPolicy?.site_id);
      if (!siteID || state.accessBySite.has(siteID)) {
        continue;
      }
      state.accessBySite.set(siteID, accessPolicy);
    }
  };

  const applyFilters = () => {
    const search = state.search.trim().toLowerCase();
    const sites = state.sites.filter((site) => {
      if (!search) {
        return true;
      }
      return `${site.id} ${site.primary_host}`.toLowerCase().includes(search);
    });
    sites.sort((left, right) => {
      if (state.sort === "name-asc") {
        return String(left.primary_host || left.id).localeCompare(String(right.primary_host || right.id));
      }
      if (state.sort === "name-desc") {
        return String(right.primary_host || right.id).localeCompare(String(left.primary_host || left.id));
      }
      if (state.sort === "created-desc") {
        return String(right.created_at || "").localeCompare(String(left.created_at || ""));
      }
      return String(right.updated_at || right.created_at || "").localeCompare(String(left.updated_at || left.created_at || ""));
    });
    state.filteredSites = sites;
  };

  const syncDraftFromRoute = async () => {
    if (state.route.mode === "list") {
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    if (state.route.mode === "create") {
      state.draft = defaultSiteDraft();
      state.activeTab = "front";
      state.settingsSearch = "";
      state.settingsMatches = [];
      state.highlightedSelector = "";
      return;
    }
    const site = state.sites.find((item) => item.id === state.route.siteID);
    const upstream = state.upstreamsBySite.get(state.route.siteID)?.[0] || null;
    const tlsConfig = state.tlsBySite.get(state.route.siteID) || null;
    const accessPolicy = state.accessBySite.get(normalizeSiteID(state.route.siteID)) || null;
    state.draft = site ? siteDraftFromData(site, upstream, tlsConfig) : defaultSiteDraft();
    if (site?.id) {
      try {
        const profile = await ctx.api.get(`/api/easy-site-profiles/${encodeURIComponent(site.id)}`);
        state.draft = applyEasyProfileToDraft(state.draft, profile);
      } catch (error) {
        const secondaryDump = await tryGetJSON("/api-app/easy-site-profiles");
        const secondaryProfile = findEasyProfile(secondaryDump, site.id);
        if (secondaryProfile) {
          state.draft = applyEasyProfileToDraft(state.draft, secondaryProfile);
        } else if (error?.status !== 404) {
          console.warn("failed to load easy-site-profile", error);
        }
      }
    }
    state.draft.access_allowlist = normalizeStringArray(accessPolicy?.allowlist);
    state.draft.access_denylist = normalizeStringArray(accessPolicy?.denylist);
    state.draft.use_allowlist = state.draft.access_allowlist.length > 0;
    if (!normalizeStringArray(state.draft.exceptions_ip).length) {
      state.draft.exceptions_ip = [...state.draft.access_allowlist];
    }
    if (!state.draft.use_exceptions) {
      state.draft.use_exceptions = normalizeStringArray(state.draft.exceptions_ip).length > 0;
    }
    state.activeTab = "front";
    state.settingsSearch = "";
    state.settingsMatches = [];
    state.highlightedSelector = "";
  };

  const render = () => {
    applyFilters();
    container.innerHTML = state.route.mode === "list" ? renderListView(state, ctx) : renderDetailView(state, ctx);
    bind();
  };

  const load = async () => {
    try {
      setLoading(container, ctx.t("sites.loading"));
      const [sitesResponse, upstreamsResponse, tlsConfigsResponse, certificatesResponse, accessPoliciesResponse, geoCatalogResponse] = await Promise.all([
        ctx.api.get("/api/sites"),
        ctx.api.get("/api/upstreams"),
        ctx.api.get("/api/tls-configs"),
        ctx.api.get("/api/certificates").catch(() => []),
        ctx.api.get("/api/access-policies").catch(() => []),
        ctx.api.get("/api/easy-site-profiles/catalog/countries").catch(() => null)
      ]);
      const [secondarySites, secondaryUpstreams, secondaryTLSConfigs, secondaryCertificates] = await Promise.all([
        tryGetJSON("/api-app/sites"),
        tryGetJSON("/api-app/upstreams"),
        tryGetJSON("/api-app/tls-configs"),
        tryGetJSON("/api-app/certificates")
      ]);
      state.sites = mergeByID(sitesResponse, unwrapList(secondarySites, ["sites"]));
      state.upstreams = mergeByID(upstreamsResponse, unwrapList(secondaryUpstreams, ["upstreams"]));
      state.tlsConfigs = mergeByID(tlsConfigsResponse, unwrapList(secondaryTLSConfigs, ["tls_configs", "tlsConfigs"]));
      state.certificates = mergeByID(certificatesResponse, unwrapList(secondaryCertificates, ["certificates"]));
      state.accessPolicies = normalizeArray(accessPoliciesResponse);
      state.selectedSiteIDs = new Set(Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => site.id === id)));
      state.geoCatalog = normalizeGeoCatalogPayload(geoCatalogResponse);
      rebuildIndexes();
      await syncDraftFromRoute();
      render();
    } catch (error) {
      container.innerHTML = `<div class="alert">${escapeHtml(ctx.t("sites.error.load"))}</div>`;
    }
  };

  const bindList = () => {
    const feedback = container.querySelector("#sites-feedback");
    container.querySelector("#services-create")?.addEventListener("click", () => go(`${routeBase()}/new`));
    container.querySelector("#services-refresh")?.addEventListener("click", load);
    container.querySelector("#services-export")?.addEventListener("click", async () => {
      feedback.innerHTML = "";
      if (!state.selectedSiteIDs.size) {
        downloadJSON("waf-services-export.json", { sites: state.sites, upstreams: state.upstreams, tls_configs: state.tlsConfigs });
        ctx.notify(ctx.t("sites.toast.exported"));
        return;
      }
      try {
        const exportedCount = await exportSelectedServicesEnv(ctx, state.sites, state.upstreamsBySite, state.tlsBySite, state.selectedSiteIDs);
        ctx.notify(ctx.t("sites.toast.exportedEnv", { count: exportedCount }));
      } catch (error) {
        setError(feedback, `${ctx.t("sites.error.exportEnv")}: ${String(error?.message || error)}`);
      }
    });
    container.querySelector("#services-import")?.addEventListener("click", () => {
      container.querySelector("#services-import-file")?.click();
    });
    container.querySelector("#services-import-file")?.addEventListener("change", async (event) => {
      const files = Array.from(event.target.files || []);
      if (!files.length) {
        return;
      }
      try {
        setLoading(feedback, ctx.t("sites.import.loading"));
        const results = await importServicesFiles(files, ctx);
        const warnings = [];
        const diffs = [];
        for (const item of results) {
          if (item.kind === "env" && item.missingFields?.length) {
            warnings.push(`${item.file}: ${ctx.t("sites.import.missingFields")}: ${item.missingFields.map((field) => toEnvKey(field)).join(", ")}`);
          }
          if (item.updatedExisting && item.diffLines?.length) {
            diffs.push(`${item.file} (${item.siteID}):\n${item.diffLines.slice(0, 40).join("\n")}`);
          }
        }
        if (warnings.length || diffs.length) {
          feedback.innerHTML = `
            <div class="waf-empty">
              ${warnings.length ? `<div><strong>${escapeHtml(ctx.t("sites.import.warnings"))}</strong><pre class="waf-code">${escapeHtml(warnings.join("\n"))}</pre></div>` : ""}
              ${diffs.length ? `<div><strong>${escapeHtml(ctx.t("sites.import.diff"))}</strong><pre class="waf-code">${escapeHtml(diffs.join("\n\n"))}</pre></div>` : ""}
            </div>
          `;
        } else {
          feedback.innerHTML = "";
        }
        ctx.notify(ctx.t("sites.toast.imported"));
        await load();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.error.import")}: ${String(error?.message || error)}`);
      } finally {
        event.target.value = "";
      }
    });
    container.querySelector("#services-search")?.addEventListener("input", (event) => {
      state.search = event.target.value;
      const cursorStart = Number(event.target.selectionStart || state.search.length);
      const cursorEnd = Number(event.target.selectionEnd || cursorStart);
      render();
      const nextInput = container.querySelector("#services-search");
      if (nextInput) {
        nextInput.focus();
        nextInput.setSelectionRange(cursorStart, cursorEnd);
      }
    });
    container.querySelector("#services-sort")?.addEventListener("change", (event) => {
      state.sort = event.target.value;
      render();
    });
    container.querySelector("#services-select-all")?.addEventListener("change", (event) => {
      const checked = Boolean(event.target.checked);
      for (const site of state.filteredSites) {
        if (checked) {
          state.selectedSiteIDs.add(site.id);
        } else {
          state.selectedSiteIDs.delete(site.id);
        }
      }
      render();
    });
    container.querySelectorAll("[data-select-site]").forEach((checkbox) => {
      checkbox.addEventListener("change", (event) => {
        event.stopPropagation();
        const siteID = String(event.target.dataset.selectSite || "");
        if (!siteID) {
          return;
        }
        if (event.target.checked) {
          state.selectedSiteIDs.add(siteID);
        } else {
          state.selectedSiteIDs.delete(siteID);
        }
      });
    });
    container.querySelectorAll("[data-open-site]").forEach((button) => {
      button.addEventListener("click", (event) => {
        event.stopPropagation();
        go(`${routeBase()}/${encodeURIComponent(button.dataset.openSite)}`);
      });
    });
    container.querySelectorAll("[data-open-service]").forEach((button) => {
      button.addEventListener("click", (event) => {
        event.stopPropagation();
        const url = String(button.dataset.openService || "").trim();
        if (!url) {
          return;
        }
        window.open(url, "_blank", "noopener,noreferrer");
      });
    });
    container.querySelectorAll("[data-open-site-edit]").forEach((row) => {
      row.addEventListener("click", (event) => {
        if (event.target.closest("button, input, select, textarea, a, label")) {
          return;
        }
        const siteID = String(row.dataset.openSiteEdit || "").trim();
        if (!siteID) {
          return;
        }
        go(`${routeBase()}/${encodeURIComponent(siteID)}`);
      });
    });
  };

  const bindDetail = () => {
    const feedback = container.querySelector("#sites-feedback");
    container.querySelectorAll("[data-wizard-tab]").forEach((button) => {
      button.addEventListener("click", () => {
        state.draft = getDraft();
        state.activeTab = button.dataset.wizardTab || "front";
        render();
      });
    });

    const getDraft = () => ({
      id: container.querySelector("#service-id").value.trim().toLowerCase(),
      primary_host: container.querySelector("#service-host").value.trim().toLowerCase(),
      enabled: container.querySelector("#service-enabled").checked,
      tls_enabled: container.querySelector("#service-tls-enabled").checked,
      tls_self_signed: container.querySelector("#service-tls-self-signed").checked,
      certificate_id: container.querySelector("#service-certificate-id").value.trim().toLowerCase(),
      security_mode: container.querySelector("#service-security-mode").value,
      upstream_id: computeUpstreamID(container.querySelector("#service-id").value),
      upstream_host: container.querySelector("#service-upstream-host").value.trim(),
      upstream_port: Number(container.querySelector("#service-upstream-port").value || "80"),
      upstream_scheme: container.querySelector("#service-upstream-scheme").value,
      auto_lets_encrypt: container.querySelector("#service-auto-lets-encrypt").checked,
      use_lets_encrypt_staging: container.querySelector("#service-lets-encrypt-staging").checked,
      use_lets_encrypt_wildcard: container.querySelector("#service-lets-encrypt-wildcard").checked,
      certificate_authority_server: container.querySelector("#service-ca-server").value,
      acme_account_email: normalizeEmail(state.draft.acme_account_email),
      use_reverse_proxy: container.querySelector("#service-use-reverse-proxy").checked,
      reverse_proxy_host: container.querySelector("#service-reverse-proxy-host").value.trim(),
      reverse_proxy_url: container.querySelector("#service-reverse-proxy-url").value.trim(),
      reverse_proxy_custom_host: container.querySelector("#service-reverse-proxy-custom-host").value.trim(),
      reverse_proxy_ssl_sni: container.querySelector("#service-reverse-proxy-ssl-sni").checked,
      reverse_proxy_ssl_sni_name: container.querySelector("#service-reverse-proxy-ssl-sni-name").value.trim(),
      reverse_proxy_websocket: container.querySelector("#service-reverse-proxy-websocket").checked,
      reverse_proxy_keepalive: container.querySelector("#service-reverse-proxy-keepalive").checked,
      pass_host_header: container.querySelector("#service-pass-host-header")?.checked ?? true,
      send_x_forwarded_for: container.querySelector("#service-send-x-forwarded-for")?.checked ?? true,
      send_x_forwarded_proto: container.querySelector("#service-send-x-forwarded-proto")?.checked ?? true,
      send_x_real_ip: container.querySelector("#service-send-x-real-ip")?.checked ?? false,
      allowed_methods: normalizeStringArray(state.draft.allowed_methods),
      max_client_size: container.querySelector("#service-max-client-size").value.trim(),
      http2: container.querySelector("#service-http2").checked,
      http3: container.querySelector("#service-http3").checked,
      ssl_protocols: normalizeStringArray(state.draft.ssl_protocols),
      cookie_flags: container.querySelector("#service-cookie-flags").value.trim(),
      content_security_policy: container.querySelector("#service-content-security-policy").value.trim(),
      permissions_policy: normalizeStringArray(state.draft.permissions_policy),
      keep_upstream_headers: normalizeStringArray(state.draft.keep_upstream_headers),
      referrer_policy: container.querySelector("#service-referrer-policy").value.trim(),
      use_cors: container.querySelector("#service-use-cors").checked,
      cors_allowed_origins: normalizeStringArray(state.draft.cors_allowed_origins),
      use_allowlist: container.querySelector("#service-use-allowlist")?.checked || false,
      use_exceptions: container.querySelector("#service-use-exceptions")?.checked || false,
      access_allowlist: normalizeStringArray(state.draft.access_allowlist),
      exceptions_ip: normalizeStringArray(state.draft.exceptions_ip),
      access_denylist: normalizeStringArray(state.draft.access_denylist),
      use_bad_behavior: container.querySelector("#service-use-bad-behavior").checked,
      bad_behavior_status_codes: normalizeArray(state.draft.bad_behavior_status_codes).map((item) => Number(item)).filter((item) => Number.isInteger(item)),
      bad_behavior_ban_time_seconds: Number(container.querySelector("#service-bad-behavior-ban-time").value || "300"),
      bad_behavior_threshold: Number(container.querySelector("#service-bad-behavior-threshold").value || "20"),
      bad_behavior_count_time_seconds: Number(container.querySelector("#service-bad-behavior-count-time").value || "30"),
      ban_escalation_enabled: container.querySelector("#service-ban-escalation-enabled")?.checked || false,
      ban_escalation_scope: container.querySelector("#service-ban-escalation-scope")?.value || "all_sites",
      ban_escalation_stages_seconds: normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, Number(container.querySelector("#service-bad-behavior-ban-time").value || "300")),
      use_blacklist: container.querySelector("#service-use-blacklist").checked,
      use_dnsbl: container.querySelector("#service-use-dnsbl").checked,
      blacklist_ip: normalizeStringArray(state.draft.blacklist_ip),
      blacklist_rdns: normalizeStringArray(state.draft.blacklist_rdns),
      blacklist_asn: normalizeStringArray(state.draft.blacklist_asn),
      blacklist_user_agent: normalizeStringArray(state.draft.blacklist_user_agent),
      blacklist_uri: normalizeStringArray(state.draft.blacklist_uri),
      blacklist_ip_urls: normalizeStringArray(state.draft.blacklist_ip_urls),
      blacklist_rdns_urls: normalizeStringArray(state.draft.blacklist_rdns_urls),
      blacklist_asn_urls: normalizeStringArray(state.draft.blacklist_asn_urls),
      blacklist_user_agent_urls: normalizeStringArray(state.draft.blacklist_user_agent_urls),
      blacklist_uri_urls: normalizeStringArray(state.draft.blacklist_uri_urls),
      use_limit_conn: container.querySelector("#service-use-limit-conn").checked,
      limit_conn_max_http1: Number(container.querySelector("#service-limit-conn-max-http1").value || "200"),
      limit_conn_max_http2: Number(container.querySelector("#service-limit-conn-max-http2").value || "400"),
      limit_conn_max_http3: Number(container.querySelector("#service-limit-conn-max-http3").value || "400"),
      use_limit_req: container.querySelector("#service-use-limit-req").checked,
      limit_req_url: container.querySelector("#service-limit-req-url").value.trim(),
      limit_req_rate: container.querySelector("#service-limit-req-rate").value.trim(),
      custom_limit_rules: Array.from(container.querySelectorAll("[data-custom-limit-path]"))
        .map((input) => {
          const index = String(input.dataset.customLimitPath || "");
          const rateInput = container.querySelector(`[data-custom-limit-rate="${index}"]`);
          return {
            path: String(input.value || "").trim(),
            rate: String(rateInput?.value || "").trim()
          };
        }),
      antibot_challenge: container.querySelector("#service-antibot-challenge").value,
      antibot_uri: container.querySelector("#service-antibot-uri").value.trim(),
      antibot_recaptcha_score: Number(container.querySelector("#service-antibot-recaptcha-score").value || "0.7"),
      antibot_recaptcha_sitekey: container.querySelector("#service-antibot-recaptcha-sitekey").value.trim(),
      antibot_recaptcha_secret: container.querySelector("#service-antibot-recaptcha-secret").value.trim(),
      antibot_hcaptcha_sitekey: container.querySelector("#service-antibot-hcaptcha-sitekey").value.trim(),
      antibot_hcaptcha_secret: container.querySelector("#service-antibot-hcaptcha-secret").value.trim(),
      antibot_turnstile_sitekey: container.querySelector("#service-antibot-turnstile-sitekey").value.trim(),
      antibot_turnstile_secret: container.querySelector("#service-antibot-turnstile-secret").value.trim(),
      use_auth_basic: container.querySelector("#service-use-auth-basic").checked,
      auth_basic_location: container.querySelector("#service-auth-basic-location").value.trim(),
      auth_basic_user: container.querySelector("#service-auth-basic-user").value.trim(),
      auth_basic_password: container.querySelector("#service-auth-basic-password").value.trim(),
      auth_basic_text: container.querySelector("#service-auth-basic-text").value.trim(),
      blacklist_country: normalizeStringArray(state.draft.blacklist_country),
      whitelist_country: normalizeStringArray(state.draft.whitelist_country),
      use_modsecurity: container.querySelector("#service-use-modsecurity").checked,
      use_modsecurity_crs_plugins: container.querySelector("#service-use-modsecurity-crs-plugins").checked,
      use_modsecurity_custom_configuration: container.querySelector("#service-use-modsecurity-custom-configuration").checked,
      modsecurity_crs_version: container.querySelector("#service-modsecurity-crs-version").value.trim(),
      modsecurity_crs_plugins: normalizeStringArray(state.draft.modsecurity_crs_plugins),
      modsecurity_custom_path: container.querySelector("#service-modsecurity-custom-path").value.trim(),
      modsecurity_custom_content: container.querySelector("#service-modsecurity-custom-content").value
    });

    const syncStateDraftFromForm = () => {
      state.draft = getDraft();
      state.draft.bad_behavior_status_codes = normalizeArray(state.draft.bad_behavior_status_codes)
        .map((item) => Number(item))
        .filter((item) => Number.isInteger(item))
        .sort((a, b) => a - b);
      state.draft.ban_escalation_scope = BAN_SCOPE_VALUES.includes(String(state.draft.ban_escalation_scope || "").trim().toLowerCase())
        ? String(state.draft.ban_escalation_scope || "").trim().toLowerCase()
        : "all_sites";
      state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(
        state.draft.ban_escalation_stages_seconds,
        state.draft.bad_behavior_ban_time_seconds
      );
    };

    const back = () => go(routeBase());
    const idInput = container.querySelector("#service-id");
    const hostInput = container.querySelector("#service-host");
    const certificateInput = container.querySelector("#service-certificate-id");
    const upstreamInput = container.querySelector("#service-upstream-id");
    const normalizeAutoSiteID = (value) => String(value || "").trim().toLowerCase().replace(/[^a-z0-9.-]+/g, "-").replace(/^-+|-+$/g, "");
    const syncDerivedFieldsFromID = () => {
      const id = String(idInput?.value || "").trim().toLowerCase();
      if (upstreamInput) {
        upstreamInput.value = computeUpstreamID(id);
      }
      if (certificateInput && (!certificateInput.dataset.dirty || !certificateInput.value.trim())) {
        certificateInput.value = id ? `${id}-tls` : "";
      }
    };
    if (state.route.mode !== "create") {
      if (idInput?.value?.trim()) {
        idInput.dataset.dirty = "true";
      }
      if (certificateInput?.value?.trim()) {
        certificateInput.dataset.dirty = "true";
      }
    }
    container.querySelector("#service-back")?.addEventListener("click", back);
    container.querySelector("#service-back-bottom")?.addEventListener("click", back);
    container.querySelector("#service-host")?.addEventListener("input", (event) => {
      const host = event.target.value.trim().toLowerCase();
      if (idInput && !idInput.dataset.dirty) {
        idInput.value = normalizeAutoSiteID(host);
        syncDerivedFieldsFromID();
      }
    });
    container.querySelector("#services-delete-selected")?.addEventListener("click", async () => {
      feedback.innerHTML = "";
      const selectedIDs = Array.from(state.selectedSiteIDs).filter((id) => state.sites.some((site) => normalizeSiteID(site.id) === normalizeSiteID(id)));
      if (!selectedIDs.length) {
        setError(feedback, ctx.t("sites.error.noServicesSelected"));
        return;
      }
      if (!confirmAction(ctx.t("sites.confirm.deleteSelected", { count: selectedIDs.length }))) {
        return;
      }
      try {
        setLoading(feedback, ctx.t("sites.action.deleting"));
        const sharedSnapshot = {
          sites: state.sites,
          upstreams: state.upstreams,
          tlsConfigs: state.tlsConfigs,
          easyProfiles: await ctx.api.get("/api/easy-site-profiles").catch(() => []),
          wafPolicies: await ctx.api.get("/api/waf-policies").catch(() => []),
          ratePolicies: await ctx.api.get("/api/rate-limit-policies").catch(() => []),
          accessPolicies: await ctx.api.get("/api/access-policies").catch(() => [])
        };
        let deletedCount = 0;
        for (const siteID of selectedIDs) {
          await deleteServiceWithResources(siteID, ctx, sharedSnapshot);
          deletedCount += 1;
          sharedSnapshot.sites = normalizeArray(sharedSnapshot.sites).filter((item) => normalizeSiteID(item?.id) !== normalizeSiteID(siteID));
          sharedSnapshot.upstreams = normalizeArray(sharedSnapshot.upstreams).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
          sharedSnapshot.tlsConfigs = normalizeArray(sharedSnapshot.tlsConfigs).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
          sharedSnapshot.easyProfiles = normalizeArray(sharedSnapshot.easyProfiles).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
          sharedSnapshot.wafPolicies = normalizeArray(sharedSnapshot.wafPolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
          sharedSnapshot.ratePolicies = normalizeArray(sharedSnapshot.ratePolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
          sharedSnapshot.accessPolicies = normalizeArray(sharedSnapshot.accessPolicies).filter((item) => normalizeSiteID(item?.site_id) !== normalizeSiteID(siteID));
        }
        state.selectedSiteIDs = new Set();
        ctx.notify(ctx.t("sites.toast.servicesDeleted", { count: deletedCount }));
        await load();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.error.deleteSite")}: ${String(error?.message || error)}`);
      }
    });
    container.querySelector("#service-id")?.addEventListener("input", (event) => {
      const id = event.target.value.trim().toLowerCase();
      const autoID = normalizeAutoSiteID(hostInput?.value || "");
      event.target.dataset.dirty = id && id !== autoID ? "true" : "";
      syncDerivedFieldsFromID();
    });
    container.querySelector("#service-certificate-id")?.addEventListener("input", (event) => {
      event.target.dataset.dirty = event.target.value.trim() ? "true" : "";
    });

    const toggleCertificateImportActions = () => {
      const caServer = String(container.querySelector("#service-ca-server")?.value || "").trim().toLowerCase();
      const row = container.querySelector("#service-certificate-import-actions");
      const picker = container.querySelector("#service-certificate-picker");
      if (!row) {
        return;
      }
      row.style.display = caServer === "import" ? "" : "none";
      if (picker) {
        picker.style.display = caServer === "import" ? "" : "none";
      }
    };

    container.querySelector("#service-ca-server")?.addEventListener("change", toggleCertificateImportActions);
    const certificateArchiveInput = container.querySelector("#service-certificate-archive-file");
    container.querySelector("#service-import-certificate-search")?.addEventListener("change", (event) => {
      const selectedID = String(event?.target?.value || "").trim().toLowerCase();
      if (!selectedID) {
        return;
      }
      const certificateInput = container.querySelector("#service-certificate-id");
      certificateInput.value = selectedID;
      certificateInput.dataset.dirty = "true";
    });

    container.querySelector("#service-certificate-import")?.addEventListener("click", () => {
      certificateArchiveInput?.click();
    });

    certificateArchiveInput?.addEventListener("change", async () => {
      const archiveFile = certificateArchiveInput?.files?.[0] || null;
      if (!archiveFile) {
        return;
      }
      const formData = new FormData();
      formData.set("archive_file", archiveFile);
      try {
        const result = await ctx.api.post("/api/certificate-materials/import-archive", formData);
        const importedCount = Number(result?.imported_count || 0);
        const firstCertificateID = String(result?.items?.[0]?.certificate?.id || "").trim();
        if (firstCertificateID) {
          const certificateInput = container.querySelector("#service-certificate-id");
          if (!String(certificateInput.value || "").trim()) {
            certificateInput.value = firstCertificateID;
            certificateInput.dataset.dirty = "true";
          }
        }
        ctx.notify(importedCount > 0 ? ctx.t("tls.certificates.importedArchive", { count: importedCount }) : ctx.t("sites.tls.imported"));
        await load();
      } catch (error) {
        setError(feedback, `${ctx.t("sites.tls.importFailed")}: ${String(error?.message || error)}`);
      } finally {
        if (certificateArchiveInput) {
          certificateArchiveInput.value = "";
        }
      }
    });

    container.querySelector("#service-certificate-export")?.addEventListener("click", async () => {
      const certificateID = String(container.querySelector("#service-certificate-id")?.value || "").trim().toLowerCase();
      if (!certificateID) {
        ctx.notify(ctx.t("sites.tls.certificateIdRequired"), "error");
        return;
      }
      try {
        const response = await fetch("/api/certificate-materials/export", {
          method: "POST",
          credentials: "include",
          headers: {
            Accept: "application/zip, application/json",
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ certificate_ids: [certificateID] })
        });
        if (!response.ok) {
          const bodyText = await response.text();
          let message = `HTTP ${response.status}`;
          if (bodyText) {
            try {
              const payload = JSON.parse(bodyText);
              message = String(payload?.error || payload?.message || message);
            } catch {
              message = bodyText;
            }
          }
          throw new Error(message);
        }
        const blob = await response.blob();
        downloadBlob(`${certificateID}-materials.zip`, blob);
        ctx.notify(ctx.t("sites.tls.exported"));
      } catch (error) {
        setError(feedback, `${ctx.t("sites.tls.exportFailed")}: ${String(error?.message || error)}`);
      }
    });

    toggleCertificateImportActions();
    container.querySelector("#service-use-modsecurity-custom-configuration")?.addEventListener("change", () => {
      syncStateDraftFromForm();
      render();
    });
    container.querySelector("#service-pass-host-header")?.addEventListener("change", () => {
      syncStateDraftFromForm();
      render();
    });

    const highlightSelector = (selector) => {
      if (!selector) {
        return;
      }
      const target = container.querySelector(selector);
      if (!target) {
        return;
      }
      target.classList.add("waf-search-highlight");
      window.setTimeout(() => target.classList.remove("waf-search-highlight"), 2200);
      if (typeof target.scrollIntoView === "function") {
        target.scrollIntoView({ behavior: "smooth", block: "center" });
      }
      if (typeof target.focus === "function") {
        target.focus({ preventScroll: true });
      }
    };

    container.querySelector("#service-settings-search")?.addEventListener("input", (event) => {
      state.settingsSearch = String(event.target.value || "");
      state.highlightedSelector = "";
      const cursorStart = Number(event.target.selectionStart || state.settingsSearch.length);
      const cursorEnd = Number(event.target.selectionEnd || cursorStart);
      render();
      const nextInput = container.querySelector("#service-settings-search");
      if (nextInput) {
        nextInput.focus();
        nextInput.setSelectionRange(cursorStart, cursorEnd);
      }
    });

    container.querySelectorAll("[data-settings-result]").forEach((button) => {
      button.addEventListener("click", () => {
        syncStateDraftFromForm();
        state.activeTab = String(button.dataset.settingsTab || "front");
        state.settingsSearch = "";
        state.highlightedSelector = String(button.dataset.settingsSelector || "");
        render();
        window.setTimeout(() => highlightSelector(state.highlightedSelector), 30);
      });
    });

    container.querySelectorAll("[data-list-add]").forEach((button) => {
      button.addEventListener("click", () => {
        const field = button.dataset.listAdd || "";
        if (!LIST_FIELD_SET.has(field)) {
          return;
        }
        const input = container.querySelector(`#list-input-${field}`);
        if (!input) {
          return;
        }
        const value = String(input.value || "").trim();
        if (!value) {
          return;
        }
        syncStateDraftFromForm();
        const current = normalizeStringArray(state.draft[field]);
        if (!current.includes(value)) {
          current.push(value);
          state.draft[field] = current;
        }
        render();
      });
    });

    container.querySelectorAll("[data-list-template-add]").forEach((button) => {
      button.addEventListener("click", () => {
        const field = button.dataset.listTemplateAdd || "";
        if (!LIST_FIELD_SET.has(field)) {
          return;
        }
        const select = container.querySelector(`#list-template-${field}`);
        const presetID = String(select?.value || state.listTemplateSelection[field] || "").trim();
        if (!presetID) {
          return;
        }
        syncStateDraftFromForm();
        const preset = getQuickListTemplates(field).find((item) => item.id === presetID);
        if (!preset) {
          return;
        }
        const current = new Set(normalizeStringArray(state.draft[field]));
        for (const item of normalizeStringArray(preset.items)) {
          current.add(item);
        }
        state.listTemplateSelection[field] = presetID;
        state.draft[field] = Array.from(current);
        render();
      });
    });

    container.querySelectorAll("[id^='list-template-']").forEach((select) => {
      select.addEventListener("change", () => {
        const field = String(select.id || "").replace("list-template-", "");
        if (!field) {
          return;
        }
        state.listTemplateSelection[field] = String(select.value || "");
      });
    });

    container.querySelectorAll("[data-country-toggle]").forEach((checkbox) => {
      checkbox.addEventListener("change", () => {
        const field = checkbox.dataset.countryToggle || "";
        const value = String(checkbox.dataset.countryValue || "").trim().toUpperCase();
        if (!LIST_FIELD_SET.has(field) || !value) {
          return;
        }
        syncStateDraftFromForm();
        const current = new Set(normalizeStringArray(state.draft[field]));
        if (checkbox.checked) {
          current.add(value);
        } else {
          current.delete(value);
        }
        state.draft[field] = Array.from(current);
        render();
      });
    });

    container.querySelectorAll("[id^='country-search-']").forEach((input) => {
      input.addEventListener("input", (event) => {
        const field = String(input.id || "").replace("country-search-", "");
        if (!field) {
          return;
        }
        state.countryFilters[field] = String(event.target.value || "");
        const cursorStart = Number(event.target.selectionStart || state.countryFilters[field].length);
        const cursorEnd = Number(event.target.selectionEnd || cursorStart);
        render();
        const nextInput = container.querySelector(`#country-search-${field}`);
        if (nextInput) {
          nextInput.focus();
          nextInput.setSelectionRange(cursorStart, cursorEnd);
        }
      });
    });

    container.querySelectorAll("[data-list-remove]").forEach((button) => {
      button.addEventListener("click", () => {
        const field = button.dataset.listRemove || "";
        if (!LIST_FIELD_SET.has(field)) {
          return;
        }
        const index = Number(button.dataset.listIndex || "-1");
        syncStateDraftFromForm();
        const current = normalizeStringArray(state.draft[field]);
        if (index < 0 || index >= current.length) {
          return;
        }
        current.splice(index, 1);
        state.draft[field] = current;
        render();
      });
    });

    container.querySelector("[data-custom-limit-add]")?.addEventListener("click", () => {
      syncStateDraftFromForm();
      state.draft.custom_limit_rules = [...normalizeCustomLimitRules(state.draft.custom_limit_rules), { path: "/", rate: "10r/s" }];
      render();
    });

    container.querySelectorAll("[data-custom-limit-remove]").forEach((button) => {
      button.addEventListener("click", () => {
        const index = Number.parseInt(String(button.dataset.customLimitRemove || "-1"), 10);
        if (!Number.isInteger(index) || index < 0) {
          return;
        }
        syncStateDraftFromForm();
        const current = normalizeCustomLimitRules(state.draft.custom_limit_rules);
        if (index >= current.length) {
          return;
        }
        current.splice(index, 1);
        state.draft.custom_limit_rules = current;
        render();
      });
    });

    container.querySelectorAll("[data-bad-code]").forEach((checkbox) => {
      checkbox.addEventListener("change", () => {
        const code = Number(checkbox.dataset.badCode || "0");
        if (!Number.isInteger(code) || code <= 0) {
          return;
        }
        syncStateDraftFromForm();
        const selected = new Set(
          normalizeArray(state.draft.bad_behavior_status_codes)
            .map((item) => Number(item))
            .filter((item) => Number.isInteger(item))
        );
        if (checkbox.checked) {
          selected.add(code);
        } else {
          selected.delete(code);
        }
        state.draft.bad_behavior_status_codes = Array.from(selected).sort((a, b) => a - b);
      });
    });

    container.querySelector("[data-ban-stage-add]")?.addEventListener("click", () => {
      const input = container.querySelector("#service-ban-stage-input");
      if (!input) {
        return;
      }
      const parsed = parseBanDurationSeconds(input.value);
      if (parsed === null) {
        setError(feedback, ctx.t("sites.validation.banStageFormat"));
        return;
      }
      syncStateDraftFromForm();
      const current = normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, state.draft.bad_behavior_ban_time_seconds);
      current.push(parsed);
      state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(current, state.draft.bad_behavior_ban_time_seconds);
      render();
    });

    container.querySelectorAll("[data-ban-stage-remove]").forEach((button) => {
      button.addEventListener("click", () => {
        const index = Number.parseInt(String(button.dataset.banStageRemove || "-1"), 10);
        if (!Number.isInteger(index) || index < 0) {
          return;
        }
        syncStateDraftFromForm();
        const current = normalizeBanEscalationStages(state.draft.ban_escalation_stages_seconds, state.draft.bad_behavior_ban_time_seconds);
        if (index >= current.length) {
          return;
        }
        current.splice(index, 1);
        state.draft.ban_escalation_stages_seconds = normalizeBanEscalationStages(current, state.draft.bad_behavior_ban_time_seconds);
        render();
      });
    });

    container.querySelector("#service-editor-form")?.addEventListener("submit", async (event) => {
      event.preventDefault();
      syncStateDraftFromForm();
      const draft = ensureControlPlaneAccessManagementMethods(getDraft());
      const validationError = validateDraft(draft, ctx);
      if (validationError) {
        setError(feedback, validationError);
        return;
      }
      try {
        setLoading(feedback, ctx.t("sites.editor.saving"));
        const existingSite = state.sites.find((item) => normalizeSiteID(item?.id) === normalizeSiteID(state.route.siteID) || normalizeSiteID(item?.id) === normalizeSiteID(draft.id));
        const existingUpstream = state.upstreams.find((item) => item.id === draft.upstream_id);
        const existingTLSConfig = state.tlsBySite.get(draft.id) || null;
        const existingAccessPolicy = state.accessBySite.get(normalizeSiteID(draft.id)) || null;
        if (shouldUpsertBaseResources(draft, existingSite, existingUpstream, existingTLSConfig)) {
          await upsertSiteResources(draft, ctx, existingSite, existingUpstream, existingTLSConfig);
        }
        await upsertAccessPolicy(draft, ctx, existingAccessPolicy);
        const easyProfilePath = `/api/easy-site-profiles/${encodeURIComponent(draft.id)}`;
        await putWithPostFallback(ctx, easyProfilePath, draftToEasyProfile(draft), {
          tolerateAutoApplyError: true,
          verifyPersisted: async () => {
            try {
              const persisted = await ctx.api.get(easyProfilePath);
              return normalizeSiteID(persisted?.site_id) === normalizeSiteID(draft.id);
            } catch (_error) {
              return false;
            }
          }
        });
        ctx.notify(ctx.t("toast.siteSaved"));
        go(`${routeBase()}/${encodeURIComponent(draft.id)}`);
      } catch (error) {
        console.warn("save site failed", error);
        const backendMessage = String(error?.message || "").trim();
        setError(feedback, backendMessage ? `${ctx.t("sites.error.saveSite")}: ${backendMessage}` : ctx.t("sites.error.saveSite"));
      }
    });

    container.querySelector("#service-delete")?.addEventListener("click", async () => {
      const siteID = state.route.siteID;
      if (!siteID) {
        return;
      }
      if (!confirmAction(ctx.t("sites.confirm.deleteSite", { id: siteID }))) {
        return;
      }
      try {
        await deleteServiceWithResources(siteID, ctx, {
          upstreams: state.upstreams
        });
        ctx.notify(ctx.t("toast.siteDeleted"));
        back();
      } catch (error) {
        setError(feedback, ctx.t("sites.error.deleteSite"));
      }
    });

    if (state.highlightedSelector) {
      window.setTimeout(() => highlightSelector(state.highlightedSelector), 30);
    }
  };

  const bind = () => {
    if (state.route.mode === "list") {
      bindList();
      return;
    }
    bindDetail();
  };

  await load();
}

