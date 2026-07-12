param(
  [string]$Mode = "",
  [string]$Version = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Title([string]$Text) {
  Write-Host ""
  Write-Host ("== " + $Text + " ==") -ForegroundColor Cyan
}

function Write-KeyValue([string]$Key, [string]$Value) {
  Write-Host ($Key + ": ") -NoNewline -ForegroundColor DarkGray
  Write-Host $Value -ForegroundColor White
}

function Write-Status([string]$Status, [string]$Text) {
  $label = "[" + $Status + "]"
  switch ($Status.ToUpperInvariant()) {
    "OK" {
      Write-Host ($label.PadRight(7) + " " + $Text) -ForegroundColor Green
    }
    "FAIL" {
      Write-Host ($label.PadRight(7) + " " + $Text) -ForegroundColor Red
    }
    "WARN" {
      Write-Host ($label.PadRight(7) + " " + $Text) -ForegroundColor Yellow
    }
    "RUN" {
      Write-Host ($label.PadRight(7) + " " + $Text) -ForegroundColor DarkCyan
    }
    default {
      Write-Host ($label.PadRight(7) + " " + $Text)
    }
  }
}

function Write-Info([string]$Text) {
  Write-Host $Text -ForegroundColor DarkGray
}

function Invoke-Checked {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Command,
    [Parameter(Mandatory = $false)]
    [string[]]$Arguments = @(),
    [Parameter(Mandatory = $false)]
    [switch]$Quiet
  )
  $prevErrorAction = $ErrorActionPreference
  $hasNativePref = $null -ne (Get-Variable -Name PSNativeCommandUseErrorActionPreference -ErrorAction SilentlyContinue)
  if ($hasNativePref) {
    $prevNativePref = $PSNativeCommandUseErrorActionPreference
    $PSNativeCommandUseErrorActionPreference = $false
  }
  $ErrorActionPreference = "Continue"
  try {
    if ($Quiet) {
      $output = & $Command @Arguments 2>&1
      if ($LASTEXITCODE -ne 0) {
        $argsText = ($Arguments -join " ")
        $tail = ($output | Select-Object -Last 40) -join [Environment]::NewLine
        throw "command failed ($LASTEXITCODE): $Command $argsText`n$tail"
      }
      return
    }
    else {
      & $Command @Arguments
      if ($LASTEXITCODE -ne 0) {
        throw "command failed ($LASTEXITCODE): $Command $($Arguments -join ' ')"
      }
    }
  } finally {
    $ErrorActionPreference = $prevErrorAction
    if ($hasNativePref) {
      $PSNativeCommandUseErrorActionPreference = $prevNativePref
    }
  }
}

function Invoke-LocalPreflight {
  param(
    [switch]$SkipGoTest,
    [switch]$CompactOutput
  )
  $args = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", (Join-Path $root "scripts/local-ci-preflight.ps1"))
  if ($SkipGoTest) {
    $args += "-SkipGoTest"
  }
  if ($CompactOutput) {
    $args += "-CompactOutput"
  }
  $hostPowerShell = if ($PSVersionTable.PSEdition -eq "Core") { "pwsh" } else { "powershell" }
  Invoke-Checked -Command $hostPowerShell -Arguments $args
}

function Invoke-NpmSecurityGate {
  $hostPowerShell = if ($PSVersionTable.PSEdition -eq "Core") { "pwsh" } else { "powershell" }
  Invoke-Checked -Command $hostPowerShell -Arguments @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", (Join-Path $root "scripts/npm-security-gate.ps1"),
    "-AuditLevel", "moderate",
    "-AutoFix"
  )
}

