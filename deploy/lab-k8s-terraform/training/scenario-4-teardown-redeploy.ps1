$ErrorActionPreference = "Stop"

Write-Host "Scenario 4: Teardown and Re-Deploy" -ForegroundColor Cyan

Push-Location deploy/lab-k8s-terraform/terraform
try {
  .\terraform.exe init
  .\terraform.exe destroy -auto-approve -var-file=terraform.tfvars.example
  .\terraform.exe apply -auto-approve -var-file=terraform.tfvars.example
}
finally {
  Pop-Location
}

powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-basic.ps1
Write-Host "Scenario 4 completed." -ForegroundColor Green
