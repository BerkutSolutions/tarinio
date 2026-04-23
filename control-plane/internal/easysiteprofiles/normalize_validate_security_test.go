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
