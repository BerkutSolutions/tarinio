param(
  [string]$BaseComposeFile = "deploy/compose/default/docker-compose.yml",
  [string]$OverlayComposeFile = "deploy/compose/k8s-lab/docker-compose.yml",
  [string]$K8sManifestFile = "deploy/compose/k8s-lab/manifests/lab-apps.yaml",
  [string]$Namespace = "waf-k8s-lab",
  [string]$AdminUsername = "admin",
  [string]$AdminEmail = "admin@example.test",
  [string]$AdminPassword = "admin",
  [string]$ResultDir = ".work/k8s-ingress-waf-smoke",
  [switch]$NoCleanup
)

$ErrorActionPreference = "Stop"

function New-FreeTcpPort {
  $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
  $listener.Start()
  try {
    return ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port
  } finally {
    $listener.Stop()
  }
}

function Invoke-Native {
  param(
    [Parameter(Mandatory = $true)][string]$Command,
    [Parameter(Mandatory = $true)][string[]]$Arguments
  )
  & $Command @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$Command failed: $($Arguments -join ' ')"
  }
}

function Invoke-NativeText {
  param(
    [Parameter(Mandatory = $true)][string]$Command,
    [Parameter(Mandatory = $true)][string[]]$Arguments
  )
  $output = & $Command @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$Command failed: $($Arguments -join ' ')"
  }
  return ($output -join "`n")
}

function Compose-Args {
  param([string[]]$Tail)
  return @(
    "compose",
    "-p", $script:ProjectName,
    "--env-file", $script:EnvFile,
    "-f", $BaseComposeFile,
    "-f", $OverlayComposeFile,
    "-f", $script:OverrideFile
  ) + $Tail
}

function Invoke-Compose {
  param([string[]]$Tail)
  Invoke-Native "docker" (Compose-Args $Tail)
}

function Invoke-ComposeText {
  param([string[]]$Tail)
  return Invoke-NativeText "docker" (Compose-Args $Tail)
}

function Invoke-CliJson {
  param(
    [string[]]$WafCliArgs,
    [switch]$NoAuth
  )
  $cli = @(
    "compose",
    "-p", $script:ProjectName,
    "--env-file", $script:EnvFile,
    "-f", $BaseComposeFile,
    "-f", $OverlayComposeFile,
    "-f", $script:OverrideFile,
    "--profile", "tools",
    "run", "--rm",
    "-v", "${script:TmpRootAbs}:/work",
    "cli",
    "--base-url", "http://control-plane:8080",
    "--username", $AdminUsername,
    "--password", $AdminPassword,
    "--json"
  )
  if ($NoAuth) {
    $cli += "--no-auth"
  }
  $cli += $WafCliArgs
  $raw = Invoke-NativeText "docker" $cli
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $null
  }
  return ($raw | ConvertFrom-Json)
}

function To-CliFilePath {
  param([Parameter(Mandatory = $true)][string]$HostFilePath)
  return "/work/$([IO.Path]::GetFileName($HostFilePath))"
}

function Write-JsonUtf8NoBom {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)]$Value,
    [int]$Depth = 20
  )
  $json = $Value | ConvertTo-Json -Depth $Depth
  [System.IO.File]::WriteAllText($Path, $json, [System.Text.UTF8Encoding]::new($false))
}

function Wait-ForControlPlane {
  param([int]$TimeoutSeconds = 180)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $lastError = ""
  while ((Get-Date) -lt $deadline) {
    try {
      [void](Invoke-CliJson -WafCliArgs @("setup", "status") -NoAuth)
      return
    } catch {
      $lastError = $_.Exception.Message
      if ($lastError -match "unknown command" -or $lastError -match "usage:" -or $lastError -match "invalid argument") {
        throw "control-plane readiness probe failed with non-retryable CLI error: $lastError"
      }
      Start-Sleep -Seconds 3
    }
  }
  if ([string]::IsNullOrWhiteSpace($lastError)) {
    throw "control-plane did not become ready within $TimeoutSeconds seconds"
  }
  throw "control-plane did not become ready within $TimeoutSeconds seconds. last error: $lastError"
}

