<#
.SYNOPSIS
    qsdev installer for Windows.

.DESCRIPTION
    Downloads and installs the qsdev binary for Windows.
    Verifies SHA256 checksums and, when cosign is installed, the Sigstore
    signature of checksums.txt, and optionally adds the install directory to
    the user PATH.

.PARAMETER Version
    Pin to a specific version (e.g. "1.2.3"). If omitted, installs the latest
    release. Defaults to $env:QSDEV_INSTALL_VERSION. $env:QSDEV_VERSION is
    deliberately ignored: qsdev-generated dev environments export it.

.PARAMETER InstallDir
    Override the install directory. Default: $env:LOCALAPPDATA\qsdev\bin

.PARAMETER NoModifyPath
    Skip adding the install directory to the user PATH.

.PARAMETER DryRun
    Show what would be done without making changes.

.PARAMETER AllowUnsigned
    Install even if the release has no Sigstore bundle (also $env:QSDEV_ALLOW_UNSIGNED=1).

.PARAMETER RequireSignature
    Fail unless the Sigstore signature is verified; requires cosign (also $env:QSDEV_REQUIRE_SIGNATURE=1).

.PARAMETER ForceArch
    Override the detected architecture (x86_64 or arm64).

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
    [string]$Version = $env:QSDEV_INSTALL_VERSION,
    [string]$InstallDir = $(if ($env:QSDEV_INSTALL_DIR) { $env:QSDEV_INSTALL_DIR } else { "$env:LOCALAPPDATA\qsdev\bin" }),
    [switch]$NoModifyPath,
    [switch]$DryRun,
    [switch]$AllowUnsigned = ($env:QSDEV_ALLOW_UNSIGNED -eq "1"),
    [switch]$RequireSignature = ($env:QSDEV_REQUIRE_SIGNATURE -eq "1"),
    [ValidateSet("", "x86_64", "arm64")]
    [string]$ForceArch = ""
)

$ErrorActionPreference = "Stop"

$GithubOrg = "Quantum-Serendipity"
$GithubRepo = "qsdev"
$BinaryName = "qsdev"

function Detect-Architecture {
    if ($ForceArch) {
        $arch = $ForceArch
    } else {
        switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture) {
            ([System.Runtime.InteropServices.Architecture]::X64) { $arch = "x86_64" }
            ([System.Runtime.InteropServices.Architecture]::Arm64) { $arch = "arm64" }
            default { throw "Unsupported architecture: $_" }
        }
    }

    # No windows/arm64 build is published (.goreleaser.yaml ignores it); the
    # x86_64 build runs on Windows on ARM under emulation.
    if ($arch -eq "arm64") {
        Write-Host "No native Windows arm64 build is published; installing the x86_64 build, which runs under emulation." -ForegroundColor Yellow
        $arch = "x86_64"
    }
    return $arch
}

# Strips a leading "v" and rejects anything that is not a release version.
function Normalize-Version {
    param([string]$Value, [string]$Source)
    $v = $Value -replace '^v', ''
    if ($v -notmatch '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$') {
        throw "Invalid version '$Value' (from $Source); expected e.g. 1.2.3 or 1.2.3-rc.1."
    }
    return $v
}

function Resolve-Version {
    if ($Version) {
        return Normalize-Version -Value $Version -Source "-Version/QSDEV_INSTALL_VERSION"
    }
    if ($env:QSDEV_VERSION) {
        Write-Host "Ignoring QSDEV_VERSION=$($env:QSDEV_VERSION) (exported by qsdev-generated dev environments); set QSDEV_INSTALL_VERSION to pin a version." -ForegroundColor Cyan
    }

    Write-Host "Fetching latest version..." -ForegroundColor Cyan
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$GithubOrg/$GithubRepo/releases/latest"
        if (-not $release.tag_name) {
            throw "Could not parse version from tag_name: $($release.tag_name)"
        }
    }
    catch {
        throw "Could not determine latest version. Set -Version or `$env:QSDEV_INSTALL_VERSION to install a specific version. Error: $_"
    }
    return Normalize-Version -Value $release.tag_name -Source "latest release"
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

    # Match the file-name field exactly: a substring match also hits the
    # archive's SBOM entries (e.g. "<filename>.cdx.json").
    $checksumLine = Get-Content "$TmpDir\checksums.txt" |
        Where-Object { $fields = $_.Trim() -split '\s+'; $fields.Count -eq 2 -and $fields[1] -eq $filename } |
        Select-Object -First 1
    if (-not $checksumLine) {
        throw "Could not find checksum for $filename in checksums.txt"
    }

    $expectedHash = ($checksumLine.Trim() -split '\s+')[0].ToLower()
    $actualHash = (Get-FileHash "$TmpDir\$filename" -Algorithm SHA256).Hash.ToLower()

    if ($expectedHash -ne $actualHash) {
        throw "Checksum verification failed!`n  Expected: $expectedHash`n  Got:      $actualHash"
    }

    Write-Host "Checksum verified." -ForegroundColor Green

    Verify-Sigstore -ResolvedVersion $ResolvedVersion -TmpDir $TmpDir
    return $filename
}

