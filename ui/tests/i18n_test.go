package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

var (
	reDataI18n            = regexp.MustCompile(`data-i18n="([^"]+)"`)
	reDataI18nPlaceholder = regexp.MustCompile(`data-i18n-placeholder="([^"]+)"`)
	reDataI18nTitle       = regexp.MustCompile(`data-i18n-title="([^"]+)"`)
	reDataI18nAria        = regexp.MustCompile(`data-i18n-aria-label="([^"]+)"`)
	reLocalT              = regexp.MustCompile(`\bt\(\s*["']([^"']+)["']`)
	reUnicodeEscape       = regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
)

var mojibakeMarkers = []string{
	"\uFFFD", // Unicode replacement character.
	"�",
	"Ð",
	"Ñ",
	"â€",
	"Р ",
	"РЋ",
	"С™",
	"вЂ",
}

func TestI18NKeyCoverage(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "en.json"))

	if missing := diffKeys(ru, en); len(missing) > 0 {
		t.Fatalf("missing keys in en.json: %v", sample(missing))
	}
	if missing := diffKeys(en, ru); len(missing) > 0 {
		t.Fatalf("missing keys in ru.json: %v", sample(missing))
	}

	used := collectUsedKeys(t, filepath.Join("..", "app"))
	var missing []string
	for _, key := range used {
		if _, ok := ru[key]; !ok {
			missing = append(missing, key+" (ru)")
		}
		if _, ok := en[key]; !ok {
			missing = append(missing, key+" (en)")
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing used i18n keys: %v", sample(missing))
	}
}

func TestI18N_RuContainsAllEnKeys(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "en.json"))

	var missing []string
	for key := range en {
		if _, ok := ru[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("ru.json is missing keys present in en.json: %v", sample(missing))
	}
}

func TestI18NValuesNonEmpty(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "en.json"))

	var empty []string
	for key, value := range ru {
		if strings.TrimSpace(value) == "" {
			empty = append(empty, "ru:"+key)
		}
	}
	for key, value := range en {
		if strings.TrimSpace(value) == "" {
			empty = append(empty, "en:"+key)
		}
	}
	if len(empty) > 0 {
		sort.Strings(empty)
		t.Fatalf("empty i18n values: %v", sample(empty))
	}
}

func TestI18NRuFileNoEscapedUnicodeAndValidUTF8(t *testing.T) {
	path := filepath.Join("..", "app", "static", "i18n", "ru.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !utf8.Valid(raw) {
		t.Fatalf("ru.json is not valid UTF-8")
	}
	if reUnicodeEscape.Match(raw) {
		t.Fatalf("ru.json contains \\uXXXX escape sequences; keep real Cyrillic characters in file")
	}
}

func TestI18NRuValuesNoMojibake(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))

	var broken []string
	for key, value := range ru {
		if hasMojibakeMarker(value) || hasUnexpectedCyrillicRune(value) {
			broken = append(broken, key)
		}
	}
	if len(broken) > 0 {
		sort.Strings(broken)
		t.Fatalf("ru i18n values contain mojibake or invalid cyrillic runes: %v", sample(broken))
	}
}

func mustLoadLang(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return out
}

func collectUsedKeys(t *testing.T, root string) []string {
	t.Helper()
	seen := map[string]struct{}{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.Contains(path, string(filepath.Separator)+"i18n") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".js" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(raw)
		addMatches(content, reDataI18n, seen)
		addMatches(content, reDataI18nPlaceholder, seen)
		addMatches(content, reDataI18nTitle, seen)
		addMatches(content, reDataI18nAria, seen)
		addMatches(content, reLocalT, seen)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func addMatches(content string, re *regexp.Regexp, seen map[string]struct{}) {
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		key := strings.TrimSpace(match[1])
		if key != "" {
			seen[key] = struct{}{}
		}
	}
}

func diffKeys(base, compare map[string]string) []string {
	out := make([]string, 0)
	for key := range base {
		if _, ok := compare[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func sample(items []string) []string {
	if len(items) <= 20 {
		return items
	}
	return append(items[:20], "...")
}

func hasMojibakeMarker(value string) bool {
	for _, marker := range mojibakeMarkers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func hasUnexpectedCyrillicRune(value string) bool {
	for _, r := range value {
		// Fast-path: Latin, punctuation, digits, spaces are all fine.
		if r <= unicode.MaxASCII {
			continue
		}
		// Only care about Cyrillic block runes.
		if r < 0x0400 || r > 0x04FF {
			continue
		}
		// Allowed Russian letters: А-Я, а-я, Ё, ё.
		if (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё' {
			continue
		}
		return true
	}
	return false
}
