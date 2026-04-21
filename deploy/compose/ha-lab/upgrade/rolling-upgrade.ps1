$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

function Wait-Healthy {
  param(
    [Parameter(Mandatory = $true)][string]$Container,
    [int]$Attempts = 60
  )

  for ($i = 0; $i -lt $Attempts; $i++) {
    $status = docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' $Container 2>$null
    if ($status -eq "healthy" -or $status -eq "running") {
      return
    }
    Start-Sleep -Seconds 2
  }
  throw "Container $Container did not become healthy"
}

function Test-Probe {
  docker compose exec -T toolbox sh -lc 'curl -fsS http://api-lb:8080/healthz >/dev/null'
}

function Invoke-RollingStep {
  param(
    [Parameter(Mandatory = $true)][string]$Service,
    [Parameter(Mandatory = $true)][string]$Container
  )

  $probeFile = Join-Path $env:TEMP ("tarinio-ha-probe-" + [guid]::NewGuid().ToString() + ".log")
  $job = Start-Job -ScriptBlock {
    param($repoRoot, $logFile)
    Set-Location $repoRoot
    while ($true) {
      try {
        docker compose exec -T toolbox sh -lc 'curl -fsS http://api-lb:8080/healthz >/dev/null' | Out-Null
        Add-Content -Path $logFile -Value "ok"
      } catch {
        Add-Content -Path $logFile -Value "fail"
      }
      Start-Sleep -Seconds 1
    }
  } -ArgumentList $root, $probeFile

  try {
    docker compose up -d --build --no-deps $Service | Out-Null
    Wait-Healthy -Container $Container
  } finally {
    Stop-Job $job | Out-Null
    Receive-Job $job | Out-Null
    Remove-Job $job | Out-Null
  }

  $failures = 0
  if (Test-Path $probeFile) {
    $failures = (Get-Content $probeFile | Where-Object { $_ -eq "fail" }).Count
    Remove-Item $probeFile -Force
  }
  if ($failures -gt 0) {
    throw "Zero-downtime check failed for ${Service}: $failures API probe failures observed"
  }
  Write-Host "Rolling upgrade step passed: $Service"
}

try {
  docker compose --profile tools up -d toolbox | Out-Null
  Wait-Healthy -Container tarinio-ha-control-plane-a
  Wait-Healthy -Container tarinio-ha-control-plane-b

  Invoke-RollingStep -Service control-plane-a -Container tarinio-ha-control-plane-a
  Invoke-RollingStep -Service control-plane-b -Container tarinio-ha-control-plane-b

  Write-Host "Rolling control-plane upgrade passed without API downtime."
} finally {
  Pop-Location
}
