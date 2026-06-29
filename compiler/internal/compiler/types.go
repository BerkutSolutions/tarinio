package compiler

// SiteInput is the MVP compiler input slice for host-based routing and listeners.
type SiteInput struct {
	ID                string
	Name              string
	Enabled           bool
	PrimaryHost       string
	Aliases           []string
	ListenHTTP          bool
	ListenHTTPS         bool
	DefaultUpstreamID   string
	UseEasyConfig       bool
	UseCustomErrorPages bool
}

// UpstreamInput is the MVP compiler input slice for upstream mapping.
type UpstreamInput struct {
	ID             string
	SiteID         string
	Name           string
	Scheme         string
	Host           string
	Port           int
	BasePath       string
	PassHostHeader bool
}

// ArtifactKind matches the accepted manifest contents kinds.
type ArtifactKind string

const (
	ArtifactKindNginxConfig ArtifactKind = "nginx_config"
	ArtifactKindModSecurity ArtifactKind = "modsecurity_config"
	ArtifactKindCRSConfig   ArtifactKind = "crs_config"
	ArtifactKindTLSRef      ArtifactKind = "tls_ref"
)

// TLSConfigInput is the MVP compiler input slice for TLS site wiring.
type TLSConfigInput struct {
	ID                  string
	SiteID              string
	CertificateID       string
	RedirectHTTPToHTTPS bool
}

// CertificateInput is the MVP compiler input slice for certificate references.
type CertificateInput struct {
	ID            string
	SiteID        string
	StorageRef    string
	PrivateKeyRef string
}

// WAFMode is the accepted MVP WAF execution mode.
type WAFMode string

const (
	WAFModeDetection  WAFMode = "detection"
	WAFModePrevention WAFMode = "prevention"
)

// WAFPolicyInput is the MVP compiler input slice for ModSecurity and CRS wiring.
type WAFPolicyInput struct {
	ID                 string
	SiteID             string
	Enabled            bool
	Mode               WAFMode
	CRSEnabled         bool
	CustomRuleIncludes []string
}

// AccessPolicyInput is the MVP compiler input slice for site-level IP allow/deny rules.
type AccessPolicyInput struct {
	ID                string
	SiteID            string
	DefaultAction     string
	AllowCIDRs        []string
	DenyCIDRs         []string
	TrustedProxyCIDRs []string
}

// RateLimitPolicyInput is the MVP compiler input slice for basic nginx request
// and connection limiting.
type RateLimitPolicyInput struct {
	ID            string
	SiteID        string
	Enabled       bool
	Requests      int
	WindowSeconds int
	Burst         int
	StatusCode    int
}

type CustomRateLimitRuleInput struct {
	Path string
	Rate string
}

type APIPositiveEndpointPolicyInput struct {
	Path         string
	Methods      []string
	TokenIDs     []string
	ContentTypes []string
	Mode         string
}

type ServiceAuthUserInput struct {
	Username    string
	Password    string
	Enabled     bool
	LastLoginAt string
}

type ServiceAuthTokenInput struct {
	ServiceName string
	Token       string
	Enabled     bool
	LastUsedAt  string
}

type AntibotChallengeRuleInput struct {
	Path      string
	Challenge string
}

type AntibotExclusionRuleInput struct {
	Path    string
	Methods []string
}

type AuthExclusionRuleInput struct {
	Path    string
	Methods []string
}

// ArtifactOutput is a rendered compiler artifact ready to be placed into a bundle.
type ArtifactOutput struct {
	Path     string
	Kind     ArtifactKind
	Content  []byte
	Checksum string
}