function Publish-ReleaseMetadata {
  Write-Status "RUN" "Generating signed release artifacts"
  Invoke-Checked -Command "powershell" -Arguments @("-ExecutionPolicy", "Bypass", "-File", (Join-Path $root "scripts/generate-release-artifacts.ps1"))

  Write-Status "RUN" "Staging repository changes"
  Invoke-Checked -Command "git" -Arguments @("add", ".")

  $stagedFiles = git diff --cached --name-only
  if ($LASTEXITCODE -ne 0) {
    throw "failed to check staged changes"
  }
  $hasStagedChanges = -not [string]::IsNullOrWhiteSpace(($stagedFiles -join ""))

  if ($hasStagedChanges) {
    Write-Status "RUN" ("Creating commit " + $tag)
    Invoke-Checked -Command "git" -Arguments @("commit", "-m", $tag)
  } else {
    Write-Status "WARN" "No staged changes to commit"
  }

  Write-Status "RUN" "Syncing with origin/main"
  Invoke-Checked -Command "git" -Arguments @("pull", "--rebase", "origin", "main")
  Invoke-Checked -Command "git" -Arguments @("push", "origin", "main")

  $existingTag = git tag --list $tag
  if ($LASTEXITCODE -ne 0) {
    throw "failed to check existing tag"
  }
  $tagExists = -not [string]::IsNullOrWhiteSpace(($existingTag -join ""))

  if ($tagExists) {
    Write-Status "WARN" "Tag already exists locally: $tag"
  } else {
    Write-Status "RUN" ("Creating git tag " + $tag)
    Invoke-Checked -Command "git" -Arguments @("tag", "-a", $tag, "-m", "Release $version")
  }

  Write-Status "RUN" ("Pushing git tag " + $tag)
  Invoke-Checked -Command "git" -Arguments @("push", "origin", $tag)
  Publish-GitHubRelease
  Write-Status "OK" ("Release metadata published for " + $version)
}

function Get-ChangelogReleaseNotes {
  $changelogPath = Join-Path $root "CHANGELOG.md"
  if (-not (Test-Path $changelogPath)) {
    throw "changelog not found: $changelogPath"
  }
  Write-Status "RUN" ("Reading changelog section [" + $version + "] from " + $changelogPath)
  $lines = [System.IO.File]::ReadAllLines($changelogPath)
  $headingPattern = "^##\s+\[" + [regex]::Escape($version) + "\](?:\s|$)"
  $start = -1
  for ($index = 0; $index -lt $lines.Length; $index++) {
    if ($lines[$index] -match $headingPattern) {
      $start = $index
      break
    }
  }
  if ($start -lt 0) {
    $available = @()
    foreach ($line in $lines) {
      if ($line -match '^##\s+\[([^\]]+)\]') {
        $available += $Matches[1]
      }
    }
    if ($available.Count -gt 0) {
      Write-Status "WARN" ("Available changelog versions: " + ($available -join ", "))
    }
    throw "CHANGELOG.md does not contain section for [$version]"
  }
  $end = $lines.Length
  for ($index = $start + 1; $index -lt $lines.Length; $index++) {
    if ($lines[$index] -match "^##\s+") {
      $end = $index
      break
    }
  }
  return (($lines[$start..($end - 1)]) -join "`n").Trim()
}

function Publish-GitHubRelease {
  $ghCommand = Get-Command "gh" -ErrorAction SilentlyContinue
  if ($null -eq $ghCommand) {
    throw "gh CLI is required to publish GitHub Release"
  }
  $repo = "BerkutSolutions/tarinio"
  $notesDir = Join-Path $root ("build/release/" + $version)
  if (-not (Test-Path $notesDir)) {
    New-Item -ItemType Directory -Path $notesDir -Force | Out-Null
  }
  $notesPath = Join-Path $notesDir "github-release-notes.md"
  [System.IO.File]::WriteAllText($notesPath, (Get-ChangelogReleaseNotes) + "`n", [System.Text.UTF8Encoding]::new($false))

  $releaseExists = $false
  try {
    Invoke-Checked -Command "gh" -Arguments @("release", "view", $tag, "--repo", $repo) -Quiet
    $releaseExists = $true
  } catch {
    $releaseExists = $false
  }
  if ($releaseExists) {
    Write-Status "RUN" ("Updating GitHub release " + $tag)
    Invoke-Checked -Command "gh" -Arguments @("release", "edit", $tag, "--repo", $repo, "--title", ("TARINIO " + $version), "--notes-file", $notesPath)
  } else {
    Write-Status "RUN" ("Creating GitHub release " + $tag)
    Invoke-Checked -Command "gh" -Arguments @("release", "create", $tag, "--repo", $repo, "--title", ("TARINIO " + $version), "--notes-file", $notesPath)
  }
}

