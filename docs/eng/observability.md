# Observability

This page belongs to the current documentation branch.

## Scope

TARINIO `2.0.10` includes a production-oriented observability stack for HA deployments:

- Prometheus-compatible metrics from the control-plane;
- Prometheus-compatible metrics from the runtime launcher;
- PostgreSQL and Redis exporters in the HA lab;
- pre-provisioned Grafana dashboards for operators;
- strict post-upgrade smoke validation that can verify metrics availability.

## Metrics Endpoints

Control-plane:

- `/metrics`
- protected with `CONTROL_PLANE_METRICS_TOKEN`
- token can be passed through `X-TARINIO-Metrics-Token` or `?token=...`

Runtime:

- `/metrics`
- protected with `WAF_RUNTIME_METRICS_TOKEN`
- token can be passed through `X-TARINIO-Metrics-Token` or `?token=...`

## What Is Measured

Control-plane metrics include:

- HTTP request totals by route, method, status, and node;
- HTTP request duration histograms;
- revision compile totals by result;
- revision apply totals by result;
- HA lock acquisition totals by lock name and result;
- HA lock wait duration histograms;
- leader-only task execution totals by task and result;
- runtime reload call totals by result;
- build/version presence.

Runtime metrics include:

- HTTP request totals and latency for the launcher API;
- runtime reload totals;
- bundle load totals;
- liveness and readiness gauges;
- active revision gauge.

## HA Lab Observability Profile

The bundled lab exposes an operator-ready stack:

- `prometheus`
- `grafana`
- `postgres-exporter`
- `redis-exporter`

Start it with:

```powershell
cd deploy/compose/ha-lab
docker compose --profile tools --profile observability up -d --build
```

Default endpoints:

- `http://localhost:19090` -> Prometheus
- `http://localhost:13000` -> Grafana

Default lab credentials:

- Grafana user: `admin`
- Grafana password: `admin`

## Dashboard Coverage

The packaged Grafana dashboard covers:

- control-plane request rate and p95 latency;
- revision compile/apply activity;
- HA coordination and lock behavior;
- runtime readiness;
- runtime control API rate;
- PostgreSQL and Redis exporter health.

## Production Guidance

For production environments:

- keep metrics endpoints token-protected;
- scrape every control-plane node directly, not only the API load balancer;
- alert on failed apply spikes, lock contention, and loss of runtime readiness;
- retain Prometheus data long enough to compare release windows and attack periods;
- include metrics validation in upgrade smoke tests.

