package easysiteprofiles

import "testing"

func TestValidateProfileRejectsUnsafeHeaderInjection(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.HTTPHeaders.ContentSecurityPolicy = "default-src \"self\""
	if err := validateProfile(profile); err == nil {
		t.Fatalf("expected validation error for unsafe header value")
	}
}

func TestValidateProfileRejectsUnsafeReverseProxyCustomHost(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.UpstreamRouting.ReverseProxyCustomHost = "backend.example.com;proxy_pass http://evil"
	if err := validateProfile(profile); err == nil {
		t.Fatalf("expected validation error for reverse proxy custom host")
	}
}

func TestValidateProfileRejectsHSTSPreloadWithoutIncludeSubdomains(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.HTTPHeaders.HSTSEnabled = true
	profile.HTTPHeaders.HSTSIncludeSubdomains = false
	profile.HTTPHeaders.HSTSPreload = true
	profile.HTTPHeaders.HSTSMaxAgeSeconds = 31536000
	if err := validateProfile(profile); err == nil {
		t.Fatalf("expected validation error for preload without includeSubDomains")
	}
}

func TestValidateProfileRejectsHSTSPreloadWithSmallMaxAge(t *testing.T) {
	profile := DefaultProfile("site-a")
	profile.HTTPHeaders.HSTSEnabled = true
	profile.HTTPHeaders.HSTSIncludeSubdomains = true
	profile.HTTPHeaders.HSTSPreload = true
	profile.HTTPHeaders.HSTSMaxAgeSeconds = 10800
	if err := validateProfile(profile); err == nil {
		t.Fatalf("expected validation error for preload max-age")
	}
}
