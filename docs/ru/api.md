# API

Эта страница относится к текущей ветке документации.

Документ описывает актуальный HTTP API control-plane для версии `2.0.3`. Каталог собран по реально зарегистрированным маршрутам из `control-plane/internal/httpserver/server.go`.

## Общие правила

- Основной формат ответов — JSON.
- Исключение составляют сценарии загрузки и выгрузки файлов.
- Основной способ аутентификации — сессионные cookie.
- Проверка прав выполняется на сервере для каждого маршрута.
- Интерфейс использует этот же API, поэтому перечисленные ниже маршруты соответствуют реальной работе продукта.

## Системные маршруты

### `GET /healthz`

Системная проверка состояния. Используется для контроля готовности control-plane и проверяет:

- хранилище ревизий;
- каталог ревизий;
- ключевые внутренние компоненты, необходимые для старта.

### `GET /api/setup/status`

Статус первичной инициализации. Используется:

- при входе;
- в первичной настройке;
- в логике предвходных проверок интерфейса.

### `GET /api/app/meta`

Метаданные приложения:

- версия;
- название продукта;
- ссылка на репозиторий;
- сведения о проверке обновлений;
- признаки HA-режима и идентификатор текущего узла.

### `POST /api/app/ping`

Фоновая проверка активной пользовательской сессии.

### `GET /api/app/compat`

Отчёт о совместимости компонентов приложения и runtime.

### `POST /api/app/compat/fix`

Попытка автоматически исправить обнаруженную проблему совместимости.

## Системные настройки runtime

### `GET /api/settings/runtime`

Чтение настроек runtime для раздела `Настройки -> Общие`.

Используется для отображения:

- режима развёртывания;
- состояния проверки обновлений;
- метаданных текущей версии.

### `PUT /api/settings/runtime`

Обновление настроек runtime. Через интерфейс изменяются:

- `update_checks_enabled`;
- сроки хранения `logs`, `activity`, `events`, `bans`.

### `POST /api/settings/runtime/check-updates`

Ручная или фоновая проверка доступности обновлений.

### `GET /api/settings/runtime/storage-indexes?storage_indexes_limit=N&storage_indexes_offset=N`

Просмотр индексов хранения для раздела `Настройки -> Хранилище`.

### `DELETE /api/settings/runtime/storage-indexes?date=YYYY-MM-DD`

Удаление индекса хранения за указанную дату.

## OWASP CRS

### `GET /api/owasp-crs/status`

Статус установленного релиза CRS.

### `POST /api/owasp-crs/check-updates`

Проверка доступности обновления без фактической установки.

### `POST /api/owasp-crs/update`

Запуск обновления CRS. Используется также для включения почасовой автоматической проверки обновлений.

## Аутентификация и учётная запись

### Первичный вход и сессия

- `POST /api/auth/bootstrap`
- `POST /api/auth/login`
- `POST /api/auth/login/2fa`
- `POST /api/auth/logout`
- `GET /api/auth/me`

### Второй фактор

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

### Сайты

- `GET /api/sites`
- `POST /api/sites`
- `GET /api/sites/{id}`
- `PUT /api/sites/{id}`
- `DELETE /api/sites/{id}`
- `POST /api/sites/{id}/ban`
- `POST /api/sites/{id}/unban`

### Upstream-сервисы

- `GET /api/upstreams`
- `POST /api/upstreams`
- `GET /api/upstreams/{id}`
- `PUT /api/upstreams/{id}`
- `DELETE /api/upstreams/{id}`

### Сертификаты

- `GET /api/certificates`
- `POST /api/certificates`
- `GET /api/certificates/{id}`
- `PUT /api/certificates/{id}`
- `DELETE /api/certificates/{id}`

### TLS-конфигурации

- `GET /api/tls-configs`
- `POST /api/tls-configs`
- `GET /api/tls-configs/{siteID}`
- `PUT /api/tls-configs/{siteID}`
- `DELETE /api/tls-configs/{siteID}`

