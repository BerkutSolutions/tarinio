$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultsDir = Join-Path $PSScriptRoot "results\$timestamp"
New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null

Push-Location $root
try {
  docker compose --profile tools --profile observability up -d --build
  docker compose exec -T toolbox /tools/provision-20-services.sh | Out-Null
  $summaryPath = Join-Path $resultsDir "summary.json"
  $stdout = docker compose exec -T toolbox /tools/benchmark-pack.sh
  Set-Content -Path $summaryPath -Value $stdout -NoNewline
  Write-Host "Benchmark summary saved to $summaryPath"
} finally {
  Pop-Location
}
