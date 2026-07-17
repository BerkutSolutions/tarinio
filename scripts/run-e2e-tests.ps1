[CmdletBinding()]
param(
    [string]$Filter = "TestE2E",
    [int]$TimeoutSeconds = 180,
    [switch]$KeepStack,
    [switch]$FreshOnboarding
)

$ErrorActionPreference = "Stop"
# PowerShell 7 can promote stderr written by a successful native process to an
# error record even when ErrorActionPreference is temporarily Continue. Docker
# BuildKit uses stderr for ordinary progress, so keep native output nonfatal and
# validate docker/go commands exclusively through their exit codes below.
$PSNativeCommandUseErrorActionPreference = $false
$repoRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repoRoot "deploy/compose/e2e/docker-compose.yml"
$logDir = Join-Path $repoRoot ".work/logs"
$logFile = Join-Path $logDir ("e2e-{0}.log" -f (Get-Date -Format "yyyyMMdd_HHmmss"))
$baseUrl = "http://127.0.0.1:18080"
$controlPlaneUrl = "http://127.0.0.1:18082"
$runtimeUrl = "http://127.0.0.1:10080"
$runtimeHealthUrl = "http://127.0.0.1:18081"
$user = "e2e-admin"
$password = "e2e-password-1234"
$stackStarted = $false

New-Item -ItemType Directory -Path $logDir -Force | Out-Null

function Invoke-Compose {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    $outputFile = Join-Path $logDir ("docker-compose-{0}.out" -f [guid]::NewGuid().ToString("N"))
    $errorFile = Join-Path $logDir ("docker-compose-{0}.err" -f [guid]::NewGuid().ToString("N"))
    try {
        # Docker Compose writes ordinary BuildKit progress to stderr. Capture
        # both streams outside PowerShell's native-error pipeline and decide
        # success solely from the process exit code.
        $composeArguments = @("compose", "-f", $composeFile) + $Arguments
        $process = Start-Process -FilePath "docker" -ArgumentList $composeArguments -NoNewWindow -Wait -PassThru -RedirectStandardOutput $outputFile -RedirectStandardError $errorFile
        Get-Content -LiteralPath $outputFile, $errorFile -ErrorAction SilentlyContinue | Tee-Object -FilePath $logFile -Append
        $composeExitCode = $process.ExitCode
    } finally {
        Remove-Item -LiteralPath $outputFile, $errorFile -Force -ErrorAction SilentlyContinue
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
    if ($FreshOnboarding) {
        $env:E2E_BOOTSTRAP_ADMIN_ENABLED = "false"
        $env:E2E_DEV_FAST_START_ENABLED = "false"
        $env:E2E_RUNTIME_STARTUP_BUNDLE_WAIT_SECONDS = "0"
    }

    & docker compose version | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose is required"
    }

    Invoke-Compose -Arguments @("down", "--volumes", "--remove-orphans")
    Invoke-Compose -Arguments @("up", "-d", "--build")
    $stackStarted = $true

    Wait-ForUrl "$controlPlaneUrl/healthz" "control-plane"
    Wait-ForUrl "$baseUrl/login" "ui"

    if (-not $FreshOnboarding) {
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
    }

    $env:WAF_E2E_BASE_URL = $baseUrl
    $env:WAF_E2E_USERNAME = $user
    $env:WAF_E2E_PASSWORD = $password
    $env:WAF_E2E_RUNTIME_URL = $runtimeUrl
    $env:WAF_E2E_RUNTIME_HTTPS_URL = "https://127.0.0.1:10443"
    $env:WAF_E2E_RUNTIME_HEALTH_URL = $runtimeHealthUrl
    $env:WAF_E2E_RUNTIME_API_TOKEN = "e2e-test-runtime-token"
    $env:WAF_E2E_MANAGEMENT_HOST = "e2e-management.test"
    $env:WAF_E2E_ANTIBOT_HOST = "e2e-antibot.test"
    $env:WAF_E2E_FRESH_ONBOARDING = if ($FreshOnboarding) { "1" } else { "" }

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
