# Обновление / откат (RU)

Базовая версия документации: `1.1.1`

Минимально поддерживаемый путь обновления: `latest-1 -> latest`.

## Обязательный lifecycle

1. Preflight
2. Backup
3. Upgrade/migrate
4. Smoke validation
5. Rollback decision

## 1. Preflight

Перед production-обновлением:

- Зафиксировать maintenance window и rollback owner.
- Зафиксировать текущую и целевую версии.
- Проверить обязательные переменные окружения целевой версии.
- Убедиться, что нет активных apply jobs и система здорова.

## 2. Backup (обязательно)

Перед апгрейдом сделать полный бэкап:

- PostgreSQL volume
- runtime и certificate volumes (если используются)
- `.env` и секретные файлы

См.: `docs/ru/backups.md`.

## 3. Upgrade/migrate

Рекомендуемая последовательность:

1. Обновить deployment artifacts.
2. Применить compose/config изменения.
3. Перезапустить сервисы.
4. Дождаться readiness control-plane и runtime.
5. Проверить, что миграции завершились успешно.

## 4. Smoke validation

Минимум после апгрейда:

- `GET /healthz` возвращает healthy.
- Логин в UI работает.
- `GET /api/app/meta` показывает ожидаемую версию.
- compile + apply проходят успешно.
- HTTPS на целевом хосте работает.

## 5. Rollback decision

Откат обязателен, если:

- health не стабилизируется,
- control-plane функции деградировали,
- compile/apply системно падает,
- поведение трафика небезопасно.

Путь отката:

1. Вернуть предыдущую версию deployment.
2. При необходимости восстановить состояние из бэкапа.
3. Применить последнюю known-good revision.
4. Повторить smoke checks.

## Примечания

- Нельзя пропускать backup ради скорости.
- Не смешивайте много рискованных изменений в одном окне.
- Ведите короткий upgrade log с таймстемпами.



