package compiler

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GeneratedErrorPage keeps extended statuses on the same canonical 502 WAF
// layout. Only the status copy, accent and emblem outside the card vary.
func GeneratedErrorPage(code string) ([]byte, bool) {
	page, ok := generatedErrorPages[code]
	if !ok {
		return nil, false
	}
	base, err := TemplatesFS.ReadFile("templates/errors/502.preview.html")
	if err != nil {
		return nil, false
	}
	copy := errorPageCopy(code, page.Copy)
	payload, err := json.Marshal(struct {
		Code  string               `json:"code"`
		Color string               `json:"color"`
		Glow  string               `json:"glow"`
		Icon  string               `json:"icon"`
		Copy  map[string]errorCopy `json:"copy"`
		Guide errorGuide           `json:"guide"`
	}{code, page.Color, page.Glow, errorIcon(page.Color, page.Symbol), copy, errorPageGuides[code]})
	if err != nil {
		return nil, false
	}
	script := fmt.Sprintf(extendedErrorPageScript, page.Color, page.Glow, payload)
	guide, err := json.Marshal(struct {
		Guides map[string]errorGuide `json:"guides"`
		Route  errorRouteVisual      `json:"route"`
	}{guidesForErrorPage(code), errorRouteVisuals[code]})
	if err != nil {
		return nil, false
	}
	script += fmt.Sprintf(errorGuideOverrideScript, guide)
	script += routeTargetLabelScript
	script = strings.Replace(script, `set("l-policy",x?"\u041a\u043e\u0434":"Code");`, "", 1)
	script = strings.Replace(script, `policy.textContent="extended / "+p.code`, `policy.textContent=p.code`, 1)
	script = strings.Replace(script, `new URLSearchParams(location.search).get("rid")||"n/a"`, `new URLSearchParams(location.search).get("rid")||(l==="ru"?"\u043d/\u0434":l==="sr"?"\u043d/\u043f":"n/a")`, 1)
	content := "<!-- HTTP " + code + " " + copy["en"].Title + " -->" + string(base)
	return []byte(strings.Replace(content, "</body>", script+"</body>", 1)), true
}

// NormalizeErrorPageRouteLabels keeps legacy static pages aligned with the
// extended-page terminology and restores request metadata after nginx makes an
// internal error_page redirect. The browser URL remains the original request,
// so query parameters on the internal target are intentionally unavailable.
func NormalizeErrorPageRouteLabels(content []byte) []byte {
	page := string(content)
	if !strings.Contains(page, `id="n-host"`) {
		return content
	}
	const script = `<script>(function(){var l=(document.documentElement.lang||navigator.language||"en").toLowerCase().split(/[-_]/)[0],e=document.getElementById("n-host");if(e)e.textContent=l==="ru"?"\u0421\u0435\u0440\u0432\u0435\u0440":"Origin";var n=(performance.getEntriesByType&&performance.getEntriesByType("navigation")[0]),s=n&&n.serverTiming||[];function v(k){for(var i=0;i<s.length;i++)if(s[i].name===k)return String(s[i].description||"").trim()}function set(k,x){var q=document.getElementById(k);if(q&&x)q.textContent=x}var rid=v("rid"),ip=v("ip"),ts=v("ts");set("m-rid",rid);set("m-ip",ip);if(ts){var d=new Date(Number(ts)*1000);set("m-time",isNaN(d.getTime())?ts:d.toLocaleString(l))}})()</script>`
	return []byte(strings.Replace(page, "</body>", script+"</body>", 1))
}

type errorCopy struct{ Title, Subtitle, State string }
type generatedErrorPage struct {
	Color, Glow, Symbol string
	Copy                map[string]errorCopy
}

