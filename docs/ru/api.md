# API (RU)

Р‘Р°Р·РѕРІР°СЏ РІРµСЂСЃРёСЏ РґРѕРєСѓРјРµРЅС‚Р°С†РёРё: `1.0.2`

## Release notes

### 1.0.2 (2026-04-03)

- Р‘СЂРµРЅРґРёРЅРі TARINIO Рё РµРґРёРЅР°СЏ РІРµСЂСЃРёСЏ С‡РµСЂРµР· `meta.go`.
- Р”РѕР±Р°РІР»РµРЅ СЂР°СЃС€РёСЂРµРЅРЅС‹Р№ `/api/app/meta` (version/name/links).

## РћР±С‰РёРµ РїСЂР°РІРёР»Р°

- Р’СЃРµ РѕС‚РІРµС‚С‹ вЂ” JSON.
- РђСѓС‚РµРЅС‚РёС„РёРєР°С†РёСЏ вЂ” С‡РµСЂРµР· СЃРµСЃСЃРёСЋ (cookie) РїРѕСЃР»Рµ Р»РѕРіРёРЅР°.
- РџСЂР°РІР° РїСЂРѕРІРµСЂСЏСЋС‚СЃСЏ РЅР° СЃРµСЂРІРµСЂРµ РґР»СЏ РєР°Р¶РґРѕРіРѕ endpoint (zero-trust).

## РЎРёСЃС‚РµРјРЅС‹Рµ endpoints

- `GET /healthz` вЂ” liveness.
- `GET /api/setup/status` вЂ” СЃС‚Р°С‚СѓСЃ РїРµСЂРІРёС‡РЅРѕР№ РЅР°СЃС‚СЂРѕР№РєРё.
- `GET /api/app/meta` вЂ” РІРµСЂСЃРёСЏ Рё РјРµС‚Р°РґР°РЅРЅС‹Рµ РїСЂРёР»РѕР¶РµРЅРёСЏ.

## РќР°СЃС‚СЂРѕР№РєРё

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

## РљРѕРЅС„РёРіСѓСЂР°С†РёСЏ (РѕСЃРЅРѕРІРЅС‹Рµ СЂРµСЃСѓСЂСЃС‹)

- Sites: `/api/sites` Рё `/api/sites/{id}`
- Upstreams: `/api/upstreams` Рё `/api/upstreams/{id}`
- Certificates: `/api/certificates` Рё `/api/certificates/{id}`
- TLS configs: `/api/tls-configs` Рё `/api/tls-configs/{id}`
- Upload material: `POST /api/certificate-materials/upload`
- ACME:
  - `POST /api/certificates/acme/issue`
  - `POST /api/certificates/acme/renew/{certificateID}`

## Policies

- WAF policies: `/api/waf-policies` Рё `/api/waf-policies/{id}`
- Access policies: `/api/access-policies` Рё `/api/access-policies/{id}`
- Rate-limit policies: `/api/rate-limit-policies` Рё `/api/rate-limit-policies/{id}`
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

