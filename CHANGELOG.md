## [1.3.2] - 26.06.2026

### Хранилище логов
- Реализована автоматическая чистка горячего OpenSearch-индекса по retention-настройкам из UI. Раньше дневные сегменты `waf-hot-requests` удалялись только когда холодное хранилище = ClickHouse (через `migrateHotToColdLocked`). При сценарии «и горячее, и холодное — OpenSearch» (типичный prod) старые дни накапливались бесконечно — мы наблюдали 65 дневных сегментов на проде при настройках 14/30 дней. Новый `pruneOpenSearchOldDaysLocked` запускается на каждом цикле фонового ingest и удаляет дни старше:
  - `Retention.ColdDays`, если cold backend = OpenSearch (все данные хранятся в горячем индексе);
  - `Retention.HotDays`, если cold backend = ClickHouse/file (страховка поверх миграции, чтобы OpenSearch никогда не держал данные дольше горячего окна).
- Уточнён статус «Хранилище логов» в UI: для single-OpenSearch теперь показывается оба числа `{hotDays}/{coldDays}`, а не только cold. Раньше при настройках 14/30 строка отображала «срок хранения 30 дней», что путало.

### Локализация
- Во всех пяти локалях (`ru`, `en`, `de`, `sr`, `zh`) добавлен новый ключ `settings.logging.status.opensearch_full` для совмещённого hot+cold OpenSearch-режима. Старый ключ `settings.logging.status.opensearch` остаётся для смешанных сценариев.

### Тесты
- `TestRequestStreamPrunesOpenSearchWhenColdIsOpenSearch` проверяет, что при cold=OpenSearch старые дни (возраст > ColdDays) удаляются автоматически.
- `TestRequestStreamPrunesOpenSearchByHotDaysWhenColdIsClickHouse` проверяет страховку: при cold=ClickHouse дни старше HotDays вычищаются из OpenSearch независимо от миграции.

## [1.3.1] - 26.06.2026

### Healthcheck
- Healthcheck-классификатор контейнерных логов больше не помечает как warning штатные nginx-записи `a client request body is buffered to a temporary file` (POST-запросы вроде sentry envelope, тело которых превышает `client_body_buffer_size`). Раньше эти строки сыпались в healthcheck по сервису `sentry.hantico.ru`, хотя самим запросам они не мешают.

### Тесты
- Расширен `TestClassifyContainerLogIssue_IgnoresBenignOpenSearchStartupNoise` кейсом для строки `a client request body is buffered to a temporary file ... server: sentry.hantico.ru, request: "POST /api/2/envelope/..."`, чтобы этот паттерн больше не возвращался в healthcheck при следующих правках.

## [1.3.0] - 26.06.2026

### UI / Help-методички для редактора сервиса
- Каждая глава левого меню редактора сервиса (front, upstream, http, headers, blocking, антибот+аутентификация, страновая политика, modsec) получила собственный заголовок с подзаголовком и кнопку справки `?` рядом. По клику открывается модалка-методичка «поле → описание»: для каждого поля объясняется зачем оно нужно и что в него вписывать.
- Содержимое уже существующих frame-help-модалок (Плохое поведение, Лимиты, Списки и DNSBL, Прокидывание заголовков, антибот, аутентификация) переписано из «технического справочника» в полноценную методичку, чтобы операторы понимали смысл и значения, а не только название параметра.
- Введён общий шаблон модалок (`sites.help-modal-shell.js`) и два новых модуля: `sites.chapter-help-modals.js` для глав и обновлённый `sites.frame-help-modals.js` для фреймов. Старые рендеры антибота и аутентификации (`sites.auth-help-modals.js`) перенесены на тот же shell.
- Локализация полностью покрыта во всех пяти языках (`ru`, `en`, `de`, `sr`, `zh`): добавлено по 198 ключей. RU прошёл `TestI18NNoArtifacts` (запрещает примеси английских слов), для технических терминов вроде HSTS, CSP, QUIC, X-Forwarded-For добавлен whitelist.

### UI / Help-модалки фреймов
- Исправлен фон и читаемость модалки справки `?` у фреймов антибота и аутентификации: добавлены стили `.waf-modal-backdrop` и `.waf-modal-dialog` (тёмный фон, рамка, sticky-шапка таблицы), которые раньше не существовали и модалка отрисовывалась без фона.
- В каждом подраздел-фрейме внутри глав `Контроль трафика` и `Upstream` теперь есть свой заголовок с подзаголовком и кнопка `?` со справочной модалкой:
  - `Контроль трафика → Плохое поведение` (subtitle + ?)
  - `Контроль трафика → Лимиты` (subtitle + ?)
  - `Контроль трафика → Списки и DNSBL` (subtitle + ?)
  - `Upstream → Прокидывание заголовков` (subtitle + ?)
- Новый модуль `ui/app/static/js/pages/sites.frame-help-modals.js` собирает шаблон help-модалки (поле → описание) и экспортирует 4 рендера; модалки и кнопки прокинуты через `sites.stable-renderers.js`, события открытия — через `sites.detail-events-rules.js`.
- В i18n (`ru/en/de/sr/zh`) добавлены subtitle и пары `<frame>.help.{open,title,subtitle,*}` для каждого нового фрейма. Дополнительно дозалиты ранее отсутствовавшие ключи antibot/auth help-модалок и сервисных токенов, которые уже использовались в коде.

