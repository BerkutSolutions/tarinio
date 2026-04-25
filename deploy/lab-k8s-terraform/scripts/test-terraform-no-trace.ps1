param(
  [string]$TerraformRoot = "deploy/lab-k8s-terraform/terraform",
  [string]$VarFile = "terraform.tfvars.example"
)

$ErrorActionPreference = "Stop"

function Invoke-Terraform {
  param(
    [Parameter(Mandatory = $true)]
    [string]$TerraformExe,
    [Parameter(Mandatory = $true)]
    [string[]]$Args
  )

  & $TerraformExe @Args
  if ($LASTEXITCODE -ne 0) {
    throw ("terraform command failed: terraform {0}" -f ($Args -join " "))
  }
}

$sourceRoot = Resolve-Path $TerraformRoot
$terraformExe = Join-Path $sourceRoot "terraform.exe"
if (-not (Test-Path $terraformExe)) {
  throw "terraform.exe not found in $sourceRoot"
}

$requiredFiles = @("versions.tf", "variables.tf", "main.tf", "outputs.tf", $VarFile)
foreach ($file in $requiredFiles) {
  $path = Join-Path $sourceRoot $file
  if (-not (Test-Path $path)) {
    throw "required terraform file not found: $path"
  }
}

$workDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tarinio-tf-no-trace-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $workDir -Force | Out-Null

try {
  foreach ($file in $requiredFiles) {
    Copy-Item -LiteralPath (Join-Path $sourceRoot $file) -Destination (Join-Path $workDir $file) -Force
  }
  Copy-Item -LiteralPath $terraformExe -Destination (Join-Path $workDir "terraform.exe") -Force

  Push-Location $workDir
  $tf = Join-Path $workDir "terraform.exe"
  Invoke-Terraform -TerraformExe $tf -Args @("init")
  Invoke-Terraform -TerraformExe $tf -Args @("validate")
  Invoke-Terraform -TerraformExe $tf -Args @("plan", "-var-file=$VarFile", "-out=tfplan.bin")
  Pop-Location
} finally {
  if (Test-Path $workDir) {
    Remove-Item -LiteralPath $workDir -Recurse -Force -ErrorAction SilentlyContinue
  }
}

$repoArtifacts = @(
  (Join-Path $sourceRoot ".terraform"),
  (Join-Path $sourceRoot ".terraform.lock.hcl"),
  (Join-Path $sourceRoot "tfplan.bin")
)
foreach ($artifact in $repoArtifacts) {
  if (Test-Path $artifact) {
    throw "no-trace test left artifact in repository: $artifact"
  }
}

Write-Host "Terraform no-trace preflight passed." -ForegroundColor Green

