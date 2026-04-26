# Enterprise-проверка Sentinel

Дата последнего полного smoke-validation: 26.04.2026

Документ фиксирует enterprise-пакет проверок `tarinio-sentinel` для TARINIO `3.0.5`.

## Что проверяется

Проверяются три режима Anti-DDoS:

1. `classic-only`: только базовый L4 anti-DDoS, adaptive-модель отключена.
2. `hybrid`: базовый anti-DDoS и adaptive-модель работают вместе.
3. `adaptive-only`: основная эскалация идет через adaptive-модель при активном baseline L4-профиле.

Каждый режим валидируется в профилях:

- `default`
- `ha-lab`

## Итоговые результаты

| Профиль | Режим | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives | Baseline conn/rate/burst |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | --- |
| default | classic-only | True | 1116 | 0 | none | 0 | 0 | 120/180/360 |
| default | hybrid | True | 1116 | 255 | drop: 255 | 4 | 0 | 120/180/360 |
| default | adaptive-only | True | 1116 | 262 | drop: 262 | 4 | 0 | 50000/20000/40000 |
| ha-lab | classic-only | True | 1116 | 0 | none | 0 | 0 | 120/240/480 |
| ha-lab | hybrid | True | 1116 | 2 | drop: 2 | 4 | 0 | 120/240/480 |
| ha-lab | adaptive-only | True | 1116 | 259 | drop: 259 | 4 | 0 | 120/240/480 |

## Матрица на 30 акторов

| Группа акторов | Кол-во | Значимость по режимам | Ожидаемый сигнал |
| --- | ---: | --- | --- |
| Нормальные пользователи (`web`, `mobile`, `trusted`) | 10 | все режимы | отсутствие adaptive false positive |
| Scanner-источники | 8 | hybrid, adaptive-only | появление L7 suggestions и снижение trust |
| Brute-force + payload источники | 6 | hybrid, adaptive-only | adaptive-эскалация с explainable-причинами |
| Flood-источники (single + distributed) | 6+ synthetic botnet spread | все режимы | baseline-защита в classic, adaptive/emergency evidence в hybrid/adaptive-only |

Для каждого profile/mode сохраняются:

- `result.json`
- `adaptive.json`
- `l7-suggestions.json`
- `model-state.json`
- `report.md`

## Как воспроизвести

PowerShell:

- `./scripts/smoke-sentinel-enterprise.ps1` (default + ha-lab)
- `./scripts/smoke-sentinel-enterprise.ps1 -SkipHaLab` (только default)
- `./scripts/smoke-sentinel-enterprise.ps1 -SkipDefault` (только ha-lab)
