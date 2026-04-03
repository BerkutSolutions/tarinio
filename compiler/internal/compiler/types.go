package compiler

// SiteInput is the MVP compiler input slice for host-based routing and listeners.
type SiteInput struct {
	ID                string
	Name              string
	Enabled           bool
	PrimaryHost       string
	Aliases           []string
	ListenHTTP        bool
	ListenHTTPS       bool
	DefaultUpstreamID string
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

// ArtifactOutput is a rendered compiler artifact ready to be placed into a bundle.
type ArtifactOutput struct {
	Path     string
	Kind     ArtifactKind
	Content  []byte
	Checksum string
}

// EasyProfileInput is the compiler input slice for Easy-mode site settings.
type EasyProfileInput struct {
	SiteID string

	AllowedMethods []string
	MaxClientSize  string

	ReferrerPolicy        string
	ContentSecurityPolicy string
	PermissionsPolicy     []string
	UseCORS               bool
	CORSAllowedOrigins    []string

	ReverseProxyCustomHost string
	ReverseProxySSLSNI     bool
	ReverseProxySSLSNIName string
	ReverseProxyWebsocket  bool
	ReverseProxyKeepalive  bool

	UseAuthBasic      bool
	AuthBasicUser     string
	AuthBasicPassword string
	AuthBasicText     string

	AntibotChallenge      string
	AntibotURI            string
	AntibotRecaptchaScore float64
	AntibotRecaptchaKey   string
	AntibotHcaptchaKey    string
	AntibotTurnstileKey   string

	UseLimitConn              bool
	LimitConnMaxHTTP1         int
	LimitConnMaxHTTP2         int
	LimitConnMaxHTTP3         int
	UseLimitReq               bool
	LimitReqRate              string
	UseBadBehavior            bool
	BadBehaviorStatusCodes    []int
	BadBehaviorBanTimeSeconds int

	BlacklistIP        []string
	BlacklistUserAgent []string
	BlacklistURI       []string

	BlacklistCountry []string
	WhitelistCountry []string

	UseModSecurity           bool
	UseModSecurityCRSPlugins bool
	ModSecurityCRSVersion    string
	ModSecurityCRSPlugins    []string
	ModSecurityCustomPath    string
	ModSecurityCustomContent string
}