function Ensure-BootstrapAdmin {
  $status = Invoke-CliJson -WafCliArgs @("setup", "status") -NoAuth
  $needsBootstrap = $false
  if ($null -ne $status -and $null -ne $status.needs_bootstrap) {
    $needsBootstrap = [bool]$status.needs_bootstrap
  }
  if (-not $needsBootstrap) {
    return
  }
  $payload = @{
    username = $AdminUsername
    email = $AdminEmail
    password = $AdminPassword
  }
  $file = Join-Path $script:TmpRoot "bootstrap.json"
  Write-JsonUtf8NoBom -Path $file -Value $payload -Depth 8
  $cliFile = To-CliFilePath -HostFilePath $file
  [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "POST", "/api/auth/bootstrap") -NoAuth)
}

function Wait-ForK3s {
  param([string]$K3sContainer, [int]$TimeoutSeconds = 420)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $lastError = ""
  $candidates = @(
    @{ Prefix = "kubectl"; Probe = "kubectl get nodes >/dev/null 2>&1" },
    @{ Prefix = "k3s kubectl"; Probe = "k3s kubectl get nodes >/dev/null 2>&1" }
  )
  while ((Get-Date) -lt $deadline) {
    try {
      $state = Invoke-NativeText "docker" @("inspect", "--format", "{{.State.Status}}", $K3sContainer)
      $state = $state.Trim()
      if ($state -eq "exited" -or $state -eq "dead") {
        $logs = Invoke-NativeText "docker" @("logs", "--tail", "80", $K3sContainer)
        throw "k3s container state is '$state'. last logs:`n$logs"
      }
    } catch {
      $lastError = $_.Exception.Message
    }
    $ready = $false
    foreach ($candidate in $candidates) {
      try {
        Invoke-Native "docker" @("exec", $K3sContainer, "sh", "-lc", [string]$candidate.Probe)
        $script:KubectlCommandPrefix = [string]$candidate.Prefix
        $ready = $true
        break
      } catch {
        $lastError = $_.Exception.Message
      }
    }
    if ($ready) {
      return
    }
    Start-Sleep -Seconds 5
  }
  if ([string]::IsNullOrWhiteSpace($lastError)) {
    throw "k3s did not become ready within $TimeoutSeconds seconds"
  }
  throw "k3s did not become ready within $TimeoutSeconds seconds. last error: $lastError"
}

function Ensure-Site {
  param([string]$SiteID, [string]$PrimaryHost)
  $sites = @(Invoke-CliJson -WafCliArgs @("api", "GET", "/api/sites"))
  $existing = $sites | Where-Object { [string]$_.id -eq $SiteID } | Select-Object -First 1
  $payload = @{
    id = $SiteID
    primary_host = $PrimaryHost
    enabled = $true
  }
  $file = Join-Path $script:TmpRoot ("site-" + $SiteID + ".json")
  Write-JsonUtf8NoBom -Path $file -Value $payload -Depth 8
  $cliFile = To-CliFilePath -HostFilePath $file
  if ($null -eq $existing) {
    [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "POST", "/api/sites?auto_apply=false"))
  } else {
    [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "PUT", "/api/sites/${SiteID}?auto_apply=false"))
  }
}

function Ensure-Upstream {
  param(
    [string]$UpstreamID,
    [string]$SiteID,
    [string]$UpstreamHost,
    [int]$Port
  )
  $upstreams = @(Invoke-CliJson -WafCliArgs @("api", "GET", "/api/upstreams"))
  $existing = $upstreams | Where-Object { [string]$_.id -eq $UpstreamID } | Select-Object -First 1
  $payload = @{
    id = $UpstreamID
    site_id = $SiteID
    host = $UpstreamHost
    port = $Port
    scheme = "http"
  }
  $file = Join-Path $script:TmpRoot ("upstream-" + $UpstreamID + ".json")
  Write-JsonUtf8NoBom -Path $file -Value $payload -Depth 8
  $cliFile = To-CliFilePath -HostFilePath $file
  if ($null -eq $existing) {
    [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "POST", "/api/upstreams?auto_apply=false"))
  } else {
    [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "PUT", "/api/upstreams/${UpstreamID}?auto_apply=false"))
  }
}

