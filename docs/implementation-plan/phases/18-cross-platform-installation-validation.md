# Phase 18: Cross-Platform Installation & Bootstrap Validation

## Goal

Validate that gdev installs correctly and bootstraps on every supported platform: macOS (Intel + Apple Silicon), Windows (native + WSL2), and 12 Linux distro families. At the end of this phase, `gdev doctor` reports correct system info, `gdev setup --dry-run` identifies correct prerequisites, and install scripts work on all targets.

## Dependencies

Phase 17 complete (CI pipeline, test frameworks). Phase 9 complete (OS detection, package manager abstraction). Phase 10 complete (install scripts, GoReleaser, self-update).

## Phase Outputs

- Validated install scripts on macOS (Intel + ARM), Windows (PowerShell), Linux (bash) across 24 OS configurations
- Validated `gdev doctor` output for all 12 distro families + macOS + Windows
- Validated `gdev setup --dry-run` prerequisite detection on all targets
- Validated package manager installation (Homebrew tap, Scoop bucket, APT .deb, RPM)
- Validated self-update mechanism
- Validated shell completion installation (bash, zsh, fish, PowerShell, nushell)

---

### Unit 18.1: Install Script E2E Validation Across OS Matrix

**Description:** Test install.sh and install.ps1 on real OS targets using the CI matrix from Phase 17, verifying that the binary lands in the correct location, PATH is updated, version output succeeds, and integrity verification works on every platform.

**Context:** Phase 10 produced install.sh (bash, for macOS + Linux) and install.ps1 (PowerShell, for Windows). These scripts detect the OS and architecture, download the correct binary, verify its SHA256 checksum, install to a default or custom location, and update the user's PATH. They are the primary distribution channel (Tier 4 in the distribution research) — most users will encounter gdev for the first time via `curl -fsSL https://get.gdev.dev | sh`. Any failure here is a first-impression killer. Phase 17 provides the CI matrix infrastructure (GitHub Actions runners, Docker containers, Incus VMs) to test these scripts at scale. This unit runs the install scripts against every target OS and validates the full install lifecycle including idempotent re-install and version pinning.

**Desired Outcome:** install.sh succeeds on all 14 Unix targets (macOS ARM64, macOS Intel, 12 Linux distros). install.ps1 succeeds on Windows Server 2025. Every target confirms: binary in correct path, PATH updated, `gdev --version` outputs the expected version, SHA256 tampering is detected and rejected.

**Steps:**
1. Create test harness script (`test/install-validation.sh`) that wraps install.sh execution with pre/post assertions:
   - Pre: record PATH, verify no prior gdev installation
   - Execute: run install.sh
   - Post: verify binary exists at `~/.gdev/bin/gdev`, PATH includes `~/.gdev/bin/`, `gdev --version` exits 0 and outputs expected version string
2. Run install.sh on macOS targets:
   - `macos-15` (Apple Silicon ARM64) — GitHub Actions runner
   - `macos-15-large` or `macos-13` (Intel x86_64) — GitHub Actions runner
   - Bare macOS test: uninstall Homebrew first (`/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/uninstall.sh)"`), verify install.sh handles missing Homebrew gracefully (installs without Homebrew dependency)
3. Run install.sh on Linux targets via Docker containers or Incus VMs:
   - Debian family: Ubuntu 24.04 (`ubuntu:24.04`), Debian 12 (`debian:12`)
   - RHEL family: Fedora 41 (`fedora:41`), Rocky 9 (`rockylinux:9`)
   - Arch family: Arch Linux (`archlinux:base`)
   - SUSE family: openSUSE Tumbleweed (`opensuse/tumbleweed`)
   - Alpine: Alpine 3.20 (`alpine:3.20`)
   - Void: Void Linux (`voidlinux/voidlinux`)
   - Gentoo: Gentoo (`gentoo/stage3`)
   - NixOS: NixOS container (`nixos/nix`)
4. Run install.sh on WSL2 targets within Windows runners:
   - WSL2 Ubuntu (via `wsl --install -d Ubuntu` on Windows runner)
   - WSL2 Fedora (if available; skip with documented reason if not installable in CI)
5. Run install.ps1 on Windows Server 2025 (`windows-2025` GitHub Actions runner):
   - Verify binary at `$env:LOCALAPPDATA\gdev\bin\gdev.exe`
   - Verify User PATH updated (not System PATH — no admin required)
   - Verify `gdev --version` succeeds in both PowerShell and cmd.exe
