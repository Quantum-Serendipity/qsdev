# Phase 9: Cross-Platform System Detection & Package Management

## Goal

Implement the OS detection engine, package manager abstraction layer, tool prerequisite detection, shell integration, and privilege escalation handling that enable gdev to work across Windows, macOS, and all major Linux distributions. At the end of this phase, `qsdev devenv doctor` accurately reports the system environment and `qsdev devenv setup` can install missing prerequisites on any supported platform.

## Dependencies

Phase 1 complete (shared types, addon scaffolding). Phase 6 partial (wizard infrastructure — `qsdev devenv setup` shares the huh TUI patterns).

## Phase Outputs

- `OSInfo` struct populated from runtime detection (OS, distro, version, arch, shell, package managers, WSL2 status, container status)
- Package manager abstraction with implementations for 12 package managers (apt, dnf, pacman, zypper, apk, xbps, emerge, brew, winget, scoop, choco, nix)
- Tool-to-package-name registry mapping 13+ tools across all package managers
- Shell detection and RC file resolution for bash, zsh, fish, PowerShell, nushell
- Privilege escalation abstraction (sudo, doas, pkexec, gsudo, UAC)
- `qsdev devenv doctor` command with structured diagnostic output
- `qsdev devenv setup` command with interactive prerequisite installation

---

### Unit 9.1: OS Detection Engine

**Description:** Implement the `OSInfo` struct and `DetectOS()` function that identifies the operating system, distribution, version, architecture, WSL2 status, and container environment using a layered detection strategy.

**Context:** Every subsequent unit in this phase and in Phase 10 depends on accurate OS detection. The detection must run in <50ms (it runs at gdev startup). The layered approach uses `runtime.GOOS`/`GOARCH` first, then file-based detection (`/etc/os-release`, `sw_vers`), then command-based fallbacks. WSL2 detection is critical — WSL2 appears as `runtime.GOOS == "linux"` but has different capabilities and path semantics. Platform-specific code uses Go build tags (`//go:build linux`, `//go:build darwin`, `//go:build windows`) to avoid importing `golang.org/x/sys/windows` on non-Windows builds.

**Desired Outcome:** `DetectOS()` returns a fully populated `OSInfo` struct on macOS (Intel + Apple Silicon), Windows (native + WSL2), and Linux (Debian, Ubuntu, Fedora, RHEL, Arch, Manjaro, openSUSE, Alpine, Void, Gentoo, NixOS, and derivatives).

**Steps:**
1. Create `internal/sysinfo/` package with `osinfo.go` defining the `OSInfo` struct:
   ```go
   type OSInfo struct {
       OS              string   // "linux", "darwin", "windows"
       Arch            string   // "amd64", "arm64"
       Family          string   // "debian", "rhel", "arch", "suse", "alpine", "void", "gentoo", "nixos", "macos", "windows"
       Distro          string   // "ubuntu", "fedora", "manjaro", etc. (ID from os-release)
       DistroLike      string   // ID_LIKE from os-release
       Version         string   // OS/distro version
       VersionCode     string   // VERSION_CODENAME (e.g., "noble")
       PrettyName      string   // PRETTY_NAME from os-release
       Kernel          string   // uname -r or equivalent
       IsWSL           bool     // Running inside WSL (1 or 2)
       IsWSL2          bool     // Specifically WSL2
       WSLDistro       string   // $WSL_DISTRO_NAME
       IsContainer     bool     // Docker, Podman, or other container
       IsRosetta       bool     // macOS: running under Rosetta 2
       IsSELinux       bool     // SELinux enforcing
       Shell           string   // Current shell ("bash", "zsh", "fish", "powershell", "pwsh", "nushell")
       ShellPath       string   // Full path to current shell binary
       ShellRCFile     string   // Path to shell's RC file (~/.bashrc, ~/.zshrc, etc.)
       PackageManager  string   // Primary detected package manager
       AltPkgManagers  []string // Secondary/alternative package managers
       HasNix          bool     // Nix package manager available (regardless of NixOS)
       HasHomebrew     bool     // Homebrew available
       HomebrewPrefix  string   // /opt/homebrew (Silicon) or /usr/local (Intel)
       XcodeCLT        bool     // macOS: Xcode Command Line Tools installed
       WindowsTerminal bool     // Running in Windows Terminal
       GitBash         bool     // Running in Git Bash/MSYS2
   }
   ```
