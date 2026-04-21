# Benchmark Pack

This page belongs to the current documentation branch.

## Scope

The public benchmark pack in `deploy/compose/ha-lab` is designed to produce reproducible local numbers without exhausting a workstation.

It focuses on three operator questions:

- how quickly the stack responds under normal traffic;
- how it behaves on protected endpoints when limits engage;
- whether HA control-plane and observability remain healthy during the run.

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
cd deploy/compose/ha-lab
./benchmarks/run.ps1
```

Unix-like shells:

```sh
cd deploy/compose/ha-lab
./benchmarks/run.sh
```

Each run creates a timestamped directory under `deploy/compose/ha-lab/benchmarks/results` and stores `summary.json`.

## Interpretation

Healthy behavior usually looks like this:

- the baseline scenario stays mostly `200`;
- the protected scenario shows a controlled mix of `200` and `429`;
- API health remains stable;
- Prometheus reports targets as `up`.

That means the system is protecting itself by degrading safely instead of failing open or crashing.

## Resource Envelope

The HA lab keeps the full benchmark environment below the workstation budget documented in `deploy/compose/ha-lab/README.md`, including Prometheus and Grafana.
