package main

import (
	"os"
	"strconv"
	"strings"
)

func burstThresholdFromEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func shouldTrackRequestBurst(item parsedAccess) bool {
	if shouldSkipInternalManagementRequest(item) {
		return false
	}
	if shouldSkipInternalSite(item.siteID) {
		return false
	}
	if item.status == 429 || item.status == 403 || item.status == 444 {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(item.method), "OPTIONS") {
		return false
	}
	return !shouldIgnoreBurstPath(item.path)
}

func shouldSkipInternalManagementRequest(item parsedAccess) bool {
	if shouldSkipInternalSite(item.siteID) {
		return true
	}
	path := normalizeBurstPath(item.path)
	if path == "" {
		return false
	}
	if isTarinioAdminAppPath(path) {
		return true
	}
	if !isInternalManagementPath(path) {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(item.host))
	return host == "" || isInternalManagementHost(host) || sanitizeSiteID(item.siteID) == ""
}

func isTarinioAdminAppPath(path string) bool {
	canonical := normalizeBurstPath(path)
	if canonical == "" {
		return false
	}
	for _, exact := range tarinioAdminExactPaths {
		if canonical == exact {
			return true
		}
	}
	for _, prefix := range tarinioAdminPrefixPaths {
		if strings.HasPrefix(canonical, prefix) {
			return true
		}
	}
	for _, prefix := range tarinioAdminSegmentPrefixes {
		if canonical == prefix || strings.HasPrefix(canonical, prefix+"/") {
			return true
		}
	}
	return false
}

func isInternalManagementHost(host string) bool {
	switch strings.TrimSpace(strings.ToLower(host)) {
	case "localhost", "127.0.0.1", "::1", "control-plane", "ui":
		return true
	default:
		return false
	}
}

func isInternalManagementPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/api/"),
		strings.HasPrefix(path, "/static/"),
		strings.HasPrefix(path, "/dashboard"),
		strings.HasPrefix(path, "/healthz"),
		strings.HasPrefix(path, "/readyz"),
		strings.HasPrefix(path, "/login"),
		strings.HasPrefix(path, "/logout"),
		strings.HasPrefix(path, "/setup"),
		strings.HasPrefix(path, "/onboarding"),
		strings.HasPrefix(path, "/favicon"),
		strings.HasPrefix(path, "/manifest"),
		strings.HasPrefix(path, "/site.webmanifest"):
		return true
	default:
		return false
	}
}

func shouldIgnoreBurstPath(path string) bool {
	normalized := normalizeBurstPath(path)
	if normalized == "" {
		return false
	}
	for _, prefix := range []string{
		"/_static/",
		"/static/",
		"/assets/",
		"/build/",
		"/dist/",
		"/favicon",
		"/robots.txt",
		"/manifest",
		"/site.webmanifest",
		"/browserconfig.xml",
		"/apple-touch-icon",
		"/sitemap",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	for _, suffix := range []string{
		".css",
		".js",
		".mjs",
		".map",
		".png",
		".jpg",
		".jpeg",
		".gif",
		".svg",
		".ico",
		".webp",
		".avif",
		".woff",
		".woff2",
		".ttf",
		".otf",
		".eot",
		".json",
		".xml",
		".txt",
	} {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}