2. Create `internal/sysinfo/detect.go` with `DetectOS() *OSInfo` — dispatches to platform-specific detection via build tags.
3. Create `internal/sysinfo/detect_linux.go` (build tag `linux`):
   - Parse `/etc/os-release` for ID, ID_LIKE, VERSION_ID, VERSION_CODENAME, PRETTY_NAME.
   - Implement `determineFamily()` mapping: direct ID match (nixos, alpine, void, gentoo, arch, manjaro, endeavouros, garuda) then ID_LIKE chain (debian/ubuntu → debian, fedora/rhel → rhel, suse → suse, arch → arch).
   - Detect WSL via `/proc/version` containing "microsoft" (WSL1) or "microsoft-standard" (WSL2).
   - Detect containers via `/.dockerenv`, `/run/.containerenv`, and cgroup inspection.
   - Detect SELinux via `getenforce`.
4. Create `internal/sysinfo/detect_darwin.go` (build tag `darwin`):
   - Parse `sw_vers` for product version and build version.
   - Detect Apple Silicon via `runtime.GOARCH == "arm64"`.
   - Detect Rosetta via `sysctl -n sysctl.proc_translated`.
   - Detect Xcode CLI Tools via `xcode-select -p` exit code.
   - Detect Homebrew at `/opt/homebrew/bin/brew` (Silicon) and `/usr/local/bin/brew` (Intel).
5. Create `internal/sysinfo/detect_windows.go` (build tag `windows`):
   - Use `golang.org/x/sys/windows` for admin detection via `AllocateAndInitializeSid`.
   - Detect Windows version via `cmd /c ver`.
   - Detect PowerShell Core (`pwsh`) vs Windows PowerShell (`powershell`).
   - Detect Git Bash/MSYS2 via `$MSYSTEM` env var.
   - Detect Windows Terminal via `$WT_SESSION` env var.
   - Detect package managers: winget > scoop > choco priority order.
6. Create `internal/sysinfo/shell.go` — shell detection and RC file resolution:
   - Detect current shell from `$SHELL` (login shell), `/proc/$PPID/comm` (Linux), `ps -p $PPID` (macOS).
   - Map shell to RC file: bash→`~/.bashrc`, zsh→`~/.zshrc`, fish→`~/.config/fish/config.fish`, pwsh→`$PROFILE`, nushell→`~/.config/nushell/config.nu`.
7. Create `internal/sysinfo/osrelease.go` — `/etc/os-release` parser (pure string parsing, no external dependencies).
8. Write comprehensive unit tests with table-driven test cases for `determineFamily()`, `parseOSRelease()`, and shell RC file resolution. Use mock data for `/etc/os-release` content from each supported distro.

