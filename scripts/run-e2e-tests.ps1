[CmdletBinding()]
param(
    [string]$Filter = "TestE2E",
    [int]$TimeoutSeconds = 180,
    [switch]$KeepStack
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repoRoot "deploy/compose/e2e/docker-compose.yml"
$logDir = Join-Path $repoRoot ".work/logs"
$logFile = Join-Path $logDir ("e2e-{0}.log" -f (Get-Date -Format "yyyyMMdd_HHmmss"))
$baseUrl = "http://127.0.0.1:18080"
$runtimeUrl = "http://127.0.0.1:10080"
$runtimeHealthUrl = "http://127.0.0.1:18081"
$user = "e2e-admin"
$password = "e2e-password-1234"
$stackStarted = $false

New-Item -ItemType Directory -Path $logDir -Force | Out-Null

function Invoke-Compose {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    $previousErrorActionPreference = $ErrorActionPreference
    try {
        # Docker Compose writes normal progress messages to stderr on Windows.
        # Do not turn those messages into terminating PowerShell errors; use its exit code.
        $ErrorActionPreference = "Continue"
        & docker compose -f $composeFile @Arguments 2>&1 | Tee-Object -FilePath $logFile -Append
        $composeExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($composeExitCode -ne 0) {
        throw "docker compose $($Arguments -join ' ') failed; see $logFile"
    }
}

function Wait-ForUrl {
    param([string]$Url, [string]$Name)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        try {
            Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5 | Out-Null
            Write-Host "[e2e] $Name is ready"
            return
        } catch {
            Start-Sleep -Seconds 2
        }
    } while ((Get-Date) -lt $deadline)

    throw "$Name did not become ready within $TimeoutSeconds seconds"
}

try {
    & docker compose version | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose is required"
    }

    Invoke-Compose -Arguments @("down", "--volumes", "--remove-orphans")
    Invoke-Compose -Arguments @("up", "-d", "--build")
    $stackStarted = $true

    Wait-ForUrl "$baseUrl/healthz" "control-plane"

    $loginBody = @{ username = $user; password = $password } | ConvertTo-Json -Compress
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    Invoke-WebRequest -Uri "$baseUrl/api/auth/login" -Method Post -ContentType "application/json" -Body $loginBody -WebSession $session -UseBasicParsing | Out-Null
    Wait-ForUrl "$runtimeHealthUrl/healthz" "runtime"

    $compile = Invoke-RestMethod -Uri "$baseUrl/api/revisions/compile" -Method Post -ContentType "application/json" -Body "{}" -WebSession $session
    $revisionId = if ($compile.revision_id) {
        $compile.revision_id
    } elseif ($compile.id) {
        $compile.id
    } elseif ($compile.revision -and $compile.revision.id) {
        $compile.revision.id
    }
    if (-not $revisionId) {
        throw "compile did not return a revision ID: $($compile | ConvertTo-Json -Compress)"
    }
    try {
        Invoke-RestMethod -Uri "$baseUrl/api/revisions/$revisionId/apply" -Method Post -ContentType "application/json" -Body "{}" -WebSession $session | Out-Null
    } catch {
        # Match the POSIX runner: the initial apply is a warm-up. The e2e cases
        # perform their own compile/apply after runtime readiness settles.
        Write-Warning "Initial apply did not complete: $($_.Exception.Message)"
    }
    Start-Sleep -Seconds 5

    $env:WAF_E2E_BASE_URL = $baseUrl
    $env:WAF_E2E_USERNAME = $user
    $env:WAF_E2E_PASSWORD = $password
    $env:WAF_E2E_RUNTIME_URL = $runtimeUrl
    $env:WAF_E2E_RUNTIME_HTTPS_URL = "https://127.0.0.1:10443"
    $env:WAF_E2E_RUNTIME_HEALTH_URL = $runtimeHealthUrl
    $env:WAF_E2E_RUNTIME_API_TOKEN = "e2e-test-runtime-token"

    Push-Location $repoRoot
    try {
        & go test ./ui/tests -run $Filter -count=1 -v 2>&1 | Tee-Object -FilePath $logFile -Append
        if ($LASTEXITCODE -ne 0) {
            throw "e2e tests failed; see $logFile"
        }
    } finally {
        Pop-Location
    }
} finally {
    if ($stackStarted -and -not $KeepStack) {
        Invoke-Compose -Arguments @("down", "--volumes", "--remove-orphans")
    }
}
