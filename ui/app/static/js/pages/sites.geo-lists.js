export const BAD_BEHAVIOR_STATUS_OPTIONS = [
  [400, "Bad Request"], [401, "Unauthorized"], [402, "Payment Required"], [403, "Forbidden"], [404, "Not Found"],
  [405, "Method Not Allowed"], [406, "Not Acceptable"], [407, "Proxy Authentication Required"], [408, "Request Timeout"],
  [409, "Conflict"], [410, "Gone"], [411, "Length Required"], [412, "Precondition Failed"], [413, "Payload Too Large"],
  [414, "URI Too Long"], [415, "Unsupported Media Type"], [416, "Range Not Satisfiable"], [417, "Expectation Failed"],
  [418, "I'm a teapot"], [421, "Misdirected Request"], [422, "Unprocessable Entity"], [423, "Locked"], [424, "Failed Dependency"],
  [425, "Too Early"], [426, "Upgrade Required"], [428, "Precondition Required"], [429, "Too Many Requests"],
  [431, "Request Header Fields Too Large"], [444, "No Response (nginx)"], [451, "Unavailable For Legal Reasons"],
  [500, "Internal Server Error"], [501, "Not Implemented"], [502, "Bad Gateway"], [503, "Service Unavailable"],
  [504, "Gateway Timeout"], [505, "HTTP Version Not Supported"], [507, "Insufficient Storage"], [508, "Loop Detected"],
  [510, "Not Extended"], [511, "Network Authentication Required"], [520, "Unknown Error (Cloudflare)"],
  [521, "Web Server Is Down (Cloudflare)"], [522, "Connection Timed Out (Cloudflare)"], [523, "Origin Is Unreachable (Cloudflare)"],
  [524, "A Timeout Occurred (Cloudflare)"], [525, "SSL Handshake Failed (Cloudflare)"], [526, "Invalid SSL Certificate (Cloudflare)"]
];

const CONTINENT_VALUES = ["AF", "AN", "AS", "EU", "NA", "OC", "SA"];
const COUNTRY_GROUP_VALUES = ["APAC", "EMEA", "LATAM", "DACH", "CIS", "GCC", "NORAM"];
const GEO_SELECTOR_LABELS = {
  AF: "Africa", AN: "Antarctica", AS: "Asia", EU: "Europe", NA: "North America", OC: "Oceania", SA: "South America",
  APAC: "Asia-Pacific", EMEA: "Europe, Middle East and Africa", LATAM: "Latin America", DACH: "DACH", CIS: "CIS",
  GCC: "Gulf Cooperation Council", NORAM: "North America"
};

