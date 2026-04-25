param(
  [string]$Namespace = "tarinio-lab",
  [int]$ControlPlaneLocalPort = 18080
)

$ErrorActionPreference = "Stop"

function Wait-HttpOk {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [int]$TimeoutSeconds = 60
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $resp = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
      if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) {
        return
      }
    } catch {
      Start-Sleep -Seconds 2
      continue
    }
  }
  throw "Timeout waiting for URL: $Url"
}

function Decode-SecretValue {
  param(
    [Parameter(Mandatory = $true)][string]$Namespace,
    [Parameter(Mandatory = $true)][string]$SecretName,
    [Parameter(Mandatory = $true)][string]$Key
  )
  $raw = kubectl -n $Namespace get secret $SecretName -o "jsonpath={.data.$Key}"
  if ([string]::IsNullOrWhiteSpace($raw)) {
    throw "Secret key $Key not found in $SecretName"
  }
  [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($raw))
}

Write-Host "== TARINIO k8s extended smoke ==" -ForegroundColor Cyan

kubectl -n $Namespace get pods
kubectl -n $Namespace get svc

kubectl -n $Namespace rollout status deploy/control-plane --timeout=240s
kubectl -n $Namespace rollout status deploy/runtime --timeout=240s
kubectl -n $Namespace rollout status deploy/ui --timeout=180s
kubectl -n $Namespace rollout status deploy/opensearch --timeout=240s
kubectl -n $Namespace rollout status deploy/postgres --timeout=240s

$pf = Start-Process -FilePath "kubectl" -ArgumentList @("-n", $Namespace, "port-forward", "svc/control-plane", "$ControlPlaneLocalPort`:8080") -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 3

try {
  $baseUrl = "http://127.0.0.1:$ControlPlaneLocalPort"
  Wait-HttpOk -Url "$baseUrl/healthz" -TimeoutSeconds 60

  $adminPassword = Decode-SecretValue -Namespace $Namespace -SecretName "tarinio-lab-secrets" -Key "CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD"
  $loginBody = @{
    username = "admin"
    password = $adminPassword
  } | ConvertTo-Json -Depth 5

  $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
  $loginResp = Invoke-RestMethod -Uri "$baseUrl/api/auth/login" -Method Post -ContentType "application/json" -Body $loginBody -WebSession $session
  if ($loginResp.requires_2fa -eq $true) {
    throw "Smoke user requires 2FA; lab smoke expects non-2FA admin login."
  }

  $me = Invoke-RestMethod -Uri "$baseUrl/api/auth/me" -Method Get -WebSession $session
  if (-not $me.username) {
    throw "Auth check failed: /api/auth/me has no username."
  }

  Write-Host ("Authenticated as: {0}" -f $me.username) -ForegroundColor Green

  $compile = Invoke-RestMethod -Uri "$baseUrl/api/revisions/compile" -Method Post -WebSession $session
  $revisionId = [string]$compile.revision.id
  if ([string]::IsNullOrWhiteSpace($revisionId)) {
    throw "Compile response has no revision.id."
  }
  Write-Host ("Compiled revision: {0}" -f $revisionId) -ForegroundColor Green

  $applyResp = Invoke-RestMethod -Uri "$baseUrl/api/revisions/$revisionId/apply" -Method Post -WebSession $session
  if (-not $applyResp.id) {
    throw "Apply response has no job id."
  }

  Write-Host ("Apply job: {0}" -f $applyResp.id) -ForegroundColor Green

  $trafficCmd = @"
set -e
for i in \$(seq 1 15); do
  curl -sS -o /dev/null http://runtime/ || true
done
curl -sS -o /dev/null "http://runtime/?q=%3Cscript%3Ealert(1)%3C/script%3E" || true
"@

  kubectl -n $Namespace run smoke-traffic --image=curlimages/curl:8.7.1 --restart=Never --rm --command -- sh -c $trafficCmd | Out-Null
  Start-Sleep -Seconds 4

  $events = Invoke-RestMethod -Uri "$baseUrl/api/events" -Method Get -WebSession $session
  $requests = Invoke-RestMethod -Uri "$baseUrl/api/requests?limit=50" -Method Get -WebSession $session
  $settings = Invoke-RestMethod -Uri "$baseUrl/api/settings/runtime" -Method Get -WebSession $session
  $sites = Invoke-RestMethod -Uri "$baseUrl/api/sites" -Method Get -WebSession $session

  $eventCount = 0
  if ($events -and $events.events) { $eventCount = @($events.events).Count }
  $requestCount = @($requests).Count
  if ($requestCount -lt 1) {
    throw "Expected at least one request item after traffic generation."
  }
  if (-not $settings.storage -or $null -eq $settings.storage.bans_days) {
    throw "Expected storage.bans_days in runtime settings."
  }

  $banCheck = "skipped (no sites)"
  if (@($sites).Count -gt 0) {
    $siteId = [string]$sites[0].id
    if (-not [string]::IsNullOrWhiteSpace($siteId)) {
      $banBody = @{ ip = "203.0.113.77" } | ConvertTo-Json
      $null = Invoke-RestMethod -Uri "$baseUrl/api/sites/$siteId/ban" -Method Post -ContentType "application/json" -Body $banBody -WebSession $session
      $null = Invoke-RestMethod -Uri "$baseUrl/api/sites/$siteId/unban" -Method Post -ContentType "application/json" -Body $banBody -WebSession $session
      $banCheck = "ok (site: $siteId)"
    }
  }

  Write-Host ("Events: {0}, Requests: {1}, Bans check: {2}" -f $eventCount, $requestCount, $banCheck) -ForegroundColor Green
  Write-Host "Extended smoke passed." -ForegroundColor Green
}
finally {
  if ($pf -and -not $pf.HasExited) {
    Stop-Process -Id $pf.Id -Force
  }
}
