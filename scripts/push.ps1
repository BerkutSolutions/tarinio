param(
  [string]$Message = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Checked {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Command,
    [Parameter(Mandatory = $false)]
    [string[]]$Arguments = @()
  )
  & $Command @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "command failed ($LASTEXITCODE): $Command $($Arguments -join ' ')"
  }
}

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

if ([string]::IsNullOrWhiteSpace($Message)) {
  $metaFile = Join-Path $root "control-plane/internal/appmeta/meta.go"
  $metaContent = Get-Content $metaFile -Raw -Encoding UTF8
  $versionMatch = [regex]::Match($metaContent, 'var AppVersion = "([^"]+)"')
  if ($versionMatch.Success) {
    $Message = "v$($versionMatch.Groups[1].Value)"
  } else {
    $Message = "chore: update repository"
  }
}

Invoke-Checked -Command "git" -Arguments @("add", ".")

$stagedFiles = git diff --cached --name-only
if ($LASTEXITCODE -ne 0) {
  throw "failed to inspect staged changes"
}

if (-not [string]::IsNullOrWhiteSpace(($stagedFiles -join ""))) {
  Invoke-Checked -Command "git" -Arguments @("commit", "-m", $Message)
} else {
  Write-Host "no staged changes to commit"
}

Invoke-Checked -Command "git" -Arguments @("pull", "--rebase", "origin", "main")
Invoke-Checked -Command "git" -Arguments @("push", "origin", "main")

Write-Host "push completed"
