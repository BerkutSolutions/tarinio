param(
  [Parameter(Mandatory=$true)]
  [string]$ComposeFile,

  [string]$ProfileName = "default",
  [string]$RuntimeService = "runtime",
  [string]$ControlPlaneService = "control-plane",
  [string]$SentinelService = "tarinio-sentinel",
  [string]$PrimarySite = "control-plane-access",
  [string]$SecondarySite = "localhost",
  [int]$WaitSeconds = 10,
  [string]$OutputDir = "",
  [ValidateSet("hybrid", "classic-only", "adaptive-only")]
  [string]$Mode = "hybrid",
  [switch]$NoReset
)

$ErrorActionPreference = "Stop"

function Invoke-Compose {
  param([string[]]$ComposeArgs)
  & docker compose -f $ComposeFile @ComposeArgs
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose failed: $($ComposeArgs -join ' ')"
  }
}

function Invoke-ComposeText {
  param([string[]]$ComposeArgs)
  $output = & docker compose -f $ComposeFile @ComposeArgs
  if ($LASTEXITCODE -ne 0) {
    throw "docker compose failed: $($ComposeArgs -join ' ')"
  }
  return ($output -join "`n")
}

function Invoke-ComposeWithRetry {
  param(
    [string[]]$ComposeArgs,
    [int]$Attempts = 8,
    [int]$DelaySeconds = 2
  )
  $lastError = $null
  for ($i = 0; $i -lt $Attempts; $i++) {
    try {
      Invoke-Compose $ComposeArgs
      return
    } catch {
      $lastError = $_
      if ($i -lt ($Attempts - 1)) {
        Start-Sleep -Seconds $DelaySeconds
      }
    }
  }
  if ($null -ne $lastError) {
    throw $lastError
  }
  throw "docker compose retry exhausted"
}

function New-AccessLine {
  param(
    [string]$Timestamp,
    [string]$IP,
    [string]$Site,
    [int]$Status,
    [string]$Method,
    [string]$URI,
    [string]$UserAgent
  )
  $obj = [ordered]@{
    timestamp = $Timestamp
    client_ip = $IP
    site = $Site
    status = $Status
    method = $Method
    uri = $URI
    user_agent = $UserAgent
  }
  return ($obj | ConvertTo-Json -Compress)
}

function Add-ScenarioLines {
  param([string[]]$Lines)
  $batchSize = 40
  for ($start = 0; $start -lt $Lines.Count; $start += $batchSize) {
    $end = [Math]::Min($start + $batchSize - 1, $Lines.Count - 1)
    $batch = @($Lines[$start..$end])
    Add-ScenarioLineBatch $batch
  }
}

function Add-ScenarioLineBatch {
  param([string[]]$Lines)
  $payload = ($Lines -join "`n") + "`n"
  $encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($payload))
  Invoke-ComposeWithRetry @("exec", "-T", $RuntimeService, "sh", "-lc", "printf '%s' '$encoded' | base64 -d >> /var/log/nginx/access.log")
}

function Read-ContainerJSON {
  param([string]$Path)
  $raw = Invoke-ComposeText @("exec", "-T", $SentinelService, "sh", "-lc", "cat $Path")
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $null
  }
  return $raw | ConvertFrom-Json
}

function Read-ContainerJSONSafe {
  param([string]$Path)
  try {
    return Read-ContainerJSON $Path
  } catch {
    return $null
  }
}

function Read-ServiceJSONSafe {
  param([string]$Service, [string]$Path)
  try {
    $raw = Invoke-ComposeText @("exec", "-T", $Service, "sh", "-lc", "cat $Path")
    if ([string]::IsNullOrWhiteSpace($raw)) {
      return $null
    }
    return ($raw | ConvertFrom-Json)
  } catch {
    return $null
  }
}

function Read-ServiceTextSafe {
  param([string]$Service, [string]$Command)
  try {
    $raw = Invoke-ComposeText @("exec", "-T", $Service, "sh", "-lc", $Command)
    return [string]$raw
  } catch {
    return ""
  }
}

