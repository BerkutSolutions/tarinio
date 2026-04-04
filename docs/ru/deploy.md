# Deploy (RU)

Базовая версия документации: `1.0.12`

## Быстрый AIO-старт (одна команда)

Полный стек TARINIO (auto-start профиль) можно поднять одной командой:

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

По умолчанию скрипт:
- клонирует/обновляет репозиторий в `/opt/tarinio`
- выбирает ветку `main`
- поднимает `deploy/compose/default`

После запуска:
- `http://<server-ip>:8080/login` (Admin UI)
- `http://<server-ip>/` (WAF HTTP ingress)
- `https://<server-ip>/` (WAF ingress)

## Ручной запуск через Docker Compose

1. Проверьте `.env.example` и подготовьте `.env`.
2. Запустите профиль:

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Примечания

- Для быстрого локального запуска используйте профиль `auto-start`.
- Для боевого локального сценария используйте `default` (`ui` на `8080`, `runtime` на `80/443`).
- Для production задавайте недефолтные секреты и включайте HTTPS.



