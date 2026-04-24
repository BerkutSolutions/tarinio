# Sentinel Enterprise Validation

Last full smoke run: 2026-04-24 17:31:33 +03:00

This page records the reproducible validation pack for tarinio-sentinel in TARINIO 3.0.0. The run starts the Docker Compose stacks for both the standalone default profile and the High Availability-ready High Availability-lab profile, injects controlled access-log evidence, and verifies that the model publishes bounded adaptive decisions without blocking benign traffic.

## Executive Summary

The validation covers normal traffic, scanner paths, brute-force behavior, XSS probes, SQL injection probes, command-injection probes, single-source DDoS, distributed DDoS, and high-cardinality noise. A profile passes only when normal traffic produces zero adaptive entries, malicious scenarios produce adaptive action evidence, and scanner paths are emitted as L7 suggestions before permanent enforcement.

| Profile | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| default | true | 856 | 141 | drop: 141 | 4 | 0 |
| High Availability-lab | true | 856 | 142 | drop: 142 | 4 | 0 |

## Scenario Matrix

| Scenario | Traffic Pattern | Expected Enterprise Signal |
| --- | --- | --- |
| Normal baseline | 20 benign dashboard requests from one source | No adaptive entries and no false positive block |
| Scanner discovery | Repeated /.env, /wp-admin, /phpmyadmin, /vendor/phpunit from scanner sources | Trust decreases and L7 suggestions are produced |
| Brute force | Repeated /login attempts with 401, 403, and 429 | Source receives adaptive scrutiny and can escalate |
| XSS / SQLi / RCE probes | Encoded script tag, UNION SELECT, and shell metacharacter probes | Payload source contributes to adaptive risk |
| Single-source DDoS | 140 requests in one second from one IP | Emergency single-source detection activates |
| Distributed DDoS | 240 requests in one second across many IPs | Emergency botnet-like detection activates |
| High cardinality | Many unique paths and sources | State remains bounded and publish output stays capped |

## False Positive Assessment

The benign source is tracked separately from attack sources. The acceptance threshold is strict: normal_false_positive_entries must be 0 in every profile. Both validated profiles reported 0 normal false positives.

## Evidence Location

Run artifacts are stored under:

.work/sentinel-smoke/20260424-172951

Each profile directory contains result.json, adaptive.json, l7-suggestions.json, model-state.json, and report.md.

## Reproduce

PowerShell:

./scripts/smoke-sentinel-enterprise.ps1

Single-profile examples:

./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/sentinel-smoke/manual/default

./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/High Availability-lab/docker-compose.yml -ProfileName High Availability-lab -OutputDir .work/sentinel-smoke/manual/High Availability-lab

## Notes For Reviewers

This is a smoke and evidence test, not a replacement for external load testing. It validates the product control loop: runtime log ingestion, score calculation, explainable reasons, adaptive output compatibility, L7 suggestion publication, false-positive safety, and High Availability profile startup.