6. Test SHA256 integrity verification on 3 representative targets (macOS ARM64, Ubuntu 24.04, Windows 2025):
   - Download binary and checksum file
   - Tamper with checksum (modify one character)
   - Run install script — verify it exits non-zero with clear error message
   - Tamper with binary (append a byte) but leave checksum intact — verify rejection
7. Test version pinning on 3 representative targets:
   - `GDEV_VERSION=1.0.0 curl -fsSL https://get.gdev.dev | sh` — verify installs 1.0.0, not latest
   - `$env:GDEV_VERSION = "1.0.0"; irm https://get.gdev.dev/windows | iex` — same on Windows
8. Test install directory override on 3 representative targets:
   - `GDEV_INSTALL_DIR=/opt/gdev curl -fsSL https://get.gdev.dev | sh` — verify binary at `/opt/gdev/bin/gdev`
   - `$env:GDEV_INSTALL_DIR = "C:\tools\gdev"; irm ... | iex` — verify binary at custom Windows path
9. Test idempotent re-install on 3 representative targets:
   - Run install.sh twice in succession
   - Verify second run succeeds without errors
   - Verify PATH not duplicated in shell RC file
   - Verify binary is the same version (not corrupted by overwrite race)
10. Collect timing data: record install duration on each platform. Flag any target exceeding 30 seconds (excluding download time).

**Acceptance Criteria:**
- [ ] install.sh exits 0 on all 14 Unix targets (macOS ARM64, macOS Intel, Ubuntu 24.04, Debian 12, Fedora 41, Rocky 9, Arch, openSUSE TW, Alpine 3.20, Void, Gentoo, NixOS, WSL2 Ubuntu, WSL2 Fedora)
- [ ] install.ps1 exits 0 on Windows Server 2025
- [ ] Binary lands in correct default location on every target (`~/.gdev/bin/gdev` on Unix, `$env:LOCALAPPDATA\gdev\bin\gdev.exe` on Windows)
- [ ] PATH updated in correct shell RC file on every target
- [ ] `gdev --version` outputs expected version on every target
- [ ] SHA256 tampering detected and rejected with clear error message
- [ ] Version pinning installs the specified version, not latest
- [ ] Install directory override places binary at custom path
- [ ] Re-install is idempotent (no duplicate PATH entries, no errors)
- [ ] Bare macOS (no Homebrew) installs successfully
- [ ] All install tests complete within CI time budget (< 10 minutes for PR subset, < 20 minutes for nightly full matrix)

**Research Citations:**
- `artifacts/cross-platform-testing-infrastructure-research.md § 7. macOS Testing Challenges` — macOS runner labels, Apple Silicon vs Intel testing, bare macOS (no Xcode CLT) scenarios
- `artifacts/cross-platform-testing-infrastructure-research.md § 8. Windows Testing Challenges` — Windows runner capabilities, PowerShell install testing, WSL2 inside GitHub Actions
- `artifacts/cross-platform-testing-infrastructure-research.md § 3. Container-Based Testing` — Docker image tags for each Linux distro, container limitations
- `artifacts/cross-platform-distribution-research.md § 1.3 Install Script Pattern` — install.sh and install.ps1 reference implementations, SHA256 verification flow

**Status:** Not Started

---

### Unit 18.2: `gdev doctor` Validation Across OS Matrix

**Description:** Test that `gdev doctor` produces correct diagnostics on every supported target, verifying OS, distro, version, architecture, kernel, shell, package manager, and tool detection are all accurate.

**Context:** Phase 9 implemented the OS detection engine (`DetectOS()` returning an `OSInfo` struct) and the `gdev doctor` command that reports system state. `gdev doctor` is the user's first diagnostic tool when something goes wrong — if it misreports the OS or fails to detect the package manager, all downstream suggestions will be wrong. The detection engine uses `/etc/os-release` parsing on Linux, `sw_vers` on macOS, and `runtime.GOOS` + registry queries on Windows. WSL2 detection checks for `/proc/sys/fs/binfmt_misc/WSLInterop` and `$WSL_DISTRO_NAME`. Container detection checks `/.dockerenv`, cgroup v2, and container environment variables. Every detection path must be validated against real OS instances — mocked unit tests are necessary but not sufficient.