### Автопродление TLS

- `GET /api/tls/auto-renew`
- `PUT /api/tls/auto-renew`

### Операции с сертификатными материалами

- `POST /api/certificate-materials/upload`
- `POST /api/certificate-materials/import-archive`
- `POST /api/certificate-materials/export`
- `GET /api/certificate-materials/export/{certificateID}`

### ACME

- `POST /api/certificates/acme/issue`
- `POST /api/certificates/acme/renew/{certificateID}`
- `POST /api/certificates/self-signed/issue`

## Политики

### WAF-политики

- `GET /api/waf-policies`
- `POST /api/waf-policies`
- `GET /api/waf-policies/{id}`
- `PUT /api/waf-policies/{id}`
- `DELETE /api/waf-policies/{id}`

### Политики доступа

- `GET /api/access-policies`
- `POST /api/access-policies`
- `POST /api/access-policies/upsert`
- `PUT /api/access-policies/upsert`
- `GET /api/access-policies/{id}`
- `PUT /api/access-policies/{id}`
- `DELETE /api/access-policies/{id}`

### Политики ограничения скорости

- `GET /api/rate-limit-policies`
- `POST /api/rate-limit-policies`
- `GET /api/rate-limit-policies/{id}`
- `PUT /api/rate-limit-policies/{id}`
- `DELETE /api/rate-limit-policies/{id}`

### Упрощённые профили сайтов

- `GET /api/easy-site-profiles/{siteID}`
- `PUT /api/easy-site-profiles/{siteID}`
- `POST /api/easy-site-profiles/{siteID}`
- `GET /api/easy-site-profiles/catalog/countries`

### Anti-DDoS

- `GET /api/anti-ddos/settings`
- `POST /api/anti-ddos/settings`
- `PUT /api/anti-ddos/settings`

## Наблюдаемость и отчёты

### Запросы и события

- `GET /api/events`
- `GET /api/requests`

### Dashboard

- `GET /api/dashboard/stats`
- `GET /api/dashboard/containers/overview`
- `GET /api/dashboard/containers/logs`
- `GET /api/dashboard/containers/issues`

### Отчёты

- `GET /api/reports/revisions`

### Аудит

- `GET /api/audit`

## Ревизии

### `GET /api/revisions`

Агрегированный каталог ревизий, который используется разделом `Ревизии`. Маршрут возвращает:

- список сервисов;
- список ревизий;
- сводные счётчики;
- ленту статусов и событий применения.

### `POST /api/revisions/compile`

Компиляция новой ревизии.

### `POST /api/revisions/{revisionID}/apply`

Применение выбранной ревизии.

### `DELETE /api/revisions/{revisionID}`

Удаление неактивной ревизии. Связанные снимки и служебные данные удаляются вместе с ней, но активную ревизию удалить нельзя.

### `DELETE /api/revisions/statuses`

Очистка ленты статусов ревизий. При этом за самими ревизиями сохраняется информация о последнем успешном или неуспешном применении.

## Администрирование

### Пользователи

- `GET /api/administration/users`
- `POST /api/administration/users`
- `GET /api/administration/users/{id}`
- `PUT /api/administration/users/{id}`

### Роли

- `GET /api/administration/roles`
- `POST /api/administration/roles`
- `GET /api/administration/roles/{id}`
- `PUT /api/administration/roles/{id}`

Маршрут `GET /api/administration/roles` также возвращает каталог известных прав для редактора ролей.

### Проверка административной целостности

- `GET /api/administration/zero-trust/health`

Маршрут проверяет пользователей, роли, наличие обязательных ролей и полноту набора прав у встроенного `admin`.

### Административные сценарии

- `GET /api/administration/scripts`
- `POST /api/administration/scripts/{scriptID}/run`
- `GET /api/administration/scripts/runs/{runID}/download`

## Связь разделов интерфейса с API

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
