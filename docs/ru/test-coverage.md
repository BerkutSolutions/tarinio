# Покрытие тестами WAF UI — Точка опоры

**Версия:** 1.4.7
**Дата:** 2026-06-29
**Статус:** 158 tab-тестов + полный runtime e2e WAF — все зелёные

Документ служит доказательством покрытия функциональности WAF по каждой вкладке UI.
При любом регрессе достаточно запустить один из команд ниже — если тесты зелёные, система работает как ожидается.

---

## Запуск тестов

```bash
# Все тесты по вкладкам (быстро, ~1 секунда)
go test ./compiler/internal/compiler/ ./control-plane/internal/easysiteprofiles/ -count=1

# Только конкретная вкладка
go test ./compiler/internal/compiler/ -run "TestFront_" -v
go test ./compiler/internal/compiler/ -run "TestUpstream_" -v
go test ./compiler/internal/compiler/ -run "TestHeaders_" -v
go test ./compiler/internal/compiler/ -run "TestTraffic_" -v
go test ./control-plane/internal/easysiteprofiles/ -run "TestBanEscalation_" -v
go test ./compiler/internal/compiler/ -run "TestAntibot_" -v
go test ./compiler/internal/compiler/ -run "TestGeo_" -v
go test ./compiler/internal/compiler/ -run "TestModsec_" -v
go test ./compiler/internal/compiler/ -run "TestWebSocket_" -v
go test ./compiler/internal/compiler/ -run "TestVirtualPatches_" -v
go test ./compiler/internal/compiler/ -run "TestErrorPages_" -v

# Полный runtime-stack эталон WAF
E2E_FILTER=TestE2EBehavioral sh scripts/run-e2e-tests.sh

# Полный suite
go test ./...
```

---

## Результаты по вкладкам

### Вкладка 1 — Фронт (19 тестов)

