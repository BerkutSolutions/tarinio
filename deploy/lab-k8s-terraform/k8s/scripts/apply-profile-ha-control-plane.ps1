$ErrorActionPreference = "Stop"

$overlay = "deploy/lab-k8s-terraform/k8s/profiles/ha-control-plane"

Write-Host "Applying HA control-plane profile..." -ForegroundColor Cyan
kubectl apply -k $overlay

Write-Host "Waiting for Redis and control-plane rollouts..." -ForegroundColor Cyan
kubectl -n tarinio-lab rollout status deploy/redis --timeout=180s
kubectl -n tarinio-lab rollout status deploy/control-plane --timeout=300s

Write-Host "HA profile applied." -ForegroundColor Green

