# TARINIO Sentinel

`tarinio-sentinel` — адаптивный security engine в текущем релизе TARINIO. Он заменяет старый runtime-процесс `ddos-model`, сохраняя совместимость с артефактом `ddos-model/config.json` и контрактом переменных `MODEL_*`.

## Роль в runtime

Сервис остается легким и CPU-only:

- читает runtime access log через backend `file`;
- хранит состояние по IP в `/state/model-state.json`;
- публикует L4-решения в `/out/adaptive.json`;
- публикует L7-кандидаты в `/out/l7-suggestions.json`;
- читает активный профиль ревизии из `/var/lib/waf/active/current.json` и `ddos-model/config.json`.

В request path нет LLM или тяжелого инференса. Runtime использует только опубликованные adaptive-файлы.

## Режимы источника

`MODEL_SOURCE_BACKEND=file` — production MVP-режим. Читает `/logs/access.log`, хранит byte offset и переживает рестарты контейнера за счет тома `/state`.

`MODEL_SOURCE_BACKEND=redis` — задел под HA stream-режим. Интерфейс есть, но stream-ingest не является базовым сценарием: текущие HA-профили работают на file-tail.

## Включение по сайтам

Глобальный переключатель Anti-DDoS остается в `/anti-ddos` как `model_enabled`. Для фронт-сервисов дополнительно используется `front_service.adaptive_model_enabled`.

При компиляции ревизии TARINIO формирует:

- `model_enabled: false`, если глобальная модель выключена или ни один сайт не включен;
- `model_enabled_sites` со списком включенных site ID/hostname, если хотя бы один сервис opt-in.

`control-plane-access` включен по умолчанию. Новые и импортированные сервисы остаются выключенными до явного opt-in.

## Scoring

Sentinel использует легковесную статистику:

- статус-сигналы: `429`, `403`, `444`;
- частота запросов по IP;
- доля `404`;
- обращения к scanner-path (`/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`);
- число уникальных путей по IP;
- риск user-agent;
- emergency burst-сигналы для single-source и distributed flood;
- опциональный ML-сигнал `signal_ml_risk` (CPU-only logistic regression).

В состоянии хранятся:

- `risk_score`: чем выше, тем подозрительнее;
- `trust_score`: `100 - risk_score`;
- `reason_codes`: причины текущего решения;
- `top_signals`: вклад основных сигналов.

Оценки затухают по `MODEL_DECAY_LAMBDA`, поэтому нормальный трафик восстанавливает trust со временем.

## Decision engine

Лестница действий:

- `watch`;
- `throttle`;
- `drop`;
- `temp_ban`.

Агрессивные действия требуют нескольких независимых сильных сигналов, кроме emergency-сценариев. Anti-flapping и `MODEL_MAX_ACTIONS_PER_MINUTE` ограничивают дерганые пересчеты и избыточные публикации.

Для интеграции с постоянными банами используется контур sync через control-plane (`SentinelBanSyncService`), без прямой записи Sentinel в blacklist-файлы.

## Adaptive output

`adaptive.json` остается backward-compatible для `l4guard`. Новые поля (`score`, `trust_score`, `source`, `first_seen`, `last_seen`, `reason_codes`, `top_signals`, `model_version`) опциональны.

Публикация батчевая:

- запись только при реальном diff;
- не чаще `MODEL_PUBLISH_INTERVAL_SECONDS`;
- только top-N через `MODEL_MAX_PUBLISHED_ENTRIES`.

Это ограничивает churn и нагрузку на `l4guard` при высокой cardinality.

## L7 suggestions

Sentinel детектирует hot scanner paths и публикует L7-кандидаты. Control-plane работает через:

- `GET /core-docs/api/anti-ddos/rule-suggestions`;
- `POST|PUT /core-docs/api/anti-ddos/rule-suggestions`;
- `POST|PUT /core-docs/api/anti-ddos/rule-suggestions/{id}/status`.

Жизненный цикл статусов:

- `suggested`: кандидат виден оператору, блокировки нет;
- `shadow`: считается would-block и FP-метрика, блокировки нет;
- `temporary`: временное применение с TTL и барьерами rollback;
- `permanent`: кандидат дошел до порогов и требует финального operator review.

Безусловного auto-apply в permanent-режиме нет.

## Защита от false positive

Базовые меры безопасности:

- opt-in по сайтам для non-management сервисов;
- multi-signal требование для `drop` и `temp_ban`;
- suggestions стартуют как неблокирующие кандидаты;
- shadow-mode считает would-block без enforcement;
- лимиты cardinality ограничивают state и публикацию;
- rollback идет через стандартный revision flow control-plane.

## Ресурсы

Рекомендуемые лимиты контейнера:

- `default`: `0.75 CPU`, `512MB`;
- `enterprise`: `0.75 CPU`, `512MB`;
- `ha-lab`: `0.75 CPU`, `512MB`;
- практический диапазон: `0.25-0.75 CPU`, `128-512MB`.

Основные узкие места: объем логов, cardinality активных IP и разнообразие путей. В production обязательно задавайте `MODEL_MAX_ACTIVE_IPS`, `MODEL_MAX_PUBLISHED_ENTRIES`, `MODEL_MAX_UNIQUE_PATHS_PER_IP`.
