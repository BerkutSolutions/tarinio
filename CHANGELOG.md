# Журнал изменений

Все значимые изменения проекта фиксируются в этом файле.

## [1.0.11] - 2026-04-04

### Ядро (Runtime / Compiler / Policies)
- Добавлен runtime CRS manager:
  - одноразовое авто-подтягивание latest CRS при первом старте;
  - хранение active/latest версии и state;
  - ручной update и опциональный hourly auto-update.
- Для Easy-профилей CRS включен по умолчанию (не только в `auto-start`).
- Логика ModSecurity привязана к `front_service.security_mode`:
  - `block` -> `SecRuleEngine On`;
  - `monitor` -> `SecRuleEngine DetectionOnly`;
  - `transparent` -> `SecRuleEngine Off`.
- Устранено дублирование CRS include (ошибка `Rule id: 901001 is duplicated`) при запуске runtime.
- В Easy `ModSecurity` добавлен флаг `use_modsecurity_custom_configuration`:
  - `custom path/content` используются только при включенном чекбоксе;
  - базовый CRS flow остается автоматическим.
- В Easy site profile (UI `services`) исправлено сохранение upstream routing: если поле `Reverse proxy host` пустое, значение теперь автоматически собирается из `Схема + Хост + Порт` upstream.
- Устранен сценарий ложного `400 Bad Request` при включенном reverse proxy и заполненных upstream-полях (`easy site profile upstream_routing.reverse_proxy_host is required when reverse proxy is enabled`).
- Добавлена совместимость с легаси-шаблоном `http://upstream-server:8080`: это значение теперь не считается валидным целевым host и автоматически заменяется на фактический upstream target.
- Подняты дефолтные anti-abuse лимиты для новых Easy profile (меньше ложных autoban): `limit_req_rate` по умолчанию `120r/s`, `limit_conn` `200/400/400`, `bad_behavior_threshold` `120`, окно `120s`, коды bad behavior по умолчанию `400,401,405,444`.
- Глобальный Anti-DDoS L7 override больше не навязывается для management-site `control-plane-access`, чтобы UI/админ-поток не попадали под анти-DDoS профиль как обычный публичный сайт.
- Исправлено определение стран/IP-контекста в runtime security events: в лог-пайплайн и события добавлены поля `country` и `host`.

### API
- Добавлены backend/runtime endpoints для управления CRS:
  - `GET /api/owasp-crs/status`;
  - `POST /api/owasp-crs/check-updates`;
  - `POST /api/owasp-crs/update`.

### UI
- Добавлена отдельная вкладка `OWASP CRS` в боковом меню:
  - статус active/latest;
  - ручной dry-run check;
  - ручной update;
  - чекбокс hourly auto-update.
- Вкладка `OWASP CRS` доработана:
  - исправлен баг с `true/false` и перерисовкой страницы при dry-run/update;
  - добавлена явная кнопка сохранения для чекбокса hourly auto-update;
  - добавлен встроенный журнал операций (запросы/обновления/ошибки);
  - кнопка открытия релиза перенесена в правый верхний угол карточки и сделана компактной;
  - обновлены i18n-ключи (RU/EN) для статусов и уведомлений CRS.
- Исправлен flow действий в `Баны` (`Продлить`/`Разбанить`) для событий, где runtime отдает алиас site id (например `sentry_hantico_ru`): backend теперь резолвит алиас в канонический id сайта.
- В UI страницы `Баны` добавлена каноникализация site id из событий/access-policy перед действиями, чтобы исключить ложные `site ... not found`.
- Исправлен fallback времени автобана в UI `Баны`: вместо жестких `10s` используется `300s`, если per-site профиль не удалось прочитать.
- Добавлена эскалация банов по IP в UI `Баны` (с persisted state): повтор после разбанивания повышает уровень до `24h`, затем до `GLOBAL PERM` (бан по всем сайтам).
- Добавлены правила allow/deny для банов: IP из `allowlist` не эскалируются и не банятся автоматически; IP из `denylist` считаются немедленно забаненными для соответствующего сайта.
- В Easy profile добавлена вкладка `Блокировки` с гибкой эскалацией банов по этапам (`+`/удаление этапов, формат `s/m/h/d`, `0` = перманентно) и выбором scope (`текущий сервис` или `все сервисы`).
- Базовый этап банов для anti-DDoS теперь берется из `security_behavior_and_limits.bad_behavior_ban_time_seconds` конкретного сервиса (вместо жесткого общего значения).