**Desired Outcome:** `gdev doctor` outputs correct system information on every target. `gdev doctor --json` produces valid, parseable JSON. `gdev doctor --check` returns appropriate exit codes based on tool availability. No target reports incorrect OS, distro, package manager, or architecture.

**Steps:**
1. Create golden-file test fixtures for each target OS containing expected `gdev doctor` output fields:
   ```
   testdata/doctor/ubuntu-2404.golden.json
   testdata/doctor/fedora-41.golden.json
   testdata/doctor/macos-arm64.golden.json
   testdata/doctor/windows-2025.golden.json
   ...
   ```
   Each golden file specifies: OS, Distro, Version, Arch, Kernel (pattern match, not exact), Shell, PackageManager, and boolean flags (IsWSL2, IsContainer, IsAppleSilicon).
2. On each target, run `gdev doctor --json` and validate against the golden file:
   - **macOS ARM64**: OS=darwin, Arch=arm64, IsAppleSilicon=true, PackageManager=brew, HomebrewPrefix=/opt/homebrew
   - **macOS Intel**: OS=darwin, Arch=amd64, IsAppleSilicon=false, PackageManager=brew, HomebrewPrefix=/usr/local
   - **Ubuntu 24.04**: OS=linux, Distro=ubuntu, Version=24.04, PackageManager=apt
   - **Debian 12**: OS=linux, Distro=debian, Version=12, PackageManager=apt
   - **Fedora 41**: OS=linux, Distro=fedora, Version=41, PackageManager=dnf
   - **Rocky 9**: OS=linux, Distro=rocky, Version=9.*, PackageManager=dnf
   - **Arch Linux**: OS=linux, Distro=arch, Version=rolling, PackageManager=pacman
   - **openSUSE TW**: OS=linux, Distro=opensuse-tumbleweed, Version=*, PackageManager=zypper
   - **Alpine 3.20**: OS=linux, Distro=alpine, Version=3.20.*, PackageManager=apk
   - **Void Linux**: OS=linux, Distro=void, PackageManager=xbps
   - **Gentoo**: OS=linux, Distro=gentoo, PackageManager=emerge
   - **NixOS**: OS=linux, Distro=nixos, PackageManager=nix
   - **Windows Server 2025**: OS=windows, Arch=amd64, PackageManager includes winget and/or scoop and/or choco
   - **WSL2 Ubuntu**: OS=linux, Distro=ubuntu, IsWSL2=true, WSLDistro=Ubuntu
   - **WSL2 Fedora**: OS=linux, Distro=fedora, IsWSL2=true, WSLDistro=fedoraremix (or similar)
3. Validate container detection in Docker containers:
   - Run `gdev doctor --json` inside each Linux Docker container
   - Verify IsContainer=true on all container targets
   - Verify IsContainer=false on macOS and Windows runners (full VMs, not containers)
4. Validate tool detection by installing/removing specific tools and checking `gdev doctor` output:
   - On Ubuntu 24.04: install git, go, node, curl, jq — verify all detected with correct versions
   - Remove shellcheck — verify doctor reports it missing
   - On macOS: verify nix, devenv, direnv detection (if installed)
   - On Windows: verify git, node, python3, claude detection
   - On NixOS: verify nix detected as package manager, `nix profile` or `environment.systemPackages` noted
5. Validate `gdev doctor --check` exit codes:
   - On a target with all required tools: verify exit code 0
   - On a target missing a required tool (e.g., remove git): verify exit code 1
   - Verify the specific missing tool is named in stderr output
6. Validate `gdev doctor --json` JSON structure:
   - Parse output with `jq .` on Unix, `ConvertFrom-Json` on Windows — verify valid JSON
   - Verify all expected keys present (os, distro, version, arch, kernel, shell, packageManager, tools, flags)
   - Verify tool entries include name, version (or null), path (or null), installed (bool)
7. Performance validation: time `gdev doctor` on each target, verify completion within 2 seconds. If any target exceeds 2 seconds, profile and document the bottleneck (likely slow tool version commands or network checks).

