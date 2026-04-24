package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_APIPositiveSecurityRules(t *testing.T) {
	site := SiteInput{
		ID:                "site-a",
		Enabled:           true,
		PrimaryHost:       "app.example.com",
		ListenHTTP:        true,
		ListenHTTPS:       true,
		DefaultUpstreamID: "upstream-a",
	}
	profile := EasyProfileInput{
		SiteID:                 "site-a",
		SecurityMode:           "block",
		AllowedMethods:         []string{"GET", "POST"},
		MaxClientSize:          "16m",
		UseModSecurity:         true,
		UseAPIPositiveSecurity: true,
		OpenAPISchemaRef:       "openapi/site-a.yaml",
		APIEnforcementMode:     "block",
		APIDefaultAction:       "deny",
		APIEndpointPolicies: []APIPositiveEndpointPolicyInput{
			{
				Path:         "/api/v1/orders",
				Methods:      []string{"GET", "POST"},
				TokenIDs:     []string{"svc-orders"},
				ContentTypes: []string{"application/json"},
				Mode:         "block",
			},
		},
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}
	modsecEasy := byPath["modsecurity/easy/site-a.conf"]
	if modsecEasy == "" {
		t.Fatal("expected modsecurity easy artifact for site-a")
	}
	if !strings.Contains(modsecEasy, "# API Positive Security directives") {
		t.Fatalf("expected API positive rules section, got: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "X-WAF-API-TOKEN-ID") {
		t.Fatalf("expected token policy rule, got: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "unknown endpoint") {
		t.Fatalf("expected default deny unknown endpoint rule, got: %s", modsecEasy)
	}
}
