# Phase 17: Test Infrastructure & Framework Foundation

## Goal

Establish the test infrastructure, CI pipeline, and test framework foundations that enable end-to-end validation across all platforms. At the end of this phase, gdev has a working multi-platform CI matrix, testscript-based E2E test framework, install script test suites (BATS + Pester), non-interactive mode for automated testing, and golden file infrastructure for generated content verification.

## Dependencies

Phase 1 complete (Go module). Phase 10 desirable (install scripts and GoReleaser). Phase 6 desirable (wizard — for `--answers-file` and `--non-interactive` support).

## Phase Outputs

- GitHub Actions workflows: quick-validation (every PR), full-matrix (nightly), release-validation
- Docker container matrix covering 11+ Linux distros (Debian, Ubuntu, Fedora, Rocky, Alma, Arch, openSUSE TW, openSUSE Leap, Alpine, Void, Gentoo)
- WSL2 testing via `Vampire/setup-wsl@v4` on `windows-2025` runners
- testscript-based E2E framework with custom commands (`yaml_has`, `json_path`, `nix_valid`)
- BATS test suite for bash install script
- Pester test suite for PowerShell install script
- `--non-interactive` flag and `--answers-file` support for `gdev init`
- Golden file infrastructure for generated content
- Coverage collection from compiled binary (Go 1.20+ `-cover` build flag + `GOCOVERDIR`)
- gotestsum for test reporting (JUnit XML)
- Build-once-test-many artifact pipeline
- Test fixture directory structure and management conventions

---

### Unit 17.1: CI Pipeline Architecture & GitHub Actions Workflows

**Description:** Set up the three-tier CI pipeline with build-once-test-many artifact sharing, conditional matrix expansion, and cost-optimized runner allocation.

**Context:** gdev is a cross-platform static Go binary that must work on macOS (Intel + Apple Silicon), Windows (native + WSL2), Ubuntu, and 11+ Linux distributions. Testing every platform on every PR is expensive and slow. A tiered approach runs the essential matrix on every PR, the full matrix nightly, and release-specific validation on tag pushes. The build-once-test-many pattern compiles gdev binaries in a single job and distributes them as artifacts — avoiding redundant compilation across 20+ matrix cells.

GitHub Actions provides native runners for macOS ARM64 (`macos-15`), macOS Intel (`macos-15-intel`), Windows (`windows-2025`), Ubuntu x86_64 (`ubuntu-24.04`), and Ubuntu ARM64 (`ubuntu-24.04-arm`). Linux distro coverage uses Docker containers via the `container:` directive on `ubuntu-latest` runners. WSL2 testing uses `Vampire/setup-wsl@v4` on `windows-2025` runners.

**Desired Outcome:** Three GitHub Actions workflow files that provide comprehensive cross-platform coverage with `fail-fast: false`, conditional matrix expansion between PR and nightly runs, and a private-repo cost of approximately $2.93 per PR.

