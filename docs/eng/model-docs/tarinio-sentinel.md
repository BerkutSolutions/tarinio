# TARINIO Sentinel

`tarinio-sentinel` is the adaptive security engine for TARINIO current release. It replaces the old `ddos-model` runtime process while keeping the legacy `ddos-model/config.json` artifact and `MODEL_*` environment contract compatible for existing deployments.

## Runtime Role

The service is intentionally small and CPU-only:

- it tails runtime access logs through the `file` source backend;
- it keeps per-IP state in `/state/model-state.json`;
- it publishes L4 decisions to `/out/adaptive.json`;
- it publishes L7 rule candidates to `/out/l7-suggestions.json`;
- it reads the active revision profile from `/var/lib/waf/active/current.json` and `ddos-model/config.json`.

The request path never calls an LLM or a heavy model. Runtime only consumes the published adaptive files.

## Source Modes

`MODEL_SOURCE_BACKEND=file` is the production MVP mode. It reads `/logs/access.log`, persists the byte offset, and survives container restarts through the `/state` volume.

`MODEL_SOURCE_BACKEND=redis` is reserved for High Availability stream mode. The interface exists, but stream ingestion is not enabled yet; High Availability deployments still run file-tail mode against the shared runtime logs.

## Per-Site Enablement

The global Anti-DDoS switch still lives in `/anti-ddos` as `model_enabled`. Each front service also has `front_service.adaptive_model_enabled`.

At revision compile time TARINIO writes:

- `model_enabled: false` when the global model is disabled or no site opted in;
- `model_enabled_sites` with the enabled site IDs and hostnames when at least one service opted in.

`control-plane-access` is enabled by default. Newly created or imported services are disabled by default so operators can opt in deliberately.

## Scoring

Sentinel uses lightweight statistics:

- status signals: `429`, `403`, `444`;
- request rate per IP;
- `404` ratio;
- scanner path hits such as `/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit`;
- unique path count per IP;
- suspicious user-agent score;
- emergency burst detectors for single-source and distributed floods.

The state stores:

- `risk_score`: higher means more suspicious;
- `trust_score`: `100 - risk_score`;
- `reason_codes`: top reasons for the current decision;
- `top_signals`: weighted signal contributions.

Scores decay over time with `MODEL_DECAY_LAMBDA`, so clients can recover trust after the behavior stops.

## Decision Engine

The action ladder is:

- `watch`;
- `throttle`;
- `drop`;
- `temp_ban`.

Aggressive actions require multiple strong signals unless an emergency signal is present. Cooldown prevents frequent de-escalation/escalation oscillation, and `MODEL_MAX_ACTIONS_PER_MINUTE` limits the amount of changed published decisions.

## Adaptive Output

`adaptive.json` remains backward compatible for `l4guard`. New fields such as `score`, `trust_score`, `source`, `first_seen`, `last_seen`, `reason_codes`, and `top_signals` are optional additions.

Publishing is batched:

- write only when content changes;
- write no more often than `MODEL_PUBLISH_INTERVAL_SECONDS`;
- publish only top-N entries through `MODEL_MAX_PUBLISHED_ENTRIES`.

This keeps `l4guard` from receiving thousands of churned rules during high-cardinality traffic.

## L7 Suggestions

Sentinel detects hot scanner paths and writes L7 candidates. Control-plane stores them through:

- `GET /core-docs/api/anti-ddos/rule-suggestions`;
- `POST|PUT /core-docs/api/anti-ddos/rule-suggestions`;
- `POST|PUT /core-docs/api/anti-ddos/rule-suggestions/{id}/status`.

The lifecycle statuses are:

- `suggested`: visible candidate, no runtime blocking;
- `shadow`: would-block counters are collected, still no runtime blocking;
- `temporary`: runtime candidate is enabled with TTL and rollback safety;
- `permanent`: candidate reached promotion thresholds and requires final operator review/approval.

Current release keeps permanent enforcement conservative: there is no unconditional automatic permanent apply.

## False Positive Safety

The safety model is conservative:

- per-site opt-in is required for non-management services;
- `drop` and `temp_ban` require multiple signals;
- suggestions start as non-blocking candidates;
- shadow mode records would-block metrics without enforcement;
- cardinality limits bound state and published entries;
- rollback is revision-based through the normal control-plane flow.

## Resource Envelope

Recommended container limits:

- default profile: `0.75 CPU`, `512MB`;
- enterprise profile: `0.75 CPU`, `512MB`;
- High Availability lab: `0.75 CPU`, `512MB`;
- minimum practical range: `0.25-0.75 CPU`, `128-512MB`.

Main bottlenecks are log volume, active IP cardinality, and path diversity. Keep `MODEL_MAX_ACTIVE_IPS`, `MODEL_MAX_PUBLISHED_ENTRIES`, and `MODEL_MAX_UNIQUE_PATHS_PER_IP` set in production.




