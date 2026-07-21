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

function Invoke-WithGitLabSSHKey {
  param(
    [Parameter(Mandatory = $true)]
    [scriptblock]$Action
  )

  $keyPath = Join-Path $root ".work/.ssh/id_ed25519"
  if (-not (Test-Path -LiteralPath $keyPath -PathType Leaf)) {
    throw "GitLab SSH key not found: $keyPath"
  }

  $previousGitSSHCommand = $env:GIT_SSH_COMMAND
  try {
    $env:GIT_SSH_COMMAND = 'ssh -i "' + $keyPath + '" -o IdentitiesOnly=yes -o BatchMode=yes'
    & $Action
  } finally {
    if ($null -eq $previousGitSSHCommand) {
      Remove-Item Env:GIT_SSH_COMMAND -ErrorAction SilentlyContinue
    } else {
      $env:GIT_SSH_COMMAND = $previousGitSSHCommand
    }
  }
}

function Publish-GitLabMain {
  $currentBranch = git branch --show-current
  if ($LASTEXITCODE -ne 0) {
    throw "failed to determine the current git branch"
  }
  if ($currentBranch.Trim() -ne "main") {
    throw "GitLab publication requires the main branch; current branch: $($currentBranch.Trim())"
  }

  $dirtyFiles = git status --short
  if ($LASTEXITCODE -ne 0) {
    throw "failed to check git worktree status"
  }
  if (-not [string]::IsNullOrWhiteSpace(($dirtyFiles -join ""))) {
    throw "GitLab publication requires a clean worktree; commit or stash local changes first"
  }

  Write-Status "RUN" "Syncing with gitlab/main"
  Invoke-WithGitLabSSHKey {
    Invoke-Checked -Command "git" -Arguments @("pull", "--rebase", "gitlab", "main")
    Write-Status "RUN" "Pushing main to GitLab"
    Invoke-Checked -Command "git" -Arguments @("push", "gitlab", "HEAD:main")
  }
  Write-Status "OK" "Source code published to GitLab main"
}

function Normalize-Mode([string]$Raw) {
  $value = ""
  if (-not [string]::IsNullOrWhiteSpace($Raw)) {
    $value = $Raw.Trim().ToLowerInvariant()
  }
  switch ($value) {
    "1" { return "check" }
    "2" { return "version" }
    "3" { return "gitlab" }
    "check" { return "check" }
    "version" { return "version" }
    "sync-version" { return "version" }
    "gitlab" { return "gitlab" }
    "gitlab-publish" { return "gitlab" }
    default { return "" }
  }
}

function Read-ModeInteractive {
  Write-Title "Select Mode"
  Write-Host "1. Check only" -ForegroundColor White
  Write-Host "2. Version sync only" -ForegroundColor White
  Write-Host "3. Publish current main to GitLab (no checks)" -ForegroundColor White
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
    @{ Name = "e2e admin-panel ModSecurity safeguard (full runtime stack)"; Action = { Invoke-E2ERuntimeSuite -Filter "TestE2EAdminPanelModSecurityBypassesEveryAdministrativeRoute" } },
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
}

Sync-ReleaseVersionContext

Set-Location $root

Write-Title "Release Configuration"
Write-KeyValue "Current version" $version
Write-KeyValue "Release tag" $tag

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
  }
  "check" {
    Write-Title "Mode: Check Only"
    Invoke-LocalPreflight -SkipGoTest -CompactOutput
    Write-Status "OK" "Check completed"
  }
  "gitlab" {
    Write-Title "Mode: GitLab Publish Only"
    Publish-GitLabMain
  }
  default {
    throw "unsupported mode: $normalizedArgMode"
  }
}