**Steps:**
1. Create `.github/workflows/quick-validation.yml` (Tier 1 — every PR):
   - Build job: compile gdev for all 5 target OS/arch combinations (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`), upload as workflow artifacts.
   - Run unit tests with `gotestsum` and upload JUnit XML results.
   - Native runner jobs (5): macOS ARM64 (`macos-15`), macOS Intel (`macos-15-intel`), Windows (`windows-2025`), Ubuntu x86_64 (`ubuntu-24.04`), Ubuntu ARM64 (`ubuntu-24.04-arm`). Each downloads the pre-built binary and runs the testscript E2E suite.
   - Docker container matrix (11 jobs): `debian:12`, `ubuntu:22.04`, `fedora:41`, `rockylinux:9`, `almalinux:9`, `archlinux:latest`, `opensuse/tumbleweed`, `opensuse/leap:15.6`, `alpine:3.20`, `voidlinux/voidlinux`, `gentoo/stage3`. Each uses `container:` directive on `ubuntu-latest`, downloads the pre-built `linux/amd64` binary, and runs distro-specific E2E tests.
   - Set `fail-fast: false` on all matrix strategies.
   - Upload test results and coverage artifacts from every job.
2. Create `.github/workflows/full-matrix.yml` (Tier 2 — nightly + release):
   - All Tier 1 jobs plus:
   - WSL2 jobs (2): Ubuntu-24.04 and Fedora via `Vampire/setup-wsl@v4` on `windows-2025`.
   - NixOS job: `nixos/nix` container.
   - Bare-Mac job: `macos-15` with Homebrew removed to test clean-slate detection.
   - Package manager install jobs: Scoop (`windows-2025`), Chocolatey (`windows-2025`), winget (`windows-2025`), Homebrew tap (`macos-15`), APT `.deb` (`ubuntu-24.04`), RPM (`fedora:41` container).
   - Derivative distro job: Linux Mint (community container).
   - Schedule: `cron: '0 4 * * *'` (4:00 UTC nightly) and on tag push.
3. Create `.github/workflows/release-validation.yml` (Tier 3 — tag push only):
   - Triggered by `v*` tag pushes.
   - Runs the full nightly matrix.
   - Adds install-from-release tests: download from GitHub Releases, verify checksums, run smoke tests.
   - Validates GoReleaser artifacts (all archives, checksums file, SBOM).
4. Implement build-once-test-many pattern:
   - Build job uses `actions/upload-artifact@v4` with retention of 1 day.
   - Test jobs use `actions/download-artifact@v4` to retrieve the correct binary for their platform.
   - Coverage-instrumented binaries built with `go build -cover` for E2E coverage collection.
5. Add workflow-level concurrency control: `concurrency: { group: ${{ github.workflow }}-${{ github.ref }}, cancel-in-progress: true }` on PR workflows.
6. Add cost estimation comment in workflow files documenting expected per-run costs for private repos.

**Acceptance Criteria:**
- [ ] Quick-validation runs on every PR with 5 native + 11 container jobs
- [ ] Full-matrix runs nightly with WSL2, NixOS, bare-Mac, package manager, and derivative distro jobs
- [ ] Release-validation runs on tag push with install-from-release tests
- [ ] Build-once-test-many pattern avoids redundant compilation across matrix cells
- [ ] `fail-fast: false` set on all matrix strategies
- [ ] Concurrency control cancels superseded PR runs
- [ ] Test results uploaded as artifacts from every job
- [ ] Coverage-instrumented binaries used in E2E test jobs

**Research Citations:**
- `artifacts/cross-platform-testing-infrastructure-research.md § 2. GitHub Actions CI Matrix Strategy`
- `artifacts/cross-platform-testing-infrastructure-research.md § 10. Parallel Execution`
- `artifacts/cross-platform-testing-infrastructure-research.md § 11. Recommended Testing Architecture`

**Status:** Not Started

---

### Unit 17.2: testscript E2E Framework & Custom Commands

**Description:** Set up Roger Peppe's `testscript` package as the primary E2E test framework with custom commands for YAML, JSON, Nix, and section marker verification, plus platform condition predicates.

**Context:** The `testscript` package (`github.com/rogpeppe/go-internal/testscript`) is the same engine behind the Go project's own 900+ script tests. It provides a txtar-based DSL for writing E2E tests as human-readable scripts — each test is an isolated scenario with its own temp directory, environment, and file tree. The framework supports custom commands (extending the DSL), custom conditions (platform/tool predicates), and golden file comparison via `cmp`. This is the industry-standard approach for Go CLI testing — HashiCorp, CUE, and the Go toolchain itself all use it.

For gdev, testscript is ideal because each test can declare a project fixture as an embedded txtar archive, run `gdev init --answers-file answers.yaml`, and then verify every generated file with `cmp`, `yaml_has`, `json_path`, or `nix_valid`.

**Desired Outcome:** A working testscript E2E framework with 3-5 initial test scripts demonstrating the pattern, custom verification commands, and platform condition predicates.

**Steps:**
1. Create `e2e/e2e_test.go` with:
   - `TestMain` function that registers gdev's `main()` function as the `gdev` command via `testscript.RunMain`.
   - `TestScripts` function that calls `testscript.Run` with the `testdata/script/` directory.
   - Set `UpdateScripts: *update` (wired to `-update` flag) for golden file auto-updating.
   - Set `GOCOVERDIR` propagation for coverage collection from testscript runs.
2. Implement custom testscript commands:
   - `yaml_has <file> <key.path> [expected_value]` — parse YAML file, walk dot-separated key path, assert existence (no expected value) or equality (with expected value). Supports nested keys like `packages.go`, `enterShell.0`.
   - `json_path <file> <json.path> [expected_value]` — parse JSON file, evaluate JSONPath expression, assert result. Covers settings.json and .mcp.json verification.
   - `nix_valid <file>` — run `nix-instantiate --parse <file>` or equivalent syntax check. Skip if `nix` is not available (gated on `[exec:nix]` condition).
   - `file_hash <file> <expected_sha256>` — compute SHA256 of file and compare against expected hash. For verifying embedded assets are correctly extracted.
   - `section_present <file> <marker>` — assert that a section marker pair exists in the file (e.g., `<!-- gdev:semgrep -->` ... `<!-- /gdev:semgrep -->` in CLAUDE.md, or `# --- semgrep ---` ... `# --- end semgrep ---` in devenv.nix).
   - `section_absent <file> <marker>` — assert that a section marker pair does NOT exist in the file. Used to verify `gdev disable` cleanup.
3. Implement platform condition predicates:
   - `has_apt` — true if `apt-get` is in PATH (Debian/Ubuntu containers).
   - `has_brew` — true if `brew` is in PATH (macOS runners).
   - `has_nix` — true if `nix` is in PATH (NixOS container, machines with Nix installed).
   - `has_docker` — true if `docker` is in PATH and daemon is running.
   - `has_python3` — true if `python3` is in PATH (semble prerequisite).
   - `is_wsl` — true if running inside WSL2.
   - `is_container` — true if running inside a Docker/Podman container.
4. Set up isolated environment per script:
   - Fresh `$WORK` directory (testscript default).
   - Set `XDG_CONFIG_HOME=$WORK/.config` to prevent polluting host.
   - Set `GDEV_NON_INTERACTIVE=1` to suppress TUI.
   - Set `HOME=$WORK/home` to isolate shell RC file modifications.
   - Clear `PATH` additions from previous tests (testscript handles this via isolation).
5. Set up `GOCOVERDIR` for coverage collection:
   - Create `$WORK/.coverdir` in each test's setup.
   - Set `GOCOVERDIR=$WORK/.coverdir` in test environment.
   - After each test, copy coverage data to a shared directory for merging.
6. Create initial test scripts:
   - `e2e/testdata/script/init-basic.txt` — `gdev init --answers-file answers.yaml` with minimal Go project answers, verify devenv.yaml, devenv.nix, .envrc generated with expected content.
   - `e2e/testdata/script/doctor-basic.txt` — `gdev doctor` in a fresh directory, verify exit code and output format.
   - `e2e/testdata/script/version.txt` — `gdev version`, verify version string format.
   - `e2e/testdata/script/init-noninteractive.txt` — verify `GDEV_NON_INTERACTIVE=1` produces same output as `--non-interactive` flag.
   - `e2e/testdata/script/init-idempotent.txt` — run `gdev init` twice, verify second run detects existing files and handles merge strategy.

**Acceptance Criteria:**
- [ ] `go test ./e2e/...` runs all testscript files in `testdata/script/`
- [ ] `yaml_has` correctly validates YAML key paths and values
- [ ] `json_path` correctly validates JSON path expressions
- [ ] `nix_valid` validates Nix syntax (skipped when nix not available)
- [ ] `section_present` and `section_absent` correctly detect section markers
- [ ] Platform conditions gate tests to appropriate runners
- [ ] Each test runs in full isolation (no cross-test contamination)
- [ ] Coverage data collected from testscript runs via `GOCOVERDIR`
- [ ] `go test -update ./e2e/...` auto-updates golden files
- [ ] At least 3 initial test scripts pass

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 1. Go Test Framework Patterns for CLI Tools`
- `artifacts/e2e-test-automation-framework-research.md § 2. File Content Verification`
- `artifacts/e2e-test-automation-framework-research.md § 7. Test Data Management`

**Status:** Not Started

---

### Unit 17.3: Non-Interactive Mode & Answers File

**Description:** Implement `--non-interactive` flag and `--answers-file <path>` for `gdev init` to enable automated testing without TUI interaction.

**Context:** This is the single most important testing enabler. Every E2E test, every CI run, and every scripted validation depends on being able to run `gdev init` without a human operating the TUI wizard. Without this, testscript tests cannot exercise the init workflow, the CI pipeline cannot validate generated output, and golden file comparison is impossible.

The design provides three levels of non-interactive control: (1) `--non-interactive` flag or `GDEV_NON_INTERACTIVE=1` env var skips the TUI and uses defaults, (2) `--answers-file <path>` reads wizard answers from a YAML file and exercises the same config logic as the TUI but without interaction, (3) auto-detection of non-interactive terminals via `os.Stdin.Fd()` (e.g., piped input, `CI=true` environments).

The answers file format mirrors the wizard's question structure — each key corresponds to a wizard prompt, and the value is what the user would have selected. This means answers files are self-documenting and can serve as both test fixtures and documentation of gdev's configuration surface.

**Desired Outcome:** `gdev init --answers-file go-defaults.yaml` produces identical output to a human clicking through the TUI with the same choices, enabling deterministic E2E testing.

**Steps:**
1. Add `--non-interactive` flag to `gdev init` command:
   - Boolean flag, default false.
   - When set, skip all TUI prompts and use detection-based defaults.
   - Wire into `GDEV_NON_INTERACTIVE` env var (flag takes precedence).
   - Also trigger when `CI=true` is set (common in all CI environments).
2. Implement terminal detection:
   - Check `term.IsTerminal(int(os.Stdin.Fd()))` from `golang.org/x/term`.
   - When stdin is not a terminal AND `--non-interactive` is not explicitly set, auto-enable non-interactive mode with a log message explaining why.
   - When stdin IS a terminal and `--non-interactive` is set, respect the flag (user explicitly wants non-interactive).
3. Add `--answers-file <path>` flag to `gdev init` command:
   - Path to a YAML file containing wizard answers.
   - Implies `--non-interactive` (no TUI prompts even if answers are incomplete).
   - Missing answers fall back to detection-based defaults.
   - Extra answers (for tools not detected) are applied anyway — this allows testing tool combinations not naturally detected.
4. Define answers file schema:
   ```yaml
   # gdev init answers file
   project_name: my-project
   ecosystems: [go, docker]
   profile: consulting-default
   security:
     age_gating: true
     install_script_blocking: true
     lockfile_enforcement: true
   tools:
     semgrep: true
     gitleaks: true
     version_sentinel: true
     semble: false
   services: [postgresql, redis]
   ci_platform: github
   ```
5. Implement answers file loading:
   - Parse YAML with strict mode (unknown keys are errors, preventing stale answers files).
   - Validate against known answer keys.
   - Merge with detected defaults: answers file values override detection, detection fills gaps.
6. Ensure the answers file exercises the same code paths as the TUI:
   - The wizard's form fields each have a key in the answers schema.
   - The answers file feeds the same `Config` struct that the TUI populates.
   - No separate "batch mode" code path — one config-building pipeline with two input sources (TUI or file).
7. Create sample answers files for testing:
   - `testdata/answers/go-defaults.yaml` — minimal Go project with default security settings.
   - `testdata/answers/typescript-full.yaml` — TypeScript + React + Docker with all security tools enabled.
   - `testdata/answers/minimal.yaml` — bare minimum: single ecosystem, no optional tools.
   - `testdata/answers/all-tools.yaml` — every optional tool enabled, for maximum coverage testing.
   - `testdata/answers/multi-ecosystem.yaml` — Go + Python + Docker + Terraform, testing ecosystem composition.
8. Add validation: `gdev init --answers-file <path> --dry-run` previews what would be generated without writing files. Useful for CI validation of answers files themselves.

**Acceptance Criteria:**
- [ ] `gdev init --non-interactive` completes without any TUI prompts
- [ ] `gdev init --answers-file go-defaults.yaml` produces correct Go project output
- [ ] `GDEV_NON_INTERACTIVE=1 gdev init` behaves identically to `--non-interactive` flag
- [ ] `CI=true gdev init` auto-enables non-interactive mode
- [ ] Non-terminal stdin auto-detected and triggers non-interactive mode
- [ ] Answers file with unknown keys produces a clear error
- [ ] Missing answers fall back to detection-based defaults
- [ ] Same `Config` struct populated by both TUI and answers file (no divergent code paths)
- [ ] `--dry-run` with `--answers-file` previews output without writing
- [ ] At least 5 sample answers files created and validated

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 3. Non-Interactive/CI Mode Testing`

**Status:** Not Started

---

### Unit 17.4: BATS Test Suite for Bash Install Script

**Description:** Create a BATS (Bash Automated Testing System) test suite for `scripts/install.sh`, restructuring the install script for testability and adding comprehensive tests for OS detection, architecture handling, download verification, and installation paths.

**Context:** The bash install script (`curl -sSfL https://get.gdev.dev | sh`) is the primary distribution channel for Unix systems. It handles OS detection, architecture detection, download URL construction, SHA256 verification, binary installation, and PATH configuration. Each of these steps has platform-specific edge cases (macOS `uname -m` returns `arm64` not `aarch64`, Alpine uses `musl` not `glibc`, FreeBSD has different path conventions). A script this critical needs automated testing, not just manual spot-checks.

BATS is the standard test framework for bash scripts — it is used by Homebrew, nvm, and rbenv. The `bats-support` and `bats-assert` helper libraries provide readable assertion syntax. Combined with function-level testing (source the script without executing main), BATS enables both unit tests of individual functions and integration tests of the full install flow.

**Desired Outcome:** `bats tests/install.bats` runs a comprehensive test suite covering all detection paths, error handling, and installation scenarios.

**Steps:**
1. Restructure `scripts/install.sh` for testability:
   - Extract all logic into functions: `detect_os()`, `detect_arch()`, `construct_download_url()`, `verify_checksum()`, `install_binary()`, `update_path()`, `main()`.
   - Add main guard: `if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then main "$@"; fi`.
   - This allows test files to `source install.sh` and call individual functions without triggering the full install.
2. Set up BATS as git submodules:
   - `git submodule add https://github.com/bats-core/bats-core.git tests/bats`
   - `git submodule add https://github.com/bats-core/bats-support.git tests/test_helper/bats-support`
   - `git submodule add https://github.com/bats-core/bats-assert.git tests/test_helper/bats-assert`
   - Create `tests/test_helper/common-setup.bash` loading the helpers.
3. Create `tests/install.bats` with test cases:
   - **OS detection**: mock `uname -s` to return `Linux`, `Darwin`, `MINGW64_NT-10.0`, `FreeBSD`; verify `detect_os()` returns `linux`, `darwin`, `windows`, `freebsd`.
   - **Architecture detection**: mock `uname -m` to return `x86_64`, `aarch64`, `arm64`, `armv7l`; verify `detect_arch()` returns `amd64`, `arm64`, `arm64`, `armv7`.
   - **Download URL construction**: given OS=linux, ARCH=amd64, VERSION=1.2.3; verify URL is `https://github.com/.../gdev_1.2.3_linux_amd64.tar.gz`.
   - **SHA256 verification — valid binary**: create a temp file, compute its SHA256, verify `verify_checksum()` succeeds.
   - **SHA256 verification — tampered binary**: modify the file after computing SHA256, verify `verify_checksum()` fails with clear error message.
   - **Install directory creation**: verify `install_binary()` creates `~/.local/bin` (or `/usr/local/bin` with sudo) and copies the binary with executable permissions.
   - **PATH update — bash**: verify `.bashrc` gets `export PATH="$HOME/.local/bin:$PATH"` if not already present. Verify idempotent (no duplicate entries on re-run).
   - **PATH update — zsh**: verify `.zshrc` gets the same treatment.
   - **PATH update — fish**: verify `~/.config/fish/conf.d/gdev.fish` gets `set -gx PATH $HOME/.local/bin $PATH`.
   - **Version pinning**: set `GDEV_VERSION=1.2.3` env var, verify download URL uses pinned version instead of latest.
   - **Idempotent re-install**: run install twice, verify second run detects existing binary and handles upgrade cleanly.
   - **Failure modes**: unreachable download URL (mock curl to fail), missing checksum file, insufficient permissions.
4. Mock external commands for unit tests:
   - Create `tests/mocks/` directory with mock scripts for `curl`, `uname`, `shasum`/`sha256sum`.
   - Prepend mocks directory to `PATH` in test setup.
   - Real downloads used only in release-validation CI (Tier 3), never in unit tests.
5. Add static analysis to CI:
   - `shellcheck scripts/install.sh` — lint for common bash pitfalls.
   - `shfmt -d scripts/install.sh` — verify formatting.
   - Add both as pre-commit hooks.
6. Configure JUnit XML output for CI integration:
   - `bats --formatter junit tests/install.bats > test-results/bats-install.xml`
   - Upload via `actions/upload-artifact@v4`.

**Acceptance Criteria:**
- [ ] `scripts/install.sh` restructured with function extraction and main guard
- [ ] `bats tests/install.bats` runs all test cases
- [ ] OS detection tested for Linux, macOS, Windows (MINGW), FreeBSD
- [ ] Architecture detection tested for x86_64, aarch64, arm64, armv7l
- [ ] SHA256 verification tested for both valid and tampered binaries
- [ ] PATH update tested for bash, zsh, and fish shells
- [ ] Idempotent re-install verified
- [ ] `shellcheck` and `shfmt` pass with zero warnings
- [ ] JUnit XML output produced for CI consumption
- [ ] Mock `curl` prevents real network calls in unit tests

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 4. Install Script Testing`
- `artifacts/cross-platform-testing-infrastructure-research.md § 9. Test Execution Framework`

**Status:** Not Started

---

### Unit 17.5: Pester Test Suite for PowerShell Install Script

**Description:** Create a Pester 5.x test suite for `scripts/install.ps1`, restructuring the PowerShell install script for testability and adding comprehensive tests for OS detection, architecture handling, download verification, and Windows-specific installation paths.

**Context:** The PowerShell install script (`irm https://get.gdev.dev/install.ps1 | iex`) is the primary distribution channel for Windows. It handles OS detection, architecture detection, download URL construction, SHA256 verification (via `Get-FileHash`), binary installation to `$env:LOCALAPPDATA\gdev`, and PATH modification via `[Environment]::SetEnvironmentVariable`. It also supports alternative installation via Scoop, winget, and Chocolatey when those package managers are detected.

Pester 5.x is the standard test framework for PowerShell — it ships with Windows and provides `Mock`, `Should`, `BeforeAll`/`AfterAll`, and test discovery. The `Should -Invoke` assertion verifies mock calls. Version 5.x uses `InModuleScope` and container-based test isolation.

**Desired Outcome:** `Invoke-Pester tests/install.Tests.ps1` runs a comprehensive test suite covering all detection paths, error handling, and Windows-specific installation scenarios.

**Steps:**
1. Restructure `scripts/install.ps1` for testability:
   - Extract logic into functions: `Get-OSPlatform`, `Get-Architecture`, `Get-DownloadUrl`, `Test-Checksum`, `Install-GdevBinary`, `Update-UserPath`, `Install-ViaScoop`, `Install-ViaWinget`, `Install-ViaChocolatey`.
   - Add `-SourceOnly` parameter: `param([switch]$SourceOnly)`. When set, define functions but don't execute the install flow. This allows test files to dot-source the script and call individual functions.
   - Guard main execution: `if (-not $SourceOnly) { Install-Gdev }`.
2. Create `tests/install.Tests.ps1` with test cases:
   - **OS detection**: mock `[System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform()` to return different platforms; verify `Get-OSPlatform` returns `windows`, `linux`, `darwin`.
   - **Architecture detection**: mock `[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture` to return `X64`, `Arm64`; verify `Get-Architecture` returns `amd64`, `arm64`.
   - **Download URL construction**: given platform=windows, arch=amd64, version=1.2.3; verify URL is `https://github.com/.../gdev_1.2.3_windows_amd64.zip`.
   - **SHA256 verification — valid binary**: create a temp file, compute `Get-FileHash`, verify `Test-Checksum` succeeds.
   - **SHA256 verification — tampered binary**: modify file after hash computation, verify `Test-Checksum` throws with descriptive error.
   - **Install to LOCALAPPDATA**: verify `Install-GdevBinary` creates `$env:LOCALAPPDATA\gdev\bin` and copies the binary.
   - **PATH modification**: mock `[Environment]::SetEnvironmentVariable`; verify `Update-UserPath` adds gdev's bin directory to the user-scope PATH. Verify idempotent (no duplicate entries).
   - **Scoop install path**: mock `Get-Command scoop`; verify `Install-ViaScoop` runs `scoop install gdev` when Scoop is available.
   - **winget install path**: mock `Get-Command winget`; verify `Install-ViaWinget` runs appropriate winget command.
   - **Chocolatey install path**: mock `Get-Command choco`; verify `Install-ViaChocolatey` runs `choco install gdev -y`.
   - **Failure modes**: unreachable URL (mock `Invoke-WebRequest` to throw), checksum mismatch, insufficient permissions.
3. Mock external calls for unit tests:
   - `Mock Invoke-WebRequest` for all download tests.
   - `Mock Get-FileHash` where needed for controlled checksum values.
   - `Mock [Environment]::SetEnvironmentVariable` to verify PATH changes without modifying the real environment.
   - `Mock Start-Process` for elevated permission scenarios.
4. Configure JUnit XML output for CI integration:
   - `Invoke-Pester tests/install.Tests.ps1 -OutputFormat JUnitXml -OutputFile test-results/pester-install.xml`
   - Upload via `actions/upload-artifact@v4`.
5. Add PSScriptAnalyzer as static analysis:
   - `Invoke-ScriptAnalyzer -Path scripts/install.ps1 -Severity Warning`
   - Add to CI pipeline for Windows jobs.

**Acceptance Criteria:**
- [ ] `scripts/install.ps1` restructured with function extraction and `-SourceOnly` parameter
- [ ] `Invoke-Pester tests/install.Tests.ps1` runs all test cases
- [ ] OS detection tested for Windows, Linux, macOS
- [ ] Architecture detection tested for X64, Arm64
- [ ] SHA256 verification tested for both valid and tampered binaries
- [ ] PATH modification tested with `[Environment]::SetEnvironmentVariable` mock
- [ ] Scoop, winget, and Chocolatey install paths tested
- [ ] Idempotent PATH update verified
- [ ] PSScriptAnalyzer passes with zero warnings
- [ ] JUnit XML output produced for CI consumption
- [ ] Mock `Invoke-WebRequest` prevents real network calls in unit tests

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 4. Install Script Testing`

**Status:** Not Started

---

### Unit 17.6: Test Fixture Management & Golden Files

**Description:** Establish the test fixture directory structure, golden file conventions, OS release fixtures, and shared test helper functions that all test suites depend on.

**Context:** gdev generates files across multiple formats (YAML, JSON, Nix, Markdown, shell scripts) for multiple platforms. Verifying generated output requires a disciplined approach to test fixtures (input) and golden files (expected output). Without conventions, fixtures accumulate ad hoc, golden files become stale, and platform-specific variations create maintenance burden.

The golden file pattern — storing expected output in version-controlled files and comparing against actual output — is the standard approach for generator tools. Go's `testscript` framework has built-in golden file support via `cmp`, and the `-update` flag regenerates golden files when the output changes intentionally. Platform-specific variants (Linux vs macOS vs Windows) need their own golden files since generated content varies by platform.

**Desired Outcome:** A well-organized fixture directory structure with documented conventions, golden file update workflow, OS release fixtures for all supported distros, and reusable test helper functions.

**Steps:**
1. Create directory structure:
   - `e2e/testdata/script/` — testscript `.txt` files (txtar format).
   - `e2e/testdata/answers/` — wizard answer YAML files for E2E tests.
   - `e2e/testdata/golden/` — expected output snapshots organized by test scenario.
   - `internal/*/testdata/` — per-package unit test fixtures (Go convention).
   - `test-fixtures/` — cross-cutting fixtures shared by multiple test packages.
   - `test-fixtures/os-release/` — `/etc/os-release` content for each supported distro family.
   - `test-fixtures/projects/` — sample project directories for detection testing (e.g., Go project with `go.mod`, TypeScript project with `package.json`).
2. Establish golden file conventions:
   - Suffix: `*.golden` for golden files.
   - Platform variants: `envrc-linux.golden`, `envrc-darwin.golden`, `devenv-nix-windows-wsl.golden`.
   - Update mechanism: `go test -update ./...` regenerates all golden files from current output.
   - CI verification: `go test ./...` (without `-update`) fails if golden files are stale.
   - Golden files are committed to version control — diffs in PRs show exactly what changed in generated output.
3. Create OS release fixtures using `embed.FS`:
   - `test-fixtures/os-release/debian-12` — Debian Bookworm `/etc/os-release` content.
   - `test-fixtures/os-release/ubuntu-24.04` — Ubuntu Noble.
   - `test-fixtures/os-release/fedora-41` — Fedora 41.
   - `test-fixtures/os-release/rocky-9` — Rocky Linux 9.
   - `test-fixtures/os-release/alma-9` — AlmaLinux 9.
   - `test-fixtures/os-release/arch` — Arch Linux (rolling, no version).
   - `test-fixtures/os-release/opensuse-tw` — openSUSE Tumbleweed.
   - `test-fixtures/os-release/opensuse-leap-15.6` — openSUSE Leap 15.6.
   - `test-fixtures/os-release/alpine-3.20` — Alpine Linux 3.20.
   - `test-fixtures/os-release/void` — Void Linux.
   - `test-fixtures/os-release/gentoo` — Gentoo.
   - `test-fixtures/os-release/nixos-25.11` — NixOS 25.11.
   - `test-fixtures/os-release/mint-22` — Linux Mint 22.
   - Embed via `//go:embed os-release/*` in detector test files.
4. Create state file assertion helpers in `internal/testutil/`:
   - `LoadState(t *testing.T, dir string) *GeneratedState` — load `.devinit/.gdev-init-state.yaml` from a test directory.
   - `AssertStateHasFile(t *testing.T, state *GeneratedState, path string)` — verify a file is tracked in state.
   - `AssertStateHashMatches(t *testing.T, state *GeneratedState, path string)` — verify the on-disk file's SHA256 matches the state-tracked hash.
   - `AssertStateFileOwner(t *testing.T, state *GeneratedState, path string, owner string)` — verify file ownership (for lifecycle management testing in Phase 12).
5. Create shared test helper functions in `internal/testutil/`:
   - `SetupTestProject(t *testing.T, fixtures ...string) string` — create a temp directory with specified fixture files, return path. Accepts fixture names that map to `test-fixtures/projects/` entries.
   - `AssertValidNix(t *testing.T, path string)` — run `nix-instantiate --parse` on the file, skip if nix not in PATH.
   - `AssertValidJSON(t *testing.T, path string)` — parse file as JSON, fail on syntax error.
   - `AssertValidYAML(t *testing.T, path string)` — parse file as YAML, fail on syntax error.
   - `AssertSectionPresent(t *testing.T, path string, marker string)` — verify section marker pair exists.
   - `AssertSectionAbsent(t *testing.T, path string, marker string)` — verify section marker pair absent.
6. Document conventions in `test-fixtures/README.md`:
   - Directory layout.
   - Golden file naming and update workflow.
   - How to add new OS release fixtures.
   - How to create new project fixtures.

**Acceptance Criteria:**
- [ ] Directory structure created with all specified subdirectories
- [ ] Golden file convention documented and `-update` flag wired
- [ ] OS release fixtures cover all 12+ supported distro families
- [ ] `embed.FS` loads OS release fixtures in detector tests
- [ ] State assertion helpers load and verify state files correctly
- [ ] `AssertValidNix`, `AssertValidJSON`, `AssertValidYAML` validate generated files
- [ ] `SetupTestProject` creates isolated test directories with fixtures
- [ ] `AssertSectionPresent`/`AssertSectionAbsent` verify section markers
- [ ] `go test -update ./...` regenerates all golden files without error
- [ ] Golden file staleness detected in CI (test failure when output changes)

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 7. Test Data Management`
- `artifacts/e2e-test-automation-framework-research.md § 5. State Management Testing`

**Status:** Not Started

---

### Unit 17.7: Test Reporting, Coverage & Performance Baselines

**Description:** Set up test reporting (JUnit XML with PR annotations), coverage collection (unit + E2E merged), performance baselines, and the build tag strategy that separates test tiers.

**Context:** With four test tiers (unit, integration, E2E, distro) running across 20+ matrix cells, test results need aggregation and clear reporting. Individual test failures in a matrix of 20 jobs are hard to find without PR annotations. Coverage must combine unit test coverage (trivial — `go test -cover`) with E2E coverage (non-trivial — requires the compiled binary to be instrumented). Performance regressions in `gdev doctor` or `gdev init` must be caught before they ship.

Go 1.20 introduced coverage collection from compiled binaries via the `-cover` build flag and `GOCOVERDIR` environment variable. When a `-cover`-built binary exits, it writes coverage data to `GOCOVERDIR`. This enables collecting coverage from testscript E2E runs where the binary is invoked as an external process. Merging unit and E2E coverage via `gocovmerge` gives a true picture of what code is exercised.

**Desired Outcome:** Every PR shows test results as GitHub Check annotations, coverage reports include both unit and E2E data, and performance regressions are flagged automatically.

**Steps:**
1. Set up gotestsum as the test runner:
   - Install via `go install gotest.tools/gotestsum@latest` in CI build step.
   - Replace `go test` with `gotestsum --junitfile test-results/unit.xml -- ./...` for unit tests.
   - Use `gotestsum --junitfile test-results/e2e.xml -- ./e2e/...` for E2E tests.
   - Configure `--format testdox` for local development readability.
2. Set up PR annotations:
   - Add `mikepenz/action-junit-report@v5` step after each test job to annotate PRs with test failures.
   - Configure `check_name` per job for clear identification (e.g., "Unit Tests", "E2E macOS ARM64", "E2E Fedora 41").
   - Set `fail_on_failure: true` to block merge on test failures.
3. Set up matrix result aggregation:
   - Add `test-summary/action@v2` as a final job that runs after all matrix jobs complete.
   - Downloads all `test-results/*.xml` artifacts and produces a summary table.
   - Posts summary as a PR comment or check run summary.
4. Set up Go 1.20+ coverage from compiled binary:
   - Build step: `go build -cover -o gdev-cover ./cmd/gdev`.
   - E2E test environment: `GOCOVERDIR=$WORK/.coverdir` set per testscript run.
   - Post-test: `go tool covdata textfmt -i=$GOCOVERDIR -o e2e-coverage.out` converts binary coverage data to standard Go coverage format.
5. Merge unit and E2E coverage:
   - Unit coverage: `go test -coverprofile=unit-coverage.out ./...`
   - E2E coverage: collected via `GOCOVERDIR` and converted with `go tool covdata textfmt`.
   - Merge: `go install github.com/wadey/gocovmerge@latest && gocovmerge unit-coverage.out e2e-coverage.out > merged-coverage.out`.
   - Generate HTML report: `go tool cover -html=merged-coverage.out -o coverage.html`.
   - Upload as artifact for inspection.
6. Establish performance baselines:
   - `gdev doctor` must complete in < 2 seconds on a clean system.
   - `gdev init --answers-file <path>` must complete in < 60 seconds (including detection + generation + file writes).
   - `gdev enable <tool>` and `gdev disable <tool>` must complete in < 2 seconds.
   - Write benchmarks: `func BenchmarkDoctor(b *testing.B)`, `func BenchmarkInit(b *testing.B)`, `func BenchmarkEnable(b *testing.B)`.
   - Store baseline results in `testdata/benchstat-baseline.txt`.
7. Set up benchstat for regression detection:
   - Run benchmarks in CI: `go test -bench=. -benchmem -count=5 ./... > bench-current.txt`.
   - Compare against baseline: `benchstat testdata/benchstat-baseline.txt bench-current.txt`.
   - Flag regressions exceeding 20% as warnings in PR annotations.
   - Update baseline on release branches.
8. Define build tag strategy:
   - No build tag = unit tests. Run everywhere, fast, no external dependencies.
   - `//go:build integration` = integration tests. Require external tools (nix, docker) but not a full environment.
   - `//go:build e2e` = E2E tests. Run the compiled binary in testscript. Require answers files and fixture data.
   - `//go:build distro` = distro-specific tests. Run inside Docker containers targeting specific distributions.
   - CI quick-validation runs: unit (always) + e2e (always) + integration (when tools available).
   - CI nightly runs: unit + integration + e2e + distro.
   - Local development: `go test ./...` runs only unit tests (fast feedback). `go test -tags=e2e ./e2e/...` runs E2E tests.

**Acceptance Criteria:**
- [ ] gotestsum produces JUnit XML for all test runs
- [ ] `mikepenz/action-junit-report` annotates PRs with test failures
- [ ] `test-summary/action` aggregates results across matrix jobs
- [ ] Coverage-instrumented binary writes data to `GOCOVERDIR`
- [ ] Unit and E2E coverage merged into single report
- [ ] Performance benchmarks established for `doctor`, `init`, `enable`/`disable`
- [ ] benchstat detects regressions exceeding 20% threshold
- [ ] Build tags correctly separate unit, integration, E2E, and distro tests
- [ ] `go test ./...` runs only unit tests (fast local feedback)
- [ ] `go test -tags=e2e ./e2e/...` runs E2E tests with testscript
- [ ] Coverage HTML report uploaded as CI artifact

**Research Citations:**
- `artifacts/e2e-test-automation-framework-research.md § 8. Performance Testing`
- `artifacts/e2e-test-automation-framework-research.md § 9. Test Coverage and Reporting`

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] CI pipeline runs on every PR with Tier 1 + Tier 2 matrix (5 native + 11 container jobs)
- [ ] Nightly pipeline runs full Tier 3 extended matrix (WSL2, NixOS, bare-Mac, package managers, derivative distros)
- [ ] testscript framework runs at least 3 E2E test scripts successfully
- [ ] BATS tests pass for install.sh on Ubuntu, macOS
- [ ] Pester tests pass for install.ps1 on Windows
- [ ] `gdev init --answers-file <path>` produces correct output without TUI
- [ ] Coverage collected from E2E runs via `GOCOVERDIR`
- [ ] Golden file update workflow works (`go test -update ./...`)
- [ ] Build-once-test-many artifact pipeline works across matrix jobs
- [ ] Test results visible in PR as GitHub Check annotations
