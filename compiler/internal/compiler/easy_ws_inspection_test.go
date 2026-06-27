package compiler

import (
	"strings"
	"testing"
)

func TestValidateWSInspection_ValidPatterns(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`DROP TABLE`, `(?i)select.*from`, `<script>`},
	}
	if err := ValidateWSInspection(input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWSInspection_InvalidPattern(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`valid`, `[invalid`},
	}
	err := ValidateWSInspection(input)
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "ws_block_patterns[1]") {
		t.Fatalf("error should mention index 1, got: %v", err)
	}
}

func TestBuildWSInspectionServerSnippet_Disabled(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: false,
		WSBlockPatterns: []string{`DROP TABLE`},
		WSMaxMessageBytes: 1024,
	}
	snippet := buildWSInspectionServerSnippet("mysite", input)
	if snippet != "" {
		t.Fatalf("expected empty snippet when disabled, got: %q", snippet)
	}
}

func TestBuildWSInspectionServerSnippet_NoRulesNoLimits(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection:   true,
		WSBlockPatterns:   []string{},
		WSMaxMessageBytes: 0,
		WSRateMsgPerSec:   0,
	}
	snippet := buildWSInspectionServerSnippet("mysite", input)
	if snippet != "" {
		t.Fatalf("expected empty snippet when no patterns/limits, got: %q", snippet)
	}
}

func TestBuildWSInspectionServerSnippet_WithPatterns(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`DROP TABLE`, `<script>`},
	}
	snippet := buildWSInspectionServerSnippet("testsite", input)
	if snippet == "" {
		t.Fatal("expected non-empty snippet")
	}
	if !strings.Contains(snippet, "waf.ws_inspection") {
		t.Errorf("snippet should reference waf.ws_inspection lua module, got: %s", snippet)
	}
	if !strings.Contains(snippet, `"DROP TABLE"`) {
		t.Errorf("snippet should contain DROP TABLE pattern, got: %s", snippet)
	}
	if !strings.Contains(snippet, `"<script>"`) {
		t.Errorf("snippet should contain <script> pattern, got: %s", snippet)
	}
	if !strings.Contains(snippet, "testsite") {
		t.Errorf("snippet should reference site ID, got: %s", snippet)
	}
}

func TestBuildWSInspectionServerSnippet_WithMaxBytes(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection:   true,
		WSBlockPatterns:   []string{},
		WSMaxMessageBytes: 65536,
		WSRateMsgPerSec:   0,
	}
	snippet := buildWSInspectionServerSnippet("site1", input)
	if snippet == "" {
		t.Fatal("expected non-empty snippet when max_message_bytes set")
	}
	if !strings.Contains(snippet, "65536") {
		t.Errorf("snippet should contain max_message_bytes=65536, got: %s", snippet)
	}
}

func TestBuildWSInspectionServerSnippet_WithRateLimit(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: true,
		WSRateMsgPerSec: 100,
	}
	snippet := buildWSInspectionServerSnippet("site2", input)
	if snippet == "" {
		t.Fatal("expected non-empty snippet when rate_msg_per_sec set")
	}
	if !strings.Contains(snippet, "100") {
		t.Errorf("snippet should contain rate=100, got: %s", snippet)
	}
}

func TestNormalizeWSBlockPatterns_Deduplication(t *testing.T) {
	t.Parallel()
	patterns := []string{"DROP TABLE", "  DROP TABLE  ", "SELECT", "DROP TABLE"}
	result := normalizeWSBlockPatterns(patterns)
	if len(result) != 2 {
		t.Fatalf("expected 2 unique patterns, got %d: %v", len(result), result)
	}
	if result[0] != "DROP TABLE" || result[1] != "SELECT" {
		t.Fatalf("unexpected patterns: %v", result)
	}
}

func TestNormalizeWSBlockPatterns_EmptyStrings(t *testing.T) {
	t.Parallel()
	patterns := []string{"", "  ", "valid", ""}
	result := normalizeWSBlockPatterns(patterns)
	if len(result) != 1 || result[0] != "valid" {
		t.Fatalf("expected [valid], got %v", result)
	}
}

func TestBuildWSInspectionLocationSnippet_Disabled(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{UseWSInspection: false, WSBlockPatterns: []string{"x"}}
	if s := buildWSInspectionLocationSnippet("s", input); s != "" {
		t.Fatalf("expected empty, got %q", s)
	}
}

func TestBuildWSInspectionLocationSnippet_WithPattern(t *testing.T) {
	t.Parallel()
	input := WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`evil`},
	}
	snippet := buildWSInspectionLocationSnippet("wssite", input)
	if !strings.Contains(snippet, "header_filter_by_lua_block") {
		t.Errorf("location snippet should use header_filter_by_lua_block, got: %s", snippet)
	}
	if !strings.Contains(snippet, `"evil"`) {
		t.Errorf("location snippet should contain pattern, got: %s", snippet)
	}
}
