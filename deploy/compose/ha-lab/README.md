# HA Lab Profile

Start:

```powershell
cd deploy/compose/ha-lab
docker compose --profile tools --profile observability up -d --build
```

Main endpoints:
- `http://localhost:18080` -> TARINIO UI through HA API load balancer
- `http://localhost:18085` -> WAF runtime HTTP
- `https://localhost:18443` -> WAF runtime HTTPS for the management site
- `http://localhost:19090` -> Prometheus when `observability` profile is enabled
- `http://localhost:13000` -> Grafana when `observability` profile is enabled

## What This Profile Proves

- two `control-plane` nodes share PostgreSQL state and Redis coordination;
- background jobs (`startup self-test`, `dev fast start`, `auto-apply`, `tls auto-renew`) run under leader election;
- revision compile/apply is serialized across nodes;
- the API is fronted by `api-lb`, so UI and tools do not bind to a single node;
- Prometheus scrapes both control-plane nodes and the runtime directly, while Grafana ships with a ready dashboard;
- rolling control-plane upgrades can be validated without API downtime;
- a reproducible benchmark pack writes JSON summaries for public sharing and regression tracking;
- the whole profile stays lightweight enough for a workstation: service-level CPU and memory caps keep the stack comfortably below the requested budget.

## Topology

- `api-lb`
  - nginx load balancer for `control-plane-a` and `control-plane-b`
- `control-plane-a`
- `control-plane-b`
- `postgres`
- `redis`
- `ui`
- `runtime`
- `ddos-model`
- `demo-app`
- `toolbox` profile
  - helper container with `curl`, `jq`, and `waf-cli`

## First Run

1. Start the profile.
2. Open `http://localhost:18080/login`.
3. Log in with the bootstrap credentials from `.env`.
4. Provision the 20 logical demo services:

```powershell
docker compose --profile tools exec toolbox /tools/provision-20-services.sh
```

5. Inspect node distribution:

```powershell
docker compose --profile tools exec toolbox /tools/cluster-status.sh
```

6. Run the mini stress test against the rate-limited demo tenant:

```powershell
docker compose --profile tools exec toolbox /tools/mini-ddos.sh
```

7. Open Grafana:

- `http://localhost:13000`
- user `admin`
- password `admin`

## 20 Service Lab

`/tools/provision-20-services.sh` creates:

- `tenant-01` .. `tenant-20`
- hosts `tenant-01.ha.local` .. `tenant-20.ha.local`
- upstreams pointing to `demo-app:80`
- a stricter rate-limit policy on `tenant-01`
- tuned anti-DDoS settings for repeated local tests

For enterprise smoke validation, scenario strength can be increased independently from base profile limits using synthetic load scaling (for example, 3x for `ha-lab`).

The provisioning flow persists anti-DDoS settings with:

- `X-WAF-Auto-Apply-Disabled: true`

and then performs one explicit revision compile/apply. That keeps bulk setup
predictable in HA mode instead of forcing a long synchronous apply on every
state mutation.

Use a Host header when you hit the runtime directly:

```powershell
curl.exe -H "Host: tenant-01.ha.local" http://localhost:18085/limited.html
```

## Failover Check

1. Watch the active node IDs:

```powershell
docker compose --profile tools exec toolbox /tools/cluster-status.sh
```

2. Stop one node:

```powershell
docker compose stop control-plane-a
```

3. Re-run `cluster-status.sh` and verify that requests continue through `control-plane-b`.
4. Start the node again:

```powershell
docker compose start control-plane-a
```

Because leader tasks use Redis locks, the surviving node keeps background automation moving without split-brain apply/compile activity.

## Resource Envelope

Approximate limits in this profile:

- total memory cap with `tools` and `observability`: about `8.0 GB`
- total CPU cap with `tools` and `observability`: about `7.75`

That stays well below your requested ceiling while still being large enough for two control-plane nodes, Postgres, Redis, runtime, UI, adaptive model, and tooling.

Recent validation results for this profile:

- `provision-20-services.sh` completed successfully with `Revision applied: rev-000064`
- `cluster-status.sh` showed alternating traffic between `control-plane-a` and `control-plane-b`
- `mini-ddos.sh` produced `429` responses on the rate-limited demo tenant while the stack stayed healthy
- rolling control-plane upgrade preserved API availability through `api-lb`
- Prometheus and Grafana came up with direct control-plane and runtime metrics

## Rolling Upgrade Validation

PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\upgrade\rolling-upgrade.ps1
```

Unix-like shells:

```sh
./upgrade/rolling-upgrade.sh
```

The helper rebuilds `control-plane-a` and `control-plane-b` one at a time while continuously probing `api-lb`. It fails if API availability drops during the rolling rebuild.

## Benchmark Pack

PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\benchmarks\run.ps1
```

Unix-like shells:

```sh
./benchmarks/run.sh
```

The benchmark pack saves timestamped JSON summaries under `benchmarks/results` and reports:

- status distribution;
- average latency;
- min/max latency;
- p50/p95/p99 latency;
- Prometheus target health when the observability profile is enabled.

## Useful Commands

Logs:

```powershell
docker compose logs -f api-lb control-plane-a control-plane-b runtime ddos-model
```

Status:

```powershell
docker compose ps
```

Inspect adaptive decisions:

```powershell
docker compose exec ddos-model sh -lc "cat /out/adaptive.json"
```

Stop:

```powershell
docker compose down --remove-orphans
```
