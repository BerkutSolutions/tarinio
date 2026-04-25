param(
  [string]$ResultRoot = ".work/sentinel-smoke",
  [string]$DefaultCompose = "deploy/compose/default/docker-compose.yml",
  [string]$HaLabCompose = "deploy/compose/ha-lab/docker-compose.yml",
  [string]$WikiEnglishPath = "docs/eng/operators/sentinel-enterprise-validation.md",
  [string]$WikiRussianPath = "docs/ru/operators/sentinel-enterprise-validation.md",
  [switch]$SkipDefault,
  [switch]$SkipHaLab,
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

function Paths-Summary {
  param($Paths)
  $items = @($Paths | Where-Object { -not [string]::IsNullOrWhiteSpace([string]$_) })
  if ($items.Count -eq 0) {
    return "none"
  }
  return ($items -join ", ")
}

function Write-EnglishWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | $($result.passed) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.l7_suggestions) | $($result.normal_false_positive_entries) |")
  }

  $md = @"
# Sentinel Enterprise Validation

Last full smoke run: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

This page records the reproducible validation pack for tarinio-sentinel in TARINIO 3.0.4. The run starts the Docker Compose stacks for both the standalone default profile and the HA-ready ha-lab profile, injects controlled access-log evidence, and verifies that the model publishes bounded adaptive decisions without blocking benign traffic.

## Executive Summary

The validation covers normal traffic, scanner paths, brute-force behavior, XSS probes, SQL injection probes, command-injection probes, single-source DDoS, distributed DDoS, and high-cardinality noise. A profile passes only when normal traffic produces zero adaptive entries, malicious scenarios produce adaptive action evidence, and scanner paths are emitted as L7 suggestions before permanent enforcement.

| Profile | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
$($rows -join "`n")

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

The benign source is tracked separately from attack sources. The acceptance threshold is strict: normal_false_positive_entries must be 0 in every profile. This guards against an enterprise deployment accidentally throttling normal users during noisy background scans.

## Evidence Location

Run artifacts are stored under:

$RunDir

Each profile directory contains result.json, adaptive.json, l7-suggestions.json, model-state.json, and report.md.

## Reproduce

./scripts/smoke-sentinel-enterprise.ps1

For a single profile:

./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/sentinel-smoke/manual/default
./scripts/smoke-sentinel.ps1 -ComposeFile deploy/compose/ha-lab/docker-compose.yml -ProfileName ha-lab -OutputDir .work/sentinel-smoke/manual/ha-lab

## Notes For Reviewers

This is a smoke and evidence test, not a replacement for external load testing. It validates the product control loop: runtime log ingestion, score calculation, explainable reasons, adaptive output compatibility, L7 suggestion publication, false-positive safety, and HA profile startup.
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

function Write-RussianWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | $($result.passed) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.l7_suggestions) | $($result.normal_false_positive_entries) |")
  }

  $md = @"
# Enterprise-РїСЂРѕРІРµСЂРєР° Sentinel

РџРѕСЃР»РµРґРЅРёР№ РїРѕР»РЅС‹Р№ smoke-РїСЂРѕРіРѕРЅ: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

Р­С‚Р° СЃС‚СЂР°РЅРёС†Р° С„РёРєСЃРёСЂСѓРµС‚ РІРѕСЃРїСЂРѕРёР·РІРѕРґРёРјСѓСЋ РїСЂРѕРІРµСЂРєСѓ tarinio-sentinel РІ TARINIO 3.0.4. РџСЂРѕРіРѕРЅ Р·Р°РїСѓСЃРєР°РµС‚ Docker Compose СЃС‚РµРєРё default Рё ha-lab, РґРѕР±Р°РІР»СЏРµС‚ РєРѕРЅС‚СЂРѕР»РёСЂСѓРµРјС‹Рµ СЃС‚СЂРѕРєРё access-log Рё РїСЂРѕРІРµСЂСЏРµС‚, С‡С‚Рѕ РјРѕРґРµР»СЊ РІС‹РґР°РµС‚ РѕРіСЂР°РЅРёС‡РµРЅРЅС‹Рµ adaptive-СЂРµС€РµРЅРёСЏ Р±РµР· Р±Р»РѕРєРёСЂРѕРІРєРё РЅРѕСЂРјР°Р»СЊРЅРѕРіРѕ С‚СЂР°С„РёРєР°.

## РС‚РѕРі

РџРѕРєСЂС‹С‚С‹ РЅРѕСЂРјР°Р»СЊРЅС‹Р№ С‚СЂР°С„РёРє, scanner paths, brute-force, XSS, SQL injection, command injection, single-source DDoS, distributed DDoS Рё high-cardinality noise. РџСЂРѕС„РёР»СЊ СЃС‡РёС‚Р°РµС‚СЃСЏ СѓСЃРїРµС€РЅС‹Рј С‚РѕР»СЊРєРѕ РµСЃР»Рё РЅРѕСЂРјР°Р»СЊРЅС‹Р№ РёСЃС‚РѕС‡РЅРёРє РЅРµ РїРѕРїР°Р» РІ adaptive output, РІСЂРµРґРѕРЅРѕСЃРЅС‹Рµ СЃС†РµРЅР°СЂРёРё РґР°Р»Рё evidence, Р° scanner paths РїРѕРїР°Р»Рё РІ L7 suggestions РґРѕ РїРѕСЃС‚РѕСЏРЅРЅРѕРіРѕ enforcement.

| РџСЂРѕС„РёР»СЊ | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
$($rows -join "`n")

