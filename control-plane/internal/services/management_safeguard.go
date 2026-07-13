package services

import (
	"fmt"
	"strings"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/revisionsnapshots"
)

func validateManagementSafeguard(bundle *pipeline.RevisionBundle, snapshot revisionsnapshots.Snapshot) error {
	if !snapshot.ManagementHostsConfigured {
		return nil
	}
	for _, host := range snapshot.ManagementHosts {
		siteID := managementSiteIDForHost(snapshot, host)
		if siteID == "" {
			return fmt.Errorf("management safeguard preflight: host %q has no enabled site", host)
		}
		easyPath := "nginx/easy/" + siteID + ".conf"
		sitePath := "nginx/sites/" + siteID + ".conf"
		modSecurityPath := "modsecurity/easy/" + siteID + ".conf"
		modSecurityEnabled := false
		for _, artifact := range bundle.Files {
			if artifact.Path == modSecurityPath {
				modSecurityEnabled = true
			}
			if (artifact.Path == easyPath || artifact.Path == sitePath) && hasManagementModSecurityBypass(string(artifact.Content)) {
				goto nextHost
			}
		}
		// There is no CRS request to bypass when the selected management
		// profile has ModSecurity disabled. The site still receives dedicated
		// management API routing; requiring a nonexistent ctl rule would make
		// a safe transparent-profile revision impossible to apply.
		if !modSecurityEnabled {
			goto nextHost
		}
		return fmt.Errorf("management safeguard preflight: artifact for host %q is missing bypass", host)
	nextHost:
	}
	return nil
}

// hasManagementModSecurityBypass accepts the legacy scoped ctl rule and the
// current dedicated management API location. The latter is intentionally in
// the site artifact so `/api/*` never inherits an easy profile's WAF settings.
func hasManagementModSecurityBypass(content string) bool {
	if strings.Contains(content, "ctl:ruleEngine=Off") {
		return true
	}
	apiStart := strings.Index(content, "location ^~ /api/ {")
	if apiStart < 0 {
		return false
	}
	apiBlock := content[apiStart:]
	if nextLocation := strings.Index(apiBlock[len("location ^~ /api/ {"):], "location "); nextLocation >= 0 {
		apiBlock = apiBlock[:len("location ^~ /api/ {")+nextLocation]
	}
	return strings.Contains(apiBlock, "modsecurity off;") && strings.Contains(apiBlock, "proxy_pass http://")
}

func managementSiteIDForHost(snapshot revisionsnapshots.Snapshot, host string) string {
	canonical := strings.ToLower(strings.TrimSpace(host))
	for _, site := range snapshot.Sites {
		if site.Enabled && strings.ToLower(strings.TrimSpace(site.PrimaryHost)) == canonical {
			return site.ID
		}
	}
	return ""
}
