package easysiteprofiles

import (
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	SecurityModeBlock       = "block"
	SecurityModeMonitor     = "monitor"
	SecurityModeTransparent = "transparent"

	AntibotChallengeNo         = "no"
	AntibotChallengeCookie     = "cookie"
	AntibotChallengeJavascript = "javascript"
	AntibotChallengeCaptcha    = "captcha"
	AntibotChallengeRecaptcha  = "recaptcha"
	AntibotChallengeHcaptcha   = "hcaptcha"
	AntibotChallengeTurnstile  = "turnstile"
	AntibotChallengeMcaptcha   = "mcaptcha"

	AuthBasicLocationSitewide = "sitewide"
)

type FrontServiceSettings struct {
	ServerName                 string `json:"server_name"`
	SecurityMode               string `json:"security_mode"`
	AutoLetsEncrypt            bool   `json:"auto_lets_encrypt"`
	UseLetsEncryptStaging      bool   `json:"use_lets_encrypt_staging"`
	UseLetsEncryptWildcard     bool   `json:"use_lets_encrypt_wildcard"`
	CertificateAuthorityServer string `json:"certificate_authority_server"`
}

type UpstreamRoutingSettings struct {
	UseReverseProxy        bool   `json:"use_reverse_proxy"`
	ReverseProxyHost       string `json:"reverse_proxy_host"`
	ReverseProxyURL        string `json:"reverse_proxy_url"`
	ReverseProxyCustomHost string `json:"reverse_proxy_custom_host"`
	ReverseProxySSLSNI     bool   `json:"reverse_proxy_ssl_sni"`
	ReverseProxySSLSNIName string `json:"reverse_proxy_ssl_sni_name"`
	ReverseProxyWebsocket  bool   `json:"reverse_proxy_websocket"`
	ReverseProxyKeepalive  bool   `json:"reverse_proxy_keepalive"`
}

type HTTPBehaviorSettings struct {
	AllowedMethods []string `json:"allowed_methods"`
	MaxClientSize  string   `json:"max_client_size"`
	HTTP2          bool     `json:"http2"`
	HTTP3          bool     `json:"http3"`
	SSLProtocols   []string `json:"ssl_protocols"`
}

type HTTPHeadersSettings struct {
	CookieFlags           string   `json:"cookie_flags"`
	ContentSecurityPolicy string   `json:"content_security_policy"`
	PermissionsPolicy     []string `json:"permissions_policy"`
	KeepUpstreamHeaders   []string `json:"keep_upstream_headers"`
	ReferrerPolicy        string   `json:"referrer_policy"`
	UseCORS               bool     `json:"use_cors"`
	CORSAllowedOrigins    []string `json:"cors_allowed_origins"`
}

type SecurityBehaviorAndLimitsSettings struct {
	UseBadBehavior              bool  `json:"use_bad_behavior"`
	BadBehaviorStatusCodes      []int `json:"bad_behavior_status_codes"`
	BadBehaviorBanTimeSeconds   int   `json:"bad_behavior_ban_time_seconds"`
	BadBehaviorThreshold        int   `json:"bad_behavior_threshold"`
	BadBehaviorCountTimeSeconds int   `json:"bad_behavior_count_time_seconds"`

	UseBlacklist           bool     `json:"use_blacklist"`
	UseDNSBL               bool     `json:"use_dnsbl"`
	BlacklistIP            []string `json:"blacklist_ip"`
	BlacklistRDNS          []string `json:"blacklist_rdns"`
	BlacklistASN           []string `json:"blacklist_asn"`
	BlacklistUserAgent     []string `json:"blacklist_user_agent"`
	BlacklistURI           []string `json:"blacklist_uri"`
	BlacklistIPURLs        []string `json:"blacklist_ip_urls"`
	BlacklistRDNSURLs      []string `json:"blacklist_rdns_urls"`
	BlacklistASNURLs       []string `json:"blacklist_asn_urls"`
	BlacklistUserAgentURLs []string `json:"blacklist_user_agent_urls"`
	BlacklistURIURLs       []string `json:"blacklist_uri_urls"`

	UseLimitConn      bool   `json:"use_limit_conn"`
	LimitConnMaxHTTP1 int    `json:"limit_conn_max_http1"`
	LimitConnMaxHTTP2 int    `json:"limit_conn_max_http2"`
	LimitConnMaxHTTP3 int    `json:"limit_conn_max_http3"`
	UseLimitReq       bool   `json:"use_limit_req"`
	LimitReqURL       string `json:"limit_req_url"`
	LimitReqRate      string `json:"limit_req_rate"`
}

