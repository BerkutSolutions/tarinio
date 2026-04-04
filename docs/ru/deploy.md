# Deploy (RU)

Базовая версия документации: `1.0.4`

## Быстрый AIO-старт (одна команда)

Полный стек TARINIO (auto-start профиль) можно поднять одной командой:

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

По умолчанию скрипт:
- клонирует/обновляет репозиторий в `/opt/tarinio`
- выбирает ветку `main`
- поднимает `deploy/compose/auto-start`

После запуска:
- `http://<server-ip>/login`
- `https://<server-ip>/login`

## Ручной запуск через Docker Compose

1. Проверьте `.env.example` и подготовьте `.env`.
2. Запустите профиль:

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Примечания

- Для быстрого локального запуска используйте профиль `auto-start`.
- Для боевого локального сценария используйте `default` (`ui` на `80`, `runtime` на `443`).
- Для production задавайте недефолтные секреты и включайте HTTPS.

