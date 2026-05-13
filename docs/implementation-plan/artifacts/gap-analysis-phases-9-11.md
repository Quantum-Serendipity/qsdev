# Gap Analysis: Phases 9-11

Analysis date: 2026-05-12
Scope: Cross-Platform System Detection (9), Distribution & Self-Bootstrapping (10), AI Agent Tooling Integration (11)

---

## Phase 9: Cross-Platform System Detection

### 9-G1: OSInfo struct missing fields needed by Phases 10-11 (Medium)

Phase 10's self-update mechanism needs to know the current binary path (`os.Executable()`) and install location (`~/.gdev/bin/` vs system package manager vs Homebrew cellar). Phase 10's install scripts need to detect whether gdev was installed via package manager (and therefore should not self-update — leave it to brew/apt/scoop). Neither of these are in the `OSInfo` struct. The struct should include `GdevInstallMethod` (curl-script, brew, scoop, apt, rpm, manual) and `GdevBinaryPath`.

Phase 11's semble integration needs to know whether Python >= 3.10 is available and whether `uvx` is on PATH. The prerequisite checks in Unit 9.4 list `python3` with a minimum of `>= 3.11` (for Version-Sentinel's `tomllib`), but semble only needs `>= 3.10`. There is no `uvx` check at all in the tool prerequisite list. `uvx` should be added as an optional tool detection entry.

### 9-G2: Package manager abstraction is mock-only tested (High)

Unit 9.2's acceptance criteria focus on unit tests with mock exec. There are no integration tests that run `apt-get`, `brew`, or `pacman` on real systems. The CI workflow in Phase 10 (Unit 10.2) runs tests on a matrix of Linux/macOS/Windows but Phase 9 has no analogous CI integration test requirement. The package name registry (13 tools x 12 managers = 156 mappings) has especially high potential for incorrect mappings that unit tests will not catch — the Fedora `ShellCheck` capitalization, the Gentoo `dev-util/shellcheck` category prefix, and similar platform-specific quirks are verified only via table-driven mocks, never against real package manager output.

**Recommendation:** Add an integration test requirement to Phase 9's completion criteria that runs `gdev doctor` on at least Ubuntu, macOS, and Windows in CI, validating that real package manager detection and real tool detection produce correct results.

### 9-G3: `gdev doctor` CI/no-TTY output not specified (Medium)

Unit 9.5 specifies color-coded output with emoji (green check, red X, yellow exclamation). Step 6 says "Color output works in terminals, gracefully degrades in pipes/CI" but does not specify what the degraded output looks like. There is no acceptance criterion for no-TTY mode beyond `--json`. Developers will pipe `gdev doctor` to log files or use it in CI scripts. The degradation behavior needs specification:
- Strip ANSI codes when stdout is not a TTY (`os.Stdout.Fd()` + `isatty`)
- Replace Unicode symbols with ASCII alternatives (`[OK]`, `[FAIL]`, `[WARN]`)
- Suppress progress spinners

The `--check` exit code mode is specified but there is no `--quiet` flag for scripts that only want the exit code without output.

### 9-G4: `gdev setup` partial failure recovery unspecified (High)

Unit 9.6 has no strategy for what happens when installation fails mid-way. Consider the scenario: `gdev setup --yes` on Ubuntu installs git and go successfully, then the Nix install (via Determinate Systems) fails partway. The acceptance criteria say "Post-install verification confirms all tools work" but there is no:
- Rollback strategy (is there one? Or is it just "report what failed"?)
- Re-entry safety (can the user re-run `gdev setup` and have it skip already-installed tools?)
- State tracking of what was attempted vs what succeeded

The re-entry question is partially answered by the "re-run checks to verify success" step, but this is at the end of the flow. If the Nix install fails, does the flow continue to attempt direnv, devenv, and Claude Code (which all depend on Nix)? The dependency graph (`nix -> devenv`, `node -> claude-code`) is acknowledged in Step 2f but there is no explicit error handling for "dependency failed, skip dependents."

**Recommendation:** Add explicit acceptance criteria: (1) `gdev setup` is idempotent — re-running skips already-installed tools, (2) dependency chain failures skip dependent tools with clear error messages, (3) partial success produces a summary showing what was installed and what was skipped and why.