Файл: `compiler/internal/compiler/tab01_front_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestFront_HSTS_FullDirective | HSTS с includeSubDomains и preload присутствует в конфиге |
| TestFront_HSTS_MaxAgeOnly | HSTS только с max-age, без subdomains/preload |
| TestFront_HSTS_Disabled_NoHeader | При HSTS=off директива не генерируется |
| TestFront_HSTS_DefaultMaxAge_WhenZero | При MaxAge=0 подставляется дефолтное значение |
| TestFront_AllowedMethods_LimitedSet | limit_except содержит только разрешённые методы |
| TestFront_AllowedMethods_DefaultWhenEmpty | При пустом списке генерируется дефолтный набор методов |
| TestFront_AllowedMethods_BlocksOtherMethods | Запрещённые методы возвращают 405 |
| TestFront_MaxClientSize_IsSet | client_max_body_size устанавливается в конфиге |
| TestFront_HttpStrictParsing_Enabled | ignore_invalid_headers on + underscores_in_headers off |
| TestFront_HttpStrictParsing_Disabled | При отключении — противоположные значения директив |
| TestFront_MTLS_Required_DirectivesPresent | ssl_verify_client required + ssl_client_certificate |
| TestFront_MTLS_Optional_DirectivesPresent | ssl_verify_client optional для мягкой проверки |
| TestFront_MTLS_Disabled_NoDirectives | При mTLS=off нет ssl_verify_client в конфиге |
| TestFront_MTLS_PassHeaders_Enabled | X-Client-Verify и X-Client-DN передаются апстриму |
| TestFront_MTLS_Validation_NoCA | Ошибка при mTLS без указания CA-сертификата |
| TestFront_MTLS_Validation_NegativeDepth | Ошибка при отрицательной глубине цепочки сертификатов |
| TestFront_SecurityMode_Block_ModSecOn | SecurityMode=block создаёт артефакт modsecurity/easy/&lt;id&gt;.conf |
| TestFront_SecurityMode_Disabled_NoModSec | SecurityMode=disabled не создаёт артефакт modsec |
| TestFront_SecurityMode_Monitor_NoModSecArtifact | SecurityMode=monitor не создаёт артефакт modsec |

---

### Вкладка 2 — Апстрим (20 тестов)

Файл: `compiler/internal/compiler/tab02_upstream_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestUpstream_PassHostHeader_WithCustomHost | proxy_set_header Host генерируется только при PassHostHeader+CustomHost одновременно |
| TestUpstream_PassHostHeader_NoCustomHost_NoOverride | Без CustomHost нет proxy_set_header Host |
| TestUpstream_CustomHost_IsSet | Кастомный хост подставляется в директиву |
| TestUpstream_CustomHost_Empty_NoOverride | Пустой CustomHost — нет перезаписи Host |
| TestUpstream_SSLSNI_Enabled | proxy_ssl_server_name on + proxy_ssl_name для SNI |
| TestUpstream_SSLSNI_Disabled | При отключении SNI — нет этих директив |
| TestUpstream_Websocket_Enabled | Upgrade и Connection "" для WebSocket-проксирования |
| TestUpstream_Websocket_Disabled_NoUpgrade | Без WebSocket нет Upgrade-заголовка |
| TestUpstream_Keepalive_Enabled_ConnectionEmpty | keepalive 32 + Connection "" в пуле соединений |
| TestUpstream_Keepalive_Disabled_NoConnectionEmpty | При отключении keepalive нет Connection "" |
| TestUpstream_HealthCheck_Enabled | proxy_next_upstream + keepalive в sites шаблоне |
| TestUpstream_HealthCheck_Disabled_NoNextUpstream | Без HealthCheck нет proxy_next_upstream |
| TestUpstream_XForwardedFor_Enabled | X-Forwarded-For передаётся апстриму |
| TestUpstream_XForwardedFor_Disabled_Cleared | X-Forwarded-For очищается при отключении |
| TestUpstream_XForwardedProto_Enabled | X-Forwarded-Proto передаётся апстриму |
| TestUpstream_XRealIP_Enabled | X-Real-IP передаётся апстриму |
| TestUpstream_MTLS_DirectivesPresent | proxy_ssl_certificate + proxy_ssl_certificate_key + proxy_ssl_trusted_certificate |
| TestUpstream_MTLS_Disabled_NoProxySSLCert | Без mTLS нет proxy_ssl_certificate |
| TestUpstream_MTLS_Validation_NoCert | Ошибка при mTLS апстрима без сертификата |
| TestUpstream_MTLS_Validation_NoKey | Ошибка при mTLS апстрима без ключа |

---

### Вкладка 3 — HTTP-заголовки (18 тестов)

