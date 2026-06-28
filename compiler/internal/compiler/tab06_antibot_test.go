package compiler

import (
	"strings"
	"testing"
)

// tab06_antibot_test.go — тесты вкладки 6: Антибот
// Покрывает: режимы challenge (javascript/recaptcha/hcaptcha/turnstile),
// AntibotURI, ScannerAutoBan, ExclusionRules, ChallengeEscalation,
// cookie guard, safe methods, отсутствие блоков при отключённом antibot.

// --- Antibot отключён ---

func TestAntibot_Disabled_NoGuardVars(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-off", EasyProfileInput{
		SiteID:           "ab-off",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "",
	})
	if strings.Contains(conf, "waf_antibot_guard") {
		t.Fatalf("did not expect antibot guard when challenge disabled, got:\n%s", conf)
	}
}

// --- javascript challenge ---

func TestAntibot_Javascript_ChallengeVar(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-js", EasyProfileInput{
		SiteID:           "ab-js",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
	})
	if !strings.Contains(conf, `"javascript"`) {
		t.Fatalf("expected javascript challenge value in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "waf_antibot_guard") {
		t.Fatalf("expected waf_antibot_guard when antibot enabled, got:\n%s", conf)
	}
}

func TestAntibot_Javascript_RedirectURI(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-js-uri", EasyProfileInput{
		SiteID:           "ab-js-uri",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/waf-challenge",
	})
	if !strings.Contains(conf, "/waf-challenge") {
		t.Fatalf("expected antibot URI /waf-challenge in config, got:\n%s", conf)
	}
	// Редирект на challenge при непрошедшей проверке
	if !strings.Contains(conf, "return 302") {
		t.Fatalf("expected return 302 redirect to challenge, got:\n%s", conf)
	}
}

// --- recaptcha challenge ---

func TestAntibot_Recaptcha_ChallengeVar(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-rc", EasyProfileInput{
		SiteID:               "ab-rc",
		SecurityMode:         "block",
		AllowedMethods:       []string{"GET"},
		MaxClientSize:        "10m",
		AntibotChallenge:     "recaptcha",
		AntibotURI:           "/challenge",
		AntibotRecaptchaKey:  "test-recaptcha-key",
		AntibotRecaptchaScore: 0.5,
	})
	if !strings.Contains(conf, `"recaptcha"`) {
		t.Fatalf("expected recaptcha challenge value in config, got:\n%s", conf)
	}
}

// --- hcaptcha challenge ---

func TestAntibot_Hcaptcha_ChallengeVar(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-hc", EasyProfileInput{
		SiteID:           "ab-hc",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "hcaptcha",
		AntibotURI:       "/challenge",
		AntibotHcaptchaKey: "test-hcaptcha-key",
	})
	if !strings.Contains(conf, `"hcaptcha"`) {
		t.Fatalf("expected hcaptcha challenge value in config, got:\n%s", conf)
	}
}

// --- turnstile challenge ---

func TestAntibot_Turnstile_ChallengeVar(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-ts", EasyProfileInput{
		SiteID:             "ab-ts",
		SecurityMode:       "block",
		AllowedMethods:     []string{"GET"},
		MaxClientSize:      "10m",
		AntibotChallenge:   "turnstile",
		AntibotURI:         "/challenge",
		AntibotTurnstileKey: "test-turnstile-key",
	})
	if !strings.Contains(conf, `"turnstile"`) {
		t.Fatalf("expected turnstile challenge value in config, got:\n%s", conf)
	}
}

// --- ScannerAutoBan ---

func TestAntibot_ScannerAutoBan_GuardPresent(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-scan", EasyProfileInput{
		SiteID:                "ab-scan",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		AntibotChallenge:      "javascript",
		AntibotURI:            "/challenge",
		AntibotScannerAutoBan: true,
	})
	if !strings.Contains(conf, "waf_antibot_scanner_guard") {
		t.Fatalf("expected waf_antibot_scanner_guard when ScannerAutoBan=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for scanner ban, got:\n%s", conf)
	}
}

func TestAntibot_ScannerAutoBan_Disabled_NoScannerGuard(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-scan-off", EasyProfileInput{
		SiteID:                "ab-scan-off",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		AntibotChallenge:      "javascript",
		AntibotURI:            "/challenge",
		AntibotScannerAutoBan: false,
	})
	if strings.Contains(conf, "waf_antibot_scanner_guard") {
		t.Fatalf("did not expect scanner guard when ScannerAutoBan=false, got:\n%s", conf)
	}
}

