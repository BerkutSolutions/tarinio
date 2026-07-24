param(
  [string]$ComposeFile = "deploy/compose/default/docker-compose.yml",
  [string]$RuntimeService = "runtime",
  [string]$SiteID = "localhost",
  [string]$SecondarySiteID = "dashboard-demo-secondary",
  [int]$Hours = 24,
  [int]$RequestsPerHour = 8,
  [int]$BlockedPerHour = 3,
  [string]$VerifyControlPlaneURL = "",
  [string]$Username = "e2e-admin",
  [string]$Password = "e2e-password-1234"
)

$ErrorActionPreference = "Stop"

if ($Hours -lt 1 -or $Hours -gt 24) {
  throw "Hours must be between 1 and 24."
}
if ($RequestsPerHour -lt 1 -or $BlockedPerHour -lt 1) {
  throw "RequestsPerHour and BlockedPerHour must be positive."
}
if (-not (Test-Path -LiteralPath $ComposeFile)) {
  throw "Compose file was not found: $ComposeFile"
}

function Invoke-Compose {
  param([string[]]$ComposeArgs)
  & docker compose -f $ComposeFile @ComposeArgs
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose failed: $($ComposeArgs -join ' ')"
  }
}

function New-AccessEvent {
  param(
    [datetime]$Timestamp,
    [string]$IP,
    [string]$Country,
    [string]$City,
    [int]$Status,
    [string]$URI,
    [string]$RequestID,
    [string]$EventSiteID
  )
  [ordered]@{
    timestamp = $Timestamp.ToUniversalTime().ToString("o")
    request_id = $RequestID
    client_ip = $IP
    country = $Country
    city = $City
    host = if ($EventSiteID -eq $SiteID) { "dashboard-demo.local" } else { "dashboard-demo-secondary.local" }
    method = "GET"
    uri = $URI
    status = $Status
    bytes_sent = 0
    referer = ""
    user_agent = "WAF dashboard telemetry demo"
    site = $EventSiteID
    security_reason = if ($Status -ge 400) { "dashboard_demo" } else { "" }
    upstream_addr = ""
    request_time = 0.001
  } | ConvertTo-Json -Compress
}

$locations = @(
  @{ IP = "198.51.100.24"; Country = "RU"; City = "Moscow" },
  @{ IP = "203.0.113.38"; Country = "DE"; City = "Frankfurt" },
  @{ IP = "192.0.2.57"; Country = "US"; City = "Ashburn" },
  @{ IP = "198.51.100.91"; Country = "JP"; City = "Tokyo" },
  @{ IP = "203.0.113.116"; Country = "BR"; City = "Sao Paulo" }
)
$attackPaths = @("/waf-test/payload", "/geo-block/dashboard-demo", "/login/dashboard-demo")
$lines = [System.Collections.Generic.List[string]]::new()
$start = [DateTime]::UtcNow.AddHours(-($Hours - 1)).Date.AddHours([DateTime]::UtcNow.Hour)

for ($hour = 0; $hour -lt $Hours; $hour++) {
  $bucket = $start.AddHours($hour)
  $requestCount = $RequestsPerHour
  $blockedCount = $BlockedPerHour
  for ($i = 0; $i -lt $requestCount; $i++) {
    $location = $locations[($hour + $i) % $locations.Count]
    $minute = [math]::Floor(($i * 58) / [math]::Max($requestCount, 1))
    $eventSiteID = if ($i % 2 -eq 0) { $SiteID } else { $SecondarySiteID }
    $lines.Add((New-AccessEvent $bucket.AddMinutes($minute) $location.IP $location.Country $location.City 200 "/catalog/demo-$hour-$i" "dashboard-e2e-request-$hour-$i" $eventSiteID))
  }
  for ($i = 0; $i -lt $blockedCount; $i++) {
    $location = $locations[($hour + $i + 1) % $locations.Count]
    $status = @(403, 429, 444)[$i % 3]
    $minute = 30 + [math]::Floor(($i * 28) / [math]::Max($blockedCount, 1))
    $eventSiteID = if ($i % 2 -eq 0) { $SecondarySiteID } else { $SiteID }
    $lines.Add((New-AccessEvent $bucket.AddMinutes($minute) $location.IP $location.Country $location.City $status $attackPaths[$i % $attackPaths.Count] "dashboard-e2e-attack-$hour-$i" $eventSiteID))
  }
}

