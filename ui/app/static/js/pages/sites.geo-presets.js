export const BAD_BEHAVIOR_STATUS_OPTIONS = [
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

export const CONTINENT_VALUES = ["AF", "AN", "AS", "EU", "NA", "OC", "SA"];
export const COUNTRY_GROUP_VALUES = ["APAC", "EMEA", "LATAM", "DACH", "CIS", "GCC", "NORAM"];
export const GEO_SELECTOR_LABELS = {
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
export const QUICK_LIST_TEMPLATES = {
  blacklist_user_agent: [
    {
      id: "scanner_uas",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.scanner_uas",
      items: ["sqlmap", "nikto", "nmap", "masscan", "zgrab", "gobuster", "dirbuster", "wpscan", "acunetix", "nessus"]
    },
    {
      id: "cli_clients",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.cli_clients",
      items: ["curl/.*", "python-requests", "python-httpx", "aiohttp", "Go-http-client", "libwww-perl"]
    },
    {
      id: "headless_tools",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.headless_tools",
      items: ["HeadlessChrome", "PhantomJS", "selenium", "playwright"]
    },
    {
      id: "fuzzers_discovery",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.fuzzers_discovery",
      items: ["ffuf", "feroxbuster", "wfuzz", "dirsearch", "gospider", "hakrawler"]
    },
    {
      id: "exploit_frameworks",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.exploit_frameworks",
      items: ["metasploit", "nuclei", "jaeles", "arachni", "w3af", "commix"]
    },
    {
      id: "generic_http_clients",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.generic_http_clients",
      items: ["python-urllib", "python-httplib2", "Java/", "Apache-HttpClient", "okhttp", "restsharp"]
    },
    {
      id: "legacy_scrapers",
      labelKey: "sites.easy.traffic.template.blacklistUserAgent.legacy_scrapers",
      items: ["HTTrack", "WebCopier", "WinHTTrack", "MJ12bot", "SemrushBot", "DotBot"]
    }
  ],
  blacklist_uri: [
    {
      id: "common_probe_paths",
      labelKey: "sites.easy.traffic.template.blacklistUri.common_probe_paths",
      items: ["/\\.env", "/\\.git", "/\\.svn", "/server-status", "/actuator", "/manager/html", "/cgi-bin", "/boaform", "/phpinfo"]
    },
    {
      id: "wordpress_probes",
      labelKey: "sites.easy.traffic.template.blacklistUri.wordpress_probes",
      items: ["/wp-admin", "/wp-login\\.php", "/xmlrpc\\.php", "/wp-content", "/wp-includes"]
    },
    {
      id: "admin_panels",
      labelKey: "sites.easy.traffic.template.blacklistUri.admin_panels",
      items: ["/phpmyadmin", "/pma", "/adminer", "/vendor/phpunit"]
    },
    {
      id: "sensitive_files",
      labelKey: "sites.easy.traffic.template.blacklistUri.sensitive_files",
      items: ["/\\.DS_Store", "/id_rsa", "/\\.aws/credentials", "/composer\\.(json|lock)", "/package(-lock)?\\.json", "/yarn\\.lock"]
    },
    {
      id: "backup_leaks",
      labelKey: "sites.easy.traffic.template.blacklistUri.backup_leaks",
      items: ["/backup", "/backup\\.(zip|tar|tar\\.gz|sql)", "/dump\\.sql", "/\\.git\\.zip", "/\\.bak$", "/\\.old$"]
    },
    {
      id: "framework_debug",
      labelKey: "sites.easy.traffic.template.blacklistUri.framework_debug",
      items: ["/_profiler", "/_debugbar", "/debug/default/view", "/console", "/app_dev\\.php", "/server-info"]
    },
    {
      id: "api_docs_admin",
      labelKey: "sites.easy.traffic.template.blacklistUri.api_docs_admin",
      items: ["/swagger", "/swagger-ui", "/v2/api-docs", "/v3/api-docs", "/openapi\\.json", "/graphql$"]
    },
    {
      id: "common_shells",
      labelKey: "sites.easy.traffic.template.blacklistUri.common_shells",
      items: ["/shell\\.php", "/cmd\\.php", "/r57\\.php", "/c99\\.php", "/wso\\.php", "/mini\\.php"]
    }
  ]
};
