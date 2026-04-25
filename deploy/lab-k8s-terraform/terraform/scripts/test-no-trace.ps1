param(
  [string]$VarFile = "terraform.tfvars.example"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path "deploy/lab-k8s-terraform/terraform"
$tf = Join-Path $root "terraform.exe"
if (-not (Test-Path $tf)) {
  throw "terraform.exe not found in $root"
}

$planFile = Join-Path $root "tfplan.bin"
$tfData = Join-Path $root ".terraform"
$lockFile = Join-Path $root ".terraform.lock.hcl"

function Invoke-Terraform {
  param(
    [Parameter(Mandatory = $true)][string[]]$Args
  )

  & $tf @Args
  if ($LASTEXITCODE -ne 0) {
    throw ("terraform command failed: terraform {0}" -f ($Args -join " "))
  }
}

try {
  Push-Location $root
  Invoke-Terraform -Args @("init")
  Invoke-Terraform -Args @("validate")
  Invoke-Terraform -Args @("plan", "-var-file=$VarFile", "-out=$planFile")
  Write-Host "No-trace terraform test passed." -ForegroundColor Green
}
finally {
  Pop-Location
  if (Test-Path $planFile) { Remove-Item -LiteralPath $planFile -Force -ErrorAction SilentlyContinue }
  if (Test-Path $tfData) { Remove-Item -LiteralPath $tfData -Recurse -Force -ErrorAction SilentlyContinue }
  if (Test-Path $lockFile) { Remove-Item -LiteralPath $lockFile -Force -ErrorAction SilentlyContinue }
}