var errorPageDescriptions = map[string]map[string]string{
	"451": {"en": "Access to this resource is restricted by an applicable law, legal order, or the site owner's policy.", "ru": "Доступ к ресурсу ограничен применимым законом, судебным требованием или политикой владельца сайта."},
	"494": {"en": "The WAF rejected the request before it reached the host because its HTTP headers exceed the allowed size.", "ru": "WAF отклонил запрос до передачи на хост: размер HTTP-заголовков превысил допустимый предел."},
	"495": {"en": "The TLS connection stopped because the client certificate did not pass WAF validation.", "ru": "TLS-подключение остановлено: клиентский сертификат не прошёл проверку WAF."},
	"496": {"en": "This address requires a valid client TLS certificate, but the client did not provide one.", "ru": "Для этого адреса нужен действительный клиентский TLS-сертификат, но клиент его не предоставил."},
	"497": {"en": "The client sent unencrypted HTTP to a port that expects an HTTPS connection.", "ru": "Клиент отправил незашифрованный HTTP-запрос на порт, который ожидает HTTPS."},
	"499": {"en": "The client closed the connection before processing finished, so the WAF could not return a response.", "ru": "Клиент закрыл соединение до завершения обработки запроса; WAF не может вернуть ответ."},
	"506": {"en": "The origin could not select a final resource representation because the chosen variant also requires negotiation.", "ru": "Источник не смог выбрать конечное представление ресурса: вариант ответа снова требует согласования."},
	"520": {"en": "The origin returned an unexpected or empty response that the WAF cannot safely pass to the client.", "ru": "Источник вернул непредвиденный или пустой ответ, который WAF не может безопасно передать клиенту."},
	"521": {"en": "The origin web server is unavailable: the WAF cannot establish a usable connection to it.", "ru": "Исходный веб-сервер недоступен: WAF не может установить с ним рабочее соединение."},
	"522": {"en": "The WAF could not establish a connection to the origin before the connection timeout expired.", "ru": "WAF не успел установить соединение с источником до истечения тайм-аута подключения."},
	"523": {"en": "The WAF network cannot reach the configured origin; its route, DNS, or firewall path is unavailable.", "ru": "Сеть WAF не может достичь указанного источника: маршрут, DNS или firewall недоступны."},
	"524": {"en": "The connection to the origin succeeded, but the application did not return a response before the timeout expired.", "ru": "Соединение с источником установлено, но ответ приложения не получен до истечения тайм-аута."},
	"525": {"en": "The WAF reached the origin, but their TLS handshake could not be completed.", "ru": "WAF достиг источника, но TLS-рукопожатие между ними не удалось завершить."},
	"526": {"en": "The origin presented a TLS certificate that the WAF cannot validate as trusted and valid.", "ru": "Источник предъявил TLS-сертификат, который WAF не может подтвердить как действительный."},
}

func errorPageCopy(code string, source map[string]errorCopy) map[string]errorCopy {
	copy := make(map[string]errorCopy, len(source))
	for language, value := range source {
		if description := errorPageDescriptions[code][language]; description != "" {
			value.Subtitle = description
		}
		copy[language] = value
	}
	return copy
}

// errorGuide carries the status-specific content placed into the canonical
// 502 request-path and recovery-step components.
type errorGuide struct {
	Client, Target, FirstState, SecondState string
	Steps                                   [3][2]string
}

// errorRouteVisual identifies the failing hop visually rather than relying on
// the same green-client/red-host diagram for every status.
type errorRouteVisual struct {
	Nodes, Arrows [3]string
}

var errorRouteVisuals = map[string]errorRouteVisual{
	"451": {Nodes: [3]string{"ok", "error", "muted"}, Arrows: [3]string{"ok", "error"}},
	"494": {Nodes: [3]string{"error", "ok", "muted"}, Arrows: [3]string{"error", "muted"}},
	"495": {Nodes: [3]string{"tls", "ok", "muted"}, Arrows: [3]string{"tls", "muted"}},
	"496": {Nodes: [3]string{"tls", "ok", "muted"}, Arrows: [3]string{"tls", "muted"}},
	"497": {Nodes: [3]string{"error", "ok", "muted"}, Arrows: [3]string{"error", "muted"}},
	"499": {Nodes: [3]string{"closed", "ok", "muted"}, Arrows: [3]string{"closed", "muted"}},
	"506": {Nodes: [3]string{"ok", "ok", "warn"}, Arrows: [3]string{"ok", "warn"}},
	"520": {Nodes: [3]string{"ok", "ok", "warn"}, Arrows: [3]string{"ok", "warn"}},
	"521": {Nodes: [3]string{"ok", "ok", "error"}, Arrows: [3]string{"ok", "error"}},
	"522": {Nodes: [3]string{"ok", "ok", "warn"}, Arrows: [3]string{"ok", "warn"}},
	"523": {Nodes: [3]string{"ok", "ok", "error"}, Arrows: [3]string{"ok", "error"}},
	"524": {Nodes: [3]string{"ok", "ok", "warn"}, Arrows: [3]string{"ok", "warn"}},
	"525": {Nodes: [3]string{"ok", "ok", "tls"}, Arrows: [3]string{"ok", "tls"}},
	"526": {Nodes: [3]string{"ok", "ok", "cert"}, Arrows: [3]string{"ok", "cert"}},
}

