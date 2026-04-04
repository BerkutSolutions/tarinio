# Security (EN)

Documentation baseline: `1.0.9`

## Production baseline

- `APP_ENV=prod`
- default secrets are forbidden
- HTTPS is required (built-in TLS or trusted reverse-proxy TLS)
- explicitly restrict `BERKUT_SECURITY_TRUSTED_PROXIES`
- enable tamper-evident audit with `BERKUT_AUDIT_SIGNING_KEY`

The full production baseline is defined in project release documentation and production configuration.



