package compiler

import "testing"

func TestEasyDirectiveInputsRejectNginxDirectiveInjection(t *testing.T) {
	for _, profile := range []EasyProfileInput{
		{SiteID: "site-a", LimitReqURL: "/login\nproxy_pass http://evil"},
		{SiteID: "site-a", BlacklistURI: []string{`/safe"; return 200; #`}},
		{SiteID: "site-a", AuthExclusionRules: []AuthExclusionRuleInput{{Path: "/safe\ninclude /tmp/evil"}}},
	} {
		if err := validateEasyDirectiveInputs(profile); err == nil {
			t.Fatalf("expected unsafe profile %#v to be rejected", profile)
		}
	}
	if err := validateEasyDirectiveInputs(EasyProfileInput{SiteID: "site-a", LimitReqURL: "/login", BlacklistURI: []string{"/admin"}}); err != nil {
		t.Fatalf("expected safe paths: %v", err)
	}
}
