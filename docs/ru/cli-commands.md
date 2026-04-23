# CLI-команды

Эта страница относится к текущей ветке документации.

`waf-cli` — изолированный контейнер интерфейса командной строки для работы с `control-plane`.

Запускайте его через помощник из корня репозитория:

```powershell
.\waf-cli.ps1 <команда>
```

Для Linux используйте прямой запуск через `docker compose`:

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli <команда>
```

Скрипт уже использует:

- `deploy/compose/default/docker-compose.yml`;
- профиль `tools`;
- `run --rm --no-deps`;
- предварительную проверку, что `control-plane` запущен.

## Предварительные требования

Сначала поднимите сервисы платформы:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d
```

## Быстрые примеры

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 sites list
.\waf-cli.ps1 events --limit 50
.\waf-cli.ps1 bans list
.\waf-cli.ps1 unban 172.18.0.1
```

Linux (bash):

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli health
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli sites list
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli events --limit 50
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli bans list
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli unban 172.18.0.1
```

## Режимы вывода

По умолчанию используется удобочитаемый вывод (таблицы и сводки).

Сырой JSON:

```powershell
.\waf-cli.ps1 --json events --limit 20
```

Linux (bash):

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli --json events --limit 20
```

## Основные команды

### Состояние и авторизация

```powershell
.\waf-cli.ps1 health
.\waf-cli.ps1 setup status
.\waf-cli.ps1 me
```

### Сайты и блокировки

```powershell
.\waf-cli.ps1 sites list
.\waf-cli.ps1 sites delete site-a

.\waf-cli.ps1 ban 172.18.0.1 --site control-plane-access
.\waf-cli.ps1 unban 172.18.0.1 --site control-plane-access
.\waf-cli.ps1 bans list
.\waf-cli.ps1 bans list --site control-plane-access
```

### События и аудит

```powershell
.\waf-cli.ps1 events --limit 100
.\waf-cli.ps1 events --type security_waf --severity warning --site-id control-plane-access

.\waf-cli.ps1 audit --limit 50
.\waf-cli.ps1 audit --action site.create --site-id control-plane-access --status success --limit 100 --offset 0
```

### Ревизии и отчёты

```powershell
.\waf-cli.ps1 revisions compile
.\waf-cli.ps1 revisions apply rev-000001

# совместимые алиасы
.\waf-cli.ps1 compile
.\waf-cli.ps1 apply rev-000001

.\waf-cli.ps1 reports revisions
```

### Базовые модули бэкенда (list/get)

```powershell
.\waf-cli.ps1 upstreams list
.\waf-cli.ps1 tls list
.\waf-cli.ps1 certificates list

.\waf-cli.ps1 access-policies list
.\waf-cli.ps1 waf-policies list
.\waf-cli.ps1 rate-limit-policies list
```

### Упрощённый профиль и Anti-DDoS из файлов

```powershell
.\waf-cli.ps1 easy get control-plane-access
.\waf-cli.ps1 easy upsert control-plane-access --file .\profile.json

.\waf-cli.ps1 antiddos get
.\waf-cli.ps1 antiddos upsert --file .\antiddos.json
```

### Универсальный режим API (полное покрытие бэкенда)

Используйте для маршрутов, у которых ещё нет выделенной CLI-команды.

```powershell
.\waf-cli.ps1 api GET /api/sites
.\waf-cli.ps1 api GET /api/reports/revisions
.\waf-cli.ps1 api PUT /api/anti-ddos/settings --file .\antiddos.json
.\waf-cli.ps1 api DELETE /api/sites/site-a
```

## Флаги авторизации и подключения

Параметры по умолчанию:

- базовый URL: `http://127.0.0.1:8080`;
- имя пользователя/пароль: `admin/admin`.

Переопределение:

```powershell
.\waf-cli.ps1 --base-url http://control-plane:8080 --username admin --password admin sites list
```

Режим без авторизации (только для публичных конечных точек):

```powershell
.\waf-cli.ps1 --no-auth health
```

Пропуск TLS-проверки сертификата:

```powershell
.\waf-cli.ps1 --insecure --base-url https://control-plane:8443 health
```

