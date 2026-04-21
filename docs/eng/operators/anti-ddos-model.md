# Anti-DDoS Model Architecture And Principles

This document explains how the TARINIO anti-DDoS control loop works, which layers it contains, which signals it uses, and how it makes decisions.

## 1. General Idea

Anti-DDoS in TARINIO is not a single magic toggle. It is a cascade of three related but independent layers:

1. L4 guard
   Blocks or limits harmful TCP traffic before it reaches NGINX, TLS handshake, or ModSecurity.
2. Global L7 rate-limit override
   Compiles shared HTTP request limits into the active revision artifacts.
3. Adaptive model
   Analyzes runtime access logs, accumulates score per IP, and publishes temporary throttle/drop rules in `adaptive.json`, which are then enforced by the L4 guard.

These layers do not replace one another:

- L4 guard protects sockets and CPU before HTTP parsing;
- the L7 override keeps baseline HTTP pressure under control;
- the adaptive model reacts to source behavior and escalates dynamically.

## 2. Source Of Truth And Runtime Path

Source of truth:

- control-plane storage: `anti_ddos_settings.json`

The settings then follow this path:

- operator changes settings in the control-plane;
- compile creates a snapshot;
- validate checks the bundle and runtime contract;
- apply publishes the revision as active;
- runtime reads the artifacts of the active revision.

At minimum, the bundle includes:

- `l4guard/config.json` for baseline L4 protection;
- `ddos-model/config.json` for adaptive model configuration.

Runtime does not read the domain model directly. It operates only on compiled artifacts of the active revision.

## 3. L4 Guard

L4 guard runs before NGINX startup and applies deterministic `iptables` rules.

Baseline logic:

- limit concurrent TCP connections per source IP via `connlimit`;
- limit the rate of new TCP connections via `hashlimit`;
- enforce `DROP` or `REJECT` on violations.

This protects against:

- high-concurrency floods;
- high-rate connection floods;
- keep-alive abuse where one source holds too many open sockets.

L4 guard attaches to a parent chain:

- `DOCKER-USER` when available and appropriate for Docker-aware deployments;
- otherwise `INPUT`.

In Docker bridge / DNAT scenarios, `destination_ip` is used so that rules target the correct post-DNAT runtime address.

## 4. Global L7 Override

When `enforce_l7_rate_limit` is enabled, the compiler injects a global HTTP rate-limit override for all enabled sites except the management UI.

Purpose:

- prevent a single source or narrow source set from sustaining high HTTP pressure;
- provide a predictable baseline load ceiling before adaptive escalation kicks in.

Main parameters:

- `l7_requests_per_second`
- `l7_burst`
- `l7_status_code`

This layer is static for the lifetime of the active revision. It does not learn or adapt on its own until a new revision is created.

## 5. Adaptive Model Data Sources

The adaptive model runs as a separate `ddos-model` process.

It reads:

- runtime access log;
- current model state from `model-state.json`;
- the active revision model config from `ddos-model/config.json`.

It writes:

- `adaptive.json` containing temporary IP-based actions.

The L4 guard consumes `adaptive.json` and extends baseline firewall behavior with temporary throttle/drop rules.

## 6. Signals Used By The Model

The model does not react to generic noise. It reacts to concrete indicators of pressure or abusive behavior.

Base HTTP-status signals:

- `429`: weighted by `model_weight_429`, typically indicating rate pressure;
- `403`: weighted by `model_weight_403`, often reflecting rejected or forbidden access;
- `444`: weighted by `model_weight_444`, often indicating clearly bad traffic or early runtime close behavior.

Additional emergency signals:

- global RPS exceeding `model_emergency_rps`;
- unique IP count in the same second exceeding `model_emergency_unique_ips`;
- per-IP RPS exceeding `model_emergency_per_ip_rps`.

## 7. Score Calculation

For each IP, the model stores:

- current `score`;
- last event timestamp;
- last update timestamp;
- current stage;
- `expires_at`.

When a signal arrives:

- the corresponding status weight is added to the IP score;
- exponential decay is applied first using `model_decay_lambda`;
- quiet behavior over time naturally reduces previous score.

This means:

- the model does not remember random spikes forever;
- score decays naturally over time;
- escalation happens only with repeated or sufficiently intense signals.

## 8. Thresholds And Stages

The model uses two major stages:

- `throttle` when `score >= model_throttle_threshold`;
- `drop` when `score >= model_drop_threshold`.

If the score is below the throttle threshold:

- the IP does not appear in adaptive firewall output.

If the score crosses the throttle threshold:

- the IP is published as `throttle` in `adaptive.json`.

If the score crosses the drop threshold:

- the IP is published as `drop` in `adaptive.json`.

The `drop` threshold should always be higher than the `throttle` threshold.

## 9. Emergency Detectors

### 9.1 Botnet-Like Burst Detector

