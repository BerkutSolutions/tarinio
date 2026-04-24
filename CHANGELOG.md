## [3.0.0] - 24.04.2026

### Архитектура Sentinel
- Добавлен новый пакет `internal/sentinel` как общее ядро adaptive-движка (эволюция `ddos-model`).
- Добавлен новый entrypoint `cmd/tarinio-sentinel/main.go`.
- Три профиля `ddos-model` (`default`, `auto-start`, `testpage`) переведены на тонкие обёртки, использующие общее ядро `internal/sentinel`.

### Scoring и Explainability
- В state добавлены `risk_score`, `trust_score`, `reason_codes`, `top_signals`.
- В scoring добавлены дополнительные сигналы: `rps_per_ip`, `404_ratio`, `scanner_path_hits`, `ua_risk`, `unique_paths`, `blocked_ratio`.
- Сохранено экспоненциальное затухание score (decay) с переносом на signal-contributions.

### Adaptive Output
- Расширен `adaptive.json` backward-compatible полями explainability (`score`, `trust_score`, `reason_codes`, `top_signals`) без удаления существующих полей.
- Добавлена поддержка `max_published_entries` (публикация только top-N по score для снижения нагрузки на `l4guard`).
- Добавлены поля `source`, `first_seen`, `last_seen` в adaptive entries без поломки текущего контракта.
- Добавлен publish batching: adaptive-файл пишется только при изменениях и не чаще `MODEL_PUBLISH_INTERVAL_SECONDS`.

### L7 Suggestions (MVP)
- В `internal/sentinel` добавлена детекция hot scanner paths (`/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit` и др.) и генерация `l7-suggestions.json`.
- Добавлен отдельный control-plane модуль `control-plane/internal/antiddossuggestions` (file/postgres store) для хранения кандидатов L7-правил.
- Добавлен API для кандидатов:
  - `GET /api/anti-ddos/rule-suggestions`
  - `POST|PUT /api/anti-ddos/rule-suggestions`
  - `POST|PUT /api/anti-ddos/rule-suggestions/{id}/status`
- На MVP поддержаны статусы `suggested` и `shadow` без auto-apply в runtime.
- Для `shadow`-кандидатов добавлены счетчики `shadow_hits`, `shadow_false_positive_hits`, `shadow_false_positive_rate` без активного блокирования.

### Sentinel Runtime Modes
- Добавлен интерфейс источников событий `internal/sentinel/source` с production file-tail backend и Redis-заглушкой для будущего HA stream режима.
- Добавлен `MODEL_SOURCE_BACKEND` (`file` по умолчанию, `redis` зарезервирован для HA без полной реализации).
- Runtime profile `ddos-model/config.json` расширен `model_enabled_sites`, а sentinel фильтрует анализ по включенным сайтам/хостам.

### UI
- В easy-profile `front_service` добавлен флаг `adaptive_model_enabled`.
- На вкладке "Фронтовый сервис" добавлен чек-бокс включения adaptive model для конкретного сервиса.
- Для `control-plane-access` default profile модель включена, для новых/импортированных сервисов в UI выключена по умолчанию.

### Контейнеризация Sentinel
- В `deploy/compose/default/docker-compose.yml` сервис `ddos-model` заменен на `tarinio-sentinel` с контейнером `tarinio-sentinel`.
- В `deploy/compose/ha-lab/docker-compose.yml` сервис `ddos-model` заменен на `tarinio-sentinel` с контейнером `tarinio-ha-sentinel`.
- Добавлены переменные `MODEL_SOURCE_BACKEND`, `MODEL_SUGGESTIONS_OUTPUT_PATH`, `MODEL_PUBLISH_INTERVAL_SECONDS`, `MODEL_MAX_ACTIVE_IPS`, `MODEL_MAX_PUBLISHED_ENTRIES`, `MODEL_MAX_ACTIONS_PER_MINUTE`.
- Добавлены отдельные state volumes `waf-sentinel-state` и `waf-ha-sentinel-state`; adaptive output volume остается совместимым с `l4guard`.
- Для default profile задан ресурсный ориентир `0.5 CPU / 256MB`, для HA lab - `0.75 CPU / 512MB`.