### 9-G5: NixOS declarative config output — format and completeness (Medium)

Unit 9.6 Step 6 says "output Nix expressions for `environment.systemPackages`" but does not specify:
- What format? A copy-pasteable snippet? A complete `configuration.nix` fragment?
- Does it handle NixOS users who use home-manager instead of `environment.systemPackages`? The research (artifacts/os-prerequisite-detection-research.md section 1.11) lists both `environment.systemPackages` (NixOS system config) and `home.packages` (home-manager). The plan only mentions the former.
- Does the output include flake vs non-flake variants? NixOS users increasingly use flakes.
- Is `programs.direnv.enable = true;` suggested instead of adding direnv to systemPackages? The research notes this as an alternative (section 2.6) but the plan does not differentiate.

The acceptance criterion "NixOS users get declarative Nix expressions" is too vague. A NixOS user pasting a half-wrong snippet into their `configuration.nix` will get a build error that erodes trust in gdev immediately.

**Recommendation:** Specify the exact format. Output a comment-annotated `configuration.nix` fragment that uses `programs.direnv.enable = true` (when available), `environment.systemPackages` for the rest, and includes a note about home-manager as an alternative.

### 9-G6: Windows native (no WSL2) developer experience gaps (Medium)

Phase 9 correctly identifies that Nix, devenv, and nix-direnv are unavailable on native Windows. But the plan does not address what the developer experience actually is in native-only mode:
- `gdev init` generates devenv.nix, devenv.yaml, and .envrc — all useless without Nix/devenv/direnv. What does `gdev init` produce on native Windows? Is the devenv addon entirely skipped?
- The wizard (Phase 6) has a "Dev Environment" group with direnv toggle and git hooks. How does this render when direnv is unavailable?
- Pre-commit hooks use prek (devenv) — what is the fallback on Windows without devenv?

The plan says `gdev doctor` "reports which features are available" but does not specify how this feeds back into `gdev init`'s behavior. A native Windows developer running `gdev init --yes` should not get files that reference nonexistent tools.

**Recommendation:** Add an explicit "Windows native feature matrix" to Phase 9 or Phase 6, documenting which addons/features are available, degraded, or disabled. The wizard should suppress irrelevant form groups.

---

## Phase 10: Distribution & Self-Bootstrapping

### 10-G1: GoReleaser references `Quantum-Serendipity` org — repos not confirmed to exist (Medium)

Phase 10, Unit 10.2, Step 4 says "Create Homebrew tap repo skeleton (`Quantum-Serendipity/homebrew-tap`)" and Step 5 says "Create Scoop bucket repo skeleton (`Quantum-Serendipity/scoop-bucket`)." These are listed as steps, not prerequisites. But the GoReleaser config also references these repos — if they do not exist when the first release is tagged, GoReleaser will fail.

The research artifact (`cross-platform-distribution-research.md`) uses placeholder `your-org` throughout. The plan correctly substitutes `Quantum-Serendipity` but does not:
- Confirm the GitHub org exists and has these repos
- Specify who creates the repos (Phase 10 implementer? Infra team?)
- Define a fallback if the repos cannot be created (e.g., disable brew/scoop publishing in GoReleaser and rely only on GitHub Releases)

Additionally, the GoReleaser config in the research uses `homebrew_casks` (line 106) which is for macOS .app-style casks, not CLI formulas. The correct GoReleaser key for Homebrew CLI formulas is `brews` (formerly `homebrew_taps`). This is a bug in the research artifact that will propagate to the implementation if not caught.

**Recommendation:** (1) Verify the GoReleaser YAML key — `brews` not `homebrew_casks`. (2) Add a Phase 10 prerequisite step: "Ensure `Quantum-Serendipity/homebrew-tap` and `Quantum-Serendipity/scoop-bucket` GitHub repos exist before first release."

### 10-G2: Install script hosting — `get.gdev.dev` undefined (Low)

Unit 10.3 specifies `curl -fsSL https://get.gdev.dev/install.sh | sh` as the install URL. The domain `get.gdev.dev` is not set up and there is no step to configure it. The fallback is raw GitHub URLs (`https://raw.githubusercontent.com/Quantum-Serendipity/gdev/main/scripts/install.sh`).

