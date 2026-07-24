[CmdletBinding()]
param(
    [string]$Filter = "TestE2E",
    [int]$TimeoutSeconds = 180,
    [string]$ProjectName = "e2e",
    [int]$UIPort = 18080,
    [int]$RuntimePort = 10080,
    [int]$RuntimeHTTPSPort = 10443,
    [int]$RuntimeHealthPort = 18081,
    [int]$ControlPlanePort = 18082,
    [int]$DASTCanaryPort = 18083,
    [int]$MTLSUpstreamPort = 18084,
    [switch]$KeepStack,
    [switch]$FreshOnboarding,
    [switch]$Browser,
    [switch]$BrowserOnly,
    [switch]$RequestsBackendFault,
    [string[]]$BrowserSpecs = @()
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
$baseUrl = "http://127.0.0.1:$UIPort"
$controlPlaneUrl = "http://127.0.0.1:$ControlPlanePort"
$runtimeUrl = "http://127.0.0.1:$RuntimePort"
$runtimeHealthUrl = "http://127.0.0.1:$RuntimeHealthPort"
$user = "e2e-admin"
$password = [string]$env:WAF_E2E_PASSWORD
$env:E2E_PASS = $password
$env:COMPOSE_PROJECT_NAME = $ProjectName
$env:E2E_PORT = "$UIPort"
$env:E2E_RT_PORT = "$RuntimePort"
$env:E2E_RT_HTTPS_PORT = "$RuntimeHTTPSPort"
$env:E2E_RT_HLT_PORT = "$RuntimeHealthPort"
$env:E2E_CONTROL_PLANE_DIRECT_PORT = "$ControlPlanePort"
$env:E2E_DAST_CANARY_PORT = "$DASTCanaryPort"
$stackStarted = $false

if ([string]::IsNullOrWhiteSpace($ProjectName) -or $ProjectName -notmatch '^[a-z0-9][a-z0-9_-]*$') {
    throw "ProjectName must use lowercase letters, digits, '_' or '-' and start with a letter or digit"
}
if ($MTLSUpstreamPort -lt 1 -or $MTLSUpstreamPort -gt 65535) {
    throw "MTLSUpstreamPort must be a valid TCP port"
}
$env:E2E_MTLS_UPSTREAM_PORT = [string]$MTLSUpstreamPort

if ([string]::IsNullOrWhiteSpace($password)) {
    throw "WAF_E2E_PASSWORD is required; browser/Go E2E credentials must not be embedded in source"
}
if ($BrowserOnly -and -not $Browser) {
    throw "BrowserOnly requires Browser"
}

New-Item -ItemType Directory -Path $logDir -Force | Out-Null

function Invoke-Compose {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    $outputFile = Join-Path $logDir ("docker-compose-{0}.out" -f [guid]::NewGuid().ToString("N"))
    $errorFile = Join-Path $logDir ("docker-compose-{0}.err" -f [guid]::NewGuid().ToString("N"))
    try {
        # Docker Compose writes ordinary BuildKit progress to stderr. Capture
        # both streams outside PowerShell's native-error pipeline and decide
        # success solely from the process exit code.
        $composeArguments = @("compose", "-p", $ProjectName, "-f", $composeFile) + $Arguments
        $previousErrorActionPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = "Continue"
            & docker @composeArguments 1> $outputFile 2> $errorFile
            $composeExitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousErrorActionPreference
        }
        Get-Content -LiteralPath $outputFile, $errorFile -ErrorAction SilentlyContinue | Tee-Object -FilePath $logFile -Append
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
    $env:WAF_E2E_CONTROL_PLANE_URL = $controlPlaneUrl
    $env:WAF_E2E_USERNAME = $user
    $env:WAF_E2E_PASSWORD = $password
    $env:WAF_E2E_RUNTIME_URL = $runtimeUrl
    $env:WAF_E2E_RUNTIME_HTTPS_URL = "https://127.0.0.1:$RuntimeHTTPSPort"
    $env:WAF_E2E_MTLS_FIXTURE_URL = "http://127.0.0.1:$MTLSUpstreamPort"
    $env:WAF_E2E_DAST_CANARY_URL = "http://127.0.0.1:$DASTCanaryPort"
    $env:WAF_E2E_RUNTIME_HEALTH_URL = $runtimeHealthUrl
    # Compose names the service container with the project prefix; expose the
    # actual disposable runtime container to Go artifact assertions.
    $env:WAF_E2E_RUNTIME_CONTAINER = "$ProjectName-runtime-1"
    $env:WAF_E2E_CONTROL_PLANE_CONTAINER = "$ProjectName-control-plane-1"
    $env:WAF_E2E_ATTACKER_CONTAINER = "$ProjectName-e2e-attacker-1"
    $env:WAF_E2E_L4_ATTACKER_CONTAINER = "$ProjectName-e2e-l4-attacker-1"
    $env:WAF_E2E_ATTACKER_IP = (& docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $env:WAF_E2E_ATTACKER_CONTAINER).Trim()
    $env:WAF_E2E_L4_ATTACKER_IP = (& docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $env:WAF_E2E_L4_ATTACKER_CONTAINER).Trim()
    $env:WAF_E2E_RUNTIME_API_TOKEN = "e2e-test-runtime-token"
    $env:WAF_E2E_COMPOSE_FILE = $composeFile
    $env:WAF_E2E_MANAGEMENT_HOST = "e2e-management.test"
    $env:WAF_E2E_AUTH_BASE_URL = $runtimeUrl
    # The antibot scenario provisions a TLS binding; exercising it over HTTPS
    # proves the same virtual host that production clients use.
    $env:WAF_E2E_ANTIBOT_BASE_URL = $env:WAF_E2E_RUNTIME_HTTPS_URL
    $env:WAF_E2E_AUTOSTART_SMART = "1"
    $env:WAF_E2E_L4_L7_PROTECTION = "1"
    $env:WAF_E2E_ANTIBOT_HOST = "e2e-antibot.test"
    $env:WAF_E2E_FRESH_ONBOARDING = if ($FreshOnboarding) { "1" } else { "" }
    $env:WAF_E2E_DISPOSABLE = "1"
    $env:WAF_BROWSER_BASE_URL = "https://e2e-management.test:$RuntimeHTTPSPort"

    Push-Location $repoRoot
    try {
        if (-not $BrowserOnly) {
            & go test -tags=e2e ./ui/tests -run $Filter -count=1 -v 2>&1 | Tee-Object -FilePath $logFile -Append
            if ($LASTEXITCODE -ne 0) {
                throw "e2e tests failed; see $logFile"
            }
            if (Select-String -LiteralPath $logFile -Pattern '^--- SKIP:' -Quiet) {
                throw "e2e tests skipped one or more scenarios; skipped tests are not accepted as proof; see $logFile"
            }
        }
        if ($Browser) {
            $selectedBrowserSpecs = @($BrowserSpecs | ForEach-Object { $_ -split '[,\s]+' } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
            if ($selectedBrowserSpecs -match '(?:dashboard|requests-complete)\.spec\.ts$') {
                & (Join-Path $repoRoot "scripts/seed-dashboard-telemetry.ps1") -ComposeFile $composeFile -VerifyControlPlaneURL $baseUrl -Username $user -Password $password
                if ($LASTEXITCODE -ne 0) {
                    throw "real telemetry seed failed; see $logFile"
                }
            }
            $runtimePaused = $false
            if ($RequestsBackendFault) {
                Push-Location (Join-Path $repoRoot "e2e\browser")
                try {
                    & node (Join-Path $repoRoot "e2e\browser\node_modules\@playwright\test\cli.js") test --project=setup 2>&1 | Tee-Object -FilePath $logFile -Append
                    if ($LASTEXITCODE -ne 0) { throw "browser auth setup before requests backend fault failed; see $logFile" }
                }
                finally {
                    Pop-Location
                }
            }
            Push-Location (Join-Path $repoRoot "e2e/browser")
            try {
                $browserEvidenceDirectory = Join-Path $repoRoot ("test-results\" + $ProjectName + "\browser")
                New-Item -ItemType Directory -Path $browserEvidenceDirectory -Force | Out-Null
                $env:WAF_BROWSER_RESULTS_FILE = Join-Path $browserEvidenceDirectory "results.json"
                $env:WAF_BROWSER_OUTPUT_DIR = Join-Path $browserEvidenceDirectory "artifacts"
                $faultSyncFile = Join-Path $browserEvidenceDirectory "requests-backend-fault.sync"
                if ($RequestsBackendFault) {
                    $env:WAF_BROWSER_FAULT_SYNC_FILE = $faultSyncFile
                }
                $npmArguments = @("run", "test", "--")
                $npmArguments += $selectedBrowserSpecs
                if ($RequestsBackendFault) {
                    foreach ($browserProject in @("desktop", "mobile")) {
                        $projectSyncFile = "$faultSyncFile.$browserProject"
                        $browserOutputFile = Join-Path $browserEvidenceDirectory "requests-backend-$browserProject-playwright.out"
                        $browserErrorFile = Join-Path $browserEvidenceDirectory "requests-backend-$browserProject-playwright.err"
                        Remove-Item -LiteralPath "$projectSyncFile.ready", ($faultSyncFile + ".paused"), $browserOutputFile, $browserErrorFile -Force -ErrorAction SilentlyContinue
                        $projectNpmArguments = @("run", "test", "--", "--project=$browserProject", "--no-deps") + $selectedBrowserSpecs
                        $browserProcess = Start-Process -FilePath "npm.cmd" -ArgumentList $projectNpmArguments -NoNewWindow -RedirectStandardOutput $browserOutputFile -RedirectStandardError $browserErrorFile -PassThru
                        try {
                            $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
                            do {
                                $browserProcess.Refresh()
                                if ($browserProcess.HasExited) {
                                    Get-Content -LiteralPath $browserOutputFile, $browserErrorFile -ErrorAction SilentlyContinue | Tee-Object -FilePath $logFile -Append
                                    throw "browser $browserProject fault test exited before the Requests ready barrier (exit code $($browserProcess.ExitCode))"
                                }
                                if (Test-Path -LiteralPath "$projectSyncFile.ready") { break }
                                Start-Sleep -Milliseconds 250
                            } while ((Get-Date) -lt $deadline)
                            if (-not (Test-Path -LiteralPath "$projectSyncFile.ready")) {
                                throw "browser $browserProject fault test did not reach the Requests ready barrier within $TimeoutSeconds seconds"
                            }
                            & docker pause $env:WAF_E2E_RUNTIME_CONTAINER
                            if ($LASTEXITCODE -ne 0) { throw "pause runtime for browser $browserProject requests backend fault failed" }
                            $runtimePaused = $true
                            New-Item -ItemType File -Path ($faultSyncFile + ".paused") -Force | Out-Null
                            $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
                            do {
                                $browserProcess.Refresh()
                                if ($browserProcess.HasExited) { break }
                                Start-Sleep -Milliseconds 250
                            } while ((Get-Date) -lt $deadline)
                            if (-not $browserProcess.HasExited) {
                                Stop-Process -Id $browserProcess.Id -Force -ErrorAction SilentlyContinue
                                throw "browser $browserProject fault test did not finish within $TimeoutSeconds seconds"
                            }
                            Get-Content -LiteralPath $browserOutputFile, $browserErrorFile -ErrorAction SilentlyContinue | Tee-Object -FilePath $logFile -Append
                            if ($browserProcess.ExitCode -ne 0) {
                                throw "browser $browserProject fault test failed with exit code $($browserProcess.ExitCode); see $logFile"
                            }
                        } finally {
                            if ($runtimePaused) {
                                & docker unpause $env:WAF_E2E_RUNTIME_CONTAINER
                                if ($LASTEXITCODE -ne 0) { throw "unpause runtime after browser $browserProject requests backend fault failed" }
                                $runtimePaused = $false
                            }
                            Remove-Item -LiteralPath "$projectSyncFile.ready", ($faultSyncFile + ".paused") -Force -ErrorAction SilentlyContinue
                        }
                    }
                } else {
                    & npm.cmd @npmArguments 2>&1 | Tee-Object -FilePath $logFile -Append
                }
                if ($LASTEXITCODE -ne 0) {
                    throw "browser e2e tests failed; see $logFile"
                }
            } finally {
                Pop-Location
                if ($runtimePaused) {
                    & docker unpause $env:WAF_E2E_RUNTIME_CONTAINER
                    if ($LASTEXITCODE -ne 0) { throw "unpause runtime after requests backend fault failed" }
                }
            }
        }
    } finally {
        Pop-Location
    }
} finally {
    if ($stackStarted -and -not $KeepStack) {
        Invoke-Compose -Arguments @("down", "--volumes", "--remove-orphans")
    }
}
