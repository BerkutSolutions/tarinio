# Quick Start (RU)

Минимальный локальный запуск Berkut Solutions - TARINIO.

## 1. Подготовка

```powershell
Copy-Item .env.example .env
```

Проверьте `.env`:
- задайте недефолтные секреты для non-dev окружений
- установите корректную таймзону (`TZ`)

## 2. Запуск

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## 3. Проверка

- UI: `https://localhost/` (reverse-proxy профиль) или URL вашего control-plane.
- API liveness: `GET /healthz`
- Версия продукта: `GET /api/app/meta`

## 4. Базовые действия после входа

1. Выполните первичную настройку администратора.
2. Создайте upstream и сайт.
3. Скомпилируйте и примените ревизию.

## 5. Полезные ссылки

- Полный деплой: [`docs/ru/deploy.md`](docs/ru/deploy.md)
- Runbook: [`docs/ru/runbook.md`](docs/ru/runbook.md)
- Обновление/откат: [`docs/ru/upgrade.md`](docs/ru/upgrade.md)
- CLI: [`docs/CLI_COMMANDS.md`](docs/CLI_COMMANDS.md)
