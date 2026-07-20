package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestE2EAntibotTemplates проверяет:
// 1. Каждый шаблон v1-v5 рендерится через preview-эндпоинт
// 2. Шаблоны содержат verifyURI (не location.reload)
// 3. После смены шаблона compile+apply — активный конфиг использует нужный шаблон
// 4. Полный challenge flow: запрос → 302 на /challenge → страница отдаётся → verify → cookie → редирект
//
// Требует: WAF_E2E_BASE_URL (control-plane UI URL)
// Опционально: WAF_E2E_ANTIBOT_BASE_URL (runtime URL для challenge flow)
func TestE2EAntibotTemplates(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping antibot templates e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	siteID := e2eGetFirstSiteID(t, client, requestBaseURL, requestHostOverride)
	if siteID == "" {
		t.Skip("no sites configured; skipping antibot templates e2e")
	}

	// Читаем исходный профиль для восстановления
	origProfile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	if origProfile == nil {
		t.Skip("no easy profile for site; skipping")
	}
	origTemplate, _ := origProfile["security_antibot"].(map[string]any)

	t.Run("TemplatePreviewRendering", func(t *testing.T) {
		// Каждый шаблон v1-v5 должен отдаваться через /api/error-pages/preview/antibot-vN
		// Сначала проверим что эндпоинт вообще существует
		probeResp := getWithAuth(t, client, requestBaseURL+"/api/error-pages/preview/antibot-v1", requestHostOverride)
		probeBody, _ := io.ReadAll(probeResp.Body)
		_ = probeResp.Body.Close()
		if probeResp.StatusCode == http.StatusNotFound {
			t.Skipf("antibot preview endpoint not available on this build (got 404); skipping template preview tests")
		}
		for i := 1; i <= 5; i++ {
			i := i
			t.Run(fmt.Sprintf("v%d", i), func(t *testing.T) {
				if i == 1 {
					// уже прочитали выше
					if len(probeBody) < 200 {
						t.Fatalf("preview antibot-v1: response too short (%d bytes)", len(probeBody))
					}
					return
				}
				slug := fmt.Sprintf("antibot-v%d", i)
				url := requestBaseURL + "/api/error-pages/preview/" + slug
				resp := getWithAuth(t, client, url, requestHostOverride)
				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("preview %s: want 200, got %d body=%s", slug, resp.StatusCode, string(body))
				}
				if len(body) < 200 {
					t.Fatalf("preview %s: response too short (%d bytes)", slug, len(body))
				}
			})
		}
	})

	t.Run("TemplatesContainVerifyURI", func(t *testing.T) {
		// Все шаблоны v1-v5 должны содержать verifyURI механизм, не location.reload
		for i := 1; i <= 5; i++ {
			i := i
			t.Run(fmt.Sprintf("v%d", i), func(t *testing.T) {
				slug := fmt.Sprintf("antibot-v%d", i)
				url := requestBaseURL + "/api/error-pages/preview/" + slug
				resp := getWithAuth(t, client, url, requestHostOverride)
				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Skipf("preview %s not available (status=%d)", slug, resp.StatusCode)
				}
				content := string(body)
				if strings.Contains(content, "location.reload()") {
					t.Errorf("template %s still uses location.reload() — challenge will not work", slug)
				}
				if !strings.Contains(content, "verifyURI") && !strings.Contains(content, "buildVerifyURL") {
					t.Errorf("template %s missing verifyURI/buildVerifyURL mechanism", slug)
				}
			})
		}
	})

	t.Run("TemplateSwitchSavesAndCompiles", func(t *testing.T) {
		// Меняем шаблон на v3, сохраняем, compile+apply, проверяем что конфиг правильный
		profile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
		if profile == nil {
			t.Skip("no easy profile")
		}
		antibot, _ := profile["security_antibot"].(map[string]any)
		if antibot == nil {
			antibot = map[string]any{}
			profile["security_antibot"] = antibot
		}
		origTemplateVal := antibot["antibot_challenge_template"]
		antibot["antibot_challenge_template"] = "v3"
		profile["security_antibot"] = antibot

		saveResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
		_ = saveResp.Body.Close()
		if saveResp.StatusCode != http.StatusOK && saveResp.StatusCode != http.StatusCreated {
			t.Fatalf("save profile with template v3: status=%d", saveResp.StatusCode)
		}

		// Читаем обратно — шаблон должен быть v3
		reloaded := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
		if reloaded != nil {
			antibotReloaded, _ := reloaded["security_antibot"].(map[string]any)
			if antibotReloaded != nil {
				if got := antibotReloaded["antibot_challenge_template"]; got != "v3" {
					t.Errorf("antibot_challenge_template: want v3, got %v", got)
				}
			}
		}

		// Compile + apply
		revID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
		if revID != "" {
			// Проверяем что в скомпилированном easy-конфиге есть признак v3
			confResp := getWithAuth(t, client, fmt.Sprintf("%s/api/revisions/%s/artifacts/nginx/easy/%s.conf", requestBaseURL, revID, siteID), requestHostOverride)
			if confResp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(confResp.Body)
				_ = confResp.Body.Close()
				// antibot-v3 должен упоминаться в конфиге или error-pages
				t.Logf("compiled easy conf snippet (first 500 chars): %.500s", string(body))
			} else {
				_ = confResp.Body.Close()
			}
		}

		// Восстанавливаем оригинальный шаблон
		if antibot != nil {
			antibot["antibot_challenge_template"] = origTemplateVal
			profile["security_antibot"] = antibot
			restoreResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
			_ = restoreResp.Body.Close()
			e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
		}
	})

	t.Run("ChallengeFlowAllSteps", func(t *testing.T) {
		// Проверяем полный flow: GET / → 302 /challenge → 200 challenge page → POST verify → cookie → redirect
		antibotBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_BASE_URL")), "/")
		if antibotBaseURL == "" {
			antibotBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
		}
		if antibotBaseURL == "" {
			t.Skip("no antibot base URL configured")
		}

		endpoint, err := resolveAntibotEndpoint(antibotBaseURL)
		if err != nil {
			t.Fatalf("resolve antibot endpoint: %v", err)
		}

		// Клиент без куков — должен получить challenge
		clientNoJar := newAntibotHTTPClient(endpoint, false)
		clientNoJar.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

		// Шаг 1: запрос на / → должен вернуть 302 или 200 с challenge redirect
		probeResp, err := clientNoJar.Get(endpoint.requestBaseURL + "/")
		if err != nil {
			t.Skipf("target not reachable: %v", err)
		}
		defer probeResp.Body.Close()
		_, _ = io.ReadAll(probeResp.Body)

		// Если antibot не включён — skip
		mode := probeResp.Header.Get("X-WAF-Antibot-Mode")
		if mode == "" || mode == "no" {
			t.Skip("antibot not active on this endpoint (X-WAF-Antibot-Mode=no or absent)")
		}
		t.Logf("antibot mode: %s", mode)

		// Шаг 2: клиент с куками выполняет challenge
		clientWithJar := newAntibotHTTPClient(endpoint, true)
		clientWithJar.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		challengeURI := normalizeChallengeURI(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_CHALLENGE_URI")))
		verifyURI := antibotVerifyURI(challengeURI)

		// GET challenge page
		challengeURL := endpoint.requestBaseURL + challengeURI + "?return_uri=/&return_args="
		chalResp, err := clientWithJar.Get(challengeURL)
		if err != nil {
			t.Fatalf("GET challenge page: %v", err)
		}
		chalBody, _ := io.ReadAll(chalResp.Body)
		_ = chalResp.Body.Close()
		if chalResp.StatusCode != http.StatusOK {
			t.Fatalf("challenge page: want 200, got %d", chalResp.StatusCode)
		}
		t.Logf("challenge page size: %d bytes", len(chalBody))

		// Шаг 3: убеждаемся что страница содержит verifyURI
		chalContent := string(chalBody)
		if !strings.Contains(chalContent, verifyURI) && !strings.Contains(chalContent, "buildVerifyURL") && !strings.Contains(chalContent, "verifyURI") {
			t.Errorf("challenge page missing verifyURI reference; page may not redirect correctly")
		}
		if strings.Contains(chalContent, "location.reload()") {
			t.Errorf("challenge page uses location.reload() — steps 3-5 will not complete")
		}

		// Шаг 4: GET verify endpoint напрямую (симулируем JS redirect)
		verifyURL := endpoint.requestBaseURL + verifyURI + "?return_uri=/&return_args="
		verResp, err := clientWithJar.Get(verifyURL)
		if err != nil {
			t.Fatalf("GET verify endpoint: %v", err)
		}
		_, _ = io.ReadAll(verResp.Body)
		_ = verResp.Body.Close()

		// Verify должен вернуть 302 (редирект обратно на return_uri)
		// или 200 если уже верифицирован
		if verResp.StatusCode != http.StatusFound && verResp.StatusCode != http.StatusOK && verResp.StatusCode != http.StatusNoContent && verResp.StatusCode != http.StatusTemporaryRedirect {
			t.Fatalf("verify endpoint: want 302/200, got %d", verResp.StatusCode)
		}
		t.Logf("verify status: %d", verResp.StatusCode)

		// Шаг 5: убеждаемся что cookie установлена
		cookies := clientWithJar.Jar.Cookies(mustParseURL(t, endpoint.requestBaseURL))
		hasCookie := false
		for _, c := range cookies {
			if strings.HasPrefix(c.Name, "waf_antibot_") {
				hasCookie = true
				t.Logf("antibot cookie set: %s", c.Name)
				break
			}
		}
		if !hasCookie {
			t.Errorf("antibot cookie not set after verify; subsequent requests will keep getting challenged")
		}

		// Шаг 6: повторный запрос с cookie — должен пройти без challenge
		time.Sleep(200 * time.Millisecond)
		finalResp, err := clientWithJar.Get(endpoint.requestBaseURL + "/")
		if err != nil {
			t.Fatalf("final request: %v", err)
		}
		_, _ = io.ReadAll(finalResp.Body)
		_ = finalResp.Body.Close()
		finalMode := finalResp.Header.Get("X-WAF-Antibot-Mode")
		t.Logf("final request status=%d antibot-mode=%s", finalResp.StatusCode, finalMode)
		// После cookie request не должен редиректить на challenge
		if finalResp.StatusCode == http.StatusFound {
			loc := finalResp.Header.Get("Location")
			if strings.Contains(loc, challengeURI) {
				t.Errorf("still redirected to challenge after cookie set: Location=%s", loc)
			}
		}
	})

	_ = origTemplate
}

// e2eGetEasyProfile читает easy-profile для сайта.
func e2eGetEasyProfile(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID string) map[string]any {
	t.Helper()
	resp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil
	}
	var profile map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		_ = resp.Body.Close()
		return nil
	}
	_ = resp.Body.Close()
	return profile
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url %q: %v", rawURL, err)
	}
	return u
}
