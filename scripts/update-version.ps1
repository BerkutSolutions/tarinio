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

function Sync-I18NAppVersion([string]$RepoRoot, [string]$TargetVersion) {
    # Updates app.version in all 5 locale files.
    # Each locale has its own display prefix before the version number:
    #   en, de, zh -> "v<version>"
    #   sr         -> "в<version>"
    #   ru         -> "Версия <version>"
    # Docs are intentionally excluded from version sync
    # (documentation carries version as editorial context, not a build artifact).
    $utf8 = [System.Text.UTF8Encoding]::new($false)
    # Cyrillic prefixes defined via char codes to avoid encoding issues in the script file itself:
    #   ru: "Версия " (0x412 0x435 0x440 0x441 0x438 0x44F 0x20)
    #   sr: "в" (0x432)
    $prefixRu = [string][char]0x412 + [char]0x435 + [char]0x440 + [char]0x441 + [char]0x438 + [char]0x44F + [char]0x20
    $prefixSr = [string][char]0x432
    $localeValues = @{
        "en.json" = "v$TargetVersion"
        "de.json" = "v$TargetVersion"
        "zh.json" = "v$TargetVersion"
        "sr.json" = ($prefixSr + $TargetVersion)
        "ru.json" = ($prefixRu + $TargetVersion)
    }
    $i18nDir = Join-Path $RepoRoot "ui/app/static/i18n"
    foreach ($file in $localeValues.Keys) {
        $path = Join-Path $i18nDir $file
        if (-not (Test-Path $path)) {
            throw "i18n file not found: $path"
        }
        $raw = [System.Text.Encoding]::UTF8.GetString([System.IO.File]::ReadAllBytes($path))
        if ($raw -notmatch '"app\.version"\s*:\s*"[^"]+"') {
            throw "app.version key not found in $file"
        }
        $newValue = $localeValues[$file]
        $updated = [regex]::Replace($raw, '("app\.version"\s*:\s*)"[^"]+"', ('$1"' + $newValue + '"'))
        if ($updated -ne $raw) {
            [System.IO.File]::WriteAllText($path, $updated, $utf8)
        }
    }
}

function Sync-PackageMetadataVersion([string]$RepoRoot, [string]$TargetVersion) {
    # Updates package.json and package-lock.json (root + packages[""] entries) via npm version.
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
            throw ("package-lock.json packages[`"`"].version was not updated to " + $TargetVersion)
        }
    }
}

function Sync-DocsAndFrontendVersion([string]$RepoRoot, [string]$TargetVersion) {
    $utf8 = [System.Text.UTF8Encoding]::new($false)

    # docusaurus.config.js — navbar version label and dropdown item
    $docusaurusPath = Join-Path $RepoRoot "docusaurus.config.js"
    if (Test-Path $docusaurusPath) {
        $raw = [System.IO.File]::ReadAllText($docusaurusPath, $utf8)
        $updated = [regex]::Replace($raw, "label: 'Version \d+\.\d+\.\d+'", "label: 'Version $TargetVersion'")
        $updated = [regex]::Replace($updated, "Current Release \d+\.\d+\.\d+", "Current Release $TargetVersion")
        if ($updated -ne $raw) {
            [System.IO.File]::WriteAllText($docusaurusPath, $updated, $utf8)
        }
    }

    # src/pages/index.jsx — hero eyebrow label
    $indexJsxPath = Join-Path $RepoRoot "src/pages/index.jsx"
    if (Test-Path $indexJsxPath) {
        $raw = [System.IO.File]::ReadAllText($indexJsxPath, $utf8)
        $updated = [regex]::Replace($raw, "TARINIO \d+\.\d+\.\d+", "TARINIO $TargetVersion")
        if ($updated -ne $raw) {
            [System.IO.File]::WriteAllText($indexJsxPath, $updated, $utf8)
        }
    }

    # ui/tests/release_docs_test.go — version string in test assertion
    $releaseDocsTestPath = Join-Path $RepoRoot "ui/tests/release_docs_test.go"
    if (Test-Path $releaseDocsTestPath) {
        $raw = [System.IO.File]::ReadAllText($releaseDocsTestPath, $utf8)
        $updated = [regex]::Replace($raw, "default \d+\.\d+\.\d+ profile", "default $TargetVersion profile")
        if ($updated -ne $raw) {
            [System.IO.File]::WriteAllText($releaseDocsTestPath, $updated, $utf8)
        }
    }
}

function Sync-LegacyRuntimeVersion([string]$RepoRoot, [string]$TargetVersion) {
    # The runtime shell is served independently from the documentation site.
    # Keep its pre-i18n fallback version in sync for users whose locale has not
    # loaded yet, and for pages rendered before app.js finishes bootstrapping.
    $utf8 = [System.Text.UTF8Encoding]::new($false)
    $targets = @(
        @{ Path = "ui/app/static/js/app.js"; Pattern = 'v\d+\.\d+\.\d+' },
        @{ Path = "ui/app/index.html"; Pattern = '>v\d+\.\d+\.\d+<' }
    )
    foreach ($target in $targets) {
        $path = Join-Path $RepoRoot $target.Path
        if (-not (Test-Path $path)) {
            throw "runtime version file not found: $path"
        }
        $raw = [System.IO.File]::ReadAllText($path, $utf8)
        $replacement = if ($target.Path.EndsWith("index.html")) { ">v$TargetVersion<" } else { "v$TargetVersion" }
        $updated = [regex]::Replace($raw, $target.Pattern, $replacement)
        if ($updated -eq $raw) {
            throw "legacy runtime version marker not found in $path"
        }
        [System.IO.File]::WriteAllText($path, $updated, $utf8)
    }
}

Assert-Version -Value $Version
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

Update-AppMeta -RepoRoot $repoRoot -TargetVersion $Version
Sync-PackageMetadataVersion -RepoRoot $repoRoot -TargetVersion $Version
Sync-I18NAppVersion -RepoRoot $repoRoot -TargetVersion $Version
Sync-DocsAndFrontendVersion -RepoRoot $repoRoot -TargetVersion $Version
Sync-LegacyRuntimeVersion -RepoRoot $repoRoot -TargetVersion $Version

Write-Host ("Version synchronized to " + $Version)