**Acceptance Criteria:**
- [ ] `DetectOS()` compiles on linux, darwin, and windows (build tags)
- [ ] Correctly identifies macOS Intel vs Apple Silicon vs Rosetta
- [ ] Correctly identifies WSL1 vs WSL2 vs native Linux
- [ ] Correctly identifies Debian, Ubuntu, Fedora, RHEL, Arch, Manjaro, openSUSE, Alpine, Void, Gentoo, NixOS from `/etc/os-release` fixtures
- [ ] Correctly identifies derivatives (Pop!_OS, Mint, Rocky, Alma, EndeavourOS) via ID_LIKE chain
- [ ] Shell detection returns correct RC file path for bash, zsh, fish, PowerShell, nushell
- [ ] Container detection works for Docker and Podman
- [ ] Detection completes in <50ms (no slow commands in the hot path)
- [ ] Unit tests pass with fixture data covering all 12+ distro families
- [ ] `golang.org/x/sys/windows` only imported on Windows builds (build tags correct)

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 1. OS/Environment Detection Matrix` — detection methods per OS/distro
- `artifacts/os-prerequisite-detection-research.md § 6. Go Implementation Patterns` — OSInfo struct, determineFamily(), parseOSRelease()
- `artifacts/cross-platform-distribution-research.md § 3. OS Detection and Package Manager Abstraction` — detection approach recommendation

**Status:** Not Started

---

### Unit 9.2: Package Manager Abstraction Layer

**Description:** Implement a `PackageManager` interface with concrete implementations for 12 package managers, plus a tool-to-package-name registry that resolves platform-specific package names.

**Context:** Different OS families use different package managers, and the same tool often has different package names across them (e.g., `shellcheck` on Debian vs `ShellCheck` on Fedora vs `dev-util/shellcheck` on Gentoo). The abstraction provides a uniform `Install(tool)` API that resolves the correct package name and invokes the right command. This powers `qsdev devenv setup` and the bootstrap step registration from Phase 6.

**Desired Outcome:** A `PackageManager` interface that can install any prerequisite tool on any supported OS with the correct package name and elevation.

**Steps:**
1. Create `internal/pkgmanager/` package with `interface.go`:
   ```go
   type PackageManager interface {
       Name() string
       Available() bool
       NeedsElevation() bool
       UpdateIndex(ctx context.Context) error
       Install(ctx context.Context, packages ...string) error
       IsInstalled(pkg string) bool
       SearchCmd() string
   }
   ```
2. Implement `apt.go` — `apt-get install -y`, needs sudo, `apt-get update` for index refresh.
3. Implement `dnf.go` — `dnf install -y`, needs sudo. Includes yum fallback for older RHEL/CentOS.
4. Implement `pacman.go` — `pacman -S --noconfirm`, needs sudo. Includes AUR helper detection (paru > yay) for AUR-only packages.
5. Implement `zypper.go` — `zypper install -y`, needs sudo.
6. Implement `apk.go` — `apk add`, needs sudo.
7. Implement `xbps.go` — `xbps-install -Sy`, needs sudo.
8. Implement `emerge.go` — `emerge`, needs sudo. Note: Gentoo uses category/package format.
9. Implement `brew.go` — `brew install`, no sudo needed. Handles Apple Silicon vs Intel prefix.
10. Implement `winget.go` — `winget install --id <id> -e`, no elevation needed.
11. Implement `scoop.go` — `scoop install`, no elevation needed.
12. Implement `choco.go` — `choco install -y`, handles its own UAC.
13. Implement `nix.go` — `nix profile install nixpkgs#<pkg>`, no elevation needed. Provides an escape hatch for NixOS users who prefer declarative config (output nix expression instead of running install).
14. Create `internal/pkgmanager/registry.go` — tool-to-package-name mapping table:
    - Covers: git, go, nodejs, python3, curl, wget, jq, shellcheck, shfmt, hadolint, direnv, pre-commit
    - Maps generic name → distro/family/package-manager-specific name
    - Lookup order: exact distro → family → package manager → generic fallback
15. Create `internal/pkgmanager/detect.go` — `DetectPackageManager(osInfo *sysinfo.OSInfo) PackageManager` resolving the primary package manager from OS detection results.
16. Write unit tests for package name resolution covering all tool × package manager combinations.