function Publish-DockerPackage {
  Write-Status "RUN" ("Docker build " + $localImage)
  Invoke-Checked -Command "docker" -Arguments @("build", "-t", $localImage, "-f", "control-plane/Dockerfile", ".") -Quiet
  Write-Status "RUN" ("Tag image as " + $ghcrVersion)
  Invoke-Checked -Command "docker" -Arguments @("tag", $localImage, $ghcrVersion)
  Write-Status "RUN" ("Tag image as " + $ghcrLatest)
  Invoke-Checked -Command "docker" -Arguments @("tag", $localImage, $ghcrLatest)
  Write-Status "RUN" ("Push " + $ghcrVersion)
  Invoke-Checked -Command "docker" -Arguments @("push", $ghcrVersion) -Quiet
  Write-Status "RUN" ("Push " + $ghcrLatest)
  Invoke-Checked -Command "docker" -Arguments @("push", $ghcrLatest) -Quiet
  Write-Status "OK" ("Docker package published for " + $version)
}

function Normalize-Mode([string]$Raw) {
  $value = ""
  if (-not [string]::IsNullOrWhiteSpace($Raw)) {
    $value = $Raw.Trim().ToLowerInvariant()
  }
  switch ($value) {
    "1" { return "full" }
    "2" { return "check" }
    "3" { return "release" }
    "4" { return "docker" }
    "5" { return "version" }
    "6" { return "publish" }
    "full" { return "full" }
    "check" { return "check" }
    "release" { return "release" }
    "publish" { return "publish" }
    "docker" { return "docker" }
    "version" { return "version" }
    "sync-version" { return "version" }
    default { return "" }
  }
}

function Read-ModeInteractive {
  Write-Title "Select Mode"
  Write-Host "1. Check and publish (ALL: checks + GitHub Release + Docker)" -ForegroundColor White
  Write-Host "2. Check only" -ForegroundColor White
  Write-Host "3. Release publish only (git tag + GitHub Release)" -ForegroundColor White
  Write-Host "4. Docker package publish only" -ForegroundColor White
  Write-Host "5. Version sync only" -ForegroundColor White
  Write-Host "6. Publish only (release + docker, no checks)" -ForegroundColor White
  $choice = Read-Host "Enter number and press Enter"
  if ([string]::IsNullOrWhiteSpace($choice)) {
    Write-Status "WARN" "No mode selected. Exiting."
    exit 0
  }
  $mode = Normalize-Mode $choice
  if ([string]::IsNullOrWhiteSpace($mode)) {
    throw "invalid selection: $choice"
  }
  return $mode
}

function Invoke-VersionSyncOnly {
  param(
    [string]$RequestedVersion,
    [switch]$ForcePrompt
  )
  $targetVersion = ""
  if (-not $ForcePrompt) {
    $targetVersion = $RequestedVersion
  }
  if ([string]::IsNullOrWhiteSpace($targetVersion)) {
    $targetVersion = Read-Host "Enter new version (example: 3.0.8)"
  }
  if ([string]::IsNullOrWhiteSpace($targetVersion)) {
    throw "version is required for mode 'version'"
  }
  if ($targetVersion -notmatch '^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z\.-]+)?$') {
    throw "invalid version format: $targetVersion"
  }
  Write-Title "Version Sync"
  Write-Status "RUN" ("Updating project version to " + $targetVersion)
  Invoke-Checked -Command "powershell" -Arguments @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", (Join-Path $root "scripts/update-version.ps1"),
    "-Version", $targetVersion
  )
  Write-Status "OK" ("Version sync completed: " + $targetVersion)
}

