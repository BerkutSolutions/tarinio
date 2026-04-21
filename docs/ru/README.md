# Русская wiki

Эта страница относится к текущей ветке документации.

Это основной русскоязычный индекс документации TARINIO. Он собран как операторская wiki для реальной версии `2.0.2` и покрывает не только архитектуру и deploy, но и повседневную работу в интерфейсе, жизненный цикл ревизий, безопасность, инциденты, резервное копирование и эксплуатационные сценарии.

## С чего начать

- Что такое TARINIO и где искать остальную документацию: `README.md`
- Полный обзор разделов интерфейса: `docs/ru/ui.md`
- Архитектура и границы продукта: `docs/ru/architecture.md`
- API-каталог: `docs/ru/api.md`

## Базовые документы

1. Архитектура: `docs/ru/architecture.md`
2. Интерфейс и сценарии работы: `docs/ru/ui.md`
3. API: `docs/ru/api.md`
4. Безопасность: `docs/ru/security.md`
5. Развёртывание: `docs/ru/deploy.md`
6. Runbook: `docs/ru/runbook.md`
7. Обновление и откат: `docs/ru/upgrade.md`
8. Бэкапы и восстановление: `docs/ru/backups.md`
9. CLI-команды: `docs/CLI_COMMANDS.md`

## Операторские руководства

- Anti-DDoS runbook: `docs/ru/operators/anti-ddos-runbook.md`
- Anti-DDoS модель: `docs/ru/operators/anti-ddos-model.md`
- Runtime L4 guard: `docs/ru/operators/runtime-l4-guard.md`
- Runtime filesystem contract: `docs/ru/operators/runtime-filesystem-contract.md`
- Тюнинг WAF: `docs/ru/operators/waf-tuning-guide.md`
- E2E-валидация: `docs/ru/operators/stage-1-e2e-validation.md`
- OWASP CRS: `docs/ru/operators/owasp-crs.md`
- Let’s Encrypt DNS-01: `docs/ru/operators/letsencrypt-dns.md`

## Что нового и важно в 2.0.2

- Документация синхронизирована с версией приложения из `control-plane/internal/appmeta/meta.go`.
- В wiki отражён реальный UI с вкладками `Dashboard`, `Сайты`, `Anti-DDoS`, `OWASP CRS`, `TLS`, `Запросы`, `Ревизии`, `События`, `Баны`, `Администрирование`, `Активность`, `Настройки` и `Профиль`.
- Учтён новый API-каталог ревизий `GET /api/revisions`, очистка status timeline и обновлённый healthcheck.
- Отдельно описаны onboarding, login, 2FA, passkeys и healthcheck-поток.

## Рекомендуемая последовательность чтения

### Для первого знакомства

1. `README.md`
2. `docs/ru/architecture.md`
3. `docs/ru/ui.md`

### Для внедрения

1. `docs/ru/deploy.md`
2. `docs/ru/security.md`
3. `docs/ru/runbook.md`
4. `docs/ru/backups.md`

### Для эксплуатации

1. `docs/ru/ui.md`
2. `docs/ru/runbook.md`
3. `docs/ru/api.md`
4. `docs/ru/operators/`

## Источник истины

Архитектурные решения Stage 0 остаются обязательной основой и лежат в:

- `docs/architecture/`

Именно эти документы задают продуктовые границы, модель ревизий, компиляции и развёртывания.