The model aggregates:

- requests per second;
- unique IP count within the same second.

When both the total RPS and unique-IP thresholds are crossed in the same second:

- score is increased for all IPs participating in the burst;
- in extreme cases, immediate drop can be applied.

This helps against distributed noise where a single IP looks harmless in isolation but the group pattern is clearly DDoS-like.

### 9.2 Single-Source Flood Detector

Per-IP RPS is tracked by second.

If one IP exceeds the emergency per-IP threshold:

- the model escalates immediately;
- at strong enough intensity, it can emit an immediate drop instead of waiting for gradual score accumulation.

This helps with intense single-source floods.

## 10. Why Adaptive Output Is Aggregated Across Sites

The model can track score by `site + ip`, but publication into `adaptive.json` is aggregated by IP.

An IP is promoted into adaptive output when:

- it has a global marker; or
- it shows abusive behavior on at least two sites.

This is intentional:

- to avoid banning an IP based on a single local false positive;
- to treat cross-site noise as a stronger sign of automation or attack activity.

## 11. What Goes Into `adaptive.json`

Adaptive output contains:

- `updated_at`;
- `throttle_rate_per_second`;
- `throttle_burst`;
- `throttle_target`;
- `entries[]`, where each IP has an `action` and `expires_at`.

Possible actions:

- `throttle`
- `drop`

The L4 guard then turns those entries into concrete `iptables` rules.

## 12. How The L4 Guard Uses Adaptive Entries

If the entry is `drop`:

- a direct firewall rule is installed for that IP.

If the entry is `throttle`:

- a hashlimit-based rule is created using `model_throttle_rate_per_second`, `model_throttle_burst`, and `model_throttle_target`.

The adaptive model does not apply firewall rules itself. It only publishes decisions. Enforcement is done by the L4 guard.

## 13. Hold Time And Expiry

After escalation, an IP gets `expires_at` based on `model_hold_seconds`.

This means:

- protection does not disappear immediately when traffic drops;
- runtime gets a short hold period to avoid oscillation loops during active attacks.

After the hold expires and score has decayed, the entry disappears from model state and from adaptive output.

## 14. Why The Model Respects Operator Ceilings

Effective emergency thresholds take operator-configured ceilings into account:

- `rate_per_second`;
- `l7_requests_per_second`;
- `conn_limit`.

This prevents the model from:

- entering emergency mode too early;
- conflicting with deliberate operator policy;
- drifting away from the chosen control posture.

## 15. Practical Meaning Of Key Parameters

`model_decay_lambda`

- controls how quickly old events are forgotten;
- higher value means faster decay and shorter model memory.

`model_throttle_threshold`

- soft escalation threshold;
- too low a value causes false throttling.

`model_drop_threshold`

- hard escalation threshold;
- too low a value causes aggressive blocks.

`model_weight_429`

- weight of rate-pressure signals;
- useful for gradual accumulation.

`model_weight_403` and `model_weight_444`

- stronger signals that often indicate more clearly abusive behavior.

`model_emergency_rps`, `model_emergency_unique_ips`, `model_emergency_per_ip_rps`

- emergency safeguards for situations where waiting for score buildup would be unsafe.

## 16. Operator Mental Model

Operators should think of the adaptive model as a local automatic escalation layer:

- baseline protection is established by L4 + L7;
- the model reacts to abnormal behavior;
- the revision defines the operating rules;
- runtime applies temporary measures while the attack is ongoing.

This is not opaque machine learning. It is a deterministic scoring system with explicit thresholds, weights, and TTL behavior.

## 17. What The Model Does Not Do

The model does not:

- replace WAF policy tuning;
- analyze payload semantics like a content-aware detector;
- make operator decisions on behalf of humans;
- modify control-plane state by itself;
- rewrite the active revision.

It only:

- reads runtime telemetry;
- recalculates IP scores;
- writes adaptive firewall output.

## 18. Monitoring And Debugging

Use the following to observe behavior:

- `/healthz` and `/healthcheck`;
- `Revisions` and rollout timeline;
- logs and activity views;
- security events and archived requests;
- `adaptive.json` when low-level runtime debugging is needed.

Healthy signs:

- `adaptive.json` updates regularly;
- abusive IPs receive `throttle` or `drop` stages;
- entries disappear after decay and hold expiry;
- false blocks do not grow uncontrollably.

## 19. Recommended Adoption Path

1. Enable an L4 baseline and a soft L7 baseline first.
2. Keep adaptive thresholds conservative at the beginning.
3. Observe real `429`, `403`, `444`, and emergency bursts.
4. Only then lower thresholds or increase weights.

## 20. Related Documents

- `docs/eng/operators/anti-ddos-runbook.md`
- `docs/eng/operators/runtime-l4-guard.md`
- `docs/eng/operators/waf-tuning-guide.md`
