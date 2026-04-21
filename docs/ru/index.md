# Русская wiki

Эта страница относится к текущей ветке документации.

Это основной русскоязычный индекс документации TARINIO. Он собран как операторская wiki для реальной версии `2.0.2` и покрывает архитектуру, интерфейс, API, развёртывание, HA, эксплуатацию, безопасность, резервное копирование и обновления.

## С чего начать

- Общее описание продукта: `README.md`
- Навигатор по документации: [Навигатор](navigator.md)
- Полный обзор интерфейса: [Интерфейс](ui.md)
- Архитектура и границы продукта: [Архитектура](architecture.md)
- Каталог API: [API](api.md)

## Базовые документы

1. [Архитектура](architecture.md)
2. [Интерфейс и сценарии работы](ui.md)
3. [API](api.md)
4. [Безопасность](security.md)
5. [Развёртывание](deploy.md)
6. [HA / multi-node](ha.md)
7. [Observability](observability.md)
8. [Benchmarks](benchmarks.md)
9. [Эксплуатация](runbook.md)
10. [Troubleshooting](troubleshooting.md)
11. [Обновление и откат](upgrade.md)
12. [Резервные копии и восстановление](backups.md)
13. [CLI-команды](cli-commands.md)

## Документы enterprise-уровня

- [Политика поддержки и жизненного цикла](support-lifecycle.md)
- [Матрица совместимости](compatibility-matrix.md)
- [Sizing Guide](sizing.md)
- [Disaster Recovery Guide](disaster-recovery.md)
- [Hardening Guide](hardening.md)
- [Threat Model](threat-model.md)
- [Референсные архитектуры](reference-architectures.md)
- [Operations Cookbook](cookbook.md)
- [Политика release notes](release-policy.md)
- [Известные ограничения](limitations.md)
- [Compliance Mapping](compliance-mapping.md)

## Операторские руководства

- [Anti-DDoS runbook](operators/anti-ddos-runbook.md)
- [Anti-DDoS модель](operators/anti-ddos-model.md)
- [Runtime L4 guard](operators/runtime-l4-guard.md)
- [Контракт файловой системы runtime](operators/runtime-filesystem-contract.md)
- [Тюнинг WAF](operators/waf-tuning-guide.md)
- [E2E-валидация Stage 1](operators/stage-1-e2e-validation.md)
- [OWASP CRS](operators/owasp-crs.md)
- [Let's Encrypt DNS-01](operators/letsencrypt-dns.md)

## Что важно в 2.0.2

- Документация синхронизирована с версией приложения из `control-plane/internal/appmeta/meta.go`.
- В wiki отражён реальный UI с ключевыми разделами и эксплуатационными сценариями.
- Отдельно задокументированы PostgreSQL-backed storage, HA / multi-node, observability и benchmark workflows.
- Onboarding, login, `2FA`, passkeys и healthcheck описаны как реальные product flows.

## Рекомендуемая последовательность чтения

### Для первого знакомства

1. `README.md`
2. [Архитектура](architecture.md)
3. [Интерфейс](ui.md)

### Для внедрения

1. [Развёртывание](deploy.md)
2. [Безопасность](security.md)
3. [Эксплуатация](runbook.md)
4. [Резервные копии](backups.md)
5. [Обновление](upgrade.md)

### Для ежедневной эксплуатации

1. [Интерфейс](ui.md)
2. [Эксплуатация](runbook.md)
3. [API](api.md)
4. `operators/`

## Источник истины

Архитектурные решения Stage 0 остаются обязательной основой:

- `docs/architecture/`

Именно эти документы задают продуктовые границы, модель ревизий, компиляции и развёртывания.