// errorGuideOverrideScript runs after the shared page-localization script.
// The shared script provides labels and chrome; this script deliberately keeps
// the status-specific route and recovery copy intact for every locale.
const errorGuideOverrideScript = `<script>(function(){var p=%s,l=document.documentElement.lang||"en",g=p.guides[l]||p.guides.en,r=p.route;function s(id,v){var e=document.getElementById(id);if(e)e.textContent=v}var P={ok:{c:"rgba(74,222,152,.65)",d:""},error:{c:"rgba(255,107,107,.72)",d:"3 2"},warn:{c:"rgba(250,204,21,.72)",d:"2 2"},muted:{c:"rgba(148,163,184,.38)",d:"1 3"},closed:{c:"rgba(148,163,184,.72)",d:"4 2"},tls:{c:"rgba(167,139,250,.78)",d:"6 2"},cert:{c:"rgba(6,182,212,.78)",d:"1 2"}};function mode(k){return P[k]||P.muted}function node(e,k){var q=mode(k),i=e.querySelector(".flow-icon"),v=i.querySelector("svg");i.className="flow-icon"+(e===document.querySelectorAll(".flow-node")[1]?" fi-shield":"");i.style.setProperty("border-color",q.c,"important");i.style.setProperty("background",q.c.replace(/,[^)]+\)/,",.10)"),"important");i.style.setProperty("box-shadow","0 0 12px "+q.c.replace(/,[^)]+\)/,",.18)"),"important");v.style.setProperty("stroke",q.c,"important")}function arrow(e,k){var q=mode(k),v=e.querySelector("svg");v.querySelectorAll("line,polyline").forEach(function(x){x.setAttribute("stroke",q.c);if(q.d)x.setAttribute("stroke-dasharray",q.d);else x.removeAttribute("stroke-dasharray")});var t=e.querySelector(".flow-arrow-lbl");t.style.color=q.c}function hostIcon(e,k){var q=mode(k),v=e.querySelector("svg");if(k==="error"){e.querySelectorAll("line").forEach(function(x){x.setAttribute("y1","7");x.setAttribute("y2","13")});return}v.innerHTML='<rect x="3" y="4" width="14" height="12" rx="2"/><line x1="7" y1="9" x2="13" y2="9"/><line x1="7" y1="12" x2="11" y2="12"/>';v.style.setProperty("stroke",q.c,"important")}s("n-you",g.Client);s("n-host",g.Target);s("a1-lbl",g.FirstState);s("a2-lbl",g.SecondState);s("s1t",g.Steps[0][0]);s("s1d",g.Steps[0][1]);s("s2t",g.Steps[1][0]);s("s2d",g.Steps[1][1]);s("s3t",g.Steps[2][0]);s("s3d",g.Steps[2][1]);var n=document.querySelectorAll(".flow-node"),a=document.querySelectorAll(".flow-arrow");for(var j=0;j<3;j++)node(n[j],r.Nodes[j]);for(var k=0;k<2;k++)arrow(a[k],r.Arrows[k]);hostIcon(n[2],r.Nodes[2]);var policy=document.querySelector(".card-meta .meta-cell:last-child .val");if(policy)policy.textContent="HTTP "+document.querySelector(".badge").textContent.replace("HTTP ","")})()</script>`

// Keep the terminal route endpoint consistent across all extended pages.
// Unicode escapes make this client-side label independent of source encoding.
const routeTargetLabelScript = `<script>(function(){var l=(document.documentElement.lang||"en").toLowerCase().split(/[-_]/)[0],e=document.getElementById("n-host");if(e)e.textContent=l==="ru"?"\u0421\u0435\u0440\u0432\u0435\u0440":"Origin"})()</script>`

