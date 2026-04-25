$ErrorActionPreference = "Stop"

$overlay = "deploy/lab-k8s-terraform/k8s/profiles/guardrails"

Write-Host "Applying guardrails profile..." -ForegroundColor Cyan
kubectl apply -k $overlay

Write-Host "Verifying rollouts..." -ForegroundColor Cyan
kubectl -n tarinio-lab rollout status deploy/control-plane --timeout=240s
kubectl -n tarinio-lab rollout status deploy/runtime --timeout=240s
kubectl -n tarinio-lab rollout status deploy/ui --timeout=180s

Write-Host "Guardrails profile applied." -ForegroundColor Green

