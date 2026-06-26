## [1.3.4] - 26.06.2026

### Исправления / Security Modes
- `monitor` больше не применяет security-политики: отключаются ModSecurity, CRS plugins, custom ModSecurity configuration, rate limiting, connection limiting, bad-behavior, blacklist/geo rules, antibot, basic auth и API positive security; в этом режиме остается только обычный сбор логов запросов.
- `transparent` сохраняет полностью прозрачное поведение: runtime не получает security-конфигурацию и не применяет политики.
- Компилятор runtime больше не генерирует `modsecurity`-артефакты и `limit_req`/custom rate-limit зоны для `monitor` и `transparent`.

### Исправления / Backend — дашборд, точный счётчик запросов за сутки
- **Баг: «Запросов за 24 часа» показывал ровно 50 000 при высоком трафике** — корневая причина: если локальный архив или OpenSearch возвращал ровно 50 000 строк (лимит `maxItems`), `summarizeRequests` считал их все и выдавал 50 000 вместо реального числа. Исправлено: добавлен endpoint `GET /requests/count` в runtime, который делает запрос к OpenSearch с `size:0` (нулевая передача документов, только `hits.total.value`) и возвращает точный count. Добавлен интерфейс `runtimeRequestCounter` и метод `collectRequestsDay` в `DashboardService`. При достижении лимита (≥50 000 строк в выборке) `RequestsDay` заменяется точным серверным count; при меньшем трафике используется прежняя логика `summarizeRequests`.
- Fallback: если OpenSearch недоступен — count берётся из локального `.jsonl` архива путём прямого подсчёта строк с фильтром по `since`.

### Исправления / UI — экспорт сервисов
- **Баг: при экспорте выбранных сервисов скачивались два файла — пустой `.env` и `[object Object].json`** — корневая причина: в `sites.stable-resources.js` в вызов `exportSelectedServicesEnvModule` был ошибочно передан лишний аргумент `downloadJSON` между `downloadText` и `draftToEnvText`. Из-за этого `draftToEnvText` (позиция 3) получала функцию `downloadJSON`, а внутри цикла `draftToEnvText(draft)` вызывала `downloadJSON(draft)` — объект становился именем файла → `[object Object].json`. `downloadText` при этом получала `undefined` как контент → пустой `.env`. Исправлено: убран лишний аргумент `downloadJSON` из вызова; импорт `downloadJSON` перенесён из `sites.stable-resources.js` напрямую в `sites.stable-page.js` (из `sites.import-pipeline.js`).

### Исправления / UI — виджет «Сервисы» на дашборде
- **Фильтрация системных сервисов** — `control-plane` и `runtime` больше не отображаются в виджете «Сервисы»; показываются только пользовательские сайты из вкладки «Сервисы», загруженные из `/api/sites`.
- **Варны вместо ошибок** — недоступный сервис (`up: false`) теперь помечается жёлтым `warning` вместо красного `danger` и в виджете, и в модалке.
- **Алерт «Хост недоступен» в модалке** — при ЛКМ на недоступном сервисе между строкой «Проверен» и метриками запросов/атак/заблокированных появляется жёлтый алерт «Хост недоступен».
- **CSS** — добавлен класс `.alert.warning` (жёлтая рамка) рядом с существующим `.alert.success`.
- **i18n** — добавлен ключ `dashboard.services.hostDown` во все 5 локалей (ru/en/de/sr/zh).
# Unreleased

## core
- fixed dashboard 24-hour request totals from OpenSearch so counts above `10,000` are returned exactly instead of stopping at the default search hit threshold
- fixed dashboard request detail widgets to load only the last 24 hours from `/api/requests`, and fixed the unique IP metric so it shows the real distinct count instead of the capped top-20 list
- fixed dashboard attack detail widgets so attacked pages show the target site/host next to each URL
- fixed the requests widget to serve its 24-hour site/page breakdown from `/api/dashboard/stats`, so a failing `/api/requests` no longer blanks the widget detail
