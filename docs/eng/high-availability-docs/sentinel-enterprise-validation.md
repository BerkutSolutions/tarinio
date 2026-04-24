# Sentinel Enterprise Validation

Last full smoke run: 2026-04-24 17:31:33 +03:00

This page records the reproducible enterprise validation pack for `tarinio-sentinel` in TARINIO `3.0.1`.

## What Is Validated

The validation checks the full Sentinel decision loop:

1. Access-log ingestion from runtime events.
2. Multi-signal scoring and trust updates.
3. Explainable adaptive output (`reason_codes`, `top_signals`).
4. Bounded publish behavior (`max_published_entries`, publish interval).
5. L7 suggestion generation for scanner patterns.
6. False-positive safety for normal traffic.

It is executed against both supported profiles:

- `default`
- `high-availability-lab`

## Executive Test Result

| Profile | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| default | true | 856 | 141 | drop: 141 | 4 | 0 |
| high-availability-lab | true | 856 | 142 | drop: 142 | 4 | 0 |

## Enterprise Traffic Matrix

| Scenario | Traffic Pattern | Expected Enterprise Signal |
| --- | --- | --- |
| Normal baseline | Benign dashboard/API traffic | No adaptive blocks for normal users |
| Scanner discovery | `/.env`, `/wp-admin`, `/phpmyadmin`, `/vendor/phpunit` | Trust decreases, L7 suggestions appear |
| Brute force | Repeated `/login` attempts | Escalating adaptive score for attacker source |
| XSS / SQLi / RCE probes | Encoded script, UNION payloads, shell metacharacters | Risk score rises with explainable reasons |
| Single-source flood | One IP burst | Emergency single-source path activates |
| Distributed flood | Multi-IP burst | Emergency botnet-like path activates |
| High cardinality noise | Many unique paths and sources | State remains bounded, output remains capped |

## Separate Anti-Bot Validation

Anti-bot validation is now tracked in a dedicated enterprise document:

- [Anti-Bot Enterprise Validation](./antibot-enterprise-validation.md)

## Final Enterprise Conclusion

`tarinio-sentinel` passed the reproducible enterprise validation criteria for both standalone and High Availability-ready profiles:

- adaptive detection is active and explainable,
- benign traffic remains clean (`0` normal false positives),
- scanner behavior is surfaced via L7 suggestions before permanent policy apply,
- anti-bot gate behavior is validated through the dedicated anti-bot enterprise pack.

For enterprise procurement and architecture review, this result supports the statement: this is the expected product behavior for controlled security automation in production-like environments.

## Evidence Location

Run artifacts are stored under:

`.work/sentinel-smoke/20260424-172951`

Each profile directory includes:

- `result.json`
- `adaptive.json`
- `l7-suggestions.json`
- `model-state.json`
- `report.md`

## Reproduce

PowerShell:

`./scripts/smoke-sentinel-enterprise.ps1`

Single-profile examples:

`./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/sentinel-smoke/manual/default`

`./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/ha-lab/docker-compose.yml -ProfileName ha-lab -OutputDir .work/sentinel-smoke/manual/ha-lab`

