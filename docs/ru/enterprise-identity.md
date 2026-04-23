# Enterprise Identity

Эта страница относится к текущей ветке документации.

Документ описывает фактически реализованную enterprise-модель идентификации в TARINIO `2.0.8`.

## Что реально реализовано

В TARINIO `2.0.8` доступны:

- интерактивный вход через `OIDC`
- provisioning пользователей через `SCIM 2.0`
- маппинг внешних групп на локальные роли
- approval workflow для ревизий

Это реальные маршруты control-plane и реальные экраны интерфейса, а не декларативные настройки без рабочего поведения.

## Важная граница: LDAP / AD

В TARINIO `2.0.8` **нет** отдельного прямого сценария входа по `LDAP`-паролю в control-plane.

Поддержка `LDAP` / `Active Directory` реализована через тот путь, который обычно и используется в enterprise:

- пользователи каталога проходят интерактивную аутентификацию через внешний `OIDC`-провайдер
- пользователи каталога могут автоматически заводиться через `SCIM`-bridge
- группы каталога маппятся в роли TARINIO через external group mappings

То есть TARINIO поддерживает `LDAP/AD group mapping`, когда группы каталога проецируются во:

- внешний claim групп в `OIDC`
- значения групп в `SCIM` payload

Это позволяет не встраивать отдельный LDAP-клиент в критический путь входа и не размывать границу между identity provider и control-plane.

## OIDC-вход

Интерактивный enterprise-вход работает через:

- `GET /api/auth/providers`
- `GET /api/auth/oidc/start`
- `GET /api/auth/oidc/callback`

### Поток OIDC

1. Экран входа проверяет через `GET /api/auth/providers`, включён ли enterprise SSO.
2. Если провайдер включён, интерфейс показывает кнопку enterprise SSO.
3. `GET /api/auth/oidc/start` создаёт короткоживущий login challenge и перенаправляет браузер на authorization endpoint провайдера.
4. `GET /api/auth/oidc/callback` обменивает code на token set, валидирует `id_token` и создаёт локальную сессию TARINIO.

### Что именно валидируется

Control-plane проверяет:

- `issuer`
- `audience` (`client_id`)
- подпись токена по `JWKS`
- `nonce`
- признак `email_verified`, если это включено политикой
- допустимый домен email, если настроен allow-list

### Маппинг групп

Настройки OIDC хранят:

- default role IDs
- external group to role mappings
- имя claim, из которого берутся группы

Во время входа TARINIO вычисляет эффективные роли из:

- ролей по умолчанию
- ролей, сопоставленных внешним группам

Итоговый набор ролей фиксируется в локальной записи пользователя и попадает в сессию.

## SCIM provisioning

Маршруты provisioning:

- `GET /scim/v2/ServiceProviderConfig`
- `GET /scim/v2/Users`
- `POST /scim/v2/Users`
- `GET /scim/v2/Users/{id}`
- `PUT /scim/v2/Users/{id}`
- `PATCH /scim/v2/Users/{id}`
- `DELETE /scim/v2/Users/{id}`
- `GET /scim/v2/Groups`

### Аутентификация SCIM

SCIM использует bearer tokens.

Токены создаются и удаляются через:

- `POST /api/administration/enterprise/scim-tokens`
- `DELETE /api/administration/enterprise/scim-tokens/{id}`

Сырые значения токенов показываются только в момент создания. При повторном чтении отдаются только метаданные:

- display name
- prefix
- created at
- last used at

### Как ведут себя пользователи SCIM

Provisioned-пользователи сохраняются в общем каталоге TARINIO с полями:

- `auth_source=scim`
- external identity metadata
- external groups
- `last_synced_at`

Повторный provisioning обновляет существующую запись пользователя, а не создаёт параллельные “теневые” аккаунты.

## Approval workflow

Политика approval настраивается в `Администрирование -> Enterprise` и применяется к маршрутам:

- `POST /api/revisions/compile`
- `POST /api/revisions/{revisionID}/approve`
- `POST /api/revisions/{revisionID}/apply`

### Поведение

Когда approval policy включена:

- новая ревизия получает статус `pending_approval`
- в ревизии сохраняется требуемое число approvals
- reviewer-пользователи могут её approve-ить
- apply блокируется до достижения порога

Запись approval хранит:

- идентификатор пользователя
- username
- комментарий
- время approve

### Self-approval

Self-approval можно отключить. В этом режиме пользователь, который собрал ревизию, не может засчитать approve самому себе.

## Расширение модели пользователя

Enterprise identity добавляет в запись пользователя:

- `auth_source`
- `external_id`
- `external_groups`
- `last_synced_at`

Это позволяет держать в одном каталоге:

- локальных bootstrap/local пользователей
- пользователей `OIDC`
- пользователей `SCIM`

и при этом не терять источник идентичности и audit context.

## Что есть в UI

Раздел `Администрирование -> Enterprise` используется для:

- включения `OIDC`
- настройки issuer/client параметров
- задания default roles
- настройки external group mappings
- включения `SCIM`
- выпуска и отзыва `SCIM` tokens
- настройки approval policy
- выгрузки signed support bundle

## Рекомендуемая enterprise-схема

Для корпоративной directory-интеграции рекомендуется:

1. Оставить локальный bootstrap-доступ только как аварийный контур.
2. Подключить `OIDC`-провайдер, который сам работает с `LDAP/AD`.
3. Проецировать directory-группы в OIDC groups claim.
4. При необходимости автоматического lifecycle использовать тот же словарь групп через `SCIM`.
5. Сопоставить внешние группы с локальными ролями TARINIO в enterprise settings.

## Связанные документы

- [API](api.md)
- [Безопасность](security.md)
- [Evidence And Releases](evidence-and-releases.md)
- [Интерфейс](ui.md)
