package main

import (
	"regexp"
	"sync"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

var tarinioAdminExactPaths = []string{
	"/",
	"/login",
	"/login/2fa",
	"/challenge",
	"/challenge/verify",
}

var tarinioAdminPrefixPaths = []string{
	"/static/",
	"/api/app/",
	"/api/auth/",
	"/api/dashboard/",
	"/api/reports/",
	"/api/sites",
	"/api/upstreams",
	"/api/certificates",
	"/api/tls-configs",
	"/api/easy-site-profiles",
	"/api/access-policies",
	"/api/requests",
	"/api/revisions",
	"/api/events",
	"/api/bans",
	"/api/jobs",
	"/api/settings",
	"/api/administration",
}

var tarinioAdminSegmentPrefixes = []string{
	"/dashboard",
	"/sites",
	"/services",
	"/anti-ddos",
	"/tls",
	"/requests",
	"/revisions",
	"/events",
	"/bans",
	"/jobs",
	"/administration",
	"/activity",
	"/settings",
	"/about",
	"/profile",
	"/healthcheck",
	"/onboarding",
}

type securityEvent struct {
	Type            string         `json:"type"`
	Severity        string         `json:"severity"`
	SiteID          string         `json:"site_id,omitempty"`
	SourceComponent string         `json:"source_component"`
	OccurredAt      string         `json:"occurred_at"`
	Summary         string         `json:"summary"`
	Details         map[string]any `json:"details,omitempty"`
}

type securityEventSource struct {
	mu          sync.Mutex
	path        string
	offset      int64
	threatIntel *threatIntelMatcher
}

const (
	defaultBurstRequestsPerSecondThreshold     = 25
	defaultBurstPathRequestsPerSecondThreshold = 10
)