This is a low-priority gap because raw GitHub URLs work fine for initial distribution. However:
- The plan should explicitly state which URL is used in v1.0 (raw GitHub) vs future (custom domain)
- The install scripts hardcode the GitHub org/repo — this should be parameterized or at least prominently placed for easy updating

### 10-G3: Self-update replacing running binary on Windows (High)

Unit 10.4 Step 3 says "Replace current binary (rename current -> backup, write new, remove backup on success, restore backup on failure)." On Windows, a running executable has a file lock — it cannot be renamed or deleted while the process is running.

The research artifact does not address this. Real-world Go self-updaters on Windows use one of:
- Rename the running binary to `.old`, write the new binary, then launch the new binary which deletes `.old` on startup
- Use a separate updater process spawned by the main binary
- Copy to temp, exec the temp copy, temp copy replaces the original

The `rename current -> backup` approach described will fail on Windows with "The process cannot access the file because it is being used by another process."

**Recommendation:** Add Windows-specific self-update logic in Unit 10.4. The standard pattern is: (1) rename running binary to `gdev.old.exe`, (2) write new binary as `gdev.exe`, (3) print "Update complete. Restart gdev to use v$NEW_VERSION", (4) on next startup, delete `gdev.old.exe` if present. This works because Windows allows renaming a locked file — just not deleting it.

### 10-G4: PowerShell `$PROFILE` path may not exist (Medium)

Unit 10.5 Step 2 says completions for PowerShell involve adding to `$PROFILE`. The `$PROFILE` variable always resolves to a path, but the file (and its parent directories) may not exist — PowerShell does not create it automatically. If gdev writes to `$PROFILE` without ensuring the file/directory exists first, it will fail for users who have never customized their PowerShell profile.

The research artifact (section 3.5) shows `$PROFILE` resolving to `~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1` — neither the `Documents/PowerShell/` directory nor the file may exist.

**Recommendation:** Add a step: "Create `$PROFILE` parent directory and file if they do not exist before writing completions."

### 10-G5: nFPM `.archlinux` package — untested, AUR more appropriate (Low)

Unit 10.2 Step 1 lists `.archlinux` as an nFPM format. nFPM does support generating Arch Linux packages, but the standard Arch Linux distribution mechanism is the AUR (Arch User Repository), not direct package files. The research artifact (section 1.2) lists AUR as "Tier 2: Secondary" with "Community maintained" review process.

Potential issues:
- Arch users expect packages from the AUR, not from random `.pkg.tar.zst` downloads
- The `.archlinux` nFPM format generates a package, but there is no mechanism to distribute it to Arch users (it is attached to the GitHub Release, but `pacman` cannot install from a URL without `pacman -U`)
- AUR packages are not mentioned in the phase plan at all

**Recommendation:** Keep the nFPM `.archlinux` format for completeness (it is free to generate) but add a note that AUR publication is the proper Arch distribution channel and should be pursued as a follow-up.

### 10-G6: Phase completion criteria says "All five units" but there are five units (OK)

The phase completion criteria says "All five units pass acceptance criteria." Units 10.1 through 10.5 — this is consistent. No gap here. (Noted because Phase 9 says "all six units" and has six, which is also correct.)

---

## Phase 11: AI Agent Tooling Integration

### 11-G1: agent-postmortem-skill abandonment contingency missing (Medium)