**Acceptance Criteria:**
- [ ] `gdev doctor` reports correct OS on macOS (darwin), Windows (windows), all Linux distros (linux)
- [ ] Distro correctly identified for all 12 Linux families (ubuntu, debian, fedora, rocky, arch, opensuse-tumbleweed, alpine, void, gentoo, nixos, plus WSL2 variants)
- [ ] Package manager correctly detected: apt (Debian/Ubuntu), dnf (Fedora/Rocky), pacman (Arch), zypper (openSUSE), apk (Alpine), xbps (Void), emerge (Gentoo), nix (NixOS), brew (macOS), winget/scoop/choco (Windows)
- [ ] Apple Silicon detection: IsAppleSilicon=true on ARM64 macOS, false on Intel macOS
- [ ] Homebrew prefix: /opt/homebrew on ARM64, /usr/local on Intel
- [ ] WSL2 detection: IsWSL2=true and WSLDistro populated on WSL2 targets
- [ ] Container detection: IsContainer=true inside Docker containers, false on bare metal/VM
- [ ] Tool detection reports correct installed/missing status for git, go, node, npm, nix, devenv, direnv, claude, python3, curl, jq, shellcheck
- [ ] `gdev doctor --json` produces valid JSON parseable by jq / ConvertFrom-Json on all platforms
- [ ] `gdev doctor --check` returns 0 when all required tools present, 1 when missing
- [ ] `gdev doctor` completes in < 2 seconds on all platforms

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 1. OS/Environment Detection Matrix` — detection strategies per OS family, /etc/os-release parsing, macOS sw_vers, Windows registry, WSL2 detection, container detection
- `artifacts/os-prerequisite-detection-research.md § 1.12 Container Detection` — /.dockerenv, cgroup v2, container env vars
- `artifacts/os-prerequisite-detection-research.md § 1.13 Shell Detection` — current shell, RC file locations, shell-specific version variables
- `artifacts/cross-platform-testing-infrastructure-research.md § 3. Container-Based Testing` — what works and what breaks in containers, Docker image tags per distro
- `artifacts/cross-platform-distribution-research.md § 3.1 Runtime OS/Arch Detection` — Go runtime.GOOS/GOARCH, OSInfo struct design

**Status:** Not Started

---

### Unit 18.3: `gdev setup` Validation (Dry-Run) Across Package Managers

**Description:** Test that `gdev setup --dry-run` correctly identifies missing prerequisites and proposes the right install commands with the correct package manager and package names for each target OS.

**Context:** Phase 9 implemented the package manager abstraction layer mapping each prerequisite tool to its correct package name per package manager (e.g., `go` is `golang` on apt, `go` on dnf/pacman/brew, `GoLang.Go` on winget, `dev-lang/go` on Gentoo). `gdev setup --dry-run` runs the detection engine, identifies missing tools, resolves the correct package names for the detected package manager, and prints the install commands that would be executed — without actually executing them. This is the critical pre-flight check: if the proposed commands are wrong, `gdev setup --yes` will fail or install the wrong packages. The package name resolution table spans 13 tools across 12+ package managers, creating 150+ resolution paths that must be individually validated.

**Desired Outcome:** `gdev setup --dry-run` proposes syntactically correct install commands using the right package manager and the right package names on every target. Batch elevation groups all packages into a single sudo invocation. NixOS gets declarative `environment.systemPackages` instead of `nix profile install`. Windows without WSL2 gets a `wsl --install` proposal when Nix-dependent features are needed.

**Steps:**
1. Create test fixtures defining expected install commands per OS:
   ```
   testdata/setup/ubuntu-2404-missing-go-shellcheck.expected
   # Expected output:
   # sudo apt-get install -y golang shellcheck
   
   testdata/setup/fedora-41-missing-go-shellcheck.expected
   # Expected output:
   # sudo dnf install -y golang ShellCheck
   
   testdata/setup/gentoo-missing-go-shellcheck.expected
   # Expected output:
   # sudo emerge dev-lang/go dev-util/shellcheck
   ```
2. On each target, remove specific tools and run `gdev setup --dry-run`:
   - **Ubuntu 24.04**: Remove go, shellcheck → verify `sudo apt-get install -y golang shellcheck`
   - **Debian 12**: Remove go, jq → verify `sudo apt-get install -y golang jq`
   - **Fedora 41**: Remove go, shellcheck → verify `sudo dnf install -y golang ShellCheck`
   - **Rocky 9**: Remove go → verify `sudo dnf install -y golang`
   - **Arch Linux**: Remove go, shellcheck → verify `sudo pacman -S --noconfirm go shellcheck`
   - **openSUSE TW**: Remove go → verify `sudo zypper install -y go`
   - **Alpine 3.20**: Remove go, shellcheck → verify `sudo apk add go shellcheck`
   - **Void Linux**: Remove go → verify `sudo xbps-install -y go`
   - **Gentoo**: Remove go, shellcheck → verify `sudo emerge dev-lang/go dev-util/shellcheck`
   - **NixOS**: Remove shellcheck → verify proposes `environment.systemPackages = with pkgs; [ shellcheck ];` (declarative, not `nix profile install`)
   - **macOS**: Remove go, shellcheck → verify `brew install go shellcheck`
   - **Windows**: Remove go → verify `winget install --id GoLang.Go` or `scoop install go` (based on detected package manager)
3. Validate batch elevation — all packages proposed in a single command:
   - On Ubuntu: remove go, shellcheck, jq → verify single `sudo apt-get install -y golang shellcheck jq` (not three separate sudo commands)
   - On Fedora: same pattern with `sudo dnf install -y`
   - On macOS: no sudo needed for brew — verify `brew install go shellcheck jq` (single invocation, no sudo)
4. Validate dependency ordering in dry-run output:
   - nix listed before devenv (devenv depends on nix)
   - node listed before claude (claude code requires node/npm)
   - git listed before all others (nearly everything depends on git)
5. Validate Windows WSL2 handling:
   - On Windows without WSL2: when nix or devenv is needed, verify `gdev setup --dry-run` proposes `wsl --install` as a prerequisite step before nix installation
   - On Windows with WSL2: verify nix installation proposed inside WSL2 context
6. Validate NixOS declarative handling:
   - On NixOS: verify all proposals use `environment.systemPackages` pattern, not imperative `nix profile install`
   - Verify the output includes the appropriate Nix package attribute paths (e.g., `pkgs.go`, `pkgs.shellcheck`)
7. Test `gdev setup --yes` in throwaway containers (not on persistent CI runners):
   - On Ubuntu 24.04 container: remove go and shellcheck, run `gdev setup --yes`, verify both installed afterward
   - On Fedora 41 container: same test
   - On Alpine 3.20 container: same test
   - Verify exit code 0 when all installations succeed
   - Verify exit code non-zero when a package fails to install (simulate with a non-existent package name)
8. Validate that `gdev setup --dry-run` on a system with all tools present:
   - Output indicates nothing to install
   - Exit code 0

**Acceptance Criteria:**
- [ ] Proposed install commands use the correct package manager on every target (apt, dnf, pacman, zypper, apk, xbps, emerge, nix, brew, winget/scoop)
- [ ] Package name resolution correct for all tested tool x package manager combinations (go->golang on apt, go->GoLang.Go on winget, shellcheck->ShellCheck on Fedora, shellcheck->dev-util/shellcheck on Gentoo)
- [ ] Batch elevation: all apt packages proposed in single `sudo apt-get install -y pkg1 pkg2 pkg3` (not separate invocations)
- [ ] Batch elevation pattern applies to all sudo-requiring package managers (dnf, pacman, zypper, apk, xbps, emerge)
- [ ] NixOS proposes declarative `environment.systemPackages` instead of `nix profile install`
- [ ] Windows proposes `wsl --install` when Nix needed but no WSL2 detected
- [ ] Dependency ordering: nix before devenv, node before claude, git before everything
- [ ] `gdev setup --yes` actually installs missing tools in container environments
- [ ] `gdev setup --dry-run` on a fully-equipped system reports nothing to install (exit 0)
- [ ] All setup tests complete within CI time budget

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 2. Tool Prerequisite Mapping` — per-tool package names across all 12+ package managers, version constraints, alternative install methods
- `artifacts/os-prerequisite-detection-research.md § 5. Privilege Escalation Patterns` — sudo batching, elevation scope minimization, macOS no-sudo-for-brew
- `artifacts/os-prerequisite-detection-research.md § 1.11 NixOS` — NixOS declarative package management patterns, environment.systemPackages
- `artifacts/os-prerequisite-detection-research.md § 4. Windows-Specific Considerations` — WSL2 detection, native vs WSL2 strategy, tools requiring WSL2
- `artifacts/cross-platform-distribution-research.md § 3.4 Package Name Mapping` — package name resolution table, cross-platform abstraction patterns

