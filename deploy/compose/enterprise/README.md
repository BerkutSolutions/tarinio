# TARINIO Compose Profile - enterprise

`enterprise` is the full multi-node profile for HA control-plane and full logging stack validation.

## Included components

- two `control-plane` nodes behind `api-lb`
- shared `PostgreSQL`
- shared `Redis` coordination
- `Vault` for secret management
- `OpenSearch` for hot search / investigations
- `ClickHouse` for cold analytics / long retention
- `runtime`, `ui`, and `tarinio-sentinel`
- optional `toolbox`
- optional `Prometheus` and `Grafana`

## Start

```powershell
cd deploy/compose/enterprise
docker compose --profile tools --profile observability up -d --build
```

## Secure storage prerequisites

Before an Enterprise Compose upgrade, prepare external storage material and
set its paths in `.env`. Compose mounts it read-only; it never creates,
replaces, or rotates a certificate, key, password, or user hash.

- `CLICKHOUSE_TLS_DIR` must contain `tls.crt`, `tls.key`, and `ca.crt` for the
  `clickhouse` DNS name. `CLICKHOUSE_USERS_DIR` must provide a
  `50-waf-runtime.xml` with only the non-default `waf-runtime` user and its
  password hash.
- `OPENSEARCH_TLS_DIR` must contain `node.crt`, `tls.key`, and `ca.crt` for
  `opensearch`. `OPENSEARCH_SECURITY_DIR` must contain the existing OpenSearch
  Security configuration, including a non-demo `waf-runtime` user whose hash
  matches `OPENSEARCH_PASSWORD`.
- Set `CLICKHOUSE_PASSWORD` and `OPENSEARCH_PASSWORD` to the corresponding
  existing credentials. The application uses HTTPS and the mounted CA files;
  plaintext ports and the OpenSearch demo security configuration are disabled.

For a certificate rollover, retain the old trust chain while adding the new
one, roll consumers, then rotate the server certificate. Preserve the existing
named volumes (`waf-ha-clickhouse-data`, `waf-ha-opensearch-data`) throughout
the migration. Missing or mismatched material fails the storage service rather
than falling back to plaintext or default accounts.

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
docker compose logs -f api-lb control-plane-a control-plane-b runtime tarinio-sentinel
docker compose --profile tools exec toolbox /tools/cluster-status.sh
docker compose down --remove-orphans
```
## Observability credentials

Before starting `--profile observability`, set unique
`HA_GRAFANA_ADMIN_USER` and `HA_GRAFANA_ADMIN_PASSWORD` in `.env`. Prometheus
and Grafana are intentionally bound to loopback; publish them remotely only
through an authenticated ingress or tunnel.
