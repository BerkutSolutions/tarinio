# Berkut Solutions - TARINIO - Документация (RU)

Базовая версия документации: `1.0.10`

## Содержание

1. Архитектура (RU): `docs/ru/architecture.md`
2. API: `docs/ru/api.md`
3. Безопасность: `docs/ru/security.md`
4. Deploy: `docs/ru/deploy.md`
5. Runbook (операции и восстановление): `docs/ru/runbook.md`
6. Обновление и откат: `docs/ru/upgrade.md`
7. Бэкапы: `docs/ru/backups.md`
8. CLI команды: `docs/CLI_COMMANDS.md`
9. Operator docs:
   - Anti-DDoS: `docs/ru/operators/anti-ddos-runbook.md`
   - L4 guard: `docs/ru/operators/runtime-l4-guard.md`
   - Runtime filesystem: `docs/ru/operators/runtime-filesystem-contract.md`
   - WAF tuning: `docs/ru/operators/waf-tuning-guide.md`
   - E2E validation: `docs/ru/operators/stage-1-e2e-validation.md`
10. Stage 0 исходные материалы: `docs/architecture/`
11. OSS политика (RU):
   - `docs/ru/oss/SECURITY.md`
   - `docs/ru/oss/CONTRIBUTING.md`
   - `docs/ru/oss/CODE_OF_CONDUCT.md`
   - `docs/ru/oss/SUPPORT.md`

## О продукте

- Продуктовая линейка: `Berkut Solutions - TARINIO`.
- Версия приложения задается в `control-plane/internal/appmeta/meta.go`.
- UI и i18n покрывают управление сайтами, TLS и Anti-DDoS профилями.
- Контур ревизий обеспечивает controlled rollout и rollback конфигураций.

## Примечание

Архитектурные артефакты Stage 0 сохранены в каталоге:
- `docs/architecture/`


