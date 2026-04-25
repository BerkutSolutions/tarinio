param(
  [string]$PostgresPassword = "",
  [string]$SecurityPepper = "",
  [string]$RuntimeToken = "",
  [string]$BootstrapAdminPassword = "",
  [string]$OpenSearchPassword = ""
)

$ErrorActionPreference = "Stop"

function New-RandomSecret {
  param([int]$Length = 40)
  $chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_!@#$%^&*"
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  $bytes = New-Object byte[] $Length
  $rng.GetBytes($bytes)
  $out = New-Object System.Text.StringBuilder
  for ($i = 0; $i -lt $Length; $i++) {
    [void]$out.Append($chars[$bytes[$i] % $chars.Length])
  }
  $out.ToString()
}

if ([string]::IsNullOrWhiteSpace($PostgresPassword)) { $PostgresPassword = New-RandomSecret 28 }
if ([string]::IsNullOrWhiteSpace($SecurityPepper)) { $SecurityPepper = New-RandomSecret 48 }
if ([string]::IsNullOrWhiteSpace($RuntimeToken)) { $RuntimeToken = New-RandomSecret 48 }
if ([string]::IsNullOrWhiteSpace($BootstrapAdminPassword)) { $BootstrapAdminPassword = New-RandomSecret 24 }
if ([string]::IsNullOrWhiteSpace($OpenSearchPassword)) { $OpenSearchPassword = New-RandomSecret 28 }

$dsn = "postgres://waf:{0}@postgres:5432/waf?sslmode=disable" -f $PostgresPassword
$target = "deploy/lab-k8s-terraform/k8s/manifests/02-secrets.yaml"

$content = @"
apiVersion: v1
kind: Secret
metadata:
  name: tarinio-lab-secrets
type: Opaque
stringData:
  POSTGRES_PASSWORD: "$PostgresPassword"
  POSTGRES_DSN: "$dsn"
  CONTROL_PLANE_SECURITY_PEPPER: "$SecurityPepper"
  WAF_RUNTIME_API_TOKEN: "$RuntimeToken"
  CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD: "$BootstrapAdminPassword"
  OPENSEARCH_PASSWORD: "$OpenSearchPassword"
"@

Set-Content -LiteralPath $target -Value $content -Encoding utf8
Write-Host ("Generated {0}" -f $target) -ForegroundColor Green

