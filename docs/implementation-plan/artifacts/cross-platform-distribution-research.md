# Cross-Platform Go CLI Distribution Research

> **Context**: gdev is a developer environment bootstrapping CLI tool that needs to be distributed to hundreds of software engineers at a consulting firm. Target platforms: Windows, macOS (Intel + Apple Silicon), Linux (Debian, Ubuntu, Fedora, Arch, Mint, RHEL, and others). The tool must be a single self-contained binary that can detect what's missing on a developer's machine and install prerequisites.

---

## Table of Contents

1. [Cross-Platform Go Binary Distribution](#1-cross-platform-go-binary-distribution)
2. [Self-Bootstrapping Installer Patterns](#2-self-bootstrapping-installer-patterns)
3. [OS Detection and Package Manager Abstraction in Go](#3-os-detection-and-package-manager-abstraction-in-go)
4. [Prerequisite Installation Strategies](#4-prerequisite-installation-strategies)
5. [Prior Art and Patterns](#5-prior-art-and-patterns)
6. [Recommended Architecture for gdev](#6-recommended-architecture-for-gdev)

---

## 1. Cross-Platform Go Binary Distribution

### 1.1 GoReleaser Configuration

GoReleaser is the industry-standard tool for building and distributing Go binaries across platforms. It handles cross-compilation, packaging, checksums, signing, and publishing to multiple package managers from a single `.goreleaser.yaml` file.

#### Complete .goreleaser.yaml for gdev

```yaml
# .goreleaser.yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: gdev
    main: ./cmd/gdev
    binary: gdev
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.CommitDate}}
      - -X main.builtBy=goreleaser
    mod_timestamp: "{{ .CommitTimestamp }}"
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      # Windows ARM64 is rare in enterprise
      - goos: windows
        goarch: arm64

archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - LICENSE
      - README.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
      - "^chore:"

release:
  github:
    owner: your-org
    name: gdev
  prerelease: auto
  draft: false
  name_template: "v{{ .Version }}"

# --- Package Manager Publishing ---

homebrew_casks:
  - name: gdev
    binaries:
      - gdev
    repository:
      owner: your-org
      name: homebrew-tap
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/your-org/gdev
    description: "Developer environment bootstrapping tool"
    dependencies:
      - formula: git

scoops:
  - name: gdev
    homepage: https://github.com/your-org/gdev
    description: "Developer environment bootstrapping tool"
    license: MIT
    repository:
      owner: your-org
      name: scoop-bucket
      branch: main
      token: "{{ .Env.SCOOP_TAP_GITHUB_TOKEN }}"

chocolateys:
  - name: gdev
    title: gdev
    owners: your-org
    authors: Your Org
    project_url: https://github.com/your-org/gdev
    license_url: https://github.com/your-org/gdev/blob/main/LICENSE
    require_license_acceptance: false
    description: "Developer environment bootstrapping tool"
    api_key: "{{ .Env.CHOCOLATEY_API_KEY }}"
    source_repo: "https://push.chocolatey.org/"
    skip_publish: false

winget:
  - name: gdev
    publisher: YourOrg
    short_description: "Developer environment bootstrapping tool"
    license: MIT
    publisher_url: https://github.com/your-org
    publisher_support_url: https://github.com/your-org/gdev/issues
    package_identifier: YourOrg.gdev
    repository:
      owner: microsoft
      name: winget-pkgs
      branch: main
      token: "{{ .Env.WINGET_GITHUB_TOKEN }}"
    pull_request:
      enabled: true

# --- Linux Packages ---

nfpms:
  - id: gdev-linux
    package_name: gdev
    file_name_template: "{{ .ConventionalFileName }}"
    vendor: YourOrg
    homepage: https://github.com/your-org/gdev
    maintainer: "DevEx Team <devex@your-org.com>"
    description: "Developer environment bootstrapping tool"
    license: MIT
    formats:
      - deb
      - rpm
      - apk
      - archlinux
    bindir: /usr/local/bin
    contents:
      - src: ./completions/gdev.bash
        dst: /usr/share/bash-completion/completions/gdev
        file_info:
          mode: 0644
      - src: ./completions/gdev.zsh
        dst: /usr/share/zsh/vendor-completions/_gdev
        file_info:
          mode: 0644
      - src: ./completions/gdev.fish
        dst: /usr/share/fish/vendor_completions.d/gdev.fish
        file_info:
          mode: 0644

# --- Signing ---

signs:
  - artifacts: checksum
    cmd: gpg
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sig"
      - "${artifact}"
```

#### GitHub Actions Release Workflow

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.GPG_PASSPHRASE }}

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
          SCOOP_TAP_GITHUB_TOKEN: ${{ secrets.SCOOP_TAP_GITHUB_TOKEN }}
          CHOCOLATEY_API_KEY: ${{ secrets.CHOCOLATEY_API_KEY }}
          WINGET_GITHUB_TOKEN: ${{ secrets.WINGET_GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ secrets.GPG_FINGERPRINT }}
```

### 1.2 Distribution Channels

#### Tier 1: Primary (automated by GoReleaser)

| Channel | Platform | Mechanism | Review Process |
|---------|----------|-----------|----------------|
| **GitHub Releases** | All | Direct download + checksums | None (self-publish) |
| **Homebrew Tap** | macOS, Linux | `brew install your-org/tap/gdev` | None (own tap repo) |
| **Scoop Bucket** | Windows | `scoop bucket add your-org ...; scoop install gdev` | None (own bucket repo) |
| **APT/DEB** | Debian/Ubuntu/Mint | `.deb` packages on GitHub Releases | None |
| **RPM** | Fedora/RHEL | `.rpm` packages on GitHub Releases | None |

#### Tier 2: Secondary (automated but with external review)

| Channel | Platform | Mechanism | Review Process |
|---------|----------|-----------|----------------|
| **Chocolatey** | Windows | `choco install gdev` | Manual review by Chocolatey moderators |
| **Winget** | Windows | `winget install YourOrg.gdev` | PR to microsoft/winget-pkgs |
| **AUR** | Arch Linux | `yay -S gdev` or `paru -S gdev` | Community maintained |

#### Tier 3: Not Recommended for CLI Tools

| Channel | Why Not |
|---------|---------|
| **Snap** | Good for server daemons and GUI apps, overkill for CLI tools. Confinement model adds friction. |
| **Flatpak** | Built exclusively for desktop GUI apps. Not suitable for CLI tools. |
| **AppImage** | Single-file executables, but Go already produces these. No value added for CLI tools. |
| **npm wrapper** | Works (mise does this) but adds Node.js as a dependency, which is ironic for a tool that might install Node.js. |

#### Tier 4: Direct Install Scripts (highest priority for gdev)

The `curl | sh` pattern is the most important channel for gdev because it requires zero prerequisites on the target machine. Every other channel requires the user to already have a package manager installed.

### 1.3 Install Script Pattern

#### Bash Install Script (macOS + Linux)

```bash
#!/bin/sh
# install.sh - gdev installer
# Usage: curl -fsSL https://get.gdev.dev | sh
#    or: curl -fsSL https://raw.githubusercontent.com/your-org/gdev/main/install.sh | sh

set -e

# --- Configuration ---
GITHUB_ORG="your-org"
GITHUB_REPO="gdev"
INSTALL_DIR="${GDEV_INSTALL_DIR:-$HOME/.gdev/bin}"

# --- Platform Detection ---
detect_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    linux*)  echo "linux" ;;
    darwin*) echo "darwin" ;;
    mingw*|msys*|cygwin*) echo "windows" ;;
    *)
      echo "Error: Unsupported operating system: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64)  echo "x86_64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l)        echo "armv7" ;;
    *)
      echo "Error: Unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

