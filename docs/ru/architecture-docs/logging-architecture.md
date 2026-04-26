# Архитектура логирования

Эта страница относится к текущей ветке документации.

Релизный baseline для этой ревизии документа: `3.0.5`.

Документ описывает фактическую модель обработки и хранения request-логов в TARINIO.

## Кратко

Контур запросов разделен на три роли:

- `OpenSearch` — основной backend для записи и чтения запросов;
- `ClickHouse` — опциональный cold-tier для долгой аналитической истории;
- локальный `JSONL` архив — совместимый spool/fallback слой, а не primary-source для UI.

`PostgreSQL` продолжает хранить состояние control-plane (пользователи, роли, ревизии, настройки, метаданные) и не является request-log базой.

## Поток запросных данных

Для каждого запроса runtime:

1. читает access log;
2. нормализует запись в единый формат;
3. пишет данные в активный backend;
4. держит локальную копию как кратковременный fallback;
5. отдает `GET /core-docs/api/requests` из backend-контура в приоритете над локальными файлами.

Практически это означает:

- standalone использует `OpenSearch` как основной источник для `Requests`;
- enterprise может использовать `OpenSearch` (hot) + `ClickHouse` (cold);
- UI работает через API и не читает архив запросов напрямую через `nginx`.

## Роль локального архива

Локальный архив остается только для:

- безопасных апгрейдов со старых standalone-инсталляций;
- временного буфера при недоступности backend;
- контролируемой миграции legacy `*.jsonl` в `OpenSearch`/`ClickHouse`;
- диагностики и восстановления.

Локальный архив не должен быть основной request-базой для UI.

## Поведение при апгрейде

При переходе со старых версий:

1. runtime сканирует legacy day-файлы;
2. импортирует их в активный backend;
3. валидирует наличие данных в backend;
4. удаляет только подтвержденные legacy-файлы;
5. фиксирует migration-state.

Так обеспечивается безопасный переход `legacy release -> current release` без потери истории.

## Default и Enterprise

### Default

Штатный `deploy/compose/default` профиль:

- `PostgreSQL`
- `OpenSearch`
- `Vault`
- `runtime`
- `ui`
- `tarinio-sentinel`

В этом режиме `OpenSearch` — основной request backend, локальный архив — fallback.

### Enterprise

`deploy/compose/enterprise` добавляет:

- multi-node control-plane;
- `Redis` для HA-координации;
- `ClickHouse` для cold analytics.

Здесь `OpenSearch` остается hot-path, `ClickHouse` — cold-tier.

## Что считать нормальным состоянием

Нормальное поведение в `current release`:

1. свежие запросы видны через API/UI без прямого file-read;
2. `OpenSearch` принимает новый трафик;
3. legacy-архив удаляется только после валидации миграции;
4. при включенном `ClickHouse` hot-to-cold перенос сохраняет подтвержденные данные;
5. локальный архив используется как fallback, а не как основная база.

Для lab-валидации используйте:

- `deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1`
- `deploy/lab-k8s-terraform/k8s/scripts/apply-profile-opensearch-clickhouse.ps1`
