# Adaptive DDoS Model (Independent Container)

## Why this exists

The adaptive model is an isolated sidecar service that:

- observes incoming request behavior per runtime (`8080` and `8081` separately),
- computes an IP risk score using a decay model,
- escalates mitigations from soft throttle to hard drop,
- writes adaptive rules for L4 guard.

If this model fails or is stopped, WAF runtime keeps serving traffic and enforcing base protections.  
This gives resilience: DDoS adaptation is independent, not a single point of failure for proxying.

## High-level flow

1. Each model container reads runtime `access.log` from a dedicated log volume:
   - `ddos-model-mgmt` -> management runtime (`https://localhost:8080`)
   - `ddos-model-app` -> test runtime (`https://localhost:8081`)
2. For each source IP, model updates score:
   - weight added for blocking statuses (`429`, `403`, `444`)
   - exponential decay between events
   - emergency detectors:
     - high requests/sec + high unique IP fanout (botnet-like burst)
     - very high per-IP requests/sec (single-source flood)
3. Stage is selected by score:
   - below threshold -> no adaptive action
   - above throttle threshold -> `throttle`
   - above drop threshold -> `drop`
4. Model writes `/out/adaptive.json` (shared volume).
5. Runtime launcher periodically re-runs `waf-runtime-l4-guard bootstrap`.
6. L4 guard merges static config with adaptive entries and applies `iptables` rules.

## Escalation logic

- First wave usually triggers `429` (L7 rate-limit signal).
- If client continues during active hold window, score grows and reaches:
  - `throttle` first (packet/connection trimming),
  - then `drop` (hard block).
- If traffic calms down, score decays and entry disappears after hold timeout.

This mirrors behavior: suspicious burst gets checked with partial limitation first, persistent attack gets fully cut.

## Fast emergency reaction (botnet / k6-style spikes)

For very heavy synthetic floods, model has fast-path detectors:

- `DDOS_MODEL_EMERGENCY_RPS` + `DDOS_MODEL_EMERGENCY_UNIQUE_IPS`
  - when both are exceeded in the same second, all active IPs in that second get emergency score boost.
- `DDOS_MODEL_EMERGENCY_PER_IP_RPS`
  - single-source flood detector for one IP.

Emergency score weights:

- `DDOS_MODEL_WEIGHT_EMERGENCY_BOTNET`
- `DDOS_MODEL_WEIGHT_EMERGENCY_SINGLE`

These weights can push an attacker into `drop` almost instantly, while normal users usually stay below thresholds.

## Per-service isolation

Adaptive model instances are split by runtime/log volume, so actions do not mix across services:

- attacks on `8081` do not auto-ban `8080`,
- attacks on `8080` do not auto-ban `8081`.

## Files and paths

- Model source:
  - `deploy/compose/testpage/ddos-model/main.go`
- Model image:
  - `deploy/compose/testpage/ddos-model/Dockerfile`
- Runtime adaptive input (mounted volume):
  - `/etc/waf/l4guard-adaptive/adaptive.json`
- Persisted model state:
  - `/state/model-state.json`

## Adaptive JSON format

Example:

```json
{
  "updated_at": "2026-04-02T15:30:00Z",
  "throttle_rate_per_second": 3,
  "throttle_burst": 6,
  "throttle_target": "REJECT",
  "entries": [
    {
      "ip": "172.18.0.1",
      "action": "throttle",
      "expires_at": "2026-04-02T15:31:00Z"
    },
    {
      "ip": "172.18.0.2",
      "action": "drop",
      "expires_at": "2026-04-02T15:31:15Z"
    }
  ]
}
```

## ENV tuning

Set in `deploy/compose/testpage/.env`:

- `WAF_L4_GUARD_REAPPLY_INTERVAL_SECONDS`  
  how often runtime re-applies L4 rules.

Model math/behavior:

- `DDOS_MODEL_POLL_INTERVAL_SECONDS`  
  log polling period.
- `DDOS_MODEL_DECAY_LAMBDA`  
  exponential decay coefficient for score.
- `DDOS_MODEL_THROTTLE_THRESHOLD`  
  score to enter `throttle`.
- `DDOS_MODEL_DROP_THRESHOLD`  
  score to enter `drop`.
- `DDOS_MODEL_HOLD_SECONDS`  
  ban/adaptive hold window.
- `DDOS_MODEL_THROTTLE_RATE_PER_SECOND`  
  per-IP rate for soft trimming.
- `DDOS_MODEL_THROTTLE_BURST`  
  burst allowance in throttle mode.
- `DDOS_MODEL_THROTTLE_TARGET` (`REJECT` or `DROP`)  
  target action for throttle overflow.
- `DDOS_MODEL_WEIGHT_429`
- `DDOS_MODEL_WEIGHT_403`
- `DDOS_MODEL_WEIGHT_444`  
  score contribution by status type.
- `DDOS_MODEL_EMERGENCY_RPS`
- `DDOS_MODEL_EMERGENCY_UNIQUE_IPS`
- `DDOS_MODEL_EMERGENCY_PER_IP_RPS`
- `DDOS_MODEL_WEIGHT_EMERGENCY_BOTNET`
- `DDOS_MODEL_WEIGHT_EMERGENCY_SINGLE`  
  fast-path emergency detection tuning.

## Operational checks

```powershell
cd deploy/compose/testpage
docker compose ps
docker compose logs --tail=120 ddos-model-mgmt ddos-model-app runtime runtime-test
docker compose exec ddos-model-mgmt sh -lc "cat /out/adaptive.json"
docker compose exec ddos-model-app sh -lc "cat /out/adaptive.json"
```

## Failure behavior

- If model crashes:
  - existing adaptive entries remain until replaced/expired;
  - base WAF protections continue.
- If adaptive file is missing/corrupt:
  - L4 guard ignores adaptive part and keeps static rules.
- If runtime cannot apply adaptive rules:
  - error is logged, next periodic cycle retries.