function Reset-SmokeState {
  Write-Host "[$ProfileName] resetting runtime log and sentinel state"
  Invoke-ComposeWithRetry @("exec", "-T", $RuntimeService, "sh", "-lc", "truncate -s 0 /var/log/nginx/access.log || : > /var/log/nginx/access.log")
  Invoke-ComposeWithRetry @("exec", "-T", $SentinelService, "sh", "-lc", "rm -f /state/model-state.json /out/adaptive.json /out/l7-suggestions.json")
  Invoke-Compose @("restart", $SentinelService)
  Start-Sleep -Seconds 5
}

function Count-EntriesByAction {
  param($Adaptive)
  $out = [ordered]@{}
  foreach ($entry in @($Adaptive.entries)) {
    $action = [string]$entry.action
    if (-not $out.Contains($action)) {
      $out[$action] = 0
    }
    $out[$action]++
  }
  return $out
}

function Get-L4BaselineSnapshot {
  $cfg = Read-ServiceJSONSafe -Service $RuntimeService -Path "/etc/waf/l4guard/config.json"
  if ($null -eq $cfg) {
    $pointer = Read-ServiceJSONSafe -Service $RuntimeService -Path "/var/lib/waf/active/current.json"
    if ($null -ne $pointer -and -not [string]::IsNullOrWhiteSpace([string]$pointer.candidate_path)) {
      $candidate = [string]$pointer.candidate_path
      if (-not $candidate.StartsWith("/")) {
        $candidate = "/var/lib/waf/$candidate"
      }
      $cfg = Read-ServiceJSONSafe -Service $RuntimeService -Path "$candidate/l4guard/config.json"
    }
  }
  if ($null -eq $cfg) {
    $rawBaseline = Read-ServiceTextSafe -Service $ControlPlaneService -Command "printenv WAF_DEFAULT_ANTIDDOS_CONN_LIMIT; printenv WAF_DEFAULT_ANTIDDOS_RATE_PER_SECOND; printenv WAF_DEFAULT_ANTIDDOS_RATE_BURST"
    $parts = @($rawBaseline -split "\r?\n" | ForEach-Object { $_.Trim() })
    if ($parts.Count -ge 3 -and -not [string]::IsNullOrWhiteSpace($parts[0]) -and -not [string]::IsNullOrWhiteSpace($parts[1]) -and -not [string]::IsNullOrWhiteSpace($parts[2])) {
      $conn = 0
      $rate = 0
      $burst = 0
      [void][int]::TryParse($parts[0], [ref]$conn)
      [void][int]::TryParse($parts[1], [ref]$rate)
      [void][int]::TryParse($parts[2], [ref]$burst)
      if ($conn -gt 0 -and $rate -gt 0 -and $burst -gt 0) {
        return [ordered]@{
          present = $true
          conn_limit = $conn
          rate_per_second = $rate
          rate_burst = $burst
          destination_ip = "control-plane-defaults"
          ports = @()
        }
      }
    }

    return [ordered]@{
      present = $false
      conn_limit = 0
      rate_per_second = 0
      rate_burst = 0
      destination_ip = ""
      ports = @()
    }
  }

  $ports = @()
  if ($null -ne $cfg.ports) {
    foreach ($item in @($cfg.ports)) {
      $ports += [int]$item
    }
  }
  return [ordered]@{
    present = $true
    conn_limit = [int]$cfg.conn_limit
    rate_per_second = [int]$cfg.rate_per_second
    rate_burst = [int]$cfg.rate_burst
    destination_ip = [string]$cfg.destination_ip
    ports = $ports
  }
}

function Count-StateRecords {
  param($State)
  if ($null -eq $State -or $null -eq $State.ips) {
    return 0
  }
  return @($State.ips.PSObject.Properties).Count
}

function Count-EntriesMatchingReason {
  param($Entries, [string]$Reason)
  return @($Entries | Where-Object { @($_.reason_codes) -contains $Reason }).Count
}

function Join-List {
  param($Items)
  $values = @($Items | Where-Object { -not [string]::IsNullOrWhiteSpace([string]$_) })
  if ($values.Count -eq 0) {
    return "none"
  }
  return ($values -join ", ")
}

