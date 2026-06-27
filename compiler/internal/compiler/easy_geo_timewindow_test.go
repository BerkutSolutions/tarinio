package compiler

import (
	"strings"
	"testing"
)

// --- ValidateGeoTimeWindow ---

func TestValidateGeoTimeWindow_Valid(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DE", "FR"},
		Action:     "block",
		DaysOfWeek: []int{1, 2, 3},
		HoursStart: 9,
		HoursEnd:   17,
	}
	if err := ValidateGeoTimeWindow(w); err != nil {
		t.Fatalf("expected valid window, got error: %v", err)
	}
}

func TestValidateGeoTimeWindow_HoursStartGEHoursEnd(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DE"},
		Action:     "block",
		HoursStart: 17,
		HoursEnd:   9,
	}
	if err := ValidateGeoTimeWindow(w); err == nil {
		t.Fatal("expected error when hours_start >= hours_end")
	}
}

func TestValidateGeoTimeWindow_HoursStartEqualsHoursEnd(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DE"},
		Action:     "block",
		HoursStart: 10,
		HoursEnd:   10,
	}
	if err := ValidateGeoTimeWindow(w); err == nil {
		t.Fatal("expected error when hours_start == hours_end")
	}
}

func TestValidateGeoTimeWindow_InvalidAction(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DE"},
		Action:     "deny",
		HoursStart: 9,
		HoursEnd:   17,
	}
	if err := ValidateGeoTimeWindow(w); err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestValidateGeoTimeWindow_InvalidCountryCode(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DEU"},
		Action:     "block",
		HoursStart: 9,
		HoursEnd:   17,
	}
	if err := ValidateGeoTimeWindow(w); err == nil {
		t.Fatal("expected error for invalid country code DEU")
	}
}

func TestValidateGeoTimeWindow_InvalidDayOfWeek(t *testing.T) {
	w := GeoTimeWindowInput{
		Countries:  []string{"DE"},
		Action:     "block",
		DaysOfWeek: []int{7},
		HoursStart: 9,
		HoursEnd:   17,
	}
	if err := ValidateGeoTimeWindow(w); err == nil {
		t.Fatal("expected error for day_of_week=7")
	}
}

// --- buildGeoTimeWindowHttpConf ---

func TestBuildGeoTimeWindowHttpConf_Empty(t *testing.T) {
	out := buildGeoTimeWindowHttpConf("test-site", nil)
	if out != "" {
		t.Fatalf("expected empty output for nil windows, got: %s", out)
	}
}

func TestBuildGeoTimeWindowHttpConf_OneWindow(t *testing.T) {
	windows := []GeoTimeWindowInput{
		{Countries: []string{"RU", "CN"}, Action: "block", HoursStart: 9, HoursEnd: 17},
	}
	out := buildGeoTimeWindowHttpConf("test-site", windows)
	if out == "" {
		t.Fatal("expected non-empty http conf for one window")
	}
	// Should contain map blocks
	if !strings.Contains(out, "map $time_iso8601") {
		t.Errorf("expected map block for hour, got:\n%s", out)
	}
	if !strings.Contains(out, "\"RU\" 1;") {
		t.Errorf("expected country RU in map, got:\n%s", out)
	}
	if !strings.Contains(out, "\"CN\" 1;") {
		t.Errorf("expected country CN in map, got:\n%s", out)
	}
}

// --- buildGeoTimeWindowServerSnippet ---

func TestBuildGeoTimeWindowServerSnippet_Empty(t *testing.T) {
	out := buildGeoTimeWindowServerSnippet("test-site", nil, "$waf_easy_exception_guard")
	if out != "" {
		t.Fatalf("expected empty snippet for nil windows, got: %s", out)
	}
}

func TestBuildGeoTimeWindowServerSnippet_BlockAction(t *testing.T) {
	windows := []GeoTimeWindowInput{
		{Countries: []string{"RU"}, Action: "block", HoursStart: 9, HoursEnd: 17},
	}
	out := buildGeoTimeWindowServerSnippet("test-site", windows, "$waf_easy_exception_guard")
	if !strings.Contains(out, "return 403") {
		t.Errorf("expected return 403 for block action, got:\n%s", out)
	}
	// Hour pattern should cover hours 9..16
	if !strings.Contains(out, "09") {
		t.Errorf("expected hour 09 in snippet, got:\n%s", out)
	}
	if !strings.Contains(out, "16") {
		t.Errorf("expected hour 16 in snippet, got:\n%s", out)
	}
	// Hour 17 should NOT be in pattern (exclusive end)
	if strings.Contains(out, "|17|") || strings.Contains(out, "(17)") || strings.Contains(out, "17)$") {
		t.Errorf("hour 17 should not be in hour pattern (exclusive end), got:\n%s", out)
	}
}

func TestBuildGeoTimeWindowServerSnippet_AllowAction_NoReturn403(t *testing.T) {
	windows := []GeoTimeWindowInput{
		{Countries: []string{"DE"}, Action: "allow", HoursStart: 0, HoursEnd: 8},
	}
	out := buildGeoTimeWindowServerSnippet("test-site", windows, "$waf_easy_exception_guard")
	if out == "" {
		t.Fatal("expected non-empty snippet for allow window")
	}
	// allow action does NOT generate return 403
	if strings.Contains(out, "return 403") {
		t.Errorf("expected no return 403 for allow action, got:\n%s", out)
	}
}

// --- normalizeGeoTimeWindows ---

func TestNormalizeGeoTimeWindows_SkipsInvalidWindows(t *testing.T) {
	windows := []GeoTimeWindowInput{
		{Countries: []string{"DE"}, Action: "block", HoursStart: 17, HoursEnd: 9}, // invalid
		{Countries: []string{"FR"}, Action: "block", HoursStart: 9, HoursEnd: 17}, // valid
	}
	out := normalizeGeoTimeWindows(windows)
	if len(out) != 1 {
		t.Fatalf("expected 1 valid window, got %d", len(out))
	}
	if out[0].Countries[0] != "FR" {
		t.Errorf("expected FR window to survive, got %v", out[0].Countries)
	}
}

func TestNormalizeGeoTimeWindows_DeduplicatesCountries(t *testing.T) {
	windows := []GeoTimeWindowInput{
		{Countries: []string{"de", "DE", "de"}, Action: "block", HoursStart: 9, HoursEnd: 17},
	}
	out := normalizeGeoTimeWindows(windows)
	if len(out) != 1 {
		t.Fatalf("expected 1 window")
	}
	if len(out[0].Countries) != 1 || out[0].Countries[0] != "DE" {
		t.Errorf("expected deduplicated [DE], got %v", out[0].Countries)
	}
}
