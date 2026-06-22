param(
  [ValidateSet("low", "moderate", "high", "critical")]
  [string]$AuditLevel = "moderate",
  [switch]$AutoFix
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

function Invoke-NpmChecked {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Arguments,
    [switch]$AllowFailure
  )

  $output = & npm.cmd @Arguments 2>&1
  $exitCode = $LASTEXITCODE
  if (($exitCode -ne 0) -and (-not $AllowFailure)) {
    $tail = ($output | Select-Object -Last 80) -join [Environment]::NewLine
    throw "npm.cmd $($Arguments -join ' ') failed with exit code $exitCode`n$tail"
  }
  return [PSCustomObject]@{
    ExitCode = $exitCode
    Output   = $output
  }
}

function Get-SeverityRank([string]$Severity) {
  switch ($Severity) {
    "low" { return 1 }
    "moderate" { return 2 }
    "high" { return 3 }
    "critical" { return 4 }
    default { return 0 }
  }
}

function Get-RemainingVulnerabilities([pscustomobject]$AuditReport, [string]$MinimumSeverity) {
  $minRank = Get-SeverityRank $MinimumSeverity
  $items = @()
  foreach ($entry in $AuditReport.vulnerabilities.PSObject.Properties) {
    $item = $entry.Value
    if ((Get-SeverityRank $item.severity) -lt $minRank) {
      continue
    }
    $items += [PSCustomObject]@{
      Name         = $item.name
      Severity     = $item.severity
      FixAvailable = [bool]$item.fixAvailable
      Via          = @($item.via | ForEach-Object {
        if ($_ -is [string]) {
          $_
        } else {
          $_.title
        }
      })
    }
  }
  return $items | Sort-Object @{ Expression = { Get-SeverityRank $_.Severity }; Descending = $true }, Name
}

if ($AutoFix) {
  Write-Host "[RUN] npm audit fix --package-lock-only" -ForegroundColor DarkCyan
  $null = Invoke-NpmChecked -Arguments @("audit", "fix", "--package-lock-only", "--omit=dev", "--ignore-scripts", "--fund=false") -AllowFailure
}

Write-Host "[RUN] npm audit --json" -ForegroundColor DarkCyan
$auditResult = Invoke-NpmChecked -Arguments @("audit", "--omit=dev", "--json") -AllowFailure
$auditJson = ($auditResult.Output -join "`n").Trim()
if ([string]::IsNullOrWhiteSpace($auditJson)) {
  throw "npm audit returned empty output"
}

try {
  $report = $auditJson | ConvertFrom-Json
} catch {
  throw "failed to parse npm audit JSON: $($_.Exception.Message)"
}

$remaining = @(Get-RemainingVulnerabilities -AuditReport $report -MinimumSeverity $AuditLevel)
if ($remaining.Count -eq 0) {
  Write-Host "[OK] npm audit gate passed" -ForegroundColor Green
  exit 0
}

$summary = $remaining | ForEach-Object {
  $via = if ($_.Via.Count -gt 0) { ($_.Via | Select-Object -First 2) -join "; " } else { "no advisory details" }
  "$($_.Severity.ToUpperInvariant()) $($_.Name) (fixAvailable=$($_.FixAvailable)): $via"
}

$message = @(
  "npm security gate failed at audit level '$AuditLevel'.",
  "Remaining vulnerabilities: $($remaining.Count)",
  ($summary -join [Environment]::NewLine)
) -join [Environment]::NewLine

throw $message
