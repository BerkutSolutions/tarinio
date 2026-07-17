package compiler

import (
	"fmt"
	"strings"
)

func validateEasyDirectiveInputs(profile EasyProfileInput) error {
	if err := validateNginxPathPattern(profile.LimitReqURL, "limit_req_url", true); err != nil {
		return err
	}
	for _, value := range profile.BlacklistURI {
		if err := validateNginxPathPattern(value, "blacklist URI pattern", false); err != nil {
			return err
		}
	}
	for _, value := range profile.ExceptionsURI {
		if err := validateNginxPathPattern(value, "exception URI pattern", true); err != nil {
			return err
		}
	}
	for _, rule := range profile.AuthExclusionRules {
		if err := validateNginxPathPattern(rule.Path, "auth exclusion path", true); err != nil {
			return err
		}
	}
	return nil
}

func validateNginxPathPattern(value, field string, requireLeadingSlash bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if (requireLeadingSlash && !strings.HasPrefix(value, "/")) || strings.ContainsAny(value, "\r\n\t \"';{}#") {
		return fmt.Errorf("%s contains unsafe nginx directive characters", field)
	}
	return nil
}
