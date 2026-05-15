# Phase 10: Distribution & Self-Bootstrapping

## Goal

Package gdev as a single downloadable binary/installer for every major platform, with install scripts, package manager publishing, self-update, and a first-run bootstrap experience that requires zero prerequisites beyond downloading the binary. At the end of this phase, a new developer can install gdev with a single command and have a working tool within minutes.

## Dependencies

Phase 1 complete (Go module, addon scaffolding). Phase 9 complete (OS detection, package manager abstraction, `qsdev devenv doctor`, `qsdev devenv setup`).

## Phase Outputs

- GoReleaser configuration building static binaries for linux/darwin/windows × amd64/arm64
- Bash install script for macOS and Linux (curl-pipe-sh with SHA256 verification)
- PowerShell install script for Windows (with SHA256 verification)
- Homebrew tap and Scoop bucket for `brew install` / `scoop install`
- APT (.deb) and RPM (.rpm) packages via nFPM
- GitHub Actions release pipeline triggered on tag push
- Shell completion generation and installation (bash, zsh, fish, PowerShell)
- `qsdev self-update` command
- `qsdev version` command with build metadata
- Embedded assets (templates, skills, completions) via `embed.FS`

---

### Unit 10.1: Static Binary Build Configuration

**Description:** Configure the Go build for cross-platform static binary compilation with version injection, trimpath, and embedded assets.