// EasyProfileInput is the compiler input slice for Easy-mode site settings.
type EasyProfileInput struct {
	SiteID       string
	SecurityMode string

	AllowedMethods []string
	MaxClientSize  string

	ReferrerPolicy        string
	ContentSecurityPolicy string
	PermissionsPolicy     []string
	HSTSEnabled           bool
	HSTSMaxAgeSeconds     int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
	UseCORS               bool
	CORSAllowedOrigins    []string

	ReverseProxyCustomHost string
	ReverseProxySSLSNI     bool
	ReverseProxySSLSNIName string
	ReverseProxyWebsocket  bool
	ReverseProxyKeepalive  bool
	PassHostHeader         bool
	SendXForwardedFor      bool
	SendXForwardedProto    bool
	SendXRealIP            bool

	UseAuthBasic      bool
	AuthMode          string
	AuthOrder         string
	AuthBasicUser     string
	AuthBasicPassword string
	AuthBasicText     string
	AuthUsers         []ServiceAuthUserInput
	AuthServiceTokens []ServiceAuthTokenInput
	AuthExclusionRules []AuthExclusionRuleInput
	AuthSessionTTLMin int

	AntibotChallenge           string
	AntibotChallengeTemplate   string
	AntibotURI                 string
	AntibotScannerAutoBan      bool
	AntibotRecaptchaScore      float64
	AntibotRecaptchaKey        string
	AntibotHcaptchaKey         string
	AntibotTurnstileKey        string
	AntibotCookieName          string
	AntibotCookieValue         string
	AntibotExclusionRules      []AntibotExclusionRuleInput
	ChallengeEscalationEnabled bool
	ChallengeEscalationMode    string
	AntibotChallengeRules      []AntibotChallengeRuleInput

	UseLimitConn              bool
	LimitConnMaxHTTP1         int
	LimitConnMaxHTTP2         int
	LimitConnMaxHTTP3         int
	UseLimitReq               bool
	LimitReqRate              string
	CustomLimitRules          []CustomRateLimitRuleInput
	UseBadBehavior            bool
	BadBehaviorStatusCodes    []int
	BadBehaviorBanTimeSeconds int

	BlacklistIP        []string
	BlacklistUserAgent []string
	BlacklistURI       []string
	BlacklistJA3       []string
	BlacklistJA3URLs   []string

	ExceptionsURI []string

	BlacklistCountry    []string
	WhitelistCountry    []string
	ShowGeoBlockPage    bool
	// GeoTimeWindows defines time-based geo-fencing rules compiled into nginx.
	GeoTimeWindows []GeoTimeWindowInput

	UseModSecurity                    bool
	UseModSecurityCRSPlugins          bool
	UseModSecurityCustomConfiguration bool
	ModSecurityCRSVersion             string
	ModSecurityCRSPlugins             []string
	ModSecurityCustomPath             string
	ModSecurityCustomContent          string

	CookieFlags         string
	KeepUpstreamHeaders []string

	UseAPIPositiveSecurity bool
	OpenAPISchemaRef       string
	APIEnforcementMode     string
	APIDefaultAction       string
	APIEndpointPolicies    []APIPositiveEndpointPolicyInput

	HttpStrictParsing bool

	// HealthCheckEnabled enables passive upstream health checking.
	// When true, nginx uses proxy_next_upstream to retry on 502/503.
	HealthCheckEnabled         bool
	HealthCheckPath            string
	HealthCheckIntervalSeconds int
	HealthCheckFailThreshold   int

	// WSInspection holds WebSocket frame inspection settings.
	// Active only when UseWSInspection=true and ReverseProxyWebsocket=true.
	WSInspection WSInspectionInput

	// MTLS holds incoming mTLS client certificate verification settings (TASK-8.1).
	MTLS MTLSInput

	// UpstreamMTLS holds outgoing mTLS client certificate settings (TASK-8.2).
	UpstreamMTLS UpstreamMTLSInput

	VirtualPatches []VirtualPatchInput

	// UseCustomErrorPages enables WAF custom error pages for this site.
	// When true, proxy_intercept_errors is on and error_page directives are wired.
	// When false, upstream error responses pass through unmodified.
	UseCustomErrorPages bool
	// DisabledErrorPages lists slugs (e.g. "404", "500") that are individually disabled.
	// When a slug is in this list, its error_page directive is omitted from the config.
	DisabledErrorPages []string
}

// VirtualPatchInput is the compiler input slice for a single virtual patch rule.
type VirtualPatchInput struct {
	ID      string
	Pattern string // regex
	Target  string // "uri" | "body" | "header"
	Action  string // "block" | "monitor"
}