Файл: `compiler/internal/compiler/tab03_headers_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestHeaders_ReferrerPolicy_Set | add_header Referrer-Policy устанавливается |
| TestHeaders_ReferrerPolicy_Empty_NoHeader | Пустое значение — нет директивы |
| TestHeaders_CSP_Set | add_header Content-Security-Policy устанавливается |
| TestHeaders_CSP_Empty_NoHeader | Пустое значение — нет директивы |
| TestHeaders_PermissionsPolicy_Set | add_header Permissions-Policy устанавливается |
| TestHeaders_PermissionsPolicy_Empty_NoHeader | Пустое значение — нет директивы |
| TestHeaders_CORS_Enabled_AllowOrigin | Access-Control-Allow-Origin добавляется при CORS=on |
| TestHeaders_CORS_Disabled_NoAllowOrigin | Без CORS нет CORS-заголовков |
| TestHeaders_CORS_MultipleOrigins | Несколько разрешённых origin в заголовке |
| TestHeaders_CookieFlags_Set | proxy_cookie_flags устанавливается для куки |
| TestHeaders_CookieFlags_Empty_NoDirective | Пустые флаги — нет директивы |
| TestHeaders_CookieFlags_Secure | Флаг Secure передаётся в proxy_cookie_flags |
| TestHeaders_KeepUpstreamHeaders_Single | proxy_pass_header для одного заголовка апстрима |
| TestHeaders_KeepUpstreamHeaders_Multiple | proxy_pass_header для нескольких заголовков |
| TestHeaders_KeepUpstreamHeaders_Empty_NoDirective | Пустой список — нет proxy_pass_header |
| TestHeaders_HSTS_FullDirective | Strict-Transport-Security с полным набором параметров |
| TestHeaders_HSTS_Disabled_NoSTS | Без HSTS нет Strict-Transport-Security |
| TestHeaders_AllSecurityHeaders_Together | Все заголовки безопасности одновременно в одном конфиге |

---

### Вкладка 4 — Трафик (24 теста)

Файл: `compiler/internal/compiler/tab04_traffic_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestTraffic_BlacklistIP_Single | deny &lt;IP&gt; генерируется в конфиге |
| TestTraffic_BlacklistIP_Multiple | Несколько IP в чёрном списке |
| TestTraffic_BlacklistIP_Empty_NoDeny | Пустой список — нет директив deny |
| TestTraffic_BlacklistUA_Single | User-Agent в чёрном списке → guard + return 403 |
| TestTraffic_BlacklistUA_BlockReturns403 | Заблокированный UA возвращает 403 |
| TestTraffic_BlacklistUA_ExceptionGuard | Исключения UA обходятся через exception guard |
| TestTraffic_BlacklistURI_Single | URI в чёрном списке → waf guard |
| TestTraffic_BlacklistURI_Returns403 | Заблокированный URI возвращает 403 |
| TestTraffic_BlacklistURI_Multiple | Несколько URI в чёрном списке |
| TestTraffic_ExceptionsURI_Single | Одно исключение URI генерирует exception guard |
| TestTraffic_ExceptionsURI_Multiple | Несколько исключений URI |
| TestTraffic_ExceptionsURI_Empty_NoExtraGuard | Пустые исключения — нет лишних guard-директив |
| TestTraffic_ExceptionsURI_BypassesBlacklistIP | Исключение URI обходит блокировку IP |
| TestTraffic_BlacklistCountry_Single | waf_country_guard для заблокированной страны |
| TestTraffic_BlacklistCountry_Multiple | Несколько стран в чёрном списке |
| TestTraffic_WhitelistCountry_Single | Режим whitelist: !~ для разрешённых стран |
| TestTraffic_WhitelistCountry_BlocksOthers | Whitelist блокирует все страны кроме разрешённых |
| TestTraffic_LimitConn_Enabled | l4guard/config.json содержит conn_limit |
| TestTraffic_LimitConn_Disabled_NoDirective | Без LimitConn нет conn_limit в l4guard |
| TestTraffic_LimitReq_Enabled | l4guard/config.json содержит rate_per_second |
| TestTraffic_LimitReq_Disabled_NoDirective | Без LimitReq нет rate_per_second в l4guard |
| TestTraffic_BadBehavior_Enabled | Обнаружение плохого поведения включено |
| TestTraffic_BadBehavior_BanReturns429 | Бан за плохое поведение возвращает 429 |
| TestTraffic_BadBehavior_EscalationReturns403 | Эскалация бана возвращает 403 |

---

### Вкладка 5 — Эскалация банов (16 тестов)

Файл: `control-plane/internal/easysiteprofiles/tab05_ban_escalation_test.go`
Пакет: `control-plane/internal/easysiteprofiles`