**Status:** Not Started

---

### Unit 18.4: Package Manager Distribution Validation

**Description:** Test installation via every package manager distribution channel — Homebrew tap, Scoop bucket, Chocolatey, winget, APT .deb, and RPM — verifying that each produces a working gdev binary with shell completions and correct PATH configuration.

**Context:** Phase 10 configured GoReleaser to produce Homebrew formula, Scoop manifest, Chocolatey package, nFPM-generated .deb and .rpm packages, and winget manifest. These are secondary distribution channels (Tier 1-2 in the distribution research) behind the install scripts, but they serve users who prefer managing gdev through their system package manager. Each channel has its own artifact format, metadata requirements, and installation conventions (e.g., Homebrew installs completions to `$(brew --prefix)/share/zsh/site-functions/`, APT installs to `/usr/share/bash-completion/completions/`). These tests run only on nightly/release CI — not every PR — to conserve CI minutes.

**Desired Outcome:** `brew install`, `scoop install`, `choco install`, `winget install`, `dpkg -i`, and `rpm -i` all produce a working gdev binary with correct version, shell completions in system paths, and proper PATH configuration.

**Steps:**
1. **Homebrew tap on macOS ARM64** (`macos-15` runner):
   - `brew tap quantum-serendipity/tap`
   - `brew install quantum-serendipity/tap/gdev`
   - Verify `gdev --version` outputs expected version
   - Verify shell completions installed: `$(brew --prefix)/share/zsh/site-functions/_gdev` exists
   - Verify bash completion: `$(brew --prefix)/etc/bash_completion.d/gdev` exists
   - `brew uninstall gdev` — verify clean removal
