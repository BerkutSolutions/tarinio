$ErrorActionPreference = "Stop"

Write-Host "== Guardrails smoke ==" -ForegroundColor Cyan

$ns = kubectl get ns tarinio-lab -o json | ConvertFrom-Json
$labels = $ns.metadata.labels

if ($labels."pod-security.kubernetes.io/enforce" -ne "baseline") {
  throw "Namespace PSS enforce label is not baseline."
}

$netpolCount = (kubectl -n tarinio-lab get networkpolicy --no-headers 2>$null | Measure-Object -Line).Lines
if ($netpolCount -lt 3) {
  throw "Expected at least 3 network policies, got $netpolCount"
}

$cp = kubectl -n tarinio-lab get deploy control-plane -o json | ConvertFrom-Json
$rt = kubectl -n tarinio-lab get deploy runtime -o json | ConvertFrom-Json
$ui = kubectl -n tarinio-lab get deploy ui -o json | ConvertFrom-Json

if (-not $cp.spec.template.spec.securityContext.seccompProfile) {
  throw "control-plane seccompProfile is missing"
}
if (-not $rt.spec.template.spec.securityContext.seccompProfile) {
  throw "runtime seccompProfile is missing"
}
if (-not $ui.spec.template.spec.securityContext.seccompProfile) {
  throw "ui seccompProfile is missing"
}

Write-Host "Guardrails smoke passed." -ForegroundColor Green