function Invoke-InstallerSyntaxChecks {
  $installerScripts = @(
    "scripts/install-aio.sh",
    "scripts/install-aio-enterprise.sh"
  )
  function Invoke-InstallerStaticFallback {
    foreach ($scriptPath in $installerScripts) {
      $fullPath = Join-Path $root $scriptPath
      if (-not (Test-Path $fullPath)) {
        throw "installer script not found: $scriptPath"
      }
      $raw = Get-Content $fullPath -Raw
      if ($raw -notmatch '^#!.*\b(bash|sh)\b') {
        throw "installer script must start with a POSIX shell shebang (bash/sh): $scriptPath"
      }
      if ($raw.Contains("`r`n")) {
        throw "installer script must use LF line endings: $scriptPath"
      }
    }
  }

  $syntaxChecked = $false
  $isWindowsHost = ($env:OS -eq "Windows_NT")
  $allowBashFallback = -not $isWindowsHost
  $shCommand = Get-Command "sh" -ErrorAction SilentlyContinue
  $bashCommand = Get-Command "bash" -ErrorAction SilentlyContinue
  if ($null -ne $shCommand) {
    try {
      Invoke-Checked -Command "sh" -Arguments @("-n", "scripts/install-aio.sh")
      Invoke-Checked -Command "sh" -Arguments @("-n", "scripts/install-aio-enterprise.sh")
      $syntaxChecked = $true
    } catch {
      Write-Warning "sh -n failed, trying fallback: $($_.Exception.Message)"
    }
  }
  if ((-not $syntaxChecked) -and $allowBashFallback -and ($null -ne $bashCommand)) {
    try {
      Invoke-Checked -Command "bash" -Arguments @("-n", "scripts/install-aio.sh")
      Invoke-Checked -Command "bash" -Arguments @("-n", "scripts/install-aio-enterprise.sh")
      $syntaxChecked = $true
    } catch {
      Write-Warning "bash -n failed, trying fallback: $($_.Exception.Message)"
    }
  }
  if ((-not $syntaxChecked) -and (Get-Command "docker" -ErrorAction SilentlyContinue)) {
    try {
      $repoMount = $root.Replace('\', '/')
      Invoke-Checked -Command "docker" -Arguments @("run", "--rm", "-v", "${repoMount}:/repo", "busybox:1.36", "sh", "-n", "/repo/scripts/install-aio.sh")
      Invoke-Checked -Command "docker" -Arguments @("run", "--rm", "-v", "${repoMount}:/repo", "busybox:1.36", "sh", "-n", "/repo/scripts/install-aio-enterprise.sh")
      $syntaxChecked = $true
    } catch {
      Write-Warning "docker shell syntax check failed, trying static fallback: $($_.Exception.Message)"
    }
  }
  if (-not $syntaxChecked) {
    Invoke-InstallerStaticFallback
    $syntaxChecked = $true
  }
}

function Invoke-ReleaseSigningCheck {
  $checkVersion = "startup-check-$version"
  $checkOutput = Join-Path $root ("build/release/" + $checkVersion)
  if (Test-Path $checkOutput) {
    Remove-Item -LiteralPath $checkOutput -Recurse -Force
  }
  try {
    Invoke-Checked -Command "go" -Arguments @(
      "run",
      "./cmd/release-artifacts",
      "-repo-root", $root,
      "-version", $checkVersion,
      "-commit", "startup-check",
      "-tag", "startup-check",
      "-output", $checkOutput,
      "-docker-tags", "local/tarinio:$checkVersion"
    )
    $signingDir = Join-Path $root ".work/release-signing"
    $requiredFiles = @(
      (Join-Path $signingDir "release-ed25519-private.pem"),
      (Join-Path $signingDir "release-ed25519-public.pem"),
      (Join-Path $signingDir "key-id.txt")
    )
    foreach ($path in $requiredFiles) {
      if (-not (Test-Path $path)) {
        throw "release signing material not found: $path"
      }
    }
  } finally {
    if (Test-Path $checkOutput) {
      Remove-Item -LiteralPath $checkOutput -Recurse -Force
    }
  }
}

function Invoke-LabK8sNoTraceCheck {
  $scriptPath = Join-Path $root "deploy/lab-k8s-terraform/scripts/test-k8s-no-trace.ps1"
  if (-not (Test-Path $scriptPath)) {
    throw "k8s no-trace script not found: $scriptPath"
  }
  Invoke-Checked -Command "powershell" -Arguments @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $scriptPath)
}