2. **Homebrew tap on macOS Intel** (`macos-13` or equivalent Intel runner):
   - Same test as ARM64
   - Verify Homebrew prefix is `/usr/local` (not `/opt/homebrew`)
   - Verify completions at `/usr/local/share/zsh/site-functions/_gdev`
3. **Scoop bucket on Windows** (`windows-2025` runner):
   - `scoop bucket add gdev https://github.com/quantum-serendipity/scoop-gdev`
   - `scoop install gdev`
   - Verify `gdev --version` in PowerShell
   - Verify shim created at `~/scoop/shims/gdev.exe`
   - `scoop uninstall gdev` — verify clean removal
4. **Chocolatey on Windows** (`windows-2025` runner — Chocolatey pre-installed):
   - `choco install gdev -y --source="path/to/nupkg"` (test against local package, not public repo)
   - Verify `gdev --version`
   - Verify installed to `C:\ProgramData\chocolatey\bin\gdev.exe` (or shimmed equivalent)
   - `choco uninstall gdev -y` — verify clean removal
5. **winget on Windows** (`windows-2025` runner — winget available on Server 2025):
   - `winget install --id QuantumSerendipity.gdev --source winget` (or local manifest)
   - Verify `gdev --version`
   - `winget uninstall --id QuantumSerendipity.gdev` — verify clean removal
6. **APT .deb on Ubuntu 24.04** (Docker container):
   - `sudo dpkg -i gdev_*.deb`
   - Verify `gdev --version`
   - Verify binary at `/usr/bin/gdev` or `/usr/local/bin/gdev`
   - Verify bash completion: `/usr/share/bash-completion/completions/gdev` exists
   - Verify zsh completion: `/usr/share/zsh/vendor-completions/_gdev` exists
   - Verify man page: `man gdev` works (if man pages are generated)
   - `sudo dpkg -r gdev` — verify clean removal
7. **APT .deb on Debian 12** (Docker container):
   - Same as Ubuntu to verify cross-Debian compatibility
8. **RPM on Fedora 41** (Docker container):
   - `sudo rpm -i gdev-*.rpm`
   - Verify `gdev --version`
   - Verify bash completion: `/usr/share/bash-completion/completions/gdev` exists
   - Verify zsh completion: `/usr/share/zsh/site-functions/_gdev` exists
   - `sudo rpm -e gdev` — verify clean removal
9. **RPM on Rocky 9** (Docker container):
   - Same as Fedora to verify cross-RHEL compatibility
10. Tag these tests to run only on nightly and release CI workflows (not PR CI):
    - CI matrix label: `schedule: nightly` or `on: release`
    - PR CI runs only install.sh/install.ps1 tests (Unit 18.1)

