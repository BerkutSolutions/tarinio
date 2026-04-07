package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"
)

type nginxMainData struct {
	SitesIncludeGlob string
}

type nginxSiteData struct {
	SiteID                    string
	SiteIDSlug                string
	ServerNames               []string
	ListenHTTP                bool
	ListenHTTPS               bool
	UpstreamName              string
	UpstreamAddress           string
	ProxyPassTarget           string
	PassHostHeader            bool
	RateLimitCookie           string
	RateLimitEscalationCookie string
}

type errorPageData struct {
	StatusCode         int
	Title              string
	Category           string
	Summary            string
	ClientStateLabel   string
	ClientStateTone    string
	ClientStateText    string
	WAFStateLabel      string
	WAFStateTone       string
	WAFStateText       string
	UpstreamStateLabel string
	UpstreamStateTone  string
	UpstreamStateText  string
	Suggestions        []string
	Accent             string
	AccentSoft         string
}

const globalErrorSiteID = "_global"

var supportedErrorStatusCodes = []int{
	400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415,
	416, 417, 418, 421, 422, 423, 424, 425, 426, 428, 429, 431, 444, 451, 500, 501,
	502, 503, 504, 505, 507, 508, 510, 511, 520, 521, 522, 523, 524, 525, 526,
}

// RenderSiteUpstreamArtifacts produces deterministic nginx artifacts for the MVP
// Site and Upstream compiler mapping.
func RenderSiteUpstreamArtifacts(sites []SiteInput, upstreams []UpstreamInput) ([]ArtifactOutput, error) {
	sortedSites, upstreamByID, err := normalizeInputs(sites, upstreams)
	if err != nil {
		return nil, err
	}

	var artifacts []ArtifactOutput

	mainContent, err := renderTemplate(filepath.Join(templatesRoot(), "nginx.conf.tmpl"), nginxMainData{
		SitesIncludeGlob: "sites/*.conf",
	})
	if err != nil {
		return nil, fmt.Errorf("render nginx main template: %w", err)
	}
	artifacts = append(artifacts, newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, mainContent))

	baseContent, err := renderTemplate(filepath.Join(templatesRoot(), "conf.d", "base.conf.tmpl"), nginxBaseData{
		ErrorStatusCodes: append([]int(nil), supportedErrorStatusCodes...),
	})
	if err != nil {
		return nil, fmt.Errorf("render nginx base template: %w", err)
	}
	artifacts = append(artifacts, newArtifact("nginx/conf.d/base.conf", ArtifactKindNginxConfig, baseContent))

	globalErrorArtifacts, err := renderSiteErrorArtifacts(globalErrorSiteID)
	if err != nil {
		return nil, fmt.Errorf("render global error pages: %w", err)
	}
	artifacts = append(artifacts, globalErrorArtifacts...)

	for _, site := range sortedSites {
		if !site.Enabled {
			continue
		}

		upstream := upstreamByID[site.DefaultUpstreamID]
		siteContent, err := renderTemplate(filepath.Join(templatesRoot(), "sites", "site.conf.tmpl"), nginxSiteData{
			SiteID:                    site.ID,
			SiteIDSlug:                slugSiteID(site.ID),
			ServerNames:               collectServerNames(site),
			ListenHTTP:                site.ListenHTTP,
			ListenHTTPS:               site.ListenHTTPS,
			UpstreamName:              upstreamBlockName(site.ID, upstream.ID),
			UpstreamAddress:           buildUpstreamAddress(upstream),
			ProxyPassTarget:           "http://" + upstreamBlockName(site.ID, upstream.ID),
			PassHostHeader:            upstream.PassHostHeader,
			RateLimitCookie:           rateLimitCookieName(site.ID),
			RateLimitEscalationCookie: rateLimitEscalationCookieName(site.ID),
		})
		if err != nil {
			return nil, fmt.Errorf("render site template for %s: %w", site.ID, err)
		}

		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("nginx/sites/%s.conf", site.ID),
			ArtifactKindNginxConfig,
			siteContent,
		))
		errorArtifacts, err := renderSiteErrorArtifacts(site.ID)
		if err != nil {
			return nil, fmt.Errorf("render error pages for %s: %w", site.ID, err)
		}
		artifacts = append(artifacts, errorArtifacts...)
	}

	return artifacts, nil
}