**Acceptance Criteria:**
- [ ] All 12 package manager implementations compile and satisfy the interface
- [ ] Package name registry correctly maps `shellcheck` → `ShellCheck` on Fedora, `dev-util/shellcheck` on Gentoo, `shellcheck` elsewhere
- [ ] Package name registry correctly maps `go` → `golang` on Debian, `GoLang.Go` on winget, `dev-lang/go` on Gentoo
- [ ] `DetectPackageManager()` selects the correct implementation based on `OSInfo`
- [ ] Nix implementation offers declarative alternative for NixOS users
- [ ] AUR helper integration works for Arch packages not in official repos
- [ ] Unit tests verify name resolution for all 13 tools × 12 package managers

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 2. Tool Prerequisite Mapping` — install commands per tool per platform
- `artifacts/os-prerequisite-detection-research.md § 6.3 Package Manager Resolution` — detectPackageManagers() implementation
- `artifacts/os-prerequisite-detection-research.md § 6.5 Tool Installation Dispatch` — ToolInstaller, resolvePackageName()
- `artifacts/cross-platform-distribution-research.md § 3. OS Detection and Package Manager Abstraction` — PackageManager interface pattern

**Status:** Not Started

---

### Unit 9.3: Privilege Escalation Abstraction

**Description:** Implement a cross-platform privilege escalation layer that minimizes elevation prompts by batching elevated operations and detecting cached credentials.

**Context:** System package managers (apt, dnf, pacman) need root. The escalation layer detects available elevation tools (sudo, doas, pkexec on Linux; gsudo, native sudo, Start-Process on Windows), checks for cached credentials (`sudo -n true`), and batches multiple package installs into a single elevated command to minimize password prompts. macOS Homebrew does not need elevation. Windows winget/scoop do not need elevation; choco handles its own UAC.

**Desired Outcome:** A `Privilege` package that wraps any command with appropriate elevation, batches elevated operations, and presents a single sudo/UAC prompt for multiple installs.

**Steps:**
1. Create `internal/privilege/` package with `escalate.go`.
2. Implement `NeedsElevation() bool` — checks `os.Getuid() == 0` on Unix, `windows.AllocateAndInitializeSid` on Windows.
3. Implement `DetectElevationTool() string` — finds best available: sudo > doas > pkexec (Linux/macOS), gsudo > native-sudo (Windows 11 24H2+) > Start-Process (Windows).
4. Implement `HasCachedCredentials() bool` — runs `sudo -n true` (Linux/macOS) to check if sudo session is active.
5. Implement `ElevatedExec(ctx context.Context, cmd string, args ...string) error` — wraps command with elevation tool.
6. Implement `BatchElevatedInstall(ctx context.Context, pm PackageManager, packages []string) error` — collects all packages, runs a single elevated install command (e.g., `sudo apt-get install -y pkg1 pkg2 pkg3`).
7. Create `internal/privilege/escalate_windows.go` (build tag `windows`) — Windows-specific admin detection via `golang.org/x/sys/windows`.
8. Create `internal/privilege/escalate_unix.go` (build tag `!windows`) — Unix `os.Getuid()` path.
9. Write tests verifying detection logic (mock exec.LookPath).

**Acceptance Criteria:**
- [ ] Correctly detects root/admin status on Linux, macOS, and Windows
- [ ] Detects sudo, doas, pkexec, gsudo availability
- [ ] Cached credential check avoids unnecessary prompts
- [ ] Batch install produces a single `sudo apt-get install -y pkg1 pkg2 pkg3` rather than N separate sudo calls
- [ ] Windows admin detection uses proper SID check (not `os.Getuid()` which returns -1 on Windows)
- [ ] Build tags prevent `golang.org/x/sys/windows` import on non-Windows

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 5. Privilege Escalation Patterns` — sudo/doas/pkexec/gsudo detection, cached credentials, batch elevation
- `artifacts/os-prerequisite-detection-research.md § 1.3 Windows — Admin/Elevation Detection` — Windows SID-based admin check

**Status:** Not Started

---

### Unit 9.4: Tool Prerequisite Detection & Health Checks

**Description:** Implement the prerequisite detection engine that checks for the presence and version of all required/optional tools, producing a structured health report used by `qsdev devenv doctor` and `qsdev devenv setup`.

**Context:** Before gdev can bootstrap a development environment, it must know what's already installed and what's missing. This engine checks for 13+ tools, extracts versions where possible, and classifies each as: installed (with version), missing (installable), or missing (requires manual steps). The results feed into `qsdev devenv doctor` (diagnostic display) and `qsdev devenv setup` (guided installation).

**Desired Outcome:** A `ToolCheck` engine that produces a complete prerequisite report in <2 seconds across all platforms.

**Steps:**
1. Create `internal/doctor/` package with `checks.go`.
2. Define `ToolStatus` struct: `Name string`, `Required bool`, `Installed bool`, `Version string`, `MinVersion string`, `Path string`, `InstallMethod string`, `AutoInstallable bool`, `Notes string`.
3. Define tool check registry — each tool has a check function:
   - `git` — `git --version`, required, auto-installable everywhere
   - `go` — `go version`, required for gdev development, extract version
   - `node` — `node --version`, required for Claude Code, extract version
   - `npm` — `npm --version`, required for Claude Code
   - `nix` — `nix --version`, optional (required for devenv), auto-installable on Linux/macOS/WSL2, NOT on native Windows
   - `devenv` — `devenv version`, optional, requires nix
   - `direnv` — `direnv version`, optional, auto-installable
   - `claude` — `claude --version`, optional (Claude Code CLI), installed via npm
   - `pre-commit` / `prek` — version check, optional
   - `shellcheck` — `shellcheck --version`, optional
   - `shfmt` — `shfmt --version`, optional
   - `hadolint` — `hadolint --version`, optional
   - `jq` — `jq --version`, recommended (used by Version-Sentinel)
   - `curl` — `curl --version`, recommended
   - `python3` — `python3 --version`, recommended (used by Version-Sentinel), check >=3.11
