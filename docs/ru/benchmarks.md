# Набор бенчмарков

Эта страница относится к текущей ветке документации.

## Область

Публичный benchmark pack в `deploy/compose/ha-lab` предназначен для воспроизводимых локальных измерений без перегруза рабочей станции.

Он отвечает на три вопроса:

- насколько быстро стек отвечает под нормальным трафиком;
- как он ведёт себя на защищённых endpoint, когда включаются лимиты;
- остаются ли HA control-plane и observability стабильными во время прогона.

## Сценарии

Benchmark pack запускает:

- baseline traffic по обычной tenant-странице;
- rate-limited сценарий для `tenant-01`;
- API health checks против `api-lb`;
- опциональный Prometheus query для проверки observability availability.

## Что сохраняется

`benchmark-http.sh` возвращает JSON с:

- total request count;
- distribution по status code;
- average latency;
- minimum и maximum latency;
- `p50`, `p95`, `p99`.

`benchmark-pack.sh` собирает всё в один итоговый JSON документ.
