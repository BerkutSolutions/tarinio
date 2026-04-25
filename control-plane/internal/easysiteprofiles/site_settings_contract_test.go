package easysiteprofiles

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSiteSettings_FieldContract(t *testing.T) {
	t.Parallel()

	got := collectJSONFieldPaths(reflect.TypeOf(EasySiteProfile{}), "")
	want := []string{
		"created_at",
		"front_service.acme_account_email",
		"front_service.adaptive_model_enabled",
		"front_service.auto_lets_encrypt",
		"front_service.certificate_authority_server",
		"front_service.profile",
		"front_service.security_mode",
		"front_service.server_name",
		"front_service.use_lets_encrypt_staging",
		"front_service.use_lets_encrypt_wildcard",
		"http_behavior.allowed_methods",
		"http_behavior.http2",
		"http_behavior.http3",
		"http_behavior.max_client_size",
		"http_behavior.ssl_protocols",
		"http_headers.content_security_policy",
		"http_headers.cookie_flags",
		"http_headers.cors_allowed_origins",
		"http_headers.keep_upstream_headers",
		"http_headers.permissions_policy",
		"http_headers.referrer_policy",
		"http_headers.use_cors",
		"security_antibot.antibot_challenge",
		"security_antibot.antibot_hcaptcha_secret",
		"security_antibot.antibot_hcaptcha_sitekey",
		"security_antibot.antibot_recaptcha_score",
		"security_antibot.antibot_recaptcha_secret",
		"security_antibot.antibot_recaptcha_sitekey",
		"security_antibot.scanner_auto_ban_enabled",
		"security_antibot.antibot_turnstile_secret",
		"security_antibot.antibot_turnstile_sitekey",
		"security_antibot.antibot_uri",
		"security_antibot.challenge_escalation_enabled",
		"security_antibot.challenge_escalation_mode",
		"security_antibot.challenge_rules.challenge",
		"security_antibot.challenge_rules.path",
		"security_auth_basic.auth_basic_location",
		"security_auth_basic.auth_basic_password",
		"security_auth_basic.auth_basic_text",
		"security_auth_basic.auth_basic_user",
		"security_auth_basic.session_inactivity_minutes",
		"security_auth_basic.use_auth_basic",
		"security_auth_basic.users.enabled",
		"security_auth_basic.users.last_login_at",
		"security_auth_basic.users.password",
		"security_auth_basic.users.username",
		"security_behavior_and_limits.bad_behavior_ban_time_seconds",
		"security_behavior_and_limits.bad_behavior_count_time_seconds",
		"security_behavior_and_limits.bad_behavior_status_codes",
		"security_behavior_and_limits.bad_behavior_threshold",
		"security_behavior_and_limits.ban_escalation_enabled",
		"security_behavior_and_limits.ban_escalation_scope",
		"security_behavior_and_limits.ban_escalation_stages_seconds",
		"security_behavior_and_limits.exceptions_ip",
		"security_behavior_and_limits.blacklist_asn",
		"security_behavior_and_limits.blacklist_asn_urls",
		"security_behavior_and_limits.blacklist_ip",
		"security_behavior_and_limits.blacklist_ip_urls",
		"security_behavior_and_limits.blacklist_rdns",
		"security_behavior_and_limits.blacklist_rdns_urls",
		"security_behavior_and_limits.blacklist_uri",
		"security_behavior_and_limits.blacklist_uri_urls",
		"security_behavior_and_limits.blacklist_user_agent",
		"security_behavior_and_limits.blacklist_user_agent_urls",
		"security_behavior_and_limits.limit_conn_max_http1",
		"security_behavior_and_limits.limit_conn_max_http2",
		"security_behavior_and_limits.limit_conn_max_http3",
		"security_behavior_and_limits.custom_limit_rules.path",
		"security_behavior_and_limits.custom_limit_rules.rate",
		"security_behavior_and_limits.limit_req_rate",
		"security_behavior_and_limits.limit_req_url",
		"security_behavior_and_limits.use_bad_behavior",
		"security_behavior_and_limits.use_blacklist",
		"security_behavior_and_limits.use_exceptions",
		"security_behavior_and_limits.use_dnsbl",
		"security_behavior_and_limits.use_limit_conn",
		"security_behavior_and_limits.use_limit_req",
		"security_country_policy.blacklist_country",
		"security_country_policy.whitelist_country",
		"security_api_positive.default_action",
		"security_api_positive.enforcement_mode",
		"security_api_positive.endpoint_policies.content_types",
		"security_api_positive.endpoint_policies.methods",
		"security_api_positive.endpoint_policies.mode",
		"security_api_positive.endpoint_policies.path",
		"security_api_positive.endpoint_policies.token_ids",
		"security_api_positive.openapi_schema_ref",
		"security_api_positive.use_api_positive_security",
		"security_modsecurity.custom_configuration.content",
		"security_modsecurity.custom_configuration.path",
		"security_modsecurity.modsecurity_crs_plugins",
		"security_modsecurity.modsecurity_crs_version",
		"security_modsecurity.use_modsecurity_custom_configuration",
		"security_modsecurity.use_modsecurity",
		"security_modsecurity.use_modsecurity_crs_plugins",
		"site_id",
		"updated_at",
		"upstream_routing.reverse_proxy_custom_host",
		"upstream_routing.reverse_proxy_host",
		"upstream_routing.reverse_proxy_keepalive",
		"upstream_routing.disable_host_header",
		"upstream_routing.disable_x_forwarded_for",
		"upstream_routing.disable_x_forwarded_proto",
		"upstream_routing.enable_x_real_ip",
		"upstream_routing.reverse_proxy_ssl_sni",
		"upstream_routing.reverse_proxy_ssl_sni_name",
		"upstream_routing.reverse_proxy_url",
		"upstream_routing.reverse_proxy_websocket",
		"upstream_routing.use_reverse_proxy",
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("site settings field contract changed\nwant: %s\ngot:  %s", strings.Join(want, ", "), strings.Join(got, ", "))
	}
}

