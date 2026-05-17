<#
.SYNOPSIS
    qsdev installer for Windows.

.DESCRIPTION
    Downloads and installs the qsdev binary for Windows.
    Verifies SHA256 checksums and optionally adds the install directory to the user PATH.

.PARAMETER Version
    Pin to a specific version (e.g. "1.2.3"). If omitted, installs the latest release.

.PARAMETER InstallDir
    Override the install directory. Default: $env:LOCALAPPDATA\qsdev\bin

.PARAMETER NoModifyPath
    Skip adding the install directory to the user PATH.

.PARAMETER DryRun
    Show what would be done without making changes.

.EXAMPLE
    # Install latest version
    irm https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.ps1 | iex

.EXAMPLE
    # Install specific version
    .\install.ps1 -Version 1.2.3

.EXAMPLE
    # Preview without installing
    .\install.ps1 -DryRun
#>

[CmdletBinding()]
param(
    [string]$Version = $env:QSDEV_VERSION,
    [string]$InstallDir = $(if ($env:QSDEV_INSTALL_DIR) { $env:QSDEV_INSTALL_DIR } else { "$env:LOCALAPPDATA\qsdev\bin" }),
    [switch]$NoModifyPath,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$GithubOrg = "Quantum-Serendipity"
$GithubRepo = "qsdev"
$BinaryName = "qsdev"

function Detect-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture

    switch ($arch) {
        ([System.Runtime.InteropServices.Architecture]::X64) {
            return "x86_64"
        }
        ([System.Runtime.InteropServices.Architecture]::Arm64) {
            return "arm64"
        }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

function Resolve-Version {
    if ($Version) {
        return $Version
    }

    Write-Host "Fetching latest version..." -ForegroundColor Cyan
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$GithubOrg/$GithubRepo/releases/latest"
        $ver = $release.tag_name -replace '^v', ''
        if (-not $ver) {
            throw "Could not parse version from tag_name: $($release.tag_name)"
        }
        return $ver
    }
    catch {
        throw "Could not determine latest version. Set -Version or `$env:QSDEV_VERSION to install a specific version. Error: $_"
    }
}

function Download-AndVerify {
    param(
        [string]$ResolvedVersion,
        [string]$Arch,
        [string]$TmpDir
    )

    $filename = "${BinaryName}_${ResolvedVersion}_Windows_${Arch}.zip"
    $archiveUrl = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v${ResolvedVersion}/$filename"
    $checksumUrl = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v${ResolvedVersion}/checksums.txt"

    Write-Host "Downloading $BinaryName v${ResolvedVersion} for Windows/${Arch}..." -ForegroundColor Cyan

    if ($DryRun) {
        Write-Host "[dry-run] Would download: $archiveUrl" -ForegroundColor Cyan
        Write-Host "[dry-run] Would download: $checksumUrl" -ForegroundColor Cyan
        return $filename
    }

    Invoke-WebRequest -Uri $archiveUrl -OutFile "$TmpDir\$filename" -UseBasicParsing
    Invoke-WebRequest -Uri $checksumUrl -OutFile "$TmpDir\checksums.txt" -UseBasicParsing

    Write-Host "Verifying SHA256 checksum..." -ForegroundColor Cyan

    $checksumLine = Get-Content "$TmpDir\checksums.txt" | Where-Object { $_ -match [regex]::Escape($filename) }
    if (-not $checksumLine) {
        throw "Could not find checksum for $filename in checksums.txt"
    }

    $expectedHash = ($checksumLine -split '\s+')[0].ToLower()
    $actualHash = (Get-FileHash "$TmpDir\$filename" -Algorithm SHA256).Hash.ToLower()

    if ($expectedHash -ne $actualHash) {
        throw "Checksum verification failed!`n  Expected: $expectedHash`n  Got:      $actualHash"
    }

    Write-Host "Checksum verified." -ForegroundColor Green
    return $filename
}

function Extract-AndInstall {
    param(
        [string]$Filename,
        [string]$ResolvedVersion,
        [string]$TmpDir
    )

    if ($DryRun) {
        Write-Host "[dry-run] Would extract to: $TmpDir\extracted\" -ForegroundColor Cyan
        Write-Host "[dry-run] Would install ${BinaryName}.exe to: $InstallDir\${BinaryName}.exe" -ForegroundColor Cyan
        return
    }

    Write-Host "Installing to $InstallDir..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

    $extractDir = "$TmpDir\extracted"
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Expand-Archive -Path "$TmpDir\$Filename" -DestinationPath $extractDir -Force

    Copy-Item "$extractDir\${BinaryName}.exe" "$InstallDir\${BinaryName}.exe" -Force

    Write-Host "$BinaryName v${ResolvedVersion} installed to $InstallDir\${BinaryName}.exe" -ForegroundColor Green
}

function Update-Path {
    if ($NoModifyPath) {
        return
    }

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

    if ($currentPath -and $currentPath.Split(';') -contains $InstallDir) {
        return
    }

    if ($DryRun) {
        Write-Host "[dry-run] Would add $InstallDir to user PATH" -ForegroundColor Cyan
        return
    }

    if ($currentPath) {
        $newPath = "$InstallDir;$currentPath"
    } else {
        $newPath = $InstallDir
    }

    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "Added $InstallDir to user PATH." -ForegroundColor Green
    Write-Host "Restart your terminal for the change to take effect."
}

function Install-Qsdev {
    $arch = Detect-Architecture
    $resolvedVersion = Resolve-Version

    Write-Host ""
    Write-Host "qsdev installer" -ForegroundColor White
    Write-Host "===============" -ForegroundColor White
    Write-Host "  Version:     v$resolvedVersion" -ForegroundColor Cyan
    Write-Host "  OS:          Windows" -ForegroundColor Cyan
    Write-Host "  Arch:        $arch" -ForegroundColor Cyan
    Write-Host "  Install dir: $InstallDir" -ForegroundColor Cyan
    Write-Host ""

    if ($DryRun) {
        Write-Host "[dry-run mode -- no changes will be made]" -ForegroundColor Cyan
        Write-Host ""
    }

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "qsdev-install-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        $filename = Download-AndVerify -ResolvedVersion $resolvedVersion -Arch $arch -TmpDir $tmpDir
        Extract-AndInstall -Filename $filename -ResolvedVersion $resolvedVersion -TmpDir $tmpDir
        Update-Path

        Write-Host ""
        Write-Host "Installation complete!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. Restart your terminal (or open a new one)" -ForegroundColor Cyan
        Write-Host "  2. Verify the installation:  $BinaryName version" -ForegroundColor Cyan
        Write-Host "  3. Check your environment:   $BinaryName doctor" -ForegroundColor Cyan
        Write-Host "  4. Initialize a project:     $BinaryName init" -ForegroundColor Cyan
    }
    finally {
        if (Test-Path $tmpDir) {
            Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
        }
    }
}

Install-Qsdev