**Acceptance Criteria:**
- [ ] `brew install quantum-serendipity/tap/gdev` succeeds on macOS ARM64 and Intel
- [ ] `scoop install gdev` succeeds on Windows Server 2025
- [ ] `choco install gdev -y` succeeds on Windows Server 2025
- [ ] `winget install` succeeds on Windows Server 2025
- [ ] `dpkg -i gdev_*.deb` succeeds on Ubuntu 24.04 and Debian 12
- [ ] `rpm -i gdev-*.rpm` succeeds on Fedora 41 and Rocky 9
- [ ] Every channel: `gdev --version` outputs correct version
- [ ] Shell completions installed to system paths by .deb and .rpm packages
- [ ] Shell completions installed by Homebrew formula
- [ ] Clean uninstall works for every package manager (no orphaned files)
- [ ] Package manager tests run only on nightly/release (not every PR)

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 1.1 GoReleaser Configuration` — GoReleaser brews, scoops, chocolateys, nfpms configuration for multi-channel packaging
- `artifacts/cross-platform-distribution-research.md § 1.2 Distribution Channels` — tier classification (Tier 1 primary: Homebrew/Scoop/APT/RPM; Tier 4 direct: install scripts)
- `artifacts/cross-platform-testing-infrastructure-research.md § 8.5 Testing Package Manager Installation` — Scoop, winget, Chocolatey test patterns on Windows runners

**Status:** Not Started

---

### Unit 18.5: Self-Update & Shell Completion Validation

**Description:** Test the self-update mechanism and shell completion installation across platforms, verifying that `gdev self-update` downloads, verifies, and replaces the binary; that rollback works on failure; and that `gdev completion install` correctly configures completions for bash, zsh, fish, PowerShell, and nushell.

**Context:** Phase 10 implemented `gdev self-update` (downloads latest release, verifies SHA256, atomically replaces the binary) and `gdev completion` (generates and installs shell completion scripts). Self-update is safety-critical: a failed update must not leave the user with a corrupted or missing binary. The rollback mechanism copies the current binary to a temp location before replacement and restores it if the download or verification fails. Shell completions must be installed to the correct RC file or directory for the detected shell, and running `gdev completion install` twice must not duplicate entries. Nushell support is included as it's increasingly popular among the NixOS/developer audience that gdev targets.

**Desired Outcome:** Self-update works reliably on all platforms, rolls back on failure, and supports version pinning for downgrades. Shell completions install correctly for all five supported shells and are idempotent.

**Steps:**
1. **Self-update — latest version** (test on macOS ARM64, Ubuntu 24.04, Windows 2025):
   - Install an older version of gdev (e.g., via version-pinned install script)
   - Run `gdev self-update`
   - Verify binary replaced: `gdev --version` reports newer version
   - Verify SHA256 of the new binary matches published checksum
   - Verify old version no longer on disk (atomic replacement, not side-by-side)
2. **Self-update — specific version (rollback/downgrade)** (test on 3 representative targets):
   - Install current version
   - Run `gdev self-update --version 1.0.0`
   - Verify `gdev --version` reports 1.0.0
   - This validates that users can pin to a known-good version
3. **Self-update — suppression via environment variable** (test on 1 target):
   - Set `GDEV_NO_UPDATE_CHECK=1`
   - Run `gdev self-update`
   - Verify the command exits early with a message (not silently) or is blocked entirely
   - Verify no network requests made (can verify via proxy/mock or by disconnecting network)
4. **Self-update — failed download rollback** (test on macOS ARM64, Ubuntu 24.04, Windows 2025):
   - Record current binary hash
   - Simulate download failure: set `GDEV_UPDATE_URL` to a non-existent URL (or use a mock server returning 500)
   - Run `gdev self-update`
   - Verify exit code non-zero with clear error message
   - Verify original binary is still functional: `gdev --version` works and hash matches pre-update hash
5. **Self-update — corrupted download rollback** (test on 1 target):
   - Mock the download to return an invalid binary (e.g., truncated file)
   - Run `gdev self-update`
   - Verify SHA256 verification catches the corruption
   - Verify rollback restores the original binary
6. **Shell completion — generation** (test on Ubuntu 24.04 with all shells available):
   - `gdev completion bash` → verify output is a valid bash completion script (contains `_gdev` function or `complete -F` / `complete -C`)
   - `gdev completion zsh` → verify output starts with `#compdef gdev` or contains appropriate zsh completion syntax
   - `gdev completion fish` → verify output contains `complete -c gdev` directives
   - `gdev completion powershell` → verify output contains `Register-ArgumentCompleter` (test on Windows runner)
   - `gdev completion nushell` → verify output contains nushell `extern` or `def` completion syntax
