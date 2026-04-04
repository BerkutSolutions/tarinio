# Berkut Solutions - TARINIO - Документация (RU)

Базовая версия документации: `1.0.4`

## Разделы

1. Архитектура (обзор): `docs/ru/architecture.md`
2. API: `docs/ru/api.md`
3. Безопасность: `docs/ru/security.md`
4. Deploy: `docs/ru/deploy.md`
5. Runbook (запуск и восстановление): `docs/ru/runbook.md`
6. Обновление и откат: `docs/ru/upgrade.md`
7. Бэкапы: `docs/ru/backups.md`
8. CLI команды: `docs/CLI_COMMANDS.md`
9. Operator docs:
   - Anti-DDoS: `docs/ru/operators/anti-ddos-runbook.md`
   - L4 guard: `docs/ru/operators/runtime-l4-guard.md`
   - Runtime filesystem: `docs/ru/operators/runtime-filesystem-contract.md`
   - WAF tuning: `docs/ru/operators/waf-tuning-guide.md`
   - E2E validation: `docs/ru/operators/stage-1-e2e-validation.md`
10. Stage 0 архитектурные документы: `docs/architecture/`
11. OSS документация (RU):
   - `docs/ru/oss/SECURITY.md`
   - `docs/ru/oss/CONTRIBUTING.md`
   - `docs/ru/oss/CODE_OF_CONDUCT.md`
   - `docs/ru/oss/SUPPORT.md`

## Контекст релиза 1.0.4

- Брендинг продукта приведен к `Berkut Solutions - TARINIO`.
- Версия приложения централизована через `control-plane/internal/appmeta/meta.go`.
- UI и словари i18n синхронизированы для ключей Easy site и Anti-DDoS сценариев.
- Добавлены глобальные настройки Anti-DDoS и их применение через pipeline ревизий.
- Документация синхронизирована с текущей runtime-моделью standalone WAF.

## Важно

Архитектурные решения Stage 0 зафиксированы и являются источником истины:
- `docs/architecture/`