4. For each check: `exec.LookPath` for existence, version command for version extraction, compare against minimum version where applicable.
5. Classify auto-installability: map each tool × current OS to whether `qsdev devenv setup` can install it automatically.
6. Handle nix-on-Windows special case: nix is NOT installable on native Windows, only in WSL2. Flag this clearly.
7. Handle NixOS special case: suggest declarative configuration instead of `nix profile install` when `OSInfo.Family == "nixos"`.
8. Implement `RunAllChecks(osInfo *sysinfo.OSInfo) []ToolStatus` — runs all checks in parallel (each check is independent).
9. Write unit tests with mock exec for version extraction parsing.

**Acceptance Criteria:**
- [ ] Detects all 15 tools with correct version extraction
- [ ] Correctly classifies auto-installability per OS
- [ ] Nix flagged as not-auto-installable on native Windows
- [ ] NixOS users get declarative config suggestion
- [ ] Parallel execution completes in <2 seconds
- [ ] Version comparison correctly flags outdated tools (e.g., python3 < 3.11)
- [ ] Unit tests verify version parsing for each tool

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 2. Tool Prerequisite Mapping` — tool-by-tool install commands and detection
- `artifacts/cross-platform-distribution-research.md § 4. Prerequisite Installation Strategies` — auto-install feasibility analysis
- `artifacts/os-prerequisite-detection-research.md § 4. Windows-Specific Considerations` — nix/WSL2 constraints

**Status:** Not Started

---

### Unit 9.5: `qsdev devenv doctor` Command

**Description:** Implement the `qsdev devenv doctor` command that displays a comprehensive system diagnostic report: OS info, detected tools, package managers, shell, and recommended actions.

**Context:** `qsdev devenv doctor` is the first thing a developer runs when troubleshooting. It should produce a clear, structured report that support engineers can read. The output format follows the pattern established by `flutter doctor`, `brew doctor`, and `mise doctor`. The command uses Units 9.1-9.4 to gather all information.

**Desired Outcome:** `qsdev devenv doctor` prints a complete diagnostic report with actionable recommendations.

**Steps:**
1. Create `internal/doctor/report.go` with `FormatReport(osInfo *sysinfo.OSInfo, checks []ToolStatus) string`.
2. Report sections:
   - **System**: OS, distro, version, architecture, kernel, WSL status, container status
   - **Shell**: Current shell, version, RC file path
   - **Package Managers**: Primary and alternatives, with versions
   - **Required Tools**: Status table with ✓/✗, version, path
   - **Optional Tools**: Status table with ✓/✗/—, version, path
   - **Recommendations**: Numbered list of actions to resolve missing/outdated tools
3. Register `qsdev devenv doctor` command via Cobra in the devenv addon.
4. Support `qsdev devenv doctor --json` for machine-readable output (JSON marshaling of all data).
5. Support `qsdev devenv doctor --check` exit code mode: exit 0 if all required tools present, exit 1 if any missing.
6. Color-code output: green ✓ for installed, red ✗ for missing required, yellow ! for missing optional.
7. Include `gdev` version and build info in output header.

**Acceptance Criteria:**
- [ ] `qsdev devenv doctor` produces readable diagnostic on macOS, Linux, and Windows
- [ ] Required tools with ✗ show install instructions for the detected OS
- [ ] `qsdev devenv doctor --json` produces valid JSON
- [ ] `qsdev devenv doctor --check` returns correct exit code
- [ ] Report includes actionable fix commands (e.g., "Run: brew install direnv")
- [ ] Color output works in terminals, gracefully degrades in pipes/CI

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 6.4 Binary Architecture` — doctor/ package structure
- `artifacts/cross-platform-distribution-research.md § 2. Self-Bootstrapping Installer Patterns` — `qsdev devenv doctor` in the first-run sequence
- `artifacts/os-prerequisite-detection-research.md § Key Recommendations` — detection priority and fallback chain