### Документация и Wiki
- Добавлена operator wiki-страница `docs/ru/operators/tarinio-sentinel.md`.
- Добавлена operator wiki-страница `docs/eng/operators/tarinio-sentinel.md`.
- Обновлены RU/EN индексы документации ссылками на TARINIO Sentinel.
- Расширен `docs/eng/operators/anti-ddos-runbook.md` операционными проверками `tarinio-sentinel`.
- Расширен `docs/ru/operators/anti-ddos-runbook.md` ссылкой и командами проверки `tarinio-sentinel`.
- В `docs/ru/benchmarks.md` и `docs/eng/benchmarks.md` добавлена enterprise test matrix для Smart WAF: normal traffic, scanner, brute force, single-source flood, distributed flood, high cardinality.
- Added reproducible enterprise validation reports: `docs/eng/operators/sentinel-enterprise-validation.md` and `docs/ru/operators/sentinel-enterprise-validation.md`.

### Decision Engine
- Добавлены действия `watch`, `throttle`, `drop`, `temp_ban` с порогами (`watch_threshold`, `throttle_threshold`, `drop_threshold`, `temp_ban_threshold`).
- Добавлен multi-signal gating для агрессивных действий (`drop`/`temp_ban`) через `promotion_min_signals`.
- Добавлен anti-flapping механизм: cooldown на деэскалацию через `action_cooldown_seconds`.
- Добавлен глобальный rate limit изменения действий через `max_actions_per_minute`.

### Cardinality Control
- Добавлены лимиты `max_active_ips` и eviction по score/last_seen.
- Добавлен лимит `max_unique_paths_per_ip` при анализе path-энтропии.
- Добавлен `inactive_ttl_seconds` для автоматической очистки неактивных записей state.

### Сборка и Контейнеры
- Dockerfile для `ddos-model` в `default`, `auto-start`, `testpage` обновлены на сборку из `cmd/tarinio-sentinel` + `internal/sentinel` через Go modules.
- Legacy `ddos-model` Dockerfiles now build with `golang:1.26.2-bookworm`, matching the repository `go.mod` requirement.
- `ha-lab` now supplies `CONTROL_PLANE_SECURITY_PEPPER` by default for local lab startup.

### Тестирование
- Прогнаны таргетные пакеты после рефактора:
  - `go test ./internal/sentinel`
  - `go test ./deploy/compose/default/ddos-model`
  - `go test ./deploy/compose/auto-start/ddos-model`
  - `go test ./deploy/compose/testpage/ddos-model`
  - `go test ./cmd/tarinio-sentinel`
- Добавлены unit-тесты decision/cardinality в `internal/sentinel/sentinel_test.go`.
- Добавлены unit-тесты по новому блоку suggestions:
  - `go test ./control-plane/internal/antiddossuggestions`
  - `go test ./control-plane/internal/handlers` (включая `antiddos_suggestions`)
- Добавлены тесты в sentinel для publish interval и suggestions payload.
- Проверена валидность compose-файлов:
  - `docker compose -f deploy/compose/default/docker-compose.yml config --quiet`
  - `docker compose -f deploy/compose/ha-lab/docker-compose.yml config --quiet`
- Проверен documentation build:
  - `npm.cmd run docs:build`
- Повторно проверены UI/doc release tests:
  - `go test ./ui/tests`
- Full repository verification passed:
  - `go test ./...`
- Full Sentinel enterprise smoke passed for both Docker Compose profiles:
  - `default`: 856 synthetic events, 141 adaptive drops, 4 L7 suggestions, 0 normal false positives
  - `ha-lab`: 856 synthetic events, 142 adaptive drops, 4 L7 suggestions, 0 normal false positives

### План работ
- Обновлён `.work/TASKS.md`: первые четыре задачи переведены в `completed`.
- В `.work/TASKS.md` добавлены новые задачи:
  - унификация версии `3.0.0` + расширенная wiki-документация;
  - полный прогон `go test ./...` и исправление багов.
- В `.work/TASKS.md` задачи shadow mode, standalone/HA-ready, контейнеризация, документация/тест-отчет и версия/wiki переведены в `completed`; добавлена отдельная задача на проверку обратной совместимости legacy Anti-DDoS/ddos-model функций перед финальным changelog.
