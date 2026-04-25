$ErrorActionPreference = "Stop"

param(
  [string]$Context = ""
)

if (-not [string]::IsNullOrWhiteSpace($Context)) {
  kubectl config use-context $Context | Out-Null
}

kubectl cluster-info | Out-Null
Write-Host "kubectl context is ready." -ForegroundColor Green
