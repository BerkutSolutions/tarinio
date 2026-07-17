package compiler

import (
	"errors"
	"net"
	"path"
	"strings"
)

func validateNginxIdentifier(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New(field + " is required")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return errors.New(field + " contains unsafe characters")
	}
	return nil
}

func validateNginxHost(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\t ;'\"{}\\/") {
		return errors.New(field + " contains unsafe characters")
	}
	if strings.HasPrefix(value, "*.") {
		value = strings.TrimPrefix(value, "*.")
	}
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return errors.New(field + " is not a valid host")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == ':' || r == '[' || r == ']' {
			continue
		}
		return errors.New(field + " is not a valid host")
	}
	return nil
}

func validateNginxCertificatePath(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\t ;'\"{}\\") || !strings.HasPrefix(value, "/") {
		return errors.New(field + " must be a safe absolute certificate path")
	}
	clean := path.Clean(value)
	if clean != value || !(strings.HasPrefix(clean, "/etc/ssl/") || strings.HasPrefix(clean, "/etc/waf/tls/materials/")) {
		return errors.New(field + " must be under an approved certificate root")
	}
	return nil
}

func validateNginxCIDROrIP(value, field string) error {
	value = strings.TrimSpace(value)
	if net.ParseIP(value) != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(value); err == nil {
		return nil
	}
	return errors.New(field + " must be a valid IP address or CIDR")
}
