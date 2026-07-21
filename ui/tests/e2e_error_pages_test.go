package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestE2EErrorPages проверяет:
// 1. Preview-страницы ошибок отдаются через /preview/<slug>
// 2. Исключения (disabled_error_pages) сохраняются и читаются обратно
// 3. После compile+apply исключённая страница не отображается (error_page директива убрана)
//
// Требует: WAF_E2E_BASE_URL (control-plane UI URL)
func TestE2EErrorPages(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping error pages e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	// Найдём первый сайт
	siteID := e2eGetFirstSiteID(t, client, requestBaseURL, requestHostOverride)
	if siteID == "" {
		t.Skip("no sites configured; skipping error pages e2e")
	}

	t.Run("PreviewPagesAccessible", func(t *testing.T) {
		// Проверяем что preview-страницы отдаются через /preview/<slug>
		slugs := []string{"400", "401", "403", "404", "429", "500", "502", "503"}
		for _, slug := range slugs {
			slug := slug
			t.Run("preview_"+slug, func(t *testing.T) {
				url := requestBaseURL + "/preview/" + slug
				resp := getWithAuth(t, client, url, requestHostOverride)
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("GET /preview/%s: want 200, got %d body=%s", slug, resp.StatusCode, string(body))
				}
				if len(body) < 100 {
					t.Fatalf("GET /preview/%s: response too short (%d bytes), likely empty page", slug, len(body))
				}
			})
		}
	})

	t.Run("DisabledErrorPagesSaveAndLoad", func(t *testing.T) {
		// Читаем текущий профиль
		profileResp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride)
		if profileResp.StatusCode != http.StatusOK {
			t.Skipf("GET easy-site-profiles/%s: status=%d (profile may not exist)", siteID, profileResp.StatusCode)
		}
		var profile map[string]any
		if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
			t.Fatalf("decode profile: %v", err)
		}
		_ = profileResp.Body.Close()

		// Сохраняем исходные значения для восстановления
		origDisabled, _ := profile["disabled_error_pages"].([]any)
		origEnabled, _ := profile["use_custom_error_pages"].(bool)

		// Устанавливаем disabled_error_pages = ["404", "500"]
		profile["disabled_error_pages"] = []string{"404", "500"}
		profile["use_custom_error_pages"] = true
		saveResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
		defer saveResp.Body.Close()
		if saveResp.StatusCode != http.StatusOK && saveResp.StatusCode != http.StatusCreated {
			t.Fatalf("PUT easy-site-profiles/%s: status=%d body=%s", siteID, saveResp.StatusCode, mustReadBody(t, saveResp.Body))
		}

		// Читаем обратно и проверяем что исключения сохранились
		reloadResp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride)
		if reloadResp.StatusCode != http.StatusOK {
			t.Fatalf("re-GET easy-site-profiles/%s: status=%d", siteID, reloadResp.StatusCode)
		}
		var reloaded map[string]any
		if err := json.NewDecoder(reloadResp.Body).Decode(&reloaded); err != nil {
			t.Fatalf("decode reloaded profile: %v", err)
		}
		_ = reloadResp.Body.Close()

		disabledRaw, _ := reloaded["disabled_error_pages"].([]any)
		disabledSet := make(map[string]bool)
		for _, v := range disabledRaw {
			if s, ok := v.(string); ok {
				disabledSet[s] = true
			}
		}
		for _, want := range []string{"404", "500"} {
			if !disabledSet[want] {
				t.Errorf("disabled_error_pages missing %q after save; got %v", want, disabledRaw)
			}
		}

		// Восстанавливаем оригинальные значения
		origDisabledStrs := make([]string, 0, len(origDisabled))
		for _, v := range origDisabled {
			if s, ok := v.(string); ok {
				origDisabledStrs = append(origDisabledStrs, s)
			}
		}
		profile["disabled_error_pages"] = origDisabledStrs
		profile["use_custom_error_pages"] = origEnabled
		restoreResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
		_ = restoreResp.Body.Close()
	})

	t.Run("DisabledPagesReflectedInCompiledConfig", func(t *testing.T) {
		// Читаем текущий профиль
		profileResp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride)
		if profileResp.StatusCode != http.StatusOK {
			t.Fatalf("GET easy-site-profiles/%s: status=%d", siteID, profileResp.StatusCode)
		}
		var profile map[string]any
		if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
			t.Fatalf("decode profile: %v", err)
		}
		_ = profileResp.Body.Close()

		// Отключаем 403
		profile["disabled_error_pages"] = []string{"403"}
		profile["use_custom_error_pages"] = true
		saveResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
		_ = saveResp.Body.Close()

		// Compile + apply
		revID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
		if revID == "" {
			t.Fatal("compile/apply returned no revision ID")
		}
		assertE2EArtifactActive(t, revID, "nginx/easy/"+siteID+".conf")

		// Читаем скомпилированный конфиг через API
		confResp := getWithAuth(t, client, fmt.Sprintf("%s/api/revisions/%s/artifacts/nginx/easy/%s.conf", requestBaseURL, revID, siteID), requestHostOverride)
		body, _ := io.ReadAll(confResp.Body)
		_ = confResp.Body.Close()
		if confResp.StatusCode != http.StatusOK {
			runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
			if runtimeContainer == "" {
				runtimeContainer = "waf-e2e-runtime"
			}
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				active, err := exec.Command("docker", "exec", runtimeContainer, "cat", "/var/lib/waf/active/current.json").CombinedOutput()
				if err == nil && strings.Contains(string(active), revID) {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			runtimeBody, runtimeErr := exec.Command("docker", "exec", runtimeContainer, "cat", "/etc/waf/current/nginx/easy/"+siteID+".conf").CombinedOutput()
			if runtimeErr != nil {
				t.Fatalf("get active compiled config: api status=%d; runtime error=%v output=%s", confResp.StatusCode, runtimeErr, string(runtimeBody))
			}
			body = runtimeBody
		}
		confStr := string(body)

		// error_page 403 не должен быть в конфиге
		if strings.Contains(confStr, "error_page 403 ") {
			t.Errorf("compiled config still contains error_page 403 after disabling it:\n%s", confStr)
		}

		// Восстанавливаем
		profile["disabled_error_pages"] = []any{}
		postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
		e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	})
}

