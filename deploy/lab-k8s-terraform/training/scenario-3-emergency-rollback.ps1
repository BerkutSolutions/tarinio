$ErrorActionPreference = "Stop"

Write-Host "Scenario 3: Emergency Rollback to Base Profile" -ForegroundColor Cyan

$ns = "tarinio-lab"

# Roll back profile overlays to base by deleting profile-specific resources where safe.
kubectl -n $ns delete deploy redis --ignore-not-found=true
kubectl -n $ns delete svc redis --ignore-not-found=true
kubectl -n $ns delete deploy clickhouse --ignore-not-found=true
kubectl -n $ns delete svc clickhouse --ignore-not-found=true
kubectl -n $ns delete pvc clickhouse-data --ignore-not-found=true

# Re-apply base manifests and wait for stable baseline.
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-lab.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-basic.ps1

Write-Host "Scenario 3 completed (rollback to base baseline)." -ForegroundColor Green
