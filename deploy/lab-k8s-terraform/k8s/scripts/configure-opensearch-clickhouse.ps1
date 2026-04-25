param(
  [string]$Namespace = "tarinio-lab",
  [int]$ControlPlaneLocalPort = 18080
)

$ErrorActionPreference = "Stop"

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

$pf = Start-Process -FilePath "kubectl" -ArgumentList @("-n", $Namespace, "port-forward", "svc/control-plane", "$ControlPlaneLocalPort`:8080") -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 3

try {
  $baseUrl = "http://127.0.0.1:$ControlPlaneLocalPort"
  $adminPassword = Decode-SecretValue -Namespace $Namespace -SecretName "tarinio-lab-secrets" -Key "CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD"
  $opensearchPassword = Decode-SecretValue -Namespace $Namespace -SecretName "tarinio-lab-secrets" -Key "OPENSEARCH_PASSWORD"

  $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
  $loginBody = @{
    username = "admin"
    password = $adminPassword
  } | ConvertTo-Json -Depth 5
  $loginResp = Invoke-RestMethod -Uri "$baseUrl/api/auth/login" -Method Post -ContentType "application/json" -Body $loginBody -WebSession $session
  if ($loginResp.requires_2fa -eq $true) {
    throw "Admin user requires 2FA; cannot continue non-interactive profile configuration."
  }

  $payload = @{
    logging = @{
      backend = "opensearch"
      hot = @{
        backend = "opensearch"
      }
      cold = @{
        backend = "clickhouse"
      }
      retention = @{
        hot_days  = 7
        cold_days = 30
      }
      routing = @{
        write_requests_to_hot  = $true
        write_requests_to_cold = $true
        write_events_to_hot    = $true
        write_events_to_cold   = $false
        write_activity_to_hot  = $true
        write_activity_to_cold = $false
      }
      opensearch = @{
        endpoint       = "http://opensearch:9200"
        username       = "admin"
        password       = $opensearchPassword
        index_prefix   = "waf-hot"
        requests_index = "waf-requests"
        events_index   = "waf-events"
        activity_index = "waf-activity"
      }
      clickhouse = @{
        endpoint          = "http://clickhouse:8123"
        username          = "default"
        password          = ""
        database          = "waf_logs"
        table             = "request_logs"
        migration_enabled = $true
      }
      secret_provider = "file"
      vault = @{
        enabled = $false
      }
    }
  } | ConvertTo-Json -Depth 12

  $null = Invoke-RestMethod -Uri "$baseUrl/api/settings/runtime" -Method Put -ContentType "application/json" -Body $payload -WebSession $session
  $settings = Invoke-RestMethod -Uri "$baseUrl/api/settings/runtime" -Method Get -WebSession $session

  if ($settings.logging_summary.cold_backend -ne "clickhouse") {
    throw "Expected cold_backend=clickhouse after profile configuration."
  }
  if ($settings.logging_summary.hot_backend -ne "opensearch") {
    throw "Expected hot_backend=opensearch after profile configuration."
  }
  if (-not $settings.logging_summary.clickhouse_enabled) {
    throw "Expected clickhouse_enabled=true in logging_summary."
  }

  Write-Host "OpenSearch+ClickHouse profile configured." -ForegroundColor Green
}
finally {
  if ($pf -and -not $pf.HasExited) {
    Stop-Process -Id $pf.Id -Force
  }
}