function Write-MarkdownReport {
  param(
    [string]$Path,
    $Result
  )
  $actionLines = New-Object System.Collections.Generic.List[string]
  if ($Result.adaptive_actions -is [System.Collections.IDictionary]) {
    foreach ($key in $Result.adaptive_actions.Keys) {
      $actionLines.Add("- ${key}: $($Result.adaptive_actions[$key])")
    }
  } else {
    foreach ($property in $Result.adaptive_actions.PSObject.Properties) {
      $actionLines.Add("- $($property.Name): $($property.Value)")
    }
  }
  if ($actionLines.Count -eq 0) {
    $actionLines.Add("- none")
  }

  $md = @"
# Sentinel smoke result: $($Result.profile)

Generated at: $($Result.generated_at)
Mode: $($Result.mode)

## Verdict

- Passed: $($Result.passed)
- Injected synthetic access-log events: $($Result.injected_events)
- Adaptive entries: $($Result.adaptive_entries)
- L7 suggestions: $($Result.l7_suggestions)
- Tracked state records: $($Result.tracked_state_records)
- L4 baseline present: $($Result.l4_baseline.present)
- L4 conn_limit/rate/burst: $($Result.l4_baseline.conn_limit) / $($Result.l4_baseline.rate_per_second) / $($Result.l4_baseline.rate_burst)

## Scenario Coverage

| Scenario | Evidence | Result |
| --- | --- | --- |
| Normal traffic | IP $($Result.normal_ip), benign 200 responses | False positive adaptive entries: $($Result.normal_false_positive_entries) |
| Scanner paths | /.env, /wp-admin, /phpmyadmin, /vendor/phpunit | Suggestions matched: $($Result.scenario_counts.scanner_suggestions) |
| Brute force | Repeated /login 401/403/429 from $($Result.scenario_ips.brute_force) | Adaptive entries for source: $($Result.scenario_counts.brute_force_entries) |
| XSS / SQLi / RCE payload probes | Encoded script, UNION SELECT, shell metacharacters from $($Result.scenario_ips.payloads) | Adaptive entries for source: $($Result.scenario_counts.payload_entries) |
| Single-source DDoS | 220 requests in one second from $($Result.scenario_ips.single_source_flood) | Emergency single-source entries: $($Result.scenario_counts.single_source_emergency_entries) |
| Distributed DDoS | 420 requests in one second across many IPs | Emergency botnet entries: $($Result.scenario_counts.botnet_emergency_entries) |
| High cardinality | 160 unique paths over many sources | State records: $($Result.tracked_state_records) |

## Adaptive Actions

$($actionLines -join "`n")

## False Positive Assessment

The benign normal traffic source produced $($Result.normal_false_positive_entries) adaptive entries. A passing result requires this number to stay at zero.

## Mode-specific expectation

| Mode | Expected behavior |
| --- | --- |
| classic-only | No adaptive entries; baseline L4 Anti-DDoS limits active |
| hybrid | Baseline L4 limits active and adaptive entries present for attack traffic |
| adaptive-only | Adaptive entries drive mitigation while baseline L4 profile remains active |

## L7 Suggestions

Matched path prefixes: $(Join-List $Result.l7_suggestion_paths)

## Artifacts

- result.json
- adaptive.json
- l7-suggestions.json
- model-state.json
"@
  Set-Content -LiteralPath $Path -Value $md -Encoding UTF8
}

$site = $PrimarySite
$attackSite = $SecondarySite
$now = [DateTime]::UtcNow
$normalIP = "198.51.100.10"
$scannerIPs = @("203.0.113.11","203.0.113.12","203.0.113.13","203.0.113.14","203.0.113.15","203.0.113.16")
$bruteIP = "203.0.113.30"
$payloadIP = "203.0.113.31"
$floodIP = "203.0.113.40"

$lines = New-Object System.Collections.Generic.List[string]

for ($i = 0; $i -lt 20; $i++) {
  $lines.Add((New-AccessLine $now.AddSeconds($i).ToString("o") $normalIP $site 200 "GET" "/dashboard" "Mozilla/5.0 enterprise-smoke-normal"))
}

