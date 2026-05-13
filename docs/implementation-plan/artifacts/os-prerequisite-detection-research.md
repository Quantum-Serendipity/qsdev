# OS-Level Prerequisite Detection & Installation Research

Research date: 2026-05-12
Context: gdev CLI tool (Go) for bootstrapping secure development environments across hundreds of engineers at a consulting firm.

---

## Table of Contents

1. [OS/Environment Detection Matrix](#1-osenvironment-detection-matrix)
2. [Tool Prerequisite Mapping](#2-tool-prerequisite-mapping)
3. [Shell Integration](#3-shell-integration)
4. [Windows-Specific Considerations](#4-windows-specific-considerations)
5. [Privilege Escalation Patterns](#5-privilege-escalation-patterns)
6. [Go Implementation Patterns](#6-go-implementation-patterns)

---

## 1. OS/Environment Detection Matrix

### 1.1 Detection Strategy Overview

The detection engine should use a layered approach:

1. **`runtime.GOOS` and `runtime.GOARCH`** — compile-time OS and architecture (fast, always available)
2. **File-based detection** — `/etc/os-release`, `/proc/version`, `sw_vers` (Linux/macOS specifics)
3. **Command-based detection** — `uname`, `lsb_release`, `wmic` (fallback for edge cases)
4. **Environment variable detection** — `$SHELL`, `$WSL_DISTRO_NAME`, `$WT_SESSION`, etc.
5. **Binary existence detection** — `which`/`command -v`/`exec.LookPath` for package managers and tools

### 1.2 macOS

#### OS and Version Detection

| Method | Command/API | Returns |
|--------|-------------|---------|
| Go runtime | `runtime.GOOS` | `"darwin"` |
| macOS version | `sw_vers -productVersion` | e.g., `"15.4.1"` |
| macOS name | `sw_vers -productName` | `"macOS"` |
| Build version | `sw_vers -buildVersion` | e.g., `"24E263"` |
| Architecture | `runtime.GOARCH` | `"arm64"` (Apple Silicon) or `"amd64"` (Intel) |
| Rosetta detection | `sysctl -n sysctl.proc_translated` | `1` if Rosetta, `0` if native, error if Intel |

#### Apple Silicon vs Intel

```go
// In Go:
if runtime.GOARCH == "arm64" {
    // Apple Silicon (M1/M2/M3/M4) — native
} else if runtime.GOARCH == "amd64" {
    // Intel Mac — or possibly Rosetta 2 translation
    // To distinguish: exec sysctl -n sysctl.proc_translated
    // Returns "1" under Rosetta, error on native Intel
}
```

#### Homebrew Detection

| Check | Apple Silicon Path | Intel Path |
|-------|-------------------|------------|
| Default prefix | `/opt/homebrew` | `/usr/local` |
| Binary location | `/opt/homebrew/bin/brew` | `/usr/local/bin/brew` |
| `exec.LookPath("brew")` | Works if in PATH | Works if in PATH |
| Environment var | `$HOMEBREW_PREFIX` | `$HOMEBREW_PREFIX` |

Detection order:
1. `exec.LookPath("brew")` — fast, respects PATH
2. Check `$HOMEBREW_PREFIX` environment variable
3. Check `/opt/homebrew/bin/brew` (Apple Silicon default)
4. Check `/usr/local/bin/brew` (Intel default)

#### Xcode Command Line Tools Detection

```bash
# Check if installed
xcode-select -p
# Returns path like /Library/Developer/CommandLineTools or /Applications/Xcode.app/Contents/Developer
# Exit code 0 = installed, non-zero = not installed

# Check version
pkgutil --pkg-info=com.apple.pkg.CLTools_Executables 2>/dev/null
# Returns version info if installed

# Install (triggers GUI dialog)
xcode-select --install
```

Go implementation: `exec.Command("xcode-select", "-p")` — check exit code.

### 1.3 Windows

#### Native vs WSL2 Detection

**From inside WSL2** (appears as Linux to `runtime.GOOS`):

```go
// runtime.GOOS == "linux" even inside WSL2
// Must check /proc/version or kernel name
func isWSL() bool {
    data, err := os.ReadFile("/proc/version")
    if err != nil {
        return false
    }
    lower := strings.ToLower(string(data))
    return strings.Contains(lower, "microsoft")
}

func isWSL2() bool {
    data, err := os.ReadFile("/proc/version")
    if err != nil {
        return false
    }
    lower := strings.ToLower(string(data))
    // WSL2 uses "microsoft-standard" kernel; WSL1 uses "Microsoft" without "standard"
    return strings.Contains(lower, "microsoft-standard")
}
```

Additional WSL indicators:
- `$WSL_DISTRO_NAME` environment variable (set inside WSL, e.g., `"Ubuntu"`)
- `$WSLENV` environment variable (set when WSL interop is active)
- `/proc/sys/fs/binfmt_misc/WSLInterop` exists in WSL
- `wslpath` command available inside WSL

**From native Windows** (`runtime.GOOS == "windows"`):

| Method | Command/API | Returns |
|--------|-------------|---------|
| Go runtime | `runtime.GOOS` | `"windows"` |
| Windows version | `cmd /c ver` | `"Microsoft Windows [Version 10.0.22631.4890]"` |
| Detailed version | `wmic os get Caption,Version,BuildNumber /value` | Full OS info |
| Architecture | `runtime.GOARCH` | `"amd64"` or `"arm64"` |
| PowerShell version | `$PSVersionTable.PSVersion` | Version object |
| PowerShell detection | `exec.LookPath("pwsh")` (Core) or `exec.LookPath("powershell")` (Windows) | Path or error |
| Git Bash/MSYS2 | `$MSYSTEM` env var | `"MINGW64"`, `"MSYS"`, etc. |
| Windows Terminal | `$WT_SESSION` env var | Non-empty if in Windows Terminal |

#### Windows Package Manager Detection

| Package Manager | Detection Method | Typical Location |
|-----------------|-----------------|------------------|
| **winget** | `exec.LookPath("winget")` | Ships with Windows 11; App Installer on Windows 10 |
| **Chocolatey** | `exec.LookPath("choco")` | `C:\ProgramData\chocolatey\bin\choco.exe` |
| **Scoop** | `exec.LookPath("scoop")` | `%USERPROFILE%\scoop\shims\scoop` |

Priority order for gdev: winget (pre-installed on Win11) > scoop (user-space, no admin) > choco (needs admin for install).

#### PowerShell Version Detection

```go
// Detect PowerShell Core (cross-platform)
if path, err := exec.LookPath("pwsh"); err == nil {
    // PowerShell 7+ (Core) — preferred
    out, _ := exec.Command("pwsh", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()").Output()
}

// Detect Windows PowerShell (5.1, Windows-only)
if path, err := exec.LookPath("powershell"); err == nil {
    // Windows PowerShell 5.1 — legacy
    out, _ := exec.Command("powershell", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()").Output()
}
```

#### Git Bash / MSYS2 Detection

```go
// Check if running inside Git Bash or MSYS2
msystem := os.Getenv("MSYSTEM") // "MINGW64", "MINGW32", "MSYS", "UCRT64", "CLANG64"
mintty := os.Getenv("TERM_PROGRAM") // "mintty" for Git Bash terminal

// MSYS2 root
msysHome := os.Getenv("MSYS2_HOME") // or check C:\msys64
```

#### Admin/Elevation Detection (Windows)

```go
// Cannot use os.Getuid() on Windows (returns -1)
// Must use Windows API via golang.org/x/sys/windows

import "golang.org/x/sys/windows"

func isAdmin() (bool, error) {
    var sid *windows.SID
    err := windows.AllocateAndInitializeSid(
        &windows.SECURITY_NT_AUTHORITY,
        2,
        windows.SECURITY_BUILTIN_DOMAIN_RID,
        windows.DOMAIN_ALIAS_RID_ADMINS,
        0, 0, 0, 0, 0, 0,
        &sid,
    )
    if err != nil {
        return false, err
    }
    defer windows.FreeSid(sid)

    member, err := windows.Token(0).IsMember(sid)
    if err != nil {
        return false, err
    }
    return member, nil
}
```

### 1.4 Debian/Ubuntu

#### Detection

```go
func isDebianBased() bool {
    // Primary: /etc/os-release
    data, err := os.ReadFile("/etc/os-release")
    if err != nil {
        return false
    }
    // Parse ID= and ID_LIKE= fields
    // ID=ubuntu, ID=debian, ID=pop, ID=linuxmint
    // ID_LIKE=debian or ID_LIKE="ubuntu debian"
}
```

Key `/etc/os-release` fields:

| Field | Example (Ubuntu) | Example (Debian) |
|-------|------------------|-------------------|
| `ID` | `ubuntu` | `debian` |
| `ID_LIKE` | `debian` | (absent) |
| `VERSION_ID` | `"24.04"` | `"12"` |
| `VERSION_CODENAME` | `noble` | `bookworm` |
| `PRETTY_NAME` | `"Ubuntu 24.04.2 LTS"` | `"Debian GNU/Linux 12 (bookworm)"` |

Derivatives with `ID_LIKE=debian` or `ID_LIKE="ubuntu debian"`: Pop!_OS, Linux Mint, Elementary OS, Zorin OS, Kali Linux, Raspberry Pi OS.

#### Package Managers

| Manager | Detection | Notes |
|---------|-----------|-------|
| `apt` | `exec.LookPath("apt")` | Modern frontend (Debian 8+, Ubuntu 14.04+) |
| `apt-get` | `exec.LookPath("apt-get")` | Low-level, always available, better for scripts |
| `snap` | `exec.LookPath("snap")` | Pre-installed on Ubuntu; optional on Debian |
| `dpkg` | `exec.LookPath("dpkg")` | Always present; low-level package tool |

Prefer `apt-get` over `apt` in scripts (stable output format, no progress bars).

### 1.5 Fedora / RHEL / CentOS / Rocky / Alma

#### Detection

| Distro | `ID` | `ID_LIKE` | `VERSION_ID` |
|--------|------|-----------|--------------|
| Fedora | `fedora` | (absent) | `"41"` |
| RHEL | `rhel` | `fedora` | `"9.4"` |
| CentOS Stream | `centos` | `rhel fedora` | `"9"` |
| Rocky Linux | `rocky` | `rhel centos fedora` | `"9.4"` |
| AlmaLinux | `almalinux` | `rhel centos fedora` | `"9.4"` |
| Amazon Linux | `amzn` | `fedora` | `"2023"` |
| Oracle Linux | `ol` | `fedora` | `"9.4"` |

Detection strategy: check `ID_LIKE` for `fedora` or `rhel` to catch all derivatives.

#### Package Managers

| Manager | Detection | Notes |
|---------|-----------|-------|
| `dnf` | `exec.LookPath("dnf")` | Fedora 22+, RHEL 8+, all modern derivatives |
| `yum` | `exec.LookPath("yum")` | RHEL 7 and older; symlinked to dnf on modern systems |
| `rpm` | `exec.LookPath("rpm")` | Always present; low-level package tool |

Use `dnf` when available, fall back to `yum`.

### 1.6 Arch / Manjaro

#### Detection

| Distro | `ID` | `ID_LIKE` |
|--------|------|-----------|
| Arch Linux | `arch` | (absent) |
| Manjaro | `manjaro` | `arch` |
| EndeavourOS | `endeavouros` | `arch` |
| Garuda Linux | `garuda` | `arch` |

Arch is rolling release — no `VERSION_ID` field.

#### Package Managers

| Manager | Detection | Notes |
|---------|-----------|-------|
| `pacman` | `exec.LookPath("pacman")` | Always present on Arch-based |
| `yay` | `exec.LookPath("yay")` | Popular AUR helper |
| `paru` | `exec.LookPath("paru")` | Modern AUR helper (Rust-based) |

For AUR packages, detect `yay` or `paru` and prefer `paru` > `yay`. Never invoke `makepkg` directly from gdev.

### 1.7 openSUSE

#### Detection

| Distro | `ID` | `ID_LIKE` | `VERSION_ID` |
|--------|------|-----------|--------------|
| openSUSE Tumbleweed | `opensuse-tumbleweed` | `opensuse suse` | `"20260510"` (date-based) |
| openSUSE Leap | `opensuse-leap` | `opensuse suse` | `"15.6"` |
| SLES | `sles` | `suse` | `"15.6"` |

Tumbleweed is rolling (no stable version); Leap is point release.

#### Package Manager

| Manager | Detection | Notes |
|---------|-----------|-------|
| `zypper` | `exec.LookPath("zypper")` | Always present on SUSE-based |
| `rpm` | `exec.LookPath("rpm")` | Low-level package tool |

### 1.8 Alpine

#### Detection

| Field | Value |
|-------|-------|
| `ID` | `alpine` |
| `VERSION_ID` | `"3.20.0"` |

Note: Alpine uses musl libc, not glibc. This affects binary compatibility — Go static binaries (`CGO_ENABLED=0`) work fine; dynamically linked binaries may not.

#### Package Manager

| Manager | Detection | Notes |
|---------|-----------|-------|
| `apk` | `exec.LookPath("apk")` | Always present on Alpine |

### 1.9 Void Linux

#### Detection

| Field | Value |
|-------|-------|
| `ID` | `void` |
| `DISTRIB_ID` | `VoidLinux` (in `/etc/lsb-release`) |

#### Package Manager

| Manager | Detection | Notes |
|---------|-----------|-------|
| `xbps-install` | `exec.LookPath("xbps-install")` | Void's native package manager |
| `xbps-query` | `exec.LookPath("xbps-query")` | Query installed packages |

### 1.10 Gentoo

#### Detection

| Field | Value |
|-------|-------|
| `ID` | `gentoo` |
| File check | `/etc/gentoo-release` exists |

#### Package Manager

| Manager | Detection | Notes |
|---------|-----------|-------|
| `emerge` | `exec.LookPath("emerge")` | Portage package manager |
| `equery` | `exec.LookPath("equery")` | From `gentoolkit`; query installed packages |

### 1.11 NixOS

#### Detection

| Field | Value |
|-------|-------|
| `ID` | `nixos` |
| File check | `/etc/NIXOS` exists |
| File check | `/nix/store` directory exists |

NixOS vs Nix-on-other-distro: on NixOS, `ID=nixos` in `/etc/os-release`. On other distros with Nix installed, `ID` will be the host distro, but `/nix/store` exists and `nix` is in PATH.

#### Package Management Patterns

| Pattern | Command | Notes |
|---------|---------|-------|
| `nix profile` | `nix profile install nixpkgs#<pkg>` | Modern, flake-compatible, imperative |
| `nix-env` | `nix-env -iA nixpkgs.<pkg>` | Legacy imperative |
| NixOS config | `environment.systemPackages` in `configuration.nix` | Declarative, NixOS only |
| home-manager | `home.packages` in `home.nix` | Declarative, any Nix system |
| Flake devShell | `nix develop` | Per-project, reproducible |
| `nix run` | `nix run nixpkgs#<pkg>` | Ephemeral, no install |

For gdev on NixOS: prefer `nix profile install` for user-level tools, or document the declarative approach and let the user choose. Never use `nix-env` on NixOS (it conflicts with declarative config).

### 1.12 Container Detection

Detect if running inside a container (affects package installation strategy):

```go
func isContainer() bool {
    // Method 1: /.dockerenv exists
    if _, err := os.Stat("/.dockerenv"); err == nil {
        return true
    }
    // Method 2: /run/.containerenv exists (Podman)
    if _, err := os.Stat("/run/.containerenv"); err == nil {
        return true
    }
    // Method 3: cgroup contains "docker" or "containerd"
    data, err := os.ReadFile("/proc/1/cgroup")
    if err == nil {
        s := string(data)
        if strings.Contains(s, "docker") || strings.Contains(s, "containerd") {
            return true
        }
    }
    // Method 4: check for container env vars
    if os.Getenv("container") != "" { // systemd-nspawn, podman
        return true
    }
    return false
}
```

### 1.13 Shell Detection

#### Current Shell

```go
func detectShell() string {
    // Method 1: $SHELL environment variable (login shell, not necessarily current)
    shell := os.Getenv("SHELL")

    // Method 2: parent process name (actual current shell)
    // On Unix: read /proc/$PPID/comm or use ps
    ppid := os.Getppid()
    comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", ppid))
    if err == nil {
        return strings.TrimSpace(string(comm)) // "bash", "zsh", "fish", etc.
    }

    // Method 3: On macOS (no /proc), use ps
    out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=").Output()
    if err == nil {
        return strings.TrimSpace(string(out))
    }

    // Method 4: On Windows
    if runtime.GOOS == "windows" {
        // Check $PSModulePath for PowerShell
        if os.Getenv("PSModulePath") != "" {
            return "powershell"
        }
        // Check $MSYSTEM for Git Bash/MSYS2
        if os.Getenv("MSYSTEM") != "" {
            return "bash" // Git Bash
        }
        return "cmd"
    }

    // Fallback: parse $SHELL
    if shell != "" {
        return filepath.Base(shell)
    }
    return "unknown"
}
```

#### Shell-Specific Version Variables

| Shell | Version Variable | Example |
|-------|-----------------|---------|
| bash | `$BASH_VERSION` | `"5.2.37(1)-release"` |
| zsh | `$ZSH_VERSION` | `"5.9"` |
| fish | `$FISH_VERSION` | `"3.7.1"` |
| PowerShell | `$PSVersionTable.PSVersion` | `"7.4.6"` |
| nushell | `$nu.version` | `"0.104.0"` |

#### RC File Locations

| Shell | Primary RC File | Login RC File |
|-------|----------------|---------------|
| bash | `~/.bashrc` | `~/.bash_profile` or `~/.profile` |
| zsh | `~/.zshrc` | `~/.zprofile` |
| fish | `~/.config/fish/config.fish` | (same) |
| PowerShell Core | `$PROFILE` (`~/.config/powershell/Microsoft.PowerShell_profile.ps1` on Linux/macOS, `~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1` on Windows) | (same) |
| Windows PowerShell | `$PROFILE` (`~/Documents/WindowsPowerShell/Microsoft.PowerShell_profile.ps1`) | (same) |
| nushell | `~/.config/nushell/config.nu` | `~/.config/nushell/env.nu` |

On macOS, note that the default shell is zsh (since Catalina/10.15). On most Linux distros, bash is default. On Windows, cmd.exe is the legacy default; PowerShell is the modern default.

---

## 2. Tool Prerequisite Mapping

### 2.1 git

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS** | `xcode-select --install` | Installs Apple's git with Xcode CLT; or `brew install git` for latest |
| **Debian/Ubuntu** | `sudo apt-get install -y git` | |
| **Fedora/RHEL** | `sudo dnf install -y git` | |
| **CentOS 7** | `sudo yum install -y git` | |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm git` | |
| **openSUSE** | `sudo zypper install -y git` | |
| **Alpine** | `sudo apk add git` | |
| **Void** | `sudo xbps-install -Sy git` | |
| **Gentoo** | `sudo emerge --ask dev-vcs/git` | |
| **NixOS** | `nix profile install nixpkgs#git` | Or add to `environment.systemPackages` |
| **Windows (winget)** | `winget install --id Git.Git -e --source winget` | |
| **Windows (choco)** | `choco install git` | |
| **Windows (scoop)** | `scoop install git` | |
| **Detection** | `exec.LookPath("git")` | |

### 2.2 Go

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install go` | |
| **macOS (official)** | Download `.pkg` from `go.dev/dl/` | |
| **Debian/Ubuntu** | `sudo apt-get install -y golang` | Often outdated; prefer official tarball |
| **Fedora/RHEL** | `sudo dnf install -y golang` | |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm go` | |
| **openSUSE** | `sudo zypper install -y golang` | Older name: `go` |
| **Alpine** | `sudo apk add go` | |
| **Void** | `sudo xbps-install -Sy go` | |
| **Gentoo** | `sudo emerge --ask dev-lang/go` | |
| **NixOS** | `nix profile install nixpkgs#go` | Or `environment.systemPackages = [ pkgs.go ];` |
| **Windows (winget)** | `winget install GoLang.Go` | |
| **Windows (choco)** | `choco install golang` | |
| **Windows (scoop)** | `scoop install go` | |
| **Official tarball** | `curl -fsSL https://go.dev/dl/go1.24.3.linux-amd64.tar.gz \| sudo tar -C /usr/local -xzf -` | Universal Linux; requires PATH setup |
| **mise** | `mise install go@latest && mise use go@latest` | Cross-platform version manager |
| **Detection** | `exec.LookPath("go")` | Check `go version` output for version |

**Recommendation for gdev:** Since gdev itself is written in Go, Go may already be present. For ensuring a specific version, prefer `mise` or the official tarball over distro packages (which lag behind).

### 2.3 Node.js / npm

Required for Claude Code (`npm install -g @anthropic-ai/claude-code`).

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install node` | |
| **Debian/Ubuntu (NodeSource)** | `curl -fsSL https://deb.nodesource.com/setup_22.x \| sudo -E bash - && sudo apt-get install -y nodejs` | Recommended over distro package |
| **Debian/Ubuntu (apt)** | `sudo apt-get install -y nodejs npm` | Often very outdated |
| **Fedora/RHEL (NodeSource)** | `curl -fsSL https://rpm.nodesource.com/setup_22.x \| sudo bash - && sudo dnf install -y nodejs` | |
| **Fedora/RHEL** | `sudo dnf install -y nodejs npm` | May be outdated |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm nodejs npm` | Usually current |
| **openSUSE** | `sudo zypper install -y nodejs npm` | |
| **Alpine** | `sudo apk add nodejs npm` | |
| **Void** | `sudo xbps-install -Sy nodejs` | |
| **Gentoo** | `sudo emerge --ask net-libs/nodejs` | |
| **NixOS** | `nix profile install nixpkgs#nodejs` | Includes npm |
| **Windows (winget)** | `winget install OpenJS.NodeJS.LTS` | |
| **Windows (choco)** | `choco install nodejs-lts` | |
| **Windows (scoop)** | `scoop install nodejs-lts` | |
| **fnm (cross-platform)** | `fnm install --lts && fnm use lts-latest` | Fast Node Manager (Rust) |
| **volta (cross-platform)** | `volta install node` | Automatic per-project switching |
| **nvm (Linux/macOS)** | `nvm install --lts` | Widely used but slower than fnm |
| **nvm-windows** | `nvm install lts && nvm use lts` | Windows-specific fork |
| **Detection** | `exec.LookPath("node")` and `exec.LookPath("npm")` | Check `node --version` |

**Recommendation for gdev:** Prefer version managers (fnm or volta) for developer machines; use NodeSource for CI/server environments.

### 2.4 Nix

Required for devenv. NOT available on native Windows.

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **Linux (Determinate)** | `curl -fsSL https://install.determinate.systems/nix \| sh -s -- install` | Recommended; enables flakes by default |
| **macOS (Determinate)** | `curl -fsSL https://install.determinate.systems/nix \| sh -s -- install` | Same command; handles macOS specifics |
| **WSL2 (Determinate)** | `curl -fsSL https://install.determinate.systems/nix \| sh -s -- install` | Add `--init none` if no systemd |
| **Linux (Official)** | `sh <(curl -L https://nixos.org/nix/install) --daemon` | Multi-user recommended |
| **macOS (Official)** | `curl -sSfL https://artifacts.nixos.org/nix-installer \| sh -s -- install` | NixOS fork of Determinate installer |
| **WSL2 (Official)** | `sh <(curl -L https://nixos.org/nix/install) --no-daemon` | Single-user for WSL without systemd |
| **NixOS** | Pre-installed | `nix` is always available |
| **Windows (native)** | **NOT SUPPORTED** | Must use WSL2 |
| **Detection** | `exec.LookPath("nix")` | Check `nix --version` |

**Recommendation for gdev:** Use the Determinate Systems installer. It enables flakes by default, has better error recovery, and supports SELinux. For WSL2 without systemd, pass `--init none`.

### 2.5 devenv

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **Any (nix-env)** | `nix-env --install --attr devenv -f https://github.com/NixOS/nixpkgs/tarball/nixpkgs-unstable` | Works without experimental features |
| **Any (nix profile)** | `nix profile install nixpkgs#devenv` | Requires `nix-command` experimental feature |
| **NixOS config** | `environment.systemPackages = [ pkgs.devenv ];` | Declarative |
| **home-manager** | `home.packages = [ pkgs.devenv ];` | Declarative, user-level |
| **Windows (native)** | **NOT SUPPORTED** | Requires Nix (WSL2 on Windows) |
| **Detection** | `exec.LookPath("devenv")` | Check `devenv version` |

**Prerequisites:** Nix must be installed first. On macOS, upgrade bash: `nix profile install nixpkgs#bashInteractive` (macOS ships bash 3.2 due to GPL licensing).

### 2.6 direnv

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install direnv` | |
| **Debian/Ubuntu** | `sudo apt-get install -y direnv` | |
| **Fedora/RHEL** | `sudo dnf install -y direnv` | |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm direnv` | |
| **openSUSE** | `sudo zypper install -y direnv` | |
| **Gentoo** | `sudo emerge --ask dev-util/direnv` | |
| **NixOS** | `nix profile install nixpkgs#direnv` | Or `programs.direnv.enable = true;` in NixOS config |
| **Alpine** | `sudo apk add direnv` | |
| **Void** | `sudo xbps-install -Sy direnv` | |
| **Windows (winget)** | `winget install direnv` | |
| **Windows (scoop)** | `scoop install direnv` | |
| **Universal (curl)** | `curl -sfL https://direnv.net/install.sh \| bash` | |
| **Any (go install)** | `go install github.com/direnv/direnv@latest` | Requires Go |
| **Detection** | `exec.LookPath("direnv")` | |

**Shell hook setup required** — see Section 3.

### 2.7 Claude Code

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **Any (npm)** | `npm install -g @anthropic-ai/claude-code` | Requires Node.js 18+ |
| **Detection** | `exec.LookPath("claude")` | Check `claude --version` |

**Prerequisites:** Node.js and npm must be installed. Works on all platforms where Node.js runs (Linux, macOS, Windows native, WSL2).

### 2.8 pre-commit / prek

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **Any (pip)** | `pip install pre-commit` | May need `--user` flag |
| **Any (pipx)** | `pipx install pre-commit` | Recommended for global install |
| **macOS (Homebrew)** | `brew install pre-commit` | |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm pre-commit` | |
| **NixOS/Nix** | `nix profile install nixpkgs#pre-commit` | |
| **Any (conda)** | `conda install -c conda-forge pre-commit` | |
| **devenv** | Built-in via `pre-commit.hooks` in `devenv.nix` | prek replaces pre-commit in devenv 1.11+ |
| **Detection** | `exec.LookPath("pre-commit")` or `exec.LookPath("prek")` | |

**Note:** devenv 1.11+ ships `prek` as a drop-in replacement for `pre-commit`. Same config format, different binary.

### 2.9 shellcheck

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install shellcheck` | |
| **Debian/Ubuntu** | `sudo apt-get install -y shellcheck` | |
| **Fedora/RHEL** | `sudo dnf install -y ShellCheck` | Note: capital S and C |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm shellcheck` | |
| **openSUSE** | `sudo zypper install -y ShellCheck` | |
| **Alpine** | `sudo apk add shellcheck` | |
| **Gentoo** | `sudo emerge --ask dev-util/shellcheck` | |
| **NixOS/Nix** | `nix profile install nixpkgs#shellcheck` | |
| **Void** | `sudo xbps-install -Sy shellcheck` | |
| **Windows (scoop)** | `scoop install shellcheck` | |
| **Windows (choco)** | `choco install shellcheck` | |
| **Snap** | `sudo snap install --channel=edge shellcheck` | |
| **Detection** | `exec.LookPath("shellcheck")` | |

### 2.10 shfmt

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install shfmt` | |
| **Debian/Ubuntu** | `sudo snap install shfmt` | Not in default apt repos |
| **Fedora/RHEL** | `sudo dnf install -y shfmt` | |
| **Arch/Manjaro** | `sudo pacman -S --noconfirm shfmt` | |
| **openSUSE** | `sudo zypper install -y shfmt` | |
| **Alpine** | `sudo apk add shfmt` | |
| **NixOS/Nix** | `nix profile install nixpkgs#shfmt` | |
| **Void** | `sudo xbps-install -Sy shfmt` | |
| **Windows (scoop)** | `scoop install shfmt` | |
| **Snap** | `sudo snap install shfmt` | |
| **Any (Go)** | `go install mvdan.cc/sh/v3/cmd/shfmt@latest` | |
| **Any (webi)** | `curl -sS https://webi.sh/shfmt \| sh` | |
| **Detection** | `exec.LookPath("shfmt")` | |

### 2.11 hadolint

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew)** | `brew install hadolint` | |
| **Windows (scoop)** | `scoop install hadolint` | |
| **NixOS/Nix** | `nix profile install nixpkgs#hadolint` | |
| **Docker** | `docker pull hadolint/hadolint` | Universal fallback |
| **Any (binary)** | Download from `github.com/hadolint/hadolint/releases/latest` | Linux, macOS, Windows binaries |
| **Detection** | `exec.LookPath("hadolint")` | |

**Note:** hadolint has limited package manager availability. For distros without a package (Debian, Fedora, Arch), use the binary download or Docker image.

### 2.12 goreleaser

| Platform | Install Command | Notes |
|----------|----------------|-------|
| **macOS (Homebrew tap)** | `brew install --cask goreleaser/tap/goreleaser` | Official tap, latest |
| **macOS (Homebrew)** | `brew install goreleaser` | Community-maintained |
| **Debian/Ubuntu (apt)** | `echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' \| sudo tee /etc/apt/sources.list.d/goreleaser.list && sudo apt update && sudo apt install goreleaser` | |
| **Fedora/RHEL (yum)** | Add goreleaser repo, then `sudo yum install goreleaser` | |
| **Arch/Manjaro (AUR)** | `yay -S goreleaser-bin` | |
| **NixOS/Nix** | `nix-shell -p goreleaser` or `nix profile install nixpkgs#goreleaser` | |
| **Windows (winget)** | `winget install goreleaser` | |
| **Windows (scoop)** | `scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git && scoop install goreleaser` | |
| **Windows (choco)** | `choco install goreleaser` | |
| **Snap** | `sudo snap install --classic goreleaser` | |
| **Any (Go)** | `go install github.com/goreleaser/goreleaser/v2@latest` | Requires Go 1.24+ |
| **Any (npm)** | `npm i -g @goreleaser/goreleaser` | |
| **Detection** | `exec.LookPath("goreleaser")` | |

### 2.13 curl, wget, jq

| Tool | Platform | Install Command |
|------|----------|----------------|
| **curl** | macOS | Pre-installed |
| | Debian/Ubuntu | `sudo apt-get install -y curl` |
| | Fedora/RHEL | `sudo dnf install -y curl` |
| | Arch | `sudo pacman -S --noconfirm curl` |
| | Alpine | `sudo apk add curl` |
| | openSUSE | `sudo zypper install -y curl` |
| | Void | `sudo xbps-install -Sy curl` |
| | Gentoo | `sudo emerge --ask net-misc/curl` |
| | Windows | Pre-installed on Windows 10+; or `winget install curl` |
| **wget** | macOS | `brew install wget` |
| | Debian/Ubuntu | `sudo apt-get install -y wget` |
| | Fedora/RHEL | `sudo dnf install -y wget` |
| | Arch | `sudo pacman -S --noconfirm wget` |
| | Alpine | `sudo apk add wget` |
| | Windows | `winget install GnuWin32.Wget` or `choco install wget` |
| **jq** | macOS | `brew install jq` |
| | Debian/Ubuntu | `sudo apt-get install -y jq` |
| | Fedora/RHEL | `sudo dnf install -y jq` |
| | Arch | `sudo pacman -S --noconfirm jq` |
| | Alpine | `sudo apk add jq` |
| | openSUSE | `sudo zypper install -y jq` |
| | Void | `sudo xbps-install -Sy jq` |
| | Gentoo | `sudo emerge --ask app-misc/jq` |
| | NixOS | `nix profile install nixpkgs#jq` |
| | Windows (winget) | `winget install jqlang.jq` |
| | Windows (choco) | `choco install jq` |
| | Windows (scoop) | `scoop install jq` |
| **Detection** | All | `exec.LookPath("<tool>")` |

---

## 3. Shell Integration

### 3.1 Adding to PATH

#### bash

```bash
# ~/.bashrc or ~/.bash_profile
export PATH="$HOME/.local/bin:$PATH"
```

#### zsh

```zsh
# ~/.zshrc or ~/.zprofile
export PATH="$HOME/.local/bin:$PATH"
```

#### fish

```fish
# ~/.config/fish/config.fish
fish_add_path ~/.local/bin
# Or: set -gx PATH $HOME/.local/bin $PATH
```

#### PowerShell (Core and Windows)

```powershell
# $PROFILE
$env:PATH = "$HOME\.local\bin;$env:PATH"
# Persistent:
[Environment]::SetEnvironmentVariable("PATH", "$HOME\.local\bin;$env:PATH", "User")
```

#### nushell

```nu
# ~/.config/nushell/env.nu
$env.PATH = ($env.PATH | prepend ($env.HOME | path join ".local" "bin"))
```

### 3.2 direnv Hook Setup

| Shell | RC File | Hook Line |
|-------|---------|-----------|
| **bash** | `~/.bashrc` | `eval "$(direnv hook bash)"` |
| **zsh** | `~/.zshrc` | `eval "$(direnv hook zsh)"` |
| **fish** | `~/.config/fish/config.fish` | `direnv hook fish \| source` |
| **PowerShell** | `$PROFILE` | `Invoke-Expression "$(direnv hook pwsh)"` |
| **nushell** | `~/.config/nushell/config.nu` | See below |
| **tcsh** | `~/.cshrc` | `` eval `direnv hook tcsh` `` |
| **elvish** | `~/.config/elvish/rc.elv` | `use direnv` (after `direnv hook elvish > ~/.config/elvish/lib/direnv.elv`) |
| **murex** | `~/.murex_profile` | `direnv hook murex -> source` |

**Nushell direnv hook** (requires nushell 0.104+):

```nu
# In config.nu
$env.config.hooks.env_change.PWD = (
    $env.config.hooks.env_change.PWD | default [] | append {||
        if (which direnv | is-not-empty) {
            direnv export json | from json | default {} | load-env
        }
    }
)
```

**Important notes:**
- For bash, the direnv hook must appear AFTER rvm, git-prompt, and other prompt-manipulating extensions
- For zsh with Oh My Zsh, can add `direnv` to the `plugins` array instead
- Nushell has a known issue where direnv exports PATH as a string but nushell expects a list; requires `env-conversions` from the standard library
- devenv 2.1+ offers native nushell support as an alternative to direnv

### 3.3 Shell Completions for gdev

gdev uses Cobra, which has built-in completion generation for bash, zsh, fish, and PowerShell.

#### Completion Generation Commands

```bash
# bash (system-wide)
gdev completion bash | sudo tee /etc/bash_completion.d/gdev > /dev/null

# bash (user)
gdev completion bash > ~/.local/share/bash-completion/completions/gdev

# zsh (system-wide)
gdev completion zsh > /usr/local/share/zsh/site-functions/_gdev

# zsh (user — add to fpath in .zshrc first)
gdev completion zsh > ~/.zsh/completions/_gdev

# fish
gdev completion fish > ~/.config/fish/completions/gdev.fish

# PowerShell
gdev completion powershell >> $PROFILE
```

#### Per-Shell Setup

| Shell | Completion Directory | Activation |
|-------|---------------------|------------|
| bash | `/etc/bash_completion.d/` or `~/.local/share/bash-completion/completions/` | Auto-loaded by bash-completion package |
| zsh | `/usr/local/share/zsh/site-functions/` or `~/.zsh/completions/` | Must be in `$fpath`; run `compinit` |
| fish | `~/.config/fish/completions/` | Auto-loaded by fish |
| PowerShell | Appended to `$PROFILE` | Auto-loaded on shell start |
| nushell | Not natively supported by Cobra | Would need custom implementation or `carapace` bridge |

**Recommendation:** Generate completions during `gdev init` and install them to the user-level directory for the detected shell. Avoid system-wide installation (requires root).

### 3.4 Sourcing Environment Files

#### bash / zsh

```bash
# Source a file
source ~/.gdev/env.sh
# Or
. ~/.gdev/env.sh
```

#### fish

```fish
# Source a fish-compatible file
source ~/.gdev/env.fish
# Or for POSIX files, use bass plugin or fenv
```

#### PowerShell

```powershell
# Dot-source a script
. $HOME\.gdev\env.ps1
```

#### nushell

```nu
# Source a nu file
source ~/.gdev/env.nu
# Or load environment from a file
open ~/.gdev/env.json | load-env
```

### 3.5 Detecting Current Shell and RC File

```go
type ShellInfo struct {
    Name    string // "bash", "zsh", "fish", "pwsh", "powershell", "nu", "cmd"
    Version string // e.g., "5.2.37"
    RCFile  string // absolute path to the shell's rc file
    Path    string // absolute path to the shell binary
}

func detectShellInfo() ShellInfo {
    name := detectShell() // from Section 1.13
    info := ShellInfo{Name: name}

    home, _ := os.UserHomeDir()

    switch name {
    case "bash":
        info.RCFile = filepath.Join(home, ".bashrc")
        // On macOS, .bash_profile is often used instead
        if runtime.GOOS == "darwin" {
            if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
                info.RCFile = filepath.Join(home, ".bash_profile")
            }
        }
    case "zsh":
        info.RCFile = filepath.Join(home, ".zshrc")
    case "fish":
        info.RCFile = filepath.Join(home, ".config", "fish", "config.fish")
    case "pwsh", "powershell":
        // PowerShell Core on Linux/macOS
        if runtime.GOOS != "windows" {
            info.RCFile = filepath.Join(home, ".config", "powershell",
                "Microsoft.PowerShell_profile.ps1")
        } else {
            // Windows PowerShell or PowerShell Core on Windows
            info.RCFile = filepath.Join(home, "Documents", "PowerShell",
                "Microsoft.PowerShell_profile.ps1")
        }
    case "nu":
        info.RCFile = filepath.Join(home, ".config", "nushell", "config.nu")
    case "cmd":
        info.RCFile = "" // cmd.exe has no rc file; use registry or AutoRun
    }

    return info
}
```

---

## 4. Windows-Specific Considerations

### 4.1 WSL2 vs Native Windows Development

| Aspect | Native Windows | WSL2 |
|--------|---------------|------|
| **Nix** | Not supported | Fully supported |
| **devenv** | Not supported | Fully supported |
| **direnv** | Partial (winget install, limited shell integration) | Fully supported |
| **Git** | Full (Git for Windows) | Full (Linux git) |
| **Go** | Full | Full |
| **Node.js/npm** | Full | Full |
| **Claude Code** | Full (npm global) | Full |
| **Docker** | Docker Desktop with WSL2 backend | Native Docker in WSL2 |
| **shellcheck** | Via scoop/choco | Native apt/dnf |
| **hadolint** | Via scoop or binary download | Full package manager support |
| **pre-commit** | Via pip/pipx | Full support |
| **File system perf** | Native NTFS — fast | Linux fs fast; `/mnt/c/` via 9P — 3x slower |
| **PATH interop** | N/A | WSL can access Windows PATH (default on) |

### 4.2 Tools That Require WSL2 on Windows

The following tools have NO native Windows support:
- **Nix** — requires Linux kernel; WSL2 is the only Windows option
- **devenv** — requires Nix
- **nix-direnv** — requires Nix (plain direnv works natively)

### 4.3 gdev Strategy for Windows

```
Is user on Windows? (runtime.GOOS == "windows")
├── YES
│   ├── Check for WSL2: wsl --list --verbose
│   │   ├── WSL2 available with a distro installed
│   │   │   └── Offer two paths:
│   │   │       1. Install tools IN WSL2 (full Nix/devenv support)
│   │   │       2. Install native-only tools (no Nix/devenv)
│   │   └── No WSL2
│   │       └── Offer to help install WSL2:
│   │           wsl --install
│   │           (requires admin, triggers reboot)
│   └── For native-only path:
│       Install: git, Go, Node.js, Claude Code, pre-commit, shellcheck, shfmt
│       Skip: Nix, devenv, direnv (or install direnv without Nix integration)
└── NO (Linux or macOS)
    └── Full tool installation
```

### 4.4 WSL2 Detection and Setup

**Detect WSL2 availability from native Windows:**

```go
// Check if WSL is installed and has distributions
func checkWSL() (bool, []string, error) {
    out, err := exec.Command("wsl", "--list", "--verbose").Output()
    if err != nil {
        return false, nil, err // WSL not installed
    }
    // Parse output for running distros with VERSION 2
    // Output is UTF-16LE on Windows — must decode
    // Lines look like: "* Ubuntu    Running    2"
    distros := parseWSLOutput(out)
    return len(distros) > 0, distros, nil
}

// Install WSL2 (requires admin)
func installWSL() error {
    return exec.Command("wsl", "--install").Run()
    // This installs WSL2 + Ubuntu by default
    // Requires reboot
}
```

**Run a command inside WSL from native Windows:**

```go
func runInWSL(distro string, command string) ([]byte, error) {
    return exec.Command("wsl", "-d", distro, "--", "bash", "-c", command).Output()
}
```

### 4.5 Path Translation Between Windows and WSL2

```go
// Windows path to WSL path
// C:\Users\colin\project → /mnt/c/Users/colin/project
func windowsToWSLPath(winPath string) string {
    out, err := exec.Command("wsl", "wslpath", "-u", winPath).Output()
    if err != nil {
        // Manual fallback
        p := strings.ReplaceAll(winPath, "\\", "/")
        if len(p) >= 2 && p[1] == ':' {
            return "/mnt/" + strings.ToLower(string(p[0])) + p[2:]
        }
        return p
    }
    return strings.TrimSpace(string(out))
}

// WSL path to Windows path
// /home/colin/project → \\wsl$\Ubuntu\home\colin\project
func wslToWindowsPath(wslPath string) string {
    out, err := exec.Command("wsl", "wslpath", "-w", wslPath).Output()
    if err != nil {
        return wslPath
    }
    return strings.TrimSpace(string(out))
}
```

**Performance warning:** Files on `/mnt/c/` (Windows filesystem accessed from WSL2) are ~3x slower due to 9P protocol translation. gdev should warn users to keep projects in the Linux filesystem (`/home/user/`) when using WSL2.

### 4.6 Visual Studio Build Tools / MSVC

Required for C/C++ compilation on Windows, and occasionally needed by npm packages with native addons.

**Detection:**

```go
func detectMSVC() bool {
    // Check common paths
    paths := []string{
        `C:\Program Files\Microsoft Visual Studio\2022\BuildTools`,
        `C:\Program Files\Microsoft Visual Studio\2022\Community`,
        `C:\Program Files\Microsoft Visual Studio\2022\Professional`,
        `C:\Program Files\Microsoft Visual Studio\2022\Enterprise`,
        `C:\Program Files (x86)\Microsoft Visual Studio\2019\BuildTools`,
    }
    for _, p := range paths {
        if _, err := os.Stat(p); err == nil {
            return true
        }
    }
    // Also check: vswhere.exe
    vswhere := `C:\Program Files (x86)\Microsoft Visual Studio\Installer\vswhere.exe`
    if _, err := os.Stat(vswhere); err == nil {
        out, err := exec.Command(vswhere, "-latest", "-property", "installationPath").Output()
        if err == nil && len(out) > 0 {
            return true
        }
    }
    return false
}
```

**Installation:**

```powershell
# Via winget (requires --wait to block until complete)
winget install Microsoft.VisualStudio.2022.BuildTools --override "--quiet --wait --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"

# Via choco
choco install visualstudio2022buildtools --package-parameters "--add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"
```

### 4.7 Windows Terminal Detection

```go
func isWindowsTerminal() bool {
    return os.Getenv("WT_SESSION") != ""
}

func isConhost() bool {
    // If WT_SESSION is empty and we're on Windows, likely conhost
    return runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == ""
}
```

Windows 11 defaults to Windows Terminal; Windows 10 defaults to conhost. Windows Terminal supports ANSI colors, emoji, and modern terminal features. gdev should degrade gracefully in conhost (no emoji, simpler progress indicators).

---

## 5. Privilege Escalation Patterns

### 5.1 Detecting Current Privileges

#### Linux / macOS

```go
func isRoot() bool {
    return os.Getuid() == 0
}

func isEffectiveRoot() bool {
    return os.Geteuid() == 0
}

func hasSudo() bool {
    _, err := exec.LookPath("sudo")
    return err == nil
}

// Check if user can sudo without a password (cached credentials)
func canSudoNoPassword() bool {
    err := exec.Command("sudo", "-n", "true").Run()
    return err == nil
}
```

#### Windows

```go
// See Section 1.3 for full isAdmin() implementation using
// windows.AllocateAndInitializeSid with DOMAIN_ALIAS_RID_ADMINS

func hasGsudo() bool {
    _, err := exec.LookPath("gsudo")
    return err == nil
}

func hasNativeSudo() bool {
    // Windows 11 24H2+
    _, err := exec.LookPath("sudo")
    return err == nil && runtime.GOOS == "windows"
}
```

### 5.2 Elevation Methods

#### Linux

| Method | Command | When to Use |
|--------|---------|-------------|
| `sudo` | `sudo apt-get install -y git` | Standard elevation; available on most systems |
| `pkexec` | `pkexec apt-get install -y git` | PolicyKit; for GUI environments, prompts graphically |
| `su -c` | `su -c "apt-get install -y git"` | When sudo is not configured |
| `doas` | `doas apt-get install -y git` | OpenBSD-origin alternative to sudo; used on Alpine, Void |

Detection priority: `sudo` > `doas` > `pkexec` > `su`

```go
func findElevationCommand() string {
    for _, cmd := range []string{"sudo", "doas", "pkexec"} {
        if _, err := exec.LookPath(cmd); err == nil {
            return cmd
        }
    }
    return "" // No elevation command available
}
```

#### macOS

Same as Linux: `sudo` is always available and is the standard method. macOS also supports `osascript -e 'do shell script "..." with administrator privileges'` for GUI-based elevation, but `sudo` is preferred for CLI tools.

#### Windows

| Method | Command | When to Use |
|--------|---------|-------------|
| Native sudo | `sudo netstat -ab` | Windows 11 24H2+; built-in |
| gsudo | `gsudo choco install git` | Third-party; credential caching, broad shell support |
| runas | `runas /user:Administrator cmd` | Built-in; prompts for password (no UAC), opens new window |
| PowerShell | `Start-Process -Verb RunAs` | PowerShell-native; spawns elevated process |

**gsudo Installation:**

```
winget install gerardog.gsudo
scoop install gsudo
choco install gsudo
```

**gsudo Credential Caching Modes:**

| Mode | Behavior |
|------|----------|
| `Explicit` (default) | Manual cache control via `gsudo cache {on\|off}` |
| `Auto` | First elevation starts cache automatically (Unix sudo-like) |
| `Disabled` | Every elevation requires UAC popup |

Cache expires after 5 minutes of inactivity (configurable).

### 5.3 Minimizing Elevation Scope

**Principle:** Never run the entire gdev process as root/admin. Elevate only specific commands that need it.

```go
// Pattern: run a single command with elevation
func runElevated(command string, args ...string) error {
    switch runtime.GOOS {
    case "linux", "darwin":
        elevator := findElevationCommand()
        if elevator == "" {
            return fmt.Errorf("no elevation command available (sudo, doas, pkexec)")
        }
        fullArgs := append([]string{command}, args...)
        cmd := exec.Command(elevator, fullArgs...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        return cmd.Run()

    case "windows":
        // Prefer gsudo > native sudo > Start-Process
        if _, err := exec.LookPath("gsudo"); err == nil {
            fullArgs := append([]string{command}, args...)
            cmd := exec.Command("gsudo", fullArgs...)
            cmd.Stdin = os.Stdin
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr
            return cmd.Run()
        }
        if _, err := exec.LookPath("sudo"); err == nil {
            fullArgs := append([]string{command}, args...)
            cmd := exec.Command("sudo", fullArgs...)
            cmd.Stdin = os.Stdin
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr
            return cmd.Run()
        }
        // Fallback: PowerShell Start-Process
        psCmd := fmt.Sprintf("Start-Process -Verb RunAs -Wait -FilePath '%s' -ArgumentList '%s'",
            command, strings.Join(args, "','"))
        return exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run()
    }
    return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}
```

**Operations that require elevation:**

| Operation | Platform | Why |
|-----------|----------|-----|
| `apt-get install` | Debian/Ubuntu | System package installation |
| `dnf install` | Fedora/RHEL | System package installation |
| `pacman -S` | Arch | System package installation |
| `zypper install` | openSUSE | System package installation |
| `apk add` | Alpine | System package installation |
| `xbps-install` | Void | System package installation |
| `emerge` | Gentoo | System package installation |
| `wsl --install` | Windows | WSL2 setup |
| VS Build Tools install | Windows | System-level installation |
| Nix multi-user install | Linux/macOS | Creates `/nix` store and daemon |

**Operations that do NOT require elevation:**

| Operation | Platform | Why |
|-----------|----------|-----|
| `brew install` | macOS | User-space package manager |
| `scoop install` | Windows | User-space package manager |
| `nix profile install` | Any | User-level profile |
| `npm install -g` | Any | User-level if using nvm/fnm/volta |
| `pipx install` | Any | User-level |
| `go install` | Any | Goes to `$GOPATH/bin` |
| Shell rc file edits | Any | User files |
| `winget install` | Windows | UAC prompt is handled by winget itself |
| `choco install` | Windows | Needs admin, but choco handles UAC |

### 5.4 Prompting for Elevation

```go
// Ask user before any elevated operation
func promptForElevation(description string) bool {
    fmt.Printf("\n%s requires administrator/root privileges.\n", description)
    fmt.Printf("gdev will run: %s\n", description)
    fmt.Print("Continue? [y/N] ")
    var response string
    fmt.Scanln(&response)
    return strings.ToLower(response) == "y"
}

// Batch elevated operations to minimize prompts
type ElevatedBatch struct {
    commands [][]string
}

func (b *ElevatedBatch) Add(command string, args ...string) {
    b.commands = append(b.commands, append([]string{command}, args...))
}

func (b *ElevatedBatch) Execute() error {
    // On Linux: combine into a single sudo bash -c "cmd1 && cmd2 && cmd3"
    // This prompts for password once instead of N times
    if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
        combined := make([]string, len(b.commands))
        for i, cmd := range b.commands {
            combined[i] = shellescape(cmd)
        }
        script := strings.Join(combined, " && ")
        return runElevated("bash", "-c", script)
    }
    // On Windows: use gsudo cache to batch elevation
    // ...
}
```

---

## 6. Go Implementation Patterns

### 6.1 Complete OS Detection Struct

```go
type OSInfo struct {
    // Core identification
    OS       string // "linux", "darwin", "windows"
    Arch     string // "amd64", "arm64"
    Family   string // "debian", "rhel", "arch", "suse", "alpine", "void", "gentoo", "nixos", "macos", "windows"

    // Detailed identification
    Distro       string // "ubuntu", "fedora", "arch", "nixos", etc.
    DistroLike   string // "debian", "fedora", etc. (from ID_LIKE)
    Version      string // "24.04", "41", "15.4.1", "10.0.22631"
    VersionCode  string // "noble", "bookworm" (Debian/Ubuntu only)
    PrettyName   string // "Ubuntu 24.04.2 LTS", "Fedora Linux 41", etc.
    KernelVersion string // from uname -r

    // Environment flags
    IsWSL        bool   // Running inside WSL
    IsWSL2       bool   // Specifically WSL2
    WSLDistro    string // WSL_DISTRO_NAME value
    IsContainer  bool   // Running inside Docker/Podman/LXC
    IsRosetta    bool   // Running under Rosetta 2 translation
    IsSELinux    bool   // SELinux is enforcing

    // Package manager
    PackageManager    string   // Primary: "apt", "dnf", "pacman", "brew", etc.
    AltPackageManagers []string // Secondary: "snap", "flatpak", "yay", etc.

    // Shell
    Shell     ShellInfo

    // Elevation
    HasSudo   bool
    HasDoas   bool
    IsRoot    bool
    IsAdmin   bool // Windows admin

    // Homebrew (macOS)
    HomebrewPrefix string // "/opt/homebrew" or "/usr/local"
    HasXcodeCLT    bool
}
```

### 6.2 Detection Function

```go
func DetectOS() (*OSInfo, error) {
    info := &OSInfo{
        OS:   runtime.GOOS,
        Arch: runtime.GOARCH,
    }

    switch runtime.GOOS {
    case "darwin":
        info.Family = "macos"
        info.Distro = "macos"
        detectMacOS(info)
    case "linux":
        detectLinux(info)
    case "windows":
        info.Family = "windows"
        info.Distro = "windows"
        detectWindows(info)
    }

    detectShellInfo(info)
    detectElevation(info)
    detectPackageManagers(info)

    return info, nil
}

func detectLinux(info *OSInfo) {
    // Parse /etc/os-release
    osRelease := parseOSRelease("/etc/os-release")
    info.Distro = osRelease["ID"]
    info.DistroLike = osRelease["ID_LIKE"]
    info.Version = strings.Trim(osRelease["VERSION_ID"], "\"")
    info.VersionCode = osRelease["VERSION_CODENAME"]
    info.PrettyName = strings.Trim(osRelease["PRETTY_NAME"], "\"")

    // Determine family from ID and ID_LIKE
    info.Family = determineFamily(info.Distro, info.DistroLike)

    // Check WSL
    info.IsWSL = isWSL()
    info.IsWSL2 = isWSL2()
    info.WSLDistro = os.Getenv("WSL_DISTRO_NAME")

    // Check container
    info.IsContainer = isContainer()

    // Check SELinux
    if out, err := exec.Command("getenforce").Output(); err == nil {
        info.IsSELinux = strings.TrimSpace(string(out)) == "Enforcing"
    }
}

func determineFamily(id, idLike string) string {
    // Direct ID match first
    switch id {
    case "nixos":
        return "nixos"
    case "alpine":
        return "alpine"
    case "void":
        return "void"
    case "gentoo":
        return "gentoo"
    case "arch", "manjaro", "endeavouros", "garuda":
        return "arch"
    }

    // Check ID_LIKE for family grouping
    idLikeLower := strings.ToLower(idLike)
    switch {
    case strings.Contains(idLikeLower, "debian") || strings.Contains(idLikeLower, "ubuntu"):
        return "debian"
    case id == "debian" || id == "ubuntu":
        return "debian"
    case strings.Contains(idLikeLower, "fedora") || strings.Contains(idLikeLower, "rhel"):
        return "rhel"
    case id == "fedora":
        return "rhel"
    case strings.Contains(idLikeLower, "suse"):
        return "suse"
    case strings.Contains(idLikeLower, "arch"):
        return "arch"
    }
    return "unknown"
}
```

### 6.3 Package Manager Resolution

```go
type PackageManagerInfo struct {
    Name           string   // "apt", "dnf", "pacman", etc.
    InstallCmd     string   // "apt-get install -y"
    UpdateCmd      string   // "apt-get update"
    SearchCmd      string   // "apt-cache search"
    NeedsSudo      bool     // true for system package managers
    Available      bool     // detected on system
}

func detectPackageManagers(info *OSInfo) {
    switch info.Family {
    case "debian":
        info.PackageManager = "apt"
        if _, err := exec.LookPath("snap"); err == nil {
            info.AltPackageManagers = append(info.AltPackageManagers, "snap")
        }
    case "rhel":
        if _, err := exec.LookPath("dnf"); err == nil {
            info.PackageManager = "dnf"
        } else {
            info.PackageManager = "yum"
        }
    case "arch":
        info.PackageManager = "pacman"
        for _, aur := range []string{"paru", "yay"} {
            if _, err := exec.LookPath(aur); err == nil {
                info.AltPackageManagers = append(info.AltPackageManagers, aur)
                break // prefer paru over yay
            }
        }
    case "suse":
        info.PackageManager = "zypper"
    case "alpine":
        info.PackageManager = "apk"
    case "void":
        info.PackageManager = "xbps"
    case "gentoo":
        info.PackageManager = "emerge"
    case "nixos":
        info.PackageManager = "nix"
    case "macos":
        if _, err := exec.LookPath("brew"); err == nil {
            info.PackageManager = "brew"
        }
        // Also check MacPorts
        if _, err := exec.LookPath("port"); err == nil {
            info.AltPackageManagers = append(info.AltPackageManagers, "macports")
        }
    case "windows":
        // Priority: winget > scoop > choco
        for _, pm := range []string{"winget", "scoop", "choco"} {
            if _, err := exec.LookPath(pm); err == nil {
                if info.PackageManager == "" {
                    info.PackageManager = pm
                } else {
                    info.AltPackageManagers = append(info.AltPackageManagers, pm)
                }
            }
        }
    }

    // Universal package managers (available on any platform with Nix)
    if info.Family != "nixos" {
        if _, err := exec.LookPath("nix"); err == nil {
            info.AltPackageManagers = append(info.AltPackageManagers, "nix")
        }
    }
}
```

### 6.4 /etc/os-release Parser

```go
func parseOSRelease(path string) map[string]string {
    result := make(map[string]string)
    data, err := os.ReadFile(path)
    if err != nil {
        return result
    }
    for _, line := range strings.Split(string(data), "\n") {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := parts[0]
        value := strings.Trim(parts[1], "\"")
        result[key] = value
    }
    return result
}
```

### 6.5 Tool Installation Dispatch

```go
type ToolInstaller struct {
    osInfo *OSInfo
}

// InstallPackage resolves the correct install command for the detected OS
func (t *ToolInstaller) InstallCommand(pkg string) (string, []string, bool) {
    // Returns: (command, args, needsElevation)
    // Maps generic package names to distro-specific names
    pkgName := t.resolvePackageName(pkg)

    switch t.osInfo.Family {
    case "debian":
        return "apt-get", []string{"install", "-y", pkgName}, true
    case "rhel":
        pm := "dnf"
        if t.osInfo.PackageManager == "yum" {
            pm = "yum"
        }
        return pm, []string{"install", "-y", pkgName}, true
    case "arch":
        return "pacman", []string{"-S", "--noconfirm", pkgName}, true
    case "suse":
        return "zypper", []string{"install", "-y", pkgName}, true
    case "alpine":
        return "apk", []string{"add", pkgName}, true
    case "void":
        return "xbps-install", []string{"-Sy", pkgName}, true
    case "gentoo":
        return "emerge", []string{"--ask", pkgName}, true
    case "nixos":
        return "nix", []string{"profile", "install", "nixpkgs#" + pkgName}, false
    case "macos":
        return "brew", []string{"install", pkgName}, false
    case "windows":
        switch t.osInfo.PackageManager {
        case "winget":
            return "winget", []string{"install", "--id", pkgName, "-e"}, false
        case "scoop":
            return "scoop", []string{"install", pkgName}, false
        case "choco":
            return "choco", []string{"install", pkgName, "-y"}, false // choco handles UAC
        }
    }
    return "", nil, false
}

// resolvePackageName maps a generic tool name to the distro-specific package name
func (t *ToolInstaller) resolvePackageName(generic string) string {
    // Package name mapping table
    nameMap := map[string]map[string]string{
        "git": {
            "winget": "Git.Git",
            // All others use "git"
        },
        "go": {
            "debian": "golang",
            "suse":   "golang",
            "gentoo": "dev-lang/go",
            "winget": "GoLang.Go",
            "choco":  "golang",
            // Others: "go"
        },
        "nodejs": {
            "gentoo": "net-libs/nodejs",
            "winget": "OpenJS.NodeJS.LTS",
            "choco":  "nodejs-lts",
            "scoop":  "nodejs-lts",
            // Others: "nodejs"
        },
        "shellcheck": {
            "rhel":   "ShellCheck", // Capital S and C on Fedora/RHEL
            "suse":   "ShellCheck",
            "gentoo": "dev-util/shellcheck",
            // Others: "shellcheck"
        },
        "jq": {
            "gentoo": "app-misc/jq",
            "winget": "jqlang.jq",
            // Others: "jq"
        },
        "direnv": {
            "gentoo": "dev-util/direnv",
            // Others: "direnv"
        },
        "curl": {
            "gentoo": "net-misc/curl",
            // Others: "curl"
        },
    }

    if mapping, ok := nameMap[generic]; ok {
        // Check distro-specific name first
        if name, ok := mapping[t.osInfo.Distro]; ok {
            return name
        }
        // Then family
        if name, ok := mapping[t.osInfo.Family]; ok {
            return name
        }
        // Then package manager
        if name, ok := mapping[t.osInfo.PackageManager]; ok {
            return name
        }
    }
    return generic // Default: use generic name
}
```

---

## Summary: Detection Priority and Fallback Chain

### OS Detection Order

1. `runtime.GOOS` / `runtime.GOARCH` — instant, always available
2. `/etc/os-release` — Linux distro identification (standard on all modern distros)
3. `sw_vers` — macOS version
4. `/proc/version` — WSL detection
5. `uname -a` — fallback for missing `/etc/os-release`
6. `/etc/<distro>-release` — legacy fallback (gentoo-release, redhat-release, etc.)

### Package Manager Detection Order

1. Match OS family to known default package manager
2. Verify with `exec.LookPath()` that the binary exists
3. Detect alternative/secondary package managers
4. For macOS: check Homebrew at known paths (`/opt/homebrew/bin/brew`, `/usr/local/bin/brew`)
5. For Windows: check winget > scoop > choco in priority order
6. For any system: check if `nix` is available as a universal fallback

### Shell Detection Order

1. `$SHELL` environment variable (login shell)
2. `/proc/$PPID/comm` (Linux — actual current shell)
3. `ps -p $PPID -o comm=` (macOS/BSD fallback)
4. `$BASH_VERSION`, `$ZSH_VERSION`, `$FISH_VERSION` (shell-specific env vars)
5. `$PSModulePath` (PowerShell indicator)
6. `$MSYSTEM` (Git Bash/MSYS2 indicator)

### Privilege Detection Order

1. `os.Getuid() == 0` (Linux/macOS — instant)
2. `windows.AllocateAndInitializeSid` + `IsMember` (Windows — requires x/sys/windows)
3. `exec.LookPath("sudo")` / `exec.LookPath("doas")` / `exec.LookPath("gsudo")` — elevation tools
4. `sudo -n true` — check if sudo is cached (no password needed)

### Key Recommendations for gdev Implementation

1. **Build `OSInfo` once at startup** and pass it through the entire tool chain. Detection is cheap but should not be repeated.
2. **Use `exec.LookPath` as the primary detection method** for tools and package managers. It is fast, cross-platform, and respects PATH.
3. **Never shell out when Go APIs exist.** Use `os.ReadFile("/etc/os-release")` instead of `exec.Command("cat", "/etc/os-release")`.
4. **Handle WSL2 as a first-class Linux environment,** not as Windows. When `runtime.GOOS == "linux"` and `/proc/version` contains "microsoft", treat it as Linux with the WSL2 flag set.
5. **Batch elevated operations** to minimize sudo/UAC prompts. Collect all packages that need system-level installation and run them in a single elevated command.
6. **Provide clear fallback paths.** If a preferred installer is missing (e.g., no Homebrew on macOS), offer to install it or suggest alternatives.
7. **Test on NixOS specially.** NixOS users will have Nix pre-installed but may resist `nix profile install` if they prefer declarative config. Provide instructions for both imperative and declarative approaches.
8. **Use Go build tags** (`//go:build linux`, `//go:build darwin`, `//go:build windows`) to keep platform-specific code clean and avoid importing `golang.org/x/sys/windows` on non-Windows builds.