func normalizeInputs(sites []SiteInput, upstreams []UpstreamInput) ([]SiteInput, map[string]UpstreamInput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool {
		return sortedSites[i].ID < sortedSites[j].ID
	})

	upstreamByID := make(map[string]UpstreamInput, len(upstreams))
	for _, upstream := range upstreams {
		if upstream.ID == "" {
			return nil, nil, errors.New("upstream id is required")
		}
		if upstream.SiteID == "" {
			return nil, nil, fmt.Errorf("upstream %s site id is required", upstream.ID)
		}
		if upstream.Scheme == "" || upstream.Host == "" || upstream.Port <= 0 {
			return nil, nil, fmt.Errorf("upstream %s must define scheme, host, and port", upstream.ID)
		}
		upstreamByID[upstream.ID] = upstream
	}

	for i := range sortedSites {
		site := &sortedSites[i]
		if site.ID == "" {
			return nil, nil, errors.New("site id is required")
		}
		if !site.Enabled {
			continue
		}
		if site.PrimaryHost == "" {
			return nil, nil, fmt.Errorf("site %s primary host is required", site.ID)
		}
		if !site.ListenHTTP && !site.ListenHTTPS {
			return nil, nil, fmt.Errorf("site %s must enable at least one listener", site.ID)
		}
		if site.DefaultUpstreamID == "" {
			return nil, nil, fmt.Errorf("site %s default upstream is required", site.ID)
		}
		upstream, ok := upstreamByID[site.DefaultUpstreamID]
		if !ok {
			return nil, nil, fmt.Errorf("site %s default upstream %s not found", site.ID, site.DefaultUpstreamID)
		}
		if upstream.SiteID != site.ID {
			return nil, nil, fmt.Errorf("site %s default upstream %s belongs to site %s", site.ID, upstream.ID, upstream.SiteID)
		}

		site.Aliases = sortedUnique(site.Aliases)
	}

	return sortedSites, upstreamByID, nil
}

func collectServerNames(site SiteInput) []string {
	names := append([]string{site.PrimaryHost}, site.Aliases...)
	return sortedUnique(names)
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func upstreamBlockName(siteID, upstreamID string) string {
	return fmt.Sprintf("site_%s_upstream_%s", siteID, upstreamID)
}

func buildUpstreamAddress(upstream UpstreamInput) string {
	return fmt.Sprintf("%s:%d", upstream.Host, upstream.Port)
}

func rateLimitCookieName(siteID string) string {
	return "waf_rate_limited_" + slugSiteID(siteID)
}

func rateLimitEscalationCookieName(siteID string) string {
	return "waf_rate_limited_escalated_" + slugSiteID(siteID)
}

func slugSiteID(siteID string) string {
	var out []rune
	for _, r := range strings.ToLower(strings.TrimSpace(siteID)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			continue
		}
		out = append(out, '_')
	}
	value := strings.Trim(string(out), "_")
	if value == "" {
		return "site"
	}
	return value
}

func renderTemplate(path string, data any) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newArtifact(path string, kind ArtifactKind, content []byte) ArtifactOutput {
	sum := sha256.Sum256(content)
	return ArtifactOutput{
		Path:     path,
		Kind:     kind,
		Content:  content,
		Checksum: hex.EncodeToString(sum[:]),
	}
}

func templatesRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("compiler", "templates", "nginx")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..", "templates", "nginx")
}

func renderSiteErrorArtifacts(siteID string) ([]ArtifactOutput, error) {
	pages := buildErrorPageCatalog()

	artifacts := make([]ArtifactOutput, 0, len(pages))
	for _, page := range pages {
		content, err := renderTemplate(filepath.Join(templatesRoot(), "..", "errors", "status.html.tmpl"), page)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("errors/%s/%d.html", siteID, page.StatusCode),
			ArtifactKindNginxConfig,
			content,
		))
	}
	return artifacts, nil
}

