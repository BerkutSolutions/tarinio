$ErrorActionPreference = "Stop"

function Test-Tool {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$VersionArg
  )

  $cmd = Get-Command $Name -ErrorAction SilentlyContinue
  if (-not $cmd) {
    Write-Host ("[MISSING] {0}" -f $Name) -ForegroundColor Red
    return $false
  }

  try {
    $version = & $Name $VersionArg 2>$null | Select-Object -First 1
    if (-not $version) { $version = "<version not reported>" }
    Write-Host ("[OK] {0} => {1}" -f $Name, $version) -ForegroundColor Green
    return $true
  } catch {
    Write-Host ("[WARN] {0} found but version check failed: {1}" -f $Name, $_.Exception.Message) -ForegroundColor Yellow
    return $true
  }
}

Write-Host "== TARINIO lab-k8s-terraform preflight ==" -ForegroundColor Cyan

$required = @(
  @{ Name = "docker"; VersionArg = "version" },
  @{ Name = "kubectl"; VersionArg = "version" },
  @{ Name = "helm"; VersionArg = "version" },
  @{ Name = "terraform"; VersionArg = "version" }
)

$allOk = $true
foreach ($item in $required) {
  if (-not (Test-Tool -Name $item.Name -VersionArg $item.VersionArg)) {
    $allOk = $false
  }
}

$kindPresent = Test-Tool -Name "kind" -VersionArg "version"
$k3dPresent = Test-Tool -Name "k3d" -VersionArg "version"

if (-not $kindPresent -and -not $k3dPresent) {
  Write-Host "[MISSING] Need one Kubernetes local runtime: kind or k3d" -ForegroundColor Red
  $allOk = $false
}

if ($allOk) {
  Write-Host "Preflight passed." -ForegroundColor Green
  exit 0
}

Write-Host "Preflight failed. Install missing tools and retry." -ForegroundColor Red
exit 1

