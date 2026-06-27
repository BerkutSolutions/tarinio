# Развертывание через Docker

Это основной production-гайд по развертыванию TARINIO v1.3.5+ через Docker/Docker Compose.

## Полный стек сервисов

| Сервис | Роль |
|---|---|
| `control-plane` | API, компилятор конфигов, UI-бэкенд |
| `runtime` | nginx/OpenResty, L4-guard, обслуживание трафика |
| `tarinio-sentinel` | DDoS-модель, credential stuffing detection, JA3 |
| `postgres` | Основная БД (сайты, ревизии, virtual patches) |
| `opensearch` | Индекс запросов и событий |
| `vault` | Хранилище секретов (mTLS сертификаты, токены) |
| `ui` | Статический фронтенд |

Vault обязателен начиная с v1.3.x — в нём хранятся mTLS-сертификаты (ClientCA, клиентские ключи upstream).

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

## Важные тома

| Том | Назначение |
|---|---|
| `waf-runtime-data` | Скомпилированные конфиги nginx |
| `waf-control-plane-data` | Данные control-plane |
| `waf-certificates-data` | mTLS сертификаты (incoming + upstream) |
| `waf-l4-adaptive` | Adaptive L4 ban-list от sentinel |
| `waf-sentinel-state` | Состояние DDoS-модели |
| `waf-vault-data` | Данные Vault |

## Post-deploy чеклист

- доступен `/login`;
- стабильно отвечает `/healthz`;
- `GET /core-docs/api/app/meta` возвращает ожидаемую версию;
- первый compile/apply проходит без ошибок;
- runtime обслуживает ожидаемый host-трафик;
- `tarinio-sentinel` healthcheck зелёный (`/out/adaptive.json` не пустой);
- Vault инициализирован и unsealed (`vault status` возвращает `Sealed: false`).

## Production рекомендации

- заменить дефолтные секреты (`CONTROL_PLANE_SECURITY_PEPPER`, `WAF_RUNTIME_API_TOKEN`, `POSTGRES_PASSWORD`) до первого внешнего доступа;
- ограничить доступ к control-plane (порт 8080) сетевыми правилами;
- использовать HTTPS для административного контура;
- для mTLS — загружать сертификаты через Vault, не монтировать файлами;
- заранее определить владельцев backup/rollback процесса.