### Compose Profiles
- Синхронизирован `auto-start` профиль для локальной проверки (`CONTROL_PLANE_ACME_USE_DEVELOPMENT_CLIENT=true`).
- Актуализирован `testpage` профиль:
  - runtime-контейнеры получают read-only mount соответствующего `control-plane-data`;
  - для fast-start включен development ACME client;
  - обновлены `.env` и `.env.example` под актуальные переменные management/app стеков.

### Тестирование
- Обновлен smoke-тест XSS: `deploy/compose/default/test-xss.ps1`:
  - поддержка запуска через runtime container (без Windows `schannel` проблем);
  - проверка факта загрузки CRS (`rules loaded ... local > 0`);
  - настраиваемый порог blocked-запросов.
- Добавлен отдельный `go test` на защитные режимы и изоляцию по сервисам:
  - `compiler/internal/compiler/easy_protection_test.go`.

### Документация
- Обновлены API docs под CRS endpoints.
- Разделены операторские документы:
  - `docs/operators/owasp-crs.md`;
  - `docs/operators/letsencrypt-dns.md`.
- Добавлены/обновлены зеркала EN/RU и ссылки в индексах документации.

## [1.0.10] - 2026-04-04

### Исправлено
- Исправлен redirect после onboarding: переход на `/login` теперь выполняется по доменному имени созданного сервиса, а не по текущему IP-хосту в адресной строке.
- Исправлен выпуск сертификатов для новых сервисов в UI: ACME-запрос теперь использует сохраненный `account_email` (из Easy profile/onboarding) вместо fallback на `admin@example.com`.
- В Easy site profile добавлено и сохранено поле `front_service.acme_account_email` для повторного использования при выпуске TLS-сертификатов.

## [1.0.9] - 2026-04-04

### Исправлено
- Исправлен production compose для ACME HTTP-01: `runtime` получает read-only mount `waf-control-plane-data` в `/var/lib/waf/control-plane`, чтобы challenge-файлы `/.well-known/acme-challenge/*` были доступны nginx и проходила валидация Let's Encrypt.
- Аналогичный fix применен для профиля `auto-start`.

## [1.0.8] - 2026-04-04

### Исправлено
- Исправлен flow первичной настройки: если ACME/self-signed job завершается со статусом `failed`, API теперь возвращает ошибку сразу, чтобы onboarding не продолжал TLS bind с отсутствующим сертификатом.
- В onboarding исправлен post-apply redirect и текст подтверждения: переход на `https://<host>/login` (production-путь через `443`, без `:8080`).
- Уточнен финальный вывод `install-aio.sh`: `:8080` помечен как временный порт только для первичной настройки.

## [1.0.7] - 2026-04-04

### Исправлено
- Добавлен полноценный выбор ACME-центра сертификации в onboarding (этап 2): `Let's Encrypt` / `ZeroSSL`.
- Для `ZeroSSL` добавлены обязательные поля EAB (`kid` и `hmac`) с динамическим отображением в UI.
- Добавлен `Let's Encrypt DNS-01` через `Cloudflare` (нативная интеграция через API, без скриптовых обходов).
- На вкладке сертификатов добавлены связанные поля DNS-валидации и валидация обязательных параметров.
- Добавлена защита от спама проверок обновлений: автоматическая проверка выполняется не чаще одного раза в час и только при включенной опции авто-проверки.
- Система обновлений и метаданные репозитория синхронизированы с `https://github.com/BerkutSolutions/tarinio`.

## [1.0.6] - 2026-04-04

### Исправлено
- Включен реальный Let's Encrypt ACME HTTP-01 (вместо development self-signed заглушки):
  - подключен ACME-клиент `lego` и включен по умолчанию;
  - добавлен сервисный клиент `control-plane/internal/services/letsencrypt_client_acme.go`;
  - wiring обновлен в `control-plane/internal/app/app.go`;
  - ACME-конфигурация и env-переменные добавлены в `control-plane/internal/config/config.go`.
- Добавлена раздача challenge `/.well-known/acme-challenge/` в runtime nginx:
  - `compiler/templates/nginx/conf.d/base.conf.tmpl`;
  - `compiler/templates/nginx/sites/site.conf.tmpl`.
- Добавлены ACME env-переменные:
  - `deploy/compose/default/.env`;
  - `deploy/compose/default/.env.example`.