7. **Shell completion — install** (test on Ubuntu 24.04 for bash/zsh/fish, Windows for PowerShell):
   - **bash**: Run `gdev completion install` in a bash shell → verify `~/.bashrc` contains gdev completion source line
   - **zsh**: Run `gdev completion install` in a zsh shell → verify `~/.zshrc` contains gdev completion source line or file placed in fpath directory
   - **fish**: Run `gdev completion install` → verify `~/.config/fish/completions/gdev.fish` exists with valid content
   - **PowerShell**: Run `gdev completion install` → verify `$PROFILE` contains gdev completion registration
   - **nushell**: Run `gdev completion install` → verify `~/.config/nushell/config.nu` or equivalent updated (if nushell is available in test environment; skip with documented reason if not)
8. **Shell completion — idempotency** (test on 3 shells):
   - Run `gdev completion install` twice for bash, zsh, and fish
   - Verify the RC file / completion file does not contain duplicate entries
   - Count occurrences of the gdev completion line — must be exactly 1
9. **Shell completion — detection** (test on Ubuntu with multiple shells installed):
   - Verify `gdev completion install` auto-detects the current shell (via `$SHELL` or parent process)
   - If `$SHELL` is zsh, verify zsh completion path is used without requiring `--shell zsh` flag

**Acceptance Criteria:**
- [ ] `gdev self-update` downloads, verifies SHA256, and replaces binary on macOS, Linux, and Windows
- [ ] `gdev self-update --version 1.0.0` installs the specified version (for rollback/pinning)
- [ ] `GDEV_NO_UPDATE_CHECK=1` suppresses self-update
- [ ] Failed download triggers rollback — original binary restored and functional
- [ ] Corrupted download caught by SHA256 verification — original binary restored
- [ ] `gdev completion bash` outputs valid bash completion script
- [ ] `gdev completion zsh` outputs valid zsh completion script
- [ ] `gdev completion fish` outputs valid fish completion script
- [ ] `gdev completion powershell` outputs valid PowerShell completion script
- [ ] `gdev completion nushell` outputs valid nushell completion script
- [ ] `gdev completion install` modifies the correct RC file for the detected shell (bash->~/.bashrc, zsh->~/.zshrc, fish->~/.config/fish/completions/gdev.fish, PowerShell->$PROFILE)
- [ ] Running `gdev completion install` twice does not duplicate entries
- [ ] All self-update and completion tests pass within CI time budget

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 6.5 Self-Update Mechanism` — self-update design, atomic replacement, SHA256 verification, rollback on failure, GDEV_NO_UPDATE_CHECK
- `artifacts/os-prerequisite-detection-research.md § 3. Shell Integration` — shell completion generation, per-shell RC file locations, PATH modification patterns
- `artifacts/os-prerequisite-detection-research.md § 3.3 Shell Completions for gdev` — completion generation commands, per-shell setup (bash, zsh, fish, PowerShell, nushell)
- `artifacts/os-prerequisite-detection-research.md § 3.5 Detecting Current Shell and RC File` — shell auto-detection via $SHELL and parent process

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] Install script succeeds on all 24 OS configurations (14 Unix targets from Unit 18.1 + 1 Windows target + 9 package manager channel targets from Unit 18.4)
- [ ] `gdev doctor` reports correct system info on macOS, Windows, all 12 Linux families
- [ ] `gdev doctor --json` produces valid JSON on all platforms
- [ ] `gdev setup --dry-run` proposes correct install commands per package manager
- [ ] Package name resolution correct for all tool x package manager combinations
- [ ] Homebrew, Scoop, Chocolatey, winget, APT, RPM installs all verified
- [ ] Self-update downloads, verifies, and replaces binary
- [ ] Self-update rollback works on simulated failure
- [ ] Shell completions install correctly for bash, zsh, fish, PowerShell, nushell
- [ ] All tests pass within the CI time budgets (PR < 10 minutes, nightly < 20 minutes)