type nginxBaseData struct {
	ErrorStatusCodes []int
}

func buildErrorPageCatalog() []errorPageData {
	overrides := map[int]errorPageData{
		400: {
			StatusCode:         400,
			Title:              "Bad Request",
			Category:           "Client-side request issue",
			Summary:            "The request reached TARINIO, but it was malformed or incomplete before it could be served normally.",
			ClientStateLabel:   "Needs fix",
			ClientStateTone:    "warn",
			ClientStateText:    "The browser or client sent a request that could not be processed safely.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is responding and able to return a controlled fallback page.",
			UpstreamStateLabel: "Unknown",
			UpstreamStateTone:  "warn",
			UpstreamStateText:  "The upstream was not the primary reason for this response.",
			Suggestions:        []string{"Retry the action with a clean URL or form payload.", "Check the request body, headers, and path for invalid syntax.", "If this persists for normal traffic, inspect client-side integrations."},
			Accent:             "#ffbe63",
			AccentSoft:         "rgba(255, 190, 99, 0.18)",
		},
		401: {
			StatusCode:         401,
			Title:              "Authentication Required",
			Category:           "Protected resource",
			Summary:            "The destination is reachable, but authentication is required before the request can continue.",
			ClientStateLabel:   "Sign in",
			ClientStateTone:    "warn",
			ClientStateText:    "The client must provide a valid session or credentials.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is healthy and forwarding the protected flow correctly.",
			UpstreamStateLabel: "Online",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The application is reachable but requires authentication.",
			Suggestions:        []string{"Sign in again and retry the request.", "Check whether your session expired.", "Confirm that the protected path is expected to require login."},
			Accent:             "#86a8ff",
			AccentSoft:         "rgba(134, 168, 255, 0.16)",
		},
		403: {
			StatusCode:         403,
			Title:              "Forbidden",
			Category:           "Access denied",
			Summary:            "The request reached the WAF, but policy or access controls refused it before the application could serve the response.",
			ClientStateLabel:   "Blocked",
			ClientStateTone:    "bad",
			ClientStateText:    "The request did not satisfy the required access rules.",
			WAFStateLabel:      "Active",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is working and intentionally enforcing policy.",
			UpstreamStateLabel: "Protected",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The application remains behind the WAF boundary.",
			Suggestions:        []string{"Check IP allow/deny rules and authentication state.", "Review WAF and access-policy logs for the matched rule.", "If this is expected traffic, relax the policy instead of bypassing the WAF."},
			Accent:             "#ff7f7f",
			AccentSoft:         "rgba(255, 127, 127, 0.18)",
		},
		404: {
			StatusCode:         404,
			Title:              "Not Found",
			Category:           "Missing route",
			Summary:            "The WAF and upstream are alive, but the requested path does not exist at the destination.",
			ClientStateLabel:   "Wrong path",
			ClientStateTone:    "warn",
			ClientStateText:    "The client requested a route or asset that is not available.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is up and able to return a controlled page.",
			UpstreamStateLabel: "Online",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The application responded, but the target resource was missing.",
			Suggestions:        []string{"Verify the URL path and trailing slash.", "Check whether the route was renamed or removed.", "Inspect the application routing table if the page should exist."},
			Accent:             "#86a8ff",
			AccentSoft:         "rgba(134, 168, 255, 0.16)",
		},
		405: {
			StatusCode:         405,
			Title:              "Method Not Allowed",
			Category:           "Request method rejected",
			Summary:            "The destination exists, but the HTTP method used for this request is not allowed.",
			ClientStateLabel:   "Adjust method",
			ClientStateTone:    "warn",
			ClientStateText:    "The client used a method like POST, PUT, or DELETE on a route that does not accept it.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF stayed in the path and returned a readable fallback.",
			UpstreamStateLabel: "Online",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The application route is present but method-specific.",
			Suggestions:        []string{"Retry using the expected method.", "Confirm the endpoint contract in your client or integration.", "Inspect API docs or frontend form wiring."},
			Accent:             "#ffbe63",
			AccentSoft:         "rgba(255, 190, 99, 0.18)",
		},
		408: {
			StatusCode:         408,
			Title:              "Request Timeout",
			Category:           "Client-side timeout",
			Summary:            "The request did not complete in time. The WAF remained responsive and closed the request safely.",
			ClientStateLabel:   "Timed out",
			ClientStateTone:    "warn",
			ClientStateText:    "The client connection stalled or the request body was not delivered in time.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is still available and handling timeouts predictably.",
			UpstreamStateLabel: "Unknown",
			UpstreamStateTone:  "warn",
			UpstreamStateText:  "The upstream may not have received a complete request.",
			Suggestions:        []string{"Retry the request from a stable connection.", "Reduce large uploads or slow client behavior.", "Inspect timeout settings if this happens repeatedly."},
			Accent:             "#ffbe63",
			AccentSoft:         "rgba(255, 190, 99, 0.18)",
		},
		421: {
			StatusCode:         421,
			Title:              "Host Not Configured",
			Category:           "WAF direct address protection",
			Summary:            "The request reached WAF by IP or an unknown host header. This address is reserved and serves a protection stub page.",
			ClientStateLabel:   "Reconfigure host",
			ClientStateTone:    "warn",
			ClientStateText:    "Use the configured domain name routed through your proxy/CDN instead of direct WAF IP access.",
			WAFStateLabel:      "Protected",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is online and intentionally blocks direct host/IP access for this service.",
			UpstreamStateLabel: "Hidden",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The upstream remains reachable only via the configured protected host.",
			Suggestions:        []string{"Point DNS to the expected host configured in WAF.", "Ensure reverse proxy/CDN forwards requests with the configured Host.", "Avoid direct requests to raw WAF IP addresses."},
			Accent:             "#86a8ff",
			AccentSoft:         "rgba(134, 168, 255, 0.16)",
		},
		429: {
			StatusCode:         429,
			Title:              "Too Many Requests",
			Category:           "Rate limiting",
			Summary:            "The WAF is online and intentionally slowed this client because the request volume crossed the configured threshold.",
			ClientStateLabel:   "Rate limited",
			ClientStateTone:    "bad",
			ClientStateText:    "The current request rate is higher than the allowed burst or sustained quota.",
			WAFStateLabel:      "Protecting",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is active and defending the upstream from overload or abuse.",
			UpstreamStateLabel: "Protected",
			UpstreamStateTone:  "ok",
			UpstreamStateText:  "The upstream remains available behind the rate limit shield.",
			Suggestions:        []string{"Wait briefly before retrying.", "Reduce polling, retries, or bursty client behavior.", "Tune per-site rate limits if this traffic is legitimate."},
			Accent:             "#ff7f7f",
			AccentSoft:         "rgba(255, 127, 127, 0.18)",
		},
		500: {
			StatusCode:         500,
			Title:              "Internal Server Error",
			Category:           "Application-side failure",
			Summary:            "The request looked valid and the WAF stayed healthy, but the upstream application failed while processing it.",
			ClientStateLabel:   "Looks normal",
			ClientStateTone:    "ok",
			ClientStateText:    "The request itself was not the primary problem.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF is available and relaying the application failure clearly.",
			UpstreamStateLabel: "Error",
			UpstreamStateTone:  "bad",
			UpstreamStateText:  "The upstream returned an internal error and needs investigation.",
			Suggestions:        []string{"Check the upstream application logs.", "Confirm environment variables, database access, and internal dependencies.", "Retry after the application error is resolved."},
			Accent:             "#ff7f7f",
			AccentSoft:         "rgba(255, 127, 127, 0.18)",
		},
		502: {
			StatusCode:         502,
			Title:              "Bad Gateway",
			Category:           "Upstream connection issue",
			Summary:            "TARINIO is online, but it could not get a valid response from the upstream service behind it.",
			ClientStateLabel:   "Looks normal",
			ClientStateTone:    "ok",
			ClientStateText:    "The request reached the edge successfully.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF accepted the request and attempted to proxy it.",
			UpstreamStateLabel: "Unavailable",
			UpstreamStateTone:  "bad",
			UpstreamStateText:  "The upstream is down, misrouted, or returned an invalid gateway response.",
			Suggestions:        []string{"Check upstream host, port, and scheme configuration.", "Verify that the application container or service is running.", "Inspect reverse-proxy and application logs for startup or networking errors."},
			Accent:             "#ff7f7f",
			AccentSoft:         "rgba(255, 127, 127, 0.18)",
		},
		503: {
			StatusCode:         503,
			Title:              "Service Unavailable",
			Category:           "Temporary unavailability",
			Summary:            "The WAF is healthy, but the protected application is currently unable to serve traffic.",
			ClientStateLabel:   "Looks normal",
			ClientStateTone:    "ok",
			ClientStateText:    "The client reached the site successfully.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF remains available and is returning a controlled fallback.",
			UpstreamStateLabel: "Busy / down",
			UpstreamStateTone:  "bad",
			UpstreamStateText:  "The application is overloaded, restarting, or intentionally unavailable.",
			Suggestions:        []string{"Retry after a short delay.", "Check maintenance mode, rollout state, and readiness probes.", "Inspect upstream resource pressure and dependency health."},
			Accent:             "#ffbe63",
			AccentSoft:         "rgba(255, 190, 99, 0.18)",
		},
		504: {
			StatusCode:         504,
			Title:              "Gateway Timeout",
			Category:           "Slow upstream response",
			Summary:            "The WAF stayed online, but the upstream service took too long to answer this request.",
			ClientStateLabel:   "Looks normal",
			ClientStateTone:    "ok",
			ClientStateText:    "The request reached the WAF and was forwarded normally.",
			WAFStateLabel:      "Online",
			WAFStateTone:       "ok",
			WAFStateText:       "The WAF remained available while waiting for the upstream.",
			UpstreamStateLabel: "Too slow",
			UpstreamStateTone:  "bad",
			UpstreamStateText:  "The upstream exceeded the time budget for a valid response.",
			Suggestions:        []string{"Inspect upstream latency and long-running handlers.", "Review database or dependency timeouts.", "Tune timeout values only after confirming the root cause."},
			Accent:             "#ffbe63",
			AccentSoft:         "rgba(255, 190, 99, 0.18)",
		},
	}

	out := make([]errorPageData, 0, len(supportedErrorStatusCodes))
	for _, code := range supportedErrorStatusCodes {
		if page, ok := overrides[code]; ok {
			out = append(out, page)
			continue
		}
		out = append(out, buildGenericErrorPageData(code))
	}
	return out
}