detect_extension() {
  if [ "$1" = "windows" ]; then
    echo "zip"
  else
    echo "tar.gz"
  fi
}

# --- Version Detection ---
get_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest" |
      grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/'
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest" |
      grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/'
  else
    echo "Error: curl or wget is required" >&2
    exit 1
  fi
}

# --- Download and Install ---
download_and_install() {
  local version="$1"
  local os="$2"
  local arch="$3"
  local ext="$4"

  local filename="${GITHUB_REPO}_${version}_$(echo "$os" | sed 's/.*/\u&/')_${arch}.${ext}"
  local url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${version}/${filename}"
  local checksum_url="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  echo "Downloading gdev v${version} for ${os}/${arch}..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "${tmp_dir}/${filename}"
    curl -fsSL "$checksum_url" -o "${tmp_dir}/checksums.txt"
  else
    wget -q "$url" -O "${tmp_dir}/${filename}"
    wget -q "$checksum_url" -O "${tmp_dir}/checksums.txt"
  fi

  # Verify checksum
  echo "Verifying checksum..."
  local expected_checksum
  expected_checksum=$(grep "$filename" "${tmp_dir}/checksums.txt" | awk '{print $1}')
  local actual_checksum
  if command -v sha256sum >/dev/null 2>&1; then
    actual_checksum=$(sha256sum "${tmp_dir}/${filename}" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    actual_checksum=$(shasum -a 256 "${tmp_dir}/${filename}" | awk '{print $1}')
  else
    echo "Warning: Cannot verify checksum (no sha256sum or shasum)" >&2
  fi

  if [ -n "$actual_checksum" ] && [ "$expected_checksum" != "$actual_checksum" ]; then
    echo "Error: Checksum verification failed!" >&2
    echo "  Expected: $expected_checksum" >&2
    echo "  Got:      $actual_checksum" >&2
    exit 1
  fi

  # Extract
  echo "Installing to ${INSTALL_DIR}..."
  mkdir -p "$INSTALL_DIR"
  if [ "$ext" = "zip" ]; then
    unzip -oq "${tmp_dir}/${filename}" -d "${tmp_dir}/extracted"
  else
    tar -xzf "${tmp_dir}/${filename}" -C "${tmp_dir}/extracted" 2>/dev/null ||
      mkdir -p "${tmp_dir}/extracted" && tar -xzf "${tmp_dir}/${filename}" -C "${tmp_dir}/extracted"
  fi

  cp "${tmp_dir}/extracted/gdev" "${INSTALL_DIR}/gdev"
  chmod +x "${INSTALL_DIR}/gdev"

  echo ""
  echo "gdev v${version} installed to ${INSTALL_DIR}/gdev"

  # PATH setup
  if ! echo "$PATH" | tr ':' '\n' | grep -q "^${INSTALL_DIR}$"; then
    echo ""
    echo "Add gdev to your PATH by adding this to your shell profile:"
    echo ""
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""

    # Auto-detect and offer to update shell rc
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.config/fish/config.fish"; do
      if [ -f "$rc" ]; then
        printf "Add to %s? [y/N] " "$rc"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
          if echo "$rc" | grep -q "fish"; then
            echo "set -gx PATH ${INSTALL_DIR} \$PATH" >> "$rc"
          else
            echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "$rc"
          fi
          echo "Added! Restart your shell or run: source $rc"
        fi
      fi
    done
  fi
}

# --- Main ---
main() {
  local version="${QSDEV_VERSION:-$(get_latest_version)}"
  local os
  os=$(detect_os)
  local arch
  arch=$(detect_arch)
  local ext
  ext=$(detect_extension "$os")

  echo "gdev installer"
  echo "=============="
  echo "  Version: v${version}"
  echo "  OS:      ${os}"
  echo "  Arch:    ${arch}"
  echo ""

  download_and_install "$version" "$os" "$arch" "$ext"
}

main "$@"
```

#### PowerShell Install Script (Windows)

```powershell
# install.ps1 - gdev installer for Windows
# Usage: irm https://get.gdev.dev/windows | iex
#    or: irm https://raw.githubusercontent.com/your-org/gdev/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$GithubOrg = "your-org"
$GithubRepo = "gdev"
$InstallDir = if ($env:GDEV_INSTALL_DIR) { $env:GDEV_INSTALL_DIR } else { "$env:LOCALAPPDATA\gdev\bin" }

function Get-LatestVersion {
    $release = Invoke-RestMethod "https://api.github.com/repos/$GithubOrg/$GithubRepo/releases/latest"
    return $release.tag_name -replace '^v', ''
}

function Install-Gdev {
    $version = if ($env:QSDEV_VERSION) { $env:QSDEV_VERSION } else { Get-LatestVersion }
    $arch = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i386" }
    $filename = "${GithubRepo}_${version}_Windows_${arch}.zip"
    $url = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v$version/$filename"
    $checksumUrl = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v$version/checksums.txt"

    Write-Host "gdev installer" -ForegroundColor Cyan
    Write-Host "==============" -ForegroundColor Cyan
    Write-Host "  Version: v$version"
    Write-Host "  OS:      windows"
    Write-Host "  Arch:    $arch"
    Write-Host ""

    $tmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    try {
        Write-Host "Downloading gdev v$version..."
        Invoke-WebRequest -Uri $url -OutFile "$tmpDir\$filename"
        Invoke-WebRequest -Uri $checksumUrl -OutFile "$tmpDir\checksums.txt"

        # Verify checksum
        Write-Host "Verifying checksum..."
        $expectedHash = (Get-Content "$tmpDir\checksums.txt" | Where-Object { $_ -match $filename }) -split '\s+' | Select-Object -First 1
        $actualHash = (Get-FileHash "$tmpDir\$filename" -Algorithm SHA256).Hash.ToLower()
        if ($expectedHash -ne $actualHash) {
            throw "Checksum verification failed! Expected: $expectedHash, Got: $actualHash"
        }

        # Extract and install
        Write-Host "Installing to $InstallDir..."
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Expand-Archive -Path "$tmpDir\$filename" -DestinationPath "$tmpDir\extracted" -Force
        Copy-Item "$tmpDir\extracted\gdev.exe" "$InstallDir\gdev.exe" -Force

        # Add to PATH
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($currentPath -notlike "*$InstallDir*") {
            [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$currentPath", "User")
            Write-Host ""
            Write-Host "Added $InstallDir to your PATH." -ForegroundColor Green
            Write-Host "Restart your terminal for the change to take effect."
        }

        Write-Host ""
        Write-Host "gdev v$version installed successfully!" -ForegroundColor Green
    }
    finally {
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    }
}

Install-Gdev
```

### 1.4 Code Signing

#### macOS Notarization

**Requirements:**
- Apple Developer Account ($99/year)
- Developer ID Application certificate
- Xcode 11+ (for the build machine running GoReleaser)

**Process:**
1. Create Developer ID certificate via Xcode
2. Sign binary: `codesign -s "<CERTIFICATE-ID>" -o runtime -v gdev`
3. Submit: `xcrun notarytool submit gdev.zip --apple-id <EMAIL> --password <APP-PASSWORD> --team-id <TEAM-ID> --wait`

**CI Integration:**
The `gon` tool by Mitchell Hashimoto automates this. GoReleaser can call `gon` as a post-build hook. However, `gon` requires a macOS runner (GitHub Actions `macos-latest`), so you need a split pipeline: build on Linux, sign on macOS.

**Pragmatic alternative for enterprise internal tools:** Unsigned binaries trigger a Gatekeeper warning but can be run via right-click > Open, or `xattr -dr com.apple.quarantine gdev`. Document this in installation instructions. Most enterprise macOS machines have MDM that can whitelist binaries.

#### Windows Authenticode

**Azure Trusted Signing (recommended):**
- $9.99/month via Azure
- Available to individuals and businesses (US, Canada, EU, UK)
- CRITICAL: Timestamps are mandatory. Azure certificates expire in 3 days; without timestamping, signatures die with the certificate.
- Process: Azure account > Trusted Signing resource > identity verification > SignTool + DLIB

**Pragmatic alternative for enterprise internal tools:** Windows SmartScreen warnings can be bypassed by running from terminal. For enterprise environments, push the binary via SCCM/Intune with a whitelist policy. Internal distribution over a trusted network often makes signing optional.

#### GPG Signing (All Platforms)

GoReleaser generates checksums.txt and can GPG-sign it. Users verify with:
```bash
gpg --verify checksums.txt.sig checksums.txt
sha256sum --check checksums.txt
```

**Recommendation for gdev:** Start with GPG-signed checksums only. Add macOS notarization and Windows Authenticode only if SmartScreen/Gatekeeper warnings cause enough friction to justify the complexity and cost.

### 1.5 Hosted APT/RPM Repository (Optional Enhancement)

For a more seamless Linux experience, you can host your own APT and RPM repositories so users do a one-time setup and then get updates via their package manager:

```bash
# User adds your repo once
curl -fsSL https://get.gdev.dev/gpg | sudo gpg --dearmor -o /usr/share/keyrings/gdev-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/gdev-archive-keyring.gpg] https://apt.gdev.dev stable main" | sudo tee /etc/apt/sources.list.d/gdev.list
sudo apt update && sudo apt install gdev

# Then updates come naturally
sudo apt upgrade gdev
```

This can be hosted on GitHub Pages, Cloudflare R2, or any static hosting. Tools like `aptly` or `createrepo` generate the repository metadata. GoReleaser Pro has built-in support for publishing to custom APT/RPM repos.

---

## 2. Self-Bootstrapping Installer Patterns

### 2.1 How Major Tools Bootstrap

| Tool | Language | Install Method | First-Run Bootstrap | Update Mechanism |
|------|----------|---------------|---------------------|------------------|
| **rustup** | Rust | `curl \| sh` downloads rustup-init binary | rustup-init downloads Rust toolchain | `rustup update` |
| **mise** | Rust | `curl \| sh`, brew, apt, npm, scoop | Downloads tool versions on first use | `mise self-update` |
| **volta** | Rust | `curl \| sh` installs shims | Shims download correct Node version on first use | `volta setup` |
| **devenv** | Nix | Requires Nix first, then `nix profile install` | `devenv init` scaffolds project | `nix profile upgrade` |
| **nix installer** | Rust | `curl \| sh` runs static Rust binary | Creates /nix store, daemon, users | `/nix/nix-installer uninstall` + reinstall |
| **Terraform** | Go | brew, apt, choco, direct download | N/A (standalone) | Download new version |
| **kubectl** | Go | brew, apt, choco, direct download | N/A (standalone) | Download new version |
| **Claude Code** | Node.js | `npm install -g @anthropic-ai/claude-code` | First run prompts for API key/auth | `npm update -g @anthropic-ai/claude-code` |

### 2.2 The "First Tool" Problem

gdev faces a bootstrapping paradox: it needs Go to build from source, but gdev is the tool that sets up the Go environment. The solution used by every successful tool is the same: **do not require building from source for installation**.

**Resolution:** Ship pre-compiled static binaries for every target platform. The install script downloads the correct pre-built binary. Go is only needed by gdev's developers, not its users.

Key design decisions:
- `CGO_ENABLED=0` produces fully static binaries (no libc dependency)
- Go's cross-compilation handles all OS/arch combinations from a single build machine
- The binary is self-contained: no runtime dependencies, no shared libraries, no interpreter

### 2.3 Embedded Assets with embed.FS

Go 1.16+ provides `embed.FS` for bundling files directly into the binary at compile time. This is critical for gdev, which needs to ship templates, default configurations, and possibly shell completions.

```go
package main

import (
    "embed"
    "io/fs"
)

//go:embed templates/*
var templates embed.FS

//go:embed configs/default.yaml
var defaultConfig []byte

//go:embed completions/*
var completions embed.FS

func extractTemplate(name string, destPath string) error {
    data, err := templates.ReadFile("templates/" + name)
    if err != nil {
        return fmt.Errorf("reading embedded template %s: %w", name, err)
    }
    return os.WriteFile(destPath, data, 0644)
}
```

**What to embed in gdev:**
- Default configuration templates (`.qsdev.yaml` scaffolding)
- Shell completion scripts (bash, zsh, fish)
- Pre-commit hook configurations
- Devenv/flake templates for new projects
- Skill/recipe definitions for the `qsdev init` workflow

**Trade-offs:**
- Every embedded file increases binary size directly (1:1 ratio)
- For typical config/template files (tens of KB), this is negligible
- For large assets (>1MB), consider downloading on first use instead
- Embedded files are immutable -- they update only when the binary updates

### 2.4 First-Run Bootstrap Sequence

Based on patterns from rustup, mise, and volta, here is the recommended first-run sequence for gdev:

```
qsdev devenv doctor    # Diagnose what's missing
qsdev devenv setup     # Interactive bootstrap (installs prerequisites)
qsdev init             # Scaffold a project's dev environment
```

The `qsdev devenv doctor` command is critical infrastructure. It should:
1. Detect the OS, distro, and architecture
2. Check for each prerequisite (git, nix, devenv, direnv, pre-commit, Claude Code)
3. Report what's present, what's missing, and what version is installed
4. Output actionable commands for fixing each issue
5. Return a non-zero exit code if anything critical is missing

The `qsdev devenv setup` command should:
1. Run `qsdev devenv doctor` to identify gaps
2. Present a plan: "I will install X, Y, Z using [package manager]. Continue? [Y/n]"
3. Handle privilege escalation (sudo) only when needed
4. Install prerequisites in dependency order
5. Configure shell integration (PATH, completions, direnv hooks)
6. Run `qsdev devenv doctor` again to verify

---

## 3. OS Detection and Package Manager Abstraction in Go

### 3.1 Runtime OS/Arch Detection

Go provides built-in constants at compile time and runtime:

```go
import "runtime"

func main() {
    os := runtime.GOOS     // "linux", "darwin", "windows"
    arch := runtime.GOARCH  // "amd64", "arm64"
}
```

For Linux distro detection, parse `/etc/os-release`:

```go
package sysinfo

import (
    "bufio"
    "os"
    "strings"
)

type OSRelease struct {
    ID         string // "ubuntu", "fedora", "arch", "debian"
    IDLike     string // "debian", "rhel fedora"
    VersionID  string // "22.04", "39"
    PrettyName string // "Ubuntu 22.04.3 LTS"
    Name       string // "Ubuntu"
}

func ParseOSRelease() (*OSRelease, error) {
    paths := []string{"/etc/os-release", "/usr/lib/os-release"}
    var f *os.File
    var err error
    for _, path := range paths {
        f, err = os.Open(path)
        if err == nil {
            break
        }
    }
    if err != nil {
        return nil, fmt.Errorf("could not open os-release: %w", err)
    }
    defer f.Close()

    release := &OSRelease{}
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        key, value, ok := strings.Cut(line, "=")
        if !ok {
            continue
        }
        value = strings.Trim(value, "\"")
        switch key {
        case "ID":
            release.ID = value
        case "ID_LIKE":
            release.IDLike = value
        case "VERSION_ID":
            release.VersionID = value
        case "PRETTY_NAME":
            release.PrettyName = value
        case "NAME":
            release.Name = value
        }
    }
    return release, scanner.Err()
}
```

### 3.2 Existing Go Libraries

| Library | Purpose | Strengths | Weaknesses |
|---------|---------|-----------|------------|
| `wille/osutil` | OS name/version detection | Cross-platform, clean API | No package manager detection |
| `Hayao0819/go-distro` | Linux distro detection | Uses os-release + package manager hints | Linux-only |
| `makifdb/packer` | Package manager abstraction | Install/Remove/Check API, 8 managers | Small project, may be unmaintained |
| `quay/claircore/osrelease` | os-release parsing | Battle-tested (Clair scanner) | Heavy dependency for just parsing |
| `zcalusic/sysinfo` | Full system info | OS, kernel, hardware | Linux-only, heavy |

**Recommendation:** Do not depend on any of these small libraries. The os-release parsing is ~40 lines of Go (shown above). Package manager detection is similarly simple. Rolling your own avoids supply chain risk and maintenance burden from tiny, low-activity dependencies.

### 3.3 Package Manager Detection and Abstraction

```go
package pkgmanager

import (
    "fmt"
    "os/exec"
    "runtime"
)

type PackageManager interface {
    Name() string
    Install(pkg string) *exec.Cmd
    IsInstalled(pkg string) bool
    NeedsSudo() bool
}

type Apt struct{}
func (a Apt) Name() string                 { return "apt" }
func (a Apt) Install(pkg string) *exec.Cmd { return exec.Command("sudo", "apt", "install", "-y", pkg) }
func (a Apt) IsInstalled(pkg string) bool  { return exec.Command("dpkg", "-l", pkg).Run() == nil }
func (a Apt) NeedsSudo() bool              { return true }

type Dnf struct{}
func (d Dnf) Name() string                 { return "dnf" }
func (d Dnf) Install(pkg string) *exec.Cmd { return exec.Command("sudo", "dnf", "install", "-y", pkg) }
func (d Dnf) IsInstalled(pkg string) bool  { return exec.Command("rpm", "-q", pkg).Run() == nil }
func (d Dnf) NeedsSudo() bool              { return true }

type Pacman struct{}
func (p Pacman) Name() string                 { return "pacman" }
func (p Pacman) Install(pkg string) *exec.Cmd { return exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg) }
func (p Pacman) IsInstalled(pkg string) bool  { return exec.Command("pacman", "-Q", pkg).Run() == nil }
func (p Pacman) NeedsSudo() bool              { return true }

