param(
  [string]$Namespace = "tarinio-lab",
  [switch]$SkipRuntimeProfileConfig
)

$ErrorActionPreference = "Stop"

$profileManifest = "deploy/lab-k8s-terraform/k8s/profiles/opensearch-clickhouse"

Write-Host "Applying OpenSearch+ClickHouse k8s profile..." -ForegroundColor Cyan
kubectl apply -k $profileManifest

kubectl -n $Namespace rollout status deploy/clickhouse --timeout=300s
kubectl -n $Namespace rollout status deploy/opensearch --timeout=300s
kubectl -n $Namespace rollout status deploy/control-plane --timeout=300s
kubectl -n $Namespace rollout status deploy/runtime --timeout=300s

if (-not $SkipRuntimeProfileConfig) {
  powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/configure-opensearch-clickhouse.ps1 -Namespace $Namespace
}

Write-Host "OpenSearch+ClickHouse profile apply completed." -ForegroundColor Green

