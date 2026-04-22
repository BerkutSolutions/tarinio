## [2.0.6] - 22.04.2026

### Архитектура развёртывания
- Профиль `deploy/compose/default` упрощён до standalone-контура без `Redis` и `ClickHouse`: по умолчанию используются `PostgreSQL`, `OpenSearch`, `Vault`, `runtime`, `ui` и `ddos-model`.
- Для full / HA-сценариев добавлен отдельный профиль `deploy/compose/enterprise` с многонодовым control-plane, `Redis`, `Vault`, `OpenSearch` и `ClickHouse`.
- Добавлен отдельный AIO-сценарий `scripts/install-aio-enterprise.sh` для полной enterprise-версии.

### Логгирование и хранение
- Standalone-модель логгирования переведена на `OpenSearch` как backend по умолчанию для hot и cold хранения, если `ClickHouse` не подключён.
- `ClickHouse` сохранён как отдельный cold analytics tier для enterprise-профиля и HA-сценариев.
- Runtime и control-plane синхронизированы с OpenSearch-only режимом без деградации старой логики и без поломки fallback-контуров.
- `Requests` переведён на backend-first модель: основным источником чтения стал API/runtime с `OpenSearch`, а локальный JSONL-архив больше не используется как пользовательский read-path через `ui/nginx`.
- Локальный request archive сохранён как transient spool и recovery/fallback-слой, а не как основная база запросов для интерфейса.

### Миграции и совместимость
- Усилена миграция legacy request-данных: перед удалением старых записей теперь выполняется проверка, что данные реально появились в целевом хранилище.
- Импорт старого локального request-архива теперь также валидируется до фиксации состояния миграции.
- Standalone fallback с локальным файловым архивом продолжает работать даже при отключённом `ClickHouse`.
- Добавлена verified-миграция legacy `*.jsonl` day-архивов в `OpenSearch` для безопасного обновления с `2.0.5`: сначала import, потом проверка по backend, потом очистка подтверждённого legacy-файла.

### Control-plane и координация
- `Redis` больше не обязателен для standalone и используется только там, где действительно нужен для HA-координации, leader election и сериализации ревизий.
- Конфигурация control-plane обновлена так, чтобы single-node установка не зависела от `Redis`, а HA по-прежнему требовал явный coordinator backend.

### Enterprise identity и доступ
- Добавлен enterprise-контур идентификации с `OIDC`-входом через `/api/auth/providers`, `/api/auth/oidc/start` и `/api/auth/oidc/callback`.
- Добавлен `SCIM 2.0` provisioning через `/scim/v2/*` с bearer-токенами, которые выпускаются и отзываются из раздела `Администрирование`.
- Пользователи теперь хранят `auth_source`, `external_id`, `external_groups` и `last_synced_at`, что позволяет безопасно связывать локальные записи с внешними identity-источниками.
- Реализован маппинг внешних групп в локальные роли через enterprise settings для сценариев `OIDC` и `SCIM`, включая LDAP/AD-backed группы, приходящие из внешнего identity provider.

### Approval workflow и ревизии
- Для ревизий добавлена approval policy с настраиваемым числом подтверждений, запретом self-approval и выделенными reviewer-ролями.
- Ревизии могут переходить в `pending_approval`, а apply блокируется до достижения требуемого порога подтверждений.
- В каталоге ревизий теперь видны compiled-by metadata, approval status, approvals, approved-at и подпись manifest-полей ревизии.
- Добавлен маршрут `POST /api/revisions/{revisionID}/approve` и UI-действие подтверждения ревизии.

### Аудит, доказательства и release artifacts
- Audit log стал tamper-evident: каждое audit-событие теперь хранит `prev_hash` и `hash`, образуя проверяемую цепочку.
- При compile ревизии подписываются её ключевые manifest-поля, а подпись и key ID сохраняются в metadata ревизии.
- Добавлен signed support bundle через `/api/administration/support-bundle`: архив содержит `manifest.json`, `signature.json`, публичный ключ, audits, events, jobs и revisions.
- Manifest support bundle теперь включает `sha256` файлов, summary по audit chain и количество подписанных ревизий.
- Добавлена генерация release artifacts в `build/release/<version>/`: `release-manifest.json`, `signature.json`, `checksums.txt`, `sbom.cdx.json`, `provenance.json` и `release-public-key.pem`.
- `scripts/release.ps1` теперь обязательно генерирует signed release artifacts до публикации.

### Интерфейс и API
- В экране входа добавлена кнопка enterprise SSO, которая автоматически появляется при включённом `OIDC`.
- В разделе `Администрирование` появился отдельный enterprise-блок для `OIDC`, `SCIM`, approval policy и signed support bundle.
- Раздел `Ревизии` теперь показывает статус согласования, автора compile, количество approvals и даёт approve без ручной работы вне интерфейса.
- API-документация дополнена enterprise-маршрутами, approval flow и SCIM surface.
- UI-запросы на `/api/requests` переведены на JSON API-ответ и больше не зависят от прямого чтения локального request archive через reverse proxy.

### Healthcheck и эксплуатация
- Классификация container log issues обновлена так, чтобы healthcheck не поднимал ложные предупреждения по штатным startup-сообщениям `OpenSearch`, `JDK` и single-node plugin noise.
- Убраны ложные `error`-срабатывания на строках `JVM arguments`, `ErrorFile` и `ShowCodeDetailsInExceptionMessages` в логах `OpenSearch`.
- Для `ui` отключено временное файловое буферизование ответа на маршруте fallback `/api/requests`, чтобы `nginx` не создавал `proxy_temp` warning при чтении крупных архивных ответов.
- Маршрут `/api/requests` в `ui/nginx` упрощён до прямого API-proxy без попытки отдавать legacy-файл как первичный источник данных.
- Профиль `deploy/compose/default` повторно проверен через `docker compose up -d --build`: `control-plane`, `runtime`, `ui`, `postgres`, `opensearch`, `vault` и `ddos-model` поднимаются в статусе `healthy`.

### Документация и качество
- Добавлены полноценные документы по enterprise identity и evidence/release model в обеих ветках документации.
- Документация обновлена так, чтобы прямо отражать реальное поведение `OIDC`, `SCIM`, approval workflow, audit chain, support bundle и release artifacts.
- В release/docs consistency test добавлены проверки на наличие enterprise-документов и на обязательную генерацию release artifacts в `scripts/release.ps1`.
- Добавлены регрессионные тесты на фильтрацию benign container log issues для `OpenSearch` и на сохранение детекта реальных ошибок.
- Полный прогон `go test ./...` проходит успешно.
- `npm run docs:build` проходит успешно.