func TestSiteSettings_CreateAndSave_AllFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	profile := allFieldsProfile("site-contract")
	expectedCreate := normalizeProfile(profile)

	created, err := store.Create(profile)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("timestamps must be set: %+v", created)
	}
	expectedCreate.CreatedAt = created.CreatedAt
	expectedCreate.UpdatedAt = created.UpdatedAt
	if !reflect.DeepEqual(created, expectedCreate) {
		t.Fatalf("create mismatch\nwant: %+v\ngot:  %+v", expectedCreate, created)
	}

	got, ok, err := store.Get("site-contract")
	if err != nil {
		t.Fatalf("get after create: %v", err)
	}
	if !ok {
		t.Fatal("expected created profile")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("get mismatch\nwant: %+v\ngot:  %+v", created, got)
	}

	updatedProfile := allFieldsProfile("site-contract")
	updatedProfile.FrontService.ServerName = "api.changed.example.com"
	updatedProfile.FrontService.SecurityMode = SecurityModeMonitor
	updatedProfile.FrontService.Profile = ServiceProfileAPI
	updatedProfile.UpstreamRouting.ReverseProxyHost = "https://backend.changed.internal:8443"
	updatedProfile.HTTPBehavior.AllowedMethods = []string{"DELETE", "PATCH", "GET"}
	updatedProfile.SecurityBehaviorAndLimits.LimitReqRate = "21r/s"
	updatedProfile.SecurityBehaviorAndLimits.CustomLimitRules = []CustomLimitRule{{Path: "/login", Rate: "5r/s"}, {Path: "/api/", Rate: "20r/s"}}
	updatedProfile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 4
	updatedProfile.SecurityAntibot.AntibotChallenge = AntibotChallengeTurnstile
	updatedProfile.SecurityAntibot.AntibotTurnstileSitekey = "turnstile-key-2"
	updatedProfile.SecurityAntibot.AntibotTurnstileSecret = "turnstile-secret-2"
	updatedProfile.SecurityAuthBasic.AuthBasicPassword = "updated-password"
	updatedProfile.SecurityCountryPolicy.BlacklistCountry = []string{"DE", "EU"}
	updatedProfile.SecurityCountryPolicy.WhitelistCountry = []string{"GB", "NA"}
	updatedProfile.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-z", "plugin-a"}
	updatedProfile.SecurityModSecurity.UseCustomConfiguration = true
	updatedProfile.SecurityModSecurity.CustomConfiguration.Content = "SecAction \"id:1003,phase:1,pass\""

	expectedUpdate := normalizeProfile(updatedProfile)
	updated, err := store.Update(updatedProfile)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.CreatedAt != created.CreatedAt {
		t.Fatalf("created_at must stay stable: create=%s update=%s", created.CreatedAt, updated.CreatedAt)
	}
	createdAt, err := time.Parse(time.RFC3339, created.CreatedAt)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, updated.UpdatedAt)
	if err != nil {
		t.Fatalf("parse updated_at: %v", err)
	}
	if updatedAt.Before(createdAt) {
		t.Fatalf("updated_at must not be earlier than created_at: create=%s update=%s", created.CreatedAt, updated.UpdatedAt)
	}
	expectedUpdate.CreatedAt = created.CreatedAt
	expectedUpdate.UpdatedAt = updated.UpdatedAt
	if !reflect.DeepEqual(updated, expectedUpdate) {
		t.Fatalf("update mismatch\nwant: %+v\ngot:  %+v", expectedUpdate, updated)
	}
}