function Invoke-LabTerraformNoTraceCheck {
  $scriptPath = Join-Path $root "deploy/lab-k8s-terraform/scripts/test-terraform-no-trace.ps1"
  if (-not (Test-Path $scriptPath)) {
    throw "terraform no-trace script not found: $scriptPath"
  }
  Invoke-Checked -Command "powershell" -Arguments @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $scriptPath)
}

function Get-GitBashPath {
  $gitBash = @(
    "C:\Program Files\Git\bin\bash.exe",
    "C:\Program Files\Git\usr\bin\bash.exe",
    "$env:ProgramFiles\Git\bin\bash.exe"
  ) | Where-Object { Test-Path $_ } | Select-Object -First 1
  if (-not $gitBash) {
    $gitBash = Get-Command bash -ErrorAction SilentlyContinue | Where-Object { $_.Source -notmatch "WindowsApps|wsl" } | Select-Object -First 1 -ExpandProperty Source
  }
  if (-not $gitBash) {
    throw "Git bash not found; install Git for Windows"
  }
  return $gitBash
}

function Invoke-E2ERuntimeSuite {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Filter
  )

  $repoRoot = (Get-Location).Path
  $scriptPath = Join-Path $repoRoot "scripts\run-e2e-tests.sh"
  if (-not (Test-Path $scriptPath)) {
    throw "run-e2e-tests.sh not found at $scriptPath"
  }

  $gitBash = Get-GitBashPath
  $scriptMsys = $scriptPath -replace '\\','/' -replace '^([A-Za-z]):','/$1'
  $repoMsys = $repoRoot -replace '\\','/' -replace '^([A-Za-z]):','/$1'

  try {
    $env:E2E_FILTER = $Filter
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $gitBash
    $psi.Arguments = "--login `"$scriptMsys`" `"$repoMsys`""
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $false
    $psi.RedirectStandardError = $false
    $proc = [System.Diagnostics.Process]::Start($psi)
    $proc.WaitForExit()
    if ($proc.ExitCode -ne 0) {
      throw "e2e suite failed (filter=$Filter, exit $($proc.ExitCode))"
    }
  } finally {
    Remove-Item Env:\E2E_FILTER -ErrorAction SilentlyContinue
  }
}