type Brew struct{}
func (b Brew) Name() string                 { return "brew" }
func (b Brew) Install(pkg string) *exec.Cmd { return exec.Command("brew", "install", pkg) }
func (b Brew) IsInstalled(pkg string) bool  { return exec.Command("brew", "list", pkg).Run() == nil }
func (b Brew) NeedsSudo() bool              { return false }

type Choco struct{}
func (c Choco) Name() string                 { return "choco" }
func (c Choco) Install(pkg string) *exec.Cmd { return exec.Command("choco", "install", "-y", pkg) }
func (c Choco) IsInstalled(pkg string) bool  { return exec.Command("choco", "list", "--local-only", pkg).Run() == nil }
func (c Choco) NeedsSudo() bool              { return true } // needs elevated prompt

type Scoop struct{}
func (s Scoop) Name() string                 { return "scoop" }
func (s Scoop) Install(pkg string) *exec.Cmd { return exec.Command("scoop", "install", pkg) }
func (s Scoop) IsInstalled(pkg string) bool  { return exec.Command("scoop", "info", pkg).Run() == nil }
func (s Scoop) NeedsSudo() bool              { return false }

// DetectPackageManager returns the best available package manager for the current system.
func DetectPackageManager() (PackageManager, error) {
    switch runtime.GOOS {
    case "darwin":
        if commandExists("brew") {
            return Brew{}, nil
        }
        return nil, fmt.Errorf("Homebrew not found; install from https://brew.sh")

    case "linux":
        // Check in order of specificity
        if commandExists("apt") {
            return Apt{}, nil
        }
        if commandExists("dnf") {
            return Dnf{}, nil
        }
        if commandExists("pacman") {
            return Pacman{}, nil
        }
        if commandExists("brew") {
            return Brew{}, nil // Linuxbrew
        }
        return nil, fmt.Errorf("no supported package manager found")

    case "windows":
        if commandExists("scoop") {
            return Scoop{}, nil
        }
        if commandExists("choco") {
            return Choco{}, nil
        }
        if commandExists("winget") {
            // winget needs its own implementation
        }
        return nil, fmt.Errorf("no supported package manager found; install Scoop from https://scoop.sh")

    default:
        return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
    }
}

