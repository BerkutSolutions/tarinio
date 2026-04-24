# API

Эта страница относится к текущей ветке документации.

Документ описывает актуальный HTTP API control-plane для версии `3.0.2`. Каталог собран по реально зарегистрированным маршрутам из `control-plane/internal/httpserver/server.go`.

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

### `GET /core-docs/api/setup/status`

Статус первичной инициализации. Используется:

- при входе;
- в первичной настройке;
- в логике предвходных проверок интерфейса.

### `GET /core-docs/api/app/meta`

Метаданные приложения:

- версия;
- название продукта;
- ссылка на репозиторий;
- сведения о проверке обновлений;
- признаки режима высокой доступности и идентификатор текущего узла.

### `POST /core-docs/api/app/ping`

Фоновая проверка активной пользовательской сессии.

### `GET /core-docs/api/app/compat`

Отчёт о совместимости компонентов приложения и рантайма.

### `POST /core-docs/api/app/compat/fix`

Попытка автоматически исправить обнаруженную проблему совместимости.

## Системные настройки рантайма

### `GET /core-docs/api/settings/runtime`

Чтение настроек рантайма для раздела `Настройки -> Общие`.

Используется для отображения:

- режима развёртывания;
- состояния проверки обновлений;
- метаданных текущей версии.

### `PUT /core-docs/api/settings/runtime`

Обновление настроек рантайма. Через интерфейс изменяются:

- `update_checks_enabled`;
- сроки хранения `logs`, `activity`, `events`, `bans`.

### `POST /core-docs/api/settings/runtime/check-updates`

Ручная или фоновая проверка доступности обновлений.

### `GET /core-docs/api/settings/runtime/storage-indexes?storage_indexes_limit=N&storage_indexes_offset=N`

Просмотр индексов хранения для раздела `Настройки -> Хранилище`.

### `DELETE /core-docs/api/settings/runtime/storage-indexes?date=YYYY-MM-DD`

Удаление индекса хранения за указанную дату.

## OWASP CRS

### `GET /core-docs/api/owasp-crs/status`

Статус установленного релиза CRS.

### `POST /core-docs/api/owasp-crs/check-updates`

Проверка доступности обновления без фактической установки.

### `POST /core-docs/api/owasp-crs/update`

Запуск обновления CRS. Используется также для включения почасовой автоматической проверки обновлений.

## Аутентификация и учётная запись

### Первичный вход и сессия

- `POST /core-docs/api/auth/bootstrap`
- `POST /core-docs/api/auth/login`
- `POST /core-docs/api/auth/login/2fa`
- `POST /core-docs/api/auth/logout`
- `GET /core-docs/api/auth/me`

### Второй фактор

- `GET /core-docs/api/auth/2fa/status`
- `POST /core-docs/api/auth/2fa/setup`
- `POST /core-docs/api/auth/2fa/enable`
- `POST /core-docs/api/auth/2fa/disable`

### Пароль

- `POST /core-docs/api/auth/change-password`

### Passkeys

- `POST /core-docs/api/auth/passkeys/login/begin`
- `POST /core-docs/api/auth/passkeys/login/finish`
- `POST /core-docs/api/auth/login/2fa/passkey/begin`
- `POST /core-docs/api/auth/login/2fa/passkey/finish`
- `GET /core-docs/api/auth/passkeys`
- `POST /core-docs/api/auth/passkeys/register/begin`
- `POST /core-docs/api/auth/passkeys/register/finish`
- `PUT /core-docs/api/auth/passkeys/{id}/rename`
- `DELETE /core-docs/api/auth/passkeys/{id}`

## Конфигурационные ресурсы

### Сайты

- `GET /core-docs/api/sites`
- `POST /core-docs/api/sites`
- `GET /core-docs/api/sites/{id}`
- `PUT /core-docs/api/sites/{id}`
- `DELETE /core-docs/api/sites/{id}`
- `POST /core-docs/api/sites/{id}/ban`
- `POST /core-docs/api/sites/{id}/unban`

### Upstream-сервисы

- `GET /core-docs/api/upstreams`
- `POST /core-docs/api/upstreams`
- `GET /core-docs/api/upstreams/{id}`
- `PUT /core-docs/api/upstreams/{id}`
- `DELETE /core-docs/api/upstreams/{id}`

### Сертификаты

- `GET /core-docs/api/certificates`
- `POST /core-docs/api/certificates`
- `GET /core-docs/api/certificates/{id}`
- `PUT /core-docs/api/certificates/{id}`
- `DELETE /core-docs/api/certificates/{id}`

### TLS-конфигурации

- `GET /core-docs/api/tls-configs`
- `POST /core-docs/api/tls-configs`
- `GET /core-docs/api/tls-configs/{siteID}`
- `PUT /core-docs/api/tls-configs/{siteID}`
- `DELETE /core-docs/api/tls-configs/{siteID}`

### Автопродление TLS

- `GET /core-docs/api/tls/auto-renew`
- `PUT /core-docs/api/tls/auto-renew`

### Операции с сертификатными материалами

- `POST /core-docs/api/certificate-materials/upload`
- `POST /core-docs/api/certificate-materials/import-archive`
- `POST /core-docs/api/certificate-materials/export`
- `GET /core-docs/api/certificate-materials/export/{certificateID}`

### ACME

- `POST /core-docs/api/certificates/acme/issue`
- `POST /core-docs/api/certificates/acme/renew/{certificateID}`
- `POST /core-docs/api/certificates/self-signed/issue`