function Run-StartupChecks {
  $results = @()
  $checks = @(
    @{ Name = "npm security audit gate"; Action = { Invoke-NpmSecurityGate } },
    @{ Name = "installer script syntax"; Action = { Invoke-InstallerSyntaxChecks } },
    @{ Name = "release signing certificate/key"; Action = { Invoke-ReleaseSigningCheck } },
    @{ Name = "ui installer toolchain guard"; Action = {
        Invoke-Checked -Command "powershell" -Arguments @(
          "-NoProfile",
          "-ExecutionPolicy", "Bypass",
          "-Command",
          "if (-not (Select-String -Path 'ui/Dockerfile' -Pattern '^RUN GOTOOLCHAIN=auto go test ./ui/tests$' -Quiet)) { throw 'ui/Dockerfile is missing required toolchain guard for ui tests' }"
        )
      }
    },
    @{ Name = "tab feature tests (compiler, tabs 1-10)"; Action = { Invoke-Checked -Command "go" -Arguments @("test", "./compiler/internal/compiler/", "-run", "^TestFront_|^TestUpstream_|^TestHeaders_|^TestTraffic_|^TestAntibot_|^TestGeo_|^TestModsec_|^TestWebSocket_|^TestVirtualPatches_", "-count=1") } },
    @{ Name = "tab feature tests (ban escalation, tab 5)"; Action = { Invoke-Checked -Command "go" -Arguments @("test", "./control-plane/internal/easysiteprofiles/", "-run", "^TestBanEscalation_", "-count=1") } },
    @{ Name = "go test ./..."; Action = { Invoke-Checked -Command "go" -Arguments @("test", "./...", "-count=1") } },
    @{ Name = "ui i18n quality"; Action = { Invoke-Checked -Command "go" -Arguments @("test", "./ui/tests", "-run", "TestI18NValuesNonEmpty", "-count=1") } },
    @{ Name = "ui e2e security modes reality (full runtime stack)"; Action = { Invoke-E2ERuntimeSuite -Filter "TestE2ESecurityModesReality" } },
    @{ Name = "e2e behavioral suite (full runtime stack)"; Action = { Invoke-E2ERuntimeSuite -Filter "TestE2EBehavioral" } },
    @{ Name = "docs ru wiki quality"; Action = { Invoke-Checked -Command "go" -Arguments @("test", "./ui/tests", "-run", "TestDocsRuWikiNoMixedEnglish", "-count=1") } },
    @{ Name = "lab k8s no-trace preflight"; Action = { Invoke-LabK8sNoTraceCheck } },
    @{ Name = "lab terraform no-trace preflight"; Action = { Invoke-LabTerraformNoTraceCheck } }
  )
  foreach ($check in $checks) {
    Write-Status "RUN" $check.Name
    $ok = $false
    $errText = ""
    try {
      & $check.Action | Out-Null
      $ok = $true
      Write-Status "OK" $check.Name
    } catch {
      $ok = $false
      $errText = $_.Exception.Message
      Write-Status "FAIL" $check.Name
      if (-not [string]::IsNullOrWhiteSpace($errText)) {
        Write-Info ("      " + $errText)
      }
    }
    $results += [PSCustomObject]@{
      Name  = $check.Name
      Ok    = $ok
      Error = $errText
    }
  }
  return @($results)
}

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$metaFile = Join-Path $root "control-plane/internal/appmeta/meta.go"

if (-not [string]::IsNullOrWhiteSpace($Version)) {
  Write-Title "Version Sync"
  Write-Status "RUN" ("Updating project version to " + $Version)
  Invoke-Checked -Command "powershell" -Arguments @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", (Join-Path $root "scripts/update-version.ps1"),
    "-Version", $Version
  )
}

if (-not (Test-Path $metaFile)) {
  throw "meta.go not found: $metaFile"
}

function Sync-ReleaseVersionContext {
  $script:metaContent = Get-Content $metaFile -Raw
  $versionMatch = [regex]::Match($script:metaContent, 'var AppVersion = "([^"]+)"')
  if (-not $versionMatch.Success) {
    throw "failed to parse AppVersion from $metaFile"
  }
  $script:version = $versionMatch.Groups[1].Value
  $script:tag = "v$script:version"
  $script:localImage = "tarinio:$script:version"
  $script:ghcrVersion = "ghcr.io/berkutsolutions/tarinio:$script:version"
  $script:ghcrLatest = "ghcr.io/berkutsolutions/tarinio:latest"
}

Sync-ReleaseVersionContext

Set-Location $root