func commandExists(name string) bool {
    _, err := exec.LookPath(name)
    return err == nil
}
```

### 3.4 Package Name Mapping

Different package managers use different names for the same tool. gdev needs a mapping table:

```go
// PackageMapping maps a canonical tool name to its package name per manager.
type PackageMapping struct {
    Canonical string            // Our internal name
    Binary    string            // Binary name to check in PATH
    Packages  map[string]string // Package manager -> package name
    // Some tools need special installation (not via package manager)
    CustomInstall func() error
}

var ToolRegistry = map[string]PackageMapping{
    "git": {
        Canonical: "git",
        Binary:    "git",
        Packages: map[string]string{
            "apt": "git", "dnf": "git", "pacman": "git",
            "brew": "git", "choco": "git", "scoop": "git",
        },
    },
    "direnv": {
        Canonical: "direnv",
        Binary:    "direnv",
        Packages: map[string]string{
            "apt": "direnv", "dnf": "direnv", "pacman": "direnv",
            "brew": "direnv", "choco": "direnv", "scoop": "direnv",
        },
    },
    "pre-commit": {
        Canonical: "pre-commit",
        Binary:    "pre-commit",
        Packages: map[string]string{
            "apt": "pre-commit", "dnf": "pre-commit", "pacman": "python-pre-commit",
            "brew": "pre-commit",
        },
        // Not available on choco/scoop -- needs pip install
    },
    "nix": {
        Canonical: "nix",
        Binary:    "nix",
        // Not installed via package managers -- uses Determinate Systems installer
        CustomInstall: installNix,
    },
    "devenv": {
        Canonical: "devenv",
        Binary:    "devenv",
        // Installed via nix profile
        CustomInstall: installDevenv,
    },
    "claude-code": {
        Canonical: "claude-code",
        Binary:    "claude",
        // Installed via npm
        CustomInstall: installClaudeCode,
    },
}
```

### 3.5 Privilege Escalation Patterns

```go
package privilege

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"
)