**Status:** Not Started

---

### Unit 9.6: `qsdev devenv setup` Command — Interactive Prerequisite Installation

**Description:** Implement the `qsdev devenv setup` command that walks the developer through installing missing prerequisites detected by `qsdev devenv doctor`, using the package manager abstraction and privilege escalation layers.

**Context:** `qsdev devenv setup` is the second step in the first-run sequence (`qsdev devenv doctor` → `qsdev devenv setup` → `qsdev init`). It reads the health check results, presents missing tools, and offers to install them. On NixOS, it suggests declarative configuration. On native Windows without WSL2, it offers to install WSL2 first for Nix-dependent features. The command uses huh for TUI if interactive, or accepts `--yes` for unattended mode.

**Desired Outcome:** A developer on any supported OS can run `qsdev devenv setup` and get all prerequisites installed with minimal friction.

**Steps:**
1. Create `internal/setup/` package with `installer.go`.
2. Implement the setup flow:
   a. Run `RunAllChecks()` to get current status.
   b. Filter to missing tools.
   c. Separate into auto-installable vs manual-only.
   d. Present huh form: checkboxes for which tools to install (all auto-installable pre-checked).
   e. Group by elevation requirement: collect all sudo-requiring packages for a single elevated command.
   f. Execute installations in dependency order (nix before devenv, node before claude).
   g. Re-run checks to verify success.
   h. Display results and remaining manual steps.
3. Handle special installation flows:
   - **Nix**: Use Determinate Systems installer (`curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install`). Requires restart of shell after install.
   - **Claude Code**: `npm install -g @anthropic-ai/claude-code` (requires node).
   - **Homebrew** (macOS, if missing): Offer to install Homebrew first.
   - **WSL2** (Windows, if nix needed): Offer `wsl --install` then restart terminal.
4. Implement `--yes` flag: auto-install all auto-installable tools without prompting.
5. Implement `--dry-run` flag: show what would be installed without executing.
6. Handle NixOS specially: output Nix expressions for `environment.systemPackages` instead of running install commands.
7. Shell integration step: after tool installation, offer to add direnv hook and shell completions to the detected RC file.
8. Register `qsdev devenv setup` command via Cobra.

**Acceptance Criteria:**
- [ ] `qsdev devenv setup` installs git, go, node, direnv on Debian via apt with a single sudo prompt
- [ ] `qsdev devenv setup` installs tools via brew on macOS without sudo
- [ ] `qsdev devenv setup` handles Windows winget/scoop/choco
- [ ] Nix installation uses Determinate Systems installer
- [ ] Claude Code installed after node is confirmed present
- [ ] `--yes` mode works without interaction
- [ ] `--dry-run` shows plan without executing
- [ ] NixOS users get declarative Nix expressions
- [ ] Post-install verification confirms all tools work
- [ ] Shell integration (direnv hook, completions) offered and applied

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 2. Self-Bootstrapping Installer Patterns` — rustup/mise/volta bootstrap patterns
- `artifacts/os-prerequisite-detection-research.md § 2. Tool Prerequisite Mapping` — complete install command table
- `artifacts/os-prerequisite-detection-research.md § 3. Shell Integration` — direnv hooks, completions, PATH setup per shell
- `artifacts/os-prerequisite-detection-research.md § 4. Windows-Specific Considerations` — WSL2 setup, path translation
- `artifacts/os-prerequisite-detection-research.md § 5. Privilege Escalation Patterns` — batch elevation, minimize prompts

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] `qsdev devenv doctor` produces correct diagnostics on macOS (Intel + Apple Silicon), Windows (native + WSL2), Ubuntu, Fedora, Arch, NixOS
- [ ] `qsdev devenv setup --yes` installs all auto-installable prerequisites on a fresh Ubuntu system
- [ ] `qsdev devenv setup` on NixOS produces declarative Nix configuration
- [ ] `qsdev devenv setup` on native Windows detects WSL2 absence and offers installation
- [ ] Package name resolution is correct for all tool × package manager combinations
- [ ] Single sudo prompt for all apt/dnf/pacman packages (batch elevation)
- [ ] Shell completions installed for detected shell
- [ ] `go vet ./...` and `go build ./...` pass with correct build tags on all platforms
