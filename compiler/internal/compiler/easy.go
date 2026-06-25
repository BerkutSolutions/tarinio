package compiler

import (
	"encoding/json"
	"fmt"
	"sort"
)

type easySiteData struct {
	SiteID                       string
	RateLimitCookieVar           string
	RateLimitEscalationCookieVar string
	ExceptionVar                 string
	AllowedMethodsPattern        string
	MaxClientSize                string

	ReferrerPolicy        string
	ContentSecurityPolicy string
	PermissionsPolicy     string
	HSTSHeader            string
	UseCORS               bool
	CORSAllowedOrigins    string

	ReverseProxyCustomHost string
	ReverseProxySSLSNI     bool
	ReverseProxySSLSNIName string
	ReverseProxyWebsocket  bool
	ReverseProxyKeepalive  bool
	PassHostHeader         bool
	SendXForwardedFor      bool
	SendXForwardedProto    bool
	SendXRealIP            bool
	RateLimitBanSeconds    int
	AdminBypassPathPattern string

	UseAuthBasic      bool
	AuthBasicRealm    string
	AuthBasicUserFile string
	AuthGateLoginURI  string
	AuthGateVerifyURI string
	AuthGateCookieKey string
	AuthGateCookieVal string
	AuthGateCookieTTL int

	AntibotEnabled           bool
	AntibotTwoLayerEnabled   bool
	AntibotUsesInterstitial  bool
	AntibotChallenge         string
	AntibotEscalationMode    string
	AntibotURI               string
	AntibotVerifyURI         string
	AntibotStage1URI         string
	AntibotStage1VerifyURI   string
	AntibotRedirectURI       string
	AntibotStage1RedirectURI string
	AntibotStage1CookieName  string
	AntibotStage1CookieValue string
	AntibotCookieName        string
	AntibotCookieValue       string
	AntibotRecaptchaHint     string
	AntibotHcaptchaHint      string
	AntibotTurnstileHint     string
	AntibotExclusionRules    []easyAntibotExclusionRuleData
	AntibotRuleOverrides     []easyAntibotRuleData
	AntibotScannerAutoBan    bool
	AntibotScannerPattern    string

	BlacklistIP        []string
	BlacklistUserAgent []string
	BlacklistURI       []string

	BlacklistCountryGuardPattern string
	WhitelistCountryGuardPattern string

	UseModSecurity         bool
	UseModSecurityEasyFile bool
	ModSecurityEasyRules   string
	ModSecurityEasyRulesOn bool
}

type l4GuardConfigData struct {
	Enabled       bool   `json:"enabled"`
	ChainMode     string `json:"chain_mode"`
	ConnLimit     int    `json:"conn_limit"`
	RatePerSec    int    `json:"rate_per_second"`
	RateBurst     int    `json:"rate_burst"`
	Ports         []int  `json:"ports"`
	Target        string `json:"target"`
	DestinationIP string `json:"destination_ip"`
}

type antibotChallengePageData struct {
	VerifyURI string
}

type authGatePageData struct {
	VerifyURI string
}

type easyAntibotRuleData struct {
	GuardPattern string
	Challenge    string
	RedirectURI  string
}

type easyAntibotExclusionRuleData struct {
	MatchPattern string
}

// RenderEasyArtifacts compiles Easy-mode site directives into per-site nginx snippets.
func RenderEasyArtifacts(sites []SiteInput, profiles []EasyProfileInput) ([]ArtifactOutput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool { return sortedSites[i].ID < sortedSites[j].ID })

	profileBySite, err := buildEasyProfileBySite(profiles)
	if err != nil {
		return nil, err
	}

	artifacts := make([]ArtifactOutput, 0, len(sortedSites)*2)
	l4ConnLimit := 200
	l4RatePerSec := 100
	l4Enabled := false
	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}
		profile, ok := profileBySite[site.ID]
		if !ok {
			profile = EasyProfileInput{
				SiteID:                    site.ID,
				SecurityMode:              "block",
				AllowedMethods:            []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"},
				MaxClientSize:             "100m",
				UseModSecurity:            true,
				UseModSecurityCRSPlugins:  true,
				UseLimitConn:              true,
				LimitConnMaxHTTP1:         200,
				UseLimitReq:               true,
				LimitReqRate:              "100r/s",
				PassHostHeader:            true,
				SendXForwardedFor:         true,
				SendXForwardedProto:       true,
				SendXRealIP:               false,
				AntibotScannerAutoBan:     true,
				UseBadBehavior:            true,
				BadBehaviorStatusCodes:    []int{400, 401, 403, 404, 405, 429, 444},
				BadBehaviorBanTimeSeconds: 30,
			}
		}
		siteArtifacts, siteL4Enabled, siteConnLimit, siteRatePerSec, err := renderEasySiteArtifacts(site, profile)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, siteArtifacts...)
		if siteL4Enabled {
			l4Enabled = true
			if siteConnLimit > l4ConnLimit {
				l4ConnLimit = siteConnLimit
			}
			if siteRatePerSec > l4RatePerSec {
				l4RatePerSec = siteRatePerSec
			}
		}
	}
	if l4Enabled {
		l4 := l4GuardConfigData{
			Enabled:       true,
			ChainMode:     "auto",
			ConnLimit:     l4ConnLimit,
			RatePerSec:    l4RatePerSec,
			RateBurst:     l4RatePerSec * 2,
			Ports:         []int{80, 443},
			Target:        "DROP",
			DestinationIP: "",
		}
		raw, err := json.MarshalIndent(l4, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode l4 guard config: %w", err)
		}
		raw = append(raw, '\n')
		artifacts = append(artifacts, newArtifact(
			"l4guard/config.json",
			ArtifactKindNginxConfig,
			raw,
		))
	}

	return artifacts, nil
}
