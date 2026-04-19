# Развёртывание TARINIO 2.0.0

Wiki baseline: `2.0.0`

Документ описывает практический deploy TARINIO для локальных, тестовых и production-сценариев.

## Что разворачивается

Минимальный стек включает:

- `control-plane`
- `runtime`
- `postgres`
- связанные volumes, сети и конфигурацию compose-профиля

В зависимости от профиля могут присутствовать дополнительные сервисы и вспомогательные компоненты.

## Варианты запуска

### AIO one-command install

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

Сценарий удобен для:

- быстрых PoC;
- стендов;
- первичного знакомства с продуктом.

### Docker Compose

Основная документация по профилям лежит в:

- `deploy/compose/README.md`
- `deploy/compose/default/README.md`
- `deploy/compose/auto-start/README.md`

Базовый запуск:

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## После запуска

Обычно проверяются:

- административный интерфейс: `/login`
- health endpoint: `/healthz`
- runtime ingress: HTTP/HTTPS порты вашего профиля

После первого старта система может перенаправить на onboarding, если bootstrap ещё не выполнен.

## Production checklist

- использовать недефолтные секреты;
- ограничить сетевой доступ к control-plane;
- включить HTTPS;
- настроить trusted proxies корректно;
- обеспечить резервное копирование volume и секретов;
- проверить, что retention и update checks соответствуют политике эксплуатации;
- заранее определить rollback owner и порядок отката.

## Что подготовить до production

- DNS и сетевые правила;
- политику хранения логов и событий;
- TLS-стратегию: import, self-signed, ACME, DNS-01;
- схему доступа операторов;
- сценарий backup/restore;
- сценарий compile/apply/rollback.

## Первичная настройка после deploy

1. Открыть `/login`.
2. Если система не инициализирована, пройти onboarding.
3. Создать первого администратора.
4. Создать первый сайт и upstream.
5. Настроить TLS.
6. Убедиться, что healthcheck проходит.
7. Выполнить первый compile/apply.

## Что считать успешным развёртыванием

Установка считается завершённой, если:

- `/healthz` стабилен;
- login и onboarding работают;
- `GET /api/app/meta` возвращает `2.0.0`;
- можно создать сайт и получить рабочую ревизию;
- после apply runtime обслуживает ожидаемый хост;
- события, запросы и аудит появляются в UI.

## Связанные документы

- `docs/ru/security.md`
- `docs/ru/runbook.md`
- `docs/ru/upgrade.md`
- `docs/ru/backups.md`
- `docs/ru/ui.md`
