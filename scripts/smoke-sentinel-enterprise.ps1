param(
  [string]$ResultRoot = ".work/sentinel-smoke",
  [string]$DefaultCompose = "deploy/compose/default/docker-compose.yml",
  [string]$HaLabCompose = "deploy/compose/ha-lab/docker-compose.yml",
  [string]$WikiEnglishPath = "docs/eng/high-availability-docs/sentinel-enterprise-validation.md",
  [string]$WikiRussianPath = "docs/ru/high-availability-docs/sentinel-enterprise-validation.md",
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
  if ($null -eq $Actions) { return "none" }
  $parts = New-Object System.Collections.Generic.List[string]
  foreach ($property in $Actions.PSObject.Properties) {
    $parts.Add("$($property.Name): $($property.Value)")
  }
  if ($parts.Count -eq 0) { return "none" }
  return ($parts -join ", ")
}

function Write-EnglishWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | $($result.mode) | $($result.passed) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.l7_suggestions) | $($result.normal_false_positive_entries) | $($result.l4_baseline.conn_limit)/$($result.l4_baseline.rate_per_second)/$($result.l4_baseline.rate_burst) |")
  }

  $md = @"
# Sentinel Enterprise Validation

Last full smoke run: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

This page records the reproducible enterprise validation pack for tarinio-sentinel in TARINIO 3.0.5.

## What Is Validated

Three anti-DDoS operating modes are tested:

1. classic-only: baseline L4 anti-DDoS only, adaptive model disabled.
2. hybrid: baseline anti-DDoS and adaptive model together.
3. adaptive-only: adaptive model becomes primary escalation path while baseline L4 profile remains active.

Each mode is executed in:

- default profile
- high-availability-lab profile

## Executive Test Result

| Profile | Mode | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives | Baseline conn/rate/burst |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | --- |
$($rows -join "`n")

## 30-actor Enterprise Matrix

| Actor group | Count | Mode relevance | Expected signal |
| --- | ---: | --- | --- |
| Normal users (web + mobile + trusted) | 10 | all modes | no adaptive false positives |
| Scanner actors | 8 | hybrid, adaptive-only | L7 suggestions and trust degradation |
| Brute-force + payload actors | 6 | hybrid, adaptive-only | adaptive escalation with explainable reasons |
| Flood actors (single + distributed) | 6+ synthetic botnet spread | all modes | baseline protection in classic, emergency/adaptive escalation in hybrid/adaptive-only |

## Evidence Location

$RunDir

Each profile/mode directory includes:

- result.json
- adaptive.json
- l7-suggestions.json
- model-state.json
- report.md

## Reproduce

./scripts/smoke-sentinel-enterprise.ps1
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