**Context:** gdev must ship as a single binary with zero runtime dependencies. `CGO_ENABLED=0` produces static binaries that work on any Linux without glibc version concerns (critical for Alpine's musl). Version/commit/date are injected via ldflags at build time. All templates, skills, and completions are embedded via `embed.FS` — no external file extraction needed.

**Desired Outcome:** `go build` produces a static binary for any GOOS/GOARCH combination that includes all embedded assets and reports correct version info.

**Steps:**
1. Create `cmd/gdev/main.go` entry point with version variables:
   ```go
   var (
       version = "dev"
       commit  = "none"
       date    = "unknown"
       builtBy = "manual"
   )
   ```
2. Create `cmd/gdev/version.go` — register `qsdev version` command displaying version, commit, date, GOOS, GOARCH, Go version.
3. Organize `embed.FS` declarations:
   - `addons/devenv/templates/` — devenv.nix, devenv.yaml, .envrc templates
   - `addons/claudecode/templates/` — settings.json, CLAUDE.md templates
   - `addons/claudecode/skills/` — embedded skill files (agent-postmortem, etc.)
   - `addons/claudecode/agents/` — embedded agent definitions (semble-search, etc.)
   - `completions/` — pre-generated shell completions
4. Verify `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" ./cmd/gdev` produces static binary on each platform.
5. Create `Makefile` or `Taskfile.yml` with build targets: `build`, `build-all`, `test`, `lint`, `completions`.
6. Create `internal/version/` package exposing `Info()` struct for use by doctor and self-update.

**Acceptance Criteria:**
- [ ] `CGO_ENABLED=0 go build` succeeds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- [ ] Binary reports injected version via `qsdev version`
- [ ] `embed.FS` templates accessible at runtime
- [ ] Binary is static (no dynamic library dependencies — verify with `ldd` on Linux)
- [ ] Binary size is reasonable (<50MB with all embedded assets)

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 1.1 GoReleaser Configuration` — build config with CGO_ENABLED=0, ldflags, trimpath
- `artifacts/cross-platform-distribution-research.md § 2. Self-Bootstrapping Installer Patterns` — embed.FS for bundling
- `artifacts/cross-platform-distribution-research.md § 6.4 Binary Architecture` — directory layout

**Status:** Not Started

---

### Unit 10.2: GoReleaser Configuration & Release Pipeline

**Description:** Configure GoReleaser to produce binaries, archives, checksums, Linux packages, and publish to Homebrew/Scoop/Chocolatey on tag push via GitHub Actions.

**Context:** GoReleaser is the industry-standard tool for Go binary distribution. A single `.goreleaser.yaml` handles cross-compilation, archive creation, checksum generation, GPG signing, and publishing to multiple package managers. The GitHub Actions workflow triggers on `v*` tag push and runs GoReleaser with the `GITHUB_TOKEN`.

**Desired Outcome:** Pushing a git tag produces a GitHub Release with binaries for all platforms, updates Homebrew tap, Scoop bucket, and Chocolatey package.

**Steps:**
1. Create `.goreleaser.yaml` with the complete configuration from research (see `artifacts/cross-platform-distribution-research.md § 1.1`):
   - Builds: gdev binary for linux/darwin × amd64/arm64 + windows/amd64
   - Archives: tar.gz for Unix, zip for Windows
   - Checksums: SHA256
   - Signs: GPG on checksums
   - nFPM: .deb, .rpm, .apk, .archlinux packages with shell completions in standard locations
   - Homebrew tap: `Quantum-Serendipity/homebrew-tap`
   - Scoop bucket: `Quantum-Serendipity/scoop-bucket`
   - Chocolatey: push to chocolatey.org (optional, can be deferred)
2. Create `.github/workflows/release.yml`:
   - Trigger on `v*` tag push
   - Install GoReleaser
   - Import GPG key from secrets
   - Run `goreleaser release --clean`
   - Required secrets: `HOMEBREW_TAP_GITHUB_TOKEN`, `SCOOP_TAP_GITHUB_TOKEN`, `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`
3. Create `.github/workflows/ci.yml`:
   - Trigger on push/PR to main
   - Matrix: go build on linux/darwin/windows
   - Run `go vet`, `go test`, `golangci-lint`
   - Run GoReleaser in snapshot mode (`--snapshot --skip=publish`)
4. Create Homebrew tap repo skeleton (`Quantum-Serendipity/homebrew-tap`) — GoReleaser auto-updates the formula.
5. Create Scoop bucket repo skeleton (`Quantum-Serendipity/scoop-bucket`) — GoReleaser auto-updates the manifest.
6. Add Cobra shell completion generation to the build: `qsdev completion bash/zsh/fish/powershell` commands that GoReleaser includes in packages.

**Acceptance Criteria:**
- [ ] `goreleaser release --snapshot --clean` produces binaries for all 5 targets
- [ ] Linux packages (.deb, .rpm) include shell completions in correct system paths
- [ ] CI workflow runs tests on push/PR
- [ ] Release workflow publishes to GitHub Releases on tag
- [ ] Homebrew formula updated automatically in tap repo
- [ ] Scoop manifest updated automatically in bucket repo
- [ ] GPG signatures on checksums verify correctly

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 1.1 GoReleaser Configuration` — complete .goreleaser.yaml
- `artifacts/cross-platform-distribution-research.md § 6.2 Build Pipeline` — pipeline diagram
- `artifacts/cross-platform-distribution-research.md § 6.3 Recommended GoReleaser + GitHub Actions Setup` — workflow and secrets

**Status:** Not Started

---

### Unit 10.3: Install Scripts (Bash + PowerShell)

**Description:** Create install scripts that download the correct gdev binary for the user's platform, verify SHA256 checksums, install to a standard location, and set up PATH.

**Context:** Install scripts are the zero-prerequisite entry point. A developer with no tools except curl/PowerShell can install gdev with a single command. The bash script covers macOS and Linux; the PowerShell script covers Windows. Both scripts detect the platform, download the correct archive from GitHub Releases, verify the SHA256 checksum, extract to a standard location, and add to PATH. This is the rustup/mise pattern — the primary distribution method.

**Desired Outcome:** `curl -fsSL https://get.gdev.dev/install.sh | sh` and `irm https://get.gdev.dev/install.ps1 | iex` both install gdev correctly.

**Steps:**
1. Create `scripts/install.sh` (bash):
   - Detect OS (`uname -s`) and arch (`uname -m`), map to GOOS/GOARCH naming
   - Construct download URL from GitHub Releases (latest or pinned version via `QSDEV_VERSION` env var)
   - Download binary archive and checksums.txt
   - Verify SHA256 (`sha256sum` or `shasum -a 256`)
   - Extract to `~/.gdev/bin/` (user-local, no sudo needed)
   - Add `~/.gdev/bin` to PATH in detected shell RC file (bash/zsh/fish)
   - Print success message with next steps (`qsdev devenv doctor`, `qsdev devenv setup`)
   - Error handling: fail on any step, clean up partial downloads
2. Create `scripts/install.ps1` (PowerShell):
   - Detect arch (`[System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture`)
   - Download from GitHub Releases
   - Verify SHA256 (`Get-FileHash`)
   - Extract to `$env:LOCALAPPDATA\gdev\bin\`
   - Add to user PATH via `[Environment]::SetEnvironmentVariable`
   - Print success message
3. Host scripts at a short URL (GitHub Pages on a `get.gdev.dev` subdomain, or raw GitHub URL).
4. Support version pinning: `QSDEV_VERSION=1.2.3 curl ... | sh` installs a specific version.
5. Support install directory override: `GDEV_INSTALL_DIR=/custom/path curl ... | sh`.
6. Test scripts in CI (download from snapshot release, verify install on Ubuntu, macOS, Windows).

**Acceptance Criteria:**
- [ ] Bash script installs correctly on macOS (Intel + Apple Silicon) and Linux (amd64 + arm64)
- [ ] PowerShell script installs correctly on Windows (amd64)
- [ ] SHA256 verification catches tampered downloads
- [ ] PATH is updated in the correct RC file for the detected shell
- [ ] Version pinning works via env var
- [ ] Install directory override works
- [ ] Scripts fail cleanly on unsupported platforms with actionable error message
- [ ] Scripts are idempotent (re-running updates to latest)

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 1.4 Install Scripts` — complete bash and PowerShell scripts
- `artifacts/cross-platform-distribution-research.md § 5. Prior Art` — rustup, mise install script patterns
- `artifacts/cross-platform-distribution-research.md § 6.1 Distribution Strategy` — install script as primary distribution method

**Status:** Not Started

---

### Unit 10.4: Self-Update Mechanism

**Description:** Implement `qsdev self-update` for in-place binary update and a periodic update check that notifies without blocking.

**Context:** Once installed, gdev needs to stay current. `qsdev self-update` downloads the latest release, verifies checksums, and replaces the running binary. A background update check runs periodically (default: weekly, configurable) and prints a non-blocking notice if a newer version exists. Self-update re-uses the install script logic internally: detect platform, download correct binary, verify, replace.

**Desired Outcome:** `qsdev self-update` updates to latest, and `gdev` commands show a one-line notice when outdated.

**Steps:**
1. Create `internal/selfupdate/` package with `update.go`.
2. Implement `CheckForUpdate(currentVersion string) (*Release, error)`:
   - Query `https://api.github.com/repos/Quantum-Serendipity/qsdev/releases/latest`
   - Compare semver (using `golang.org/x/mod/semver` or equivalent)
   - Cache the check result to `~/.gdev/update-check.json` with timestamp
   - Only check if last check was > configured interval ago
3. Implement `DoUpdate(release *Release) error`:
   - Download binary archive for current GOOS/GOARCH
   - Download checksums.txt and verify SHA256
   - Replace current binary (rename current → backup, write new, remove backup on success, restore backup on failure)
   - Print changelog summary (from GitHub Release body)
4. Register `qsdev self-update` command.
5. Add update check to root command's `PersistentPreRun` — runs async, only prints if update available, never blocks, respects `GDEV_NO_UPDATE_CHECK=1`.
6. Support `--force` flag to skip version comparison and reinstall current version.
7. Support `--version <version>` to install a specific version (for rollback).

**Acceptance Criteria:**
- [ ] `qsdev self-update` downloads and replaces binary successfully
- [ ] SHA256 verification prevents installing corrupted updates
- [ ] Failed update restores previous binary (rollback on failure)
- [ ] Update check is non-blocking and respects interval setting
- [ ] `GDEV_NO_UPDATE_CHECK=1` suppresses check entirely
- [ ] `qsdev self-update --version 1.0.0` installs specific version
- [ ] Update check cache prevents excessive API calls

**Research Citations:**
- `artifacts/cross-platform-distribution-research.md § 6.5 Self-Update Mechanism` — CheckForUpdate pattern, GitHub API usage
- `artifacts/cross-platform-distribution-research.md § 2. Self-Bootstrapping Installer Patterns` — update patterns from rustup, mise
- `artifacts/cross-platform-distribution-research.md § 6.6 Key Design Decisions` — non-intrusive, user-controlled update

**Status:** Not Started

---

### Unit 10.5: Shell Completions & PATH Integration

**Description:** Generate and install shell completions for bash, zsh, fish, and PowerShell, and manage PATH integration across shells.

**Context:** Cobra generates shell completions automatically from the command tree. Completions need to be installed in the right location per shell and platform. On macOS with Homebrew, completions go to `$(brew --prefix)/share/zsh/site-functions/`. On Linux, system-wide paths differ by distro. For user-local install, completions go alongside the binary. PATH integration must be idempotent — don't add duplicate entries.

**Desired Outcome:** After install, `qsdev <tab>` completes commands and flags on all supported shells.

**Steps:**
1. Implement Cobra completion commands: `qsdev completion bash`, `qsdev completion zsh`, `qsdev completion fish`, `qsdev completion powershell`.
2. Implement `qsdev completion install` — auto-detect shell and install completion to the correct location:
   - bash: `~/.gdev/completions/gdev.bash`, source from `~/.bashrc`
   - zsh: `~/.gdev/completions/_gdev`, add `~/.gdev/completions` to fpath in `~/.zshrc`
   - fish: `~/.config/fish/completions/gdev.fish` (fish auto-loads from this directory)
   - PowerShell: add `Register-ArgumentCompleter` to `$PROFILE`
   - nushell: generate extern commands for `~/.config/nushell/completions/gdev.nu`
3. Implement PATH management:
   - `ensurePath(dir string, rcFile string)` — add `export PATH="dir:$PATH"` to RC file if not already present
   - Idempotent: check if PATH entry already exists before adding
   - Platform-specific: use `$PROFILE` for PowerShell, `[Environment]::SetEnvironmentVariable` for Windows system PATH
4. Include pre-generated completions in `embed.FS` for the nFPM packages (installed to system completion directories).
5. Wire completion install into `qsdev devenv setup` flow (Unit 9.6).

**Acceptance Criteria:**
- [ ] `qsdev completion bash` outputs valid bash completion script
- [ ] `qsdev completion install` installs to correct location for detected shell
- [ ] Fish completions auto-load without sourcing (correct directory)
- [ ] PowerShell completions work in both pwsh and powershell.exe
- [ ] PATH addition is idempotent (running twice doesn't duplicate)
- [ ] nFPM packages include completions in system directories

**Research Citations:**
- `artifacts/os-prerequisite-detection-research.md § 3. Shell Integration` — completion installation per shell, PATH setup, RC files
- `artifacts/cross-platform-distribution-research.md § 1.1 GoReleaser Configuration` — nFPM contents for completions

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] `goreleaser release --snapshot --clean` produces binaries for all platforms
- [ ] Install script works on fresh Ubuntu, macOS, and Windows environments
- [ ] `qsdev version` shows correct version, commit, OS, arch
- [ ] `qsdev self-update` updates binary and verifies checksums
- [ ] Shell completions work for bash, zsh, fish on macOS and Linux
- [ ] `brew install quantum-serendipity/tap/gdev` works (Homebrew tap)
- [ ] `scoop install gdev` works (Scoop bucket)
- [ ] `.deb` and `.rpm` packages install correctly with completions
- [ ] CI pipeline runs tests on all platforms per push/PR
- [ ] Release pipeline produces a complete GitHub Release on tag push
