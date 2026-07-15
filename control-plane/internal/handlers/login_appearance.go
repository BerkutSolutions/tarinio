package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

const defaultLoginAppearance = "command-center"

func normalizeLoginAppearance(value string) string {
	switch strings.TrimSpace(value) {
	case "command-center", "incident-console", "command-center-classic", "security-card", "incident-console-classic":
		return strings.TrimSpace(value)
	default:
		return defaultLoginAppearance
	}
}

func CurrentLoginAppearance() string {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return normalizeLoginAppearance(runtimeSettingsState.loginAppearance)
}

type LoginAppearanceHandler struct{}

func (h *LoginAppearanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"login_appearance": CurrentLoginAppearance()})
}

type LoginAppearancePreviewHandler struct{}

func (h *LoginAppearancePreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	theme := normalizeLoginAppearance(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/login-appearance/preview/"), "/"))
	if theme != strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/login-appearance/preview/"), "/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	screen := strings.TrimSpace(r.URL.Query().Get("screen"))
	if screen != "2fa" {
		screen = "login"
	}
	title, description := loginAppearancePreviewCopy(theme)
	if theme == "command-center-classic" && screen != "2fa" {
		title = "Защищённый вход"
		description = "Доступ к контуру управления проверяется anti-bot и политиками учётной записи."
	}
	if screen == "2fa" {
		title, description = "Подтверждение второго фактора", "Тот же шаблон используется для кода, ключа доступа и возврата ко входу."
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprint(w, loginAppearancePreviewHTML(theme, screen, html.EscapeString(title), html.EscapeString(description)))
}

func loginAppearancePreviewCopy(theme string) (string, string) {
	switch theme {
	case "command-center", "command-center-classic":
		return "Вход в контур управления", "Компактная панель повторяет язык нового бокового меню: тёмные слои, холодный синий акцент и статус runtime."
	case "security-card":
		return "Авторизация", "Используйте учётную запись администратора."
	case "incident-console-classic":
		return "Авторизация оператора", "Строгий вариант в языке консоли инцидентов: сетка, служебная строка и деликатные границы."
	case "incident-console":
		return "Авторизация оператора", "Строгий вид консоли инцидентов с сервисной строкой и сеткой."
	default:
		return "Вход в контур управления", "Компактная панель повторяет язык нового бокового меню: тёмные слои, холодный синий акцент и статус runtime."
	}
}

func loginAppearancePreviewHTML(theme, screen, title, description string) string {
	preview := loginAppearancePreviewHTMLBase(theme, screen, title, description)
	preview = strings.Replace(preview, "</head>", `<link rel="stylesheet" href="/static/login-appearance-refinements.css?v=20260714-5"></head>`, 1)
	if theme == "command-center-classic" {
		preview = strings.Replace(preview, `</p></div><div class="login-theme-health">`, `</p><div class="login-theme-session-info"><span>Локальное время</span><strong>12:34:56</strong></div></div><div class="login-theme-health">`, 1)
	}
	return preview
}

func loginAppearancePreviewHTMLBase(theme, screen, title, description string) string {
	cardTitle := "Авторизация"
	controls := ""
	if screen == "2fa" {
		cardTitle = "2FA confirmation"
		controls = `<p class="login-theme-help">Второй фактор защищает доступ к контуру управления.</p><label>Код двухфакторной проверки</label><input placeholder="123456"><label class="checkbox"><input type="checkbox"> Использовать код восстановления</label><button class="btn primary">Подтвердить</button><button class="btn ghost">Войти с ключом доступа</button><button class="btn ghost">Назад</button>`
	} else {
		controls = `<p class="login-theme-help">%s</p><label>Имя пользователя</label><input placeholder="admin"><label>Пароль</label><input placeholder="••••••••" type="password"><button class="btn primary">Войти</button><button class="btn ghost">Войти с ключом доступа</button><button class="btn ghost">Войти через SSO</button>`
		controls = fmt.Sprintf(controls, description)
	}
	return fmt.Sprintf(`<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Предпросмотр входа</title><link rel="stylesheet" href="/static/styles.css"><link rel="stylesheet" href="/static/waf.css"><link rel="stylesheet" href="/static/login-appearance.css?v=20260714-4"></head><body class="login-body" data-login-appearance="%s"><div class="login-wrapper"><div class="login-theme-shell"><section class="login-theme-brand"><div class="login-theme-brand-main"><img class="login-theme-brand-logo" src="/static/logo800x300.png" alt=""><div class="login-theme-kicker">Security operations</div><h1 class="login-theme-brand-title">%s</h1><p class="login-theme-brand-copy">%s</p></div><div class="login-theme-health"><span></span>Runtime healthy · защищённое соединение</div></section><div class="login-theme-grid"></div><div class="login-card"><div class="login-theme-emblem">⌾</div><img class="login-logo" src="/static/logo800x300.png" alt=""><div class="login-theme-state">Защищённый доступ</div><div class="login-theme-caption">Management access / secure session</div><h1 class="login-theme-title">%s</h1><div class="alert" hidden></div><form>%s</form><div class="login-theme-meta"><span>Audit enabled</span><span>TLS protected</span></div></div></div></div></body></html>`, theme, title, description, cardTitle, controls)
}