| Тест | Что проверяется |
|------|----------------|
| TestBanEscalation_Normalize_ScopeDefault | Пустой scope → дефолт "all_sites" |
| TestBanEscalation_Normalize_ScopeUpperCase | Scope в верхнем регистре нормализуется в нижний |
| TestBanEscalation_Normalize_CurrentSite | Scope "current_site" нормализуется корректно |
| TestBanEscalation_Normalize_StagesDefault | Пустые stages → дефолт [300, 86400, 0] |
| TestBanEscalation_Normalize_StagesDeduped | Дублирующиеся нули в stages удаляются (оставляется один) |
| TestBanEscalation_Validate_InvalidScope | Невалидный scope → ошибка валидации |
| TestBanEscalation_Validate_AllSitesScope_Valid | Scope "all_sites" проходит валидацию |
| TestBanEscalation_Validate_CurrentSiteScope_Valid | Scope "current_site" проходит валидацию |
| TestBanEscalation_Validate_EmptyStages_WhenEnabled | Пустые stages при enabled=true → ошибка |
| TestBanEscalation_Validate_TooManyStages | Более 12 стадий → ошибка |
| TestBanEscalation_Validate_NegativeStage | Отрицательное значение стадии → ошибка |
| TestBanEscalation_Validate_PermanentNotLast | Нулевой (permanent) бан не последний → ошибка |
| TestBanEscalation_Validate_PermanentAsLastStage_Valid | Нулевой бан последний → валидно |
| TestBanEscalation_Validate_SingleFiniteStage_Valid | Одна конечная стадия без нуля → валидно |
| TestBanEscalation_Validate_MaxStages_Valid | Ровно 12 стадий → валидно |
| TestBanEscalation_Validate_Disabled_NoStagesRequired | При enabled=false пустые stages не вызывают ошибку |

---

### Вкладка 6 — Антибот (14 тестов)

Файл: `compiler/internal/compiler/tab06_antibot_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestAntibot_Disabled_NoGuardVars | При отключённом антиботе нет guard-переменных в конфиге |
| TestAntibot_Javascript_ChallengeVar | Challenge mode "javascript" устанавливается в конфиге |
| TestAntibot_Javascript_RedirectURI | AntibotURI генерирует redirect 302 |
| TestAntibot_Recaptcha_ChallengeVar | Challenge mode "recaptcha" устанавливается в конфиге |
| TestAntibot_Hcaptcha_ChallengeVar | Challenge mode "hcaptcha" устанавливается в конфиге |
| TestAntibot_Turnstile_ChallengeVar | Challenge mode "turnstile" (Cloudflare) устанавливается |
| TestAntibot_ScannerAutoBan_GuardPresent | При ScannerAutoBan=true добавляется scanner guard + return 403 |
| TestAntibot_ScannerAutoBan_Disabled_NoScannerGuard | Без ScannerAutoBan нет scanner guard |
| TestAntibot_ExclusionRule_BypassesChallenge | ExclusionRule генерирует waf_antibot_exception_guard |
| TestAntibot_CookieGuard_VerifiesSession | CookieGuard проверяет waf_antibot_verified cookie |
| TestAntibot_ChallengeEscalation_Enabled | Эскалация challenge: turnstile в конфиге + X-WAF-Antibot-Provider |
| TestAntibot_ChallengeEscalation_WithRules | Эскалация с правилами по URI |
| TestAntibot_UnverifiedRequest_RedirectOrBlock | Непроверенный запрос → redirect или block |
| TestAntibot_DebugHeader_Present | Отладочный заголовок X-WAF-Antibot-Mode присутствует |

---

### Вкладка 7 — Гео-фильтрация / Временные окна (16 тестов)

Файл: `compiler/internal/compiler/tab07_geo_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestGeo_TimeWindow_SnippetInSiteConf | Snippet "geo time-window enforcement" появляется в site.conf |
| TestGeo_TimeWindow_BlockAction_Returns403 | Action=block генерирует return 403 |
| TestGeo_TimeWindow_AllowAction_No403 | Action=allow не генерирует return 403 |
| TestGeo_TimeWindow_HourRange_InSnippet | Диапазон часов присутствует в server snippet |
| TestGeo_TimeWindow_DaysOfWeek_InSnippet | Дни недели присутствуют в server snippet |
| TestGeo_TimeWindow_ExceptionGuard_Bypass | Exception guard обходит гео-ограничение |
| TestGeo_TimeWindow_MapArtifact_Generated | Артефакт nginx/geo-timewindow/&lt;id&gt;.conf создаётся |
| TestGeo_TimeWindow_HttpConf_HourMap | HTTP-конфиг содержит map $time_iso8601 для часов |
| TestGeo_TimeWindow_HttpConf_CountryMap | HTTP-конфиг содержит маппинг стран ("JP" 1, "KR" 1) |
| TestGeo_TimeWindow_InvalidWindow_Ignored | Окно с HoursStart >= HoursEnd отбрасывается |
| TestGeo_TimeWindow_Empty_NoSnippet | Пустой GeoTimeWindows — нет snippet в конфиге |
| TestGeo_TimeWindow_MultipleWindows | Несколько окон генерируют индексы _0_ и _1_ |
| TestGeo_Validate_InvalidAction | Action не block/allow → ошибка валидации |
| TestGeo_Validate_InvalidCountryCode | Несуществующий код страны → ошибка валидации |
| TestGeo_Validate_HoursStartGEHoursEnd | HoursStart >= HoursEnd → ошибка валидации |
| TestGeo_Validate_InvalidDayOfWeek | День недели вне диапазона 0-6 → ошибка валидации |

