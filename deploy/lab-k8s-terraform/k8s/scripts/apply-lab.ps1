$ErrorActionPreference = "Stop"

$manifestRoot = "deploy/lab-k8s-terraform/k8s/manifests"
$secretPath = Join-Path $manifestRoot "02-secrets.yaml"

if (-not (Test-Path $secretPath)) {
  throw "Missing $secretPath"
}

if ((Get-Content -Raw -Encoding utf8 $secretPath) -match "CHANGE_ME") {
  throw "Secret file still contains CHANGE_ME placeholders. Run init-secrets.ps1 or update secrets manually."
}

Write-Host "Applying TARINIO k8s lab manifests..." -ForegroundColor Cyan
kubectl apply -k $manifestRoot

Write-Host "Waiting for workloads..." -ForegroundColor Cyan
kubectl -n tarinio-lab rollout status deploy/postgres --timeout=240s
kubectl -n tarinio-lab rollout status deploy/opensearch --timeout=300s
kubectl -n tarinio-lab rollout status deploy/control-plane --timeout=300s
kubectl -n tarinio-lab rollout status deploy/runtime --timeout=300s
kubectl -n tarinio-lab rollout status deploy/ui --timeout=180s

Write-Host "Kubernetes lab apply completed." -ForegroundColor Green