function Ensure-EasyProfile {
  param(
    [string]$SiteID,
    [string]$PublicHost,
    [string]$UpstreamHost,
    [int]$UpstreamPort
  )
  $profile = Invoke-CliJson -WafCliArgs @("easy", "get", $SiteID)
  $profile.front_service.server_name = $PublicHost
  $profile.upstream_routing.use_reverse_proxy = $true
  $profile.upstream_routing.reverse_proxy_host = "http://${UpstreamHost}:${UpstreamPort}"
  $profile.upstream_routing.reverse_proxy_url = "/"
  $profile.upstream_routing.reverse_proxy_custom_host = $PublicHost
  $profile.upstream_routing.reverse_proxy_ssl_sni = $false
  $profile.upstream_routing.reverse_proxy_ssl_sni_name = ""
  $profile.upstream_routing.reverse_proxy_keepalive = $true
  $profile.upstream_routing.disable_host_header = $false

  $file = Join-Path $script:TmpRoot ("easy-" + $SiteID + ".json")
  Write-JsonUtf8NoBom -Path $file -Value $profile -Depth 40
  $cliFile = To-CliFilePath -HostFilePath $file
  [void](Invoke-CliJson -WafCliArgs @("api", "--file", $cliFile, "PUT", "/api/easy-site-profiles/${SiteID}?auto_apply=false"))
}

function Compile-AndApply {
  $compiled = Invoke-CliJson -WafCliArgs @("revisions", "compile")
  $revisionID = [string]$compiled.revision.id
  if ([string]::IsNullOrWhiteSpace($revisionID)) {
    throw "failed to resolve revision id from compile response"
  }
  $applyJob = Invoke-CliJson -WafCliArgs @("revisions", "apply", $revisionID)
  $status = [string]$applyJob.status
  if ($status -eq "failed") {
    throw "revision apply failed for $revisionID"
  }
  return $revisionID
}

