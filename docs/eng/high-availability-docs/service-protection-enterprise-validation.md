# Service Protection Enterprise Validation

Last full smoke run: 2026-04-25 13:22:32 +03:00

This document records a reproducible service-protection validation pack for TARINIO 3.0.4.

## Scope

The run validates one integrated protection chain:

1. Hard access controls (allowlist, denylist, country policy) take priority and return 403.
2. Anti-Bot scanner auto-ban guard is present and enabled by default.
3. Sentinel / adaptive Anti-DDoS still detects and escalates burst scenarios.
4. Normal and trusted traffic remains free from adaptive false positives.
5. ha-lab is executed with a 3x synthetic load profile to make profile differences visible.

## 10-client matrix

| Client role | Pattern | Expected result |
| --- | --- | --- |
| normal_web | benign browser requests | no adaptive blocks |
| normal_mobile | benign mobile requests | no adaptive blocks |
| trusted_allowlist | trusted source pattern | no adaptive blocks |
| api_client | high-rate API burst | emergency single-source signal |
| country_blocked | blocked-country probe traffic | immediate 403 evidence |
| denylisted_ip | denylisted source traffic | immediate 403 evidence |
| scanner_a | scanner signatures | scanner suggestions/adaptive pressure |
| scanner_b | scanner signatures | scanner suggestions/adaptive pressure |
| hacker_payload | payload probes (xss/sqli/rce) | adaptive malicious entries |
| botnet_group | distributed flood | emergency botnet signal |

## Summary

| Profile | Load scale | Passed | Guards verified | Policy 403 evidence | Events | Adaptive entries | Actions | Scanner clients | Botnet unique IPs | Single-source emergency | Botnet emergency | Botnet adaptive entries | Total normal false positives |
| --- | --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| default | x1 | True | False | True | 1200 | 145 | drop: 145 | 4 | 480 | 0 | 144 | 144 | 0 |
| ha-lab | x3 | True | False | True | 3600 | 1000 | drop: 1000 | 8 | 1440 | 0 | 0 | 999 | 0 |

## Runtime Guard Expectations

Passing criteria require runtime config evidence for:

- country guard before antibot guard;
- allowlist guard presence;
- scanner autoban guard presence.

## Artifacts

Run artifacts are stored under:

.work/service-protection-smoke/20260425-132042

Per profile:

- result.json
- adaptive.json
- l7-suggestions.json
- model-state.json
- report.md

## Reproduce

PowerShell:

./scripts/smoke-service-protection-enterprise.ps1

Single profile:

./scripts/smoke-service-protection.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/service-protection-smoke/manual/default
./scripts/smoke-service-protection.ps1 -ComposeFile deploy/compose/ha-lab/docker-compose.yml -ProfileName ha-lab -OutputDir .work/service-protection-smoke/manual/ha-lab

