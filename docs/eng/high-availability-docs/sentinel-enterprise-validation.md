# Sentinel Enterprise Validation

Last full smoke validation update: 2026-04-26

This page records the enterprise smoke validation pack for `tarinio-sentinel` in TARINIO `3.0.5`.

## What Is Validated

Three anti-DDoS operating modes are tested:

1. `classic-only`: baseline L4 anti-DDoS only, adaptive model disabled.
2. `hybrid`: baseline anti-DDoS and adaptive model together.
3. `adaptive-only`: adaptive model drives escalation while baseline L4 profile remains active.

Each mode is validated for:

- `default`
- `ha-lab`

## Executive Test Result

| Profile | Mode | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives | Baseline conn/rate/burst |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | --- |
| default | classic-only | True | 1116 | 0 | none | 0 | 0 | 120/180/360 |
| default | hybrid | True | 1116 | 255 | drop: 255 | 4 | 0 | 120/180/360 |
| default | adaptive-only | True | 1116 | 262 | drop: 262 | 4 | 0 | 50000/20000/40000 |
| ha-lab | classic-only | True | 1116 | 0 | none | 0 | 0 | 120/240/480 |
| ha-lab | hybrid | True | 1116 | 2 | drop: 2 | 4 | 0 | 120/240/480 |
| ha-lab | adaptive-only | True | 1116 | 259 | drop: 259 | 4 | 0 | 120/240/480 |

## 30-actor Enterprise Matrix

| Actor group | Count | Mode relevance | Expected signal |
| --- | ---: | --- | --- |
| Normal users (`web`, `mobile`, `trusted`) | 10 | all modes | no adaptive false positives |
| Scanner actors | 8 | hybrid, adaptive-only | L7 suggestions and trust degradation |
| Brute-force + payload actors | 6 | hybrid, adaptive-only | adaptive escalation with explainable reasons |
| Flood actors (single + distributed) | 6+ synthetic botnet spread | all modes | baseline protection in classic, adaptive/emergency evidence in hybrid/adaptive-only |

Each profile/mode directory includes:

- `result.json`
- `adaptive.json`
- `l7-suggestions.json`
- `model-state.json`
- `report.md`

## Reproduce

PowerShell:

- `./scripts/smoke-sentinel-enterprise.ps1` (default + ha-lab)
- `./scripts/smoke-sentinel-enterprise.ps1 -SkipHaLab` (default only)
- `./scripts/smoke-sentinel-enterprise.ps1 -SkipDefault` (ha-lab only)
