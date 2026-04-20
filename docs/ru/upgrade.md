# Обновление и откат TARINIO 2.0.1

Wiki baseline: `2.0.1`

Документ описывает рекомендованный жизненный цикл обновления и критерии отката.

## Поддерживаемый подход

Рекомендуемый путь обновления:

- `latest-1 -> latest`

Для production обновление должно проходить как контролируемое окно изменений, а не как фоновая операция.

## Обязательный lifecycle

1. Preflight
2. Backup
3. Upgrade
4. Smoke validation
5. Rollback decision

## 1. Preflight

Перед обновлением:

- зафиксировать текущую и целевую версии;
- убедиться, что система здорова;
- убедиться, что нет зависших apply jobs;
- проверить свободное место и состояние volumes;
- назначить rollback owner;
- зафиксировать maintenance window.

## 2. Backup

Перед upgrade обязателен полный backup:

- БД;
- runtime state;
- TLS/certificate materials;
- `.env` и секреты.

Подробности: `docs/ru/backups.md`.

## 3. Upgrade

Рекомендуемая последовательность:

1. Обновить deployment artifacts.
2. Применить compose/config изменения.
3. Перезапустить сервисы.
4. Дождаться readiness.
5. Проверить завершение миграций.

## 4. Smoke validation

После обновления минимум нужно проверить:

- `GET /healthz`
- `/login`
- `/healthcheck`
- `GET /api/app/meta` и ожидаемую версию
- открытие ключевых вкладок UI
- compile/apply новой или существующей ревизии
- доступность ingress по HTTP/HTTPS

## 5. Rollback decision

Откат обязателен, если:

- health не стабилизируется;
- login, onboarding или базовый UI не работают;
- compile/apply системно падает;
- runtime обслуживает трафик небезопасно или с заметной деградацией;
- есть сомнение в целостности состояния.

## Путь отката

1. Вернуть предыдущую версию deployment.
2. При необходимости восстановить backup.
3. Проверить `/healthz`.
4. Применить known-good ревизию.
5. Повторить smoke checks.

## Что не делать

- не обновлять production без backup;
- не совмещать upgrade с большой пачкой рискованных policy-изменений;
- не пропускать smoke validation;
- не считать upgrade завершённым только по факту старта контейнеров.

## Связанные документы

- `docs/ru/backups.md`
- `docs/ru/runbook.md`
- `docs/ru/security.md`