var errorPageGuides = map[string]errorGuide{
	"451": {"Client", "Origin", "OK", "RESTRICTED", [3][2]string{{"Read the restriction notice", "Review the legal notice or the policy that applies to this resource."}, {"Use an eligible location or account", "Access may depend on your location, identity, or legal eligibility."}, {"Contact the site owner", "Ask the administrator to review the restriction with the Request ID."}}},
	"494": {"Client", "Origin", "HEADER TOO LARGE", "BLOCKED", [3][2]string{{"Reduce request headers", "Remove oversized cookies or unnecessary custom headers."}, {"Clear site data", "Clear cookies for this site, then send the request again."}, {"Inspect the client", "If it persists, check the client or proxy that is adding headers."}}},
	"495": {"TLS client", "Origin", "CERTIFICATE ERROR", "BLOCKED", [3][2]string{{"Choose a valid certificate", "Select a client certificate that belongs to this service."}, {"Verify the certificate chain", "Check the certificate validity period and issuing CA."}, {"Ask for access help", "Contact the administrator with the Request ID and certificate details."}}},
	"496": {"TLS client", "Origin", "CERTIFICATE REQUIRED", "BLOCKED", [3][2]string{{"Select a client certificate", "This service accepts only requests with a valid client certificate."}, {"Reconnect to the service", "Open the address again after selecting the certificate."}, {"Request a certificate", "Contact the administrator if no suitable certificate is available."}}},
	"497": {"HTTP client", "Origin", "WRONG PROTOCOL", "NOT FORWARDED", [3][2]string{{"Use an HTTPS address", "Replace http:// with https:// in the address."}, {"Check the port", "Make sure the selected port is configured for HTTPS."}, {"Update the integration", "Correct links, health checks, or clients that still use plain HTTP."}}},
	"499": {"Client", "Origin", "CONNECTION CLOSED", "CANCELLED", [3][2]string{{"Retry only when needed", "Repeat the operation if it was interrupted unexpectedly."}, {"Check client timeouts", "Increase client timeout values for long-running requests."}, {"Avoid duplicate operations", "Confirm the original request was not completed before retrying."}}},
	"506": {"Client", "Origin", "OK", "NEGOTIATION FAILED", [3][2]string{{"Request a supported variant", "Use an acceptable media type, language, or representation."}, {"Review content negotiation", "Check Accept headers and the origin's variant configuration."}, {"Contact the site owner", "Provide the Request ID so the resource configuration can be reviewed."}}},
	"520": {"Client", "Origin", "OK", "UNKNOWN RESPONSE", [3][2]string{{"Retry after a short pause", "A transient origin failure may resolve without any change."}, {"Check origin health", "Review application and reverse-proxy logs for the unexpected response."}, {"Contact the administrator", "Include the Request ID if the error continues."}}},
	"521": {"Client", "Origin", "OK", "SERVER DOWN", [3][2]string{{"Wait for recovery", "The origin service may be starting or temporarily unavailable."}, {"Start or restore the service", "Verify that the upstream application is running and healthy."}, {"Check service monitoring", "Review deployment, health-check, and infrastructure alerts."}}},
	"522": {"Client", "Origin", "OK", "CONNECT TIMEOUT", [3][2]string{{"Retry the connection", "The origin may become reachable after a short delay."}, {"Check network reachability", "Verify routing, firewall rules, and the upstream address."}, {"Review connection limits", "Check whether the origin is overloaded or refusing new connections."}}},
	"523": {"Client", "Origin", "OK", "ORIGIN UNREACHABLE", [3][2]string{{"Verify the origin address", "Confirm the upstream hostname and port are correct."}, {"Check DNS and firewall rules", "Ensure the WAF can resolve and reach the origin network."}, {"Restore origin availability", "Start the origin service or repair its network path."}}},
	"524": {"Client", "Origin", "OK", "RESPONSE TIMEOUT", [3][2]string{{"Wait for the response", "The origin may finish normally when its load decreases."}, {"Inspect slow operations", "Review application traces and database or dependency latency."}, {"Tune the origin response", "Optimize the slow operation or adjust timeouts when appropriate."}}},
	"525": {"WAF", "Origin", "OK", "HANDSHAKE FAILED", [3][2]string{{"Check TLS compatibility", "Verify the origin supports a compatible TLS protocol and cipher set."}, {"Review the certificate chain", "Ensure the origin sends the required intermediate certificates."}, {"Inspect TLS logs", "Use the Request ID to correlate proxy and origin TLS failures."}}},
	"526": {"WAF", "Origin", "OK", "CERTIFICATE INVALID", [3][2]string{{"Renew or replace the certificate", "Install a valid, non-expired certificate on the origin."}, {"Verify the hostname", "The origin certificate must match the configured upstream host."}, {"Repair the trust chain", "Install the complete certificate chain trusted by the WAF."}}},
}

