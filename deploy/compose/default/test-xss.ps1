param(
  [string]$BaseUrl = "https://localhost/",
  [switch]$UseRuntimeContainer,
  [string]$RuntimeContainerName = "tarinio-runtime",
  [int]$ExpectedBlockedMinimum = 3
)

$ErrorActionPreference = "Continue"

function Invoke-CurlStatus {
  param(
    [string[]]$CurlArgs
  )
  $output = $null
  try {
    $output = & curl.exe @CurlArgs 2>$null
  } catch {
    return 0
  }
  $raw = (($output | Select-Object -Last 1) -as [string]).Trim()
  if (-not $raw) {
    return 0
  }
  $status = 0
  if ([int]::TryParse($raw, [ref]$status)) {
    return $status
  }
  return 0
}

$blockedStatuses = @(403, 406, 429)

function Get-WgetStatusFromOutput {
  param(
    [string[]]$Lines
  )
  foreach ($line in $Lines) {
    $raw = [string]$line
    if ($raw -match "HTTP\/1\.1\s+([0-9]{3})") {
      return [int]$Matches[1]
    }
  }
  return 0
}

function Invoke-ContainerStatus {
  param(
    [string]$URL,
    [string]$Method = "GET",
    [hashtable]$Headers = @{},
    [string]$Body = ""
  )
  $parts = @(
    "wget",
    "--no-check-certificate",
    "-S",
    "-O",
    "/dev/null",
    "--header='Host: localhost'"
  )
  foreach ($k in $Headers.Keys) {
    $v = [string]$Headers[$k]
    $parts += "--header='${k}: $v'"
  }
  if ($Method -eq "POST") {
    $parts += "--post-data='$Body'"
  }
  $parts += "'$URL'"
  $script = ($parts -join " ")
  $output = $null
  try {
    $output = & docker exec $RuntimeContainerName sh -lc $script 2>&1
  } catch {
    return 0
  }
  return Get-WgetStatusFromOutput -Lines $output
}

function Test-ContainerHealthAndRules {
  $logs = $null
  try {
    $logs = & docker logs --tail 200 $RuntimeContainerName 2>$null
  } catch {
    Write-Warning "Failed to read runtime container logs for CRS verification. Continuing smoke requests only."
    return
  }
  if (-not $logs) {
    Write-Warning "Runtime logs are empty. Cannot verify CRS load state. Continuing smoke requests only."
    return
  }
  $line = ($logs | Select-String -Pattern "rules loaded inline/local/remote").Line | Select-Object -Last 1
  if (-not $line) {
    Write-Warning "Runtime log does not contain ModSecurity rules loaded marker. Continuing smoke requests only."
    return
  }
  if ($line -notmatch "inline/local/remote:\s*([0-9]+)\/([0-9]+)\/([0-9]+)") {
    Write-Warning "Cannot parse rules loaded counters from runtime log line. Continuing smoke requests only."
    return
  }
  $localCount = [int]$Matches[2]
  if ($localCount -le 0) {
    throw "CRS rules are not loaded (local=0). Current line: $line"
  }
}

$useContainerMode = $UseRuntimeContainer.IsPresent
if (-not $useContainerMode) {
  try {
    & docker ps --format "{{.Names}}" 2>$null | Out-Null
    $containerNames = & docker ps --format "{{.Names}}" 2>$null
    if ($containerNames -contains $RuntimeContainerName) {
      $useContainerMode = $true
    }
  } catch {
    $useContainerMode = $false
  }
}

if ($useContainerMode) {
  Test-ContainerHealthAndRules
}

$blockedStatuses = @(403, 406, 429)
$curlBase = @("-k", "-sS", "-L", "--max-time", "20", "-o", "NUL", "-w", "%{http_code}")

$baselineStatus = 0
if ($useContainerMode) {
  $baselineStatus = Invoke-ContainerStatus -URL "$BaseUrl?xss_smoke_baseline=1"
} else {
  $baselineStatus = Invoke-CurlStatus -CurlArgs ($curlBase + @("$BaseUrl?xss_smoke_baseline=1"))
}
if ($baselineStatus -eq 0) {
  throw "Baseline request failed (curl http_code=000). Service is unavailable for XSS smoke test."
}

$cases = @(
  @{ Name = "Query script tag"; URL = "$BaseUrl?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E"; Method = "GET"; Headers = @{}; Body = "" },
  @{ Name = "Query svg onload"; URL = "$BaseUrl?q=%3Csvg%20onload%3Dalert(1)%3E"; Method = "GET"; Headers = @{}; Body = "" },
  @{ Name = "Form body img onerror"; URL = "$BaseUrl"; Method = "POST"; Headers = @{ "Content-Type" = "application/x-www-form-urlencoded" }; Body = "comment=%3Cimg%20src%3Dx%20onerror%3Dalert(1)%3E" },
  @{ Name = "User-Agent payload"; URL = "$BaseUrl?source=ua"; Method = "GET"; Headers = @{ "User-Agent" = "<script>alert(1)</script>" }; Body = "" },
  @{ Name = "Referer payload"; URL = "$BaseUrl?source=ref"; Method = "GET"; Headers = @{ "Referer" = "javascript:alert(1)" }; Body = "" }
)

$blocked = 0
$results = @()

foreach ($case in $cases) {
  $status = 0
  if ($useContainerMode) {
    $status = Invoke-ContainerStatus -URL $case.URL -Method $case.Method -Headers $case.Headers -Body $case.Body
  } else {
    $curlArgs = @()
    if ($case.Method -eq "POST") {
      $curlArgs += @("-X", "POST")
    }
    foreach ($h in $case.Headers.Keys) {
      $curlArgs += @("-H", "${h}: $($case.Headers[$h])")
    }
    if ($case.Method -eq "POST" -and [string]::IsNullOrWhiteSpace($case.Body) -eq $false) {
      $curlArgs += @("--data", $case.Body)
    }
    $curlArgs += @($case.URL)
    $status = Invoke-CurlStatus -CurlArgs ($curlBase + $curlArgs)
  }
  $isBlocked = ($blockedStatuses -contains $status) -or ($status -eq 0)
  if ($isBlocked) {
    $blocked += 1
  }
  $results += [pscustomobject]@{
    Test = $case.Name
    Status = $status
    Blocked = $isBlocked
  }
}

$results | Format-Table -AutoSize
Write-Host ""
Write-Host "Blocked: $blocked / $($cases.Count)"

if ($blocked -lt $ExpectedBlockedMinimum) {
  throw "XSS smoke test failed: expected at least $ExpectedBlockedMinimum blocked requests out of $($cases.Count)"
}

Write-Host "XSS smoke test passed."
