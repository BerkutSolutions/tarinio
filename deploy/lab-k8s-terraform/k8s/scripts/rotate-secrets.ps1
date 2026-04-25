$ErrorActionPreference = "Stop"

param(
  [switch]$ApplyAfterRotate = $true
)

$secretPath = "deploy/lab-k8s-terraform/k8s/manifests/02-secrets.yaml"
if (-not (Test-Path $secretPath)) {
  throw "Missing $secretPath. Run init-secrets.ps1 first."
}

Write-Host "Rotating lab secrets..." -ForegroundColor Cyan
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1

if ($ApplyAfterRotate) {
  Write-Host "Applying rotated secrets and restarting workloads..." -ForegroundColor Cyan
  kubectl apply -f $secretPath -n tarinio-lab
  kubectl -n tarinio-lab rollout restart deploy/control-plane
  kubectl -n tarinio-lab rollout restart deploy/runtime
  kubectl -n tarinio-lab rollout status deploy/control-plane --timeout=240s
  kubectl -n tarinio-lab rollout status deploy/runtime --timeout=240s
}

Write-Host "Secret rotation completed." -ForegroundColor Green

