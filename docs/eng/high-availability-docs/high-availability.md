# High Availability and Multi-Node

This page belongs to the current documentation branch.

## Scope

TARINIO `current release` now supports a practical multi-node control-plane topology on one Docker host:

- shared PostgreSQL state;
- shared Redis coordination;
- two or more `control-plane` nodes behind an API load balancer;
- leader-elected background jobs;
- globally serialized revision compile/apply operations.

This closes the earlier gap where the product was documented as PostgreSQL-backed but effectively operated as a single-node JSON-state appliance without real coordination.

## What Is High Availability Today

The supported High Availability shape in `current release` is:

```text
ui -> api-lb -> control-plane-a / control-plane-b
                    |             |
                    +---- Redis ---+
                    +-- PostgreSQL-+
                    +-- shared runtime root / artifacts
                              |
                           runtime
                              |
                         protected sites
```

Related validation docs:

- [Sentinel Enterprise Validation](./sentinel-enterprise-validation.md)
- [Anti-Bot Enterprise Validation](./antibot-enterprise-validation.md)
- [Service Protection Enterprise Validation](./service-protection-enterprise-validation.md)

In this model:

- any control-plane node can serve API traffic;
- only one node at a time performs leader-only background work;
- compile/apply is protected by distributed locks;
- all control-plane nodes see the same persistent state in PostgreSQL.

## What Changed Technically

### Shared persistent state

Control-plane state is stored in PostgreSQL, not local JSON files, and legacy file state is migrated on first boot.

### Distributed coordination

Redis now provides:

- distributed locks for revision compile/apply;
- leader election for:
  - startup self-test;
  - auto-apply;
  - dev fast start;
  - TLS auto-renew.

### Runtime-safe reload and batched writes

The runtime reload path now signals the managed nginx master directly instead of
depending on an external pid file workflow. This removes reload failures during
High Availability apply operations where the runtime process is intentionally supervised by the
launcher.

Control-plane writes can also be batched by sending:

- `X-WAF-Auto-Apply-Disabled: true`

This persists the requested change without triggering an immediate compile/apply
side effect, which is useful for bulk provisioning before one explicit revision
compile/apply at the end.

### API node visibility

`GET /core-docs/api/app/meta` now returns:

- `ha_enabled`
- `ha_node_id`

This makes it easy to verify which node is serving traffic through the load balancer.

## Required Environment

At minimum, every control-plane node in the cluster must share:

- the same `POSTGRES_DSN`
- the same `CONTROL_PLANE_REDIS_ADDR`
- the same runtime API token
- the same runtime root or equivalent shared artifact surface

And each node must have its own:

- `CONTROL_PLANE_HA_NODE_ID`

Important High Availability variables:

- `CONTROL_PLANE_HA_ENABLED=true`
- `CONTROL_PLANE_HA_OPERATION_LOCK_TTL_SECONDS`
- `CONTROL_PLANE_HA_LEADER_LOCK_TTL_SECONDS`

## Operational Semantics

### Compile / apply

Only one node can hold the global operation lock at a time. This prevents:

- duplicate revision version allocation;
- concurrent writes into the runtime activation path;
- split-brain apply behavior.

### Auto-apply

Policy mutations can arrive on any node, but only the elected leader executes the follow-up auto-apply pipeline.

### Dev fast start

Bootstrap automation may be enabled on all nodes, but only the elected leader actually runs it.

### TLS auto-renew

Certificate renewal checks run under leader election so that only one node submits renewal jobs for shared certificate state.

## Resource-Limited High Availability Lab

Use:

- `deploy/compose/High Availability-lab`

This profile includes:

- `api-lb`
- `control-plane-a`
- `control-plane-b`
- `postgres`
- `redis`
- `ui`
- `runtime`
- `tarinio-sentinel`
- `demo-app`
- `toolbox`
- optional `prometheus`, `grafana`, `postgres-exporter`, `redis-exporter` through the `observability` profile

It is intentionally capped for workstation use and stays far below a `16 GB / 10 CPU` ceiling.

Validated High Availability lab resource envelope:

- total memory cap with `tools` and `observability`: about `8.0 GB`
- total CPU cap with `tools` and `observability`: about `7.75`

## Validation Workflow

1. Start the lab:

```powershell
cd deploy/compose/High Availability-lab
docker compose --profile tools --profile observability up -d --build
```

2. Open `http://localhost:18080/login`
3. Provision twenty demo services:

```powershell
docker compose --profile tools exec toolbox /tools/provision-20-services.sh
```

4. Check node distribution:

```powershell
docker compose --profile tools exec toolbox /tools/cluster-status.sh
```

5. Run the mini stress test:

```powershell
docker compose --profile tools exec toolbox /tools/mini-ddos.sh
```

6. Stop one control-plane node and confirm the cluster still serves API traffic through the remaining node.
7. Run the rolling upgrade validation:

```powershell
powershell -ExecutionPolicy Bypass -File .\upgrade\rolling-upgrade.ps1
```

8. Run the benchmark pack:

```powershell
powershell -ExecutionPolicy Bypass -File .\benchmarks\run.ps1
```

Validated on the bundled High Availability lab:

- 20 demo tenants provisioned successfully through `api-lb`
- load-balanced `/core-docs/api/app/meta` requests alternated between `control-plane-a` and `control-plane-b`
- the mini stress test produced a mixed `200` / `429` pattern on the rate-limited tenant
- rolling control-plane upgrades preserved API availability through `api-lb`
- Prometheus and Grafana started with direct metrics from both control-plane nodes and the runtime

## Current Boundaries

This is a serious High Availability improvement, but it is important to be precise:

- control-plane High Availability is implemented;
- shared persistent state and coordination are implemented;
- the provided lab validates failover and anti-DDoS behavior in a realistic local topology.

Still outside this release:

- multi-region orchestration;
- cross-host shared filesystem abstraction without operator-provided storage;
- full fleet management;
- SIEM-grade external pipelines and centralized data-lake integrations.

## Production Guidance

For production:

- place the API behind a stable load balancer;
- run at least two control-plane nodes;
- keep PostgreSQL and Redis on persistent storage;
- monitor lock contention, runtime health, and revision apply results;
- use the High Availability lab as the operator rehearsal environment before promotion.