The plan (master plan finding #23) notes all three tools are "2-25 days old" and "integration should be modular — easy to swap out." But Unit 11.1 does not specify what "easy to swap out" means concretely:
- Is the SKILL.md embedded as a separate file that can be replaced without rebuilding?
- Is there a feature flag to disable it without code changes?
- What happens if the upstream repo is deleted or relicensed?

The SKILL.md is MIT-licensed and only 3.6KB — the real risk is not legal but quality. If the skill prompt turns out to produce noisy or unhelpful output, developers will want to disable it.

Unit 11.4 does provide a wizard toggle (`PostmortemEnabled` bool), and the file uses `MergeStrategy: Overwrite` meaning gdev owns it. This is adequate for disabling, but the plan should explicitly state that the embedded copy is gdev's fork — the upstream is reference material, not a live dependency. If upstream changes, gdev updates its embedded copy on its own schedule.

**Recommendation:** Add a note to Unit 11.1: "The embedded SKILL.md is a gdev-maintained fork. Upstream changes are reviewed and merged manually. The wizard toggle is the disable mechanism."

### 11-G2: Version-Sentinel plugin marketplace API stability unaddressed (High)

Unit 11.2 generates the command `claude plugin marketplace add https://github.com/KSEGIT/Version-Sentinel.git` and `claude plugin install version-sentinel@version-sentinel-marketplace`. This relies on:
1. The Claude Code plugin marketplace existing and being stable
2. The `marketplace add` / `plugin install` command syntax remaining unchanged
3. The Version-Sentinel plugin remaining in the marketplace

The Claude Code plugin marketplace is new and its API/CLI interface may change. The plan does not specify:
- What version of Claude Code introduced `plugin marketplace`?
- Is there a fallback if the command syntax changes?
- What if the plugin is removed from the marketplace?

Looking at the Version-Sentinel hooks.json, the plugin uses `${CLAUDE_PLUGIN_ROOT}` environment variable — this is a Claude Code plugin runtime feature. If Claude Code changes how plugins are loaded, all hooks break.

**Recommendation:** (1) Add a fallback path: manual hook wiring via `settings.json` entries that point to locally-cloned scripts, bypassing the marketplace entirely. (2) Pin the minimum Claude Code version that supports the plugin marketplace. (3) Add a check in `gdev doctor` that verifies the `claude` CLI supports the `plugin` subcommand before enabling Version-Sentinel in the wizard.

### 11-G3: semble's `uvx` availability — cross-platform gaps (High)

Unit 11.3 uses `uvx --from "semble[mcp]" semble` as the MCP command. `uvx` is the execution tool from `uv` (the Rust-based Python package manager by Astral). Issues:

1. **uvx is not installed by default anywhere.** It must be installed separately (`pip install uv`, `pipx install uv`, `brew install uv`, etc.). The plan says "gated on Python availability" but `uvx` is not the same as Python — you can have Python without `uvx`.

2. **Windows support:** `uv` and `uvx` work on Windows, but the plan does not verify this. The semble MCP config examples (artifact) show `uvx` as the command — does this work in cmd.exe? PowerShell? Git Bash? The `command` field in `.mcp.json` is executed by Claude Code, which on Windows may use different shell semantics.

3. **Alternative execution:** semble can be installed via `pip install 'semble[mcp]'` and run as `python -m semble` or directly as `semble`. The plan only uses the `uvx` path. A fallback to `pip` installation would broaden compatibility.

4. **Unit 9.4 does not check for `uvx`.** The prerequisite detection engine checks for `python3` but not `uvx` or `uv`. When semble is enabled in the wizard, `gdev doctor` should flag `uvx` as missing.

**Recommendation:** (1) Add `uvx` / `uv` to the Unit 9.4 tool check registry as an optional check when semble is enabled. (2) Implement a fallback MCP config that uses `python -m semble` instead of `uvx` when uvx is not available. (3) Explicitly test the MCP config on Windows with PowerShell as the execution environment.

### 11-G4: `.mcp.json` ThreeWayMerge — no base version for first-time generation (High)

Unit 11.3 Step 8 specifies `MergeStrategy: ThreeWayMerge` for `.mcp.json`. Three-way merge requires three inputs: base (what gdev generated last time), current (what is on disk now, possibly user-modified), and incoming (what gdev wants to generate this time).

On first-time generation (no prior gdev run), there is no base version. The hash tracking system (Phase 1, Unit 1.6) records the hash of what was written, but the ThreeWayMerge implementation (Phase 8) needs the actual content, not just a hash.

Looking at Phase 1, `GeneratedState` tracks `Hash string` per file. Phase 8 implements three-way merge, but its unit description is not included in the files reviewed. The question is: where does the base content come from?

If `.mcp.json` already exists (e.g., user has other MCP servers configured) and gdev has never run before, the three-way merge has:
- Base: empty/null (gdev never generated this file)
- Current: user's existing `.mcp.json`
- Incoming: gdev's desired `.mcp.json` with semble entry

This should reduce to a simple merge-insert (add the `semble` key to the existing `mcpServers` object). But the ThreeWayMerge strategy needs to handle this case explicitly. The plan does not specify what happens when base is absent.

**Recommendation:** Specify that when `GeneratedState` has no record of a file, ThreeWayMerge falls back to `Merge` (insert-only) strategy. For JSON files specifically, this means "add our keys to the existing object without touching other keys." Add this as an explicit requirement in Phase 8's three-way merge implementation or as a note in Unit 11.3.

### 11-G5: Wizard extension — new form group vs existing Phase 6 structure (Medium)

Unit 11.4 says "Add an 'AI Agent Tools' form group to the huh wizard" and suggests "Group 5b or merged into existing Claude Code group." Phase 6, Unit 5.2 defines 6 groups:
- Group 1: Quick Selection
- Group 2: Languages & Runtimes
- Group 3: Services
- Group 4: Dev Environment
- Group 5: Claude Code
- Group 6: Plan Preview & Confirm

Adding Group 5b (between Claude Code and Plan Preview) changes the total group count and breaks the "5 form groups for customizers" promise from the plan overview (design principle 3: "1 question on the quick path, 5 form groups for customizers"). This becomes 6 form groups.

More practically: Phase 6's acceptance criteria say "Customize path shows all 6 groups" (Groups 1-6). Adding Group 5b makes it 7. This is not a functional break, but:
- Phase 6's tests will need updating when Phase 11 lands
- The form group count in the overview/design principles needs updating
- The `WithHideFunc` logic needs to hide the AI tools group when `ClaudeCode = false` (which Unit 11.4 correctly specifies)

**Recommendation:** (1) Acknowledge in Unit 11.4 that this changes Phase 6's group count from 6 to 7. (2) Specify whether the AI tools group is a sub-group of the Claude Code group (avoids count change) or a separate group. (3) Update the plan overview's "5 form groups" claim or note it as an outdated artifact.

### 11-G6: EcosystemModule interface extension is a breaking change (High)

Unit 11.5 adds two methods to the `EcosystemModule` interface:
```go
VerificationCommands() VerificationCommands
ManifestFiles() []ManifestFileInfo
```

Phase 1, Unit 1.7 defines the `EcosystemModule` interface with 11 methods. Phase 2 implements 8 Tier 1 modules. Phase 7 implements 19 more modules. Adding 2 methods to the interface means ALL 27 modules must be updated to compile.

The plan does not address this:
- When do existing modules get updated? Phase 11 depends on Phase 2 (Tier 1 modules), so the Tier 1 modules exist. But they do not implement `VerificationCommands()` or `ManifestFiles()`.
- Unit 11.1 Step 4 mentions the alternative: "or define a supplementary interface `VerifiableModule`". This is the right approach but it is presented as an afterthought, not a decision.

Using a supplementary interface (`VerifiableModule`) is the non-breaking approach:
```go
type VerifiableModule interface {
    VerificationCommands() VerificationCommands
    ManifestFiles() []ManifestFileInfo
}
```
Code in Phase 11 type-asserts: `if vm, ok := mod.(VerifiableModule); ok { ... }`. Modules that do not implement it simply return no commands/files.

**Recommendation:** Commit to the supplementary interface approach in Unit 11.5. Remove the alternative of modifying `EcosystemModule` directly. This avoids a cascade of changes to Phases 2 and 7.

---

## Cross-Phase Issues

### X-G1: Phase 9's OSInfo and Phase 1's DetectedProject overlap (Medium)

Phase 1's `DetectedProject` struct (Unit 1.3) detects languages, existing configs, and git state by scanning the project directory. Phase 9's `OSInfo` struct (Unit 9.1) detects the operating system, package managers, and available tools.

These are conceptually distinct (project vs system), but they overlap in several areas:
- Both detect whether Nix is available (`HasNix` in OSInfo; Nix ecosystem detection in DetectedProject)
- Both need to know the shell environment (OSInfo has `Shell`, `ShellPath`, `ShellRCFile`; the detection engine needs shell info for direnv hook detection)
- Both are consumed by the wizard — `DetectedProject` pre-populates language choices, `OSInfo` determines which tools are available

The interaction is not specified. Questions:
- Does the wizard receive both structs independently? Or does one embed the other?
- When `gdev init` runs, does it call `DetectOS()` first, then `Detect(projectRoot)`, or vice versa?
- Is `OSInfo` available to the detection engine? (e.g., "don't detect devenv.nix if OSInfo says Nix is not installed" — but this seems wrong, the file could exist from a CI pipeline)

**Recommendation:** Specify the data flow explicitly. `DetectOS()` runs first (system-level, no project root needed). `Detect(projectRoot)` runs second (project-level). The wizard receives both. `OSInfo` is available to the wizard for capability gating (hide devenv options if no Nix) but NOT to the detection engine (which should report what files exist regardless of system capabilities).

### X-G2: Phase 10's embedded assets depend on Phases 2-4 templates — build ordering (Low)

Unit 10.1 Step 3 lists `embed.FS` declarations for templates from Phases 2-4:
- `addons/devenv/templates/` — from Phase 3
- `addons/claudecode/templates/` — from Phase 4
- `addons/claudecode/skills/` — from Phase 4 and Phase 11

Phase 10 depends on Phases 1 and 9 (per the dependency declaration). But its `embed.FS` declarations reference template directories created in Phases 2-4 and skill files from Phase 11.

This is not a circular dependency — it is an implicit dependency. The binary cannot embed files that do not exist. Two scenarios:
1. If Phases 2-4 and 11 are complete before Phase 10: no issue, files exist.
2. If Phase 10 is started before Phase 2: the `embed.FS` directories need placeholder files or the build will not find any templates.

The dependency graph says Phase 10 depends on Phase 1 and Phase 9 only. This is technically correct for the Go code (OS detection, package manager abstraction), but wrong for the embed declarations.

**Recommendation:** Either (1) update Phase 10's dependencies to include Phases 2-4 (making the embed requirement explicit), or (2) have Phase 10's embed declarations use placeholder/skeleton template files with a comment that real content comes from other phases. Option 1 is cleaner — Phase 10 is a final assembly phase and should logically come after the content phases.

### X-G3: Phase 11 dependency graph — not circular, but sequencing is unclear (Medium)

Phase 11 depends on Phases 2, 4, and 6. Let us trace the chain:
- Phase 2 depends on Phase 1
- Phase 4 depends on Phases 1, 2
- Phase 6 depends on Phases 3, 4 (Phase 3 depends on Phases 1, 2; Phase 4 depends on Phases 1, 2)

So Phase 11 requires Phases 1 -> 2 -> 3 -> 4 -> 6. This is not circular. But:

Phase 11 extends the wizard from Phase 6 (Unit 11.4) and extends settings.json generation from Phase 4 (Unit 11.2). The plan says Phase 6 is a dependency but labels it "partial" in Phase 9's dependencies section. For Phase 11, is Phase 6 full or partial?

Unit 11.4's Step 1 says "Extend `WizardAnswers` struct (Phase 1, Unit 1.2)". This is a modification to a Phase 1 output. If Phase 1 is "complete" when Phase 11 starts, extending WizardAnswers requires reopening Phase 1's struct definition. This is normal in Go (just add fields), but it means Phase 1's unit tests for `WizardAnswers` JSON round-trips may need updating.

**Recommendation:** Note explicitly that Phase 11 modifies Phase 1's `WizardAnswers` struct and Phase 6's wizard form builder. These are additive changes (new fields, new form groups) and should not break existing behavior, but existing tests need updating.

### X-G4: Version-Sentinel prerequisites bleed into Phase 9 (Medium)

Unit 11.2 Step 5 says "Check Version-Sentinel prerequisite availability (jq, curl, python3 >= 3.11) during `gdev doctor` — add these to the prerequisite checks in Phase 9." This means Phase 11's implementation modifies Phase 9's code.

Similarly, Unit 11.3 Step 6 says "Add to `gdev doctor` checks when semble is enabled." This is another Phase 9 modification from Phase 11.

These are not captured in Phase 9's units. Phase 9's Unit 9.4 lists `jq`, `curl`, and `python3` as optional/recommended checks, but not as conditionally-required based on Phase 11 tool enablement. The distinction matters: `python3 >= 3.11` is only required if Version-Sentinel is enabled, and `uvx` is only required if semble is enabled.

**Recommendation:** Either (1) add a generic extensibility mechanism to Phase 9's tool check registry (so Phase 11 can register additional checks without modifying Phase 9 code), or (2) explicitly note that Phase 11 retroactively extends Phase 9's tool check list and Phase 9's test suite.

### X-G5: GoReleaser config in research artifact has wrong Homebrew key (Medium)

The `cross-platform-distribution-research.md` artifact (the authoritative source for Phase 10) uses `homebrew_casks:` in the GoReleaser YAML (line 106). GoReleaser's `homebrew_casks` is for macOS `.app` cask distribution, not for CLI formula distribution. The correct key for CLI tools is `brews:`. Since gdev is a CLI binary (not a macOS .app bundle), using `homebrew_casks` will either produce an error or generate an incorrect Homebrew artifact.

Phase 10 Unit 10.2 says to use the "complete configuration from research" — if this is followed literally, the Homebrew tap will be broken.

**Recommendation:** Fix the research artifact or add a note in Unit 10.2: "Use `brews:` key in GoReleaser config, not `homebrew_casks:` as shown in the research artifact."

---

## Prioritized Recommendations

### Critical (must fix before implementation)

| ID | Phase | Issue | Impact |
|----|-------|-------|--------|
| 10-G3 | 10 | Self-update cannot replace running binary on Windows | `gdev self-update` will fail on Windows — the primary enterprise platform |
| 11-G6 | 11 | EcosystemModule interface extension breaks all 27 modules | Compilation failure across Phases 2 and 7; use supplementary interface instead |
| 11-G4 | 11 | ThreeWayMerge has no base version on first run | `.mcp.json` merge will fail or corrupt existing user config on first `gdev init` |
| X-G5 | 10 | GoReleaser uses wrong Homebrew key (`homebrew_casks` vs `brews`) | Homebrew tap will be broken on first release |

### High (fix during unit design, before coding)

| ID | Phase | Issue | Impact |
|----|-------|-------|--------|
| 9-G2 | 9 | Package manager abstraction has no integration tests | 156 package name mappings tested only via mocks; real-world failures will be invisible |
| 9-G4 | 9 | `gdev setup` partial failure recovery unspecified | Developers stuck with half-installed environment and no clear way to resume |
| 11-G2 | 11 | Version-Sentinel depends on unstable Claude Code plugin marketplace API | Plugin install command may break with any Claude Code update |
| 11-G3 | 11 | semble depends on `uvx` which is not checked or widely installed | MCP server will fail to start on most systems without manual uvx installation |

### Medium (address during implementation)

| ID | Phase | Issue | Impact |
|----|-------|-------|--------|
| 9-G1 | 9 | OSInfo missing fields for Phase 10-11 needs (install method, uvx) | Downstream phases will need to re-detect information OSInfo should have provided |
| 9-G3 | 9 | `gdev doctor` no-TTY/CI output unspecified | CI and piped output will be garbled with ANSI codes |
| 9-G5 | 9 | NixOS declarative config output underspecified | NixOS users get wrong or incomplete Nix expressions |
| 9-G6 | 9 | Windows native feature matrix undefined | Windows developers get files referencing tools that do not exist |
| 10-G1 | 10 | Tap/bucket repos not confirmed to exist | First GoReleaser release will fail if repos are missing |
| 10-G4 | 10 | PowerShell $PROFILE path may not exist | Completion install crashes for users who never customized PowerShell |
| 11-G1 | 11 | agent-postmortem-skill abandonment contingency vague | Team unclear on maintenance model for embedded skill |
| 11-G5 | 11 | Wizard group count changes break Phase 6 description | Tests and documentation will be inconsistent |
| X-G1 | Cross | OSInfo and DetectedProject data flow unspecified | Ambiguous interaction between system and project detection |
| X-G3 | Cross | Phase 11 modifies Phase 1 and Phase 6 outputs | Test updates needed for earlier phases |
| X-G4 | Cross | Phase 11 tool prerequisites bleed into Phase 9 | Phase 9 code needs retroactive modification |

### Low (note for implementers, fix if convenient)

| ID | Phase | Issue | Impact |
|----|-------|-------|--------|
| 10-G2 | 10 | `get.gdev.dev` domain undefined; raw GitHub URL works | Vanity URL is nice-to-have, not blocking |
| 10-G5 | 10 | nFPM Arch package exists but AUR is the real distribution channel | Arch users expect AUR; nFPM package is unused but harmless |
| X-G2 | Cross | Phase 10 embed.FS depends on Phases 2-4 templates | Build ordering needs placeholder files or explicit dependency |
