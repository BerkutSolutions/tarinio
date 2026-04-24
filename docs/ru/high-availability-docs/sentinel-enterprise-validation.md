# Enterprise-проверка Sentinel

Последний полный smoke-прогон: 24.04.2026 17:31:33 +03:00.

Этот документ фиксирует воспроизводимый enterprise validation pack для `tarinio-sentinel` в TARINIO `3.0.1`.

## Что проверяется

Проверка покрывает полный контур работы Sentinel:

1. Приём runtime access-log событий.
2. Multi-signal scoring и пересчёт trust.
3. Explainability-вывод (`reason_codes`, `top_signals`).
4. Ограниченный publish (`max_published_entries`, publish interval).
5. Генерация L7 suggestions по scanner-паттернам.
6. Защита от ложных срабатываний на нормальном трафике.

Проверка выполняется для двух профилей:

- `default`
- `high-availability-lab`

## Итоговый результат тестов

| Профиль | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| default | true | 856 | 141 | drop: 141 | 4 | 0 |
| high-availability-lab | true | 856 | 142 | drop: 142 | 4 | 0 |

## Enterprise-матрица трафика

| Сценарий | Паттерн трафика | Ожидаемый enterprise-сигнал |
| --- | --- | --- |
| Нормальный baseline | Benign dashboard/API трафик | Нет adaptive-блокировок для нормального пользователя |
| Scanner discovery | `/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit` | Падение trust и появление L7 suggestions |
| Brute force | Повторные запросы на `/login` | Рост adaptive-risk для источника атаки |
| XSS / SQLi / RCE probes | Script/UNION/shell payloads | Рост risk score с explainable причинами |
| Single-source flood | Burst с одного IP | Срабатывает emergency single-source путь |
| Distributed flood | Burst с множества IP | Срабатывает emergency botnet-like путь |
| High-cardinality noise | Большое число уникальных путей и источников | State остаётся bounded, publish остаётся capped |

## Финальный вывод для Enterprise

`tarinio-sentinel` прошёл воспроизводимую enterprise-проверку для standalone и High Availability-ready профилей:

- adaptive-детекция активна и объяснима,
- нормальный трафик чистый (`0` normal false positives),
- scanner-поведение выводится в L7 suggestions до постоянного применения.

Итоговый вывод теста: продукт соответствует ожидаемому enterprise-поведению для контролируемой security-автоматизации в production-like среде.

## Как воспроизвести

PowerShell:

`./scripts/smoke-sentinel-enterprise.ps1`

Запуск по отдельному профилю:

`./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/sentinel-smoke/manual/default`

`./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/ha-lab/docker-compose.yml -ProfileName ha-lab -OutputDir .work/sentinel-smoke/manual/ha-lab`
