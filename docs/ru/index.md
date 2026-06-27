# Русская wiki

Эта страница относится к текущей ветке документации.

Это основной русскоязычный раздел документации TARINIO. Он собран как практическая wiki для операторов, администраторов и команд внедрения и отражает фактическое состояние версии `1.3.5`.

## С чего начать

- Общее описание продукта: [Навигатор](navigator.md)
- Обзор разделов и маршрутов: [Навигатор](navigator.md)
- Работа в интерфейсе: [Интерфейс](core-docs/ui.md)
- Устройство платформы: [Архитектура](architecture-docs/architecture.md)
- Каталог HTTP API: [API](core-docs/api.md)

## Основные документы

1. [Архитектура](architecture-docs/architecture.md)
2. [Интерфейс](core-docs/ui.md)
3. [API](core-docs/api.md)
4. [Безопасность](core-docs/security.md)
5. [Развёртывание](core-docs/deploy.md)
6. [Security Profiles](core-docs/security-profiles.md)
7. [API Positive Security](core-docs/api-positive-security.md)
8. [Высокая доступность](high-availability-docs/high-availability.md)
9. [Наблюдаемость](core-docs/observability.md)
10. [Эксплуатация](core-docs/runbook.md)
11. [Обновление и откат](core-docs/upgrade.md)
12. [Резервное копирование и восстановление](core-docs/backups.md)
13. [CLI-команды](core-docs/cli-commands.md)
14. [Справочник env-параметров WAF](core-docs/waf-env-reference.md)
15. [Архитектура логгирования](architecture-docs/logging-architecture.md)
16. [Управление секретами](core-docs/secret-management.md)
17. [Контракт миграции и совместимости](core-docs/migration-compatibility.md)
18. [PCI/ASV baseline для внешнего WAF](core-docs/pci-asv-external-waf-baseline.md)

## Документы для эксплуатации

- [Политика поддержки и жизненного цикла](core-docs/support-lifecycle.md)
- [Матрица совместимости](core-docs/compatibility-matrix.md)
- [Подбор ресурсов](core-docs/sizing.md)
- [План аварийного восстановления](core-docs/disaster-recovery.md)
- [Укрепление безопасности](core-docs/hardening.md)
- [Модель угроз](core-docs/threat-model.md)
- [Референсные архитектуры](architecture-docs/reference-architectures.md)
- [Практические сценарии эксплуатации](core-docs/cookbook.md)
- [Политика выпуска релизов](core-docs/release-policy.md)
- [Известные ограничения](core-docs/limitations.md)
- [Карта соответствия требованиям](core-docs/compliance-mapping.md)

## Операторские руководства

- [Руководство по Anti-DDoS](model-docs/anti-ddos-runbook.md)
- [Модель Anti-DDoS](model-docs/anti-ddos-model.md)
- [TARINIO Sentinel](model-docs/tarinio-sentinel.md)
- [Защита на сетевом уровне](model-docs/runtime-l4-guard.md)
- [Контракт файловой системы runtime](model-docs/runtime-filesystem-contract.md)
- [Тюнинг WAF](operators/waf-tuning-guide.md)
- [Проверка Stage 1 end-to-end](high-availability-docs/stage-1-e2e-validation.md)
- [OWASP CRS](operators/owasp-crs.md)
- [Let's Encrypt DNS-01](operators/letsencrypt-dns.md)
- [PCI/ASV baseline и preflight для внешнего WAF](core-docs/pci-asv-external-waf-baseline.md)
- [Подключение внешнего production-сайта через WAF](integration-docs/external-prod-site-via-waf.md)

## Что важно в `1.3.x`

- Mutual TLS (mTLS): проверка клиентских сертификатов на incoming-соединениях (`ssl_verify_client`) и mTLS к upstream (`proxy_ssl_certificate`).
- JA3/JA4 TLS fingerprinting: сбор fingerprint через nginx, блокировка по blacklist, sentinel-сигнал `signal_ja3_risk`.
- Virtual Patching: временные правила блокировки (URI/Header/Body) с TTL через API без перезапуска nginx.
- Credential Stuffing Detection: счётчики неудачных авторизаций в Sentinel, auto-ban по порогу.
- Расширенные сигналы Sentinel: `antibot_fail`, `bad_behavior_hits`, JA3-корреляция.
- Upstream Health Check: nginx active health checks с auto-degradation при сбоях.
- Geo/Time Windows: ограничение доступа по GeoIP и расписанию (дни недели / часы).
- WebSocket Inspection: инспекция WS-фреймов с блокировкой по regex-паттернам, rate и size limits.
- HTTP Request Smuggling hardening: `request_id`, `chunked_transfer_encoding off`, strict header validation.
- Документация синхронизирована с фактическим состоянием продукта версии `1.3.5`.

## Рекомендуемый порядок чтения

### Для первого знакомства

1. [Навигатор](navigator.md)
2. [Архитектура](architecture-docs/architecture.md)
3. [Интерфейс](core-docs/ui.md)

### Для внедрения

1. [Развёртывание](core-docs/deploy.md)
2. [Безопасность](core-docs/security.md)
3. [Эксплуатация](core-docs/runbook.md)
4. [Резервное копирование](core-docs/backups.md)
5. [Обновление](core-docs/upgrade.md)

### Для ежедневной работы

1. [Интерфейс](core-docs/ui.md)
2. [Эксплуатация](core-docs/runbook.md)
3. [API](core-docs/api.md)
4. Каталог `operators/`

## Источник архитектурных решений

Базовые архитектурные решения Stage 0 остаются обязательной основой проекта и хранятся в каталоге:

- `docs/architecture/`

Эти документы задают продуктовые границы, модель ревизий, процесс компиляции и схему развёртывания.
