param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Version([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value)) {
        throw "Version is required"
    }
    if ($Value -notmatch '^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z\.-]+)?$') {
        throw "Invalid version format: $Value"
    }
}

function Write-Utf8NoBom([string]$Path, [string]$Content) {
    $encoding = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText($Path, $Content, $encoding)
}

function Update-AppMeta([string]$RepoRoot, [string]$TargetVersion) {
    $metaPath = Join-Path $RepoRoot "control-plane/internal/appmeta/meta.go"
    if (-not (Test-Path $metaPath)) {
        throw "meta.go not found: $metaPath"
    }
    $raw = Get-Content $metaPath -Raw
    $match = [regex]::Match($raw, 'var AppVersion = "([^"]+)"')
    if (-not $match.Success) {
        throw "failed to locate AppVersion in $metaPath"
    }
    $currentVersion = $match.Groups[1].Value
    if ($currentVersion -eq $TargetVersion) {
        return
    }
    $updated = [regex]::Replace($raw, 'var AppVersion = "[^"]+"', ('var AppVersion = "' + $TargetVersion + '"'), 1)
    Write-Utf8NoBom -Path $metaPath -Content $updated
}

function Update-PackageJsonVersion([string]$RepoRoot, [string]$TargetVersion) {
    $pkgPath = Join-Path $RepoRoot "package.json"
    if (-not (Test-Path $pkgPath)) {
        return
    }
}

function Update-PackageLockRootVersions([string]$RepoRoot, [string]$TargetVersion) {
    $lockPath = Join-Path $RepoRoot "package-lock.json"
    if (-not (Test-Path $lockPath)) {
        return
    }
}

function Sync-PackageMetadataVersion([string]$RepoRoot, [string]$TargetVersion) {
    $pkgPath = Join-Path $RepoRoot "package.json"
    if (-not (Test-Path $pkgPath)) {
        return
    }

    $npmCmd = Get-Command "npm.cmd" -ErrorAction SilentlyContinue
    if ($null -eq $npmCmd) {
        throw "npm.cmd not found; cannot synchronize package metadata version"
    }

    Push-Location $RepoRoot
    try {
        & $npmCmd.Source "version" $TargetVersion "--no-git-tag-version" "--allow-same-version" | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "npm version command failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }

    # Post-check: ensure exactly required root fields are synchronized.
    $pkgJsonRaw = Get-Content (Join-Path $RepoRoot "package.json") -Raw | ConvertFrom-Json
    if ($pkgJsonRaw.version -ne $TargetVersion) {
        throw "package.json root version was not updated to $TargetVersion"
    }
    $lockPath = Join-Path $RepoRoot "package-lock.json"
    if (Test-Path $lockPath) {
        $lockRaw = Get-Content $lockPath -Raw
        $topMatch = [regex]::Match($lockRaw, '(?ms)\A\s*\{\s*"name"\s*:\s*"[^"]+"\s*,\s*"version"\s*:\s*"([^"]+)"')
        if (-not $topMatch.Success) {
            throw "package-lock.json top-level version field not found"
        }
        if ($topMatch.Groups[1].Value -ne $TargetVersion) {
            throw "package-lock.json top-level version was not updated to $TargetVersion"
        }
        $rootMatch = [regex]::Match($lockRaw, '(?ms)"packages"\s*:\s*\{\s*""\s*:\s*\{\s*"name"\s*:\s*"[^"]+"\s*,\s*"version"\s*:\s*"([^"]+)"')
        if (-not $rootMatch.Success) {
            throw 'package-lock.json missing packages[""] root entry'
        }
        if ($rootMatch.Groups[1].Value -ne $TargetVersion) {
            throw ("package-lock.json packages[""`"`"].version was not updated to " + $TargetVersion)
        }
    }
}

Assert-Version -Value $Version
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

Update-AppMeta -RepoRoot $repoRoot -TargetVersion $Version
Sync-PackageMetadataVersion -RepoRoot $repoRoot -TargetVersion $Version

Write-Host ("Version synchronized to " + $Version)
