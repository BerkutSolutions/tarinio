## [1.4.6] - 29.06.2026

### Инфраструктура

- Исправлен шаблон `crs-setup.conf.tmpl`: теперь устанавливает `tx.crs_setup_version=4270`, что устраняет блокировку всех запросов с кодом 500 правилом CRS 901001 при свежей установке.
- `scripts/install-aio.sh`: компиляция и применение ревизии при post-upgrade health gate теперь выполняются только при обновлении существующей установки (`IS_UPGRADE=1`); при первичной установке этот шаг пропускается, так как пользователей ещё нет и waf-cli не может авторизоваться.
- `scripts/install-aio.sh`: `git checkout` заменён на `git checkout -B origin/main` — теперь корректно работает при detached HEAD (после установки через тег v1.x.x).
- `scripts/install-aio.sh`: `is_placeholder_secret` теперь нормализует значение к нижнему регистру и заменяет `_` на `-` перед проверкой — `CHANGE_ME` корректно определяется как плейсхолдер.
- `scripts/install-aio.sh`: генерация паролей `POSTGRES_PASSWORD` и `OPENSEARCH_PASSWORD` теперь не зависит от `ENV_CREATED` — пароли генерируются при любом запуске если в `.env` стоят плейсхолдеры.

### Ядро

- Убрана директива `modsecurity on;` из шаблона `nginx/easy/site.conf.tmpl`: включение ModSecurity управляется через `SecRuleEngine` внутри `modsecurity/easy/<site>.conf`, а `modsecurity on;` на уровне server-блока переопределяло `SecRuleEngine Off` management-сайта и блокировало запросы к панели управления с кодом 403.

## [1.4.5] - 29.06.2026

### Ядро

- Исправлена генерация nginx для ограничения частоты по конкретному URI: теперь зона выбирается через http-level `map`, а не через недопустимый `limit_req` внутри `if`, поэтому ревизия снова применима на PROD.
- Исправлено дублирование директив ModSecurity в easy-site локациях: базовый site-шаблон больше не включает `modsecurity on` для easy-сайтов, а easy-snippet сам задаёт `on` или `off`.
- Исправлена сборка ревизии для сайтов без сохранённого easy-профиля: автоматически созданный default easy-profile теперь помечает site-конфиг как easy-enabled и не дублирует ModSecurity в runtime.
- Исправлено дублирование `modsecurity off` в `/static/` location easy-сайтов: static-location больше не задаёт ModSecurity отдельно, когда подключает общий easy-snippet.
- Прозрачный режим и режим наблюдения сохраняют пассивное поведение без включённых модулей безопасности: не подключают ModSecurity, blacklist, антибот, auth-gate, geo-блокировки, rate-limit и L4 guard даже если черновик содержал включённые защитные настройки.

### Инфраструктура

- `scripts/install-aio.sh` после обновления control-plane теперь компилирует и применяет свежую runtime-ревизию перед проверкой публичного gateway, чтобы установка не оставалась на старой активной ревизии.

### Тесты

- Добавлена проверка, что URI-scoped rate-limit генерирует валидный nginx-конфиг через отдельный ключ зоны, а transparent/monitor остаются без активной защиты при стресс-настройках.

## [1.4.4] - 29.06.2026

### Ядро

- Удалены старые шаблоны `compiler/templates/errors/403.html.tmpl` и `compiler/templates/errors/50x.html.tmpl` — не вызывались ни из одного Go-файла; все HTTP-ошибки генерируются через `status.html.tmpl`.
- Добавлено поле `AntibotChallengeTemplate` в `EasyProfileInput` (compiler/types.go), `easyConfigData` (easy.go) и маппинг в `runtime_apply.go` + `easy_runtime_site_artifacts.go` — выбор HTML-шаблона страницы antibot-challenge по имени (v1–v5).
- Созданы шаблоны `antibot-v1.html.tmpl` ... `antibot-v5.html.tmpl` из preview-файлов; при пустом/неизвестном значении используется `antibot.html.tmpl` (оригинал).

### CI / Release

