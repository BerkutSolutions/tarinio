param(
  [string]$ResultRoot = ".work/service-protection-smoke",
  [string]$DefaultCompose = "deploy/compose/default/docker-compose.yml",
  [string]$HaLabCompose = "deploy/compose/ha-lab/docker-compose.yml",
  [string]$WikiEnglishPath = "docs/eng/high-availability-docs/service-protection-enterprise-validation.md",
  [string]$WikiRussianPath = "docs/ru/high-availability-docs/service-protection-enterprise-validation.md",
  [switch]$SkipDefault,
  [switch]$SkipHaLab,
  [int]$DefaultLoadScale = 1,
  [int]$HaLabLoadScale = 3,
  [int]$WaitSeconds = 12
)

$ErrorActionPreference = "Stop"

function Read-Result {
  param([string]$Path)
  return (Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json)
}

function Action-Summary {
  param($Actions)
  if ($null -eq $Actions) {
    return "none"
  }
  $parts = New-Object System.Collections.Generic.List[string]
  foreach ($property in $Actions.PSObject.Properties) {
    $parts.Add("$($property.Name): $($property.Value)")
  }
  if ($parts.Count -eq 0) {
    return "none"
  }
  return ($parts -join ", ")
}

function Write-EnglishWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | x$($result.load_scale) | $($result.passed) | $($result.guards_verified) | $($result.policy_403_evidence) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.scenario_shape.scanner_clients) | $($result.scenario_shape.botnet_unique_ips) | $($result.scenario_counts.single_source_emergency_entries) | $($result.scenario_counts.botnet_emergency_entries) | $($result.scenario_counts.botnet_actor_entries) | $($result.false_positive.normal_web + $result.false_positive.normal_mobile + $result.false_positive.trusted_allowlist) |")
  }

  $md = @"
# Service Protection Enterprise Validation

Last full smoke run: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

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
$($rows -join "`n")

## Runtime Guard Expectations

Passing criteria require runtime config evidence for:

- country guard before antibot guard;
- allowlist guard presence;
- scanner autoban guard presence.

## Artifacts

Run artifacts are stored under:

$RunDir

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
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

function Write-RussianWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | x$($result.load_scale) | $($result.passed) | $($result.guards_verified) | $($result.policy_403_evidence) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.scenario_shape.scanner_clients) | $($result.scenario_shape.botnet_unique_ips) | $($result.scenario_counts.single_source_emergency_entries) | $($result.scenario_counts.botnet_emergency_entries) | $($result.scenario_counts.botnet_actor_entries) | $($result.false_positive.normal_web + $result.false_positive.normal_mobile + $result.false_positive.trusted_allowlist) |")
  }

  $md = @"
# Enterprise service protection validation (RU)

Last full smoke run: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

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
$($rows -join "`n")

## Runtime Guard Expectations

Passing criteria require runtime config evidence for:

- country guard before antibot guard;
- allowlist guard presence;
- scanner autoban guard presence.

## Artifacts

Run artifacts are stored under:

$RunDir

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
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$runDir = Join-Path $ResultRoot $stamp
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$script = Join-Path $PSScriptRoot "smoke-service-protection.ps1"
$results = New-Object System.Collections.Generic.List[object]
$env:DDOS_MODEL_SUGGEST_MIN_HITS = "4"
$env:DDOS_MODEL_SUGGEST_MIN_UNIQUE_IPS = "2"
$env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf-smoke-no-runtime-profile"

if (-not $SkipDefault) {
  $defaultDir = Join-Path $runDir "default"
  & $script -ComposeFile $DefaultCompose -ProfileName "default" -OutputDir $defaultDir -WaitSeconds $WaitSeconds -LoadScale $DefaultLoadScale
  $results.Add((Read-Result (Join-Path $defaultDir "result.json")))
}

if (-not $SkipHaLab) {
  $haDir = Join-Path $runDir "ha-lab"
  & $script -ComposeFile $HaLabCompose -ProfileName "ha-lab" -OutputDir $haDir -WaitSeconds $WaitSeconds -LoadScale $HaLabLoadScale
  $results.Add((Read-Result (Join-Path $haDir "result.json")))
}

Write-EnglishWiki $WikiEnglishPath $runDir $results
Write-RussianWiki $WikiRussianPath $runDir $results

$summary = [ordered]@{
  generated_at = (Get-Date).ToUniversalTime().ToString("o")
  run_dir = $runDir
  profiles = @($results | ForEach-Object { $_.profile })
  passed = (@($results | Where-Object { -not $_.passed }).Count -eq 0)
  wiki = @($WikiEnglishPath, $WikiRussianPath)
}

$summary | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $runDir "summary.json") -Encoding UTF8
$summary | ConvertTo-Json -Depth 6

if (-not $summary.passed) {
  throw "service protection enterprise smoke failed"
}
