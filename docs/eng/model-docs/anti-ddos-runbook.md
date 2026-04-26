# Anti-DDoS Operator Runbook

This document describes safe operation of the `Anti-DDoS` section in TARINIO.

The section controls three protection layers:

- baseline L4 protection at the `iptables` level;
- a global L7 rate-limit override that is compiled into revisions;
- `tarinio-sentinel`, an adaptive engine that analyzes runtime access logs and publishes temporary throttle/drop rules plus L7 suggestions.

Source of truth:

- control-plane storage: `anti_ddos_settings.json`;
- revision snapshot field: `anti_ddos_settings`;
- runtime state: artifacts of the currently active revision after `compile -> validate -> apply`.

## What The Operator Changes

On `/anti-ddos`, the operator configures:

- `use_l4_guard`, `chain_mode`, `conn_limit`, `rate_per_second`, `rate_burst`, `ports`, `target`;
- `enforce_l7_rate_limit`, `l7_requests_per_second`, `l7_burst`, `l7_status_code`;
- adaptive model values such as poll interval, throttle/drop thresholds, hold time, signal weights, and emergency thresholds.
- per-service model opt-in on the `Front service` tab through `adaptive_model_enabled`.

Every change to these parameters is a configuration change and should flow into a new revision.

## Safe Workflow

1. Change the parameters in `Anti-DDoS`.
2. Compile a new revision in `Revisions`.
3. Apply the revision.
4. Verify:
   - `GET /healthz` and `/healthcheck`;
   - revision status and recent rollout events;
   - absence of false blocking on legitimate traffic;
   - `security_rate_limit`, `security_access`, and `security_waf` events.

## Recommended Starting Values

For the first production-like activation, use a conservative profile:

- `conn_limit`: `200`
- `rate_per_second`: `100`
- `rate_burst`: `200`
- `ports`: `80,443`
- `target`: `DROP`
- `enforce_l7_rate_limit`: `true`
- `l7_requests_per_second`: `100`
- `l7_burst`: `200`
- `l7_status_code`: `429`

Typical adaptive model baseline:

- `model_poll_interval_seconds`: `2`
- `model_decay_lambda`: `0.08`
- `model_throttle_threshold`: `2.5`
- `model_drop_threshold`: `6.0`
- `model_hold_seconds`: `60`
- `model_throttle_rate_per_second`: `3`
- `model_throttle_burst`: `6`
- `model_throttle_target`: `REJECT`

## Safe Tuning Practice

- Tighten one block at a time.
- Do not reduce `conn_limit`, `rate_per_second`, and `l7_requests_per_second` all at once.
- Build a baseline of normal traffic before enabling stricter emergency thresholds.
- For the management site (`control-plane-access`), make sure you are not creating self-lockout conditions.

## Symptoms And Actions

If legitimate traffic starts getting `429`:

- inspect the L7 override values;
- increase `l7_burst`;
- verify trusted proxies and real client IP visibility;
- confirm that the source is the Anti-DDoS layer rather than a local per-site rate-limit.

If new connections fail before reaching NGINX:

- inspect the L4 guard chain placement;
- inspect `DOCKER-USER` versus `INPUT`;
- verify `destination_ip` in Docker bridge / DNAT scenarios;
- confirm that `ports` matches the real runtime exposure.

If the adaptive model starts throttling or dropping too aggressively:

- inspect fresh access-log lines and security events;
- review emergency burst detection;
- increase `model_drop_threshold` and/or reduce `403/444` weights;
- re-apply a known-good revision if the degradation is substantial.

If L7 suggestions appear:

- keep new candidates in `suggested` until the path is clearly abusive;
- move candidates to `shadow` to collect would-block counters without blocking;
- only convert a pattern into a permanent WAF/access rule after reviewing false-positive risk.

## Sentinel Operations

In compose profiles the adaptive engine runs as `tarinio-sentinel`.

Useful checks:

```sh
docker compose ps tarinio-sentinel
docker compose logs --tail=120 tarinio-sentinel
docker compose exec tarinio-sentinel sh -lc "cat /out/adaptive.json"
docker compose exec tarinio-sentinel sh -lc "cat /out/l7-suggestions.json"
```

Expected resource envelope:

- default profile: about `0.75 CPU` and `512MB`;
- High Availability lab: about `0.75 CPU` and `512MB`.

If `adaptive.json` is empty, confirm that the global model is enabled and at least one front service has `adaptive_model_enabled` enabled.

## Rollback

If the configuration proves too aggressive:

1. Open `Revisions`.
2. Find the previous known-good revision.
3. Run `apply`.
4. Wait for successful apply and `/healthz` recovery.
5. Return to `Anti-DDoS`, relax the parameters, and repeat compile/apply.

Rollback should always be revision-based. Manual runtime filesystem edits or ad hoc `iptables` changes outside the revision flow are not considered a normal operating mode.

## Related Documents

- `docs/eng/model-docs/anti-ddos-model.md`
- `docs/eng/model-docs/tarinio-sentinel.md`
- `docs/eng/model-docs/runtime-l4-guard.md`
- `docs/eng/core-docs/runbook.md`



