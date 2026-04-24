package services

import (
	"net/url"
	"strings"
)

func runtimeEndpointCandidates(rawURL string, defaultURL string) []string {
	base := strings.TrimSpace(rawURL)
	if base == "" {
		base = strings.TrimSpace(defaultURL)
	}
	if base == "" {
		return nil
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return []string{base}
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return []string{base}
	}
	port := strings.TrimSpace(parsed.Port())
	hosts := runtimeHostFallbacks(host)
	out := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, candidateHost := range hosts {
		candidateHost = strings.TrimSpace(candidateHost)
		if candidateHost == "" {
			continue
		}
		candidate := *parsed
		if port != "" {
			candidate.Host = candidateHost + ":" + port
		} else {
			candidate.Host = candidateHost
		}
		value := candidate.String()
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return []string{base}
	}
	return out
}

func runtimeHostFallbacks(host string) []string {
	normalized := strings.ToLower(strings.TrimSpace(host))
	out := []string{host}
	appendIfMissing := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			exists := false
			for _, current := range out {
				if strings.EqualFold(strings.TrimSpace(current), value) {
					exists = true
					break
				}
			}
			if !exists {
				out = append(out, value)
			}
		}
	}

	switch normalized {
	case "runtime":
		appendIfMissing("tarinio-runtime", "127.0.0.1", "localhost")
	case "tarinio-runtime":
		appendIfMissing("runtime", "127.0.0.1", "localhost")
	case "127.0.0.1":
		appendIfMissing("localhost", "runtime", "tarinio-runtime")
	case "localhost":
		appendIfMissing("127.0.0.1", "runtime", "tarinio-runtime")
	default:
		appendIfMissing("127.0.0.1", "localhost")
	}
	return out
}
