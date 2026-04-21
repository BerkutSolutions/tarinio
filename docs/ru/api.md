# API

Эта страница относится к текущей ветке документации.

Этот документ описывает актуальный HTTP API control-plane по состоянию версии `2.0.2`. Каталог составлен по реально зарегистрированным маршрутам из `control-plane/internal/httpserver/server.go`.

## Общие правила

- Все прикладные ответы возвращаются в JSON, кроме download/export сценариев.
- Основная аутентификация сессионная, через cookie.
- Авторизация проверяется на сервере для каждого endpoint.
- Права привязаны к permission-модели RBAC.
- UI использует этот же API; если endpoint описан здесь, значит на него опирается интерфейс или системные процессы.

## Системные маршруты

### `GET /healthz`

Системный health endpoint.

Проверяет:

- revision store;
- revision catalog API;
- компонентное состояние, которое нужно для запуска control-plane.

### `GET /api/setup/status`

Статус первичной инициализации.

Используется:

- login;
- onboarding;
- guard-логикой перед входом в UI.

### `GET /api/app/meta`

Метаданные приложения:

- версия;
- product name;
- repository URL;
- данные по обновлениям, если включена проверка.

### `POST /api/app/ping`

Фоновая проверка активной сессии из UI.

### `GET /api/app/compat`

Отчёт по совместимости модулей приложения и runtime.

### `POST /api/app/compat/fix`

Попытка исправить найденную проблему совместимости.

## Runtime settings

### `GET /api/settings/runtime`

Чтение runtime-настроек для вкладки `Settings -> General`.

Используется для:

- deployment mode;
- update checks;
- метаданных текущей версии.

### `PUT /api/settings/runtime`

Обновление runtime-настроек.

Через UI изменяются:

- `update_checks_enabled`
- сроки хранения `logs`, `activity`, `events`, `bans`

### `POST /api/settings/runtime/check-updates`

Проверка обновлений.

Поддерживает ручной и фоновый режим.

### `GET /api/settings/runtime/storage-indexes?storage_indexes_limit=N&storage_indexes_offset=N`

Чтение индексов хранения для вкладки `Settings -> Storage`.

### `DELETE /api/settings/runtime/storage-indexes?date=YYYY-MM-DD`

Удаление индекса хранения за конкретную дату.

## OWASP CRS

### `GET /api/owasp-crs/status`

Статус установленного CRS-релиза.

### `POST /api/owasp-crs/check-updates`

Dry-run проверки доступности обновления CRS.

### `POST /api/owasp-crs/update`

Запуск обновления CRS.

Используется также для включения hourly auto-update флага.

## Auth и учётная запись

### Bootstrap и login

- `POST /api/auth/bootstrap`
- `POST /api/auth/login`
- `POST /api/auth/login/2fa`
- `POST /api/auth/logout`
- `GET /api/auth/me`

### 2FA

- `GET /api/auth/2fa/status`
- `POST /api/auth/2fa/setup`
- `POST /api/auth/2fa/enable`
- `POST /api/auth/2fa/disable`

### Пароль

- `POST /api/auth/change-password`

### Passkeys

- `POST /api/auth/passkeys/login/begin`
- `POST /api/auth/passkeys/login/finish`
- `POST /api/auth/login/2fa/passkey/begin`
- `POST /api/auth/login/2fa/passkey/finish`
- `GET /api/auth/passkeys`
- `POST /api/auth/passkeys/register/begin`
- `POST /api/auth/passkeys/register/finish`
- `PUT /api/auth/passkeys/{id}/rename`
- `DELETE /api/auth/passkeys/{id}`

## Конфигурационные ресурсы

### Sites

- `GET /api/sites`
- `POST /api/sites`
- `GET /api/sites/{id}`
- `PUT /api/sites/{id}`
- `DELETE /api/sites/{id}`
- `POST /api/sites/{id}/ban`
- `POST /api/sites/{id}/unban`

### Upstreams

- `GET /api/upstreams`
- `POST /api/upstreams`
- `GET /api/upstreams/{id}`
- `PUT /api/upstreams/{id}`
- `DELETE /api/upstreams/{id}`

### Certificates

- `GET /api/certificates`
- `POST /api/certificates`
- `GET /api/certificates/{id}`
- `PUT /api/certificates/{id}`
- `DELETE /api/certificates/{id}`

### TLS configs

- `GET /api/tls-configs`
- `POST /api/tls-configs`
- `GET /api/tls-configs/{siteID}`
- `PUT /api/tls-configs/{siteID}`
- `DELETE /api/tls-configs/{siteID}`

### TLS auto renew

- `GET /api/tls/auto-renew`
- `PUT /api/tls/auto-renew`

### Certificate material operations

- `POST /api/certificate-materials/upload`
- `POST /api/certificate-materials/import-archive`
- `POST /api/certificate-materials/export`
- `GET /api/certificate-materials/export/{certificateID}`

### ACME