func errorCopyFor(en, ru, de, sr, zh string) map[string]errorCopy {
	return map[string]errorCopy{
		"en": {en, "The WAF is online and identified this response while processing the request.", "ERROR"},
		"ru": {ru, "WAF работает и обнаружил этот ответ при обработке запроса.", "ОШИБКА"},
		"de": {de, "Die WAF ist online und hat diese Antwort bei der Anfrage erkannt.", "FEHLER"},
		"sr": {sr, "WAF je aktivan i prepoznao je ovaj odgovor pri obradi zahteva.", "GREŠKA"},
		"zh": {zh, "WAF 正在运行，并在处理请求时识别到了此响应。", "错误"},
	}
}

var generatedErrorPages = map[string]generatedErrorPage{
	"451": {"#d8a84e", "rgba(216,168,78,.22)", "M44 26l17 7v11c0 10-7 16-17 19-10-3-17-9-17-19V33l17-7zM36 45l5 5 10-11", errorCopyFor("Unavailable For Legal Reasons", "Недоступно по юридическим причинам", "Aus rechtlichen Gründen nicht verfügbar", "Nedostupno iz pravnih razloga", "因法律原因不可用")},
	"494": {"#f59e0b", "rgba(245,158,11,.22)", "M28 34h32M28 44h22M28 54h16", errorCopyFor("Request Header Too Large", "Слишком большой заголовок запроса", "Anfragekopfzeile zu groß", "Zaglavlje zahteva je preveliko", "请求头过大")},
	"495": {"#a78bfa", "rgba(167,139,250,.22)", "M35 27h18v34H35zM39 35h10M39 42h7M40 57l8-8m0 8-8-8", errorCopyFor("SSL Certificate Error", "Ошибка SSL-сертификата", "SSL-Zertifikatsfehler", "Greška SSL sertifikata", "SSL 证书错误")},
	"496": {"#8b5cf6", "rgba(139,92,246,.22)", "M44 28v20M34 39l10-11 10 11M34 58h20M34 48h20v10H34z", errorCopyFor("SSL Certificate Required", "Требуется SSL-сертификат", "SSL-Zertifikat erforderlich", "SSL sertifikat je obavezan", "需要 SSL 证书")},
	"497": {"#38bdf8", "rgba(56,189,248,.22)", "M30 29h28v30H30zM37 36v16M51 36v16M37 44h14", errorCopyFor("HTTP Request Sent to HTTPS Port", "HTTP-запрос отправлен на HTTPS-порт", "HTTP-Anfrage an HTTPS-Port gesendet", "HTTP zahtev je poslat na HTTPS port", "HTTP 请求发送到了 HTTPS 端口")},
	"499": {"#94a3b8", "rgba(148,163,184,.18)", "M32 32l24 24M56 32 32 56", errorCopyFor("Client Closed Request", "Клиент закрыл запрос", "Client hat Anfrage geschlossen", "Klijent je zatvorio zahtev", "客户端关闭了请求")},
	"506": {"#fb7185", "rgba(251,113,133,.22)", "M29 36h19M45 31l6 5-6 5M59 52H40M43 47l-6 5 6 5", errorCopyFor("Variant Also Negotiates", "Вариант также согласовывается", "Variante verhandelt ebenfalls", "Varijanta takođe pregovara", "变体也在协商")},
	"520": {"#f97316", "rgba(249,115,22,.22)", "M28 44h12l5-7 7 14 5-7h3M35 57l18-26", errorCopyFor("Unknown Error", "Неизвестная ошибка", "Unbekannter Fehler", "Nepoznata greška", "未知错误")},
	"521": {"#ef4444", "rgba(239,68,68,.22)", "M29 58h30M34 58V35h20v23M39 44h10", errorCopyFor("Web Server Is Down", "Веб-сервер недоступен", "Webserver ist ausgefallen", "Veb server nije dostupan", "Web 服务器不可用")},
	"522": {"#eab308", "rgba(234,179,8,.22)", "M44 31a13 13 0 1 0 0 26 13 13 0 0 0 0-26M44 38v7l5 3M38 27v-4h12v4M25 44h6M57 44h6", errorCopyFor("Connection Timed Out", "Время подключения истекло", "Verbindung hat Zeitüberschreitung", "Vreme povezivanja je isteklo", "连接超时")},
	"523": {"#f43f5e", "rgba(244,63,94,.22)", "M44 28a16 16 0 1 0 0 32M44 36v8l6 4M44 26v-4M44 66v-4M26 44h-4M62 44h4", errorCopyFor("Origin Is Unreachable", "Источник недоступен", "Origin ist nicht erreichbar", "Origin nije dostupan", "源站不可达")},
	"524": {"#facc15", "rgba(250,204,21,.22)", "M28 31h32v20H43l-7 7v-7H28zM50 34a7 7 0 1 0 0 14 7 7 0 0 0 0-14M50 38v4l3 2", errorCopyFor("A Timeout Occurred", "Произошёл тайм-аут", "Zeitüberschreitung aufgetreten", "Došlo je do prekoračenja vremena", "发生超时")},
	"525": {"#14b8a6", "rgba(20,184,166,.22)", "M28 37l10 10 6-6 6 6 10-10M28 51l10-10 6 6 6-6 10 10", errorCopyFor("SSL Handshake Failed", "Ошибка SSL-рукопожатия", "SSL-Handshake fehlgeschlagen", "SSL rukovanje nije uspelo", "SSL 握手失败")},
	"526": {"#06b6d4", "rgba(6,182,212,.22)", "M32 27h24v34H32zM38 35h8M38 42h6M47 46a6 6 0 1 0 0 12 6 6 0 0 0 0-12M47 49v3M47 54v.5", errorCopyFor("Invalid SSL Certificate", "Недействительный SSL-сертификат", "Ungültiges SSL-Zertifikat", "Nevažeći SSL sertifikat", "无效的 SSL 证书")},
}