- Исправлен smoke-стек в `scripts/local-ci-preflight.ps1`: runtime HTTP, runtime HTTPS и UI теперь получают разные host-порты, а retry при `port is already allocated` пересоздаёт весь набор портов без пересечения.
- `New-FreeTcpPort` больше не выбирает порт через ephemeral `:0`; вместо этого подбирает свободный порт из диапазона `20000-29999`, чтобы снизить race с повторным использованием ephemeral-портов Docker/Windows.

### UI: Antibot (Tab 6)

- Исправлено: `antibot_challenge_template` не сохранялся — поле отсутствовало в `draftToEasyProfile` в `sites.draft-core.js`.
- Кнопка «Просмотр» рядом с выпадающим «Шаблон страницы проверки» выровнена по нижнему краю поля (`align-items:flex-end`).

### UI: Страницы ошибок (Tab 11)

- Добавлено поле `UseCustomErrorPages bool` в `EasyProfileInput` (compiler/types.go), `EasySiteProfile` (easysiteprofiles/types.go) и маппинг в runtime_apply.go.
- По умолчанию `true` — для всех существующих сайтов кастомные страницы включены автоматически.
- При `UseCustomErrorPages=true` в easy/site.conf.tmpl добавляются `proxy_intercept_errors on` и `error_page` директивы для всех 40+ кодов (400–511, 451→geo_block).
- Добавлен тест `tab11_errorpages_test.go` — проверяет наличие/отсутствие директив.
- Контрактный тест `TestSiteSettings_FieldContract` обновлён для нового поля `use_custom_error_pages`.
- Добавлена вкладка «Error Pages» (Tab 11) в редактор сайта — чекбокс включения кастомных страниц ошибок + список всех 40+ кодов с индивидуальными чекбоксами и кнопками Preview.
- Добавлен API endpoint `GET /api/error-pages/preview/{slug}` в control-plane — отдаёт preview-страницу по slug, доступен только авторизованным пользователям.
- Добавлены i18n-ключи для вкладки Error Pages на всех 5 языках (ru, en, de, sr, zh).
- Затемнение блока списка страниц при отключённом чекбоксе — паттерн из whitelist/blacklist секций.

### UI: Антибот — выбор шаблона страницы проверки

- Добавлено поле `AntibotChallengeTemplate string` в `EasySiteProfile` (easysiteprofiles/types.go) — хранит выбранный шаблон страницы проверки браузера (v1–v5), по умолчанию `"v2"`.
- Поле сохраняется и восстанавливается через draft-слой (draft-core, draft-builder, detail-draft, profile-hydration, draft-profile-part2).
- В блоке антибота на вкладке Security добавлены: select выбора варианта шаблона (1–5 вариант) и кнопка Preview — открывает `/api/error-pages/preview/antibot-vN` в новой вкладке.
- Контрактный тест обновлён для нового поля `security_antibot.antibot_challenge_template`.
- Добавлены i18n-ключи `sites.easy.antibot.challengeTemplate`, `sites.easy.antibot.previewTemplate`, `sites.easy.antibot.template.v1`–`v5` на всех 5 языках.

### Шаблоны страницы браузерной проверки (antibot preview)

- **antibot-v1** — оригинальный шаблон; исправлен редирект в preview-режиме (добавлена проверка `isPreview` перед `window.location.replace`).
- **antibot-v2** — синий/indigo стиль; убрана горизонтальная полоса прогресса; 5 шагов (TLS, отпечаток, угрозы, токен, перенаправление); i18n (en/ru/de/sr/zh); гиперссылка; уникальный фон — угловые indigo/purple свечения + диагональные линии 135°.
- **antibot-v3** — indigo grid карточка со scanning line; 5 шагов; i18n; гиперссылка; исправлен footer ru.
- **antibot-v4** — split layout: timeline слева + SVG circle meter справа; плавное заполнение через `stroke-dashoffset`; равномерные проценты 0→20→40→60→80→100%; последний шаг показывает путь назначения; i18n; гиперссылка.
- **antibot-v5** — янтарный/amber стиль; 3 шага (отпечаток, верификационная cookie, пункт назначения); shimmer-линия внутри блока статусов; заметка про cookie; i18n; уникальный фон teal aurora; градиентная полоса teal→cyan→teal сверху карточки.