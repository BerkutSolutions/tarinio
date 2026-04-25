$ErrorActionPreference = "Stop"

Write-Host "== TARINIO k8s basic smoke ==" -ForegroundColor Cyan

kubectl -n tarinio-lab get pods
kubectl -n tarinio-lab get svc

Write-Host "Health checks from inside cluster..." -ForegroundColor Cyan
kubectl -n tarinio-lab run smoke-curl --rm -i --restart=Never --image=curlimages/curl:8.7.1 -- `
  sh -lc "curl -fsS http://control-plane:8080/healthz && curl -fsS http://runtime:8081/healthz && curl -fsS http://ui/healthcheck.html >/dev/null"

Write-Host "Smoke passed." -ForegroundColor Green
