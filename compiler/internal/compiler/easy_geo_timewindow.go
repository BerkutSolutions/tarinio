package compiler

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GeoTimeWindowInput is the compiler input for one time-based geo-fencing rule.
type GeoTimeWindowInput struct {
	Countries  []string // ISO 3166-1 alpha-2 codes, upper-cased
	Action     string   // "block" | "allow"
	DaysOfWeek []int    // 0=Sunday … 6=Saturday; empty means every day
	HoursStart int      // 0-23 UTC inclusive
	HoursEnd   int      // 0-23 UTC exclusive (must be > HoursStart)
}

var isoCountryRe = regexp.MustCompile(`^[A-Z]{2}$`)

// ValidateGeoTimeWindow returns an error if the window is misconfigured.
func ValidateGeoTimeWindow(w GeoTimeWindowInput) error {
	if w.Action != "block" && w.Action != "allow" {
		return fmt.Errorf("geo_time_window: action must be \"block\" or \"allow\", got %q", w.Action)
	}
	if w.HoursStart < 0 || w.HoursStart > 23 {
		return fmt.Errorf("geo_time_window: hours_start must be 0-23, got %d", w.HoursStart)
	}
	if w.HoursEnd < 0 || w.HoursEnd > 23 {
		return fmt.Errorf("geo_time_window: hours_end must be 0-23, got %d", w.HoursEnd)
	}
	if w.HoursStart >= w.HoursEnd {
		return fmt.Errorf("geo_time_window: hours_start (%d) must be less than hours_end (%d)", w.HoursStart, w.HoursEnd)
	}
	for _, d := range w.DaysOfWeek {
		if d < 0 || d > 6 {
			return fmt.Errorf("geo_time_window: days_of_week value must be 0-6, got %d", d)
		}
	}
	for _, c := range w.Countries {
		upper := strings.ToUpper(strings.TrimSpace(c))
		if !isoCountryRe.MatchString(upper) {
			return fmt.Errorf("geo_time_window: invalid country code %q", c)
		}
	}
	return nil
}

// normalizeGeoTimeWindows deduplicates country codes and discards invalid windows.
func normalizeGeoTimeWindows(windows []GeoTimeWindowInput) []GeoTimeWindowInput {
	if len(windows) == 0 {
		return nil
	}
	out := make([]GeoTimeWindowInput, 0, len(windows))
	for _, w := range windows {
		if err := ValidateGeoTimeWindow(w); err != nil {
			continue
		}
		seen := map[string]struct{}{}
		countries := make([]string, 0, len(w.Countries))
		for _, c := range w.Countries {
			upper := strings.ToUpper(strings.TrimSpace(c))
			if upper == "" {
				continue
			}
			if _, ok := seen[upper]; ok {
				continue
			}
			seen[upper] = struct{}{}
			countries = append(countries, upper)
		}
		sort.Strings(countries)
		if len(countries) == 0 {
			continue
		}
		days := append([]int(nil), w.DaysOfWeek...)
		sort.Ints(days)
		out = append(out, GeoTimeWindowInput{
			Countries:  countries,
			Action:     w.Action,
			DaysOfWeek: days,
			HoursStart: w.HoursStart,
			HoursEnd:   w.HoursEnd,
		})
	}
	return out
}

// buildGeoTimeWindowHttpConf generates http-context map directives.
// Returns empty string when there are no valid windows.
// The generated maps derive $waf_geo_hour from $time_iso8601 and per-window
// country-match variables from $waf_country_code.
func buildGeoTimeWindowHttpConf(siteID string, windows []GeoTimeWindowInput) string {
	windows = normalizeGeoTimeWindows(windows)
	if len(windows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Map: $time_iso8601 → two-digit hour string "00".."23".
	// $time_iso8601 looks like "2006-01-02T15:04:05+00:00"; chars 11-12 are hour.
	// We match the prefix up to the hour with a regex.
	hourVar := fmt.Sprintf("$waf_geo_%s_hour", slugSiteID(siteID))
	sb.WriteString(fmt.Sprintf("map $time_iso8601 %s {\n", hourVar))
	sb.WriteString("    default \"-\";\n")
	for h := 0; h <= 23; h++ {
		sb.WriteString(fmt.Sprintf("    ~^\\d{4}-\\d{2}-\\d{2}T%02d \"%02d\";\n", h, h))
	}
	sb.WriteString("}\n\n")

	// Per-window country-match map.
	for i, w := range windows {
		countryVar := fmt.Sprintf("$waf_geo_tw_%s_%d_country", slugSiteID(siteID), i)
		sb.WriteString(fmt.Sprintf("map $waf_country_code %s {\n", countryVar))
		sb.WriteString("    default 0;\n")
		for _, c := range w.Countries {
			sb.WriteString(fmt.Sprintf("    \"%s\" 1;\n", c))
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// buildGeoTimeWindowServerSnippet generates the server-context if-guards.
// exceptionVar is the site exception variable name (without $).
func buildGeoTimeWindowServerSnippet(siteID string, windows []GeoTimeWindowInput, exceptionVar string) string {
	windows = normalizeGeoTimeWindows(windows)
	if len(windows) == 0 {
		return ""
	}

	slug := slugSiteID(siteID)
	hourVar := fmt.Sprintf("$waf_geo_%s_hour", slug)
	var sb strings.Builder
	sb.WriteString("# geo time-window enforcement\n")

	for i, w := range windows {
		countryVar := fmt.Sprintf("$waf_geo_tw_%s_%d_country", slug, i)
		activeVar := fmt.Sprintf("$waf_geo_tw_%s_%d_active", slug, i)
		guardVar := fmt.Sprintf("$waf_geo_tw_%s_%d_guard", slug, i)

		// Build the hour regex.
		hourParts := make([]string, 0, w.HoursEnd-w.HoursStart)
		for h := w.HoursStart; h < w.HoursEnd; h++ {
			hourParts = append(hourParts, fmt.Sprintf("%02d", h))
		}
		hourPattern := "^(?:" + strings.Join(hourParts, "|") + ")$"

		sb.WriteString(fmt.Sprintf("set %s 0;\n", activeVar))
		sb.WriteString(fmt.Sprintf("if (%s = 1) {\n", countryVar))
		sb.WriteString(fmt.Sprintf("    if (%s ~ \"%s\") {\n", hourVar, hourPattern))
		sb.WriteString(fmt.Sprintf("        set %s 1;\n", activeVar))
		sb.WriteString("    }\n")
		sb.WriteString("}\n")

		if w.Action == "block" {
			sb.WriteString(fmt.Sprintf("set %s \"${%s}:${%s}\";\n",
				guardVar,
				strings.TrimPrefix(exceptionVar, "$"),
				strings.TrimPrefix(activeVar, "$"),
			))
			sb.WriteString(fmt.Sprintf("if (%s = \"0:1\") { return 403; }\n", guardVar))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