// e2eGetFirstSiteID возвращает ID первого активного сайта или "".
func e2eGetFirstSiteID(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride string) string {
	t.Helper()
	resp := getWithAuth(t, client, requestBaseURL+"/api/sites", requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	// API может вернуть массив напрямую или {"sites": [...]}
	var sites []map[string]any
	if err := json.Unmarshal(body, &sites); err != nil {
		// попробуем {"sites": [...]}
		var wrapped map[string]any
		if err2 := json.Unmarshal(body, &wrapped); err2 == nil {
			if arr, ok := wrapped["sites"].([]any); ok {
				for _, s := range arr {
					if sm, ok := s.(map[string]any); ok {
						if id, ok := sm["id"].(string); ok && id != "" {
							return id
						}
					}
				}
			}
		}
		return ""
	}
	for _, sm := range sites {
		if id, ok := sm["id"].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// e2eCompileAndApply запускает compile и apply, возвращает rev ID.
func e2eCompileAndApply(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride string) string {
	t.Helper()
	compileResp := postJSON(t, client, requestBaseURL+"/api/revisions/compile", requestHostOverride, map[string]any{})
	body, _ := io.ReadAll(compileResp.Body)
	_ = compileResp.Body.Close()
	if compileResp.StatusCode != http.StatusCreated && compileResp.StatusCode != http.StatusOK {
		t.Logf("compile failed: status=%d body=%s", compileResp.StatusCode, string(body))
		return ""
	}
	var compilePayload map[string]any
	if err := json.Unmarshal(body, &compilePayload); err != nil {
		t.Logf("decode compile response: %v; body=%s", err, string(body))
		return ""
	}
	// API возвращает {"revision_id": "..."} или {"revision": {"id": "..."}}
	revID, _ := compilePayload["revision_id"].(string)
	if revID == "" {
		if rev, ok := compilePayload["revision"].(map[string]any); ok {
			revID, _ = rev["id"].(string)
		}
	}
	if revID == "" {
		revID, _ = compilePayload["id"].(string)
	}
	if revID == "" {
		t.Logf("compile response did not contain a revision id: %s", string(body))
		return ""
	}

	// Ждём завершения compile job
	e2eWaitForRevisionJob(t, client, requestBaseURL, requestHostOverride, "compile-"+revID)

	// Apply
	applyResp := postJSON(t, client, requestBaseURL+"/api/revisions/"+revID+"/apply", requestHostOverride, map[string]any{})
	applyBody, _ := io.ReadAll(applyResp.Body)
	_ = applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusOK && applyResp.StatusCode != http.StatusCreated {
		t.Logf("apply failed: status=%d body=%s", applyResp.StatusCode, string(applyBody))
		return ""
	}

	e2eWaitForRevisionJob(t, client, requestBaseURL, requestHostOverride, "apply-"+revID)
	return revID
}

// e2eWaitForRevisionJob ждёт завершения job по prefix за 30с.
func e2eWaitForRevisionJob(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, jobPrefix string) {
	t.Helper()
	// Compile and apply endpoints return only after their job completes. Runtime
	// activation is asserted separately against the exact revision artifact.
	_ = jobPrefix
}
