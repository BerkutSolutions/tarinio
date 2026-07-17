package virtualpatches

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	TargetURI    = "uri"
	TargetBody   = "body"
	TargetHeader = "header"

	ActionBlock   = "block"
	ActionMonitor = "monitor"
)

// VirtualPatch is a temporary blocking rule with optional TTL.
type VirtualPatch struct {
	ID          string `json:"id"`
	SiteID      string `json:"site_id"`
	Pattern     string `json:"pattern"`
	Target      string `json:"target"`
	Action      string `json:"action"`
	ExpiresAt   string `json:"expires_at,omitempty"` // RFC3339, empty = permanent
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by,omitempty"`
}

// IsExpired reports whether the patch has passed its expiry time.
func (p VirtualPatch) IsExpired() bool {
	if strings.TrimSpace(p.ExpiresAt) == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(t)
}

// Normalize trims whitespace and lowercases target/action.
func Normalize(p VirtualPatch) VirtualPatch {
	out := p
	out.SiteID = strings.TrimSpace(out.SiteID)
	out.Pattern = strings.TrimSpace(out.Pattern)
	out.Target = strings.ToLower(strings.TrimSpace(out.Target))
	out.Action = strings.ToLower(strings.TrimSpace(out.Action))
	out.Description = strings.TrimSpace(out.Description)
	out.ExpiresAt = strings.TrimSpace(out.ExpiresAt)
	out.CreatedBy = strings.TrimSpace(out.CreatedBy)
	return out
}

// Validate checks that the patch fields are well-formed.
func Validate(p VirtualPatch) error {
	if !safePatchIdentifier(p.ID) {
		return errors.New("virtual patch id must be a safe identifier")
	}
	if strings.TrimSpace(p.SiteID) == "" {
		return errors.New("virtual patch site_id is required")
	}
	if strings.TrimSpace(p.Pattern) == "" {
		return errors.New("virtual patch pattern is required")
	}
	if containsControlCharacter(p.Pattern) {
		return errors.New("virtual patch pattern must not contain control characters")
	}
	if _, err := regexp.Compile(p.Pattern); err != nil {
		return errors.New("virtual patch pattern is not a valid regex: " + err.Error())
	}
	switch p.Target {
	case TargetURI, TargetBody, TargetHeader:
	default:
		return errors.New("virtual patch target must be uri, body, or header")
	}
	switch p.Action {
	case ActionBlock, ActionMonitor:
	default:
		return errors.New("virtual patch action must be block or monitor")
	}
	if p.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, p.ExpiresAt); err != nil {
			return errors.New("virtual patch expires_at must be RFC3339 format")
		}
	}
	return nil
}

func safePatchIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func containsControlCharacter(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0
}
