<#
.SYNOPSIS
    Builds the atlas-ap-remote Windows executable.

.DESCRIPTION
    Produces dist/atlas-ap-remote.exe with trimpath and stripped symbols.
    GOOS is pinned to "windows"; GOARCH defaults to amd64 and can be
    overridden via the -Arch parameter or the GOARCH environment variable.

    CGO is disabled so the produced .exe runs on any Windows machine
    without a C runtime install.

.PARAMETER Arch
    Target architecture (default: amd64). Valid values are amd64, 386,
    arm64. Override via -Arch or $env:GOARCH.

.PARAMETER Out
    Output directory for the produced binary (default: dist).

.EXAMPLE
    PS> .\build.ps1
    PS> .\build.ps1 -Version 0.2.0
    PS> .\build.ps1 -Arch arm64
    PS> .\build.ps1 -Out build\release
#>

[CmdletBinding()]
param(
    [ValidateSet("amd64", "386", "arm64")]
    [string]$Arch = "amd64",

    [string]$Out = "dist",

    [string]$Version = "dev"
)

# Strict mode: stop on unhandled errors and treat unset variables as errors.
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Resolve architecture: explicit -Arch wins, then $env:GOARCH, then default.
if (-not $Arch -and $env:GOARCH) {
    $Arch = $env:GOARCH
}

# Pin target platform to Windows.
$env:GOOS = "windows"
$env:GOARCH = $Arch
$env:CGO_ENABLED = "0"

# Ensure the output directory exists.
if (-not (Test-Path -Path $Out)) {
    New-Item -ItemType Directory -Path $Out -Force | Out-Null
}

$exePath = Join-Path -Path $Out -ChildPath "atlas-ap-remote.exe"
Write-Host "Building $env:GOOS/$env:GOARCH -> $exePath"

$ldflags = "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=$Version"
& go build -trimpath -ldflags $ldflags -o $exePath ./cmd/atlas-ap-remote
if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
}

Write-Host "Built $exePath"
Write-Host "Version: $Version"