### Testpage
- Исправлен management API upstream в `deploy/compose/testpage`: management runtime теперь компилирует `/api/*` на `control-plane`, а app runtime-test на `control-plane-test`, поэтому тестовый стек больше не должен падать на `host not found in upstream "control-plane"` при старте runtime-контейнеров.
- Профиль `deploy/compose/testpage` приведён к ожидаемому bootstrap-поведению для локальной проверки: protected app stack на `https://localhost:8081` теперь хранит bootstrap-сервис в файловом state, поэтому UI может сразу показать и открыть второй тестовый сервис без регистрации и ручного создания. Дополнительно добавлен `/api-app/certificates` alias и контейнеры testpage переименованы в явный префикс `tarinio-testpage-*`.

### Аутентификация
- Добавлен расширенный блок настройки аутентификации: режим `логин/пароль`, `сервисный токен` или смешанный режим, а также порядок запуска `аутентификация -> anti-bot` либо `anti-bot -> аутентификация`.
- Для аутентификации добавлены URL-исключения по путям и HTTP-методам, чтобы безопасно выводить из login-wall машинные API, webhook и ingest-эндпоинты без отключения защиты на всём сервисе.
- Easy nginx-конфигурация теперь поддерживает отдельные verify-endpoint'ы для basic и service-token сценариев, а auth-страница корректно работает с обоими типами входа.

### Проверки
- Добавлены и обновлены compiler/UI contract-тесты для расширенной аутентификации, порядка auth/anti-bot, сервисных токенов, help-модалок и новых контрактных маркеров страницы сервисов.

### UI
- Восстановлен стабильный bridge импорта в форму создания сервиса: `sites.stable-page.js` снова экспортирует `pendingImportedDraftRef`, поэтому импорт сервиса корректно передаёт распарсенный draft в `/services/new`.
- Исправлены help-кнопки сервисов в стабильном facade-пути: detail rule events снова привязываются, поэтому кнопки `?` у auth и anti-bot открывают свои help-модалки по клику.
- Исправлена загрузка facade сервисов после разделения: `sites.stable-page.js` теперь импортирует geo-catalog-хелперы из `sites.geo-lists.js`, а загрузчик приложения оборачивает ошибки импорта facade в подробный `ServicesStableFacadeLoadError`.
- Завершён рефакторинг сервисов: `sites.js` теперь однострочный facade, а основная логика стабильного renderer'а живёт в небольших модулях `sites.stable-*` (errors, resources, renderers, detail binding, page orchestration); сломанный путь `sites.page-main-runtime.js` явно помечен как неиспользуемый.
- Исправлено падение вкладки сервисов (`/services`) с `Unexpected token ':'`: модульный рендер страниц сервисов снова отдает корректные ES-модули без битых `export`-алиасов и незакрытых шаблонных строк.
- Исправлена загрузка списка сервисов после перехода на модульный runtime: `sites.page-render-runtime.js` снова прокидывает `normalizeSiteID` в `loadSitesRuntime`, поэтому вкладка `Services` не падает в общий `Failed to load services`.
- Исправлен рендер списка сервисов в модульном helper-слое: `sites.page-main-helpers.js` снова импортирует `formatCertificateExpiryByLanguage`, `certificateDaysLeft` и `formatServiceProfile`, поэтому список сервисов не падает на этапе построения таблицы.
- Исправлен detail-экран сервисов (`/services/:id`): модульный runtime снова передаёт `feedback`-узел и полный набор auth/anti-bot normalizer-зависимостей в `bindDetailRuntime`, поэтому создание и редактирование сервисов не падают при открытии внутренней формы.
- Исправлено открытие management detail-маршрута `https://localhost:8080/services/testpage-mgmt-localhost-site`: runtime bridge снова прокидывает в detail view недостающие auth help/auth token/auth exclusion зависимости, а `sites.detail-bind-runtime.js` снова вызывает `syncDerivedFieldsFromID` в исходном порядке аргументов.
- Исправлен `Failed to load services` на `https://localhost:8080/services/new` и `https://localhost:8080/services/testpage-mgmt-localhost-site`: helper-слой `sites.page-main-helpers.js` снова импортирует auth help/auth exclusion/auth token renderers, поэтому create/detail-рендер больше не падает на `ReferenceError`.
- Для runtime-вкладки `Services` добавлена расширенная диагностика загрузки: `sites.runtime-load-list.js` теперь пишет структурированный `console.error("[sites-runtime]", ...)` и показывает stage + текст ошибки прямо в alert вместо одного общего `Failed to load services`.
- Исправлен `ReferenceError: siteDraftFromData is not defined` на открытии `Services` detail/create runtime-страниц: `sites.draft-profile-part2.js` теперь берёт `siteDraftFromData` из `deps`, как и остальные bridge-зависимости после модульного разбиения.
- Исправлен `ReferenceError: formatBanDurationSeconds is not defined` на runtime-detail странице `Services`: `sites.page-main-helpers.js` снова импортирует `formatBanDurationSeconds` из `sites.page-main-core.js`, поэтому render bridge для escalation stages больше не падает.
- Исправлен `ReferenceError: renderAuthHelpModalSafe is not defined` на runtime-detail странице `Services`: `sites.page-main-helpers.js` теперь передаёт в detail-render уже импортированные safe-renderers `renderAuthHelpModal` и `renderAntibotHelpModal`, вместо несуществующих локальных alias-имён.
- Для `Services` добавлен принудительный cache-bust фронтенд-модулей: `index.html` переведён на `app.js?v=20260625-services-hotfix-11`, а `PAGE_MODULE_VERSION` в `app.js` увеличен до `20260625-04`, чтобы браузер не держал старые runtime JS-модули после hotfix-пересборок.
- `Services` переключён обратно на стабильный renderer `sites.js`, который считаем эталонным по рабочему поведению уровня `v1.2.6/1.2.7`; экспериментальный `sites.page-main-runtime.js` оставлен в репозитории, но явно помечен как legacy-broken compatibility path и не используется навигацией приложения.
- В стабильном `Services`-renderer завершён безопасный bridge-этап для detail-экрана: `sites.js` теперь делегирует `renderDetailView` в модульный `sites.detail-render-view*.js`, а старый монолитный шаблон оставлен только как закомментированный legacy-блок до финального удаления после следующих чистящих проходов.
- Исправлено добавление anti-bot исключений: теперь в редакторе можно последовательно создавать несколько правил, даже если новые строки ещё не заполнены.
- В модальном окне ревизий сервиса добавлена красная кнопка удаления всех старых ревизий, кроме текущей применённой.
- Исправлена нормализация management-сайта в списке сервисов: alias `localhost`, `ui` и `control-plane` теперь схлопываются к `control-plane-access`, поэтому в auto-start больше не должно появляться визуальных дублей одной и той же системной записи.
- Страница сервисов переведена на новый модульный entrypoint: UI больше не должен брать legacy-рендер `sites.js`, поэтому во вкладке `Anti-bot and Authentication` теперь должны появляться help-кнопки, расширенные режимы аутентификации, порядок `auth/anti-bot`, URL-исключения и сервисные токены.

