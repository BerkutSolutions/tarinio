## [1.4.2] - 28.06.2026

### Ядро

- Добавлено поле `exceptions_uri` в профиль сайта — список URI-путей, которые исключаются из проверки WAF (antibot, rate limit, escalation) независимо от IP клиента. Компилятор генерирует nginx-директивы `if ($uri ~* "...")` устанавливающие `$waf_easy_exception_guard=1`.

### UI

- В блок «Белые списки и исключения» добавлено поле «Исключения по URI» — позволяет указать пути (например `/healthz`, `/metrics`) которые всегда пропускаются без проверки.
- Исправлена подсказка к переключателю «Активировать белые списки» — теперь корректно поясняет что исключения по IP и URI работают независимо от этого флага.
- Переименованы метки: «Исключения» → «Исключения по IP» во всех 5 локалях для однозначности.

### Локализация

- Добавлен ключ `sites.easy.traffic.exceptionsUri` на всех 5 языках (ru, en, de, sr, zh).
- Исправлен ключ `sites.help.traffic.allowlist.activate.usage` на всех 5 языках.

### Тесты

- Написано 157 тестов по всем 10 вкладкам UI — полное покрытие компилятора и control-plane:
  - `tab01_front_test.go` (19) — HSTS, AllowedMethods, MaxClientSize, HttpStrictParsing, входящий mTLS, SecurityMode
  - `tab02_upstream_test.go` (20) — PassHostHeader+CustomHost, SSL SNI, WebSocket, Keepalive, X-Forwarded-*, HealthCheck, upstream mTLS
  - `tab03_headers_test.go` (18) — ReferrerPolicy, CSP, PermissionsPolicy, CORS, CookieFlags → `proxy_cookie_flags`, KeepUpstreamHeaders → `proxy_pass_header`
  - `tab04_traffic_test.go` (24) — BlacklistIP/UA/URI/Country, WhitelistCountry, ExceptionsURI, LimitConn/LimitReq → `l4guard/config.json`, BadBehavior
  - `tab05_ban_escalation_test.go` (16) — нормализация scope/stages, валидация (пустые stages, >12, отрицательные, permanent не последний)
  - `tab06_antibot_test.go` (14) — javascript/recaptcha/hcaptcha/turnstile, ScannerAutoBan, ExclusionRules, CookieGuard, ChallengeEscalation
  - `tab07_geo_test.go` (16) — GeoTimeWindow snippet, block/allow action, hour range, days of week, HTTP map-артефакт `nginx/geo-timewindow/<id>.conf`, валидация
  - `tab08_modsec_test.go` (9) — UseModSecurity артефакт, `modsecurity_rules_file`, CRS версия, плагины, custom content
  - `tab09_websocket_test.go` (11) — WSInspection Lua-snippet, WSBlockPatterns, WSMaxMessageBytes, WSRateMsgPerSec, нормализация/валидация
  - `tab10_virtualpatches_test.go` (10) — SecRule block/monitor по uri/body/header, patch ID в msg, интеграция в modsec артефакт
- Все 157 тестов зелёные: `ok waf/compiler/internal/compiler`, `ok waf/control-plane/internal/easysiteprofiles`
- Добавлен preflight в `scripts/release.sh` — все tab-тесты запускаются до `go test ./...` и до меню выбора версии; падение прерывает релиз с кодом 1
- Добавлено правило в `.work/PROMT.md`: новая фича UI → обязательный тест `tab0X_*_test.go` в том же PR

### Документация

- Создан `docs/test-coverage-ru.md` — полный отчёт по покрытию тестами на русском языке: таблицы тестов по каждой вкладке, команды запуска, архитектурные выводы подтверждённые тестами
- Создан `docs/test-coverage-en.md` — зеркальный документ на английском языке