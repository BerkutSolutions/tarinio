$ErrorActionPreference = "Stop"

Write-Host "Scenario 1: First Deploy" -ForegroundColor Cyan

powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/preflight.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1

Push-Location deploy/lab-k8s-terraform/terraform
try {
  .\terraform.exe init
  .\terraform.exe apply -auto-approve -var-file=terraform.tfvars.example
}
finally {
  Pop-Location
}

powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-basic.ps1
Write-Host "Scenario 1 completed." -ForegroundColor Green