func buildGenericErrorPageData(code int) errorPageData {
	title := strings.TrimSpace(http.StatusText(code))
	if title == "" {
		title = "Request Failed"
	}
	category := "Request failed"
	accent := "#ffbe63"
	accentSoft := "rgba(255, 190, 99, 0.18)"
	clientLabel := "Check request"
	clientTone := "warn"
	upstreamLabel := "Check upstream"
	upstreamTone := "warn"
	if code >= 500 {
		category = "Upstream/server failure"
		accent = "#ff7f7f"
		accentSoft = "rgba(255, 127, 127, 0.18)"
		clientLabel = "Looks normal"
		clientTone = "ok"
		upstreamLabel = "Investigate"
		upstreamTone = "bad"
	} else if code >= 400 {
		category = "Client or access error"
	}
	return errorPageData{
		StatusCode:         code,
		Title:              title,
		Category:           category,
		Summary:            "TARINIO returned a controlled fallback page for this response code.",
		ClientStateLabel:   clientLabel,
		ClientStateTone:    clientTone,
		ClientStateText:    "Validate request path, method, headers, and client behavior.",
		WAFStateLabel:      "Online",
		WAFStateTone:       "ok",
		WAFStateText:       "The WAF is active and handling this response path safely.",
		UpstreamStateLabel: upstreamLabel,
		UpstreamStateTone:  upstreamTone,
		UpstreamStateText:  "If this repeats, inspect upstream availability and policy match conditions.",
		Suggestions:        []string{"Retry the request after validating URL and method.", "Review WAF/access/rate-limit rules for this endpoint.", "Check upstream logs for related request IDs."},
		Accent:             accent,
		AccentSoft:         accentSoft,
	}
}
