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
	"acme":           {},
	"all_sites":      {},
	"allowlist":      {},
	"anti-ddos":      {},
	"anti-bot":       {},
	"api":            {},
	"as13335":        {},
	"auth":           {},
	"admin":          {},
	"app":            {},
	"app-example-com": {},
	"apply":          {},
	"asn":            {},
	"audit":          {},
	"backend":        {},
	"balanced":       {},
	"basic_or_token": {},
	"behavior":       {},
	"berkut":         {},
	"berkutsolutions": {},
	"basic":          {},
	"blacklist":      {},
	"block":          {},
	"botnet":         {},
	"body":           {},
	"burst":          {},
	"camera":         {},
	"captcha":        {},
	"cdn":            {},
	"challenge":      {},
	"check":          {},
	"change-password": {},
	"cli":            {},
	"cidr":           {},
	"codes":          {},
	"cloudflare":     {},
	"compat":         {},
	"compile":        {},
	"com":            {},
	"compose":        {},
	"connect":        {},
	"content-security-policy": {},
	"content-type":   {},
	"control-plane":  {},
	"cookie":         {},
	"cookie-challenge": {},
	"cookies":        {},
	"core":           {},
	"cors":           {},
	"count":          {},
	"cpu":            {},
	"crs":            {},
	"crud":           {},
	"csp":            {},
	"curl":           {},
	"current_site":   {},
	"clickhouse":     {},
	"created":        {},
	"creating":       {},
	"custom":         {},
	"database":       {},
	"date":           {},
	"ddos":           {},
	"debug":          {},
	"delete":         {},
	"deploy":         {},
	"denylist":       {},
	"details":        {},
	"dev":            {},
	"dns":            {},
	"dns-01":         {},
	"dnsbl":          {},
	"docker":         {},
	"down":           {},
	"dry-run":        {},
	"eab":            {},
	"easy":           {},
	"email":          {},
	"encrypt":        {},
	"endpoint":       {},
	"env":            {},
	"example":        {},
	"exceptions":     {},
	"failed":         {},
	"fallback":       {},
	"g20":            {},
	"geoip":          {},
	"geolocation":    {},
	"get":            {},
	"github":         {},
	"hashicorp":      {},
	"head":           {},
	"headers":        {},
	"healthcheck":    {},
	"health-check":   {},
	"headless":       {},
	"heap":           {},
	"hcaptcha":       {},
	"hmac":           {},
	"hostname":       {},
	"hot":            {},
	"hotdays":        {},
	"hsts":           {},
	"http":           {},
	"https":          {},
	"httponly":       {},
	"id":             {},
	"ingest":         {},
	"ingest-":        {},
	"ingress":        {},
	"import":         {},
	"internal":       {},
	"io":             {},
	"ip":             {},
	"ips":            {},
	"is":             {},
	"javascript":     {},
	"jobs":           {},
	"job":            {},
	"json":           {},
	"jwt":            {},
	"keep-alive":     {},
	"key":            {},
	"keypass":        {},
	"kid":            {},
	"kv":             {},
	"l4":             {},
	"l7":             {},
	"let":            {},
	"letsencrypt":    {},
	"lets":           {},
	"limit":          {},
	"live":           {},
	"local":          {},
	"localhost":      {},
	"login":          {},
	"m":              {},
	"max-age":        {},
	"microphone":     {},
	"model":          {},
	"material":       {},
	"mcaptcha":       {},
	"monitor":        {},
	"ms":             {},
	"mode":           {},
	"name":           {},
	"notice":         {},
	"modsec":         {},
	"modsecurity":    {},
	"nginx":          {},
	"no":             {},
	"no-referrer":    {},
	"offset":         {},
	"oidc":           {},
	"options":        {},
	"owasp":          {},
	"opensearch":     {},
	"passkey":        {},
	"passkeys":       {},
	"password":       {},
	"patch":          {},
	"payload":        {},
	"pem":            {},
	"permissions":    {},
	"permissions-policy": {},
	"php":            {},
	"policy":         {},
	"post":           {},
	"preload":        {},
	"probe":          {},
	"process":        {},
	"proxy":          {},
	"public-edge":    {},
	"push":           {},
	"push-":          {},
	"put":            {},
	"qr":             {},
	"quic":           {},
	"rate":           {},
	"raw":            {},
	"recaptcha":      {},
	"redirect":       {},
	"referer":        {},
	"referrer":       {},
	"referrer-policy": {},
	"reject":         {},
	"request":        {},
	"rest":           {},
	"retention":      {},
	"reverse-proxy":  {},
	"revision":       {},
	"revisionid":     {},
	"roles":          {},
	"rfc3339":        {},
	"rule":           {},
	"rps":            {},
	"runtime":        {},
	"s":              {},
	"samesite":       {},
	"san":            {},
	"scc":            {},
	"scim":           {},
	"score":          {},
	"score-based":    {},
	"script-src":     {},
	"secretprovider": {},
	"secret":         {},
	"secrule":        {},
	"secure":         {},
	"self":           {},
	"service":        {},
	"service_token":  {},
	"service-a-tls":  {},
	"saved":          {},
	"security":       {},
	"set-cookie":     {},
	"shodan":         {},
	"shop-prod":      {},
	"single-source":  {},
	"site":           {},
	"sitewide":       {},
	"slowloris":      {},
	"sni":            {},
	"soc":            {},
	"snapshot":       {},
	"spamhaus":       {},
	"sso":            {},
	"spa":            {},
	"sql":            {},
	"set":            {},
	"ssl":            {},
	"solutions":      {},
	"staging":        {},
	"status":         {},
	"step":           {},
	"strict":         {},
	"strict-origin":  {},
	"strict-origin-when-cross-origin": {},
	"strict-transport-security": {},
	"table":          {},
	"threshold":      {},
	"threads":        {},
	"cold":           {},
	"colddays":       {},
	"tcp":            {},
	"tarinio":        {},
	"throttle":       {},
	"tls":            {},
	"tlsv1":          {},
	"total":          {},
	"totp":           {},
	"trace":          {},
	"transparent":    {},
	"udp":            {},
	"udp-":           {},
	"up":             {},
	"ui":             {},
	"host":           {},
	"upstream":       {},
	"uri":            {},
	"url":            {},
	"user":           {},
	"user-agent":     {},
	"users":          {},
	"v3":             {},
	"value":          {},
	"version":        {},
	"username":       {},
	"vault":          {},
	"waf":            {},
	"webauthn":       {},
	"webhook":        {},
	"webhooks":       {},
	"webshell":       {},
	"websocket":      {},
	"wildcard":       {},
	"wordpress":      {},
	"wordpress-rule-exclusions": {},
	"wp-admin":       {},
	"x":              {},
	"x-forwarded-for": {},
	"x-forwarded-proto": {},
	"x-real-ip":      {},
	"x-request-id":   {},
	"turnstile":      {},
	"xss":            {},
	"zerossl":        {},
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
		if lang == "ru" && countEnglishWordsQuality(value) > 0 {
			issues = append(issues, lang+":"+key+": contains english words")
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
