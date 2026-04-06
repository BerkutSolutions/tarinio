# Runbook (RU)

Базовая версия документации: `1.1.1`

## Быстрые проверки

- Liveness: `GET /healthz`
- Версия/билд: `GET /api/app/meta`
- UI-маршруты: `/login`, `/dashboard`

## Ежедневный операторский минимум

1. Проверить здоровье сервисов.
2. Проверить, что последний apply имеет статус `succeeded`.
3. Проверить, что нет аномального роста блокировок и `429`.
4. Проверить, что audit stream обновляется.

## Стандартный change flow

Для любого изменения политик/конфигов:

1. Изменить конфиг в UI/API.
2. Собрать новую revision.
3. Применить revision.
4. Проверить трафик и активность.
5. При деградации выполнить rollback.

## Триаж инцидента

Если трафик ломается:

1. Сначала определить слой проблемы:
   - L4 anti-DDoS,
   - WAF policy,
   - rate-limit,
   - TLS/certificate,
   - upstream.
2. Проверить последние изменения в audit и reports.
3. При широком влиянии сразу откатить на last known good revision.

## Rollback playbook

1. Открыть страницу revisions.
2. Выбрать предыдущую known-good revision.
3. Выполнить apply.
4. Подтвердить `apply succeeded`.
5. Подтвердить восстановление трафика.

## Логи

Основные контейнеры:

- `control-plane`
- `runtime`
- `worker`

## Когда эскалировать

Эскалировать incident owner, если:

- apply падает повторно,
- `/healthz` нестабилен больше 5 минут,
- после изменения политики массовые 4xx/5xx,
- проблемы с выпуском/продлением production сертификата.

## Связанные гайды

- `docs/ru/operators/anti-ddos-runbook.md`
- `docs/ru/operators/waf-tuning-guide.md`
- `docs/ru/security.md`



