package tests

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

var (
	reManyQuestionMarksQuality = regexp.MustCompile(`\?{3,}`)
	reHasCyrillicQuality       = regexp.MustCompile(`[\p{Cyrillic}]`)
	reMojibakeMarkersQuality   = regexp.MustCompile(`(?:Ð|Ñ|â€”|â€“|â€|â„|вЂ|Р[A-Za-z]|С[A-Za-z])`)
	reEnglishWordQuality       = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_-]*\b`)
)

// Technical terms allowed in RU localization values.
var englishWordExceptionsQuality = map[string]struct{}{
	"acme":          {},
	"api":           {},
	"auth":          {},
	"autostart":     {},
	"accesspolicy":  {},
	"admin":         {},
	"and":           {},
	"app":           {},
	"available":     {},
	"behavior":      {},
	"berkut":        {},
	"blacklist":     {},
	"botnet":        {},
	"burst":         {},
	"cdn":           {},
	"challenge":     {},
	"check":         {},
	"cidr":          {},
	"codes":         {},
	"cloudflare":    {},
	"compile":       {},
	"content":       {},
	"control-plane": {},
	"cors":          {},
	"core":          {},
	"crs":           {},
	"created":       {},
	"ddos":          {},
	"denylist":      {},
	"dns":           {},
	"drop":          {},
	"eab":           {},
	"easy":          {},
	"encrypt":       {},
	"example":       {},
	"failed":        {},
	"geoip":         {},
	"github":        {},
	"headers":       {},
	"http":          {},
	"https":         {},
	"id":            {},
	"ip":            {},
	"ips":           {},
	"is":            {},
	"job":           {},
	"jwt":           {},
	"l4":            {},
	"l7":            {},
	"let":           {},
	"letsencrypt":   {},
	"localhost":     {},
	"manual":        {},
	"material":      {},
	"metadata":      {},
	"mode":          {},
	"modsec":        {},
	"modsecurity":   {},
	"nginx":         {},
	"no":            {},
	"owasp":         {},
	"policy":        {},
	"proxy":         {},
	"rate":          {},
	"reject":        {},
	"required":      {},
	"revision":      {},
	"rule":          {},
	"rps":           {},
	"runtime":       {},
	"s":             {},
	"saved":         {},
	"security":      {},
	"site":          {},
	"sni":           {},
	"spa":           {},
	"sql":           {},
	"set":           {},
	"ssl":           {},
	"solutions":     {},
	"snapshot":      {},
	"status":        {},
	"succeeded":     {},
	"tcp":           {},
	"tarinio":       {},
	"throttle":      {},
	"tls":           {},
	"total":         {},
	"ui":            {},
	"unavailable":   {},
	"update":        {},
	"updates":       {},
	"upload":        {},
	"uploaded":      {},
	"upstream":      {},
	"username":      {},
	"url":           {},
	"version":       {},
	"waf":           {},
	"xss":           {},
	"zerossl":       {},
}

func TestI18NNoArtifacts(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "en.json"))

	var issues []string
	check := func(lang, key, value string) {
		if strings.ContainsRune(value, unicode.ReplacementChar) || strings.Contains(value, "\uFFFD") {
			issues = append(issues, lang+":"+key+": contains replacement char")
		}
		if hasSuspiciousControlCharsQuality(value) {
			issues = append(issues, lang+":"+key+": contains control chars")
		}
		if reManyQuestionMarksQuality.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains ???-like sequence")
		}
		if lang == "ru" && containsForbiddenRuCyrillicQuality(value) {
			issues = append(issues, lang+":"+key+": contains suspicious Cyrillic letters (likely mojibake)")
		}
		if lang == "ru" && countEnglishWordsQuality(value) > 2 {
			issues = append(issues, lang+":"+key+": contains more than 2 english words")
		}
		if reMojibakeMarkersQuality.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains mojibake marker sequence")
		}
		if lang == "en" && reHasCyrillicQuality.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains Cyrillic characters")
		}
	}

	for k, v := range ru {
		check("ru", k, v)
	}
	for k, v := range en {
		check("en", k, v)
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		t.Fatalf("i18n artifacts found: %v", sample(issues))
	}
}

func hasSuspiciousControlCharsQuality(s string) bool {
	for _, r := range s {
		if !unicode.IsControl(r) {
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		return true
	}
	return false
}

func containsForbiddenRuCyrillicQuality(s string) bool {
	forbidden := map[rune]struct{}{
		'Ѐ': {}, 'ѐ': {},
		'Ѓ': {}, 'ѓ': {},
		'Є': {}, 'є': {},
		'Ѕ': {}, 'ѕ': {},
		'І': {}, 'і': {},
		'Ї': {}, 'ї': {},
		'Ј': {}, 'ј': {},
		'Љ': {}, 'љ': {},
		'Њ': {}, 'њ': {},
		'Ћ': {}, 'ћ': {},
		'Ќ': {}, 'ќ': {},
		'Ѝ': {}, 'ѝ': {},
		'Ў': {}, 'ў': {},
		'Џ': {}, 'џ': {},
		'Ґ': {}, 'ґ': {},
	}
	for _, r := range s {
		if _, ok := forbidden[r]; ok {
			return true
		}
	}
	return false
}

func countEnglishWordsQuality(s string) int {
	matches := reEnglishWordQuality.FindAllString(s, -1)
	count := 0
	for _, match := range matches {
		word := strings.ToLower(strings.Trim(match, "-_"))
		if _, ok := englishWordExceptionsQuality[word]; ok {
			continue
		}
		count++
	}
	return count
}
