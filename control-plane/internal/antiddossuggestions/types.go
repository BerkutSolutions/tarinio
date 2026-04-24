package antiddossuggestions

import (
	"errors"
	"strings"
)

const (
	StatusSuggested = "suggested"
	StatusShadow    = "shadow"
)

type Suggestion struct {
	ID         string `json:"id"`
	PathPrefix string `json:"path_prefix"`
	Status     string `json:"status"`
	Hits       int    `json:"hits"`
	UniqueIPs  int    `json:"unique_ips"`
	WouldBlock int    `json:"would_block_hits,omitempty"`
	ShadowHits int    `json:"shadow_hits,omitempty"`
	ShadowFP   int    `json:"shadow_false_positive_hits,omitempty"`
	ShadowRate string `json:"shadow_false_positive_rate,omitempty"`
	Source     string `json:"source,omitempty"`
	Reason     string `json:"reason,omitempty"`
	FirstSeen  string `json:"first_seen,omitempty"`
	LastSeen   string `json:"last_seen,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

func NormalizeSuggestion(item Suggestion) Suggestion {
	out := item
	out.ID = strings.TrimSpace(strings.ToLower(out.ID))
	out.PathPrefix = strings.TrimSpace(strings.ToLower(out.PathPrefix))
	if out.ID == "" && out.PathPrefix != "" {
		out.ID = "path-" + strings.TrimLeft(strings.ReplaceAll(out.PathPrefix, "/", "-"), "-")
	}
	out.Status = strings.TrimSpace(strings.ToLower(out.Status))
	if out.Status == "" {
		out.Status = StatusSuggested
	}
	out.Source = strings.TrimSpace(out.Source)
	out.Reason = strings.TrimSpace(out.Reason)
	out.FirstSeen = strings.TrimSpace(out.FirstSeen)
	out.LastSeen = strings.TrimSpace(out.LastSeen)
	if out.Hits < 0 {
		out.Hits = 0
	}
	if out.UniqueIPs < 0 {
		out.UniqueIPs = 0
	}
	if out.WouldBlock < 0 {
		out.WouldBlock = 0
	}
	if out.ShadowHits < 0 {
		out.ShadowHits = 0
	}
	if out.ShadowFP < 0 {
		out.ShadowFP = 0
	}
	out.ShadowRate = strings.TrimSpace(out.ShadowRate)
	return out
}

func ValidateSuggestion(item Suggestion) error {
	if strings.TrimSpace(item.ID) == "" {
		return errors.New("anti-ddos rule suggestion id is required")
	}
	if strings.TrimSpace(item.PathPrefix) == "" {
		return errors.New("anti-ddos rule suggestion path_prefix is required")
	}
	if !strings.HasPrefix(item.PathPrefix, "/") {
		return errors.New("anti-ddos rule suggestion path_prefix must start with /")
	}
	switch item.Status {
	case StatusSuggested, StatusShadow:
	default:
		return errors.New("anti-ddos rule suggestion status must be suggested or shadow")
	}
	return nil
}
