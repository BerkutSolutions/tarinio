# Наблюдаемость

Эта страница относится к текущей ветке документации.

## Область

В `2.0.2` TARINIO включает production-oriented observability stack для HA-развёртываний:

- Prometheus-compatible metrics от control-plane;
- Prometheus-compatible metrics от runtime launcher;
- PostgreSQL и Redis exporters в HA lab;
- готовые Grafana dashboards для операторов;
- strict post-upgrade smoke validation, который может проверять metrics availability.

## Metrics endpoints

Control-plane:

- `/metrics`
- защищён токеном `CONTROL_PLANE_METRICS_TOKEN`
- токен можно передать через `X-TARINIO-Metrics-Token` или `?token=...`

Runtime:

- `/metrics`
- защищён токеном `WAF_RUNTIME_METRICS_TOKEN`
- токен можно передать через `X-TARINIO-Metrics-Token` или `?token=...`

## Что измеряется

Control-plane metrics включают:

- HTTP request totals по route, method, status и node;
- latency histograms;
- revision compile/apply outcomes;
- HA lock acquisition и wait time;
- leader-only task execution;
- runtime reload outcomes;
- build/version presence.

Runtime metrics включают:

- HTTP requests и latency для launcher API;
- runtime reload totals;
- bundle load totals;
- liveness и readiness gauges;
- active revision gauge.