const QUICK_LIST_TEMPLATES = {
  blacklist_user_agent: [
    { id: "scanner_uas", labelKey: "sites.easy.traffic.template.blacklistUserAgent.scanner_uas", items: ["sqlmap", "nikto", "nmap", "masscan", "zgrab", "gobuster", "dirbuster", "wpscan", "acunetix", "nessus"] },
    { id: "cli_clients", labelKey: "sites.easy.traffic.template.blacklistUserAgent.cli_clients", items: ["curl/.*", "python-requests", "python-httpx", "aiohttp", "Go-http-client", "libwww-perl"] },
    { id: "headless_tools", labelKey: "sites.easy.traffic.template.blacklistUserAgent.headless_tools", items: ["HeadlessChrome", "PhantomJS", "selenium", "playwright"] },
    { id: "fuzzers_discovery", labelKey: "sites.easy.traffic.template.blacklistUserAgent.fuzzers_discovery", items: ["ffuf", "feroxbuster", "wfuzz", "dirsearch", "gospider", "hakrawler"] },
    { id: "exploit_frameworks", labelKey: "sites.easy.traffic.template.blacklistUserAgent.exploit_frameworks", items: ["metasploit", "nuclei", "jaeles", "arachni", "w3af", "commix"] },
    { id: "generic_http_clients", labelKey: "sites.easy.traffic.template.blacklistUserAgent.generic_http_clients", items: ["python-urllib", "python-httplib2", "Java/", "Apache-HttpClient", "okhttp", "restsharp"] },
    { id: "legacy_scrapers", labelKey: "sites.easy.traffic.template.blacklistUserAgent.legacy_scrapers", items: ["HTTrack", "WebCopier", "WinHTTrack", "MJ12bot", "SemrushBot", "DotBot"] }
  ],
  blacklist_uri: [
    { id: "common_probe_paths", labelKey: "sites.easy.traffic.template.blacklistUri.common_probe_paths", items: ["/\\.env", "/\\.git", "/\\.svn", "/server-status", "/actuator", "/manager/html", "/cgi-bin", "/boaform", "/phpinfo"] },
    { id: "wordpress_probes", labelKey: "sites.easy.traffic.template.blacklistUri.wordpress_probes", items: ["/wp-admin", "/wp-login\\.php", "/xmlrpc\\.php", "/wp-content", "/wp-includes"] },
    { id: "admin_panels", labelKey: "sites.easy.traffic.template.blacklistUri.admin_panels", items: ["/phpmyadmin", "/pma", "/adminer", "/vendor/phpunit"] },
    { id: "sensitive_files", labelKey: "sites.easy.traffic.template.blacklistUri.sensitive_files", items: ["/\\.DS_Store", "/id_rsa", "/\\.aws/credentials", "/composer\\.(json|lock)", "/package(-lock)?\\.json", "/yarn\\.lock"] },
    { id: "backup_leaks", labelKey: "sites.easy.traffic.template.blacklistUri.backup_leaks", items: ["/backup", "/backup\\.(zip|tar|tar\\.gz|sql)", "/dump\\.sql", "/\\.git\\.zip", "/\\.bak$", "/\\.old$"] },
    { id: "framework_debug", labelKey: "sites.easy.traffic.template.blacklistUri.framework_debug", items: ["/_profiler", "/_debugbar", "/debug/default/view", "/console", "/app_dev\\.php", "/server-info"] },
    { id: "api_docs_admin", labelKey: "sites.easy.traffic.template.blacklistUri.api_docs_admin", items: ["/swagger", "/swagger-ui", "/v2/api-docs", "/v3/api-docs", "/openapi\\.json", "/graphql$"] },
    { id: "common_shells", labelKey: "sites.easy.traffic.template.blacklistUri.common_shells", items: ["/shell\\.php", "/cmd\\.php", "/r57\\.php", "/c99\\.php", "/wso\\.php", "/mini\\.php"] }
  ],
  blacklist_ja3: [
    { id: "shodan_masscan", labelKey: "sites.easy.traffic.template.blacklistJa3.shodan_masscan", items: ["c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8", "6bea65473a4ae23c37fcaef4c3eb5b6a", "7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d", "3b5074b1b5d032e5620f69f9f700ff0e", "2a1a0cf3b09a95a3c3db069e3a7a4e66"] },
    { id: "exploit_tools", labelKey: "sites.easy.traffic.template.blacklistJa3.exploit_tools", items: ["c12f54a3f91dc7bafd92cb59fe009a35", "bc6c386f480f5a8b9c9bcf632e57e517", "6734f37431670b3ab4292b8f60f29984", "72a589da586844d7f0818ce684948eea"] },
    { id: "c2_frameworks", labelKey: "sites.easy.traffic.template.blacklistJa3.c2_frameworks", items: ["a0e9f5d64349fb13191bc781f81f42e1", "805d704c2ea8dc4c5f30a1e7dd31c2b6", "de9f2c7fd25e1b3afad3e85a0bd17d9b"] }
  ]
};

export function regionDisplayName(code) {
  const normalized = String(code || "").trim().toUpperCase();
  if (!normalized) return "";
  if (GEO_SELECTOR_LABELS[normalized]) return GEO_SELECTOR_LABELS[normalized];
  try {
    const display = new Intl.DisplayNames(["en"], { type: "region" });
    return display.of(normalized) || normalized;
  } catch (_error) {
    return normalized;
  }
}

function isCountryCode(value) {
  const normalized = String(value || "").trim().toUpperCase();
  return /^[A-Z]{2}$/.test(normalized) && !Object.prototype.hasOwnProperty.call(GEO_SELECTOR_LABELS, normalized);
}

function countryFlagEmoji(code) {
  const normalized = String(code || "").trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(normalized)) return "";
  const base = 0x1f1e6;
  const first = normalized.charCodeAt(0) - 65;
  const second = normalized.charCodeAt(1) - 65;
  if (first < 0 || first > 25 || second < 0 || second > 25) return "";
  return String.fromCodePoint(base + first, base + second);
}

