param(
    [switch]$SkipGoTest,
    [switch]$SkipDocs,
    [switch]$SkipSmoke,
    [switch]$KeepArtifacts
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Step([string]$Title, [scriptblock]$Action) {
    Write-Host ""
    Write-Host "== $Title =="
    & $Action
}

function Invoke-Native([string]$FilePath, [string[]]$Arguments = @()) {
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        $argsText = ($Arguments -join " ")
        throw "command failed ($LASTEXITCODE): $FilePath $argsText"
    }
}

function Wait-UiReady([int]$Attempts = 60, [int]$DelaySeconds = 2) {
    for ($i = 1; $i -le $Attempts; $i++) {
        try {
            & curl.exe -kfsS --resolve "localhost:443:127.0.0.1" "https://localhost/login" | Out-Null
            return
        } catch {
            Start-Sleep -Seconds $DelaySeconds
        }
    }
    throw "ui login endpoint did not become ready in time"
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$composeDir = Join-Path $repoRoot "deploy\compose\auto-start"
$composeFile = Join-Path $composeDir "docker-compose.yml"
$envFile = Join-Path $composeDir ".env"
$envBackup = "$envFile.preflight-backup"
$createdTempEnv = $false
$releaseVersion = "preflight-local-2.0.11"
$releaseDir = Join-Path $repoRoot ("build\release\" + $releaseVersion)
$smokeContainers = @(
    "tarinio-control-plane",
    "tarinio-runtime",
    "tarinio-ui",
    "tarinio-postgres",
    "tarinio-redis",
    "tarinio-ddos-model"
)

Push-Location $repoRoot
try {
    if (-not $SkipGoTest) {
        Step "Go tests (full)" {
            Invoke-Native "go" @("test", "./...", "-count=1")
        }
    }

    Step "Release artifacts contract (local)" {
        if (Test-Path $releaseDir) {
            Remove-Item -LiteralPath $releaseDir -Recurse -Force
        }
        Invoke-Native "go" @(
            "run", "./cmd/release-artifacts",
            "-repo-root", ".",
            "-version", $releaseVersion,
            "-commit", "local-preflight",
            "-tag", "preflight-local",
            "-docker-tags", "local/tarinio:$releaseVersion"
        )
    }

    if (-not $SkipDocs) {
        Step "Docs and lockfile install" {
            $localNpmCache = Join-Path $repoRoot ".tmp\npm-cache"
            New-Item -ItemType Directory -Path $localNpmCache -Force | Out-Null
            $prevNpmCache = $env:npm_config_cache
            $env:npm_config_cache = $localNpmCache
            try {
                Invoke-Native "npm.cmd" @("cache", "clean", "--force")
            } catch {
                Write-Warning "npm cache clean failed, continuing: $($_.Exception.Message)"
            }
            try {
                $installed = $false
                for ($attempt = 1; $attempt -le 3; $attempt++) {
                    try {
                        Invoke-Native "npm.cmd" @("ci", "--prefer-online")
                        $installed = $true
                        break
                    } catch {
                        Write-Warning "npm ci failed on attempt ${attempt}: $($_.Exception.Message)"
                        try {
                            Invoke-Native "npm.cmd" @("cache", "verify")
                        } catch {
                            Write-Warning "npm cache verify failed, continuing"
                        }
                        Start-Sleep -Seconds (5 * $attempt)
                    }
                }
                if (-not $installed) {
                    throw "npm ci failed after retries"
                }
                Invoke-Native "npm.cmd" @("run", "docs:build")
            } finally {
                if ($null -eq $prevNpmCache) {
                    Remove-Item Env:\npm_config_cache -ErrorAction SilentlyContinue
                } else {
                    $env:npm_config_cache = $prevNpmCache
                }
            }
        }
    }

    if (-not $SkipSmoke) {
        Step "Prepare smoke env" {
            if (Test-Path $envBackup) {
                Remove-Item -LiteralPath $envBackup -Force
            }
            if (Test-Path $envFile) {
                Copy-Item -LiteralPath $envFile -Destination $envBackup -Force
            }
            Copy-Item -LiteralPath (Join-Path $composeDir ".env.example") -Destination $envFile -Force
            Add-Content -LiteralPath $envFile -Value @"
POSTGRES_PASSWORD=waf
POSTGRES_DSN=postgres://waf:waf@postgres:5432/waf?sslmode=disable
CONTROL_PLANE_SECURITY_PEPPER=preflight-security-pepper
WAF_RUNTIME_API_TOKEN=preflight-runtime-api-token
CONTROL_PLANE_BOOTSTRAP_ADMIN_ENABLED=true
CONTROL_PLANE_BOOTSTRAP_ADMIN_ID=admin
CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME=admin
CONTROL_PLANE_BOOTSTRAP_ADMIN_EMAIL=admin@example.test
CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD=admin
CONTROL_PLANE_ACME_ENABLED=false
CONTROL_PLANE_ACME_USE_DEVELOPMENT_CLIENT=true
CONTROL_PLANE_DEV_FAST_START_ENABLED=true
CONTROL_PLANE_DEV_FAST_START_HOST=localhost
CONTROL_PLANE_DEV_FAST_START_CERTIFICATE_ID=control-plane-localhost-tls
CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID=control-plane-access
CONTROL_PLANE_DEV_FAST_START_UPSTREAM_HOST=ui
CONTROL_PLANE_DEV_FAST_START_UPSTREAM_PORT=80
"@
            $createdTempEnv = $true
        }

        $existingContainers = @(& docker ps -a --format "{{.Names}}")
        $conflicts = @($smokeContainers | Where-Object { $existingContainers -contains $_ })
        $canProvisionSmokeStack = ($conflicts.Count -eq 0)
        if (-not $canProvisionSmokeStack) {
            Write-Warning ("Skipping compose up/down: conflicting containers already exist: " + ($conflicts -join ", "))
            Write-Warning "Running smoke e2e checks against the currently running stack."
        }

        try {
            if ($canProvisionSmokeStack) {
                Step "Compose clean" {
                    Push-Location $composeDir
                    try {
                        Invoke-Native "docker" @("compose", "-f", $composeFile, "down", "-v", "--remove-orphans")
                    } finally {
                        Pop-Location
                    }
                }

                Step "Compose up --build" {
                    Push-Location $composeDir
                    try {
                        Invoke-Native "docker" @("compose", "-f", $composeFile, "up", "-d", "--build")
                    } finally {
                        Pop-Location
                    }
                }
            }

            Step "Wait UI readiness" {
                Wait-UiReady
            }

            Step "Run smoke e2e test" {
                $env:WAF_E2E_BASE_URL = "https://localhost"
                $env:WAF_E2E_USERNAME = "admin"
                $env:WAF_E2E_PASSWORD = "admin"
                Invoke-Native "go" @("test", "./ui/tests", "-run", "TestE2ESmoke_LoginHealthcheckDashboard", "-count=1", "-v")
            }
        } finally {
            if ($canProvisionSmokeStack) {
                Step "Compose teardown" {
                    Push-Location $composeDir
                    try {
                        & docker compose -f $composeFile ps
                        & docker compose -f $composeFile down -v --remove-orphans
                    } finally {
                        Pop-Location
                    }
                }
            }

            if ($createdTempEnv) {
                if (Test-Path $envBackup) {
                    Move-Item -LiteralPath $envBackup -Destination $envFile -Force
                } else {
                    Remove-Item -LiteralPath $envFile -Force -ErrorAction SilentlyContinue
                }
            }
        }
    }

    if ((-not $KeepArtifacts) -and (Test-Path $releaseDir)) {
        Step "Cleanup release artifacts" {
            Remove-Item -LiteralPath $releaseDir -Recurse -Force
        }
    }

    Write-Host ""
    Write-Host "Local CI preflight passed."
} finally {
    Pop-Location
}

