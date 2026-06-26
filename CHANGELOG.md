## [1.3.1] - 26.06.2026

### Healthcheck
- Healthcheck-классификатор контейнерных логов больше не помечает как warning штатные nginx-записи `a client request body is buffered to a temporary file` (POST-запросы вроде sentry envelope, тело которых превышает `client_body_buffer_size`). Раньше эти строки сыпались в healthcheck по сервису `sentry.hantico.ru`, хотя самим запросам они не мешают.
- В whitelist `classifyContainerLogIssue` добавлен паттерн `querygroup _id can't be null, it should be set before accessing it`. Это известное benign-предупреждение OpenSearch 2.18 (cluster bug в плагине workload management), не влияющее на работу WAF, но сыпавшее в healthcheck сотни одинаковых записей при каждом старте кластера.

### Хранилище логов
- Реализована автоматическая чистка горячего OpenSearch-индекса по retention-настройкам из UI. Раньше дневные сегменты `waf-hot-requests` удалялись только когда холодное хранилище = ClickHouse (через `migrateHotToColdLocked`). При сценарии «и горячее, и холодное — OpenSearch» (типичный prod) старые дни накапливались бесконечно — мы наблюдали 65 дневных сегментов на проде при настройках 14/30 дней. Новый `pruneOpenSearchOldDaysLocked` запускается на каждом цикле фонового ingest и удаляет дни старше:
  - `Retention.ColdDays`, если cold backend = OpenSearch (все данные хранятся в горячем индексе);
  - `Retention.HotDays`, если cold backend = ClickHouse/file (страховка поверх миграции, чтобы OpenSearch никогда не держал данные дольше горячего окна).
- Уточнён статус «Хранилище логов» в UI: для single-OpenSearch теперь показывается оба числа `{hotDays}/{coldDays}`, а не только cold. Раньше при настройках 14/30 строка отображала «срок хранения 30 дней», что путало.

### UI / Хранилище логов
- В таблице «Индексы хранилища логов» дни старше `Retention.HotDays` теперь подписываются как «холодные» (когда cold backend = OpenSearch и данные физически живут в одном индексе, но по возрасту уже относятся к cold-горизонту). Раньше все дни помечались как `opensearch:waf-hot-requests` независимо от возраста.
- Колонка «Файл» теперь рендерит уровень хранилища через i18n-ключи (`OpenSearch (горячее)`, `OpenSearch (холодное)`, `ClickHouse (холодное)`) вместо сырого имени индекса.
- Колонка «Размер» переведена с байтов в человекочитаемый формат (B / KB / MB / GB), что соответствует строкам в десятки и сотни мегабайт на дневной индекс.

### UI / Локализация дат
- Время во вкладке «События» и в TLS теперь форматируется через `formatDateTimeInZone`, который привязан к языку UI (`ru-RU` / `en-US` / `de-DE` / `sr-Cyrl-RS` / `zh-CN`) и часовому поясу пользователя. Раньше `events.js` рендерил сырое значение `occurred_at` (`2026-06-26T10:17:19.730832709Z`), а `ui.js` `formatDate` использовал `toLocaleDateString()` без явной локали, поэтому даты в TLS и других страницах отличались от языка UI.
- В `preferences.js` `formatDateTimeInZone` теперь явно использует `Intl.DateTimeFormat(locale, ...)` с локалью, выведенной из активного языка приложения. `loadPreferences` пересобирает кеш при смене языка через `setLanguage`, иначе `formatDate` оставался залипшим на старой локали до перезагрузки страницы.
- Во вкладке «События» сводки `apply started` / `apply succeeded` / `apply failed`, а также соответствующие типы `apply_started` / `apply_succeeded` / `apply_failed` / `reload_failed` / `health_check_failed` / `rollback_performed` теперь переведены на язык интерфейса. Раньше отсутствовали i18n-ключи, и UI показывал сырой английский токен из payload.

### Ревизии
- Эндпоинт `/api/revisions/compile` теперь принимает опциональное поле `target_site_ids` в теле запроса. UI-форма редактора сервиса передаёт туда id текущего сохраняемого сервиса, поэтому ревизия помечает «затронут только этот сервис». Раньше `revision_catalog_site_scoping.go` восстанавливал список «затронутых сервисов» по diff между fingerprint текущей и предыдущей ревизии — для первой ревизии в свежей БД (`rev-000122`) предыдущей не существовало, поэтому в неё попадали ВСЕ сайты, и на проде сохранение одного сервиса показывало «затронуты: waf.hantico.ru, sentry.hantico.ru».
- Когда `target_site_ids` не передан (legacy paths: import, onboarding, bulk), сохранён прежний fingerprint-diff алгоритм без изменений.

### Локализация
- Во всех пяти локалях (`ru`, `en`, `de`, `sr`, `zh`) добавлены ключи `settings.storage.indexes.tier.opensearch_hot`, `settings.storage.indexes.tier.opensearch_cold`, `settings.storage.indexes.tier.clickhouse`, `settings.logging.status.opensearch_full`. Подпись колонки `settings.storage.indexes.col.size` приведена к виду без уточнения «(байт)», поскольку UI теперь сам выбирает единицы.
- Добавлены ключи `events.type.apply_*`, `events.type.reload_failed`, `events.type.health_check_failed`, `events.type.rollback_performed` и зеркальные `events.summary.*`.
- Удалён случайный дубликат ключа `events.type.security_access` в `en.json`.

### Тесты
- Расширен `TestClassifyContainerLogIssue_IgnoresBenignOpenSearchStartupNoise` кейсами для `a client request body is buffered to a temporary file` (sentry envelope POST) и `[WARN ][o.o.w.QueryGroupTask] QueryGroup _id can't be null`.
- `TestRequestStreamPrunesOpenSearchWhenColdIsOpenSearch` проверяет, что при cold=OpenSearch старые дни (возраст > ColdDays) удаляются автоматически.
- `TestRequestStreamPrunesOpenSearchByHotDaysWhenColdIsClickHouse` проверяет страховку: при cold=ClickHouse дни старше HotDays вычищаются из OpenSearch независимо от миграции.