### Runtime
- Для easy nginx-конфига добавлен `client_body_buffer_size 512k`, чтобы POST-запросы вроде sentry envelope реже буферизовались во временные файлы и не засоряли healthcheck предупреждениями.
- Исправлено падение smoke-прогона на `/api/requests`: при отсутствии OpenSearch/ClickHouse-секретов system logging теперь корректно остаётся на локальном archive fallback и не уводит burst smoke в `502 Bad Gateway`.
- Исправлен запуск `deploy/compose/testpage`: контейнер `request-archive` больше не зависит от `apk add` во время сборки и теперь стартует на чистом `alpine:3.20` с `busybox`-утилитами. Дополнительно для runtime/test-runtime снят блокирующий `read-only` с корневого `WAF_RUNTIME_ROOT`-тома в testpage-профиле и добавлен dev-default для `CONTROL_PLANE_SECURITY_PEPPER`, чтобы стенд поднимался из коробки.
- Исправлены healthcheck-warning'и nginx для `tarinio-testpage-runtime-mgmt` и `tarinio-testpage-runtime-app`: в `compiler/templates/nginx/nginx.conf.tmpl` добавлены `proxy_headers_hash_max_size 1024;` и `proxy_headers_hash_bucket_size 128;`, чтобы runtime больше не логировал `could not build optimal proxy_headers_hash`.

### Тесты
- Добавлен compiler-тест на `client_body_buffer_size` и UI contract-тест на новую bulk-delete кнопку ревизий.
- Добавлены unit-тесты на fallback request-stream/runtime logging, чтобы отсутствие backend-секретов больше не ломало smoke e2e.
- Добавлен `ui/tests/sites_runtime_bridge_contract_test.go`, чтобы потеря detail-runtime зависимостей или нарушение wiring для `syncDerivedFieldsFromID` больше не проходили незамеченно.
- Добавлен compiler-тест `nginx_conf_hash_settings_test.go`, чтобы `proxy_headers_hash_*`-настройки runtime nginx больше не терялись при следующих правках.
- Добавлен UI contract-check на runtime-диагностику `Services`, чтобы подробный вывод ошибки и runtime logging не потерялись при следующих переносах.
- Добавлен UI contract-check на bridge-зависимость `deps.siteDraftFromData`, чтобы `hydrateSiteDraftPart2` больше не падал на detail/create маршрутах runtime-вкладки `Services`.
- Расширен UI contract-check на `formatBanDurationSeconds`, чтобы helper/runtime bridge для detail-render не терял formatter ban escalation при следующих переносах.
- Обновлён UI contract-check для auth help modals, чтобы helper bridge использовал реальные импортированные renderer-символы, а не отсутствующие локальные alias-имена.
- Добавлен UI contract-check на стабильный загрузчик `Services`, чтобы навигация держалась за `sites.js`, а broken runtime path оставался только как помеченный legacy-эксперимент.
