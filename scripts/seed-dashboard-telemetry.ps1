param(
  [string]$ComposeFile = "deploy/compose/default/docker-compose.yml",
  [string]$RuntimeService = "runtime",
  [string]$SiteID = "localhost",
  [int]$Hours = 24,
  [int]$RequestsPerHour = 8,
  [int]$BlockedPerHour = 3
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
    [string]$RequestID
  )
  [ordered]@{
    timestamp = $Timestamp.ToUniversalTime().ToString("o")
    request_id = $RequestID
    client_ip = $IP
    country = $Country
    city = $City
    host = "dashboard-demo.local"
    method = "GET"
    uri = $URI
    status = $Status
    bytes_sent = 0
    referer = ""
    user_agent = "WAF dashboard telemetry demo"
    site = $SiteID
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
  $requestCount = $RequestsPerHour + (Get-Random -Minimum 0 -Maximum 22)
  $blockedCount = $BlockedPerHour + (Get-Random -Minimum 0 -Maximum 10)
  if ($hour % 5 -eq 0) {
    $requestCount += Get-Random -Minimum 14 -Maximum 36
    $blockedCount += Get-Random -Minimum 6 -Maximum 17
  }
  for ($i = 0; $i -lt $requestCount; $i++) {
    $location = $locations[($hour + $i) % $locations.Count]
    $minute = [math]::Floor(($i * 58) / [math]::Max($requestCount, 1))
    $lines.Add((New-AccessEvent $bucket.AddMinutes($minute) $location.IP $location.Country $location.City 200 "/catalog/demo-$hour-$i" "demo-request-$hour-$i"))
  }
  for ($i = 0; $i -lt $blockedCount; $i++) {
    $location = $locations[($hour + $i + 1) % $locations.Count]
    $status = @(403, 429, 444)[$i % 3]
    $minute = 30 + [math]::Floor(($i * 28) / [math]::Max($blockedCount, 1))
    $lines.Add((New-AccessEvent $bucket.AddMinutes($minute) $location.IP $location.Country $location.City $status $attackPaths[$i % $attackPaths.Count] "demo-attack-$hour-$i"))
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
Write-Host "Done. Wait up to 10 seconds, refresh Dashboard, and inspect traffic, attacks, blocked attacks, countries, and the 24-hour chart."