function Write-RussianWiki {
  param([string]$Path, [string]$RunDir, $Results)

  $rows = New-Object System.Collections.Generic.List[string]
  foreach ($result in $Results) {
    $rows.Add("| $($result.profile) | $($result.mode) | $($result.passed) | $($result.injected_events) | $($result.adaptive_entries) | $(Action-Summary $result.adaptive_actions) | $($result.l7_suggestions) | $($result.normal_false_positive_entries) | $($result.l4_baseline.conn_limit)/$($result.l4_baseline.rate_per_second)/$($result.l4_baseline.rate_burst) |")
  }

  $md = @'
# Enterprise-proverka Sentinel

Posledniy smoke run: {RUN_DATE}

Dokument fiksiruet vosproizvodimyy enterprise validation pack dlya `tarinio-sentinel` v TARINIO `3.0.5`.

## Chto proveryaetsya

Testiruyutsya tri rezhima Anti-DDoS:

1. `classic-only`: tolko bazovyy L4 Anti-DDoS, adaptive model otklyuchena.
2. `hybrid`: bazovyy Anti-DDoS i adaptive model vmeste.
3. `adaptive-only`: osnova eskalatsii cherez adaptive model pri aktivnom baseline L4 profile.

## Itogovye rezultaty

| Profil | Rezhim | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives | Bazovyy conn/rate/burst |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | --- |
{ROWS}

## Matritsa na 30 aktorov

| Gruppa aktorov | Kol-vo | Relevatnost po rezhimam | Ozhidaemyy signal |
| --- | ---: | --- | --- |
| Obychnye polzovateli (`web`, `mobile`, `trusted`) | 10 | vse rezhimy | net adaptive false positive |
| Scanner-istochniki | 8 | `hybrid`, `adaptive-only` | poyavlyayutsya L7 suggestions i snizhaetsya trust |
| Brute-force + payload istochniki | 6 | `hybrid`, `adaptive-only` | adaptive-eskalatsiya s explainable-prichinami |
| Flood-istochniki (`single` + `distributed`) | 6+ synthetic spread | vse rezhimy | baseline zashchita v `classic`, emergency/adaptive eskalatsiya v `hybrid`/`adaptive-only` |

## Artefakty

{RUN_DIR}

## Kak vosproizvesti

`./scripts/smoke-sentinel-enterprise.ps1`
'@
  $md = $md.Replace("{RUN_DATE}", (Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz"))
  $md = $md.Replace("{ROWS}", ($rows -join "`n"))
  $md = $md.Replace("{RUN_DIR}", $RunDir)
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

function Invoke-ModeRun {
  param(
    [string]$ScriptPath,
    [string]$ComposeFile,
    [string]$ProfileName,
    [string]$Mode,
    [string]$OutputDir,
    [int]$WaitSeconds
  )

  $originalModelEnabled = $env:WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED
  $originalSentinelModelEnabled = $env:DDOS_MODEL_ENABLED
  $originalRuntimeRoot = $env:DDOS_MODEL_RUNTIME_ROOT
  $originalConnLimit = $env:WAF_DEFAULT_ANTIDDOS_CONN_LIMIT
  $originalRate = $env:WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND
  $originalBurst = $env:WAF_DEFAULT_ANTIDDOS_RATE_BURST

  try {
    switch ($Mode) {
      "classic-only" {
        $env:WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED = "false"
        $env:DDOS_MODEL_ENABLED = "false"
        $env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf-disabled"
        $env:WAF_DEFAULT_ANTIDDOS_CONN_LIMIT = "120"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND = "180"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_BURST = "360"
      }
      "adaptive-only" {
        $env:WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED = "true"
        $env:DDOS_MODEL_ENABLED = "true"
        $env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf-disabled"
        $env:WAF_DEFAULT_ANTIDDOS_CONN_LIMIT = "50000"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND = "20000"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_BURST = "40000"
      }
      default {
        $env:WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED = "true"
        $env:DDOS_MODEL_ENABLED = "true"
        $env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf"
        $env:WAF_DEFAULT_ANTIDDOS_CONN_LIMIT = "120"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND = "180"
        $env:WAF_DEFAULT_ANTIDDOS_RATE_BURST = "360"
      }
    }

    & $ScriptPath -ComposeFile $ComposeFile -ProfileName $ProfileName -Mode $Mode -OutputDir $OutputDir -WaitSeconds $WaitSeconds
  } finally {
    $env:WAF_DEFAULT_ANTIDDOS_MODEL_ENABLED = $originalModelEnabled
    $env:DDOS_MODEL_ENABLED = $originalSentinelModelEnabled
    $env:DDOS_MODEL_RUNTIME_ROOT = $originalRuntimeRoot
    $env:WAF_DEFAULT_ANTIDDOS_CONN_LIMIT = $originalConnLimit
    $env:WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND = $originalRate
    $env:WAF_DEFAULT_ANTIDDOS_RATE_BURST = $originalBurst
  }
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$runDir = Join-Path $ResultRoot $stamp
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$script = Join-Path $PSScriptRoot "smoke-sentinel.ps1"
$results = New-Object System.Collections.Generic.List[object]
$modes = @("classic-only", "hybrid", "adaptive-only")

$env:DDOS_MODEL_SUGGEST_MIN_HITS = "4"
$env:DDOS_MODEL_SUGGEST_MIN_UNIQUE_IPS = "2"
$env:DDOS_MODEL_RUNTIME_ROOT = "/var/lib/waf"

if (-not $SkipDefault) {
  foreach ($mode in $modes) {
    $defaultDir = Join-Path $runDir ("default-" + $mode)
    Invoke-ModeRun -ScriptPath $script -ComposeFile $DefaultCompose -ProfileName "default" -Mode $mode -OutputDir $defaultDir -WaitSeconds $WaitSeconds
    $results.Add((Read-Result (Join-Path $defaultDir "result.json")))
  }
}

if (-not $SkipHaLab) {
  foreach ($mode in $modes) {
    $haDir = Join-Path $runDir ("ha-lab-" + $mode)
    Invoke-ModeRun -ScriptPath $script -ComposeFile $HaLabCompose -ProfileName "ha-lab" -Mode $mode -OutputDir $haDir -WaitSeconds $WaitSeconds
    $results.Add((Read-Result (Join-Path $haDir "result.json")))
  }
}

Write-EnglishWiki -Path $WikiEnglishPath -RunDir $runDir -Results $results
Write-RussianWiki -Path $WikiRussianPath -RunDir $runDir -Results $results

$summary = [ordered]@{
  generated_at = (Get-Date).ToUniversalTime().ToString("o")
  run_dir = $runDir
  profiles = @($results | ForEach-Object { $_.profile } | Sort-Object -Unique)
  modes = @($results | ForEach-Object { $_.mode } | Sort-Object -Unique)
  passed = (@($results | Where-Object { -not $_.passed }).Count -eq 0)
  wiki = @($WikiEnglishPath, $WikiRussianPath)
}

$summary | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $runDir "summary.json") -Encoding UTF8
$summary | ConvertTo-Json -Depth 6

if (-not $summary.passed) {
  throw "sentinel enterprise smoke failed"
}