$payload = ($lines -join "`n") + "`n"
$encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($payload))
Write-Host "Appending $($lines.Count) demo telemetry entries to $RuntimeService..."
$OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$encoded | & docker compose -f $ComposeFile exec -T $RuntimeService sh -lc "tr -d '\r\n' | base64 -d >> /var/log/nginx/access.log"
if ($LASTEXITCODE -ne 0) {
  throw "docker compose failed to append demo telemetry."
}
$expectedRequests = $Hours * ($RequestsPerHour + $BlockedPerHour)
$expectedBlocked = $Hours * $BlockedPerHour
Write-Host "Done. Wait up to 10 seconds, refresh Dashboard, and inspect traffic, attacks, blocked attacks, countries, and the 24-hour chart."
Write-Host "Deterministic seed: $expectedRequests requests, $expectedBlocked blocked, two sites, five IPs/countries."

if ([string]::IsNullOrWhiteSpace($VerifyControlPlaneURL)) {
  return
}

$baseURL = $VerifyControlPlaneURL.TrimEnd('/')
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$loginBody = @{ username = $Username; password = $Password } | ConvertTo-Json
Invoke-RestMethod -Uri "$baseURL/api/auth/login" -Method Post -WebSession $session -ContentType "application/json" -Body $loginBody | Out-Null

function Invoke-E2EAPI {
  param([string]$Path, [string]$Method = "Get", [object]$Body = $null)
  $arguments = @{ Uri = "$baseURL$Path"; Method = $Method; WebSession = $session; ContentType = "application/json" }
  if ($null -ne $Body) { $arguments.Body = ($Body | ConvertTo-Json -Depth 12) }
  Invoke-RestMethod @arguments
}

function Wait-E2ECondition {
  param([scriptblock]$Condition, [string]$Description, [int]$TimeoutSeconds = 30)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    if (& $Condition) { return }
    Start-Sleep -Milliseconds 500
  } while ((Get-Date) -lt $deadline)
  throw "Timed out waiting for $Description"
}

Wait-E2ECondition -Description "seeded requests" -Condition {
  $requests = @(Invoke-E2EAPI -Path "/api/requests?limit=500")
  $count = Invoke-E2EAPI -Path "/api/requests?count=1"
  $ids = @($requests.entry.request_id | ForEach-Object { [string]$_ })
  [int]$count.count -ge $expectedRequests -and
    $ids -contains "dashboard-e2e-request-23-0" -and
    $ids -contains "dashboard-e2e-attack-23-0"
}
Wait-E2ECondition -Description "seeded dashboard aggregation" -Condition {
  $stats = Invoke-E2EAPI -Path "/api/dashboard/stats"
  @($stats.requests_series).Count -eq 24 -and @($stats.request_top_sites).Count -ge 2
}

$catalog = Invoke-E2EAPI -Path "/api/revisions"
$originalRevision = @($catalog.revisions | Where-Object { $_.is_active } | Select-Object -First 1).id
if ([string]::IsNullOrWhiteSpace($originalRevision)) { throw "Active revision was not found before observability seed" }
$compiled = Invoke-E2EAPI -Path "/api/revisions/compile" -Method Post -Body @{}
$seedRevision = [string]$compiled.revision.id
if ([string]::IsNullOrWhiteSpace($seedRevision)) { throw "Compile response did not contain revision.id" }
try {
  Invoke-E2EAPI -Path "/api/revisions/$seedRevision/apply" -Method Post -Body @{} | Out-Null
  Wait-E2ECondition -Description "seed revision activation" -TimeoutSeconds 120 -Condition {
    $current = Invoke-E2EAPI -Path "/api/revisions"
    [string](@($current.revisions | Where-Object { $_.is_active } | Select-Object -First 1).id) -eq $seedRevision
  }
  Wait-E2ECondition -Description "seeded operational event and audit entry" -Condition {
    $events = Invoke-E2EAPI -Path "/api/events?limit=500"
    $audit = Invoke-E2EAPI -Path "/api/audit?limit=500"
    @($events.events | Where-Object { $_.related_revision_id -eq $seedRevision }).Count -gt 0 -and
      @($audit.items | Where-Object { $_.related_revision_id -eq $seedRevision }).Count -gt 0
  }
} finally {
  Invoke-E2EAPI -Path "/api/revisions/$originalRevision/apply" -Method Post -Body @{} | Out-Null
  Wait-E2ECondition -Description "original revision restore" -TimeoutSeconds 120 -Condition {
    $current = Invoke-E2EAPI -Path "/api/revisions"
    [string](@($current.revisions | Where-Object { $_.is_active } | Select-Object -First 1).id) -eq $originalRevision
  }
}
Write-Host "Verified deterministic Requests, Dashboard, Events and Audit seed; restored active revision $originalRevision."
