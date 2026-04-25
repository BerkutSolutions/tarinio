param(
  [string]$K8sRoot = "deploy/lab-k8s-terraform/k8s"
)

$ErrorActionPreference = "Stop"

function Invoke-Kubectl {
  param(
    [Parameter(Mandatory = $true)]
    [string[]]$Args
  )

  & kubectl @Args
  if ($LASTEXITCODE -ne 0) {
    throw ("kubectl command failed: kubectl {0}" -f ($Args -join " "))
  }
}

function Invoke-KustomizeBuild {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  $built = & kubectl kustomize $Path 2>&1
  if ($LASTEXITCODE -ne 0) {
    throw ("kubectl kustomize failed for {0}: {1}" -f $Path, ($built -join "`n"))
  }
  if ([string]::IsNullOrWhiteSpace(($built -join "`n"))) {
    throw "kubectl kustomize produced empty output for $Path"
  }
}

$kubectl = Get-Command "kubectl" -ErrorAction SilentlyContinue
if (-not $kubectl) {
  throw "kubectl not found in PATH"
}

$k8sPath = Resolve-Path $K8sRoot
$manifestsPath = Join-Path $k8sPath "manifests"
if (-not (Test-Path (Join-Path $manifestsPath "kustomization.yaml"))) {
  throw "kustomization.yaml not found in $manifestsPath"
}

$beforeContext = ""
$afterContext = ""
try {
  $beforeContext = (& kubectl config current-context 2>$null | Select-Object -First 1)
} catch {
  $beforeContext = ""
}

Invoke-Kubectl -Args @("version", "--client")
Invoke-KustomizeBuild -Path $manifestsPath

$profilesPath = Join-Path $k8sPath "profiles"
if (Test-Path $profilesPath) {
  $profileDirs = Get-ChildItem -LiteralPath $profilesPath -Directory
  foreach ($profileDir in $profileDirs) {
    $kustomization = Join-Path $profileDir.FullName "kustomization.yaml"
    if (Test-Path $kustomization) {
      Invoke-KustomizeBuild -Path $profileDir.FullName
    }
  }
}

try {
  $afterContext = (& kubectl config current-context 2>$null | Select-Object -First 1)
} catch {
  $afterContext = ""
}

if ($beforeContext -ne $afterContext) {
  throw "kubectl current-context changed during no-trace test ($beforeContext -> $afterContext)"
}

Write-Host "Kubernetes no-trace preflight passed." -ForegroundColor Green