func allFieldsProfile(siteID string) EasySiteProfile {
	profile := DefaultProfile(siteID)
	profile.FrontService.ServerName = "api.example.com"
	profile.FrontService.SecurityMode = SecurityModeBlock
	profile.FrontService.Profile = ServiceProfileStrict
	profile.FrontService.AutoLetsEncrypt = false
	profile.FrontService.UseLetsEncryptStaging = true
	profile.FrontService.UseLetsEncryptWildcard = true
	profile.FrontService.CertificateAuthorityServer = "zerossl"
	profile.FrontService.ACMEAccountEmail = "acme@example.test"

	profile.UpstreamRouting.UseReverseProxy = true
	profile.UpstreamRouting.ReverseProxyHost = "http://backend.internal:8080"
	profile.UpstreamRouting.ReverseProxyURL = "/gateway"
	profile.UpstreamRouting.ReverseProxyCustomHost = "backend.internal"
	profile.UpstreamRouting.ReverseProxySSLSNI = true
	profile.UpstreamRouting.ReverseProxySSLSNIName = "backend.internal"
	profile.UpstreamRouting.ReverseProxyWebsocket = false
	profile.UpstreamRouting.ReverseProxyKeepalive = false
	profile.UpstreamRouting.DisableHostHeader = false
	profile.UpstreamRouting.DisableXForwardedFor = true
	profile.UpstreamRouting.DisableXForwardedProto = true
	profile.UpstreamRouting.EnableXRealIP = true

	profile.HTTPBehavior.AllowedMethods = []string{"POST", "GET", "PUT"}
	profile.HTTPBehavior.MaxClientSize = "64m"
	profile.HTTPBehavior.HTTP2 = false
	profile.HTTPBehavior.HTTP3 = true
	profile.HTTPBehavior.SSLProtocols = []string{"TLSv1.3", "TLSv1.2"}

	profile.HTTPHeaders.CookieFlags = "* SameSite=Strict"
	profile.HTTPHeaders.ContentSecurityPolicy = "default-src 'self'"
	profile.HTTPHeaders.PermissionsPolicy = []string{"camera=()", "microphone=()"}
	profile.HTTPHeaders.KeepUpstreamHeaders = []string{"X-Trace-Id", "X-Request-Id"}
	profile.HTTPHeaders.ReferrerPolicy = "strict-origin"
	profile.HTTPHeaders.UseCORS = true
	profile.HTTPHeaders.CORSAllowedOrigins = []string{"https://app.example.com", "https://admin.example.com"}

	profile.SecurityBehaviorAndLimits.UseBadBehavior = true
	profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{401, 403, 429}
	profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = 31
	profile.SecurityBehaviorAndLimits.BadBehaviorThreshold = 3
	profile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds = 11
	profile.SecurityBehaviorAndLimits.UseBlacklist = true
	profile.SecurityBehaviorAndLimits.UseExceptions = true
	profile.SecurityBehaviorAndLimits.ExceptionsIP = []string{"198.51.100.11", "203.0.113.11"}
	profile.SecurityBehaviorAndLimits.UseDNSBL = true
	profile.SecurityBehaviorAndLimits.BlacklistIP = []string{"198.51.100.10", "203.0.113.10"}
	profile.SecurityBehaviorAndLimits.BlacklistRDNS = []string{".scanner.example", ".bad.example"}
	profile.SecurityBehaviorAndLimits.BlacklistASN = []string{"AS64496", "AS64497"}
	profile.SecurityBehaviorAndLimits.BlacklistUserAgent = []string{"curl/*", "python-requests/*"}
	profile.SecurityBehaviorAndLimits.BlacklistURI = []string{"/private", "/internal"}
	profile.SecurityBehaviorAndLimits.BlacklistIPURLs = []string{"https://feeds.example/ip.txt"}
	profile.SecurityBehaviorAndLimits.BlacklistRDNSURLs = []string{"https://feeds.example/rdns.txt"}
	profile.SecurityBehaviorAndLimits.BlacklistASNURLs = []string{"https://feeds.example/asn.txt"}
	profile.SecurityBehaviorAndLimits.BlacklistUserAgentURLs = []string{"https://feeds.example/ua.txt"}
	profile.SecurityBehaviorAndLimits.BlacklistURIURLs = []string{"https://feeds.example/uri.txt"}
	profile.SecurityBehaviorAndLimits.UseLimitConn = true
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = 11
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = 12
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = 13
	profile.SecurityBehaviorAndLimits.UseLimitReq = true
	profile.SecurityBehaviorAndLimits.LimitReqURL = "/gateway"
	profile.SecurityBehaviorAndLimits.LimitReqRate = "19r/s"
	profile.SecurityBehaviorAndLimits.CustomLimitRules = []CustomLimitRule{{Path: "/login", Rate: "6r/s"}, {Path: "/api/auth/", Rate: "10r/s"}}

	profile.SecurityAntibot.AntibotChallenge = AntibotChallengeRecaptcha
	profile.SecurityAntibot.AntibotURI = "/challenge"
	profile.SecurityAntibot.ScannerAutoBanEnabled = true
	profile.SecurityAntibot.AntibotRecaptchaScore = 0.8
	profile.SecurityAntibot.AntibotRecaptchaSitekey = "recaptcha-site"
	profile.SecurityAntibot.AntibotRecaptchaSecret = "recaptcha-secret"
	profile.SecurityAntibot.AntibotHcaptchaSitekey = "hcaptcha-site"
	profile.SecurityAntibot.AntibotHcaptchaSecret = "hcaptcha-secret"
	profile.SecurityAntibot.AntibotTurnstileSitekey = "turnstile-site"
	profile.SecurityAntibot.AntibotTurnstileSecret = "turnstile-secret"
	profile.SecurityAntibot.ChallengeEscalationEnabled = true
	profile.SecurityAntibot.ChallengeEscalationMode = AntibotChallengeTurnstile
	profile.SecurityAntibot.ChallengeRules = []AntibotChallengeRule{
		{Path: "/login", Challenge: AntibotChallengeRecaptcha},
		{Path: "/api/auth/", Challenge: AntibotChallengeCookie},
	}

	profile.SecurityAuthBasic.UseAuthBasic = true
	profile.SecurityAuthBasic.AuthBasicLocation = AuthBasicLocationSitewide
	profile.SecurityAuthBasic.AuthBasicUser = "admin"
	profile.SecurityAuthBasic.AuthBasicPassword = "super-secret"
	profile.SecurityAuthBasic.AuthBasicText = "Private zone"

	profile.SecurityCountryPolicy.BlacklistCountry = []string{"RU", "APAC"}
	profile.SecurityCountryPolicy.WhitelistCountry = []string{"US", "EMEA"}

	profile.SecurityAPIPositive.UseAPIPositiveSecurity = true
	profile.SecurityAPIPositive.OpenAPISchemaRef = "openapi/petstore.yaml"
	profile.SecurityAPIPositive.EnforcementMode = APIPositiveEnforcementBlock
	profile.SecurityAPIPositive.DefaultAction = APIPositiveDefaultActionDeny
	profile.SecurityAPIPositive.EndpointPolicies = []APIPositiveEndpointPolicy{
		{
			Path:         "/api/v1/orders",
			Methods:      []string{"GET", "POST"},
			TokenIDs:     []string{"svc-orders", "svc-admin"},
			ContentTypes: []string{"application/json"},
			Mode:         APIPositiveEnforcementBlock,
		},
		{
			Path:         "/api/v1/public",
			Methods:      []string{"GET"},
			TokenIDs:     []string{},
			ContentTypes: []string{},
			Mode:         APIPositiveEnforcementMonitor,
		},
	}

	profile.SecurityModSecurity.UseModSecurity = true
	profile.SecurityModSecurity.UseModSecurityCRSPlugins = true
	profile.SecurityModSecurity.UseCustomConfiguration = true
	profile.SecurityModSecurity.ModSecurityCRSVersion = "4.1"
	profile.SecurityModSecurity.ModSecurityCRSPlugins = []string{"plugin-z", "plugin-a"}
	profile.SecurityModSecurity.CustomConfiguration.Path = "modsec/custom-rules.conf"
	profile.SecurityModSecurity.CustomConfiguration.Content = "SecAction \"id:1001,phase:1,pass\""
	return profile
}

func collectJSONFieldPaths(t reflect.Type, prefix string) []string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var paths []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			paths = append(paths, collectJSONFieldPaths(fieldType, path)...)
			continue
		}
		if fieldType.Kind() == reflect.Slice {
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Pointer {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				paths = append(paths, collectJSONFieldPaths(elemType, path)...)
				continue
			}
		}
		paths = append(paths, path)
	}
	return paths
}
