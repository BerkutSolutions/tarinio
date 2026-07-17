param(
  [string]$Namespace = "tarinio-lab"
)

$ErrorActionPreference = "Stop"

function Require-Secret {
  param([string]$Name)
  kubectl -n $Namespace get secret $Name | Out-Null
}

function Require-ServicePort {
  param([string]$Name, [int]$Port)
  $ports = kubectl -n $Namespace get service $Name -o json | ConvertFrom-Json
  if (-not ($ports.spec.ports | Where-Object { $_.port -eq $Port })) {
    throw "Service $Name does not expose expected port $Port"
  }
}

function Reject-ServicePort {
  param([string]$Name, [int]$Port)
  $ports = kubectl -n $Namespace get service $Name -o json | ConvertFrom-Json
  if ($ports.spec.ports | Where-Object { $_.port -eq $Port }) {
    throw "Service $Name still exposes forbidden plaintext port $Port"
  }
}

foreach ($secret in @(
  "tarinio-lab-opensearch-tls",
  "tarinio-lab-opensearch-security",
  "tarinio-lab-clickhouse-tls",
  "tarinio-lab-clickhouse-users"
)) {
  Require-Secret $secret
}

foreach ($deployment in @("opensearch", "clickhouse", "control-plane", "runtime")) {
  kubectl -n $Namespace rollout status "deployment/$deployment" --timeout=300s
}

Require-ServicePort "opensearch" 9200
Require-ServicePort "clickhouse" 8443
Reject-ServicePort "clickhouse" 8123
Reject-ServicePort "clickhouse" 9000

$openSearch = kubectl -n $Namespace get deployment opensearch -o json | ConvertFrom-Json
$openSearchEnv = $openSearch.spec.template.spec.containers | Where-Object { $_.name -eq "opensearch" | Select-Object -ExpandProperty env
if ($openSearchEnv | Where-Object { $_.name -eq "DISABLE_SECURITY_PLUGIN" -and $_.value -eq "true" }) {
  throw "OpenSearch security plugin is disabled"
}
if (-not ($openSearch.spec.template.spec.volumes | Where-Object { $_.secret.secretName -eq "tarinio-lab-opensearch-tls" })) {
  throw "OpenSearch TLS secret is not mounted"
}

foreach ($policy in @("allow-opensearch-ingress", "allow-clickhouse-ingress")) {
  kubectl -n $Namespace get networkpolicy $policy | Out-Null
}

Write-Host "Secure OpenSearch/ClickHouse deployment shape verified." -ForegroundColor Green
Write-Host "Run authenticated HTTPS probes from a permitted runtime/control-plane pod using the existing credentials and CA; do not print or rotate them during verification." -ForegroundColor Yellow
