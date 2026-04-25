# CLI-команды

Эта страница относится к текущей ветке документации.

`waf-cli` — контейнерный CLI для работы с `control-plane` API.

## Запуск

Windows (из корня репозитория):

```powershell
.\waf-cli.ps1 <команда>
```

Linux/macOS (через `docker compose`):

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli <команда>
```

## Глобальные флаги

- `--base-url` (по умолчанию `http://127.0.0.1:8080`)
- `--username` (по умолчанию `admin`)
- `--password` (по умолчанию `admin`)
- `--insecure` (пропуск проверки TLS-сертификата)
- `--no-auth` (не выполнять логин перед командой)
- `--json` (сырые JSON-ответы)

Пример:

```powershell
.\waf-cli.ps1 --base-url https://control-plane:8443 --username admin --password admin --insecure sites list
```

## Переменные окружения

CLI учитывает:

- `WAF_CLI_BASE_URL`
- `WAF_CLI_USERNAME`
- `WAF_CLI_PASSWORD`
- `CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME`
- `CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD`
- `WAF_CLI_DEFAULT_SITE` (для `ban` / `unban`)
- `CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID` (fallback для default site)

## Полный каталог команд

### Состояние и авторизация

- `health`
- `setup status`
- `me`

### Сайты

- `sites list`
- `sites delete <site-id>`

### Справочники политик и сертификатов

- `upstreams list`
- `tls list`
- `certificates list`
- `access-policies list`
- `waf-policies list`
- `rate-limit-policies list`

### События и аудит

- `events [--limit N] [--type TYPE] [--site-id SITE] [--severity LEVEL]`
- `audit [--action ACTION] [--site-id SITE] [--status STATUS] [--limit N] [--offset N]`

### Блокировки IP

- `ban <ip> [--site <site-id>]`
- `unban <ip> [--site <site-id>]`
- `bans list [--site <site-id>]`

### Easy profile и Anti-DDoS

- `easy get <site-id>`
- `easy upsert <site-id> --file <profile.json>`
- `antiddos get`
- `antiddos upsert --file <antiddos.json>`

### Ревизии и отчёты

- `revisions compile`
- `revisions apply <revision-id>`
- `reports revisions`

Совместимые алиасы:

- `compile` = `revisions compile`
- `apply <revision-id>` = `revisions apply <revision-id>`

### Универсальный API-режим

- `api <GET|POST|PUT|DELETE> <path> [--file <body.json>]`

`<path>` должен быть API-путём (`/api/...`). Если путь указан без `/`, CLI добавит `/` автоматически.

Примеры:

```powershell
.\waf-cli.ps1 api GET /api/sites
.\waf-cli.ps1 api GET /api/reports/revisions
.\waf-cli.ps1 api PUT /api/anti-ddos/settings --file .\antiddos.json
.\waf-cli.ps1 api DELETE /api/sites/site-a
```

## Быстрые примеры

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 sites list
.\waf-cli.ps1 events --limit 50
.\waf-cli.ps1 bans list --site control-plane-access
.\waf-cli.ps1 revisions compile
.\waf-cli.ps1 --json reports revisions
```

## Связанные документы

- `docs/ru/core-docs/api.md`
- `docs/ru/core-docs/ui.md`
- `docs/ru/core-docs/security-profiles.md`
