# Архитектура и принципы Anti-DDoS модели

Документ описывает, как работает контур Anti-DDoS в TARINIO: какие слои защиты используются, какие сигналы учитываются и как принимаются решения.

## 1. Общая схема

Anti-DDoS в TARINIO — это не один переключатель, а каскад из трех слоев:

1. `L4 guard`
2. глобальный L7 rate-limit override
3. адаптивная модель (`tarinio-sentinel`)

Слои не заменяют друг друга:

- `L4 guard` защищает сокеты и CPU до HTTP-парсинга;
- L7 override удерживает базовое HTTP-давление;
- адаптивная модель реагирует на поведение источников и усиливает защиту динамически.

## 2. Source of truth и путь до runtime

Источник настроек:

- storage control-plane: `anti_ddos_settings.json`

Путь применения:

1. оператор меняет настройки в control-plane;
2. compile формирует артефакты ревизии;
3. validate проверяет bundle и runtime-контракт;
4. apply публикует активную ревизию;
5. runtime читает только артефакты активной ревизии.

Минимальный набор артефактов:

- `l4guard/config.json`;
- `ddos-model/config.json` (legacy-compatible профиль Sentinel).

## 3. Базовый L4 guard

`L4 guard` применяет детерминированные правила `iptables`:

- ограничение одновременных TCP-соединений (`connlimit`);
- ограничение скорости новых соединений (`hashlimit`);
- `DROP`/`REJECT` при нарушении порогов.

Это закрывает high-concurrency flood, high-rate connect flood и keep-alive abuse.

## 4. Глобальный L7 override

Если включен `enforce_l7_rate_limit`, компилятор добавляет единое HTTP-ограничение для всех включенных сайтов (кроме management UI).

Ключевые параметры:

- `l7_requests_per_second`
- `l7_burst`
- `l7_status_code`

Это статический baseline в рамках активной ревизии.

## 5. Адаптивная модель: источники и выход

Адаптивный слой выполняется процессом `tarinio-sentinel` (с совместимостью по `ddos-model` артефактам и `MODEL_*` переменным).

Sentinel читает:

- runtime access log;
- состояние `model-state.json`;
- активный профиль `ddos-model/config.json`.

Sentinel пишет:

- `adaptive.json` (L4-действия и explainability);
- `l7-suggestions.json` (кандидаты L7-правил).

`L4 guard` применяет `adaptive.json` поверх baseline-политики.

## 6. Сигналы модели

Базовые сигналы:

- статусные: `429`, `403`, `444`;
- `rps_per_ip`;
- `404_ratio`;
- `unique_paths_per_ip`;
- scanner paths (`/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`);
- `ua_risk`;
- `cross_site_spread`.

Emergency-сигналы:

- общий `RPS` выше аварийного порога;
- рост unique IP за короткое окно;
- очень высокий per-IP `RPS`.

Опционально доступен CPU-only ML-сигнал `signal_ml_risk` (logistic regression вне request path).

## 7. Scoring и decay

Для каждого IP ведутся:

- `risk_score`;
- `trust_score = 100 - risk_score`;
- `reason_codes`;
- `top_signals`;
- временные поля (`last_seen`, `expires_at` и т.п.).

Расчет:

- новый сигнал добавляет вес к риску;
- перед добавлением применяется экспоненциальное затухание по `model_decay_lambda`;
- при спокойном поведении trust восстанавливается.

## 8. Лестница действий

Действия:

- `watch`;
- `throttle`;
- `drop`;
- `temp_ban`.

Переходы требуют выполнения порогов и anti-flapping барьеров (`MODEL_PROMOTION_CONSECUTIVE_TICKS`, cooldown, multi-signal gating), кроме аварийных случаев.

## 9. Anti-flapping и false-positive safety

Защитные механизмы:

- эскалация только после подтверждения в нескольких тиках;
- ограничение числа изменений через `MODEL_MAX_ACTIONS_PER_MINUTE`;
- cardinality guardrails (`MODEL_MAX_ACTIVE_IPS`, `MODEL_MAX_UNIQUE_PATHS_PER_IP`, `MODEL_MAX_PUBLISHED_ENTRIES`);
- TTL и eviction для устаревших записей;
- opt-in для non-management сервисов.

## 10. Что публикуется в `adaptive.json`

Поля верхнего уровня:

- `updated_at`;
- `throttle_rate_per_second`;
- `throttle_burst`;
- `throttle_target`;
- `entries[]`.

Для каждой записи:

- `ip`, `action`, `expires_at`;
- опционально `score`, `trust_score`, `reason_codes`, `top_signals`, `source`, `first_seen`, `last_seen`, `model_version`.

Поддерживаемые действия:

- `throttle`;
- `drop`;
- `temp_ban` (как адаптивное решение с последующей синхронизацией в ban-контур при включенном sync).

## 11. L7 suggestions pipeline

Lifecycle кандидатов:

- `suggested` -> `shadow` -> `temporary` -> `permanent`

Смысл этапов:

- `suggested`: кандидат виден оператору;
- `shadow`: считается `would_block` и FP-rate;
- `temporary`: временное применение с TTL;
- `permanent`: кандидат требует финального review (без безусловного auto-apply).

## 12. Интеграция с перманентными банами

Sentinel не пишет напрямую в blacklist-файлы.

Промоут решений в постоянные баны выполняется через control-plane сервисы (включая `SentinelBanSyncService` и `ManualBanService`) с сохранением аудита, idempotency и текущих приоритетов ban-контура.

## 13. Что модель не делает

Модель не:

- заменяет настройку WAF-политик;
- не выполняет content-aware анализ payload как полноценный IDS;
- не переписывает активную ревизию напрямую;
- не принимает окончательные operator-решения за человека.

Модель делает:

- сбор и анализ runtime-сигналов;
- пересчет оценки риска по IP;
- публикацию адаптивных решений для runtime и control-plane pipeline.

## 14. Связанные документы

- `docs/ru/model-docs/tarinio-sentinel.md`
- `docs/ru/model-docs/anti-ddos-runbook.md`
- `docs/ru/model-docs/runtime-l4-guard.md`
- `docs/ru/operators/waf-tuning-guide.md`