- `POST /api/certificates/acme/issue`
- `POST /api/certificates/acme/renew/{certificateID}`
- `POST /api/certificates/self-signed/issue`

## Политики

### WAF policies

- `GET /api/waf-policies`
- `POST /api/waf-policies`
- `GET /api/waf-policies/{id}`
- `PUT /api/waf-policies/{id}`
- `DELETE /api/waf-policies/{id}`

### Access policies

- `GET /api/access-policies`
- `POST /api/access-policies`
- `POST /api/access-policies/upsert`
- `PUT /api/access-policies/upsert`
- `GET /api/access-policies/{id}`
- `PUT /api/access-policies/{id}`
- `DELETE /api/access-policies/{id}`

### Rate-limit policies

- `GET /api/rate-limit-policies`
- `POST /api/rate-limit-policies`
- `GET /api/rate-limit-policies/{id}`
- `PUT /api/rate-limit-policies/{id}`
- `DELETE /api/rate-limit-policies/{id}`

### Easy site profiles

- `GET /api/easy-site-profiles/{siteID}`
- `PUT /api/easy-site-profiles/{siteID}`
- `POST /api/easy-site-profiles/{siteID}`
- `GET /api/easy-site-profiles/catalog/countries`

### Anti-DDoS

- `GET /api/anti-ddos/settings`
- `POST /api/anti-ddos/settings`
- `PUT /api/anti-ddos/settings`

## Observability и отчёты

### Requests и events

- `GET /api/events`
- `GET /api/requests`

### Dashboard

- `GET /api/dashboard/stats`
- `GET /api/dashboard/containers/overview`
- `GET /api/dashboard/containers/logs`
- `GET /api/dashboard/containers/issues`

### Reports

- `GET /api/reports/revisions`

### Audit

- `GET /api/audit`

## Ревизии

### `GET /api/revisions`

Новый агрегированный каталог ревизий в `2.0.2`.

Используется разделом `Ревизии` и отдаёт:

- список сервисов;
- список ревизий;
- summary-счётчики;
- timeline статусов и событий применения.

### `POST /api/revisions/compile`

Компиляция новой ревизии.

### `POST /api/revisions/{revisionID}/apply`

Применение выбранной ревизии.

### `DELETE /api/revisions/{revisionID}`

Удаление неактивной ревизии.

Удаляются связанные snapshot/job данные, но активная ревизия удалению не подлежит.

### `DELETE /api/revisions/statuses`

Очистка status timeline ревизий.

Важно:

- очищает ленту статусов и счётчики;
- не стирает у самих ревизий закреплённый факт последнего успешного или неуспешного применения.

## Администрирование

### `GET /api/administration/users`

Список пользователей для административной таблицы.

### `POST /api/administration/users`

Создание пользователя с явным набором ролей.

### `GET /api/administration/users/{id}`

Чтение одного пользователя для модального просмотра.

### `PUT /api/administration/users/{id}`

Обновление пользователя, включая статус, роли и необязательную смену пароля.

### `GET /api/administration/roles`

Список ролей и каталог известных permission-ов для редактора ролей.

### `POST /api/administration/roles`

Создание роли.

### `GET /api/administration/roles/{id}`

Чтение одной роли.

### `PUT /api/administration/roles/{id}`

Обновление роли. Встроенная роль `admin` нормализуется обратно в полный доступ.

### `GET /api/administration/zero-trust/health`

Zero-trust probe для users, roles, обязательных базовых ролей и полноты permission-набора у `admin`.

### `GET /api/administration/scripts`

Каталог административных сценариев.

### `POST /api/administration/scripts/{scriptID}/run`

Запуск сценария с входными параметрами.

### `GET /api/administration/scripts/runs/{runID}/download`

Скачивание архива результата выполнения.

## Связь с UI

Ниже указано, какие разделы UI опираются на какие группы API.

- `Dashboard`: `dashboard/*`, `events`, `requests`
- `Сайты`: `sites`, `upstreams`, `tls-configs`, `certificates`, `access-policies`, `easy-site-profiles`
- `Anti-DDoS`: `anti-ddos/settings`, `events`
- `OWASP CRS`: `owasp-crs/*`
- `TLS`: `certificates`, `tls-configs`, `tls/auto-renew`, `certificate-materials/*`, `certificates/acme/*`
- `Запросы`: `requests`, `settings/runtime`, `sites`
- `Ревизии`: `revisions`, `revisions/{id}/apply`, `revisions/statuses`
- `События`: `events`, `sites`
- `Баны`: `sites/{id}/ban`, `sites/{id}/unban`, `events`, `access-policies`
- `Администрирование`: `administration/users*`, `administration/roles*`, `administration/zero-trust/health`, `administration/scripts*`
- `Активность`: `audit`
- `Настройки`: `settings/runtime`, `app/meta`
- `Профиль`: `auth/me`, `auth/change-password`, `auth/2fa/*`, `auth/passkeys/*`
