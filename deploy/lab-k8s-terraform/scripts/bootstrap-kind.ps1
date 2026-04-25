$ErrorActionPreference = "Stop"

$clusterName = "tarinio-lab"
$namespace = "tarinio-lab"

if (-not (Get-Command kind -ErrorAction SilentlyContinue)) {
  throw "kind is not installed. Install kind or use k3d flow."
}

$clusters = kind get clusters 2>$null
if ($clusters -notcontains $clusterName) {
  Write-Host ("Creating kind cluster: {0}" -f $clusterName) -ForegroundColor Cyan
  kind create cluster --name $clusterName
} else {
  Write-Host ("kind cluster already exists: {0}" -f $clusterName) -ForegroundColor Yellow
}

Write-Host ("Ensuring namespace: {0}" -f $namespace) -ForegroundColor Cyan
kubectl create namespace $namespace --dry-run=client -o yaml | kubectl apply -f -

Write-Host "kind bootstrap completed." -ForegroundColor Green