// ElevatedCommand wraps a command with the appropriate privilege escalation
// for the current platform.
func ElevatedCommand(name string, args ...string) *exec.Cmd {
    switch runtime.GOOS {
    case "linux", "darwin":
        if os.Getuid() == 0 {
            return exec.Command(name, args...)
        }
        allArgs := append([]string{name}, args...)
        return exec.Command("sudo", allArgs...)

    case "windows":
        // On Windows, the process must already be elevated.
        // gdev should detect this and prompt the user to re-run
        // from an elevated terminal if needed.
        return exec.Command(name, args...)

    default:
        return exec.Command(name, args...)
    }
}

// IsElevated returns true if the current process has root/admin privileges.
func IsElevated() bool {
    switch runtime.GOOS {
    case "linux", "darwin":
        return os.Getuid() == 0
    case "windows":
        // Check for admin by attempting to open a privileged resource
        cmd := exec.Command("net", "session")
        return cmd.Run() == nil
    default:
        return false
    }
}

// RequestElevation prompts the user to re-run with elevated privileges.
func RequestElevation(reason string) error {
    switch runtime.GOOS {
    case "linux", "darwin":
        fmt.Printf("Root privileges needed: %s\n", reason)
        fmt.Println("You may be prompted for your password.")
        return nil // sudo will prompt inline
    case "windows":
        if !IsElevated() {
            return fmt.Errorf(
                "administrator privileges required: %s\n"+
                    "Please re-run this command from an elevated terminal:\n"+
                    "  Right-click Terminal > Run as administrator", reason)
        }
        return nil
    default:
        return fmt.Errorf("unsupported platform")
    }
}
```

---

## 4. Prerequisite Installation Strategies

### 4.1 Prerequisite Map

| Tool | Purpose | Auto-Install? | Method | Notes |
|------|---------|--------------|--------|-------|
| **Git** | Version control | Yes | Package manager | Available everywhere |
| **Go** | Language runtime | Conditional | Package manager or mise | Only needed if building Go projects |
| **Nix** | Package manager | Yes (macOS/Linux) | Determinate Systems installer | Not available natively on Windows |
| **devenv** | Dev environment | Yes (after Nix) | `nix profile install` | Requires Nix |
| **direnv** | Auto-env loading | Yes | Package manager | Available everywhere |
| **Claude Code** | AI assistant | Yes | `npm install -g` | Requires Node.js |
| **Node.js** | JS runtime | Conditional | mise or package manager | Required for Claude Code |
| **pre-commit** | Git hooks | Yes | Package manager or pip | Python-based |
| **Python** | pre-commit dep | Conditional | Package manager | Only if pre-commit needed |

### 4.2 Installation Dependency Graph

```
qsdev devenv setup
├── git (package manager)
├── nix (Determinate Systems installer)
│   └── devenv (nix profile install)
├── direnv (package manager)
├── node (mise or package manager)
│   └── claude-code (npm install -g)
└── pre-commit (package manager or pip)
```

### 4.3 Nix on Non-NixOS Systems

The Determinate Systems Nix installer is the recommended approach for all non-NixOS systems:

```go
func installNix() error {
    if runtime.GOOS == "windows" {
        return fmt.Errorf(
            "Nix is not available natively on Windows.\n" +
            "Options:\n" +
            "  1. Use WSL2: wsl --install, then run qsdev devenv setup inside WSL\n" +
            "  2. Use devcontainers with Nix pre-installed\n" +
            "  3. Skip Nix-based features (gdev will use native package managers)")
    }

    fmt.Println("Installing Nix via Determinate Systems installer...")
    cmd := exec.Command("sh", "-c",
        "curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install")
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

**Platform support:**
- macOS: Full support (graphical .pkg installer available too)
- Linux (all distros): Full support
- WSL2: Full support with `--init none` flag if systemd not available
- Windows native: Not supported. Must use WSL2.

### 4.4 Windows Strategy: WSL2 vs Native

For a consulting firm, the Windows story needs careful thought:

**Option A: WSL2-first (recommended)**
- gdev on Windows detects if WSL2 is available
- If yes: installs gdev inside WSL2 and runs all dev tools there
- If no: offers to enable WSL2 (`wsl --install`)
- Nix, devenv, direnv, and most Unix tools work natively in WSL2
- This is what most enterprise dev teams use in practice

**Option B: Native Windows**
- Use Scoop/Chocolatey for native Windows tools
- Cannot use Nix/devenv -- must fall back to native alternatives
- Git, Go, Node.js, Python all work natively
- Pre-commit works via pip
- direnv has a Windows port but is less reliable

**Option C: Hybrid (what gdev should do)**
- Detect the environment:
  - Running inside WSL2? Use Linux path.
  - Running on native Windows? Offer WSL2 or native-only mode.
- `qsdev devenv doctor` reports which features are available in each mode
- `qsdev devenv setup --wsl` explicitly targets WSL2
- `qsdev devenv setup --native` explicitly targets native Windows

### 4.5 Multi-Runtime Management

For tools that need specific versions of runtimes (Go 1.22, Node 20, Python 3.12), the best patterns are:

**Option A: Delegate to mise (recommended)**
mise handles multi-runtime management cross-platform:
```bash
# .mise.toml in each project
[tools]
go = "1.22"
node = "20"
python = "3.12"
```
gdev can install mise, and projects use `.mise.toml` for version pinning.

**Option B: Delegate to devenv/Nix**
Nix provides perfectly reproducible environments:
```nix
# devenv.nix
{ pkgs, ... }: {
  languages.go.enable = true;
  languages.go.package = pkgs.go_1_22;
  languages.javascript.enable = true;
}
```

**Option C: Direct version management in gdev**
Not recommended. This is a solved problem -- defer to mise or devenv rather than reimplementing.

**Recommendation:** Support both mise and devenv. Use mise as the default (works everywhere including native Windows), and devenv for teams that want Nix-based reproducibility.

---

## 5. Prior Art and Patterns

### 5.1 Terraform / OpenTofu

**Distribution approach:**
- GitHub Releases with pre-built binaries for all platforms
- Official APT/RPM repositories (HashiCorp manages their own)
- Homebrew formula in the official homebrew-core
- Chocolatey package
- Direct download from releases.hashicorp.com
- `hc-install` Go library for programmatic downloading

**Key pattern:** HashiCorp hosts their own download server (releases.hashicorp.com) with GPG-signed SHA256SUMS files. Every binary is signed. This is the gold standard for enterprise distribution.

**What gdev can learn:** For an internal enterprise tool, GitHub Releases + a Homebrew tap + Scoop bucket covers 90% of users. You do not need a custom download server.

### 5.2 kubectl

**Distribution approach:**
- Direct download from dl.k8s.io
- Homebrew
- Chocolatey, Scoop
- APT/RPM repos (Google-hosted)
- Snap
- `go install` for source builds

**Key pattern:** kubectl is a single static binary with no dependencies. It does not bootstrap anything. Distribution is purely about getting the binary onto the machine.

**What gdev can learn:** The single-binary model works at massive scale. kubectl serves millions of users with just pre-built binaries and package manager integrations.

### 5.3 Helm

**Distribution approach:**
- Install script: `curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash`
- Homebrew, Chocolatey, Scoop, Snap, Winget
- GoFish, APT, pkg (FreeBSD)
- Direct download from get.helm.sh

**Key pattern:** Helm's install script is a good reference implementation. It detects OS/arch, downloads from GitHub Releases, verifies checksum, and installs to `/usr/local/bin`. The script is well-tested across many environments.

### 5.4 devenv.sh

**Distribution approach:**
- Requires Nix as a prerequisite (cannot bootstrap without it)
- Installed via: `nix profile install --accept-flake-config nixpkgs#devenv`
- Getting Started guide points to Determinate Systems installer for Nix

**Key pattern:** devenv accepts the "first tool" problem by requiring Nix upfront. This limits adoption to teams already bought into Nix. gdev should NOT follow this pattern -- it needs to work without any prerequisites.

### 5.5 Claude Code

**Distribution approach:**
- `npm install -g @anthropic-ai/claude-code`
- Requires Node.js 18+ as a prerequisite
- First run prompts for authentication (API key or OAuth)
- Self-updates via npm

**Key pattern:** npm distribution works well for the Node.js ecosystem but creates a hard dependency on Node.js. For gdev, Claude Code is a downstream dependency to install, not a distribution model to follow.

### 5.6 rustup

**Distribution approach:**
- `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- Downloads platform-specific rustup-init binary
- rustup-init is a compiled Rust binary (not a shell script)
- Windows: separate .exe download page
- Manages its own updates: `rustup self update`

**Key pattern:** The two-stage bootstrap: shell script downloads a compiled binary, compiled binary does the real work. This avoids writing complex logic in shell and gives you proper error handling, TUI, and cross-platform support.

**This is the recommended pattern for gdev.** The install script should be a thin shell wrapper that downloads the gdev binary. The gdev binary itself handles all setup logic.

### 5.7 mise (formerly rtx)

**Distribution approach:**
- `curl https://mise.run | sh` (primary)
- Homebrew, apt, dnf, pacman, apk, Scoop, winget, npm, cargo
- GitHub Releases direct download
- Shell-specific variants that auto-configure activation

**Key pattern:** mise covers every distribution channel. The `mise.run` domain hosts an install script that is the canonical entry point. The script is versioned and can be pinned with checksums for CI reproducibility.

**What gdev can learn:** mise's breadth of distribution channels is ideal. The tiered approach (curl|sh as primary, package managers as secondary) maximizes reach. The shell-specific variants (`mise.run/bash`, `mise.run/zsh`) that auto-configure activation are a nice touch.

---

## 6. Recommended Architecture for gdev

### 6.1 Distribution Strategy (Priority Order)

1. **Install script** (`curl | sh` / PowerShell) -- Zero-prerequisite path. This is the primary distribution method and what the README should lead with.

2. **GitHub Releases** -- Every version produces binaries for all platforms, checksums, and GPG signatures. This is the source of truth for all other channels.

3. **Homebrew Tap** -- `brew install your-org/tap/gdev`. Covers macOS and Linuxbrew users. Own tap repo for fast publishing (no review process).

4. **Scoop Bucket** -- `scoop install gdev` via your own bucket. Covers Windows power users. No review process.

5. **APT/RPM/Arch packages** -- `.deb` and `.rpm` files attached to GitHub Releases for direct download. Optionally host your own APT/RPM repo for seamless updates.

6. **Chocolatey** -- `choco install gdev`. Broader Windows coverage. Manual review process (slower).

7. **Winget** -- `winget install YourOrg.gdev`. PR to microsoft/winget-pkgs. Slowest but widest Windows reach.

### 6.2 Build Pipeline

```
git tag v1.0.0
    │
    ▼
GitHub Actions triggered
    │
    ├─► GoReleaser builds binaries (linux/darwin/windows x amd64/arm64)
    │   ├─► Archives (.tar.gz, .zip)
    │   ├─► Linux packages (.deb, .rpm, .apk, .archlinux)
    │   ├─► Checksums (SHA256)
    │   └─► GPG signature on checksums
    │
    ├─► GitHub Release created with all artifacts
    │
    ├─► Homebrew tap repo updated (PR or direct push)
    ├─► Scoop bucket repo updated
    ├─► Chocolatey nupkg pushed
    └─► Winget PR opened to microsoft/winget-pkgs
```

### 6.3 Recommended GoReleaser + GitHub Actions Setup

Use the complete `.goreleaser.yaml` from Section 1.1 above. The GitHub Actions workflow from Section 1.1 triggers on tag push and handles everything.

**Required GitHub secrets:**
- `GITHUB_TOKEN` (automatic)
- `HOMEBREW_TAP_GITHUB_TOKEN` (PAT with repo scope for tap repo)
- `SCOOP_TAP_GITHUB_TOKEN` (PAT with repo scope for bucket repo)
- `CHOCOLATEY_API_KEY` (from chocolatey.org account)
- `WINGET_GITHUB_TOKEN` (PAT with repo scope for microsoft/winget-pkgs fork)
- `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` (for checksum signing)

### 6.4 Binary Architecture

```
gdev binary
├── cmd/gdev/main.go          # Entry point
├── internal/
│   ├── doctor/                # System diagnostics (qsdev devenv doctor)
│   │   ├── checks.go         # Individual prerequisite checks
│   │   └── report.go         # Diagnostic output formatting
│   ├── setup/                 # Bootstrap logic (qsdev devenv setup)
│   │   ├── installer.go      # Orchestrates prerequisite installation
│   │   └── shell.go          # Shell integration (PATH, completions)
│   ├── sysinfo/               # OS/distro/arch detection
│   │   ├── detect.go         # runtime.GOOS/GOARCH + os-release parsing
│   │   └── detect_test.go
│   ├── pkgmanager/            # Package manager abstraction
│   │   ├── interface.go      # PackageManager interface
│   │   ├── apt.go            # Apt implementation
│   │   ├── dnf.go            # Dnf implementation
│   │   ├── pacman.go         # Pacman implementation
│   │   ├── brew.go           # Homebrew implementation
│   │   ├── scoop.go          # Scoop implementation
│   │   ├── choco.go          # Chocolatey implementation
│   │   ├── detect.go         # Auto-detection logic
│   │   └── registry.go       # Tool -> package name mapping
│   ├── privilege/             # Sudo/UAC handling
│   │   └── escalate.go
│   └── selfupdate/           # Self-update mechanism
│       └── update.go         # Check GitHub Releases for newer version
├── templates/                 # Embedded via embed.FS
│   ├── devenv.nix.tmpl
│   ├── .envrc.tmpl
│   ├── .qsdev.yaml.tmpl
│   └── .pre-commit-config.yaml.tmpl
└── completions/               # Embedded shell completions
    ├── gdev.bash
    ├── gdev.zsh
    └── gdev.fish
```

### 6.5 Self-Update Mechanism

gdev should check for updates periodically (configurable, default weekly):

```go
func CheckForUpdate(currentVersion string) (*Release, error) {
    resp, err := http.Get("https://api.github.com/repos/your-org/gdev/releases/latest")
    if err != nil {
        return nil, err // Silently fail -- don't block the user
    }
    defer resp.Body.Close()

    var release struct {
        TagName string `json:"tag_name"`
        HTMLURL string `json:"html_url"`
    }
    json.NewDecoder(resp.Body).Decode(&release)

    latest := strings.TrimPrefix(release.TagName, "v")
    if semver.Compare("v"+currentVersion, "v"+latest) < 0 {
        return &Release{Version: latest, URL: release.HTMLURL}, nil
    }
    return nil, nil // Already up to date
}
```

On update detection, gdev should print a notice but not auto-update. The user runs `qsdev self-update` to actually update, which re-runs the install script logic (download, verify, replace binary).

### 6.6 Key Design Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Build tool | GoReleaser | Industry standard, handles everything, one config file |
| Primary install | curl \| sh + PowerShell | Zero prerequisites, maximum reach |
| Binary type | Static (CGO_ENABLED=0) | No runtime dependencies, works on any Linux |
| Package managers | Homebrew + Scoop (own repos), Chocolatey + Winget (external) | Own repos = fast, external = wider reach |
| Linux packages | nFPM via GoReleaser (.deb, .rpm) | Covers apt + dnf users |
| Code signing | GPG checksums first, add notarization/Authenticode later | Complexity vs value trade-off |
| Embedded assets | embed.FS for templates and completions | Single binary, no extraction needed |
| OS detection | Custom (40 lines) over third-party libraries | Avoid supply chain risk for trivial code |
| Package manager abstraction | Custom interface with per-manager implementations | Full control, tested, no dependency |
| Nix on Windows | WSL2 path | Nix does not run natively on Windows |
| Runtime management | Delegate to mise (primary) or devenv (Nix users) | Solved problem, do not reimplement |
| Self-update | Check GitHub Releases, manual `qsdev self-update` | Non-intrusive, user-controlled |
| Windows strategy | Hybrid: detect WSL2, offer WSL or native mode | Maximum flexibility |

---

## Sources

### GoReleaser
- [GoReleaser Go Build Configuration](https://goreleaser.com/customization/builds/go/)
- [GoReleaser Homebrew Taps](https://goreleaser.com/customization/homebrew/)
- [GoReleaser Scoop Manifests](https://goreleaser.com/customization/publish/scoop/)
- [GoReleaser Chocolatey Packages](https://goreleaser.com/customization/chocolatey/)
- [GoReleaser Winget](https://goreleaser.com/customization/winget/)
- [GoReleaser nFPM Linux Packages](https://goreleaser.com/customization/nfpm/)
- [GoReleaser Signing](https://goreleaser.com/customization/sign/)
- [GoReleaser GitHub Actions](https://goreleaser.com/ci/actions/)
- [goreleaser-action](https://github.com/goreleaser/goreleaser-action)
- [nFPM](https://nfpm.goreleaser.com/)
- [Multi-Platform GoReleaser Article](https://dev.to/akshitzatakia/from-manual-builds-to-multi-platform-magic-how-goreleaser-transformed-my-opentelemetry-sandbox-h36)

### Code Signing
- [Notarizing Go Binaries for macOS](https://artyom.dev/notarizing-go-binaries-for-macos.md)
- [Authenticode in 2025 - Azure Trusted Signing](https://textslashplain.com/2025/03/12/authenticode-in-2025-azure-trusted-signing/)
- [gon - macOS Notarization Tool](https://github.com/mitchellh/gon)
- [Azure Artifact Signing](https://azure.microsoft.com/en-us/products/artifact-signing)

### Bootstrap Patterns
- [rustup.rs](https://rustup.rs/)
- [Mise Installation](https://mise.jdx.dev/installing-mise.html)
- [Volta](https://volta.sh/)
- [Determinate Systems Nix Installer](https://github.com/DeterminateSystems/nix-installer)
- [devenv Getting Started](https://devenv.sh/getting-started/)
- [hc-install](https://github.com/hashicorp/hc-install)

### OS Detection and Package Managers
- [wille/osutil](https://github.com/wille/osutil)
- [makifdb/packer](https://github.com/makifdb/packer)
- [jpikl/pm](https://github.com/jpikl/pm)
- [Hayao0819/go-distro](https://github.com/Hayao0819/go-distro)
- [quay/claircore/osrelease](https://pkg.go.dev/github.com/quay/claircore/osrelease)

### Distribution Patterns
- [Shipping Go CLI to Every Ecosystem](https://dev.to/_402ccbd6e5cb02871506/shipping-a-go-cli-to-every-ecosystem-github-releases-homebrew-and-npm-5g27)
- [Task Installation](https://taskfile.dev/docs/installation)
- [Terraform Install](https://developer.hashicorp.com/terraform/install)
- [Snap vs Flatpak vs AppImage](https://www.baeldung.com/linux/snaps-flatpak-appimage)
