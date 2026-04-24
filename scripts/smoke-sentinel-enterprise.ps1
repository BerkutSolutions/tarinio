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

This page records the reproducible validation pack for tarinio-sentinel in TARINIO 3.0.0. The run starts the Docker Compose stacks for both the standalone default profile and the HA-ready ha-lab profile, injects controlled access-log evidence, and verifies that the model publishes bounded adaptive decisions without blocking benign traffic.

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
# Enterprise-проверка Sentinel

Последний полный smoke-прогон: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss zzz")

Эта страница фиксирует воспроизводимую проверку tarinio-sentinel в TARINIO 3.0.0. Прогон запускает Docker Compose стеки default и ha-lab, добавляет контролируемые строки access-log и проверяет, что модель выдает ограниченные adaptive-решения без блокировки нормального трафика.

## Итог

Покрыты нормальный трафик, scanner paths, brute-force, XSS, SQL injection, command injection, single-source DDoS, distributed DDoS и high-cardinality noise. Профиль считается успешным только если нормальный источник не попал в adaptive output, вредоносные сценарии дали evidence, а scanner paths попали в L7 suggestions до постоянного enforcement.

| Профиль | Passed | Events | Adaptive entries | Actions | L7 suggestions | Normal false positives |
| --- | --- | ---: | ---: | --- | ---: | ---: |
$($rows -join "`n")

## Матрица сценариев

| Сценарий | Паттерн | Ожидаемый сигнал |
| --- | --- | --- |
| Normal baseline | 20 безопасных запросов dashboard от одного источника | Нет adaptive entries и false positive block |
| Scanner discovery | /.env, /wp-admin, /phpmyadmin, /vendor/phpunit | Снижается trust и появляются L7 suggestions |
| Brute force | Повторные /login с 401, 403, 429 | Источник получает adaptive scrutiny и может эскалироваться |
| XSS / SQLi / RCE probes | Encoded script tag, UNION SELECT, shell metacharacters | Payload-источник повышает adaptive risk |
| Single-source DDoS | 140 запросов за секунду от одного IP | Срабатывает emergency single-source detection |
| Distributed DDoS | 240 запросов за секунду от многих IP | Срабатывает emergency botnet-like detection |
| High cardinality | Много уникальных path и источников | State остается ограниченным, publish output capped |

## False Positive

Нормальный источник проверяется отдельно от атакующих источников. Критерий приемки жесткий: normal_false_positive_entries должен быть 0 в каждом профиле.

## Артефакты

$RunDir

В каждом каталоге профиля лежат result.json, adaptive.json, l7-suggestions.json, model-state.json и report.md.

## Повторить прогон

./scripts/smoke-sentinel-enterprise.ps1

Это smoke и evidence test, а не замена внешнему нагрузочному тестированию. Он проверяет контур продукта: ingestion логов runtime, score calculation, explainable reasons, adaptive output compatibility, L7 suggestions, FP safety и запуск HA-профиля.
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
