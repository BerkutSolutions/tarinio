# Архитектура TARINIO (обзор)

Базовая версия документации: `1.1.2`

TARINIO — standalone self-hosted WAF на базе NGINX + ModSecurity + OWASP CRS.

## Ключевые принципы

- Source of truth для намерений оператора — control-plane (хранилище + ревизии).
- Runtime не редактируется вручную: он потребляет только активный compiled bundle.
- Любые изменения проходят через ревизии: compile > validate > apply > rollback.

## Источник истины (Stage 0)

Полный набор архитектурных документов (обязательная основа):
- `docs/architecture/adr-001-runtime-control-plane-split.md`
- `docs/architecture/adr-002-config-compilation-model.md`
- `docs/architecture/adr-003-config-rollout-and-rollback.md`
- `docs/architecture/core-domain-model.md`
- `docs/architecture/mvp-deployment-topology.md`
- `docs/architecture/mvp-ui-information-architecture.md`








