# Развертывание через Docker

Эта страница относится к текущей ветке документации.

Это основной production-гайд по развертыванию TARINIO через Docker/Docker Compose.

## Минимальный стек

- `control-plane`
- `runtime`
- `postgres`
- Docker-сети, тома и секреты окружения

## Вариант 1: AIO bootstrap

### sh

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

### PowerShell

```powershell
iwr https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh -UseBasicParsing | sh
```

## Вариант 2: Docker Compose

Основные документы по compose-профилям:

- `deploy/compose/README.md`
- `deploy/compose/default/README.md`
- `deploy/compose/auto-start/README.md`

### sh

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

### PowerShell

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Post-deploy чеклист

- доступен `/login`;
- стабильно отвечает `/healthz`;
- `GET /core-docs/api/app/meta` возвращает ожидаемую версию;
- первый compile/apply проходит без ошибок;
- runtime обслуживает ожидаемый host-трафик.

## Production рекомендации

- заменить дефолтные секреты до первого внешнего доступа;
- ограничить доступ к control-plane;
- использовать HTTPS для административного контура;
- заранее определить владельцев backup/rollback процесса.
