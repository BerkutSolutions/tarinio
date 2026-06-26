## [1.3.3] - 26.06.2026

### Исправления / Backend — дашборд, счётчик запросов
- **Баг: «Запросов за 24 часа» показывал ровно 50 000** — корневая причина: runtime создаётся с `maxItems=50000`; при высоком трафике OpenSearch возвращал ровно 50 000 записей (лимит), и все они попадали в окно 24h. `dashboard.go:collectRequests` вызывал `Collect()` без параметра времени. Исправлено: добавлен интерфейс `runtimeRequestCollectorWithOptions`; `collectRequests` теперь вызывает `CollectWithOptions(since=now-25h)` если коллектор его поддерживает — OpenSearch применяет серверный фильтр по времени и возвращает точное количество запросов за последние 24 часа без ограничения лимитом.
- Изменены файлы: `control-plane/internal/services/dashboard.go`.

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