- Обновлены зависимость `lego` в `go.mod/go.sum`.
- Обновлен one-command installer `scripts/install-aio.sh` (admin UI на `:8080`, WAF ingress на `:80/:443`).
- Локальная проверка после изменений:
  - `go test ./control-plane/internal/services ./control-plane/internal/app ./compiler/internal/compiler ./ui/tests` — `OK`.
- Смягчены дефолтные лимиты для management-site (`control-plane-access`, либо ID из `CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID`), чтобы вебморда не ловила ложные autoban при штатной работе SPA:
  - `limit_req_rate` минимум `300r/s`;
  - `limit_req_url` по умолчанию `/api/`;
  - `limit_conn` минимум `300/500/500` для HTTP/1.1, HTTP/2, HTTP/3;
  - `bad_behavior_threshold` минимум `100`, `bad_behavior_count_time_seconds` минимум `60`;
  - `bad_behavior_status_codes` для management-site: `400,401,405,444` (без `403/404/429`).
- Исправлен onboarding/Services-flow для TLS self-signed fallback:
  - добавлен backend endpoint `POST /api/certificates/self-signed/issue` (выпуск self-signed сертификата с сохранением материалов);
  - в UI (`sites.js`) при выборе `tls_self_signed` используется self-signed issue, а не ACME issue;
  - устранен сценарий `certificate <id> not found` при сохранении сервиса с self-signed режимом.

## [1.0.5] - 2026-04-04

### Исправлено
- Исправлен критичный сценарий сохранения сервиса в UI (`sites.js`):
  - добавлен rollback (компенсация) при неуспешном сохранении, чтобы не оставались частично созданные сущности;
- для нового сайта добавлена обработка промежуточной ошибки `default upstream is required`, если сайт уже был создан в рамках шага;
  - при финальной ошибке выполняется best-effort очистка созданных в рамках операции `site/upstream/tls/certificate`.
- Устранен кейс, когда после ошибки apply сервис мог создаться без корректно примененного апстрима.

## [1.0.4] - 2026-04-04

### Исправлено
- Исправлен критичный сценарий сохранения сервиса в UI (`sites.js`):
  - добавлен rollback (компенсация) при неуспешном сохранении, чтобы не оставались частично созданные сущности;
- для нового сайта добавлена обработка промежуточной ошибки `default upstream is required`, если сайт уже был создан в рамках шага;
  - при финальной ошибке выполняется best-effort очистка созданных в рамках операции `site/upstream/tls/certificate`.
- Устранен кейс, когда после ошибки apply сервис мог создаться без корректно примененного апстрима.

## [1.0.3] - 2026-04-04

### Изменено
- Текущая версия продукта переведена на `1.0.3`.
- One-command installer (`scripts/install-aio.sh`) по умолчанию запускает профиль `default` (без dev fast start), чтобы первичная настройка шла в штатном HTTP onboarding-потоке.
- Для ручного быстрого локального старта оставлен профиль `auto-start` (через `PROFILE=auto-start`).

## [1.0.2] - 2026-04-04

### Изменено
- Дефолтный набор разрешенных HTTP-методов для Easy profile расширен до:
  - `GET`, `HEAD`, `OPTIONS`, `POST`, `PUT`, `PATCH`, `DELETE`.
- Обновлены дефолты методов в:
  - `control-plane` (persisted default profile),
  - `compiler` (fallback profile при отсутствии persisted Easy profile),
  - `UI` (draft для нового сервиса).
- Профиль `deploy/compose/default` обновлен для боевого локального запуска:
  - `ui` публикуется наружу на `80:80`,
  - `runtime` публикуется наружу на `443:443`.

## [1.0.1] - 2026-04-04

### Изменено
- Compose-профили разделены на два сценария:
  - `deploy/compose/default` — production baseline.
  - `deploy/compose/auto-start` — localhost auto bootstrap/dev fast start.
- Версия приложения обновлена до `1.0.1` в `control-plane/internal/appmeta/meta.go`.

### Документация
- Удален быстрый старт из корневых README и основного индекса документации.
- Обновлена документация по compose-профилям:
  - `deploy/compose/README.md`
  - `deploy/compose/default/README.md`
  - `deploy/compose/auto-start/README.md`
- Базовые версии в индексах документации и корневых README обновлены до `1.0.1`.

## [1.0.0] - 2026-04-03

### Первый публичный релиз

Первая релизная версия продукта `Berkut Solutions - TARINIO`.

Базовая версия: `1.0.0`.