func errorIcon(color, symbol string) string {
	alert := ""
	if color == "#14b8a6" {
		alert = `<path d="M40 40l8 8m0-8-8 8" stroke="#ff6b6b" stroke-width="2.6" stroke-linecap="round"/>`
	}
	return fmt.Sprintf(`<svg viewBox="0 0 88 88" fill="none"><defs><radialGradient id="g" cx="38%%" cy="32%%" r="65%%"><stop offset="0%%" stop-color="%s" stop-opacity=".55"/><stop offset="100%%" stop-color="#080c1c"/></radialGradient></defs><circle cx="44" cy="44" r="40" fill="url(#g)" stroke="%s" stroke-opacity=".55" stroke-width="1.5"/><path d="%s" stroke="%s" stroke-opacity=".9" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/>%s</svg>`, color, color, symbol, color, alert)
}

const extendedErrorPageScript = `<style id="extended-error-theme">:root{--col:%[1]s;color-scheme:dark}body{background-image:radial-gradient(ellipse 90%% 55%% at 50%% -5%%,%[2]s,transparent),radial-gradient(ellipse 50%% 40%% at 85%% 85%%,rgba(60,30,10,.1),transparent)}.badge{border-color:%[1]s!important;background:%[2]s!important;color:%[1]s!important;box-shadow:0 0 16px %[2]s!important}</style><script>(function(){var p=%[3]s;function lang(){var a=(navigator.languages||[]).concat(navigator.language||"en");for(var i=0;i<a.length;i++){var l=String(a[i]).toLowerCase().split(/[-_]/)[0];if(p.copy[l])return l}return"en"}function set(id,v){var e=document.getElementById(id);if(e)e.textContent=v}var I={ru:{client:"\u041a\u043b\u0438\u0435\u043d\u0442",tls:"TLS-\u043a\u043b\u0438\u0435\u043d\u0442",waf:"WAF",host:"\u0425\u043e\u0441\u0442",ok:"OK",error:"\u041e\u0428\u0418\u0411\u041a\u0410",s1:"\u041f\u0440\u043e\u0432\u0435\u0440\u044c\u0442\u0435 \u043e\u0448\u0438\u0431\u043a\u0443",d1:"\u0418\u0441\u043f\u043e\u043b\u044c\u0437\u0443\u0439\u0442\u0435 \u043e\u043f\u0438\u0441\u0430\u043d\u0438\u0435 \u0432\u044b\u0448\u0435, \u0447\u0442\u043e\u0431\u044b \u0438\u0441\u043f\u0440\u0430\u0432\u0438\u0442\u044c \u0437\u0430\u043f\u0440\u043e\u0441.",s2:"\u041f\u043e\u0432\u0442\u043e\u0440\u0438\u0442\u0435 \u0437\u0430\u043f\u0440\u043e\u0441",d2:"\u041f\u043e\u0434\u043e\u0436\u0434\u0438\u0442\u0435 \u043d\u0435\u043c\u043d\u043e\u0433\u043e \u0438 \u043f\u043e\u043f\u0440\u043e\u0431\u0443\u0439\u0442\u0435 \u0441\u043d\u043e\u0432\u0430.",s3:"\u041e\u0431\u0440\u0430\u0442\u0438\u0442\u0435\u0441\u044c \u043a \u0430\u0434\u043c\u0438\u043d\u0438\u0441\u0442\u0440\u0430\u0442\u043e\u0440\u0443",d3:"\u0423\u043a\u0430\u0436\u0438\u0442\u0435 ID \u0437\u0430\u043f\u0440\u043e\u0441\u0430, \u0435\u0441\u043b\u0438 \u043e\u0448\u0438\u0431\u043a\u0430 \u043d\u0435 \u0438\u0441\u0447\u0435\u0437\u0430\u0435\u0442."},de:{client:"Client",tls:"TLS-Client",waf:"WAF",host:"Host",ok:"OK",error:"FEHLER",s1:"Fehler prüfen",d1:"Verwenden Sie die Beschreibung oben, um die Anfrage zu korrigieren.",s2:"Anfrage wiederholen",d2:"Warten Sie kurz und versuchen Sie es erneut.",s3:"Administrator kontaktieren",d3:"Geben Sie bei anhaltendem Fehler die Anfrage-ID an."},sr:{client:"Klijent",tls:"TLS klijent",waf:"WAF",host:"Host",ok:"OK",error:"GRE\u0160KA",s1:"Proverite grešku",d1:"Koristite opis iznad da ispravite zahtev.",s2:"Pokušajte ponovo",d2:"Sačekajte kratko, pa pokušajte ponovo.",s3:"Obratite se administratoru",d3:"Navedite ID zahteva ako greška ostane."},zh:{client:"\u5ba2\u6237\u7aef",tls:"TLS \u5ba2\u6237\u7aef",waf:"WAF",host:"\u4e3b\u673a",ok:"OK",error:"\u9519\u8bef",s1:"\u68c0\u67e5\u9519\u8bef",d1:"\u8bf7\u4f7f\u7528\u4e0a\u65b9\u7684\u8bf4\u660e\u6765\u4fee\u6b63\u8bf7\u6c42\u3002",s2:"\u91cd\u8bd5\u8bf7\u6c42",d2:"\u8bf7\u7a0d\u7b49\u7247\u523b\u540e\u518d\u8bd5\u3002",s3:"\u8054\u7cfb\u7ba1\u7406\u5458",d3:"\u5982\u679c\u9519\u8bef\u6301\u7eed\u5b58\u5728\uff0c\u8bf7\u63d0\u4f9b\u8bf7\u6c42 ID\u3002"}};var l=lang(),t=p.copy[l]||p.copy.en,g=p.guide,x=I[l];document.documentElement.lang=l;document.title=p.code+" — "+t.Title;set("card-title",t.Title);set("card-subtitle",t.Subtitle);set("n-you",x?(g.Client==="TLS client"?x.tls:g.Client==="WAF"?x.waf:x.client):g.Client);set("n-host",x?x.host:g.Target);set("a1-lbl",x?(g.FirstState==="OK"?x.ok:x.error):g.FirstState);set("a2-lbl",x?(g.SecondState==="OK"?x.ok:x.error):g.SecondState);set("s1t",x?x.s1:g.Steps[0][0]);set("s1d",x?x.d1:g.Steps[0][1]);set("s2t",x?x.s2:g.Steps[1][0]);set("s2d",x?x.d2:g.Steps[1][1]);set("s3t",x?x.s3:g.Steps[2][0]);set("s3d",x?x.d3:g.Steps[2][1]);set("m-rid",new URLSearchParams(location.search).get("rid")||"n/a");set("l-policy",x?"\u041a\u043e\u0434":"Code");var policy=document.querySelector(".card-meta .meta-cell:last-child .val");if(policy)policy.textContent="extended / "+p.code;var badge=document.querySelector(".badge");if(badge)badge.textContent="HTTP "+p.code;var icon=document.querySelector(".icon-wrap");if(icon)icon.innerHTML=p.icon})()</script>`
