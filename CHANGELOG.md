### Исправления / Security Modes
- `monitor` больше не применяет security-политики: отключаются ModSecurity, CRS plugins, custom ModSecurity configuration, rate limiting, connection limiting, bad-behavior, blacklist/geo rules, antibot, basic auth и API positive security; в этом режиме остается только обычный сбор логов запросов.
- `transparent` сохраняет полностью прозрачное поведение: runtime не получает security-конфигурацию и не применяет политики.
- Компилятор runtime больше не генерирует `modsecurity`-артефакты и `limit_req`/custom rate-limit зоны для `monitor` и `transparent`.

## [1.3.3] - 26.06.2026

### Исправления / Backend — дашборд, точный счётчик запросов за сутки
- **Баг: «Запросов за 24 часа» показывал ровно 50 000 при высоком трафике** — корневая причина: если локальный архив или OpenSearch возвращал ровно 50 000 строк (лимит `maxItems`), `summarizeRequests` считал их все и выдавал 50 000 вместо реального числа. Исправлено: добавлен endpoint `GET /requests/count` в runtime, который делает запрос к OpenSearch с `size:0` (нулевая передача документов, только `hits.total.value`) и возвращает точный count. Добавлен интерфейс `runtimeRequestCounter` и метод `collectRequestsDay` в `DashboardService`. При достижении лимита (≥50 000 строк в выборке) `RequestsDay` заменяется точным серверным count; при меньшем трафике используется прежняя логика `summarizeRequests`.
- Fallback: если OpenSearch недоступен — count берётся из локального `.jsonl` архива путём прямого подсчёта строк с фильтром по `since`.
- Изменены файлы: `runtime/config/launcher/request_opensearch.go`, `runtime/config/launcher/security_events_request_stream_runtime_archive.go`, `runtime/config/launcher/security_events_request_stream_runtime_ingest.go`, `runtime/config/launcher/main.go`, `runtime/config/launcher/metrics.go`, `control-plane/internal/services/runtime_requests.go`, `control-plane/internal/services/dashboard.go`, `control-plane/internal/services/dashboard_snapshot.go`, `control-plane/internal/services/dashboard_test.go`.



### Исправления / UI — экспорт сервисов
- **Баг: при экспорте выбранных сервисов скачивались два файла — пустой `.env` и `[object Object].json`** — корневая причина: в `sites.stable-resources.js` в вызов `exportSelectedServicesEnvModule` был ошибочно передан лишний аргумент `downloadJSON` между `downloadText` и `draftToEnvText`. Из-за этого `draftToEnvText` (позиция 3) получала функцию `downloadJSON`, а внутри цикла `draftToEnvText(draft)` вызывала `downloadJSON(draft)` — объект становился именем файла → `[object Object].json`. `downloadText` при этом получала `undefined` как контент → пустой `.env`. Исправлено: убран лишний аргумент `downloadJSON` из вызова; импорт `downloadJSON` перенесён из `sites.stable-resources.js` напрямую в `sites.stable-page.js` (из `sites.import-pipeline.js`).
- Изменены файлы: `sites.stable-resources.js`, `sites.stable-page.js`.

### Исправления / UI — виджет «Сервисы» на дашборде
- **Фильтрация системных сервисов** — `control-plane` и `runtime` больше не отображаются в виджете «Сервисы»; показываются только пользовательские сайты из вкладки «Сервисы», загруженные из `/api/sites`.
- **Варны вместо ошибок** — недоступный сервис (`up: false`) теперь помечается жёлтым `warning` вместо красного `danger` и в виджете, и в модалке.
- **Алерт «Хост недоступен» в модалке** — при ЛКМ на недоступном сервисе между строкой «Проверен» и метриками запросов/атак/заблокированных появляется жёлтый алерт «Хост недоступен».
- **CSS** — добавлен класс `.alert.warning` (жёлтая рамка) рядом с существующим `.alert.success`.
- **i18n** — добавлен ключ `dashboard.services.hostDown` во все 5 локалей (ru/en/de/sr/zh).
- Изменены файлы: `dashboard.widgets.js`, `dashboard.detail-builder.js`, `dashboard.page-lifecycle.js`, `styles.css`, `i18n/ru.json`, `i18n/en.json`, `i18n/de.json`, `i18n/sr.json`, `i18n/zh.json`.
