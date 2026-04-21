## [2.0.2] - 21.04.2026

### PostgreSQL / Хранилище
- Control-plane state переведён с локальных JSON-store на PostgreSQL-backed storage при наличии `POSTGRES_DSN`.
- Добавлен встроенный startup-migration path: на первом запуске legacy-данные автоматически и безопасно переносятся из volume/file state в PostgreSQL без удаления исходных файлов.
- В PostgreSQL теперь сохраняются основные control-plane сущности: `users`, `roles`, `sessions`, `passkeys`, `sites`, `upstreams`, `certificates`, `tls_configs`, `waf_policies`, `access_policies`, `rate_limit_policies`, `easy_site_profiles`, `anti_ddos`, `events`, `audit_events`, `jobs`, `revisions`.
- В PostgreSQL также перенесены `certificate-materials`, `revision-snapshots`, runtime settings и настройки TLS auto-renew.
- Добавлен backend универсального хранения `state_documents` / `state_blobs` для JSON-документов и бинарных артефактов control-plane.
- Добавлены SQL migrations и schema version tracking через `goose`-style versioned migration files и встроенный migration runner.
- Startup migration сделан non-destructive: legacy volume/state остаются на месте как rollback/fallback артефакт.
- Исправлен migration walker, чтобы бинарные файлы snapshot/certificate store не дублировались в document-таблицу.

### Развёртывание / Compose / Installer
- `deploy/compose/default` обновлён для штатной работы с PostgreSQL как основным persistent storage.
- Control-plane в compose-профилях теперь получает DSN, собранный из тех же `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB`, что использует postgres service, чтобы исключить рассинхрон паролей между приложением и БД.
- Обновлены compose-профили `default`, `auto-start` и `testpage`, чтобы bundled PostgreSQL поднимался как часть поддерживаемого storage path, а не как декларация в документации без реального backing store.
- `scripts/install-aio.sh` теперь остаётся совместимым с PostgreSQL-first rollout: upgrade path сохраняет volumes, не удаляет legacy state и позволяет control-plane выполнить безопасный first-boot перенос данных в БД.
- Лёгкий backup path в installer по-прежнему включает `postgres-data`, `control-plane-data`, `certificates-data`, `runtime-data` и остальные важные volumes перед upgrade.

### Образ PostgreSQL / Усиление безопасности
- `postgres:15-alpine` в основных compose-профилях заменён на Debian-based `postgres:15`, чтобы убрать предупреждение про отсутствующие system locales.
- Для bundled PostgreSQL добавлены `POSTGRES_INITDB_ARGS` с `--auth-local=scram-sha-256 --auth-host=scram-sha-256 --locale=C.UTF-8`.
- Убрано предупреждение `initdb: warning: enabling "trust" authentication for local connections` на чистой инициализации PostgreSQL.
- Убрано предупреждение `WARNING: no usable system locales were found` на чистой инициализации PostgreSQL.

### Сборка / Метаданные релиза
- Версия продукта обновлена до `2.0.2` в backend metadata, UI, i18n-строках, wiki baseline и release-synced документации.
- `control-plane/Dockerfile` обновлён на `golang:1.25-bookworm` для соответствия текущему `go.mod` и стабильной сборки control-plane image.

### Проверка
- Пройден `go test ./...` после перевода storage на PostgreSQL.
- На `deploy/compose/default` подтверждён рабочий first-boot сценарий: после сохранения legacy volumes и пересоздания только `waf-postgres-data` стек поднимается, control-plane становится healthy, а legacy state оказывается в PostgreSQL.
- Подтверждено наличие migrated данных в PostgreSQL для documents/blobs, включая `users`, `roles`, `sites`, `revisions`, `runtime_settings`, `revision snapshots` и `certificate materials`.

### Observability / Мониторинг
- Добавлены встроенные Prometheus-compatible metrics для control-plane: HTTP requests, latency histograms, compile/apply outcomes, HA lock acquisition, leader-only tasks, runtime reload operations, build info.
- Добавлены встроенные Prometheus-compatible metrics для runtime launcher: HTTP requests, latency histograms, reload outcomes, bundle load outcomes, readiness/liveness gauges, active revision gauge.
- Metrics endpoints защищены токенами `CONTROL_PLANE_METRICS_TOKEN` и `WAF_RUNTIME_METRICS_TOKEN`; поддерживаются `X-TARINIO-Metrics-Token` и query-параметр `?token=...`.
- В `deploy/compose/ha-lab` добавлен полноценный observability profile: `prometheus`, `grafana`, `postgres-exporter`, `redis-exporter`.
- Добавлен provisioned Grafana dashboard для HA и runtime/operator обзора.
- `scripts/install-aio.sh` теперь поддерживает строгую post-upgrade проверку через `RUN_STRICT_POST_UPGRADE_VALIDATION=1` и `scripts/post-upgrade-smoke.sh`, включая контроль `/metrics` при наличии токенов.

### Rolling Upgrade / Zero-Downtime Upgrade
- Для HA lab добавлены `deploy/compose/ha-lab/upgrade/rolling-upgrade.ps1` и `rolling-upgrade.sh`.
- Rolling upgrade сценарий rebuild'ит `control-plane-a` и `control-plane-b` по одной ноде за раз через `api-lb`, одновременно непрерывно проверяя `http://api-lb:8080/healthz`.
- Upgrade helper считает релиз неуспешным, если во время поочерёдного обновления control-plane наблюдается потеря API availability.
- Wiki upgrade guide обновлён под rolling / zero-downtime sequence и обязательный strict smoke validation после релиза.

