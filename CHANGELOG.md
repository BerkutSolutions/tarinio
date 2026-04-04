# Журнал изменений

Все значимые изменения проекта фиксируются в этом файле.

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