type SecurityAntibotSettings struct {
	AntibotChallenge        string  `json:"antibot_challenge"`
	AntibotURI              string  `json:"antibot_uri"`
	AntibotRecaptchaScore   float64 `json:"antibot_recaptcha_score"`
	AntibotRecaptchaSitekey string  `json:"antibot_recaptcha_sitekey"`
	AntibotRecaptchaSecret  string  `json:"antibot_recaptcha_secret"`
	AntibotHcaptchaSitekey  string  `json:"antibot_hcaptcha_sitekey"`
	AntibotHcaptchaSecret   string  `json:"antibot_hcaptcha_secret"`
	AntibotTurnstileSitekey string  `json:"antibot_turnstile_sitekey"`
	AntibotTurnstileSecret  string  `json:"antibot_turnstile_secret"`
}

type SecurityAuthBasicSettings struct {
	UseAuthBasic      bool   `json:"use_auth_basic"`
	AuthBasicLocation string `json:"auth_basic_location"`
	AuthBasicUser     string `json:"auth_basic_user"`
	AuthBasicPassword string `json:"auth_basic_password"`
	AuthBasicText     string `json:"auth_basic_text"`
}

type SecurityCountryPolicySettings struct {
	BlacklistCountry []string `json:"blacklist_country"`
	WhitelistCountry []string `json:"whitelist_country"`
}

type ModSecurityCustomConfiguration struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type SecurityModSecuritySettings struct {
	UseModSecurity           bool                           `json:"use_modsecurity"`
	UseModSecurityCRSPlugins bool                           `json:"use_modsecurity_crs_plugins"`
	ModSecurityCRSVersion    string                         `json:"modsecurity_crs_version"`
	ModSecurityCRSPlugins    []string                       `json:"modsecurity_crs_plugins"`
	CustomConfiguration      ModSecurityCustomConfiguration `json:"custom_configuration"`
}

// EasySiteProfile is an Easy-mode site configuration aggregate.
type EasySiteProfile struct {
	SiteID                    string                            `json:"site_id"`
	FrontService              FrontServiceSettings              `json:"front_service"`
	UpstreamRouting           UpstreamRoutingSettings           `json:"upstream_routing"`
	HTTPBehavior              HTTPBehaviorSettings              `json:"http_behavior"`
	HTTPHeaders               HTTPHeadersSettings               `json:"http_headers"`
	SecurityBehaviorAndLimits SecurityBehaviorAndLimitsSettings `json:"security_behavior_and_limits"`
	SecurityAntibot           SecurityAntibotSettings           `json:"security_antibot"`
	SecurityAuthBasic         SecurityAuthBasicSettings         `json:"security_auth_basic"`
	SecurityCountryPolicy     SecurityCountryPolicySettings     `json:"security_country_policy"`
	SecurityModSecurity       SecurityModSecuritySettings       `json:"security_modsecurity"`
	CreatedAt                 string                            `json:"created_at"`
	UpdatedAt                 string                            `json:"updated_at"`
}

