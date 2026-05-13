# E2E Test Automation Framework Research for gdev CLI

**Date**: 2026-05-12
**Scope**: End-to-end test automation frameworks and patterns for validating a Go CLI tool across multiple platforms

---

## Table of Contents

1. [Go Test Framework Patterns for CLI Tools](#1-go-test-framework-patterns-for-cli-tools)
2. [File Content Verification](#2-file-content-verification)
3. [Non-Interactive/CI Mode Testing](#3-non-interactiveci-mode-testing)
4. [Install Script Testing](#4-install-script-testing)
5. [State Management Testing](#5-state-management-testing)
6. [Multi-Platform Test Orchestration](#6-multi-platform-test-orchestration)
7. [Test Data Management](#7-test-data-management)
8. [Performance Testing](#8-performance-testing)
9. [Test Coverage and Reporting](#9-test-coverage-and-reporting)
10. [Prior Art: How Major CLI Tools Test](#10-prior-art-how-major-cli-tools-test)
11. [Recommended Test Architecture](#11-recommended-test-architecture)

---

## 1. Go Test Framework Patterns for CLI Tools

### Two Approaches: In-Process vs External Binary

**In-process testing via Cobra's `Execute()`** is what Cobra's own test suite uses. They create command trees in test code and execute them with captured buffers:

```go
func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
    buf := new(bytes.Buffer)
    root.SetOut(buf)
    root.SetErr(buf)
    root.SetArgs(args)
    _, err = root.ExecuteC()
    return buf.String(), err
}
```

Advantages: fast, no binary compilation step, easy to inject mocks. Disadvantages: doesn't test the actual compiled binary, misses build-time issues, can't test signal handling or process-level behavior.

**External binary testing via `os/exec`** is what HashiCorp uses for Terraform's e2e tests. The `e2etest` package compiles the real binary at test start, then invokes it via `exec.Command`:

```go
func TestInit(t *testing.T) {
    td := t.TempDir()
    // Copy fixture files to td...
    cmd := exec.Command(binaryPath, "init")
    cmd.Dir = td
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("unexpected error: %s\n%s", err, out)
    }
    if !strings.Contains(string(out), "Terraform has been successfully initialized") {
        t.Fatalf("unexpected output:\n%s", out)
    }
}
```

Advantages: tests the real binary exactly as users experience it, catches linking/build issues. Disadvantages: slower (must build), harder to inject test doubles.

**Recommendation for gdev**: Use both. In-process Cobra tests for unit-level command logic (flag parsing, argument validation, help text). External binary tests for E2E scenarios (init workflow, file generation, state management).

### The `testscript` Pattern (Roger Peppe)

The `testscript` package (`github.com/rogpeppe/go-internal/testscript`) is the most powerful option for CLI E2E testing in Go. It's the same engine behind the Go project's own 900+ script tests in `src/cmd/go/testdata/script/`.

**How it works**: Test scripts are `.txt` or `.txtar` files containing a domain-specific command language. The framework creates a fresh temp directory, extracts any archive files into it, and executes the script line by line.

**Built-in commands include**:
- `exec program [args...]` -- run a command, capture stdout/stderr
- `stdout pattern` / `stderr pattern` -- regex match on output
- `cmp file1 file2` -- byte-for-byte file comparison
- `exists [-readonly] file...` -- check file existence
- `env key=value` -- set environment variables
- `stdin file` -- provide stdin for next exec
- `cp`, `mkdir`, `rm`, `cd` -- filesystem operations
- `[condition]` prefix -- conditional execution (`[linux]`, `[!windows]`, `[exec:docker]`)

**Example gdev test script** (`testdata/script/init_basic.txt`):

```
# Test basic gdev init with defaults
env GDEV_NON_INTERACTIVE=1
env HOME=$WORK/home
mkdir home

exec gdev init --answers-file answers.yaml
stdout 'Project initialized successfully'
! stderr .

# Verify generated files exist
exists .devinit/devenv.nix
exists .devinit/.envrc
exists .vscode/settings.json
exists CLAUDE.md

# Verify file content
cmp .devinit/devenv.nix expected-devenv.nix
grep '"nix.enableLanguageServer"' .vscode/settings.json
grep 'language = "go"' .devinit/devenv.nix

# Verify state file
exists .devinit/.gdev-init-state.yaml
grep 'devenv.nix' .devinit/.gdev-init-state.yaml

-- answers.yaml --
project_name: myproject
language: go
enable_precommit: true
enable_claude: true

-- expected-devenv.nix --
{ pkgs, lib, config, ... }:
...
```

**Registering gdev as a testscript command**:

```go
func TestMain(m *testing.M) {
    testscript.Main(m, map[string]func(){
        "gdev": main, // Register the actual main() function
    })
}

func TestScripts(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/script",
        Setup: func(env *testscript.Env) error {
            // Set up isolated environment
            env.Setenv("XDG_CONFIG_HOME", filepath.Join(env.WorkDir, ".config"))
            env.Setenv("GDEV_NON_INTERACTIVE", "1")
            return nil
        },
        Cmds: map[string]func(*testscript.TestScript, bool, []string){
            "yaml_has": yamlHasCmd,     // Custom: check YAML key exists
            "json_path": jsonPathCmd,   // Custom: check JSON path value
            "file_hash": fileHashCmd,   // Custom: verify file hash
        },
        Condition: func(cond string) (bool, error) {
            switch cond {
            case "has_apt":
                _, err := exec.LookPath("apt-get")
                return err == nil, nil
            case "has_brew":
                _, err := exec.LookPath("brew")
                return err == nil, nil
            case "has_nix":
                _, err := exec.LookPath("nix")
                return err == nil, nil
            default:
                return false, fmt.Errorf("unknown condition: %s", cond)
            }
        },
    })
}
```

**Key advantages of testscript for gdev**:
- Each test is a self-contained text file that diffs cleanly in PRs
- Platform conditions (`[linux]`, `[darwin]`, `[windows]`) are built-in
- Custom commands let us extend the DSL for YAML/JSON/Nix assertions
- `UpdateScripts: true` enables golden-file-style auto-updating
- Archive files embedded in the script provide complete test fixtures

### Capturing and Asserting on stdout/stderr

**In-process (Cobra pattern)**:
```go
buf := new(bytes.Buffer)
cmd.SetOut(buf)
cmd.SetErr(buf)
```

**External binary (os/exec pattern)**:
```go
cmd := exec.Command(binary, args...)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
err := cmd.Run()
// Assert on stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()
```

**testscript**: stdout/stderr are automatically captured. Use `stdout 'pattern'` and `stderr 'pattern'` for regex assertions, or `cmp stdout expected.txt` for exact comparison.

### Providing stdin for Interactive Prompts

**testscript**: Use `stdin file` before `exec`:
```
stdin wizard-answers.txt
exec gdev init
```

**os/exec**:
```go
cmd := exec.Command(binary, "init")
cmd.Stdin = strings.NewReader("myproject\ngo\nyes\n")
```

**Better pattern for gdev**: Bypass stdin entirely with `--answers-file` or `GDEV_NON_INTERACTIVE=1` (see Section 3).

### Testing Exit Codes

**testscript**: `! exec gdev badcommand` expects non-zero exit. The `exec` command captures the exit status automatically.

**os/exec**:
```go
err := cmd.Run()
if exitErr, ok := err.(*exec.ExitError); ok {
    assert.Equal(t, 1, exitErr.ExitCode())
}
```

### Tests Requiring Root/Admin

For package installation tests (e.g., `gdev setup` running `apt-get install`):

1. **Skip in normal test runs**: Use build tags or conditions
   ```
   [exec:sudo] [linux] exec sudo gdev setup
   ```
2. **Run in CI containers**: Docker containers where the test user is root
3. **Mock the package manager**: For unit tests, inject a mock `PackageManager` interface
4. **Dedicated privileged CI job**: Separate GitHub Actions job with elevated permissions in a throwaway container

---

## 2. File Content Verification

### Golden File Testing (Snapshot Testing)

Golden file testing stores expected output in version-controlled files and compares actual output byte-for-byte. When the output legitimately changes, update the golden files with a flag.

**Go standard pattern**:
```go
var update = flag.Bool("update", false, "update golden files")

func TestGeneratedFiles(t *testing.T) {
    td := t.TempDir()
    runGdevInit(td, defaultAnswers)

    goldenDir := "testdata/golden/basic-init"
    for _, file := range []string{"devenv.nix", ".envrc", "settings.json"} {
        actual, err := os.ReadFile(filepath.Join(td, ".devinit", file))
        require.NoError(t, err)

        goldenPath := filepath.Join(goldenDir, file+".golden")
        if *update {
            os.MkdirAll(goldenDir, 0755)
            os.WriteFile(goldenPath, actual, 0644)
            continue
        }

        expected, err := os.ReadFile(goldenPath)
        require.NoError(t, err)
        if diff := cmp.Diff(string(expected), string(actual)); diff != "" {
            t.Errorf("%s mismatch (-want +got):\n%s", file, diff)
        }
    }
}
```

**Libraries**:
- `sebdah/goldie` -- Mature golden file library with auto-update, diff output, and `testdata/` conventions
- `bradleyjkemp/cupaloy` -- Snapshot testing with automatic file naming based on test name
- `testscript` with `UpdateScripts: true` -- Updates txtar archives on `cmp` failures

**Handling platform-specific differences**: Use `cmpenv` in testscript to substitute `$GOOS` and `$GOARCH` in expected output, or maintain separate golden files per platform:
```
testdata/golden/
  basic-init/
    devenv.nix.golden          # Universal
    settings.json.golden       # Universal
    envrc-linux.golden         # Linux-specific
    envrc-darwin.golden        # macOS-specific
```

### Structured Content Testing

For files with known structure, parse and assert on specific elements rather than full-file comparison:

**JSON (settings.json)**:
```go
func TestSettingsJSON(t *testing.T) {
    data, _ := os.ReadFile(filepath.Join(td, ".vscode/settings.json"))
    var settings map[string]interface{}
    require.NoError(t, json.Unmarshal(data, &settings))

    assert.Equal(t, true, settings["nix.enableLanguageServer"])
    assert.Equal(t, "nil", settings["nix.serverPath"])

    // Nested path assertion
    goSettings, ok := settings["go.toolsManagement.autoUpdate"]
    assert.True(t, ok)
    assert.Equal(t, "off", goSettings)
}
```

**YAML (.pre-commit-config.yaml, state files)**:
```go
func TestPreCommitConfig(t *testing.T) {
    data, _ := os.ReadFile(filepath.Join(td, ".pre-commit-config.yaml"))
    var config struct {
        Repos []struct {
            Repo  string `yaml:"repo"`
            Hooks []struct {
                ID string `yaml:"id"`
            } `yaml:"hooks"`
        } `yaml:"repos"`
    }
    require.NoError(t, yaml.Unmarshal(data, &config))
    assert.True(t, len(config.Repos) > 0)
    // Check for specific hooks...
}
```

**Nix files (devenv.nix)**: No robust Go Nix parser exists. Use pattern matching:
```go
func TestDevenvNix(t *testing.T) {
    content, _ := os.ReadFile(filepath.Join(td, ".devinit/devenv.nix"))
    s := string(content)

    // Section presence
    assert.Contains(t, s, "languages.go.enable = true")
    assert.Contains(t, s, "pre-commit.hooks")

    // Regex for structured patterns
    assert.Regexp(t, `packages\s*=\s*\[`, s)
    assert.Regexp(t, `pkgs\.go_\d+`, s)
}
```

**Custom testscript commands for structured assertions**:
```go
// yaml_has checks that a YAML file contains a key path
func yamlHasCmd(ts *testscript.TestScript, neg bool, args []string) {
    // args: [file, key.path, optional-expected-value]
    file := ts.MkAbs(args[0])
    data, err := os.ReadFile(file)
    ts.Check(err)

    var m map[string]interface{}
    ts.Check(yaml.Unmarshal(data, &m))

    val := navigateYAMLPath(m, args[1])
    if neg {
        if val != nil { ts.Fatalf("key %s exists but should not", args[1]) }
    } else {
        if val == nil { ts.Fatalf("key %s not found", args[1]) }
        if len(args) > 2 && fmt.Sprint(val) != args[2] {
            ts.Fatalf("key %s: got %v, want %s", args[1], val, args[2])
        }
    }
}
```

Usage in test scripts:
```
yaml_has .devinit/.gdev-init-state.yaml files.devenv.nix
json_path .vscode/settings.json nix.enableLanguageServer true
grep 'languages.go.enable' .devinit/devenv.nix
```

### Recommendation

Use a **hybrid approach**:
- **Golden files** for complex generated files that change infrequently (devenv.nix templates, CI workflow files) -- catch unintended regressions
- **Structured assertions** for files where specific values matter (settings.json keys, YAML state entries) -- survive template refactoring
- **Regex/contains** for Nix files where parsing is impractical

---

## 3. Non-Interactive/CI Mode Testing

### charmbracelet/huh Testing Support

huh is built on top of bubbletea. The testing ecosystem has multiple layers:

**teatest** (`github.com/charmbracelet/x/exp/teatest`): Wraps `tea.Program` in a controlled headless environment:

```go
func TestWizard(t *testing.T) {
    m := NewWizardModel(defaultConfig)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Wait for first prompt
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Project name"))
    }, teatest.WithCheckInterval(100*time.Millisecond),
       teatest.WithDuration(3*time.Second))

    // Type a project name
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("myproject")})
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

    // Wait for completion
    out, err := io.ReadAll(tm.FinalOutput(t))
    require.NoError(t, err)
    teatest.RequireEqualOutput(t, out) // Golden file comparison
}
```

**Caveats**: teatest is experimental (`x/exp` namespace), API may change. Golden files require consistent color profiles:
```go
func init() {
    lipgloss.SetColorProfile(termenv.Ascii) // Strip ANSI for deterministic output
}
```

Add to `.gitattributes`:
```
*.golden -text
```

**Emerging approaches**: The charm team is developing `x/vt` and `x/xpty` for headless virtual terminals that capture screen content at any point in time, enabling more integrated terminal testing.

### Recommended Non-Interactive Patterns for gdev

**Pattern 1: `--non-interactive` flag** (highest priority)

Skip the TUI wizard entirely. Use defaults or pre-configured values:

```go
// In the init command
if nonInteractive || os.Getenv("GDEV_NON_INTERACTIVE") == "1" {
    // Use defaults from answers file or built-in defaults
    config = loadDefaults(answersFile)
} else {
    // Run huh wizard
    config = runWizard()
}
```

**Pattern 2: `--answers-file` flag** (for parameterized testing)

Read wizard answers from a YAML/JSON file, bypassing TUI but exercising the same config logic:

```go
// answers.yaml
project_name: myproject
language: go
tools:
  - precommit
  - claude
  - devcontainer
enable_security: true
```

This is the single most important testing enabler. It lets E2E tests exercise different configuration combinations without TUI interaction.

**Pattern 3: Environment variable override**

`GDEV_NON_INTERACTIVE=1` forces non-interactive mode. Detected automatically in CI (check `CI=true`, `GITHUB_ACTIONS=true`, or `GDEV_NON_INTERACTIVE=1`):

```go
func isInteractive() bool {
    if os.Getenv("GDEV_NON_INTERACTIVE") == "1" { return false }
    if os.Getenv("CI") == "true" { return false }
    if !term.IsTerminal(int(os.Stdin.Fd())) { return false }
    return true
}
```

**Pattern 4: teatest for wizard-specific tests** (secondary)

Use teatest only for testing the wizard UI itself (navigation, validation, display). Don't route E2E tests through the TUI.

### Testing Strategy by Layer

| Layer | How to Test | Interactive? |
|-------|------------|-------------|
| Wizard UI rendering | teatest + golden files | Simulated |
| Wizard data flow | Unit test model directly | No |
| Config -> file generation | `--answers-file` + file assertions | No |
| Full E2E workflow | `--non-interactive` + testscript | No |
| TUI edge cases | teatest with specific key sequences | Simulated |

---

## 4. Install Script Testing

### BATS (Bash Automated Testing System)

BATS is the standard framework for testing bash scripts. It's TAP-compliant and uses bash's `errexit` -- every statement is an assertion.

**Test file structure** (`tests/install.bats`):

```bash
#!/usr/bin/env bats

setup() {
    # Create temp directory for each test
    TEST_DIR="$(mktemp -d)"
    export HOME="$TEST_DIR/home"
    mkdir -p "$HOME"

    # Source the script's functions without executing
    # (requires the script to be structured with a main guard)
    source scripts/install.sh --source-only 2>/dev/null || true
}

teardown() {
    rm -rf "$TEST_DIR"
}

@test "detect_os returns linux on Linux" {
    # Mock uname
    function uname() { echo "Linux"; }
    export -f uname

    run detect_os
    [ "$status" -eq 0 ]
    [ "$output" = "linux" ]
}

@test "detect_arch maps x86_64 to amd64" {
    function uname() {
        if [ "$1" = "-m" ]; then echo "x86_64"; fi
    }
    export -f uname

    run detect_arch
    [ "$status" -eq 0 ]
    [ "$output" = "amd64" ]
}

@test "download_binary fails on HTTP error" {
    # Mock curl to simulate failure
    function curl() { return 22; }
    export -f curl

    run download_binary "https://example.com/gdev" "$TEST_DIR/gdev"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Download failed"* ]]
}

@test "verify_sha256 detects tampered binary" {
    echo "good content" > "$TEST_DIR/binary"
    echo "deadbeef  binary" > "$TEST_DIR/checksums.txt"

    run verify_sha256 "$TEST_DIR/binary" "$TEST_DIR/checksums.txt"
    [ "$status" -ne 0 ]
    [[ "$output" == *"checksum mismatch"* ]]
}

@test "install creates binary in target directory" {
    # Mock download to create a fake binary
    function curl() {
        echo "#!/bin/sh" > "$2"
        chmod +x "$2"
    }
    export -f curl

    export GDEV_INSTALL_DIR="$TEST_DIR/bin"
    run install_gdev
    [ "$status" -eq 0 ]
    [ -x "$TEST_DIR/bin/gdev" ]
}

@test "shell detection finds zsh config" {
    export SHELL="/bin/zsh"
    touch "$HOME/.zshrc"

    run detect_shell_config
    [ "$status" -eq 0 ]
    [[ "$output" == *".zshrc"* ]]
}
```

**Structuring install.sh for testability**: Move all logic into functions, use a main guard:

```bash
#!/bin/bash
# ... function definitions ...

detect_os() { ... }
detect_arch() { ... }
download_binary() { ... }
verify_sha256() { ... }
install_gdev() { ... }

# Main guard: only execute when run directly, not when sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
```

**Running BATS**:
```bash
# Install
npm install -g bats    # or: brew install bats-core

# With helper libraries
git submodule add https://github.com/bats-core/bats-support test/bats-support
git submodule add https://github.com/bats-core/bats-assert test/bats-assert

# Run tests
bats tests/install.bats

# JUnit output for CI
bats --formatter junit tests/install.bats > test-results.xml
```

**Testing across shells**: BATS itself runs in bash, but you can test that the script parses correctly in other shells:

```bash
@test "install.sh is valid dash syntax" {
    run dash -n scripts/install.sh
    [ "$status" -eq 0 ]
}

@test "install.sh is valid zsh syntax" {
    skip_if_no_command zsh
    run zsh -n scripts/install.sh
    [ "$status" -eq 0 ]
}
```

**Static analysis**: ShellCheck + shfmt should run as pre-commit hooks and in CI:
```bash
shellcheck scripts/install.sh
shfmt -d scripts/install.sh
```

### Pester (PowerShell)

For `scripts/install.ps1`:

```powershell
# tests/Install.Tests.ps1
Describe "gdev install script" {
    BeforeAll {
        $TestDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "gdev-test-$(Get-Random)")
        . "$PSScriptRoot/../scripts/install.ps1" -SourceOnly
    }

    AfterAll {
        Remove-Item -Recurse -Force $TestDir
    }

    Context "OS Detection" {
        It "detects Windows correctly" {
            $result = Get-TargetOS
            $result | Should -Be "windows"
        }

        It "detects architecture correctly" {
            $result = Get-TargetArch
            $result | Should -BeIn @("amd64", "arm64")
        }
    }

    Context "Download" {
        It "constructs correct download URL" {
            $url = Get-DownloadURL -Version "1.0.0" -OS "windows" -Arch "amd64"
            $url | Should -Match "gdev_1.0.0_windows_amd64.zip"
        }

        It "fails gracefully on network error" {
            Mock Invoke-WebRequest { throw "Network error" }
            { Download-Binary -URL "https://bad.url" -OutFile "$TestDir/gdev.exe" } |
                Should -Throw
        }
    }

    Context "PATH modification" {
        It "adds install directory to user PATH" {
            $installDir = "$TestDir\bin"
            Add-ToPath -Directory $installDir
            $env:PATH | Should -Contain $installDir
        }
    }
}
```

**Running Pester**:
```powershell
Install-Module -Name Pester -Force -SkipPublisherCheck
Invoke-Pester -Path tests/Install.Tests.ps1 -OutputFormat JUnitXml -OutputFile test-results.xml
```

### How rustup and mise Test Install Scripts

**rustup**: Their `rustup-init.sh` is designed for minimal environments (dash, bash, zsh, ksh). Testing is primarily through CI matrix builds on real platforms rather than unit testing the script itself. The `-y` flag enables non-interactive mode. The script architecture separates download logic from the actual Rust binary (`rustup-init`) which handles installation.

**mise**: Uses a dedicated `e2e/` directory with bash-based tests and an `assert.sh` library. Windows testing uses separate PowerShell scripts in `e2e-win/`. Environment isolation via `setup_isolated_env()` prevents contamination.

### Mocking `curl` in BATS Tests

```bash
# Create a mock curl that serves local files
setup() {
    MOCK_SERVER_DIR="$TEST_DIR/mock-server"
    mkdir -p "$MOCK_SERVER_DIR"

    # Create mock binary
    echo '#!/bin/sh' > "$MOCK_SERVER_DIR/gdev_linux_amd64"
    echo 'echo "gdev v1.0.0"' >> "$MOCK_SERVER_DIR/gdev_linux_amd64"
    chmod +x "$MOCK_SERVER_DIR/gdev_linux_amd64"

    # Create checksums
    sha256sum "$MOCK_SERVER_DIR/gdev_linux_amd64" > "$MOCK_SERVER_DIR/checksums.txt"

    # Override curl
    function curl() {
        local url="${@: -1}"  # Last argument is URL
        local outfile=""
        # Parse -o flag
        while [[ "$#" -gt 0 ]]; do
            case "$1" in
                -o) outfile="$2"; shift 2;;
                *) shift;;
            esac
        done
        local filename=$(basename "$url")
        if [ -f "$MOCK_SERVER_DIR/$filename" ]; then
            if [ -n "$outfile" ]; then
                cp "$MOCK_SERVER_DIR/$filename" "$outfile"
            else
                cat "$MOCK_SERVER_DIR/$filename"
            fi
            return 0
        fi
        return 22  # HTTP error
    }
    export -f curl
}
```

---

## 5. State Management Testing

### State File Structure

gdev tracks state in two YAML files:
- `.devinit/.gdev-init-state.yaml` -- file hashes, ownership, timestamps
- `.devinit/.gdev-init-answers.yaml` -- wizard answers for re-runs

### State Assertion Helpers

**Go helper for state verification**:

```go
type StateFile struct {
    Version   string                `yaml:"version"`
    Files     map[string]FileState  `yaml:"files"`
    CreatedAt time.Time             `yaml:"created_at"`
    UpdatedAt time.Time             `yaml:"updated_at"`
}

type FileState struct {
    Hash      string `yaml:"hash"`
    Owner     string `yaml:"owner"` // "gdev" or "user"
    CreatedAt string `yaml:"created_at"`
    Template  string `yaml:"template,omitempty"`
}

func loadState(t *testing.T, dir string) StateFile {
    t.Helper()
    data, err := os.ReadFile(filepath.Join(dir, ".devinit/.gdev-init-state.yaml"))
    require.NoError(t, err)
    var state StateFile
    require.NoError(t, yaml.Unmarshal(data, &state))
    return state
}

func assertStateHasFile(t *testing.T, state StateFile, filename string) {
    t.Helper()
    _, ok := state.Files[filename]
    assert.True(t, ok, "state should track file: %s", filename)
}

func assertStateHashMatches(t *testing.T, state StateFile, dir, filename string) {
    t.Helper()
    entry, ok := state.Files[filename]
    require.True(t, ok)
    content, err := os.ReadFile(filepath.Join(dir, filename))
    require.NoError(t, err)
    actualHash := sha256hex(content)
    assert.Equal(t, entry.Hash, actualHash,
        "hash mismatch for %s: state says %s, actual is %s", filename, entry.Hash, actualHash)
}
```

### State Consistency Tests

```go
func TestStateConsistency(t *testing.T) {
    td := t.TempDir()
    runGdevInit(td, defaultAnswers)

    state := loadState(t, td)

    // Every generated file has a state entry
    generatedFiles := findGeneratedFiles(td)
    for _, f := range generatedFiles {
        assertStateHasFile(t, state, f)
    }

    // Every state entry has a corresponding file
    for filename := range state.Files {
        path := filepath.Join(td, filename)
        assert.FileExists(t, path, "state references %s but file doesn't exist", filename)
    }

    // All hashes match actual file content
    for filename := range state.Files {
        assertStateHashMatches(t, state, td, filename)
    }
}
```

### State Round-Trip Tests

```go
func TestStateRoundTrip(t *testing.T) {
    td := t.TempDir()

    // Initial init
    runGdevInit(td, defaultAnswers)
    state1 := loadState(t, td)

    // Modify a user file (simulate manual edit)
    settingsPath := filepath.Join(td, ".vscode/settings.json")
    appendToFile(t, settingsPath, "\n// user modification\n")

    // Re-run init (should detect modification)
    runGdevInit(td, defaultAnswers)
    state2 := loadState(t, td)

    // User-modified file should be preserved (owner changed to "user")
    assert.Equal(t, "user", state2.Files[".vscode/settings.json"].Owner)

    // Unmodified files should retain original hashes
    for name, entry := range state1.Files {
        if name == ".vscode/settings.json" { continue }
        assert.Equal(t, entry.Hash, state2.Files[name].Hash,
            "unmodified file %s hash should be unchanged", name)
    }
}
```

### Testscript Commands for State Testing

```
# Test state after init
exec gdev init --answers-file answers.yaml
yaml_has .devinit/.gdev-init-state.yaml files.devenv.nix
yaml_has .devinit/.gdev-init-state.yaml version

# Modify a file and verify detection
cp modified-settings.json .vscode/settings.json
exec gdev status
stdout 'modified.*settings.json'

# Re-init preserves user changes
exec gdev init --answers-file answers.yaml
yaml_has .devinit/.gdev-init-state.yaml files.settings.json.owner user
```

---

## 6. Multi-Platform Test Orchestration

### GitHub Actions Matrix Strategy

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./...

  e2e-tests:
    needs: unit-tests
    strategy:
      fail-fast: false
      matrix:
        include:
          # macOS
          - os: macos-latest
            name: macOS-arm64
            tags: e2e
          - os: macos-13
            name: macOS-amd64
            tags: e2e

          # Linux native runners
          - os: ubuntu-latest
            name: Ubuntu-24.04
            tags: e2e
          - os: ubuntu-22.04
            name: Ubuntu-22.04
            tags: e2e

          # Windows
          - os: windows-latest
            name: Windows-amd64
            tags: e2e,!unix

    runs-on: ${{ matrix.os }}
    name: E2E (${{ matrix.name }})
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }

      - name: Build gdev
        run: go build -cover -o gdev${{ runner.os == 'Windows' && '.exe' || '' }} .

      - name: Run E2E tests
        env:
          GDEV_BINARY: ${{ github.workspace }}/gdev${{ runner.os == 'Windows' && '.exe' || '' }}
          GOCOVERDIR: ${{ github.workspace }}/coverage
          GDEV_NON_INTERACTIVE: "1"
        run: |
          mkdir -p coverage
          go test -tags=${{ matrix.tags }} -v -timeout 10m ./e2e/...

      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage-${{ matrix.name }}
          path: coverage/

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results-${{ matrix.name }}
          path: test-results.xml

  # Linux distro testing via Docker containers
  distro-tests:
    needs: unit-tests
    strategy:
      fail-fast: false
      matrix:
        distro:
          - { name: "Fedora-40", image: "fedora:40" }
          - { name: "Debian-12", image: "debian:12" }
          - { name: "Arch", image: "archlinux:latest" }
          - { name: "Alpine-3.20", image: "alpine:3.20" }
          - { name: "RHEL-9", image: "almalinux:9" }
          - { name: "openSUSE-15", image: "opensuse/leap:15" }
          - { name: "NixOS", image: "nixos/nix:latest" }
    runs-on: ubuntu-latest
    name: Distro (${{ matrix.distro.name }})
    container:
      image: ${{ matrix.distro.image }}
    steps:
      - uses: actions/checkout@v4

      - name: Install Go
        run: |
          # Distro-specific Go installation
          # (or use pre-built gdev binary)

      - name: Run distro-specific tests
        env:
          GDEV_NON_INTERACTIVE: "1"
        run: go test -tags=e2e,distro -v -timeout 15m ./e2e/distro/...

  # WSL2 testing
  wsl-tests:
    needs: unit-tests
    runs-on: windows-latest
    name: WSL2 (Ubuntu)
    steps:
      - uses: actions/checkout@v4
      - uses: Vampire/setup-wsl@v3
        with:
          distribution: Ubuntu-24.04
      - name: Run WSL tests
        shell: wsl-bash {0}
        run: |
          # Install Go in WSL, build and test
          go test -tags=e2e,wsl -v ./e2e/...

  # Aggregate coverage from all platforms
  coverage-report:
    needs: [e2e-tests, distro-tests]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/download-artifact@v4
        with:
          pattern: coverage-*
          merge-multiple: true
          path: all-coverage

      - name: Merge coverage
        run: |
          go tool covdata merge -i=all-coverage -o=merged
          go tool covdata textfmt -i=merged -o=coverage.txt
          go tool cover -func=coverage.txt

      - name: Upload to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: coverage.txt
```

### Test Tagging Strategy

Use Go build constraints to separate test tiers:

```go
//go:build e2e
// +build e2e

package e2e

// E2E tests that need the compiled binary
```

```go
//go:build integration
// +build integration

package integration

// Integration tests that need real tools but not full binary
```

```go
//go:build distro
// +build distro

package distro

// Tests that need specific package managers (run in containers)
```

```go
//go:build !windows

package e2e

// Unix-only tests
```

**Running specific tiers**:
```bash
go test ./...                          # Unit tests only (no tags)
go test -tags=integration ./...        # Unit + integration
go test -tags=e2e ./...                # Unit + E2E
go test -tags=e2e,distro ./...         # All tests
```

### Sharing Fixtures Across Platforms with Platform-Specific Assertions

```go
func TestInitGeneratesCorrectEnvrc(t *testing.T) {
    td := t.TempDir()
    runGdevInit(td, defaultAnswers)

    content, err := os.ReadFile(filepath.Join(td, ".envrc"))
    require.NoError(t, err)

    // Universal assertions
    assert.Contains(t, string(content), "use devenv")

    // Platform-specific assertions
    if runtime.GOOS == "darwin" {
        assert.Contains(t, string(content), "export HOMEBREW_PREFIX")
    }
    if runtime.GOOS == "linux" {
        assert.NotContains(t, string(content), "HOMEBREW_PREFIX")
    }
}
```

In testscript, use built-in conditions:
```
exec gdev init --answers-file answers.yaml
exists .envrc
grep 'use devenv' .envrc
[darwin] grep 'HOMEBREW_PREFIX' .envrc
[linux] ! grep 'HOMEBREW_PREFIX' .envrc
```

### Handling Flaky Tests

1. **Retry on CI**: Use `gotestsum --rerun-fails=2` to automatically retry failed tests
2. **Network isolation**: Mock HTTP endpoints for tests that don't need real network
3. **Timeouts**: Set explicit timeouts per test, not just per job
4. **Deterministic ordering**: Use `go test -shuffle=on` to detect order-dependent tests early
5. **Skip when dependencies unavailable**:
   ```go
   func TestWithBrew(t *testing.T) {
       if _, err := exec.LookPath("brew"); err != nil {
           t.Skip("brew not available")
       }
       // ...
   }
   ```

---

## 7. Test Data Management

### Directory Structure

```
gdev/
  cmd/
    init/
      init.go
      init_test.go              # Unit tests (in-process Cobra)
  internal/
    detector/
      detector.go
      detector_test.go          # Unit tests
      testdata/
        os-release/             # OS detection fixtures
          ubuntu-24.04
          fedora-40
          nixos-24.05
          arch
        package-managers/
          apt-available
          brew-available
  e2e/
    testdata/
      script/                   # testscript txtar files
        init-basic.txt
        init-go-project.txt
        init-typescript-project.txt
        enable-disable.txt
        doctor.txt
        setup-prereqs.txt
        state-management.txt
      answers/                  # Wizard answer files
        go-defaults.yaml
        typescript-full.yaml
        minimal.yaml
      golden/                   # Expected output snapshots
        go-defaults/
          devenv.nix.golden
          settings.json.golden
          envrc.golden
        typescript-full/
          devenv.nix.golden
          settings.json.golden
    e2e_test.go                 # Test runner
    helpers_test.go             # Shared test utilities
  e2e/distro/
    distro_test.go              # Distro-specific tests (build tag: distro)
  tests/
    install.bats                # Bash install script tests
    Install.Tests.ps1           # PowerShell install script tests
    bats-support/               # BATS helper (git submodule)
    bats-assert/                # BATS helper (git submodule)
```

### Embedded Test Data via `embed.FS`

Use `embed.FS` for fixtures that should travel with the test binary (useful for cross-compilation and running tests on remote machines):

```go
package detector

import (
    "embed"
    "testing"
)

//go:embed testdata/os-release/*
var osReleaseFixtures embed.FS

func TestDetectDistro(t *testing.T) {
    tests := []struct {
        fixture  string
        wantName string
        wantVer  string
    }{
        {"ubuntu-24.04", "ubuntu", "24.04"},
        {"fedora-40", "fedora", "40"},
        {"nixos-24.05", "nixos", "24.05"},
        {"arch", "arch", ""},
    }

    for _, tt := range tests {
        t.Run(tt.fixture, func(t *testing.T) {
            data, err := osReleaseFixtures.ReadFile("testdata/os-release/" + tt.fixture)
            require.NoError(t, err)

            name, ver := parseOSRelease(string(data))
            assert.Equal(t, tt.wantName, name)
            assert.Equal(t, tt.wantVer, ver)
        })
    }
}
```

### When to Use embed.FS vs External Files

| Criterion | `embed.FS` | External `testdata/` |
|-----------|-----------|---------------------|
| Small, static fixtures | Yes | Yes |
| Golden files (need updating) | No | Yes |
| txtar scripts | No | Yes (testscript reads from disk) |
| OS-release fixtures | Yes | Also fine |
| Large fixture trees | No (bloats binary) | Yes |
| Cross-compiled test binaries | Yes (self-contained) | No (files not included) |

**Recommendation**: Use `embed.FS` for small, read-only fixtures (os-release files, sample configs). Use external `testdata/` for everything that needs updating (golden files, testscript archives).

### Versioning Expected Output

- Commit golden files to git. They form part of the test contract.
- Update with `go test -update ./...` when output legitimately changes.
- PR reviews should scrutinize golden file diffs -- they show exactly what changed in generated output.
- For OS-specific golden files, use naming conventions: `devenv.nix.golden` (universal), `envrc-linux.golden`, `envrc-darwin.golden`.

---

## 8. Performance Testing

### Wall-Clock Assertions in E2E Tests

For `gdev init` (target: <60s) and `gdev doctor` (target: <2s):

```go
func TestInitPerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping performance test in short mode")
    }
    td := t.TempDir()

    start := time.Now()
    runGdevInit(td, defaultAnswers)
    elapsed := time.Since(start)

    // Hard failure threshold
    assert.Less(t, elapsed, 60*time.Second,
        "gdev init took %s, exceeding 60s limit", elapsed)

    // Warning threshold (for trending)
    if elapsed > 30*time.Second {
        t.Logf("WARNING: gdev init took %s (>30s warning threshold)", elapsed)
    }
}

func TestDoctorPerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping performance test in short mode")
    }

    start := time.Now()
    runGdevDoctor(t)
    elapsed := time.Since(start)

    assert.Less(t, elapsed, 2*time.Second,
        "gdev doctor took %s, exceeding 2s limit", elapsed)
}
```

### Go Benchmarks for Internal Functions

```go
func BenchmarkDetectOS(b *testing.B) {
    for i := 0; i < b.N; i++ {
        DetectOS()
    }
}

func BenchmarkParseOSRelease(b *testing.B) {
    data, _ := os.ReadFile("testdata/os-release/ubuntu-24.04")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parseOSRelease(string(data))
    }
}

func BenchmarkGenerateDevenvNix(b *testing.B) {
    config := defaultConfig()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        GenerateDevenvNix(config)
    }
}
```

### CI Performance Regression Detection

Use `benchstat` to compare benchmark results between commits:

```yaml
# .github/workflows/benchmarks.yml
name: Benchmarks
on:
  pull_request:
    branches: [main]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }

      - name: Install benchstat
        run: go install golang.org/x/perf/cmd/benchstat@latest

      - name: Run benchmarks (PR)
        run: go test -bench=. -count=5 -run=^$ ./... > bench-pr.txt

      - name: Checkout base branch
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.pull_request.base.sha }}
          path: base

      - name: Run benchmarks (base)
        working-directory: base
        run: go test -bench=. -count=5 -run=^$ ./... > ../bench-base.txt

      - name: Compare benchmarks
        run: |
          benchstat bench-base.txt bench-pr.txt | tee benchmark-comparison.txt
          # Fail if any benchmark regressed by more than 20%
          if grep -E '\+[2-9][0-9]\.[0-9]+%|+[0-9]{3,}' benchmark-comparison.txt; then
            echo "::error::Significant performance regression detected"
            exit 1
          fi

      - name: Comment on PR
        if: always()
        uses: marocchino/sticky-pull-request-comment@v2
        with:
          path: benchmark-comparison.txt
```

### Alternative: gobenchdata

The `bobheadxi/gobenchdata` action provides a more integrated approach: it runs benchmarks, publishes to an interactive web dashboard, and checks for regressions in PRs. Better for ongoing trending, but heavier to set up.

---

## 9. Test Coverage and Reporting

### E2E Coverage with Go 1.20+

Since Go 1.20, you can collect coverage from compiled binaries -- this is critical for gdev since E2E tests run the actual binary.

**Build with coverage instrumentation**:
```bash
go build -cover -o gdev .
```

**Run E2E tests with coverage collection**:
```bash
mkdir -p coverdata
GOCOVERDIR=coverdata ./gdev init --answers-file test-answers.yaml
GOCOVERDIR=coverdata ./gdev doctor
GOCOVERDIR=coverdata ./gdev enable precommit
GOCOVERDIR=coverdata ./gdev status
```

**Integrate with testscript**: Set `GOCOVERDIR` in the Setup function:
```go
Setup: func(env *testscript.Env) error {
    coverDir := filepath.Join(env.WorkDir, "coverdata")
    os.MkdirAll(coverDir, 0755)
    env.Setenv("GOCOVERDIR", coverDir)
    return nil
},
```

**Merge and report**:
```bash
# Merge from all test runs
go tool covdata merge -i=coverdata -o=merged

# Generate text profile (compatible with codecov, coveralls, etc.)
go tool covdata textfmt -i=merged -o=coverage.txt

# View function-level coverage
go tool cover -func=coverage.txt

# Generate HTML report
go tool cover -html=coverage.txt -o=coverage.html
```

**Combine unit test and E2E coverage**:
```bash
# Unit test coverage
go test -coverprofile=unit-coverage.txt ./...

# E2E coverage (from instrumented binary runs)
go tool covdata textfmt -i=e2e-coverdata -o=e2e-coverage.txt

# Merge (use gocovmerge or similar)
go install github.com/wadey/gocovmerge@latest
gocovmerge unit-coverage.txt e2e-coverage.txt > combined-coverage.txt
```

**Limitations**:
- Coverage data is only written when the program exits normally (not on panics/crashes)
- Binary is slightly larger and slower when instrumented
- Only current module packages are instrumented by default (use `-coverpkg` to include more)

### Test Result Aggregation

**gotestsum** is the standard tool for Go test output formatting:

```bash
# Install
go install gotest.tools/gotestsum@latest

# Run with JUnit output
gotestsum --junitfile test-results.xml --format testname -- -tags=e2e ./...

# With automatic retry of failed tests
gotestsum --rerun-fails=2 --junitfile test-results.xml -- -tags=e2e ./...
```

**GitHub Actions integration**:
```yaml
- name: Run tests
  run: |
    gotestsum --junitfile test-results.xml --format testname -- \
      -tags=e2e -v -timeout 10m ./e2e/...

- name: Publish test results
  if: always()
  uses: mikepenz/action-junit-report@v4
  with:
    report_paths: test-results.xml
    check_name: "E2E Tests (${{ matrix.name }})"
    include_passed: true
```

**Visualizing results across the OS matrix**: The `mikepenz/action-junit-report` action creates GitHub Check annotations on the PR. Each matrix job creates its own check with the job name. The `test-summary/action` can aggregate multiple JUnit files into a single summary table:

```yaml
- name: Test Summary
  if: always()
  uses: test-summary/action@v2
  with:
    paths: "**/test-results.xml"
```

---

## 10. Prior Art: How Major CLI Tools Test

### Terraform (Go, HashiCorp)

**Framework**: Go standard `testing` + custom helpers. No external test framework.

**Architecture**: Dedicated `command/e2etest/` package that compiles the real Terraform binary at test start. Tests use `exec.Command` to invoke Terraform with fixture directories from `testdata/`. Version-specific skipping with `semver` comparisons.

**Multi-platform**: GitHub Actions matrix across macOS, Linux, Windows. Container-based testing for provider acceptance tests.

**Key pattern**: `runTest()` helper that sets up temp dirs, copies fixtures, and provides a `*tfexec.Terraform` handle for command execution. Output comparison uses `go-cmp` diffs.

**Takeaway for gdev**: The "compile binary, exec it in temp dirs" pattern is battle-tested at scale. The fixture-per-scenario approach maps well to gdev's different project types.

### GoReleaser (Go)

**Framework**: Go standard `testing` + Testify (`require` package) + golden file testing.

**Architecture**: Table-driven tests with map-based test cases. Uses `golden.RequireEqualJSON()` for snapshot assertions on generated config. Tests the actual main package via in-process execution.

**Multi-platform**: GitHub Actions matrix for build verification. Uses their own tool (GoReleaser) for cross-compilation testing.

**Takeaway for gdev**: GoReleaser's golden file approach for generated configs is directly applicable to testing gdev's generated files. Their table-driven test structure is clean and maintainable.

### Cobra (Go, spf13)

**Framework**: Go standard `testing` only. No external libraries.

**Testing approach**: In-process testing exclusively. Creates command trees in test code, executes with `SetArgs()`, captures output via `SetOut()`/`SetErr()` to `bytes.Buffer`. Helper functions `executeCommand()`, `executeCommandC()`, `checkStringContains()`, `checkStringOmits()`.

**Takeaway for gdev**: Use Cobra's in-process pattern for unit-testing individual commands (flag parsing, argument validation, help text). Don't use it for E2E.

### mise (Rust)

**Framework**: Rust `#[test]` + `insta` (snapshot testing) + bash-based E2E.

**Architecture**: Three layers: unit tests in source files, snapshot tests with `insta`, and bash E2E tests in `e2e/` directory. E2E is the preferred approach for new functionality. Rich `assert.sh` library with `assert_contains`, `assert_json`, `assert_directory_exists`. Complete environment isolation via `setup_isolated_env()`.

**Multi-platform**: Dedicated `e2e-win/` directory with PowerShell scripts for Windows. Linux/macOS testing via bash. GitHub Actions matrix.

**Key pattern**: E2E tests dominate because they catch integration issues unit tests miss. Environment isolation prevents test pollution.

**Takeaway for gdev**: mise's approach validates that E2E-heavy testing is the right strategy for dev environment tools. Their bash assertion library is a good model for install script testing. Separate Windows test scripts are pragmatic.

### rustup (Rust)

**Framework**: Rust `#[test]` + custom test harness.

**Install script testing**: Minimal -- they rely on CI matrix builds across real platforms rather than unit-testing the bash script. The `-y` flag enables non-interactive mode. Script is designed for minimal shell environments (dash/bash/zsh/ksh).

**Takeaway for gdev**: The `--yes`/`-y` pattern for non-interactive mode is standard. Testing install scripts via real platform CI (not mocking) catches the real issues.

### Volta (Rust)

**Framework**: Rust `#[test]` + `assert_cmd` crate.

**Architecture**: Focuses on consistency guarantees -- the same tool versions across all environments. Uses `volta-cli/action` GitHub Action for CI integration.

**Multi-platform**: Native Windows support (not WSL), macOS, Linux. GitHub Actions matrix with version-specific testing.

**Takeaway for gdev**: Volta's approach to testing "the right version is installed" maps to gdev's need to verify "the right tools are detected/installed."

### devenv (Nix)

**Framework**: Nix-based testing via `devenv test` command.

**Architecture**: Tests defined in `enterTest` configuration. Process lifecycle management (start/stop services for testing). Helper functions like `wait_for_port`. Tests run via `devenv test` in CI.

**Takeaway for gdev**: Limited applicability since devenv tests are Nix-native. However, the concept of "build the environment, then verify it works" is exactly what gdev E2E tests should do.

### Homebrew (Ruby)

**Framework**: RSpec + Codecov.

**Architecture**: CI tests on every PR. Network-dependent tests tagged `~needs_network` and skipped by default. CI-only tests tagged `~needs_ci`. Parallel test execution for performance.

**Takeaway for gdev**: Tag-based test filtering (network/CI/platform) is essential. Codecov integration as a guide rather than a gate is pragmatic.

---

## 11. Recommended Test Architecture

### Test Pyramid

```
                    /\
                   /  \
                  / E2E \        ~20 tests, slow, real binary
                 /--------\
                /Integration\    ~50 tests, medium, real tools
               /--------------\
              /   Unit Tests    \  ~200+ tests, fast, in-process
             /____________________\
```

### Test Tiers

| Tier | Build Tag | What It Tests | How It Runs | Speed |
|------|-----------|---------------|-------------|-------|
| **Unit** | (none) | Command logic, template rendering, config parsing, OS detection | `go test ./...` | <30s |
| **Integration** | `integration` | File generation, state management, tool detection | `go test -tags=integration ./...` | <2min |
| **E2E** | `e2e` | Full binary workflows (init, doctor, enable/disable) | `go test -tags=e2e ./e2e/...` | <10min |
| **Distro** | `e2e,distro` | Package manager integration, real installs | Docker containers in CI | <15min |
| **Install** | N/A (BATS/Pester) | Install scripts | `bats tests/` / `Invoke-Pester` | <2min |
| **Performance** | `e2e` | Wall-clock timing, benchmarks | `go test -bench=. -count=5` | <5min |

### Framework Selection

| Component | Framework | Why |
|-----------|-----------|-----|
| **Unit tests** | Go `testing` + `testify/require` | Standard, fast, good assertions |
| **E2E tests** | `testscript` (txtar-based) | Self-contained scripts, built-in platform conditions, custom commands, used by Go itself |
| **Golden files** | `testscript` `UpdateScripts` + `cmp` | Integrated with testscript, auto-update support |
| **Structured assertions** | Custom `yaml_has`/`json_path` testscript commands | Domain-specific, reusable across scripts |
| **TUI testing** | `teatest` (experimental, for wizard-specific tests only) | Official charm testing tool |
| **Bash install tests** | BATS + bats-assert + bats-support | Standard for bash testing, TAP/JUnit output |
| **PowerShell tests** | Pester 5.x | Standard for PowerShell, JUnit output |
| **Static analysis** | ShellCheck + shfmt | Catch bash issues before runtime |
| **Test runner** | `gotestsum` | JUnit XML, retries, human-readable output |
| **Coverage** | Go 1.20+ `-cover` build flag + `GOCOVERDIR` | E2E coverage from compiled binary |
| **Benchmarks** | Go `testing.B` + `benchstat` | Statistical comparison, regression detection |
| **CI orchestration** | GitHub Actions matrix | Native multi-OS, container support for distros |

### Recommended Directory Structure

```
gdev/
  cmd/
    init/
      init.go
      init_test.go                    # Unit: command flags, arg validation
    doctor/
      doctor.go
      doctor_test.go                  # Unit: diagnostic checks
    enable/
      enable.go
      enable_test.go                  # Unit: tool toggle logic
  internal/
    detector/
      detector.go
      detector_test.go                # Unit: OS/tool detection
      testdata/
        os-release/                   # Embedded fixtures
    generator/
      generator.go
      generator_test.go               # Unit: template rendering
      testdata/
        templates/                    # Template fixtures
    state/
      state.go
      state_test.go                   # Unit: state file operations
  e2e/
    e2e_test.go                       # testscript runner + custom commands
    helpers_test.go                   # Shared test utilities
    testdata/
      script/
        init-go-defaults.txt          # E2E: init with Go defaults
        init-typescript-full.txt      # E2E: init with all TS features
        init-minimal.txt              # E2E: minimal init
        init-idempotent.txt           # E2E: run init twice
        enable-disable-cycle.txt      # E2E: toggle tools
        doctor-healthy.txt            # E2E: doctor on clean system
        doctor-missing-tools.txt      # E2E: doctor detects issues
        state-round-trip.txt          # E2E: init -> modify -> re-init
        self-update.txt               # E2E: self-update flow
      answers/
        go-defaults.yaml
        typescript-full.yaml
        minimal.yaml
      golden/                         # Golden files for cmp commands
        go-defaults/
          devenv.nix.golden
          settings.json.golden
  e2e/distro/
    distro_test.go                    # Container-based distro tests
    Dockerfile.fedora
    Dockerfile.debian
    Dockerfile.arch
    Dockerfile.alpine
  tests/
    install.bats                      # BATS: bash install script
    Install.Tests.ps1                 # Pester: PowerShell install script
    bats-support/                     # git submodule
    bats-assert/                      # git submodule
  scripts/
    install.sh                        # Bash install script
    install.ps1                       # PowerShell install script
```

### The 80/20 Implementation Order

**Phase 1 -- Foundation (highest ROI, do first)**:
1. Add `--non-interactive` flag and `--answers-file` flag to `gdev init`
2. Set up `testscript` runner with 3-5 basic E2E scripts
3. Add custom testscript commands: `yaml_has`, `json_path`, `grep` (built-in)
4. Unit tests for detector, generator, and state packages
5. `gotestsum` + JUnit XML in CI

**Phase 2 -- Coverage and confidence**:
6. Golden file tests for all generated file types
7. State management round-trip tests
8. BATS tests for `install.sh`
9. Build with `-cover`, collect E2E coverage via `GOCOVERDIR`
10. GitHub Actions matrix: ubuntu, macos, windows

**Phase 3 -- Breadth**:
11. Docker-based distro tests (Fedora, Debian, Arch, Alpine, NixOS)
12. WSL2 tests
13. Pester tests for `install.ps1`
14. `teatest` tests for wizard UI (low priority -- answers-file bypasses TUI)

**Phase 4 -- Polish**:
15. Performance benchmarks + benchstat in CI
16. Coverage report aggregation across matrix
17. Flaky test detection and retry
18. PR annotations from test results

### Critical Design Decisions

1. **testscript over raw os/exec**: testscript provides platform conditions, custom commands, golden file support, and self-contained test scripts. It's the same framework the Go project uses for 900+ tests.

2. **`--answers-file` over teatest for E2E**: Don't route E2E tests through the TUI. The answers-file pattern is simpler, faster, more reliable, and easier to parameterize. Reserve teatest for testing the wizard UI specifically.

3. **Build tags over directories for test separation**: Use `//go:build e2e` rather than `if os.Getenv("RUN_E2E") ...`. Build tags are the Go-idiomatic approach, enforced by the compiler, and supported by all tooling.

4. **Golden files for generated content**: Generated files (devenv.nix, settings.json, CI workflows) should be golden-file tested. This catches accidental regressions in templates. Use structured assertions (json_path, yaml_has) as secondary verification for specific values.

5. **Docker containers for distro testing**: Don't try to install 12 distros on CI runners. Use container jobs with the actual distro images. This tests real package managers (apt, dnf, pacman, apk) without mocking.

6. **BATS for bash, Pester for PowerShell**: Don't try to test bash scripts with Go. Use the native testing frameworks for each language. Both produce JUnit XML for CI integration.