## РњР°С‚СЂРёС†Р° СЃС†РµРЅР°СЂРёРµРІ

| РЎС†РµРЅР°СЂРёР№ | РџР°С‚С‚РµСЂРЅ | РћР¶РёРґР°РµРјС‹Р№ СЃРёРіРЅР°Р» |
| --- | --- | --- |
| Normal baseline | 20 Р±РµР·РѕРїР°СЃРЅС‹С… Р·Р°РїСЂРѕСЃРѕРІ dashboard РѕС‚ РѕРґРЅРѕРіРѕ РёСЃС‚РѕС‡РЅРёРєР° | РќРµС‚ adaptive entries Рё false positive block |
| Scanner discovery | /.env, /wp-admin, /phpmyadmin, /vendor/phpunit | РЎРЅРёР¶Р°РµС‚СЃСЏ trust Рё РїРѕСЏРІР»СЏСЋС‚СЃСЏ L7 suggestions |
| Brute force | РџРѕРІС‚РѕСЂРЅС‹Рµ /login СЃ 401, 403, 429 | РСЃС‚РѕС‡РЅРёРє РїРѕР»СѓС‡Р°РµС‚ adaptive scrutiny Рё РјРѕР¶РµС‚ СЌСЃРєР°Р»РёСЂРѕРІР°С‚СЊСЃСЏ |
| XSS / SQLi / RCE probes | Encoded script tag, UNION SELECT, shell metacharacters | Payload-РёСЃС‚РѕС‡РЅРёРє РїРѕРІС‹С€Р°РµС‚ adaptive risk |
| Single-source DDoS | 140 Р·Р°РїСЂРѕСЃРѕРІ Р·Р° СЃРµРєСѓРЅРґСѓ РѕС‚ РѕРґРЅРѕРіРѕ IP | РЎСЂР°Р±Р°С‚С‹РІР°РµС‚ emergency single-source detection |
| Distributed DDoS | 240 Р·Р°РїСЂРѕСЃРѕРІ Р·Р° СЃРµРєСѓРЅРґСѓ РѕС‚ РјРЅРѕРіРёС… IP | РЎСЂР°Р±Р°С‚С‹РІР°РµС‚ emergency botnet-like detection |
| High cardinality | РњРЅРѕРіРѕ СѓРЅРёРєР°Р»СЊРЅС‹С… path Рё РёСЃС‚РѕС‡РЅРёРєРѕРІ | State РѕСЃС‚Р°РµС‚СЃСЏ РѕРіСЂР°РЅРёС‡РµРЅРЅС‹Рј, publish output capped |

## False Positive

РќРѕСЂРјР°Р»СЊРЅС‹Р№ РёСЃС‚РѕС‡РЅРёРє РїСЂРѕРІРµСЂСЏРµС‚СЃСЏ РѕС‚РґРµР»СЊРЅРѕ РѕС‚ Р°С‚Р°РєСѓСЋС‰РёС… РёСЃС‚РѕС‡РЅРёРєРѕРІ. РљСЂРёС‚РµСЂРёР№ РїСЂРёРµРјРєРё Р¶РµСЃС‚РєРёР№: normal_false_positive_entries РґРѕР»Р¶РµРЅ Р±С‹С‚СЊ 0 РІ РєР°Р¶РґРѕРј РїСЂРѕС„РёР»Рµ.

## РђСЂС‚РµС„Р°РєС‚С‹

$RunDir

Р’ РєР°Р¶РґРѕРј РєР°С‚Р°Р»РѕРіРµ РїСЂРѕС„РёР»СЏ Р»РµР¶Р°С‚ result.json, adaptive.json, l7-suggestions.json, model-state.json Рё report.md.

## РџРѕРІС‚РѕСЂРёС‚СЊ РїСЂРѕРіРѕРЅ

./scripts/smoke-sentinel-enterprise.ps1

Р­С‚Рѕ smoke Рё evidence test, Р° РЅРµ Р·Р°РјРµРЅР° РІРЅРµС€РЅРµРјСѓ РЅР°РіСЂСѓР·РѕС‡РЅРѕРјСѓ С‚РµСЃС‚РёСЂРѕРІР°РЅРёСЋ. РћРЅ РїСЂРѕРІРµСЂСЏРµС‚ РєРѕРЅС‚СѓСЂ РїСЂРѕРґСѓРєС‚Р°: ingestion Р»РѕРіРѕРІ runtime, score calculation, explainable reasons, adaptive output compatibility, L7 suggestions, FP safety Рё Р·Р°РїСѓСЃРє HA-РїСЂРѕС„РёР»СЏ.
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$runDir = Join-Path $ResultRoot $stamp
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$script = Join-Path $PSScriptRoot "smoke-sentinel.ps1"
$results = New-Object System.Collections.Generic.List[object]
$env:DDOS_MODEL_SUGGEST_MIN_HITS = "4"
$env:DDOS_MODEL_SUGGEST_MIN_UNIQUE_IPS = "2"
$env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf-smoke-no-runtime-profile"

if (-not $SkipDefault) {
  $defaultDir = Join-Path $runDir "default"
  & $script -ComposeFile $DefaultCompose -ProfileName "default" -OutputDir $defaultDir -WaitSeconds $WaitSeconds
  $results.Add((Read-Result (Join-Path $defaultDir "result.json")))
}

if (-not $SkipHaLab) {
  $haDir = Join-Path $runDir "ha-lab"
  & $script -ComposeFile $HaLabCompose -ProfileName "ha-lab" -OutputDir $haDir -WaitSeconds $WaitSeconds
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
  throw "sentinel enterprise smoke failed"
}

