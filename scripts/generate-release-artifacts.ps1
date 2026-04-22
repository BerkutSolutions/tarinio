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
$metaFile = Join-Path $root "control-plane/internal/appmeta/meta.go"

if (-not (Test-Path $metaFile)) {
  throw "meta.go not found: $metaFile"
}

$metaContent = Get-Content $metaFile -Raw
$versionMatch = [regex]::Match($metaContent, 'var AppVersion = "([^"]+)"')
if (-not $versionMatch.Success) {
  throw "failed to parse AppVersion from $metaFile"
}

$version = $versionMatch.Groups[1].Value
$tag = "v$version"
$commit = ((git -C $root rev-parse HEAD) | Select-Object -First 1).Trim()
if ([string]::IsNullOrWhiteSpace($commit)) {
  throw "failed to resolve git commit"
}

$dockerTags = @(
  "tarinio:$version",
  "ghcr.io/berkutsolutions/tarinio:$version",
  "ghcr.io/berkutsolutions/tarinio:latest"
) -join ","

Set-Location $root
Invoke-Checked -Command "go" -Arguments @(
  "run",
  "./cmd/release-artifacts",
  "-repo-root", $root,
  "-version", $version,
  "-commit", $commit,
  "-tag", $tag,
  "-output", (Join-Path $root "build/release/$version"),
  "-docker-tags", $dockerTags
)