$scannerPaths = @("/.env", "/wp-admin", "/phpmyadmin", "/vendor/phpunit")
$scannerIndex = 0
foreach ($path in $scannerPaths) {
  for ($hit = 0; $hit -lt 48; $hit++) {
    $ip = $scannerIPs[$hit % $scannerIPs.Count]
    $eventSite = @($site, $attackSite)[([math]::Floor($hit / $scannerIPs.Count)) % 2]
    $lines.Add((New-AccessLine $now.AddSeconds(30 + $scannerIndex).ToString("o") $ip $eventSite 404 "GET" $path "sqlmap/1.7 sentinel-smoke-scanner"))
    $scannerIndex++
  }
}

for ($i = 0; $i -lt 56; $i++) {
  $status = @(401, 403, 429)[$i % 3]
  $eventSite = @($site, $attackSite)[$i % 2]
  $lines.Add((New-AccessLine $now.AddSeconds(190 + $i).ToString("o") $bruteIP $eventSite $status "POST" "/login" "python-requests/2 sentinel-smoke-bruteforce"))
}

$payloadPaths = @(
  "/search?q=%3Cscript%3Ealert(1)%3C/script%3E",
  "/product?id=1%20UNION%20SELECT%20password%20FROM%20users",
  "/api/check?cmd=%3Bcat%20/etc/passwd"
)
for ($i = 0; $i -lt 48; $i++) {
  $eventSite = @($site, $attackSite)[$i % 2]
  $lines.Add((New-AccessLine $now.AddSeconds(260 + $i).ToString("o") $payloadIP $eventSite 403 "GET" $payloadPaths[$i % $payloadPaths.Count] "sqlmap/1.7 sentinel-smoke-payload"))
}

$floodSecond = $now.AddSeconds(340).ToString("yyyy-MM-ddTHH:mm:ssZ")
for ($i = 0; $i -lt 220; $i++) {
  $eventSite = @($site, $attackSite)[$i % 2]
  $lines.Add((New-AccessLine $floodSecond $floodIP $eventSite 429 "GET" "/api/flood" "go-http-client/1.1 sentinel-smoke-single-flood"))
}

$botnetSecond = $now.AddSeconds(380).ToString("yyyy-MM-ddTHH:mm:ssZ")
for ($i = 0; $i -lt 420; $i++) {
  $third = [math]::Floor($i / 200)
  $fourth = 50 + ($i % 200)
  $eventSite = @($site, $attackSite)[$i % 2]
  $lines.Add((New-AccessLine $botnetSecond "198.51.$third.$fourth" $eventSite 429 "GET" "/api/botnet" "masscan sentinel-smoke-distributed-flood"))
}

for ($i = 0; $i -lt 160; $i++) {
  $third = [math]::Floor($i / 200)
  $fourth = 60 + ($i % 190)
  $lines.Add((New-AccessLine $now.AddSeconds(260 + $i).ToString("o") "192.0.$third.$fourth" $site 404 "GET" "/random-$i" "Mozilla/5.0 sentinel-smoke-cardinality"))
}

Write-Host "[$ProfileName] validating compose"
Invoke-Compose @("config", "--quiet")

Write-Host "[$ProfileName] starting containers"
Invoke-Compose @("up", "-d", "--build")

Write-Host "[$ProfileName] waiting for runtime and sentinel"
Start-Sleep -Seconds 15
Invoke-Compose @("ps", $RuntimeService, $SentinelService)
if (-not $NoReset) {
  Reset-SmokeState
}

Write-Host "[$ProfileName] injecting $($lines.Count) synthetic access-log events"
Add-ScenarioLines $lines.ToArray()
Start-Sleep -Seconds $WaitSeconds

$adaptive = Read-ContainerJSONSafe "/out/adaptive.json"
$suggestions = Read-ContainerJSONSafe "/out/l7-suggestions.json"
$state = Read-ContainerJSONSafe "/state/model-state.json"
$l4Baseline = Get-L4BaselineSnapshot

