package loggingconfig

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ClickHouseAllowedEndpointsEnv contains exact base endpoints that the runtime
// may contact. It is a deployment egress allowlist, not a user setting.
const ClickHouseAllowedEndpointsEnv = "WAF_CLICKHOUSE_ALLOWED_ENDPOINTS"

func AllowedClickHouseEndpointsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv(ClickHouseAllowedEndpointsEnv))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("CLICKHOUSE_ENDPOINT"))
	}
	if raw == "" {
		raw = "http://clickhouse:8123"
	}
	parts := strings.Split(raw, ",")
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		endpoint, err := normalizeClickHouseEndpoint(part)
		if err == nil {
			allowed = append(allowed, endpoint)
		}
	}
	return allowed
}

func ValidateClickHouseEndpoint(endpoint string, allowed []string) (string, error) {
	normalized, err := normalizeClickHouseEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	for _, candidate := range allowed {
		candidate, err = normalizeClickHouseEndpoint(candidate)
		if err == nil && candidate == normalized {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("clickhouse endpoint is not in %s", ClickHouseAllowedEndpointsEnv)
}

func normalizeClickHouseEndpoint(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("clickhouse endpoint must be an absolute HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("clickhouse endpoint must use http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("clickhouse endpoint must not contain credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path != "" {
		return "", fmt.Errorf("clickhouse endpoint must not contain a path")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}