## Политики

### WAF-политики

- `GET /core-docs/api/waf-policies`
- `POST /core-docs/api/waf-policies`
- `GET /core-docs/api/waf-policies/{id}`
- `PUT /core-docs/api/waf-policies/{id}`
- `DELETE /core-docs/api/waf-policies/{id}`

### Политики доступа

- `GET /core-docs/api/access-policies`
- `POST /core-docs/api/access-policies`
- `POST /core-docs/api/access-policies/upsert`
- `PUT /core-docs/api/access-policies/upsert`
- `GET /core-docs/api/access-policies/{id}`
- `PUT /core-docs/api/access-policies/{id}`
- `DELETE /core-docs/api/access-policies/{id}`

### Политики ограничения скорости

- `GET /core-docs/api/rate-limit-policies`
- `POST /core-docs/api/rate-limit-policies`
- `GET /core-docs/api/rate-limit-policies/{id}`
- `PUT /core-docs/api/rate-limit-policies/{id}`
- `DELETE /core-docs/api/rate-limit-policies/{id}`

### Упрощённые профили сайтов

- `GET /core-docs/api/easy-site-profiles/{siteID}`
- `PUT /core-docs/api/easy-site-profiles/{siteID}`
- `POST /core-docs/api/easy-site-profiles/{siteID}`
- `GET /core-docs/api/easy-site-profiles/catalog/countries`

### Anti-DDoS

- `GET /core-docs/api/anti-ddos/settings`
- `POST /core-docs/api/anti-ddos/settings`
- `PUT /core-docs/api/anti-ddos/settings`

## Наблюдаемость и отчёты

### Запросы и события

- `GET /core-docs/api/events`
- `GET /core-docs/api/requests`

### Dashboard

- `GET /core-docs/api/dashboard/stats`
- `GET /core-docs/api/dashboard/containers/overview`
- `GET /core-docs/api/dashboard/containers/logs`
- `GET /core-docs/api/dashboard/containers/issues`

### Отчёты

- `GET /core-docs/api/reports/revisions`

### Аудит

- `GET /core-docs/api/audit`

## Ревизии

### `GET /core-docs/api/revisions`

Агрегированный каталог ревизий, который используется разделом `Ревизии`. Маршрут возвращает:

- список сервисов;
- список ревизий;
- сводные счётчики;
- ленту статусов и событий применения.

### `POST /core-docs/api/revisions/compile`

Компиляция новой ревизии.

### `POST /core-docs/api/revisions/{revisionID}/apply`

Применение выбранной ревизии.

Требуемое право:

- `revisions.write`

### `POST /core-docs/api/revisions/{revisionID}/approve`

Согласование ревизии, если в enterprise-настройках включена политика согласований.

Требуемое право:

- `revisions.approve`

### `DELETE /core-docs/api/revisions/{revisionID}`

Удаление неактивной ревизии. Связанные снимки и служебные данные удаляются вместе с ней, но активную ревизию удалить нельзя.

Требуемое право:

- `revisions.write`

### `DELETE /core-docs/api/revisions/statuses`

Очистка ленты статусов ревизий. При этом за самими ревизиями сохраняется информация о последнем успешном или неуспешном применении.

## Администрирование

### Пользователи

- `GET /core-docs/api/administration/users`
- `POST /core-docs/api/administration/users`
- `GET /core-docs/api/administration/users/{id}`
- `PUT /core-docs/api/administration/users/{id}`

### Роли

- `GET /core-docs/api/administration/roles`
- `POST /core-docs/api/administration/roles`
- `GET /core-docs/api/administration/roles/{id}`
- `PUT /core-docs/api/administration/roles/{id}`

Маршрут `GET /core-docs/api/administration/roles` также возвращает каталог известных прав для редактора ролей.

### Проверка административной целостности

- `GET /core-docs/api/administration/zero-trust/health`

Маршрут проверяет пользователей, роли, наличие обязательных ролей и полноту набора прав у встроенного `admin`.

### Административные сценарии

- `GET /core-docs/api/administration/scripts`
- `POST /core-docs/api/administration/scripts/{scriptID}/run`
- `GET /core-docs/api/administration/scripts/runs/{runID}/download`

## Связь разделов интерфейса с API

- `Dashboard`: `dashboard/*`, `events`, `requests`
- `Сайты`: `sites`, `upstreams`, `tls-configs`, `certificates`, `access-policies`, `easy-site-profiles`
- `Anti-DDoS`: `anti-ddos/settings`, `events`
- `OWASP CRS`: `owasp-crs/*`
- `TLS`: `certificates`, `tls-configs`, `tls/auto-renew`, `certificate-materials/*`, `certificates/acme/*`
- `Запросы`: `requests`, `settings/runtime`, `sites`
- `Ревизии`: `revisions`, `revisions/{id}/apply`, `revisions/{id}/approve`, `revisions/statuses`
- `События`: `events`, `sites`
- `Баны`: `sites/{id}/ban`, `sites/{id}/unban`, `events`, `access-policies`
- `Администрирование`: `administration/users*`, `administration/roles*`, `administration/zero-trust/health`, `administration/scripts*`
- `Активность`: `audit`
- `Настройки`: `settings/runtime`, `app/meta`
- `Профиль`: `auth/me`, `auth/change-password`, `auth/2fa/*`, `auth/passkeys/*`



