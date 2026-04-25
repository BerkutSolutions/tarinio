$ErrorActionPreference = "Stop"

$artifactsDir = "deploy/lab-k8s-terraform/ci/artifacts"
New-Item -ItemType Directory -Force -Path $artifactsDir | Out-Null

try {
  Write-Host "[1/4] Preflight" -ForegroundColor Cyan
  powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/preflight.ps1

  Write-Host "[2/4] Bootstrap + deploy" -ForegroundColor Cyan
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

  Write-Host "[3/4] Smoke + artifacts" -ForegroundColor Cyan
  powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1

  kubectl -n tarinio-lab get pods -o wide | Out-File -Encoding utf8 "$artifactsDir/pods.txt"
  kubectl -n tarinio-lab get svc -o wide | Out-File -Encoding utf8 "$artifactsDir/services.txt"
}
finally {
  Write-Host "[4/4] Teardown" -ForegroundColor Cyan
  Push-Location deploy/lab-k8s-terraform/terraform
  try {
    .\terraform.exe destroy -auto-approve -var-file=terraform.tfvars.example
  }
  finally {
    Pop-Location
  }
}

Write-Host "CI lab cycle completed." -ForegroundColor Green

