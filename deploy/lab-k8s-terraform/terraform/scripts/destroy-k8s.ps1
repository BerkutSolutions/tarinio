$ErrorActionPreference = "Stop"

param(
  [Parameter(Mandatory = $true)][string]$ManifestDir,
  [string]$Context = ""
)

if (-not [string]::IsNullOrWhiteSpace($Context)) {
  kubectl config use-context $Context | Out-Null
}

if (Test-Path $ManifestDir) {
  kubectl delete -k $ManifestDir --ignore-not-found=true
}

Write-Host "k8s lab destroy completed." -ForegroundColor Green