func DefaultProfile(siteID string) EasySiteProfile {
	allowedMethods := []string{"GET", "POST", "HEAD", "OPTIONS"}
	isManagementSite := normalizeID(siteID) == "control-plane-access"
	if isManagementSite {
		allowedMethods = []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
	}
	badBehaviorBanSeconds := envIntOrDefault("WAF_DEFAULT_BAD_BEHAVIOR_BAN_TIME_SECONDS", 300)
	badBehaviorThreshold := envIntOrDefault("WAF_DEFAULT_BAD_BEHAVIOR_THRESHOLD", 20)
	badBehaviorPeriodSeconds := envIntOrDefault("WAF_DEFAULT_BAD_BEHAVIOR_COUNT_TIME_SECONDS", 30)
	limitConnHTTP1 := envIntOrDefault("WAF_DEFAULT_LIMIT_CONN_MAX_HTTP1", 80)
	limitConnHTTP2 := envIntOrDefault("WAF_DEFAULT_LIMIT_CONN_MAX_HTTP2", 160)
	limitConnHTTP3 := envIntOrDefault("WAF_DEFAULT_LIMIT_CONN_MAX_HTTP3", 160)
	limitReqRate := envStringOrDefault("WAF_DEFAULT_LIMIT_REQ_RATE", "30r/s")
	badBehaviorStatusCodes := []int{400, 401, 403, 404, 405, 429, 444}

	if isManagementSite {
		if limitConnHTTP1 < 200 {
			limitConnHTTP1 = 200
		}
		if limitConnHTTP2 < 400 {
			limitConnHTTP2 = 400
		}
		if limitConnHTTP3 < 400 {
			limitConnHTTP3 = 400
		}
		if rate := normalizeLimitReqRate(limitReqRate); parseRatePerSecond(rate) < 80 {
			limitReqRate = "80r/s"
		} else {
			limitReqRate = rate
		}
		// Management UI can legitimately burst after login and during page bootstrap.
		// Keep 429 responses available, but do not turn them into sticky 403 escalation.
		badBehaviorStatusCodes = []int{400, 401, 403, 404, 405, 444}
	}

	return EasySiteProfile{
		SiteID: siteID,
		FrontService: FrontServiceSettings{
			ServerName:                 siteID,
			SecurityMode:               SecurityModeBlock,
			AutoLetsEncrypt:            true,
			UseLetsEncryptStaging:      false,
			UseLetsEncryptWildcard:     false,
			CertificateAuthorityServer: "letsencrypt",
		},
		UpstreamRouting: UpstreamRoutingSettings{
			UseReverseProxy:        true,
			ReverseProxyHost:       "http://upstream-server:8080",
			ReverseProxyURL:        "/",
			ReverseProxyCustomHost: "",
			ReverseProxySSLSNI:     false,
			ReverseProxySSLSNIName: "",
			ReverseProxyWebsocket:  true,
			ReverseProxyKeepalive:  true,
		},
		HTTPBehavior: HTTPBehaviorSettings{
			AllowedMethods: allowedMethods,
			MaxClientSize:  "100m",
			HTTP2:          true,
			HTTP3:          false,
			SSLProtocols:   []string{"TLSv1.2", "TLSv1.3"},
		},
		HTTPHeaders: HTTPHeadersSettings{
			CookieFlags:           "* SameSite=Lax",
			ContentSecurityPolicy: "",
			PermissionsPolicy:     []string{},
			KeepUpstreamHeaders:   []string{"*"},
			ReferrerPolicy:        "no-referrer-when-downgrade",
			UseCORS:               false,
			CORSAllowedOrigins:    []string{"*"},
		},
		SecurityBehaviorAndLimits: SecurityBehaviorAndLimitsSettings{
			UseBadBehavior:              true,
			BadBehaviorStatusCodes:      badBehaviorStatusCodes,
			BadBehaviorBanTimeSeconds:   badBehaviorBanSeconds,
			BadBehaviorThreshold:        badBehaviorThreshold,
			BadBehaviorCountTimeSeconds: badBehaviorPeriodSeconds,
			UseBlacklist:                false,
			UseDNSBL:                    false,
			BlacklistIP:                 []string{},
			BlacklistRDNS:               []string{},
			BlacklistASN:                []string{},
			BlacklistUserAgent:          []string{},
			BlacklistURI:                []string{},
			BlacklistIPURLs:             []string{},
			BlacklistRDNSURLs:           []string{},
			BlacklistASNURLs:            []string{},
			BlacklistUserAgentURLs:      []string{},
			BlacklistURIURLs:            []string{},
			UseLimitConn:                true,
			LimitConnMaxHTTP1:           limitConnHTTP1,
			LimitConnMaxHTTP2:           limitConnHTTP2,
			LimitConnMaxHTTP3:           limitConnHTTP3,
			UseLimitReq:                 true,
			LimitReqURL:                 "/",
			LimitReqRate:                limitReqRate,
		},
		SecurityAntibot: SecurityAntibotSettings{
			AntibotChallenge:        AntibotChallengeNo,
			AntibotURI:              "/challenge",
			AntibotRecaptchaScore:   0.7,
			AntibotRecaptchaSitekey: "",
			AntibotRecaptchaSecret:  "",
			AntibotHcaptchaSitekey:  "",
			AntibotHcaptchaSecret:   "",
			AntibotTurnstileSitekey: "",
			AntibotTurnstileSecret:  "",
		},
		SecurityAuthBasic: SecurityAuthBasicSettings{
			UseAuthBasic:      false,
			AuthBasicLocation: AuthBasicLocationSitewide,
			AuthBasicUser:     "changeme",
			AuthBasicPassword: "",
			AuthBasicText:     "Restricted area",
		},
		SecurityCountryPolicy: SecurityCountryPolicySettings{
			BlacklistCountry: []string{},
			WhitelistCountry: []string{},
		},
		SecurityModSecurity: SecurityModSecuritySettings{
			UseModSecurity:           true,
			UseModSecurityCRSPlugins: false,
			ModSecurityCRSVersion:    "4",
			ModSecurityCRSPlugins:    []string{},
			CustomConfiguration: ModSecurityCustomConfiguration{
				Path:    "modsec/anomaly_score.conf",
				Content: "",
			},
		},
	}
}

func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envStringOrDefault(key, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	return raw
}

func parseRatePerSecond(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimSuffix(value, "r/s")
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func allowedBadBehaviorStatusCodes() []int {
	return []int{
		400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415,
		416, 417, 418, 421, 422, 423, 424, 425, 426, 428, 429, 431, 444, 451, 500, 501,
		502, 503, 504, 505, 507, 508, 510, 511, 520, 521, 522, 523, 524, 525, 526,
	}
}

func isAllowedBadBehaviorStatusCode(code int) bool {
	return slices.Contains(allowedBadBehaviorStatusCodes(), code)
}