$entries = @()
if ($null -ne $adaptive -and $null -ne $adaptive.entries) {
  $entries = @($adaptive.entries | Where-Object { $null -ne $_ })
}
$suggestionItems = @()
if ($null -ne $suggestions -and $null -ne $suggestions.items) {
  $suggestionItems = @($suggestions.items | Where-Object { $null -ne $_ })
}
$normalFalsePositive = @($entries | Where-Object { $_.ip -eq $normalIP }).Count
$scannerSuggestionCount = @($suggestionItems | Where-Object { $_.path_prefix -in @("/.env", "/wp-admin", "/phpmyadmin", "/vendor/phpunit") }).Count
$maliciousActions = @($entries | Where-Object { $_.ip -in @($bruteIP, $payloadIP, $floodIP) -or $_.reason_codes -contains "signal_emergency_botnet" -or $_.reason_codes -contains "signal_emergency_single" }).Count
$bruteEntries = @($entries | Where-Object { $_.ip -eq $bruteIP }).Count
$payloadEntries = @($entries | Where-Object { $_.ip -eq $payloadIP }).Count
$singleEmergencyEntries = Count-EntriesMatchingReason $entries "signal_emergency_single"
$botnetEmergencyEntries = Count-EntriesMatchingReason $entries "signal_emergency_botnet"

$modePassed = $false
switch ($Mode) {
  "classic-only" {
    $modePassed = ($entries.Count -eq 0 -and $normalFalsePositive -eq 0 -and $l4Baseline.present -and $l4Baseline.conn_limit -gt 0 -and $l4Baseline.rate_per_second -gt 0 -and $l4Baseline.rate_burst -gt 0)
  }
  "adaptive-only" {
    $modePassed = ($normalFalsePositive -eq 0 -and $scannerSuggestionCount -gt 0 -and $maliciousActions -gt 0 -and $entries.Count -gt 0 -and $l4Baseline.present)
  }
  default {
    $modePassed = ($normalFalsePositive -eq 0 -and $scannerSuggestionCount -gt 0 -and $maliciousActions -gt 0 -and $entries.Count -gt 0 -and $l4Baseline.present -and $l4Baseline.conn_limit -gt 0 -and $l4Baseline.rate_per_second -gt 0)
  }
}

$result = [ordered]@{
  profile = $ProfileName
  mode = $Mode
  generated_at = (Get-Date).ToUniversalTime().ToString("o")
  injected_events = $lines.Count
  adaptive_entries = $entries.Count
  adaptive_actions = Count-EntriesByAction $adaptive
  l7_suggestions = $suggestionItems.Count
  l7_suggestion_paths = @($suggestionItems | ForEach-Object { $_.path_prefix } | Sort-Object -Unique)
  tracked_state_records = Count-StateRecords $state
  scenario_ips = [ordered]@{
    normal = $normalIP
    brute_force = $bruteIP
    payloads = $payloadIP
    single_source_flood = $floodIP
  }
  scenario_counts = [ordered]@{
    scanner_suggestions = $scannerSuggestionCount
    brute_force_entries = $bruteEntries
    payload_entries = $payloadEntries
    single_source_emergency_entries = $singleEmergencyEntries
    botnet_emergency_entries = $botnetEmergencyEntries
  }
  l4_baseline = $l4Baseline
  normal_ip = $normalIP
  normal_false_positive_entries = $normalFalsePositive
  scanner_suggestion_count = $scannerSuggestionCount
  malicious_action_count = $maliciousActions
  passed = $modePassed
}

if (-not [string]::IsNullOrWhiteSpace($OutputDir)) {
  New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
  $result | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $OutputDir "result.json") -Encoding UTF8
  $adaptive | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $OutputDir "adaptive.json") -Encoding UTF8
  $suggestions | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $OutputDir "l7-suggestions.json") -Encoding UTF8
  $state | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $OutputDir "model-state.json") -Encoding UTF8
  Write-MarkdownReport (Join-Path $OutputDir "report.md") ([pscustomobject]$result)
}

$result | ConvertTo-Json -Depth 10

if (-not $result.passed) {
  throw "sentinel smoke failed for $ProfileName"
}
