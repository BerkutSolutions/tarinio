package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func normalizeACMEDirectoryURLs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		parsed, err := validateACMEDirectoryURL(value)
		if err != nil {
			continue
		}
		key := parsed.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}

func (c *ACMELetsEncryptClient) customDirectoryAllowed(value string) bool {
	parsed, err := validateACMEDirectoryURL(value)
	if err != nil {
		return false
	}
	for _, approved := range c.cfg.CustomDirectoryURLs {
		if parsed.String() == approved {
			return true
		}
	}
	return false
}

func newACMEHTTPClient(directoryURL string) (*http.Client, error) {
	directory, err := validateACMEDirectoryURL(directoryURL)
	if err != nil {
		return nil, err
	}
	allowedHost := strings.ToLower(directory.Hostname())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(host, allowedHost) || (port != "443" && port != "") {
			return nil, errors.New("acme egress destination is not approved")
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve acme directory host: %w", err)
		}
		for _, ip := range ips {
			if !isPublicACMEAddress(ip.AsSlice()) {
				continue
			}
			dialer := net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		return nil, errors.New("acme directory host has no public IP address")
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	client.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
		if request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Hostname(), allowedHost) {
			return errors.New("acme redirect destination is not approved")
		}
		return nil
	}
	return client, nil
}

func validateACMEDirectoryURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, errors.New("acme directory URL must be a safe HTTPS URL")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return nil, errors.New("acme directory URL must use port 443")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed, nil
}

func isPublicACMEAddress(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
