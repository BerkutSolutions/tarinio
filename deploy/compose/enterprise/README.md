# TARINIO Compose Profile - enterprise

`enterprise` is the full multi-node profile for HA control-plane and full logging stack validation.

## Included components

- two `control-plane` nodes behind `api-lb`
- shared `PostgreSQL`
- shared `Redis` coordination
- `Vault` for secret management
- `OpenSearch` for hot search / investigations
- `ClickHouse` for cold analytics / long retention
- `runtime`, `ui`, and `ddos-model`
- optional `toolbox`
- optional `Prometheus` and `Grafana`

## Start

```powershell
cd deploy/compose/enterprise
docker compose --profile tools --profile observability up -d --build
```

## Main endpoints

- `http://localhost:18080` -> TARINIO UI through HA API load balancer
- `http://localhost:18085` -> WAF runtime HTTP
- `https://localhost:18443` -> WAF runtime HTTPS
- `http://localhost:19090` -> Prometheus when `observability` is enabled
- `http://localhost:13000` -> Grafana when `observability` is enabled

## What this profile validates

- multi-node control-plane with leader election and serialized rollout operations
- standalone/full split: `default` stays lean, `enterprise` keeps the heavy integrations
- `Vault` as default secret provider
- `OpenSearch + ClickHouse` hot/cold logging topology
- rolling upgrades and operational tooling in an HA topology

## Useful commands

```powershell
docker compose ps
docker compose logs -f api-lb control-plane-a control-plane-b runtime ddos-model
docker compose --profile tools exec toolbox /tools/cluster-status.sh
docker compose down --remove-orphans
```