// --- ExclusionRules ---

func TestAntibot_ExclusionRule_BypassesChallenge(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-excl", EasyProfileInput{
		SiteID:           "ab-excl",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
		AntibotExclusionRules: []AntibotExclusionRuleInput{
			{Path: "/api/health", Methods: []string{"GET"}},
		},
	})
	if !strings.Contains(conf, "/api/health") {
		t.Fatalf("expected exclusion pattern /api/health in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "waf_antibot_exception_guard") {
		t.Fatalf("expected waf_antibot_exception_guard for exclusion rule, got:\n%s", conf)
	}
}

// --- Cookie guard ---

func TestAntibot_CookieGuard_VerifiesSession(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-cookie", EasyProfileInput{
		SiteID:           "ab-cookie",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
	})
	// Наличие сессионной куки → verified=1
	if !strings.Contains(conf, "waf_antibot_verified") {
		t.Fatalf("expected waf_antibot_verified cookie check, got:\n%s", conf)
	}
	if !strings.Contains(conf, "waf_session") {
		t.Fatalf("expected waf_session cookie reference, got:\n%s", conf)
	}
}

// --- ChallengeEscalation (two-layer) ---

func TestAntibot_ChallengeEscalation_Enabled(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-esc", EasyProfileInput{
		SiteID:                     "ab-esc",
		SecurityMode:               "block",
		AllowedMethods:             []string{"GET"},
		MaxClientSize:              "10m",
		AntibotChallenge:           "javascript",
		AntibotURI:                 "/challenge",
		ChallengeEscalationEnabled: true,
		ChallengeEscalationMode:    "turnstile",
		AntibotTurnstileKey:        "test-key",
	})
	// При two-layer escalation guard должен быть активен
	if !strings.Contains(conf, "waf_antibot_guard") {
		t.Fatalf("expected waf_antibot_guard in escalation config, got:\n%s", conf)
	}
	// Escalation mode (turnstile) должен присутствовать в конфиге
	if !strings.Contains(conf, `"turnstile"`) {
		t.Fatalf("expected turnstile escalation challenge in config, got:\n%s", conf)
	}
	// X-WAF-Antibot-Provider для turnstile
	if !strings.Contains(conf, "X-WAF-Antibot-Provider") {
		t.Fatalf("expected X-WAF-Antibot-Provider header in escalation config, got:\n%s", conf)
	}
}

func TestAntibot_ChallengeEscalation_WithRules(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-esc-rules", EasyProfileInput{
		SiteID:                     "ab-esc-rules",
		SecurityMode:               "block",
		AllowedMethods:             []string{"GET"},
		MaxClientSize:              "10m",
		AntibotChallenge:           "javascript",
		AntibotURI:                 "/challenge",
		ChallengeEscalationEnabled: true,
		ChallengeEscalationMode:    "recaptcha",
		AntibotRecaptchaKey:        "key",
		AntibotChallengeRules: []AntibotChallengeRuleInput{
			{Path: "/login", Challenge: "recaptcha"},
		},
	})
	if !strings.Contains(conf, "/login") {
		t.Fatalf("expected challenge rule pattern /login in config, got:\n%s", conf)
	}
}

// --- Guard logic: unverified non-safe → 302 or 403 ---

func TestAntibot_UnverifiedRequest_RedirectOrBlock(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-block", EasyProfileInput{
		SiteID:           "ab-block",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
	})
	// Непрошедший запрос → 302 (redirect) или 403 (block mode)
	if !strings.Contains(conf, "return 302") && !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 302 or 403 for unverified request, got:\n%s", conf)
	}
}

// --- X-WAF-Antibot-Mode header ---

func TestAntibot_DebugHeader_Present(t *testing.T) {
	conf := mustRenderSiteConf(t, "ab-hdr", EasyProfileInput{
		SiteID:           "ab-hdr",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "10m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
	})
	if !strings.Contains(conf, "X-WAF-Antibot-Mode") {
		t.Fatalf("expected X-WAF-Antibot-Mode debug header in config, got:\n%s", conf)
	}
}
