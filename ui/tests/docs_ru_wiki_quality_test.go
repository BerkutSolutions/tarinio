package tests

import (
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
	reCodeFenceRuDocs = regexp.MustCompile("(?s)```.*?```")
	reInlineCodeRuDocs = regexp.MustCompile("`[^`]*`")
	reMarkdownLinkRuDocs = regexp.MustCompile(`\[[^\]]*\]\([^)]+\)`)
	reUrlRuDocs = regexp.MustCompile(`https?://[^\s)]+`)
	reEmailRuDocs = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
	rePathLikeRuDocs = regexp.MustCompile(`(?m)(^|[\s(])(?:/[A-Za-z0-9._/\-]+|[A-Za-z]:\\[^\s]+)`)
	reTemplateRuDocs = regexp.MustCompile(`\{\{[^{}]+\}\}|\{[^{}]+\}`)
	reEnglishWordRuDocs = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_-]*\b`)
)

var docsRuEnglishWordExceptions = map[string]struct{}{
	"acme":        {},
	"adr":         {},
	"admin":       {},
	"aio":         {},
	"allowlist":   {},
	"anti-bot":    {},
	"anti-ddos":   {},
	"api":         {},
	"asn":         {},
	"auth":        {},
	"bad":         {},
	"basic":       {},
	"behavior":    {},
	"blacklist":   {},
	"blacklists":  {},
	"bundle":      {},
	"ca":          {},
	"cidr":        {},
	"cli":         {},
	"cloudflare":  {},
	"control-plane": {},
	"compose":     {},
	"cookie":      {},
	"cors":        {},
	"csp":         {},
	"cpu":         {},
	"crs":         {},
	"css":         {},
	"cutover":     {},
	"ddos":        {},
	"denylist":    {},
	"dns":         {},
	"dnsbl":       {},
	"dr":          {},
	"easy-profile": {},
	"ed25519":     {},
	"email":       {},
	"env":         {},
	"exceptions":  {},
	"failed":      {},
	"fqdn":        {},
	"geo":         {},
	"github":      {},
	"ha":          {},
	"hcaptcha":    {},
	"host":        {},
	"html":        {},
	"http":        {},
	"https":       {},
	"id":          {},
	"ids":         {},
	"ip":          {},
	"ipv4":        {},
	"ipv6":        {},
	"json":        {},
	"jwt":         {},
	"keepalive":   {},
	"key":         {},
	"l4":          {},
	"l7":          {},
	"limiting":    {},
	"localhost":   {},
	"modsec":      {},
	"modsecurity": {},
	"nginx":       {},
	"oidc":        {},
	"onboarding":  {},
	"origin":      {},
	"owasp":       {},
	"passkey":     {},
	"passkeys":    {},
	"pem":         {},
	"plugin":      {},
	"plugins":     {},
	"postgresql":  {},
	"proxy":       {},
	"rate-limit":  {},
	"raw":         {},
	"rdns":        {},
	"realm":       {},
	"recaptcha":   {},
	"referer":     {},
	"regex":       {},
	"release":     {},
	"request":     {},
	"reverse":     {},
	"rfc3339":     {},
	"san":         {},
	"sbom":        {},
	"score":       {},
	"scim":        {},
	"security":    {},
	"site":        {},
	"sha256":      {},
	"self-signed": {},
	"sni":         {},
	"spa":         {},
	"stage":       {},
	"staged":      {},
	"sql":         {},
	"sso":         {},
	"tcp":         {},
	"tarinio":     {},
	"tls":         {},
	"totp":        {},
	"turnstile":   {},
	"ui":          {},
	"uri":         {},
	"upstream":    {},
	"url":         {},
	"user-agent":  {},
	"vault":       {},
	"waf":         {},
	"websocket":   {},
	"webauthn":    {},
	"wildcard":    {},
	"wiki":        {},
	"x":           {},
	"xss":         {},
	"yaml":        {},
	"zerossl":     {},
}

func TestDocsRuWikiNoMixedEnglish(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	mandatoryFiles := []string{
		filepath.Join(repoRoot, "docs", "ru", "README.md"),
		filepath.Join(repoRoot, "docs", "ru", "index.md"),
		filepath.Join(repoRoot, "docs", "ru", "core-docs", "ui.md"),
		filepath.Join(repoRoot, "docs", "ru", "core-docs", "upgrade.md"),
		filepath.Join(repoRoot, "docs", "ru", "core-docs", "backups.md"),
		filepath.Join(repoRoot, "docs", "ru", "core-docs", "waf-env-reference.md"),
	}
	var issues []string
	for _, path := range mandatoryFiles {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, path+": file is missing")
				continue
			}
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.IsDir() {
			issues = append(issues, path+": expected file, got directory")
			continue
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			issues = append(issues, path+": expected markdown file")
			continue
		}
		checkRuDocFile(t, path, &issues)
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		t.Fatalf("ru wiki contains mixed/english fragments: %v", sample(issues))
	}
}

func checkRuDocFile(t *testing.T, path string, issues *[]string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		*issues = append(*issues, path+": read error: "+err.Error())
		return
	}
	if !utf8.Valid(raw) {
		*issues = append(*issues, path+": invalid UTF-8")
		return
	}
	content := string(raw)
	if strings.ContainsRune(content, unicode.ReplacementChar) || hasMojibakeMarker(content) {
		*issues = append(*issues, path+": contains mojibake/replacement markers")
	}

	sanitized := sanitizeRuDocText(content)
	matches := reEnglishWordRuDocs.FindAllString(sanitized, -1)
	if len(matches) == 0 {
		return
	}
	found := map[string]struct{}{}
	for _, match := range matches {
		word := strings.ToLower(strings.Trim(match, "-_"))
		if _, ok := docsRuEnglishWordExceptions[word]; ok {
			continue
		}
		found[word] = struct{}{}
	}
	if len(found) == 0 {
		return
	}
	words := make([]string, 0, len(found))
	for w := range found {
		words = append(words, w)
	}
	sort.Strings(words)
	*issues = append(*issues, path+": "+strings.Join(words, ", "))
}

func sanitizeRuDocText(content string) string {
	out := reCodeFenceRuDocs.ReplaceAllString(content, " ")
	out = reInlineCodeRuDocs.ReplaceAllString(out, " ")
	out = reMarkdownLinkRuDocs.ReplaceAllString(out, " ")
	out = reUrlRuDocs.ReplaceAllString(out, " ")
	out = reEmailRuDocs.ReplaceAllString(out, " ")
	out = rePathLikeRuDocs.ReplaceAllString(out, " ")
	out = reTemplateRuDocs.ReplaceAllString(out, " ")
	return out
}
