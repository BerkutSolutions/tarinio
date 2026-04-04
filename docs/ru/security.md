# Безопасность (RU)

Базовая версия документации: `1.0.12`

## Базовые требования для production

- `APP_ENV=prod`
- дефолтные секреты запрещены
- HTTPS обязателен (built-in TLS или trusted reverse-proxy TLS)
- ограничить `BERKUT_SECURITY_TRUSTED_PROXIES`
- включить tamper-evident аудит через `BERKUT_AUDIT_SIGNING_KEY`

Базовые требования фиксируются в релизной документации проекта и production-конфигурации.





