# Benchmark Pack

This page belongs to the current documentation branch.

## Scope

The public benchmark pack in `deploy/compose/High Availability-lab` is designed to produce reproducible local numbers without exhausting a workstation.

It focuses on three operator questions:

- how quickly the stack responds under normal traffic;
- how it behaves on protected endpoints when limits engage;
- whether High Availability control-plane and observability remain healthy during the run.

## Included Scenarios

The benchmark pack runs:

- baseline public traffic against a normal tenant page;
- a rate-limited scenario against `tenant-01`;
- API health checks against `api-lb`;
- an optional Prometheus query to confirm observability availability.

## Output

`benchmark-http.sh` emits JSON with:

- total request count;
- status code distribution;
- average latency;
- minimum and maximum latency;
- p50, p95, and p99 latency.

`benchmark-pack.sh` combines the scenario outputs into one JSON document.

## How To Run

PowerShell:

```powershell
cd deploy/compose/High Availability-lab
./core-docs/benchmarks/run.ps1
```

Unix-like shells:

```sh
cd deploy/compose/High Availability-lab
./core-docs/benchmarks/run.sh
```

Each run creates a timestamped directory under `deploy/compose/High Availability-lab/core-docs/benchmarks/results` and stores `summary.json`.

## Interpretation

Healthy behavior usually looks like this:

- the baseline scenario stays mostly `200`;
- the protected scenario shows a controlled mix of `200` and `429`;
- API health remains stable;
- Prometheus reports targets as `up`.

That means the system is protecting itself by degrading safely instead of failing open or crashing.

## Resource Envelope

The High Availability lab keeps the full benchmark environment below the workstation budget documented in `deploy/compose/High Availability-lab/README.md`, including Prometheus and Grafana.

## Sentinel Enterprise Test Matrix

TARINIO 3.0.1 adds a focused Smart WAF validation pack for `tarinio-sentinel`. The pack is intentionally CPU-only and can be replayed with generated access-log lines before running destructive network tests.

Scenarios to cover:

- normal traffic: steady `2xx/3xx`, low unique path count, expected result is no adaptive entries;
- scanner: repeated `/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`, expected result is lower trust score and L7 suggestions;
- brute force: repeated authentication paths with `401/403/429`, expected result is `watch` or `throttle` before any `drop`;
- single-source flood: high RPS from one IP, expected result is bounded `throttle/drop`;
- distributed flood: high global RPS with many IPs, expected result is emergency scoring without publishing more than `MODEL_MAX_PUBLISHED_ENTRIES`;
- high cardinality: many IPs and paths, expected result is stable memory and eviction by `MODEL_MAX_ACTIVE_IPS`.

Current repository verification for this slice:

```sh
go test ./internal/sentinel/...
go test ./control-plane/internal/antiddossuggestions ./control-plane/internal/services ./control-plane/internal/handlers
go test ./core-docs/deploy/compose/default/ddos-model ./core-docs/deploy/compose/auto-start/ddos-model ./core-docs/deploy/compose/testpage/ddos-model ./cmd/tarinio-sentinel
go test ./core-docs/ui/tests
```

Acceptance criteria:

- no false positive entries for the normal traffic scenario;
- scanner paths become suggestions before enforcement;
- `drop` and `temp_ban` require multi-signal evidence unless emergency detection is active;
- adaptive output remains backward compatible for `l4guard`;
- `tarinio-sentinel` remains within the documented container limits.