## Прямой запуск compose (без вспомогательного скрипта)

```powershell
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli sites list
```

```bash
docker compose -f deploy/compose/default/docker-compose.yml --profile tools run --rm --no-deps cli sites list
```

## Тестовый профиль (полный стек + тестовая страница)

Профиль `testpage` поднимает:

- WAF/UI на `https://localhost:8080`;
- защищённую тестовую страницу на `https://localhost:8081`.

```powershell
cd deploy/compose/testpage
docker compose up -d --build
```

```bash
cd deploy/compose/testpage
docker compose up -d --build
```

Во время сборки образа `ui` автоматически выполняется `go test ./ui/tests`, поэтому ошибки UI/i18n останавливают запуск профиля.

Откройте:

- `https://localhost:8080`
- `https://localhost:8081`

Проверки:

```powershell
cd deploy/compose/testpage
docker compose logs -f control-plane runtime control-plane-test runtime-test ui
docker compose ps
```

```bash
cd deploy/compose/testpage
docker compose logs -f control-plane runtime control-plane-test runtime-test ui
docker compose ps
```

Проверка адаптивной модели:

```powershell
cd deploy/compose/testpage
docker compose logs -f ddos-model-mgmt ddos-model-app runtime runtime-test
docker compose exec ddos-model-mgmt sh -lc "cat /out/adaptive.json"
docker compose exec ddos-model-app sh -lc "cat /out/adaptive.json"
```

```bash
cd deploy/compose/testpage
docker compose logs -f ddos-model-mgmt ddos-model-app runtime runtime-test
docker compose exec ddos-model-mgmt sh -lc "cat /out/adaptive.json"
docker compose exec ddos-model-app sh -lc "cat /out/adaptive.json"
```

Проверка API запросов:

```powershell
cd deploy/compose/testpage
docker compose logs --since=10m runtime
docker compose exec control-plane wget -qO- http://127.0.0.1:8080/api/requests
```

```bash
cd deploy/compose/testpage
docker compose logs --since=10m runtime
docker compose exec control-plane wget -qO- http://127.0.0.1:8080/api/requests
```

## Полезные переменные адаптивной модели

- `WAF_L4_GUARD_REAPPLY_INTERVAL_SECONDS` (по умолчанию `5`)
- `DDOS_MODEL_POLL_INTERVAL_SECONDS` (по умолчанию `2`)
- `DDOS_MODEL_DECAY_LAMBDA` (по умолчанию `0.08`)
- `DDOS_MODEL_THROTTLE_THRESHOLD` (по умолчанию `2.5`)
- `DDOS_MODEL_DROP_THRESHOLD` (по умолчанию `6.0`)
- `DDOS_MODEL_HOLD_SECONDS` (по умолчанию `60`)
- `DDOS_MODEL_THROTTLE_RATE_PER_SECOND` (по умолчанию `3`)
- `DDOS_MODEL_THROTTLE_BURST` (по умолчанию `6`)
- `DDOS_MODEL_THROTTLE_TARGET` (по умолчанию `REJECT`)
- `DDOS_MODEL_WEIGHT_429` (по умолчанию `1.0`)
- `DDOS_MODEL_WEIGHT_403` (по умолчанию `1.8`)
- `DDOS_MODEL_WEIGHT_444` (по умолчанию `2.2`)
- `DDOS_MODEL_EMERGENCY_RPS` (по умолчанию `180`)
- `DDOS_MODEL_EMERGENCY_UNIQUE_IPS` (по умолчанию `40`)
- `DDOS_MODEL_EMERGENCY_PER_IP_RPS` (по умолчанию `60`)
- `DDOS_MODEL_WEIGHT_EMERGENCY_BOTNET` (по умолчанию `6.0`)
- `DDOS_MODEL_WEIGHT_EMERGENCY_SINGLE` (по умолчанию `4.0`)

Остановка и очистка:

```powershell
cd deploy/compose/testpage
docker compose down --remove-orphans
```

```bash
cd deploy/compose/testpage
docker compose down --remove-orphans
```

## Связанные документы

- `deploy/compose/testpage/README.md`
- `deploy/compose/testpage/DDOS_ADAPTIVE_MODEL.md`
