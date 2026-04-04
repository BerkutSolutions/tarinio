# API (RU)

Базовая версия документации: `1.0.10`

## Release notes

### 1.0.10 (2026-04-03)

- Брендинг TARINIO и единая версия через `meta.go`.
- Добавлен расширенный `/api/app/meta` (version/name/links).

## Общие правила

- Все ответы — JSON.
- Аутентификация — через сессию (cookie) после логина.
- Права проверяются на сервере для каждого endpoint (zero-trust).

## Системные endpoints

- `GET /healthz` — liveness.
- `GET /api/setup/status` — статус первичной настройки.
- `GET /api/app/meta` — версия и метаданные приложения.

## Настройки

- `GET /api/settings/runtime`
- `PUT /api/settings/runtime`
- `POST /api/settings/runtime/check-updates`

## Auth

- `POST /api/auth/bootstrap`
- `POST /api/auth/login`
- `POST /api/auth/login/2fa`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/auth/change-password`
- `GET /api/auth/2fa/status`
- `POST /api/auth/2fa/setup`
- `POST /api/auth/2fa/enable`
- `POST /api/auth/2fa/disable`
- Passkeys:
  - `POST /api/auth/passkeys/login/begin`
  - `POST /api/auth/passkeys/login/finish`
  - `POST /api/auth/login/2fa/passkey/begin`
  - `POST /api/auth/login/2fa/passkey/finish`
  - `GET /api/auth/passkeys`
  - `POST /api/auth/passkeys/register/begin`
  - `POST /api/auth/passkeys/register/finish`
  - `PUT /api/auth/passkeys/{id}/rename`
  - `DELETE /api/auth/passkeys/{id}`

## Конфигурация (основные ресурсы)

- Sites: `/api/sites` и `/api/sites/{id}`
- Upstreams: `/api/upstreams` и `/api/upstreams/{id}`
- Certificates: `/api/certificates` и `/api/certificates/{id}`
- TLS configs: `/api/tls-configs` и `/api/tls-configs/{id}`
- Upload material: `POST /api/certificate-materials/upload`
- ACME:
  - `POST /api/certificates/acme/issue`
  - `POST /api/certificates/acme/renew/{certificateID}`

## Policies

- WAF policies: `/api/waf-policies` и `/api/waf-policies/{id}`
- Access policies: `/api/access-policies` и `/api/access-policies/{id}`
- Rate-limit policies: `/api/rate-limit-policies` и `/api/rate-limit-policies/{id}`
- Easy site profiles: `/api/easy-site-profiles/{siteID}`
- Catalog: `GET /api/easy-site-profiles/catalog/countries`
- Anti-DDoS: `/api/anti-ddos/settings`

## Observability / reports

- Events: `GET /api/events`
- Requests: `GET /api/requests`
- Revisions report: `GET /api/reports/revisions`
- Dashboard stats: `GET /api/dashboard/stats`
- Audit: `GET /api/audit`

## Revisions

- Compile: `POST /api/revisions/compile`
- Apply: `POST /api/revisions/{revisionID}/apply`