### Benchmark Pack / Публичные бенчмарки
- Добавлен public benchmark pack в `deploy/compose/ha-lab`: `benchmark-http.sh`, `benchmark-pack.sh`, `benchmarks/run.ps1`, `benchmarks/run.sh`.
- Бенчмарки сохраняют timestamped JSON summaries с `total`, distribution по status code, `avg`, `min`, `max`, `p50`, `p95`, `p99`.
- В benchmark pack добавлены сценарии baseline tenant traffic, rate-limit protected tenant traffic и API health checks через `api-lb`.
- При включённом observability profile benchmark pack также фиксирует health Prometheus targets для публикации и regression tracking.

### HA Limits / Границы HA
- HA lab documentation уточнена фактическими ресурсными лимитами при включённых `tools` и `observability`: около `8.0 GB` RAM и `7.75` CPU.
- В HA wiki и lab README добавлены подтверждённые сценарии: leader-elected background tasks, serialized compile/apply, rolling upgrade without API downtime, benchmark and observability workflows.

### Docs Site / GitHub Pages
- Документационный портал переведён с `MkDocs` на `Docusaurus`.
- Добавлены `package.json`, `docusaurus.config.js`, `package-lock.json`, `src/pages/*`, `src/css/custom.css`, `static/img/*` и набор sidebar-конфигов для `ru`, `en`, `architecture-source`, `operator-source`.
- GitHub Pages workflow `.github/workflows/docs-pages.yml` переведён на Node-based сборку `Docusaurus` через `npm ci` и `npm run docs:build`.
- Публичный docs-site теперь собирает текущие деревья `docs/ru`, `docs/eng`, `docs/architecture`, `docs/operators` как отдельные Docusaurus docs roots без дублирования их в ещё одной wiki-структуре.
- В верхней панели портала добавлены отдельные dropdown-переключатели языка и текущей версии, а также локальный поиск по документации.
- Исправлено дублирование нижнего блока навигации на route-based страницах портала: `docs-overview` и `cli-commands` переведены с вложенных MDX-layout страниц на обычные Docusaurus page routes.
- Главная страница портала упрощена: убраны дублирующие карточки входа, оставлены только hero-блок и прямые переходы в русскую wiki, английскую wiki и навигатор.
- Устранена ошибка dev-сервера `Can't resolve '@theme/SearchPage'`: добавлен локальный theme bridge для search page и подтверждена успешная сборка `Docusaurus`.
- Архитектурные source-документы и operator source notes перенесены внутрь веток `ru` и `en` как подкатегории wiki вместо отдельных верхнеуровневых docs roots.
- Для русской wiki добавлены зеркальные разделы `Архитектурные документы` и `Исходные операторские заметки`, чтобы source-уровень был доступен из того же дерева, что и deploy, HA, troubleshooting, operators и OSS.

### Enterprise Docs / Wiki IA
- Документация расширена полноценным enterprise-пакетом в английской и русской ветках: support lifecycle, compatibility matrix, sizing, troubleshooting, disaster recovery, hardening, threat model, reference architectures, cookbook, release notes policy, known limitations, compliance mapping.
- Для ранее англоязычных разделов `observability` и `benchmarks` добавлены русские зеркала.
- Добавлены отдельные `navigator`-страницы для русской и английской веток, чтобы пользователю было проще понимать, куда идти в зависимости от задачи.
- Навигация Docusaurus перестроена по сценариям: старт, развёртывание, эксплуатация, безопасность и доверие, operator guides.
- В портал подключены архитектурные ADR и корневые operator source notes, чтобы docs-site покрывал весь значимый набор документации, а не только часть страниц.
- Верхняя навигация переведена в двуязычный формат, а русская и английская wiki-ветки вынесены в отдельные самостоятельные разделы портала.

### Release Preflight / Контроль wiki
- `scripts/release.sh` и `scripts/release.ps1` теперь перед релизом обязательно выполняют `go test ./...` и `npm run docs:build`.
- Добавлен тест `ui/tests/docs_wiki_nav_test.go`, который проверяет, что новые markdown-документы из `docs/` лежат внутри поддерживаемых Docusaurus docs roots; если новый документ положили вне wiki-веток, тест падает.

### CI / GitHub Actions / Ignore
- `smoke-e2e` workflow изолирован от старых Docker volumes: default compose теперь поддерживает `WAF_STACK_NAME`, а workflow поднимает стек в отдельном prefixed namespace и перед стартом выполняет `down -v --remove-orphans`.
- `smoke-e2e` workflow обновлён на Node 24 compatible GitHub Actions (`actions/checkout@v5`, `actions/setup-go@v6`) и получил более явный readiness wait перед e2e smoke.
- `docs-pages` workflow обновлён на `actions/checkout@v5` и `actions/setup-node@v6` с Node `24`, чтобы убрать предупреждения о deprecation Node 20.
- `.gitignore` и `.dockerignore` дополнены для generated docs/site state: `site/`, `.docusaurus/`, `node_modules/.cache/`, `.turbo/`.
- Из репозитория удалены устаревшие docs sidebar files `sidebars.architecture.js` и `sidebars.operators.js`, которые больше не используются после переноса architecture/operator source внутрь `ru` и `en` wiki.
