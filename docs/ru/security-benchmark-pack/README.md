# Security Benchmark Pack

Раздел задает воспроизводимый пакет security-валидации для TARINIO `3.0.2`.

## Назначение

- проверять качество защиты на реалистичном атакующем трафике;
- измерять ложные срабатывания на нормальном трафике;
- подтверждать безопасность rollout перед выходом в production.

## Матрица сценариев

1. Human baseline трафик (нормальное поведение браузера/API).
2. Scanner/recon трафик (`/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`).
3. Brute-force и auth abuse (`/login`, token endpoints).
4. L7 flood и burst-паттерны.
5. Mixed production-like трафик с легитимными пиками.

## Ключевые метрики

- `false_positive_rate` по сценариям и профилям;
- precision/recall там, где есть разметка;
- влияние на latency (`p95`, `p99`) относительно baseline;
- потребление `cpu` и `memory` под нагрузкой;
- завершение challenge и стабильность bypass в anti-bot.

## Артефакты доказательств

- manifest сценариев и replay-входы;
- raw-метрики и summary-отчеты;
- корреляция revision/result для compile/apply;
- подписанные release-артефакты (`release-manifest`, `signature`, `sbom`, `provenance`).

## Связь с 3.0.2

Benchmark pack является базой валидации для:

- security-профилей;
- API positive security;
- двухслойного anti-bot challenge;
- улучшенной explainability в Sentinel;
- threat-intel и geo-контекста.

## Pass/Fail Criteria

- alse_positive_rate stays within the approved profile baseline;
- labeled attack scenarios do not regress recall against the previous release;
- p95/p99 latency regression stays within the approved SLO window;
- cpu/memory stay inside the benchmark environment budget.
