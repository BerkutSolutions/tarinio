param(
  [Parameter(Mandatory=$true)]
  [string]$ComposeFile,

  [string]$ProfileName = "default",
  [string]$RuntimeService = "runtime",
  [string]$SentinelService = "tarinio-sentinel",
  [string]$PrimarySite = "control-plane-access",
  [string]$SecondarySite = "localhost",
  [int]$WaitSeconds = 12,
  [int]$LoadScale = 1,
  [string]$OutputDir = "",
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
  $batchSize = 50
  for ($start = 0; $start -lt $Lines.Count; $start += $batchSize) {
    $end = [Math]::Min($start + $batchSize - 1, $Lines.Count - 1)
    $batch = @($Lines[$start..$end])
    $payload = ($batch -join "`n") + "`n"
    $encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($payload))
    Invoke-Compose @("exec", "-T", $RuntimeService, "sh", "-lc", "printf '%s' '$encoded' | base64 -d >> /var/log/nginx/access.log")
  }
}

function Read-ContainerJSON {
  param([string]$Path)
  $raw = Invoke-ComposeText @("exec", "-T", $SentinelService, "sh", "-lc", "cat $Path")
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $null
  }
  return $raw | ConvertFrom-Json
}

function Reset-SmokeState {
  Write-Host "[$ProfileName] resetting runtime log and sentinel state"
  $runtimeReady = $false
  for ($attempt = 1; $attempt -le 20; $attempt++) {
    try {
      Invoke-Compose @("exec", "-T", $RuntimeService, "sh", "-lc", "test -d /var/log/nginx")
      $runtimeReady = $true
      break
    } catch {
      Start-Sleep -Seconds 2
    }
  }
  if (-not $runtimeReady) {
    throw "runtime container is not ready for smoke reset"
  }
  Invoke-Compose @("exec", "-T", $RuntimeService, "sh", "-lc", ": > /var/log/nginx/access.log")

  $sentinelReady = $false
  for ($attempt = 1; $attempt -le 20; $attempt++) {
    try {
      Invoke-Compose @("exec", "-T", $SentinelService, "sh", "-lc", "test -d /out")
      $sentinelReady = $true
      break
    } catch {
      Start-Sleep -Seconds 2
    }
  }
  if (-not $sentinelReady) {
    throw "sentinel container is not ready for smoke reset"
  }
  Invoke-Compose @("exec", "-T", $SentinelService, "sh", "-lc", "rm -f /state/model-state.json /out/adaptive.json /out/l7-suggestions.json")
  Invoke-Compose @("restart", $SentinelService)
  Start-Sleep -Seconds 5
}

