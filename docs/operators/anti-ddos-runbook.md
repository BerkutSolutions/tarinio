# Anti-DDoS Runbook

This runbook describes safe tuning and rollback for the `Anti-DDoS` section (`/anti-ddos`).

## Scope

The current Anti-DDoS workflow controls:
- L4 guard (`l4guard/config.json` in revision bundle)
- optional global L7 rate-limit override (applied to all enabled sites during compile)

Settings source of truth:
- control-plane state: `anti_ddos_settings.json`
- revision snapshot field: `anti_ddos_settings`
- runtime apply target: compiled revision artifacts after `compile + apply`

## Safe Defaults

Default profile is conservative-high to avoid false bans on first rollout:
- `conn_limit`: `200`
- `rate_per_second`: `100`
- `rate_burst`: `200`
- `ports`: `80,443`
- `target`: `DROP`
- `enforce_l7_rate_limit`: `false`
- `l7_requests_per_second`: `100`
- `l7_burst`: `200`
- `l7_status_code`: `429`

## Recommended Change Procedure

1. Save changes in `/anti-ddos`.
2. Create a new revision (`/revisions` -> compile).
3. Apply the revision.
4. Verify:
   - recent apply status is `succeeded`;
   - runtime remains healthy (`readyz`);
   - expected traffic is not blocked unexpectedly.

## Safe Ranges

For production-like ramp-up:
- `conn_limit`: start from `200-500`, reduce gradually.
- `rate_per_second`: start from `100-300`.
- `rate_burst`: keep at least `2x rate_per_second`.
- L7 global override: enable only after baseline traffic profile is understood.

Avoid:
- lowering all limits at once;
- enabling strict L7 override during unknown peak windows.

## Rollback

If change is too aggressive:
1. Open `/revisions`.
2. Re-apply the previous known-good revision.
3. Confirm `apply succeeded` and health restored.
4. Return to `/anti-ddos`, relax values, then re-run compile/apply.

Rollback is revision-based and does not require manual runtime file edits.