export function regionDisplayLabel(code) {
  const normalized = String(code || "").trim().toUpperCase();
  const name = regionDisplayName(normalized);
  if (!name) return normalized;
  if (!normalized) return name;
  if (isCountryCode(normalized)) {
    const flag = countryFlagEmoji(normalized);
    return flag ? `${name} (${flag})` : name;
  }
  return `${name} (${normalized})`;
}

export function buildGeoCatalogFallback() {
  const countries = [];
  for (let first = 65; first <= 90; first += 1) {
    for (let second = 65; second <= 90; second += 1) {
      const code = String.fromCharCode(first) + String.fromCharCode(second);
      if (regionDisplayName(code) !== code) countries.push(code);
    }
  }
  return [...CONTINENT_VALUES, ...COUNTRY_GROUP_VALUES, ...countries];
}

export function normalizeGeoCatalogPayload(payload) {
  const continents = normalizeStringArray(payload?.continents).map((value) => value.toUpperCase());
  const groups = normalizeStringArray(payload?.groups).map((value) => value.toUpperCase());
  const countries = normalizeStringArray(payload?.countries)
    .map((value) => value.toUpperCase())
    .filter((value) => /^[A-Z]{2}$/.test(value) && regionDisplayName(value) !== value);
  const merged = Array.from(new Set([...continents, ...groups, ...countries]));
  return merged.length ? merged : buildGeoCatalogFallback();
}

export function getQuickListTemplates(field) {
  return Array.isArray(QUICK_LIST_TEMPLATES[field]) ? QUICK_LIST_TEMPLATES[field] : [];
}

export const LIST_FIELD_SET = new Set([
  "allowed_methods", "ssl_protocols", "permissions_policy", "keep_upstream_headers", "cors_allowed_origins",
  "access_allowlist", "exceptions_ip", "exceptions_uri", "access_denylist", "blacklist_ip", "blacklist_rdns", "blacklist_asn",
  "blacklist_user_agent", "blacklist_uri", "blacklist_ja3", "blacklist_ip_urls", "blacklist_rdns_urls", "blacklist_asn_urls",
  "blacklist_user_agent_urls", "blacklist_uri_urls", "blacklist_ja3_urls", "blacklist_country", "whitelist_country", "modsecurity_crs_plugins"
]);

export const SETTINGS_SEARCH_INDEX = [
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
  { id: "exceptions_uri", tab: "traffic", selector: "#list-input-exceptions_uri", labelKey: "sites.easy.traffic.exceptionsUri" },
  { id: "access_denylist", tab: "traffic", selector: "#list-input-access_denylist", labelKey: "sites.lists.denylist" },
  { id: "use_blacklist", tab: "traffic", selector: "#service-use-blacklist", labelKey: "sites.easy.traffic.activateBlacklisting" },
  { id: "blacklist_ja3", tab: "traffic", selector: "#list-input-blacklist_ja3", labelKey: "sites.easy.traffic.blacklistJa3" },
  { id: "use_limit_req", tab: "traffic", selector: "#service-use-limit-req", labelKey: "sites.easy.traffic.activateLimitRequests" },
  { id: "ban_escalation_enabled", tab: "blocking", selector: "#service-ban-escalation-enabled", labelKey: "sites.easy.blocking.enabled" },
  { id: "ban_escalation_scope", tab: "blocking", selector: "#service-ban-escalation-scope", labelKey: "sites.easy.blocking.scope" },
  { id: "antibot_challenge", tab: "antibot", selector: "#service-antibot-challenge", labelKey: "sites.easy.antibot.challenge" },
  { id: "blacklist_country", tab: "geo", selector: "#country-search-blacklist_country", labelKey: "sites.easy.geo.countryBlacklist" },
  { id: "whitelist_country", tab: "geo", selector: "#country-search-whitelist_country", labelKey: "sites.easy.geo.countryWhitelist" },
  { id: "use_modsecurity", tab: "modsec", selector: "#service-use-modsecurity", labelKey: "sites.easy.modsec.useModsecurity" }
];
import { normalizeStringArray } from "./sites.normalize.js";