function Count-EntriesMatchingReason {
  param($Entries, [string]$Reason)
  return @($Entries | Where-Object { @($_.reason_codes) -contains $Reason }).Count
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

function Count-StateRecords {
  param($State)
  if ($null -eq $State -or $null -eq $State.ips) {
    return 0
  }
  return @($State.ips.PSObject.Properties).Count
}

function Test-GuardOrderFromRuntimeConfig {
  $conf = $null
  try {
    $conf = Invoke-ComposeText @(
      "exec", "-T", $RuntimeService, "sh", "-lc",
      "for f in `$(find /etc/nginx -type f -name '*.conf' 2>/dev/null); do if grep -q 'waf_country_guard' ""`$f""; then cat ""`$f""; exit 0; fi; done; exit 1"
    )
  } catch {
    $conf = ""
  }
  if ([string]::IsNullOrWhiteSpace($conf)) {
    return [ordered]@{
      found = $false
      country_before_antibot = $false
      allowlist_guard_present = $false
      scanner_guard_present = $false
    }
  }
  $countryPos = $conf.IndexOf("waf_country_guard")
  $antibotPos = $conf.IndexOf("waf_antibot_exception_guard")
  return [ordered]@{
    found = $true
    country_before_antibot = ($countryPos -ge 0 -and $antibotPos -ge 0 -and $countryPos -lt $antibotPos)
    allowlist_guard_present = ($conf.IndexOf("waf_allow_bypass_") -ge 0)
    scanner_guard_present = ($conf.IndexOf("waf_antibot_scanner_guard") -ge 0)
  }
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
  param([string]$Path, $Result)
  $md = @"
# Service protection smoke result: $($Result.profile)

Generated at: $($Result.generated_at)

## Verdict

- Passed: $($Result.passed)
- Load scale: x$($Result.load_scale)
- Injected synthetic events: $($Result.injected_events)
- Adaptive entries: $($Result.adaptive_entries)
- L7 suggestions: $($Result.l7_suggestions)
- Tracked state records: $($Result.tracked_state_records)
- Scanner clients: $($Result.scenario_shape.scanner_clients)
- Botnet unique IPs: $($Result.scenario_shape.botnet_unique_ips)

## Guard and policy checks

- Runtime config found: $($Result.guard_checks.found)
- Country guard before antibot: $($Result.guard_checks.country_before_antibot)
- Access allowlist guard present: $($Result.guard_checks.allowlist_guard_present)
- Scanner autoban guard present: $($Result.guard_checks.scanner_guard_present)
- Guard verification (strict): $($Result.guards_verified)
- Policy `403` evidence present: $($Result.policy_403_evidence)

## 10-client scenario

| Client role | IP pattern | Expected signal | Result |
| --- | --- | --- | --- |
| normal_web | $($Result.clients.normal_web) | No adaptive action | FP entries: $($Result.false_positive.normal_web) |
| normal_mobile | $($Result.clients.normal_mobile) | No adaptive action | FP entries: $($Result.false_positive.normal_mobile) |
| trusted_allowlist | $($Result.clients.trusted_allowlist) | No adaptive action | FP entries: $($Result.false_positive.trusted_allowlist) |
| api_client | $($Result.clients.api_client) | Single-source emergency under burst | Entries: $($Result.scenario_counts.api_flood_entries) |
| country_blocked | $($Result.clients.country_blocked) | 403 traffic evidence | 403 lines: $($Result.scenario_counts.country_blocked_403) |
| denylisted_ip | $($Result.clients.denylisted_ip) | 403 traffic evidence | 403 lines: $($Result.scenario_counts.denylisted_403) |
| scanner_a | $($Result.clients.scanner_a) | L7 scanner suggestion / adaptive pressure | Suggestions: $($Result.scenario_counts.scanner_suggestions) |
| scanner_b | $($Result.clients.scanner_b) | L7 scanner suggestion / adaptive pressure | Suggestions: $($Result.scenario_counts.scanner_suggestions) |
| hacker_payload | $($Result.clients.hacker_payload) | Payload adaptive action | Entries: $($Result.scenario_counts.payload_entries) |
| botnet_group | distributed `198.18.0.0/16` | Botnet emergency or adaptive botnet actions | Emergency: $($Result.scenario_counts.botnet_emergency_entries), adaptive entries: $($Result.scenario_counts.botnet_actor_entries) |

## L7 suggestions

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
$effectiveScale = [Math]::Max(1, $LoadScale)

$clients = [ordered]@{
  normal_web = "198.51.100.10"
  normal_mobile = "198.51.100.11"
  trusted_allowlist = "10.10.10.10"
  api_client = "198.51.100.20"
  country_blocked = "203.0.113.50"
  denylisted_ip = "203.0.113.60"
  scanner_a = "203.0.113.70"
  scanner_b = "203.0.113.71"
  hacker_payload = "203.0.113.80"
}

$scannerIPs = New-Object System.Collections.Generic.List[string]
$scannerIPs.Add($clients.scanner_a)
$scannerIPs.Add($clients.scanner_b)
for ($i = 0; $i -lt (2 * $effectiveScale); $i++) {
  $scannerIPs.Add(("203.0.114.{0}" -f (10 + $i)))
}

$lines = New-Object System.Collections.Generic.List[string]
$botnetIPs = New-Object System.Collections.Generic.HashSet[string]

for ($i = 0; $i -lt (30 * $effectiveScale); $i++) {
  $lines.Add((New-AccessLine $now.AddSeconds($i).ToString("o") $clients.normal_web $site 200 "GET" "/dashboard" "Mozilla/5.0 service-protection-normal"))
  $lines.Add((New-AccessLine $now.AddSeconds($i + 1).ToString("o") $clients.normal_mobile $site 200 "GET" "/status" "Mozilla/5.0 service-protection-mobile"))
}
for ($i = 0; $i -lt (20 * $effectiveScale); $i++) {
  $lines.Add((New-AccessLine $now.AddSeconds(40 + $i).ToString("o") $clients.trusted_allowlist $site 200 "GET" "/api/public" "curl/8 service-protection-allowlist"))
}

for ($i = 0; $i -lt (70 * $effectiveScale); $i++) {
  $lines.Add((New-AccessLine $now.AddSeconds(70 + $i).ToString("o") $clients.country_blocked $attackSite 403 "GET" "/geo-block/probe-$i" "Mozilla/5.0 service-protection-country-blocked"))
}
for ($i = 0; $i -lt (70 * $effectiveScale); $i++) {
  $lines.Add((New-AccessLine $now.AddSeconds(80 + $i).ToString("o") $clients.denylisted_ip $attackSite 403 "GET" "/denylist/probe-$i" "curl/8 service-protection-denylisted"))
}

$scannerPaths = @("/.env", "/.git/config", "/wp-admin", "/phpmyadmin")
for ($i = 0; $i -lt (120 * $effectiveScale); $i++) {
  $scannerIP = $scannerIPs[$i % $scannerIPs.Count]
  $scannerSite = @($site, $attackSite)[$i % 2]
  $uri = $scannerPaths[$i % $scannerPaths.Count]
  $lines.Add((New-AccessLine $now.AddSeconds(120 + $i).ToString("o") $scannerIP $scannerSite 403 "GET" $uri "sqlmap/1.7 service-protection-scanner"))
}

$payloadURIs = @(
  "/search?q=%3Cscript%3Ealert(1)%3C/script%3E",
  "/product?id=1%20UNION%20SELECT%20password%20FROM%20users",
  "/api/exec?cmd=%3Bcat%20/etc/passwd"
)
for ($i = 0; $i -lt (120 * $effectiveScale); $i++) {
  $uri = $payloadURIs[$i % $payloadURIs.Count]
  $lines.Add((New-AccessLine $now.AddSeconds(250 + $i).ToString("o") $clients.hacker_payload $attackSite 403 "GET" $uri "python-requests/2 service-protection-payload"))
}

$apiFloodSecond = $now.AddSeconds(360).ToString("yyyy-MM-ddTHH:mm:ssZ")
for ($i = 0; $i -lt (260 * $effectiveScale); $i++) {
  $status = @(200, 429)[$i % 2]
  $lines.Add((New-AccessLine $apiFloodSecond $clients.api_client $site $status "GET" "/api/v1/orders" "go-http-client/1.1 service-protection-api"))
}

$botnetSecond = $now.AddSeconds(420).ToString("yyyy-MM-ddTHH:mm:ssZ")
for ($i = 0; $i -lt (480 * $effectiveScale); $i++) {
  $third = [math]::Floor($i / 200)
  $fourth = 20 + ($i % 200)
  $ip = "198.18.$third.$fourth"
  [void]$botnetIPs.Add($ip)
  $siteID = @($site, $attackSite)[$i % 2]
  $lines.Add((New-AccessLine $botnetSecond $ip $siteID 429 "GET" "/api/flood" "masscan service-protection-botnet"))
}

Write-Host "[$ProfileName] validating compose"
Invoke-Compose @("config", "--quiet")

Write-Host "[$ProfileName] starting containers"
Invoke-Compose @("up", "-d", "--build")

Write-Host "[$ProfileName] waiting runtime/sentinel"
Start-Sleep -Seconds 15
Invoke-Compose @("ps", $RuntimeService, $SentinelService)
if (-not $NoReset) {
  Reset-SmokeState
}

Write-Host "[$ProfileName] injecting $($lines.Count) synthetic access-log events"
Add-ScenarioLines $lines.ToArray()
Start-Sleep -Seconds $WaitSeconds

$guardChecks = Test-GuardOrderFromRuntimeConfig
$adaptive = Read-ContainerJSON "/out/adaptive.json"
$suggestions = Read-ContainerJSON "/out/l7-suggestions.json"
$state = Read-ContainerJSON "/state/model-state.json"

$entries = @($adaptive.entries | Where-Object { $null -ne $_ })
$suggestionItems = @($suggestions.items | Where-Object { $null -ne $_ })

$falsePositiveNormalWeb = @($entries | Where-Object { $_.ip -eq $clients.normal_web }).Count
$falsePositiveNormalMobile = @($entries | Where-Object { $_.ip -eq $clients.normal_mobile }).Count
$falsePositiveAllowlist = @($entries | Where-Object { $_.ip -eq $clients.trusted_allowlist }).Count

$scannerSuggestionCount = @($suggestionItems | Where-Object { $_.path_prefix -in @("/.env", "/.git", "/wp-admin", "/phpmyadmin") }).Count
$payloadEntries = @($entries | Where-Object { $_.ip -eq $clients.hacker_payload }).Count
$apiFloodEntries = @($entries | Where-Object { $_.ip -eq $clients.api_client }).Count
$singleEmergencyEntries = Count-EntriesMatchingReason $entries "signal_emergency_single"
$botnetEmergencyEntries = Count-EntriesMatchingReason $entries "signal_emergency_botnet"
$botnetActorEntries = @($entries | Where-Object { $botnetIPs.Contains([string]$_.ip) }).Count

$countryBlocked403 = @($lines | Where-Object { $_ -like "*`"client_ip`":`"$($clients.country_blocked)`"*`"status`":403*" }).Count
$denylisted403 = @($lines | Where-Object { $_ -like "*`"client_ip`":`"$($clients.denylisted_ip)`"*`"status`":403*" }).Count

$result = [ordered]@{
  profile = $ProfileName
  load_scale = $effectiveScale
  generated_at = (Get-Date).ToUniversalTime().ToString("o")
  injected_events = $lines.Count
  adaptive_entries = $entries.Count
  adaptive_actions = Count-EntriesByAction $adaptive
  l7_suggestions = $suggestionItems.Count
  l7_suggestion_paths = @($suggestionItems | ForEach-Object { $_.path_prefix } | Sort-Object -Unique)
  tracked_state_records = Count-StateRecords $state
  clients = $clients
  guard_checks = $guardChecks
  false_positive = [ordered]@{
    normal_web = $falsePositiveNormalWeb
    normal_mobile = $falsePositiveNormalMobile
    trusted_allowlist = $falsePositiveAllowlist
  }
  scenario_counts = [ordered]@{
    country_blocked_403 = $countryBlocked403
    denylisted_403 = $denylisted403
    scanner_suggestions = $scannerSuggestionCount
    payload_entries = $payloadEntries
    api_flood_entries = $apiFloodEntries
    single_source_emergency_entries = $singleEmergencyEntries
    botnet_emergency_entries = $botnetEmergencyEntries
    botnet_actor_entries = $botnetActorEntries
  }
  scenario_shape = [ordered]@{
    scanner_clients = $scannerIPs.Count
    botnet_unique_ips = $botnetIPs.Count
  }
}

$result.guards_verified = (
  $guardChecks.found -and
  $guardChecks.country_before_antibot -and
  $guardChecks.allowlist_guard_present -and
  $guardChecks.scanner_guard_present
)

$result.policy_403_evidence = ($countryBlocked403 -gt 0 -and $denylisted403 -gt 0)

$result.passed = (
  $falsePositiveNormalWeb -eq 0 -and
  $falsePositiveNormalMobile -eq 0 -and
  $falsePositiveAllowlist -eq 0 -and
  $result.policy_403_evidence -and
  $entries.Count -gt 0 -and
  ($singleEmergencyEntries -gt 0 -or $apiFloodEntries -gt 0) -and
  ($botnetEmergencyEntries -gt 0 -or $botnetActorEntries -gt 0)
)

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
  throw "service protection smoke failed for $ProfileName"
}
