package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

// WSInspectionInput holds the WebSocket inspection settings for a site.
type WSInspectionInput struct {
	UseWSInspection   bool
	WSBlockPatterns   []string
	WSMaxMessageBytes int
	WSRateMsgPerSec   int
}

// ValidateWSInspection validates regex patterns in WSBlockPatterns.
// Returns a non-nil error if any pattern is not a valid RE2 regex.
func ValidateWSInspection(input WSInspectionInput) error {
	for i, p := range input.WSBlockPatterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("ws_block_patterns[%d] %q: %w", i, p, err)
		}
	}
	return nil
}

// normalizeWSBlockPatterns deduplicates and trims whitespace from patterns.
func normalizeWSBlockPatterns(patterns []string) []string {
	seen := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// buildWSInspectionServerSnippet generates the nginx server-level Lua block for
// WebSocket frame inspection. Returns an empty string if inspection is disabled
// or there are no patterns/limits configured.
func buildWSInspectionServerSnippet(siteID string, input WSInspectionInput) string {
	if !input.UseWSInspection {
		return ""
	}
	patterns := normalizeWSBlockPatterns(input.WSBlockPatterns)
	hasPatterns := len(patterns) > 0
	hasMaxBytes := input.WSMaxMessageBytes > 0
	hasRateLimit := input.WSRateMsgPerSec > 0

	if !hasPatterns && !hasMaxBytes && !hasRateLimit {
		return ""
	}

	var sb strings.Builder

	// Build Lua pattern table.
	var patternLua string
	if hasPatterns {
		parts := make([]string, 0, len(patterns))
		for _, p := range patterns {
			parts = append(parts, fmt.Sprintf("    %q", p))
		}
		patternLua = "{\n" + strings.Join(parts, ",\n") + "\n  }"
	} else {
		patternLua = "{}"
	}

	maxBytes := 0
	if hasMaxBytes {
		maxBytes = input.WSMaxMessageBytes
	}
	rateLimit := 0
	if hasRateLimit {
		rateLimit = input.WSRateMsgPerSec
	}

	sb.WriteString(fmt.Sprintf(`
  # WebSocket inspection for site %s
  access_by_lua_block {
    local wsi = require("waf.ws_inspection")
    wsi.check({
      block_patterns = %s,
      max_message_bytes = %d,
      rate_msg_per_sec = %d,
    })
  }
`, siteID, patternLua, maxBytes, rateLimit))

	return sb.String()
}

// buildWSInspectionLocationSnippet generates the nginx location-level snippet
// for WebSocket upgrade handling with inspection hooks.
// It must be included inside the WebSocket proxy_pass location block.
func buildWSInspectionLocationSnippet(siteID string, input WSInspectionInput) string {
	if !input.UseWSInspection {
		return ""
	}
	patterns := normalizeWSBlockPatterns(input.WSBlockPatterns)
	hasPatterns := len(patterns) > 0
	hasMaxBytes := input.WSMaxMessageBytes > 0
	hasRateLimit := input.WSRateMsgPerSec > 0

	if !hasPatterns && !hasMaxBytes && !hasRateLimit {
		return ""
	}

	var patternLua string
	if hasPatterns {
		parts := make([]string, 0, len(patterns))
		for _, p := range patterns {
			parts = append(parts, fmt.Sprintf("      %q", p))
		}
		patternLua = "{\n" + strings.Join(parts, ",\n") + "\n    }"
	} else {
		patternLua = "{}"
	}

	maxBytes := 0
	if hasMaxBytes {
		maxBytes = input.WSMaxMessageBytes
	}
	rateLimit := 0
	if hasRateLimit {
		rateLimit = input.WSRateMsgPerSec
	}

	return fmt.Sprintf(`
    # WebSocket frame inspection for site %s
    header_filter_by_lua_block {
      local wsi = require("waf.ws_inspection")
      wsi.on_upgrade({
        block_patterns = %s,
        max_message_bytes = %d,
        rate_msg_per_sec = %d,
      })
    }
`, siteID, patternLua, maxBytes, rateLimit)
}
