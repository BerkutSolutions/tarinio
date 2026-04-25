$ErrorActionPreference = "Stop"

Write-Host "Scenario 2: Config Update (OpenSearch + ClickHouse profile)" -ForegroundColor Cyan

powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-opensearch-clickhouse.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/configure-opensearch-clickhouse.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1

Write-Host "Scenario 2 completed." -ForegroundColor Green