Write-Title "Release Configuration"
Write-KeyValue "Current version" $version
Write-KeyValue "Release tag" $tag
Write-KeyValue "Docker image" $ghcrVersion

Write-Title "Startup Test Checks"
$expectedStartupCheckCount = 13
$startupResults = @(Run-StartupChecks)
$executedStartupCheckCount = @($startupResults).Count
$passedStartupCheckCount = @($startupResults | Where-Object { $_.Ok }).Count
$failedStartupCheckCount = $executedStartupCheckCount - $passedStartupCheckCount

Write-Title "Startup Test Summary"
Write-KeyValue "Expected checks" ([string]$expectedStartupCheckCount)
Write-KeyValue "Executed checks" ([string]$executedStartupCheckCount)
Write-KeyValue "Passed checks" ([string]$passedStartupCheckCount)
Write-KeyValue "Failed checks" ([string]$failedStartupCheckCount)

if ($executedStartupCheckCount -ne $expectedStartupCheckCount) {
  throw "startup test check count mismatch: expected $expectedStartupCheckCount, executed $executedStartupCheckCount"
}

$allStartupChecksPassed = ($failedStartupCheckCount -eq 0)

$normalizedArgMode = Normalize-Mode $Mode
$interactiveModeSelection = [string]::IsNullOrWhiteSpace($normalizedArgMode)

if ($interactiveModeSelection) {
  if (-not [string]::IsNullOrWhiteSpace($Mode)) {
    throw "unsupported mode argument: $Mode"
  }
} else {
  Write-Title "Mode Selection"
  Write-KeyValue "Selected mode (argument)" $normalizedArgMode
}

if ($interactiveModeSelection) {
  while ($true) {
    $selectedMode = Read-ModeInteractive
    if ($selectedMode -eq "version") {
      Invoke-VersionSyncOnly -ForcePrompt
      Sync-ReleaseVersionContext
      Write-Title "Release Configuration"
      Write-KeyValue "Current version" $version
      Write-KeyValue "Release tag" $tag
      Write-KeyValue "Docker image" $ghcrVersion
      continue
    }
    $normalizedArgMode = $selectedMode
    break
  }
}

switch ($normalizedArgMode) {
  "version" {
    Invoke-VersionSyncOnly -RequestedVersion $Version
    Sync-ReleaseVersionContext
    Write-Title "Release Configuration"
    Write-KeyValue "Current version" $version
    Write-KeyValue "Release tag" $tag
    Write-KeyValue "Docker image" $ghcrVersion
  }
  "check" {
    Write-Title "Mode: Check Only"
    Invoke-LocalPreflight -SkipGoTest -CompactOutput
    Write-Status "OK" "Check completed"
  }
  "release" {
    Write-Title "Mode: Release Publish Only"
    if (-not $allStartupChecksPassed) {
      throw "release publish is blocked: startup tests failed"
    }
    Publish-ReleaseMetadata
  }
  "docker" {
    Write-Title "Mode: Docker Package Publish Only"
    if (-not $allStartupChecksPassed) {
      throw "docker package publish is blocked: startup tests failed"
    }
    Publish-DockerPackage
  }
  "full" {
    Write-Title "Mode: Check and Publish (ALL: Checks + GitHub Release + Docker)"
    if (-not $allStartupChecksPassed) {
      throw "full release is blocked: startup tests failed"
    }
    Invoke-LocalPreflight -SkipGoTest -CompactOutput
    Publish-ReleaseMetadata
    Publish-DockerPackage
    Write-Status "OK" "Full release completed"
  }
  "publish" {
    Write-Title "Mode: Publish Only (Release + Docker)"
    if (-not $allStartupChecksPassed) {
      throw "publish is blocked: startup tests failed"
    }
    Publish-ReleaseMetadata
    Publish-DockerPackage
    Write-Status "OK" "Publish only completed"
  }
  default {
    throw "unsupported mode: $normalizedArgMode"
  }
}