function Invoke-HostCheck {
  param(
    [string]$TargetHost,
    [int]$Port,
    [string]$ExpectedMarker,
    [int]$MaxAttempts = 40,
    [int]$DelaySeconds = 2
  )
  $lastError = ""
  for ($attempt = 1; $attempt -le $MaxAttempts; $attempt++) {
    try {
      $response = Invoke-NativeText "curl.exe" @(
        "-fs",
        "--max-time", "20",
        "-H", "Host: $TargetHost",
        "http://127.0.0.1:$Port/"
      )
      if ($response -match [Regex]::Escape($ExpectedMarker)) {
        return $response.Trim()
      }
      $lastError = "unexpected body marker"
    } catch {
      $lastError = $_.Exception.Message
    }
    if ($attempt -lt $MaxAttempts) {
      Start-Sleep -Seconds $DelaySeconds
    }
  }
  throw "traffic check failed for $TargetHost after $MaxAttempts attempts. expected marker: $ExpectedMarker. last error: $lastError"
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$rand = Get-Random -Minimum 1000 -Maximum 9999
$smokeID = "$timestamp-$rand"
$script:ProjectName = "waf-k8s-lab-$smokeID"
$script:TmpRoot = Join-Path ".work" ("k8s-waf-smoke-" + $smokeID)
$script:EnvFile = Join-Path $script:TmpRoot "stack.env"
$script:OverrideFile = Join-Path $script:TmpRoot "stack.override.yml"
$script:ResultFile = Join-Path $script:TmpRoot "result.json"

$runtimeHttpPort = New-FreeTcpPort
$runtimeHttpsPort = New-FreeTcpPort
$script:KubectlCommandPrefix = "kubectl"

New-Item -ItemType Directory -Force -Path $script:TmpRoot | Out-Null
New-Item -ItemType Directory -Force -Path $ResultDir | Out-Null
$script:TmpRootAbs = (Resolve-Path $script:TmpRoot).Path

Copy-Item -LiteralPath "deploy/compose/default/.env.example" -Destination $script:EnvFile -Force
Add-Content -LiteralPath $script:EnvFile -Value @"
WAF_STACK_NAME=$script:ProjectName
WAF_RUNTIME_HTTP_PORT=$runtimeHttpPort
WAF_RUNTIME_HTTPS_PORT=$runtimeHttpsPort
POSTGRES_PASSWORD=waf
POSTGRES_DSN=postgres://waf:waf@postgres:5432/waf?sslmode=disable
CONTROL_PLANE_SECURITY_PEPPER=k8s-lab-security-pepper
WAF_RUNTIME_API_TOKEN=k8s-lab-runtime-api-token
OPENSEARCH_PASSWORD=opensearch
CONTROL_PLANE_ACME_ENABLED=false
CONTROL_PLANE_BOOTSTRAP_ADMIN_ENABLED=true
CONTROL_PLANE_BOOTSTRAP_ADMIN_ID=admin
CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME=$AdminUsername
CONTROL_PLANE_BOOTSTRAP_ADMIN_EMAIL=$AdminEmail
CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD=$AdminPassword
"@

$override = @"
services:
  control-plane:
    container_name: tarinio-control-plane-$smokeID
  runtime:
    container_name: tarinio-runtime-$smokeID
  postgres:
    container_name: tarinio-postgres-$smokeID
  opensearch:
    container_name: tarinio-opensearch-$smokeID
  vault:
    container_name: tarinio-vault-$smokeID
  ui:
    container_name: tarinio-ui-$smokeID
  tarinio-sentinel:
    container_name: tarinio-sentinel-$smokeID
  cli:
    container_name: tarinio-cli-$smokeID
  k3s:
    container_name: tarinio-k3s-$smokeID

networks:
  waf-net:
    name: waf-net-$smokeID

volumes:
  waf-runtime-data:
    name: waf-runtime-data-$smokeID
  waf-control-plane-data:
    name: waf-control-plane-data-$smokeID
  waf-certificates-data:
    name: waf-certificates-data-$smokeID
  waf-postgres-data:
    name: waf-postgres-data-$smokeID
  waf-nginx-logs:
    name: waf-nginx-logs-$smokeID
  waf-request-archive-data:
    name: waf-request-archive-data-$smokeID
  waf-opensearch-data:
    name: waf-opensearch-data-$smokeID
  waf-vault-data:
    name: waf-vault-data-$smokeID
  waf-vault-bootstrap-data:
    name: waf-vault-bootstrap-data-$smokeID
  waf-l4-adaptive:
    name: waf-l4-adaptive-$smokeID
  waf-sentinel-state:
    name: waf-sentinel-state-$smokeID
  waf-k3s-data:
    name: waf-k3s-data-$smokeID
  waf-k3s-output:
    name: waf-k3s-output-$smokeID
"@
[System.IO.File]::WriteAllText($script:OverrideFile, $override, [System.Text.UTF8Encoding]::new($false))

$siteMatrix = @(
  @{
    site_id = "lab-example"
    upstream_id = "lab-example-up"
    host = "example.com"
    expected = "K8S-LAB SITE A: example.com"
  },
  @{
    site_id = "lab-domen-example"
    upstream_id = "lab-domen-example-up"
    host = "domen.example.com"
    expected = "K8S-LAB SITE B: domen.example.com"
  },
  @{
    site_id = "lab-ex-example"
    upstream_id = "lab-ex-example-up"
    host = "ex.example.com"
    expected = "K8S-LAB SITE C: ex.example.com"
  }
)

$revisions = New-Object System.Collections.Generic.List[string]
$checks = New-Object System.Collections.Generic.List[object]

try {
  Write-Host "[$script:ProjectName] starting isolated stack"
  Invoke-Compose @("up", "-d", "--build")

  Write-Host "[$script:ProjectName] waiting control-plane readiness"
  Wait-ForControlPlane
  Ensure-BootstrapAdmin

  $k3sContainer = Invoke-ComposeText @("ps", "-q", "k3s")
  $k3sContainer = $k3sContainer.Trim()
  if ([string]::IsNullOrWhiteSpace($k3sContainer)) {
    throw "cannot resolve k3s container id"
  }

  Write-Host "[$script:ProjectName] waiting k3s readiness"
  Wait-ForK3s -K3sContainer $k3sContainer
  Write-Host "[$script:ProjectName] kubectl command prefix: $script:KubectlCommandPrefix"

  Write-Host "[$script:ProjectName] applying kubernetes manifests"
  $manifest = Get-Content -Raw $K8sManifestFile
  $manifest | & docker exec -i $k3sContainer sh -lc "$script:KubectlCommandPrefix apply -f -"
  if ($LASTEXITCODE -ne 0) {
    throw "failed to apply kubernetes manifests"
  }
  Invoke-Native "docker" @("exec", $k3sContainer, "sh", "-lc", "$script:KubectlCommandPrefix -n $Namespace rollout status deploy/site-a --timeout=180s")
  Invoke-Native "docker" @("exec", $k3sContainer, "sh", "-lc", "$script:KubectlCommandPrefix -n $Namespace rollout status deploy/site-b --timeout=180s")
  Invoke-Native "docker" @("exec", $k3sContainer, "sh", "-lc", "$script:KubectlCommandPrefix -n $Namespace rollout status deploy/site-c --timeout=180s")

  foreach ($item in $siteMatrix) {
    $siteID = [string]$item.site_id
    $upstreamID = [string]$item.upstream_id
    $targetHost = [string]$item.host

    Write-Host "[$script:ProjectName] upserting site stack for $targetHost"
    Ensure-Site -SiteID $siteID -PrimaryHost $targetHost
    Ensure-Upstream -UpstreamID $upstreamID -SiteID $siteID -UpstreamHost "k3s" -Port 80

    $revisionID = Compile-AndApply
    $revisions.Add($revisionID) | Out-Null
    Start-Sleep -Seconds 2
  }

  Write-Host "[$script:ProjectName] validating traffic via WAF"
  foreach ($item in $siteMatrix) {
    $body = Invoke-HostCheck -TargetHost ([string]$item.host) -Port $runtimeHttpPort -ExpectedMarker ([string]$item.expected)
    $checks.Add([ordered]@{
      host = [string]$item.host
      expected = [string]$item.expected
      response = $body
      ok = $true
    }) | Out-Null
  }

  $result = @{
    smoke_id = $smokeID
    project = $script:ProjectName
    runtime_http_port = $runtimeHttpPort
    runtime_https_port = $runtimeHttpsPort
    revisions = $revisions.ToArray()
    checks = $checks.ToArray()
    passed = $true
    cleanup_enabled = (-not $NoCleanup)
    generated_at = (Get-Date).ToUniversalTime().ToString("o")
  }
  Write-JsonUtf8NoBom -Path $script:ResultFile -Value $result -Depth 10
  $result | ConvertTo-Json -Depth 10

  $resultName = "result-$smokeID.json"
  Copy-Item -LiteralPath $script:ResultFile -Destination (Join-Path $ResultDir $resultName) -Force
}
finally {
  if (-not $NoCleanup) {
    Write-Host "[$script:ProjectName] cleaning up stack and volumes"
    try {
      Invoke-Compose @("down", "-v", "--remove-orphans")
    } catch {
      Write-Warning $_.Exception.Message
    }
    Remove-Item -LiteralPath $script:TmpRoot -Recurse -Force -ErrorAction SilentlyContinue
  } else {
    Write-Host "[$script:ProjectName] cleanup skipped (--NoCleanup)"
  }
}
