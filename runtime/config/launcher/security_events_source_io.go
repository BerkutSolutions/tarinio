package main

import (
	"os"
	"strings"
)

func newSecurityEventSource(path string) *securityEventSource {
	return &securityEventSource{
		path:        path,
		threatIntel: newThreatIntelMatcherFromEnv(),
	}
}

func normalizeSiteID(value string) string {
	site := strings.ToLower(strings.TrimSpace(value))
	site = strings.ReplaceAll(site, "_", "-")
	return site
}

func shouldSkipInternalSite(siteID string) bool {
	switch normalizeSiteID(siteID) {
	case "control-plane-access", "control-plane", "ui":
		return true
	default:
		return false
	}
}

func (s *securityEventSource) probe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	_, err = file.Stat()
	return err
}
