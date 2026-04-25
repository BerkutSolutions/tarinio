$ErrorActionPreference = "Stop"

function Get-EndpointAddressCount {
  $json = kubectl -n tarinio-lab get endpoints control-plane -o json | ConvertFrom-Json
  $count = 0
  if ($json.subsets) {
    foreach ($subset in $json.subsets) {
      if ($subset.addresses) { $count += $subset.addresses.Count }
    }
  }
  return $count
}

Write-Host "== HA smoke: control-plane ==" -ForegroundColor Cyan

kubectl -n tarinio-lab get pods -l app=control-plane
kubectl -n tarinio-lab get svc redis

$podCount = (kubectl -n tarinio-lab get pods -l app=control-plane --no-headers 2>$null | Measure-Object -Line).Lines
if ($podCount -lt 2) {
  throw "Expected at least 2 control-plane pods, got $podCount"
}

$endpointCount = Get-EndpointAddressCount
if ($endpointCount -lt 2) {
  throw "Expected at least 2 ready control-plane endpoints, got $endpointCount"
}

$metaRaw = kubectl -n tarinio-lab run ha-meta --rm -i --restart=Never --image=curlimages/curl:8.7.1 -- sh -lc "curl -fsS http://control-plane:8080/api/app/meta"
$meta = $metaRaw | ConvertFrom-Json

if (-not $meta.ha_enabled) {
  throw "Expected ha_enabled=true in /api/app/meta"
}
if (-not $meta.ha_node_id) {
  throw "Expected non-empty ha_node_id in /api/app/meta"
}

$redisPing = kubectl -n tarinio-lab exec deploy/redis -- redis-cli ping
if ($redisPing -notmatch "PONG") {
  throw "Redis ping failed: $redisPing"
}

Write-Host "HA smoke passed." -ForegroundColor Green