---

### Вкладка 8 — ModSecurity (9 тестов)

Файл: `compiler/internal/compiler/tab08_modsec_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestModsec_Enabled_ArtifactCreated | UseModSecurity=true создаёт артефакт modsecurity/easy/&lt;id&gt;.conf |
| TestModsec_Disabled_NoArtifact | UseModSecurity=false — нет артефакта modsec |
| TestModsec_RulesFileDirective_InSiteConf | modsecurity_rules_file /etc/waf/modsecurity/easy/&lt;id&gt;.conf присутствует |
| TestModsec_Disabled_NoRulesFileDirective | Без UseModSecurity нет директивы modsecurity_rules_file |
| TestModsec_CRSVersion_InArtifact | Версия CRS отражается в содержимом артефакта |
| TestModsec_Plugins_InArtifact | CRS плагины перечислены в артефакте |
| TestModsec_CustomContent_InArtifact | Кастомные SecRule-правила включаются в артефакт |
| TestModsec_SecurityMode_Disabled_ArtifactStillCreated | UseModSecurity=true создаёт артефакт даже при SecurityMode=disabled |
| TestModsec_SecurityMode_Disabled_UseModSecFalse_NoArtifact | UseModSecurity=false + disabled — нет артефакта |

---

### Вкладка 9 — WebSocket инспекция (10 тестов)

Файл: `compiler/internal/compiler/tab09_websocket_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestWebSocket_Inspection_SnippetInSiteConf | При UseWSInspection=true паттерн появляется в site.conf (Lua-блок) |
| TestWebSocket_Inspection_Disabled_NoSnippet | UseWSInspection=false — нет WS snippet в конфиге |
| TestWebSocket_BlockPattern_InSnippet | WSBlockPatterns присутствуют в сгенерированном snippet |
| TestWebSocket_MaxMessageBytes_InSnippet | WSMaxMessageBytes присутствует в snippet |
| TestWebSocket_RateMsgPerSec_InSnippet | WSRateMsgPerSec присутствует в snippet |
| TestWebSocket_NoPatterns_NoSnippet | Нет паттернов и лимитов → пустой snippet |
| TestWebSocket_Validate_InvalidPattern | Невалидный regex → ошибка с указанием индекса |
| TestWebSocket_Validate_ValidPatterns | Валидные паттерны проходят валидацию без ошибок |
| TestWebSocket_Normalize_DedupPatterns | Дублирующиеся паттерны удаляются при нормализации |
| TestWebSocket_Normalize_EmptyRemoved | Пустые строки удаляются при нормализации |

---

### Вкладка 10 — Виртуальные патчи (10 тестов)