# Runs cosign with its stderr merged into the output. Windows PowerShell 5.1
# turns a native command's redirected stderr lines into error records, which
# $ErrorActionPreference = "Stop" makes fatal, and cosign writes its normal
# output ("Verified OK") to stderr; the exit status is left in $LASTEXITCODE.
function Invoke-Cosign {
    param([string[]]$Arguments)
    $saved = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & cosign @Arguments 2>&1 | ForEach-Object { "$_" }
    }
    finally {
        $ErrorActionPreference = $saved
    }
}

# Every release publishes checksums.txt.sigstore.json, so once cosign is
# available a missing bundle is treated as tampering (an attacker who can
# replace the archive and checksums.txt can also delete the bundle) unless
# -AllowUnsigned is given.
function Verify-Sigstore {
    param(
        [string]$ResolvedVersion,
        [string]$TmpDir
    )

    if (-not (Get-Command cosign -ErrorAction SilentlyContinue)) {
        if ($RequireSignature) {
            throw "cosign not found, but a verified Sigstore signature is required (-RequireSignature)."
        }
        Write-Host "cosign not found; skipping Sigstore verification. Install cosign for enhanced security." -ForegroundColor Cyan
        return
    }

    # cosign v1 has no --bundle and cannot check this release's signature.
    if ((Invoke-Cosign @("verify-blob", "--help") | Out-String) -notmatch '--bundle') {
        if ($RequireSignature) {
            throw "Installed cosign does not support --bundle; upgrade cosign to verify the signature (-RequireSignature)."
        }
        Write-Warning "cosign version does not support --bundle; skipping Sigstore verification."
        return
    }

    $bundleUrl = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v${ResolvedVersion}/checksums.txt.sigstore.json"
    $bundlePath = "$TmpDir\checksums.txt.sigstore.json"

    Write-Host "Verifying Sigstore signature on checksums.txt..." -ForegroundColor Cyan
    try {
        Invoke-WebRequest -Uri $bundleUrl -OutFile $bundlePath -UseBasicParsing
    }
    catch {
        if ($AllowUnsigned) {
            Write-Warning "Sigstore bundle unavailable: $_"
            Write-Warning "INSTALLING WITHOUT SIGNATURE VERIFICATION (-AllowUnsigned). The checksum alone does not detect replaced release assets."
            return
        }
        throw "Sigstore bundle unavailable: $_`nEvery qsdev release is signed; a missing bundle can mean the release assets were tampered with. Nothing was installed.`nRe-run with -AllowUnsigned (or `$env:QSDEV_ALLOW_UNSIGNED=1) only if you have verified this release another way."
    }

    $identity = "https://github.com/$GithubOrg/$GithubRepo/.github/workflows/release.yml@refs/tags/v${ResolvedVersion}"
    # Out-Host: cosign's output must not become part of this function's
    # (and so Download-AndVerify's) return value.
    Invoke-Cosign @("verify-blob", "--bundle", $bundlePath, "--certificate-identity", $identity,
        "--certificate-oidc-issuer", "https://token.actions.githubusercontent.com", "$TmpDir\checksums.txt") | Out-Host
    if ($LASTEXITCODE -ne 0) {
        throw "Sigstore verification FAILED. The checksums file may have been tampered with."
    }
    Write-Host "Sigstore signature verified." -ForegroundColor Green
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
    if ($AllowUnsigned -and $RequireSignature) {
        throw "-AllowUnsigned and -RequireSignature (or QSDEV_ALLOW_UNSIGNED and QSDEV_REQUIRE_SIGNATURE) contradict each other."
    }
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