Файл: `compiler/internal/compiler/tab10_virtualpatches_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestVirtualPatches_Block_URI_Rule | SecRule REQUEST_URI с deny,status:403 для target=uri |
| TestVirtualPatches_Block_Body_Rule | SecRule REQUEST_BODY с deny,status:403 для target=body |
| TestVirtualPatches_Block_Header_Rule | SecRule REQUEST_HEADERS с deny,status:403 для target=header |
| TestVirtualPatches_Monitor_URI_Rule | Monitor action → pass без deny для target=uri |
| TestVirtualPatches_Monitor_Body_Rule | Monitor action → pass без deny для target=body |
| TestVirtualPatches_ID_InRuleMsg | ID патча присутствует в msg SecRule |
| TestVirtualPatches_Multiple_Rules | Несколько патчей — все паттерны в артефакте |
| TestVirtualPatches_Empty_NoRules | Nil patches — нет SecRule виртуальных патчей |
| TestVirtualPatches_Integration_InModsecArtifact | Патч присутствует в modsecurity/easy/&lt;id&gt;.conf артефакте |
| TestVirtualPatches_SecurityMode_Disabled_ArtifactStillCreated | UseModSecurity=true создаёт артефакт даже при SecurityMode=disabled |

---

### Вкладка 11 — Кастомные страницы ошибок (2 теста)

Файл: `compiler/internal/compiler/tab11_errorpages_test.go`

| Тест | Что проверяется |
|------|----------------|
| TestErrorPages_Enabled_HasProxyInterceptAndErrorPages | `UseCustomErrorPages=true` включает `proxy_intercept_errors on`, генерирует `error_page` и site-scoped пути `/__waf_errors/<site>/...` |
| TestErrorPages_Disabled_NoProxyIntercept | `UseCustomErrorPages=false` не добавляет `proxy_intercept_errors on` и не генерирует `error_page` директивы |

---

## Runtime e2e — эталон работы WAF (31 сценарий)

Файл: `ui/tests/e2e_behavioral_test.go`

Этот suite запускается в полном Docker Compose stack (`control-plane`, `runtime`, `postgres`, `vault`, `upstream-echo`) и проверяет не только генерацию артефактов, а реальное поведение WAF на HTTP-запросах.

| Группа сценариев | Что подтверждает |
|------------------|------------------|
| Blacklist IP / User-Agent / URI | Запросы реально блокируются кодом 403, а отключение правил возвращает pass-through |
| RateLimit и custom route limits | Burst получает 429, route-specific limit срабатывает на нужном URI и не задевает соседние пути |
| Antibot и cookie flow | Новый клиент получает 302 на challenge, verify выставляет cookie, после cookie upstream отвечает 200 |
| SecurityMode transparent / monitor | Пассивные режимы не оставляют активных deny/ModSecurity/auth/antibot блокировок |
| Custom error pages | Брендированная 403-страница включается, `disabled_error_pages` возвращает стандартную короткую страницу |
| Scanner auto-ban | Сигнатуры `sqlmap`/`nikto` блокируются 403 независимо от основного antibot challenge |
| ModSecurity CRS | SQL injection и XSS блокируются 403, легитимный запрос проходит |
| Geo policy | Blacklist/whitelist country конфигурации применяются, 451 доступен для geo-block страницы |
| Basic Auth gate | Без credentials — 302 на `/auth`, verify endpoint возвращает 204 и выставляет auth-cookie |
| Response headers | CORS, CSP, Referrer-Policy, Permissions-Policy и HSTS реально присутствуют в ответах |
| Exceptions URI | Исключение URI обходит blacklist guard |
| Virtual patches | Per-site SecRule по URI реально блокирует запрос 403 |
| Antibot exclusions/rules/escalation | Исключения обходят challenge, per-path rules меняют challenge, two-layer escalation ведёт на stage1 |
| Cookie flags / upstream headers / strict parsing / WS / geo time windows / JA3 | Конфиги применяются без nginx reload ошибок и сохраняют ожидаемые runtime-инварианты |

В последнем зелёном прогоне: 30 сценариев PASS, mTLS upload scenario SKIP из-за отсутствия тестового файла CA в окружении; общий `TestE2EBehavioral` завершился `PASS`.

---

## Сводная таблица

| Вкладка | Файл | Тестов | Пакет |
|---------|------|--------|-------|
| 1 — Фронт | tab01_front_test.go | 19 | compiler/internal/compiler |
| 2 — Апстрим | tab02_upstream_test.go | 20 | compiler/internal/compiler |
| 3 — Заголовки | tab03_headers_test.go | 18 | compiler/internal/compiler |
| 4 — Трафик | tab04_traffic_test.go | 24 | compiler/internal/compiler |
| 5 — Эскалация банов | tab05_ban_escalation_test.go | 16 | control-plane/internal/easysiteprofiles |
| 6 — Антибот | tab06_antibot_test.go | 14 | compiler/internal/compiler |
| 7 — Гео | tab07_geo_test.go | 16 | compiler/internal/compiler |
| 8 — ModSecurity | tab08_modsec_test.go | 9 | compiler/internal/compiler |
| 9 — WebSocket | tab09_websocket_test.go | 10 | compiler/internal/compiler |
| 10 — Виртуальные патчи | tab10_virtualpatches_test.go | 10 | compiler/internal/compiler |
| 11 — Страницы ошибок | tab11_errorpages_test.go | 2 | compiler/internal/compiler |
| **Итого tab-тесты** | | **158** | |
| Runtime WAF e2e | e2e_behavioral_test.go | 31 сценарий | ui/tests |

---

## Архитектурные выводы из тестов

Следующие факты подтверждены тестами — не документацией, а реальным поведением компилятора:

**Апстрим:**
`PassHostHeader` без `CustomHost` не генерирует `proxy_set_header Host`. Оба поля обязательны одновременно (шаблон: `{{- if and .PassHostHeader .ReverseProxyCustomHost }}`).

**HealthCheck и Keepalive:**
Директивы `proxy_next_upstream` и `keepalive` генерируются только в `sites/site.conf.tmpl` (не-easy шаблон), а не в easy-шаблоне. Тестируются через `nginxSiteData`.

**Ограничения трафика:**
`LimitConn` и `LimitReq` попадают в `l4guard/config.json`, а не в nginx-конфиг. `conn_limit = max(200, LimitConnMaxHTTP1)` — берётся максимум, минимум 200.

**Гео временные окна:**
Server snippet содержит переменные `$waf_geo_tw_*`, а коды стран — только в HTTP map-артефакте `nginx/geo-timewindow/&lt;id&gt;.conf`. Невалидные окна (HoursStart >= HoursEnd) молча отбрасываются.

**ModSecurity:**
`UseModSecurity=true` создаёт артефакт `modsecurity/easy/&lt;id&gt;.conf` независимо от `SecurityMode`. Отключение ModSec через `SecurityMode=disabled` без `UseModSecurity=false` не убирает артефакт.

**ChallengeEscalation:**
Эскалация заменяет base challenge на режим эскалации. Строка `"javascript"` исчезает из конфига когда активна эскалация с режимом `"turnstile"`.

**WebSocket инспекция:**
WS snippet — Lua-блок, не nginx-директивы. Snippet не генерируется при пустом списке паттернов и нулевых лимитах, даже если `UseWSInspection=true`.

**Runtime WAF:**
Поведенческий e2e является эталоном фактической работы WAF: он проверяет ответы runtime nginx, cookies, redirects, status codes и применение скомпилированных ревизий. Tab-тесты доказывают структуру артефактов, runtime e2e доказывает, что эти артефакты реально работают вместе.

---

## Интеграция в релизный процесс

Preflight в `scripts/release.ps1` запускает tab-тесты, `go test ./...` и полный `TestE2EBehavioral` до публикации релиза. Падение хотя бы одного теста прерывает релиз с кодом 1.

Правило проекта (`.work/PROMT.md`): каждая новая фича UI требует соответствующего теста в `tab0X_*_test.go` или `easysiteprofiles/tab0X_*_test.go` в том же PR.
