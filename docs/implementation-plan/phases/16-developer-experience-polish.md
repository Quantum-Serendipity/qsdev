# Phase 16: Developer Experience Polish

## Goal

Add targeted commands and integrations that make gdev feel like a complete, polished developer platform: self-healing (`qsdev repair`), quick project info (`qsdev info`), dependency freshness (`qsdev outdated`), coordinated updates (`qsdev update`), clean project exit (`qsdev teardown`), git workflow automation, and shell integration. These are the "last mile" features that turn a useful tool into one developers love. Every feature follows gdev's core pattern: generate one more file, add one more diagnostic, or thin-wrap one more ecosystem command.

## Dependencies

Phase 3 complete (devenv addon — shell integration and env vars generate into devenv.nix). Phase 9 complete (system detection — repair builds on `qsdev devenv doctor` diagnostics). Phase 10 complete (distribution/self-update — `qsdev update` reuses the self-update mechanism). Phase 12 complete (tool lifecycle — all new commands are lifecycle-managed where applicable). Phase 13 complete (project config — `qsdev info` reads `.qsdev.yaml` and `.gdev/state.yaml`). Phase 15 complete (health/status — `qsdev repair` builds on drift detection from health checks).

## Phase Outputs

- `qsdev repair` command for conservative self-healing of corrupted/drifted configs
- `qsdev info` lightweight project status command with one-line and JSON modes
- `qsdev outdated` cross-ecosystem dependency freshness checker (thin wrapper)
- `qsdev update` coordinated update command (binary + configs + devenv inputs)
- `qsdev teardown` clean project exit with three profiles (quick/default/compliance)
- Git workflow automation (PR templates, branch naming hooks, commit ticket extraction, automated PR labels)
- Shell and environment integration (Starship prompt, gdev env vars, enterShell notifications, OTEL env vars)

---

### Unit 16.1: qsdev repair Command

**Description:** Implement `qsdev repair` as a conservative auto-fix companion to `qsdev devenv doctor` (Phase 9). It detects broken/drifted gdev-managed files and fixes them automatically where safe, with mandatory backup before any mutation. Covers four failure categories: Nix/devenv failures, generated config corruption, tool/package failures, and environment drift.

**Code-Grounded Implementation Note:** The existing update infrastructure provides the foundation for repair. `FileUpdatePlan` at `addons/devinit/update.go:43-52` already has Path, Status, Strategy, Action, NewContent, OldContent, NewMode, and Reason fields. The `buildUpdatePlan()` function at `addons/devinit/update.go:180-262` already plans per-file actions (skip, overwrite, sidecar). `UpdateDevenvNix()` at `internal/update/nix_update.go:42-126` already handles the sidecar strategy for modified devenv.nix (generating `.new` + diff). Repair extends this by using the plan builder in a more aggressive mode — force regeneration of corrupted files while preserving truly user-modified ones. The existing plan builder's `Strategy` field (overwrite vs sidecar vs skip) maps directly to repair's conservative/force/reset modes.

**Context:** `qsdev devenv doctor` (Phase 9/15) diagnoses environment health but is read-only. When it reports "pre-commit hooks outdated" or ".envrc modified from generated version," the developer must manually fix each issue. `qsdev repair` closes the loop by automating the fixes that are unambiguously safe while refusing to touch files where auto-repair risks data loss.

The key design principle: **doctor is read-only, repair is write** — never mix diagnosis and modification in one command. Repair is conservative by default: machine-owned files with no user edits are regenerated; machine-owned files with user edits get a backup + `.new` file with diff; hooks are always safe to reinstall; devenv.nix is NEVER auto-modified (the plan establishes this invariant in Phase 8). The `--force` flag overrides the conservative default for machine-owned-with-additions files. The `--reset` flag is the nuclear option — regenerate everything from `.qsdev.yaml`, backing up all originals.

Repair leverages the SHA256 hash tracking from Phase 1/8 (`GeneratedState`) and the drift detection from Phase 15 health checks. Without hash tracking, corruption cannot be detected, so repair operates only on files tracked in `.gdev/state.yaml`.

**Desired Outcome:** `qsdev repair` fixes all issues that `qsdev devenv doctor` can identify, without destroying user customizations. After repair, `qsdev devenv doctor` reports clean health.

**Steps:**
1. Create `internal/repair/` package with `Repairer` struct:
   ```go
   type Repairer struct {
       State      *state.GeneratedState
       Answers    *wizard.SavedAnswers
       Registry   *lifecycle.ToolRegistry
       DryRun     bool
       Force      bool
       TargetFile string // empty = all files, set = single file repair
   }

   type RepairResult struct {
       Fixed   []RepairAction
       Skipped []RepairAction
       Failed  []RepairAction
   }

   type RepairAction struct {
       File        string
       Category    FailureCategory
       Description string
       BackupPath  string
   }
   ```
2. Implement failure category detection by reusing Phase 15 drift checks:
   - **Category 1 — Nix/devenv failures**: Check `devenv info` exit code. If devenv.nix fails evaluation, repair cannot auto-fix (never touch devenv.nix) — log as skipped with message "devenv.nix evaluation failed; review manually or run `qsdev repair --reset` to regenerate from saved answers."
   - **Category 2 — Generated config corruption**: For each tracked file in `GeneratedState`, compute current SHA256. If hash mismatches AND file ownership is `Exclusive`, regenerate from saved answers. If `Shared`, check section markers — if markers intact, regenerate only gdev-owned sections; if markers damaged, backup + regenerate with `--force`.
   - **Category 3 — Tool/package failures**: Check pre-commit hook installation (`ls .git/hooks/pre-commit`), verify hook content matches expected. Check `.gitignore` for required entries. Reinstall hooks via `prek install` or equivalent.
   - **Category 4 — Environment drift**: Compare gdev version in `.qsdev.yaml` against running binary version. If config is older, suggest `qsdev update` instead of repair. Detect new ecosystems (package.json appeared in a Go project) — suggest `qsdev init` to add.
3. Implement backup strategy: before any file mutation, copy original to `.gdev/backups/<filename>.<timestamp>.bak`. Keep last 5 backups per file, prune older ones.
   ```go
   func (r *Repairer) backup(filePath string) (string, error) {
       backupDir := filepath.Join(".gdev", "backups")
       os.MkdirAll(backupDir, 0o755)
       timestamp := time.Now().Format("20060102-150405")
       backupPath := filepath.Join(backupDir, filepath.Base(filePath)+"."+timestamp+".bak")
       return backupPath, copyFile(filePath, backupPath)
   }
   ```
4. Implement repair actions per file type:
   - **Machine-owned exclusive files** (`.envrc`, `devenv.yaml`, `.pre-commit-config.yaml`): Hash check → if unmodified, regenerate in place. If modified, backup + regenerate only with `--force`, otherwise generate `.new` + diff.
   - **Shared files with section markers** (`CLAUDE.md`, `devenv.nix`): Parse markers. If markers intact, regenerate gdev-owned sections only. If markers damaged, restore markers around existing gdev content using heuristics (look for known gdev-generated patterns). For devenv.nix specifically: NEVER auto-modify — always generate `.devenv.nix.new` + diff even with `--force`.
   - **JSON shared files** (`settings.json`, `.mcp.json`): Parse JSON. If parse fails, backup + regenerate base structure + warn about lost user additions. If parse succeeds, re-apply gdev-owned keys/sections using three-way merge from Phase 8.
   - **Git hooks** (`.git/hooks/pre-commit`, `.git/hooks/prepare-commit-msg`): Always safe to reinstall — hooks are not user-edited content. Run hook installer.
   - **`.gitignore` entries**: Check for required entries (`.devenv/`, `.gdev/`, etc.). Append missing entries without removing existing ones.
5. Implement `--dry-run` flag: run all detection and categorization but print what WOULD be fixed without writing any files:
   ```
   $ qsdev repair --dry-run
   Would fix:
     [fix] .envrc — regenerate (unmodified, hash mismatch with current templates)
     [fix] .git/hooks/pre-commit — reinstall hooks
     [skip] settings.json — has user customizations (use --force to regenerate)
     [skip] devenv.nix — never auto-modified (review .devenv.nix.new manually)
   ```
6. Implement `--file <path>` flag for targeted single-file repair. Validates path is a gdev-tracked file.
7. Implement `--reset` flag: regenerate ALL gdev-managed files from `.qsdev.yaml` saved answers. Backup every original. This is the nuclear option for "something is deeply broken and I want to start fresh without re-running the wizard."
   ```go
   func (r *Repairer) resetAll() (*RepairResult, error) {
       // Backup everything
       for _, f := range r.State.Files {
           r.backup(f.Path)
       }
       // Re-run generation pipeline with saved answers
       return r.regenerateAll(r.Answers)
   }
   ```
8. Register `qsdev repair` command with Cobra:
   ```go
   var repairCmd = &cobra.Command{
       Use:   "repair",
       Short: "Fix corrupted or drifted gdev-managed files",
       Long:  "Detects and fixes issues identified by qsdev devenv doctor. Conservative by default: only fixes unambiguously safe issues. Use --force for aggressive repair, --reset to regenerate everything.",
   }
   repairCmd.Flags().Bool("dry-run", false, "Preview what would be fixed without making changes")
   repairCmd.Flags().Bool("force", false, "Fix files even when user modifications detected (backup first)")
   repairCmd.Flags().String("file", "", "Repair a specific file only")
   repairCmd.Flags().Bool("reset", false, "Regenerate all files from saved answers (nuclear option)")
   ```
9. Implement exit codes: 0 = all issues fixed, 1 = some issues require manual action, 2 = repair failed.
10. Wire repair results back into state tracking: after successful repair, update `GeneratedState` with new file hashes so subsequent `qsdev devenv doctor` runs show clean health.

**Acceptance Criteria:**
- [ ] `qsdev repair` fixes corrupted machine-owned files without user intervention
- [ ] Mandatory backup created before any file is modified (verify `.gdev/backups/` populated)
- [ ] User-modified files are NOT overwritten without `--force` flag
- [ ] devenv.nix is NEVER auto-modified — always generates `.devenv.nix.new` + diff
- [ ] `--dry-run` shows planned actions without writing any files
- [ ] `--file <path>` repairs only the specified file
- [ ] `--reset` regenerates all files from saved answers with full backup
- [ ] Pre-commit hooks are reinstalled when missing or outdated
- [ ] `.gitignore` entries are appended when missing (existing entries preserved)
- [ ] After repair, `qsdev devenv doctor` reports clean health for all repaired files
- [ ] Exit code 0 when all issues fixed, 1 when manual action needed, 2 on failure
- [ ] `GeneratedState` updated with new hashes after successful repair

**Research Citations:**
- `research-spikes/gdev-dx-polish/error-recovery-research.md` — doctor/repair design, failure category taxonomy, auto-fix rules, backup strategy
- `research-spikes/gdev-dx-polish/error-recovery-research.md § Key Design Principles` — "doctor is read-only, repair is write," conservative default, always backup
- `phases/08-migration-update-polish.md § Unit 6.1` — hash-based modification detection, `GeneratedState` tracking
- `phases/08-migration-update-polish.md § Unit 6.4` — devenv.nix update strategy (never auto-overwrite)
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Core Principle` — SHA256 hash tracking as foundation

**Status:** Not Started

---

### Unit 16.2: gdev info Command

**Description:** Implement `qsdev info` as a lightweight, instant project status command that reads only cached state files (no evaluation, no network, no scanning). Provides a one-screen summary, a single-line mode for scripting, and JSON output for machine consumption.

**Code-Grounded Implementation Note:** The data sources for `qsdev info` already exist and are fast reads. Answers are at `.devinit/.qsdev-init-answers.yaml` (loaded via `loadAnswers()` at `addons/devinit/answers.go:50-67`). Detection runs via `detect.Detect()` at `internal/detect/detect.go:12-83` but is not needed here — info reads cached results only. The state file provides last-run time via `GeneratedState.LastRun`. Both answers and state files are small YAML reads that complete well under 100ms.

**Context:** `qsdev devenv doctor` (Phase 9/15) performs active health checks — evaluating devenv, verifying tool availability, checking file hashes. This is thorough but takes seconds. Developers frequently need a faster answer to "where am I? what's active?" — especially when switching between client projects. `qsdev info` fills this gap by reading only `.qsdev.yaml` and `.gdev/state.yaml`, producing instant output without evaluation.

This is the shell integration counterpart: when a developer enters a gdev-managed project, `qsdev info --oneline` could power a prompt segment or an enterShell notification. The command is deliberately simple — no file scanning, no hash checking, no network requests. If files don't exist, it reports "not a gdev project" and exits.

**Desired Outcome:** `qsdev info` responds in under 100ms with project name, detected ecosystems, security level, gdev version, and last updated date. Subsecond response even on cold start.

**Steps:**
1. Create `internal/info/` package with `Info` struct:
   ```go
   type ProjectInfo struct {
       ProjectName     string   `json:"project_name"`
       Ecosystems      []string `json:"ecosystems"`
       ActiveToolCount int      `json:"active_tool_count"`
       SecurityProfile string   `json:"security_profile"`
       GdevVersion     string   `json:"gdev_version"`
       ConfigVersion   string   `json:"config_version"`
       LastUpdated     string   `json:"last_updated"`
   }
   ```
2. Implement `ReadProjectInfo()` — reads `.qsdev.yaml` for project config and `.gdev/state.yaml` for state metadata. No file scanning, no evaluation. If neither file exists, return `ErrNotGdevProject`.
3. Implement default (multi-line) output format:
   ```
   $ gdev info
   Project:    acme-frontend
   Ecosystems: TypeScript (pnpm), Docker
   Tools:      12 active (6 security, 3 devex, 2 ai-agent, 1 infrastructure)
   Security:   Enhanced (consulting-default profile)
   gdev:       v0.16.2 (config v0.16.0)
   Updated:    2026-05-09 (3 days ago)
   ```
4. Implement `--oneline` flag for single-line output suitable for prompts and scripts:
   ```
   $ gdev info --oneline
   TypeScript/Docker | 12 tools | Enhanced | v0.16.2 | updated 3d ago
   ```
5. Implement `--json` flag for machine-readable output:
   ```
   $ gdev info --json
   {"project_name":"acme-frontend","ecosystems":["typescript","docker"],...}
   ```
6. Register `qsdev info` command with Cobra:
   ```go
   var infoCmd = &cobra.Command{
       Use:   "info",
       Short: "Show project status at a glance",
       Long:  "Displays project name, ecosystems, tool count, security level, and version. Instant response — reads cached state only, no evaluation.",
   }
   infoCmd.Flags().Bool("oneline", false, "Single-line output for prompts and scripts")
   infoCmd.Flags().Bool("json", false, "JSON output for machine consumption")
   ```
7. Implement relative time formatting for "last updated" field: "just now", "2 hours ago", "3 days ago", "2 weeks ago", "3 months ago". Use `time.Since()` with human-friendly bucketing.
8. Handle edge cases:
   - Not in a gdev project: print "Not a gdev-managed project. Run `qsdev init` to set up." and exit 1.
   - `.qsdev.yaml` exists but `.gdev/state.yaml` missing: print partial info with "(state unknown — run `qsdev devenv doctor`)" for missing fields.
   - Config version newer than binary version: add "(update available)" annotation.

**Acceptance Criteria:**
- [ ] `qsdev info` displays project name, ecosystems, tool count, security profile, version, and last updated
- [ ] Response time under 100ms (reads only two YAML files, no evaluation)
- [ ] `--oneline` produces single-line output suitable for shell prompt integration
- [ ] `--json` produces valid JSON with all fields
- [ ] Non-gdev directory prints actionable error and exits 1
- [ ] Partial state (missing state file) shows available info with degradation warning
- [ ] Relative time formatting is human-friendly ("3 days ago" not "2026-05-09T14:32:00Z")
- [ ] Version mismatch between binary and config annotated with "(update available)"

**Research Citations:**
- `research-spikes/gdev-dx-polish/shell-integration-research.md § Quick-Info Commands` — `qsdev info` design, output mockup, subsecond response requirement
- `research-spikes/gdev-dx-polish/shell-integration-research.md § Summary` — include `qsdev info` as lightweight status command
- `research-spikes/gdev-dx-polish/environment-switching-research.md § Gap 4` — cross-project status visibility for consultants

**Status:** Not Started

---

### Unit 16.3: gdev outdated Command

**Description:** Implement `qsdev outdated` as a thin wrapper that runs each detected ecosystem's native outdated command sequentially, printing results with ecosystem headers and returning a non-zero exit code if any ecosystem reports outdated packages.

**Context:** Every package manager has its own outdated command (`npm outdated`, `pip list --outdated`, `go list -m -u all`, etc.), but no tool runs ALL of them across a polyglot project. A TypeScript + Python + Docker project requires three separate commands with three different output formats. `qsdev outdated` fills the polyglot gap by iterating detected ecosystems and running each native command, providing a single "are any of my dependencies outdated?" answer.

This is deliberately a thin wrapper, not a unified analysis platform. It does NOT parse or normalize output formats, track versions, or duplicate Renovate's analysis. The value is "one command to check everything" — approximately 50 lines of Go per ecosystem (detect, exec, print, check exit code). Renovate handles the CI-side continuous monitoring; `qsdev outdated` handles the interactive developer question.

**Desired Outcome:** `qsdev outdated` runs all applicable ecosystem outdated commands and reports results with clear ecosystem separation. Exit code 0 if everything is current, 1 if any outdated packages found.

**Steps:**
1. Create `internal/outdated/` package with `Checker` struct:
   ```go
   type Checker struct {
       Ecosystems []string // detected ecosystems from .qsdev.yaml
       Filter     string   // --ecosystem flag value, empty = all
   }

   type EcosystemCheck struct {
       Name     string
       Command  string
       Args     []string
       ExitCode int
       Output   string
   }
   ```
2. Define the ecosystem-to-command mapping:
   ```go
   var outdatedCommands = map[string][]string{
       "npm":      {"npm", "outdated"},
       "pnpm":     {"pnpm", "outdated"},
       "yarn":     {"yarn", "outdated"},
       "pip":      {"pip", "list", "--outdated"},
       "uv":       {"uv", "pip", "list", "--outdated"},
       "go":       {"go", "list", "-m", "-u", "all"},
       "cargo":    {"cargo", "outdated"},
       "dotnet":   {"dotnet", "list", "package", "--outdated"},
       "composer": {"composer", "outdated"},
       "bundler":  {"bundle", "outdated"},
       "mix":      {"mix", "hex.outdated"},
       "maven":    {"mvn", "versions:display-dependency-updates"},
       "gradle":   {"gradle", "dependencyUpdates"},
   }
   ```
3. Implement the check loop: for each detected ecosystem, check if the command binary exists in PATH (skip with warning if not), then run the command and capture output:
   ```go
   func (c *Checker) Run() ([]EcosystemCheck, error) {
       var results []EcosystemCheck
       for _, eco := range c.Ecosystems {
           if c.Filter != "" && c.Filter != eco {
               continue
           }
           cmdSpec, ok := outdatedCommands[eco]
           if !ok {
               continue // no outdated command for this ecosystem
           }
           if _, err := exec.LookPath(cmdSpec[0]); err != nil {
               fmt.Fprintf(os.Stderr, "=== %s === (skipped: %s not found)\n", eco, cmdSpec[0])
               continue
           }
           fmt.Printf("=== %s ===\n", eco)
           cmd := exec.Command(cmdSpec[0], cmdSpec[1:]...)
           cmd.Stdout = os.Stdout
           cmd.Stderr = os.Stderr
           exitCode := 0
           if err := cmd.Run(); err != nil {
               if exitErr, ok := err.(*exec.ExitError); ok {
                   exitCode = exitErr.ExitCode()
               }
           }
           results = append(results, EcosystemCheck{Name: eco, ExitCode: exitCode})
           fmt.Println()
       }
       return results, nil
   }
   ```
4. Implement exit code logic: exit 0 if all ecosystem commands exit 0 (or no ecosystems detected), exit 1 if any ecosystem command reports outdated packages. Note: different tools use different exit codes for "outdated found" vs "error" — `npm outdated` exits 1 when outdated packages exist, `pip list --outdated` exits 0 regardless. Handle per-ecosystem:
   ```go
   // npm/pnpm/yarn: exit 1 = outdated found (not an error)
   // pip/uv: exit 0 always, check if output is non-empty
   // go: exit 0 always, check if output contains "["  (update available marker)
   // cargo outdated: exit 1 = outdated found
   ```
5. Implement `--ecosystem` flag to check only a specific ecosystem:
   ```
   $ gdev outdated --ecosystem npm
   === npm ===
   Package    Current  Wanted  Latest
   lodash     4.17.20  4.17.21 4.17.21
   ```
6. Register `qsdev outdated` command with Cobra:
   ```go
   var outdatedCmd = &cobra.Command{
       Use:   "outdated",
       Short: "Check for outdated dependencies across all ecosystems",
       Long:  "Runs each ecosystem's native outdated command and reports results. Thin wrapper — output is native tool format.",
   }
   outdatedCmd.Flags().String("ecosystem", "", "Check only a specific ecosystem (e.g., npm, pip, go)")
   ```
7. Handle the case where no ecosystems are detected: print "No ecosystems detected. Run `qsdev init` to configure." and exit 0.

**Acceptance Criteria:**
- [ ] `qsdev outdated` runs the correct native command for each detected ecosystem
- [ ] Output includes ecosystem headers (`=== npm ===`) for clear separation
- [ ] Ecosystems with missing binary are skipped with a warning (not a hard error)
- [ ] Exit code 0 when all dependencies up-to-date, 1 when outdated packages found
- [ ] `--ecosystem npm` runs only the npm outdated command
- [ ] Per-ecosystem exit code semantics handled correctly (npm exit 1 = outdated, not error)
- [ ] No output parsing or normalization — native tool output passed through directly
- [ ] No ecosystems detected produces helpful message and exits 0
- [ ] Command completes in reasonable time (sequential execution, no timeouts by default)

**Research Citations:**
- `research-spikes/gdev-dx-polish/dependency-freshness-research.md § Current Landscape` — per-ecosystem outdated commands table
- `research-spikes/gdev-dx-polish/dependency-freshness-research.md § Verdict` — "include a thin wrapper, not a full aggregator"
- `research-spikes/gdev-dx-polish/dependency-freshness-research.md § The Polyglot Gap` — no existing tool runs all outdated commands
- `research-spikes/gdev-dx-polish/what-not-to-include-research.md § Test 1` — integrate with existing tools rather than reimplement

**Status:** Not Started

---

### Unit 16.4: qsdev update Command

**Description:** Implement `qsdev update` as a coordinated three-stage update command: (1) self-update the gdev binary, (2) regenerate configs from saved answers with new templates, (3) update devenv flake inputs. Supports partial updates via flags and includes preview/rollback capabilities.

**Code-Grounded Implementation Note:** `qsdev init --update` already exists and does most of what this unit describes for Stage 2. The `runUpdate()` function at `addons/devinit/update.go:60-178` orchestrates: load answers, refresh detection, load state, check modifications, generate new files, build plan, execute plan, save state. Phase 16 adds three things on top: (1) make `qsdev update` a top-level command that delegates to `runUpdate()` for Stage 2, (2) add a self-update stage (Stage 1) before config regeneration, (3) add `devenv update` for flake inputs (Stage 3). The existing `runUpdate()` handles steps 2-9 of the update flow — this unit wraps it with the self-update preamble and devenv postamble.

**Context:** When gdev itself is updated, three separate operations need to happen: the binary updates (Phase 10 self-update), generated configs need regeneration with new templates (Phase 8 update workflow), and devenv inputs need refreshing (`devenv update`). Today these are three separate commands (`qsdev self-update`, `qsdev init --update`, `devenv update`). `qsdev update` coordinates all three into a single safe operation with stage-level granularity.

Critically, `qsdev update` does NOT update application dependencies (npm, pip, cargo packages). Application dependency updates are Renovate's domain and dangerous for unattended execution. The `qsdev update` scope is strictly gdev infrastructure: the binary, the generated configs, and the Nix/devenv inputs.

**Desired Outcome:** `qsdev update` brings all gdev-managed infrastructure current in one command. Failed stages roll back cleanly. Partial update flags allow updating only what's needed.

**Steps:**
1. Create `internal/update/` package with `Updater` struct:
   ```go
   type Updater struct {
       CurrentVersion string
       SelfUpdater    *selfupdate.Updater  // from Phase 10
       ConfigUpdater  *migration.Updater   // from Phase 8
       DryRun         bool
       SelfOnly       bool
       ConfigsOnly    bool
       DepsOnly       bool
   }

   type UpdateResult struct {
       SelfUpdate   *StageResult
       ConfigUpdate *StageResult
       DepsUpdate   *StageResult
   }

   type StageResult struct {
       Stage    string
       Status   string // "updated", "skipped", "failed", "up-to-date"
       Details  string
       Rollback func() error
   }
   ```
2. Implement Stage 1 — Self-update: reuse `internal/selfupdate/` from Phase 10 (Unit 10.4). Check GitHub Releases for newer version, download, verify SHA256, replace binary. If already current, report "up-to-date" and skip.
   ```go
   func (u *Updater) selfUpdate() (*StageResult, error) {
       release, err := u.SelfUpdater.CheckForUpdate(u.CurrentVersion)
       if err != nil {
           return &StageResult{Stage: "self-update", Status: "failed", Details: err.Error()}, err
       }
       if release == nil {
           return &StageResult{Stage: "self-update", Status: "up-to-date"}, nil
       }
       fmt.Printf("[1/3] Updating gdev %s → %s...\n", u.CurrentVersion, release.Version)
       if u.DryRun {
           return &StageResult{Stage: "self-update", Status: "skipped", Details: "dry-run"}, nil
       }
       if err := u.SelfUpdater.DoUpdate(release); err != nil {
           return &StageResult{Stage: "self-update", Status: "failed", Details: err.Error()}, err
       }
       return &StageResult{Stage: "self-update", Status: "updated", Details: release.Version}, nil
   }
   ```
3. Implement Stage 2 — Config regeneration: reuse `qsdev init --update` workflow from Phase 8 (Unit 6.1). Load saved answers from `.qsdev.yaml`, regenerate files with current templates, apply merge strategies (three-way for JSON, section markers for CLAUDE.md, `.new` + diff for devenv.nix). Track which files changed for the summary.
   ```go
   func (u *Updater) configUpdate() (*StageResult, error) {
       fmt.Println("[2/3] Regenerating configs with latest templates...")
       if u.DryRun {
           // Show diffs without writing
           diffs, err := u.ConfigUpdater.PreviewUpdate()
           if err != nil {
               return &StageResult{Stage: "config-update", Status: "failed"}, err
           }
           for _, d := range diffs {
               fmt.Printf("  %s: %s\n", d.File, d.Summary)
           }
           return &StageResult{Stage: "config-update", Status: "skipped", Details: "dry-run"}, nil
       }
       result, err := u.ConfigUpdater.RunUpdate()
       if err != nil {
           return &StageResult{
               Stage: "config-update", Status: "failed",
               Rollback: result.Rollback,
           }, err
       }
       return &StageResult{
           Stage:   "config-update",
           Status:  "updated",
           Details: fmt.Sprintf("%d files updated", result.FilesChanged),
       }, nil
   }
   ```
4. Implement Stage 3 — devenv input update: run `devenv update` to refresh flake inputs (nixpkgs, devenv itself, any custom inputs in devenv.yaml). Capture output to show which inputs changed.
   ```go
   func (u *Updater) depsUpdate() (*StageResult, error) {
       fmt.Println("[3/3] Updating devenv inputs...")
       if u.DryRun {
           return &StageResult{Stage: "deps-update", Status: "skipped", Details: "dry-run"}, nil
       }
       cmd := exec.Command("devenv", "update")
       cmd.Stdout = os.Stdout
       cmd.Stderr = os.Stderr
       if err := cmd.Run(); err != nil {
           return &StageResult{Stage: "deps-update", Status: "failed", Details: err.Error()}, err
       }
       return &StageResult{Stage: "deps-update", Status: "updated"}, nil
   }
   ```
5. Implement rollback on failure: if Stage 2 (config regeneration) fails, restore backed-up files. If Stage 3 fails, the devenv.lock can be restored from git (`git checkout devenv.lock`). Stage 1 failure uses the self-update rollback from Phase 10 (backup binary restore).
6. Implement `--dry-run` flag: check for available updates and show diffs for config changes without applying anything:
   ```
   $ qsdev update --dry-run
   [1/3] Self-update: v0.16.2 → v0.17.0 available
         New features: qsdev repair, gdev info, 3 new tool integrations
   [2/3] Config regeneration: 3 files would change
         settings.json: 2 new deny rules added
         CLAUDE.md: security section updated
         .pre-commit-config.yaml: hook versions bumped
   [3/3] devenv inputs: run 'devenv update' to check (not checked in dry-run)
   ```
7. Implement partial update flags:
   - `--self-only`: run only Stage 1 (equivalent to `qsdev self-update`)
   - `--configs-only`: run only Stage 2 (equivalent to `qsdev init --update`)
   - `--deps-only`: run only Stage 3 (equivalent to `devenv update`)
8. Implement version bump notification: after self-update, if the new version has notable changes, print a summary sourced from the GitHub Release body:
   ```
   gdev v0.16.2 → v0.17.0
   ├── 3 new tools available (qsdev list --category new)
   ├── 2 template updates (will apply in config regeneration)
   └── 1 deprecation (ripsecrets standalone — now bundled with gitleaks)
   ```
9. Register `qsdev update` command with Cobra:
   ```go
   var updateCmd = &cobra.Command{
       Use:   "update",
       Short: "Update gdev binary, configs, and devenv inputs",
       Long:  "Coordinated three-stage update: self-update binary, regenerate configs, refresh devenv inputs. Does NOT update application dependencies (use Renovate).",
   }
   updateCmd.Flags().Bool("dry-run", false, "Preview what would change without applying updates")
   updateCmd.Flags().Bool("self-only", false, "Update only the gdev binary")
   updateCmd.Flags().Bool("configs-only", false, "Regenerate only gdev-managed config files")
   updateCmd.Flags().Bool("deps-only", false, "Update only devenv flake inputs")
   ```
10. Ensure `qsdev update` is idempotent: running twice in a row should report "up-to-date" for all stages on the second run.

**Acceptance Criteria:**
- [ ] `qsdev update` runs all three stages in order: self-update, config regeneration, devenv inputs
- [ ] Stage 1 reuses Phase 10 self-update with SHA256 verification
- [ ] Stage 2 reuses Phase 8 update workflow with merge strategies (three-way, section markers, `.new` + diff)
- [ ] Stage 3 runs `devenv update` and reports which inputs changed
- [ ] `--dry-run` previews all changes without writing anything
- [ ] `--self-only`, `--configs-only`, `--deps-only` flags run only the specified stage
- [ ] Failed config regeneration rolls back to backed-up files
- [ ] Failed self-update restores previous binary
- [ ] Version bump notification shows notable changes from release notes
- [ ] Running `qsdev update` twice is idempotent (second run reports up-to-date)
- [ ] Application dependencies are explicitly NOT updated (documented in help text)

**Research Citations:**
- `research-spikes/gdev-dx-polish/dependency-freshness-research.md § Coordinated Updates` — three-stage update design, stages 1-3 only (no app deps)
- `research-spikes/gdev-dx-polish/dependency-freshness-research.md § Analysis` — "include `qsdev update` for steps 1-3 only"
- `phases/10-distribution-self-bootstrapping.md § Unit 10.4` — self-update mechanism, `CheckForUpdate`, `DoUpdate`, rollback
- `phases/08-migration-update-polish.md § Unit 6.1` — update command, modification detection, merge strategies

**Status:** Not Started

---

### Unit 16.5: gdev teardown Command

**Description:** Implement `qsdev teardown` for clean project exit with three profiles (quick, default, compliance), user-modification preservation, interactive confirmation, and optional archive creation. Designed for consultants finishing client engagements who need to cleanly decommission a project's qsdev devenv setup.

**Code-Grounded Implementation Note:** The state file at `.devinit/.qsdev-init-state.yaml` tracks all generated files via `GeneratedState.Files map[string]FileState`. The `state.CheckModified()` function at `internal/state/state.go:46-94` identifies which files have been user-modified (hash mismatch). Teardown uses these to determine safe-to-delete (unmodified, hash matches) vs warn-before-delete (modified, hash mismatch) files. The existing infrastructure means teardown does not need its own file-tracking mechanism — it reads the same state that repair and update already use.

**Context:** When a consulting engagement ends or a developer leaves a project, gdev-generated artifacts need cleanup. Without a teardown command, developers must manually identify and remove generated files — `.envrc`, `.pre-commit-config.yaml`, settings.json entries, CLAUDE.md sections, `.gdev/` state directory, etc. This is error-prone: leaving stale configs behind causes confusion for future developers; accidentally deleting user-customized files loses work.

The teardown command respects the same file ownership and hash tracking used by repair and update. Files that the developer has modified (hash mismatch) are flagged and preserved by default — the developer explicitly decides whether to keep or remove them. For compliance-sensitive engagements, teardown generates an evidence report (reusing Phase 15 health reporting) before removing anything, creating an audit trail.

**Desired Outcome:** `qsdev teardown` cleanly removes gdev from a project while preserving user work, with profile-appropriate behavior for casual exit, standard exit, and compliance-mandated exit.

**Steps:**
1. Create `internal/teardown/` package with `Teardown` struct:
   ```go
   type Profile string
   const (
       ProfileQuick      Profile = "quick"
       ProfileDefault    Profile = "default"
       ProfileCompliance Profile = "compliance"
   )

   type Teardown struct {
       State   *state.GeneratedState
       Profile Profile
       Force   bool // non-interactive
       Archive bool // create archive before removal
   }

   type TeardownPlan struct {
       Remove   []FileAction // files to delete
       Preserve []FileAction // user-modified files to keep
       Warn     []FileAction // files needing attention
   }

   type FileAction struct {
       Path     string
       Reason   string // "generated-unmodified", "user-modified", "state-directory"
       Modified bool
   }
   ```
2. Implement profile behaviors:
   - **Quick** (`qsdev teardown --quick`): Remove `.gdev/` state directory only. Leave all generated configs in place. Use case: "I want to stop using gdev tooling but keep the configs it generated."
   - **Default** (`qsdev teardown`): Remove `.gdev/` state directory. Remove generated-and-unmodified files (hash match). Warn about user-modified files and list them. Do NOT remove user-modified files without explicit confirmation.
   - **Compliance** (`qsdev teardown --compliance`): Generate Phase 15 evidence report to `.gdev/teardown-report-<date>.json`. Then execute default teardown. Archive `.qsdev.yaml` to `.gdev-archive/` for re-engagement. Use case: "Engagement over, need audit trail."
3. Implement file classification using `GeneratedState` hashes:
   ```go
   func (t *Teardown) classifyFiles() (*TeardownPlan, error) {
       plan := &TeardownPlan{}
       for path, fileState := range t.State.Files {
           currentHash, err := hashFile(path)
           if err != nil {
               // File already deleted — skip
               continue
           }
           if currentHash == fileState.Hash {
               plan.Remove = append(plan.Remove, FileAction{
                   Path: path, Reason: "generated-unmodified",
               })
           } else {
               plan.Preserve = append(plan.Preserve, FileAction{
                   Path: path, Reason: "user-modified", Modified: true,
               })
           }
       }
       // Always remove .gdev/ state directory
       plan.Remove = append(plan.Remove, FileAction{
           Path: ".gdev/", Reason: "state-directory",
       })
       return plan, nil
   }
   ```
4. Implement shared file cleanup for default/compliance profiles: for shared files (settings.json, CLAUDE.md, .mcp.json), remove only gdev-owned sections/keys rather than deleting the entire file. Use the same section marker and JSON surgery from Phase 8/12:
   ```go
   func (t *Teardown) cleanSharedFile(path string, ownership FileOwnership) error {
       switch ownership.Format {
       case "json":
           return t.removeJSONKeys(path, ownership.Keys)
       case "markdown-markers":
           return t.removeMarkedSections(path, ownership.Markers)
       case "yaml":
           return t.removeYAMLKeys(path, ownership.Keys)
       }
       return nil
   }
   ```
5. Implement interactive confirmation: before executing any removals, display the plan and prompt for confirmation:
   ```
   $ gdev teardown
   gdev Teardown Plan
   ==================

   Will remove (generated, unmodified):
     .envrc
     devenv.yaml
     .pre-commit-config.yaml
     .github/pull_request_template.md
     .github/labeler.yml
     .gdev/ (state directory)

   Will clean (remove gdev sections only):
     settings.json (removing 48 deny rules, 3 hooks — keeping user additions)
     CLAUDE.md (removing generated section — keeping user content)
     .mcp.json (removing 3 gdev-managed servers — keeping user servers)

   Will preserve (user-modified — not touching):
     devenv.nix (modified by user)
     .semgrep.yml (modified by user)

   Proceed? [y/N]
   ```
6. Implement `--force` flag for non-interactive teardown (CI use, scripting):
   ```go
   if !t.Force {
       if !promptConfirm("Proceed with teardown?") {
           return nil // cancelled
       }
   }
   ```
7. Implement `--archive` flag: create `.gdev-archive.tar.gz` containing `.qsdev.yaml`, `.gdev/state.yaml`, and all generated configs before removal. Enables re-engagement — `qsdev init --from-archive .gdev-archive.tar.gz` could restore the setup later.
   ```go
   func (t *Teardown) createArchive(plan *TeardownPlan) error {
       archivePath := fmt.Sprintf(".gdev-archive-%s.tar.gz",
           time.Now().Format("20060102"))
       // Include .qsdev.yaml, .gdev/state.yaml, all generated files
       files := []string{".qsdev.yaml", ".gdev/state.yaml"}
       for _, f := range plan.Remove {
           files = append(files, f.Path)
       }
       return createTarGz(archivePath, files)
   }
   ```
8. Implement compliance evidence generation: reuse Phase 15 health reporting to produce a final snapshot before teardown:
   ```go
   func (t *Teardown) generateEvidence() error {
       report := health.GenerateReport(t.State)
       reportPath := fmt.Sprintf(".gdev/teardown-report-%s.json",
           time.Now().Format("20060102-150405"))
       return writeJSON(reportPath, report)
   }
   ```
9. Register `qsdev teardown` command with Cobra:
   ```go
   var teardownCmd = &cobra.Command{
       Use:   "teardown",
       Short: "Remove gdev from this project",
       Long:  "Clean project exit. Removes gdev state and generated files while preserving user modifications. Three profiles: --quick (state only), default (state + unmodified), --compliance (evidence report + default + archive).",
   }
   teardownCmd.Flags().Bool("quick", false, "Remove only .gdev/ state directory, keep generated configs")
   teardownCmd.Flags().Bool("compliance", false, "Generate evidence report, archive config, then remove")
   teardownCmd.Flags().Bool("force", false, "Non-interactive mode (skip confirmation prompt)")
   teardownCmd.Flags().Bool("archive", false, "Create .gdev-archive.tar.gz before removal")
   ```
10. Post-teardown cleanup: after removing files, check if any gdev-generated directories are now empty (`.github/`, `.claude/skills/`) and remove them if so. Do NOT remove directories that contain non-gdev files.

**Acceptance Criteria:**
- [ ] `qsdev teardown --quick` removes only `.gdev/` state directory, all generated configs preserved
- [ ] `qsdev teardown` (default) removes unmodified generated files, preserves user-modified files
- [ ] `qsdev teardown --compliance` generates evidence report, then executes default teardown, archives `.qsdev.yaml`
- [ ] User-modified files are NEVER silently deleted (hash comparison required)
- [ ] Shared files cleaned surgically — only gdev-owned sections/keys removed, user additions preserved
- [ ] Interactive confirmation shows full plan before any deletion
- [ ] `--force` skips confirmation prompt for CI/scripting use
- [ ] `--archive` creates `.gdev-archive.tar.gz` with all gdev-managed files before removal
- [ ] Empty gdev-generated directories cleaned up after file removal
- [ ] After default teardown, project has no gdev artifacts except user-modified files and devenv.nix

**Research Citations:**
- `research-spikes/gdev-dx-polish/research.md § What "Polished" Actually Means` — teardown as completion of the gdev lifecycle
- `research-spikes/gdev-dx-polish/environment-switching-research.md § Gap 5` — credential cleanup context for compliance teardown
- `phases/08-migration-update-polish.md § Unit 6.1` — `GeneratedState` hash tracking used for modification detection
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — file ownership registry, shared file surgery

**Status:** Not Started

---

### Unit 16.6: Git Workflow Automation

**Description:** Implement lifecycle-managed git workflow automation: PR template generation with ecosystem-aware content, branch naming convention enforcement via git hook, commit ticket extraction from branch names, and automated PR labeling via GitHub Actions labeler workflow. All artifacts are managed through the Phase 12 tool lifecycle system.

**Context:** gdev already generates pre-commit hooks (Phase 5), commitlint (Phase 12), and git-cliff (Phase 12). This unit fills the remaining git workflow gaps identified in research: PR templates that include the right checklists for detected ecosystems, branch naming enforcement to prevent garbage branch names, auto-extraction of ticket IDs from branch names into commit messages, and automated PR labels based on file paths and commit types.

Every artifact generated by this unit is lifecycle-managed through the Phase 12 tool registry. `qsdev enable pr-templates` adds the PR template; `qsdev disable pr-templates` removes it. This allows teams to adopt individual git workflow features without taking all of them.

**Desired Outcome:** A new gdev project gets ecosystem-appropriate PR templates, branch naming enforcement, and PR labeling with zero manual configuration. Each feature is individually toggleable.

**Steps:**
1. Register four git workflow tools in the Phase 12 tool registry:
   ```go
   // Tool: pr-templates
   Tool{
       Name:        "pr-templates",
       DisplayName: "PR Template Generator",
       Category:    "devex",
       Default:     AlwaysOn,
       OwnedFiles: []FileOwnership{
           {Path: ".github/pull_request_template.md", Ownership: Exclusive},
       },
   }

   // Tool: branch-naming
   Tool{
       Name:        "branch-naming",
       DisplayName: "Branch Naming Convention",
       Category:    "devex",
       Default:     AlwaysOn,
       OwnedFiles: []FileOwnership{
           // Hook contributed to devenv.nix git-hooks section
           {Path: "devenv.nix", Ownership: Shared, SectionID: "branch-naming"},
       },
   }

   // Tool: commit-ticket
   Tool{
       Name:        "commit-ticket",
       DisplayName: "Commit Ticket Extraction",
       Category:    "devex",
       Default:     OptIn, // requires ticket pattern configuration
       OwnedFiles: []FileOwnership{
           {Path: "devenv.nix", Ownership: Shared, SectionID: "commit-ticket"},
       },
   }

   // Tool: pr-labels
   Tool{
       Name:        "pr-labels",
       DisplayName: "Automated PR Labels",
       Category:    "devex",
       Default:     AlwaysOn,
       OwnedFiles: []FileOwnership{
           {Path: ".github/labeler.yml", Ownership: Exclusive},
           {Path: ".github/workflows/labeler.yml", Ownership: Exclusive},
       },
   }
   ```
2. Implement PR template generation — content varies by detected ecosystems:
   - **Base template** (all projects): Summary, Type of change (checkboxes: feature, fix, refactor, docs, chore), Breaking changes, Reviewer notes.
   - **Security section** (when security hardening enabled): "Security Checklist" with items: no secrets in code, dependency versions pinned, SAST scan passed, new endpoints authenticated.
   - **Testing section** (when test framework detected): "Testing Checklist" with items: unit tests added/updated, integration tests updated, manual testing performed.
   - **Per-ecosystem additions**: Python → "type hints added," Go → "linter passes," TypeScript → "types exported," Rust → "clippy clean," Docker → "image scanned."
   ```go
   func generatePRTemplate(ecosystems []string, securityEnabled bool) string {
       var b strings.Builder
       b.WriteString("## Summary\n\n<!-- What changed and why -->\n\n")
       b.WriteString("## Type of Change\n\n")
       b.WriteString("- [ ] Feature\n- [ ] Bug fix\n- [ ] Refactor\n- [ ] Documentation\n- [ ] Chore\n\n")
       if securityEnabled {
           b.WriteString("## Security Checklist\n\n")
           b.WriteString("- [ ] No secrets or credentials in code\n")
           b.WriteString("- [ ] Dependency versions pinned\n")
           b.WriteString("- [ ] SAST scan passes\n")
           b.WriteString("- [ ] New endpoints require authentication\n\n")
       }
       // ... ecosystem-specific sections
       return b.String()
   }
   ```
3. Implement branch naming enforcement as a devenv git hook. Default pattern: `^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$`. Pattern is configurable via `.qsdev.yaml`:
   ```yaml
   # .qsdev.yaml
   git:
     branch_pattern: "^(feat|fix|chore|docs|refactor|test|ci)/[A-Z]+-[0-9]+-[a-z0-9-]+$"
   ```
   Hook implementation in devenv.nix:
   ```nix
   # --- branch-naming ---
   git-hooks.hooks.branch-naming = {
     enable = true;
     entry = ''
       branch=$(git rev-parse --abbrev-ref HEAD)
       pattern="^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$"
       if [[ "$branch" == "main" || "$branch" == "master" || "$branch" == "develop" ]]; then
         exit 0
       fi
       if ! [[ "$branch" =~ $pattern ]]; then
         echo "Branch name '$branch' does not match convention: $pattern"
         echo "Examples: feat/add-login, fix/null-pointer, chore/update-deps"
         exit 1
       fi
     '';
     stages = ["pre-push"];
   };
   # --- end branch-naming ---
   ```
4. Implement commit ticket extraction as a `prepare-commit-msg` hook. Extracts ticket ID from branch name using configurable regex and prepends to commit message:
   ```nix
   # --- commit-ticket ---
   git-hooks.hooks.commit-ticket = {
     enable = true;
     entry = ''
       COMMIT_MSG_FILE=$1
       COMMIT_SOURCE=$2
       # Only prepend for regular commits, not merges/amends
       if [ -n "$COMMIT_SOURCE" ]; then exit 0; fi
       branch=$(git rev-parse --abbrev-ref HEAD)
       ticket=$(echo "$branch" | grep -oP '[A-Z]+-[0-9]+' | head -1)
       if [ -n "$ticket" ]; then
         current=$(cat "$COMMIT_MSG_FILE")
         if ! echo "$current" | grep -q "$ticket"; then
           echo "[$ticket] $current" > "$COMMIT_MSG_FILE"
         fi
       fi
     '';
     stages = ["prepare-commit-msg"];
   };
   # --- end commit-ticket ---
   ```
5. Implement PR labeler configuration — generate `.github/labeler.yml` mapping file paths to labels:
   ```yaml
   documentation:
     - changed-files:
         - any-glob-to-any-file: ['docs/**', '*.md', 'README*']
   infrastructure:
     - changed-files:
         - any-glob-to-any-file: ['devenv.nix', 'devenv.yaml', '.envrc', 'flake.*', 'Dockerfile*', '.github/**']
   security:
     - changed-files:
         - any-glob-to-any-file: ['.semgrep.yml', '.gitleaks.toml', '.scancode.yml', 'security/**']
   # Per-ecosystem labels generated dynamically
   ```
   Generate corresponding `.github/workflows/labeler.yml`:
   ```yaml
   name: PR Labeler
   on:
     pull_request_target:
       types: [opened, synchronize]
   permissions:
     contents: read
     pull-requests: write
   jobs:
     label:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/labeler@v5  # SHA-pinned in actual generation
           with:
             repo-token: "${{ secrets.GITHUB_TOKEN }}"
   ```
6. Wire all four tools into the Phase 12 lifecycle system so `qsdev enable/disable` works for each.
7. Add wizard integration: the customize path shows git workflow options. Quick path uses defaults (pr-templates and branch-naming on, commit-ticket off, pr-labels on).

**Acceptance Criteria:**
- [ ] PR template generated with ecosystem-appropriate checklists (security section only when hardening enabled)
- [ ] Multi-ecosystem project (Go + TypeScript) gets combined checklist items
- [ ] Branch naming hook rejects non-conforming branch names on push, allows main/master/develop
- [ ] Branch naming pattern configurable via `.qsdev.yaml`
- [ ] Commit ticket extraction prepends `[TICKET-123]` to commit messages from branch name
- [ ] Commit ticket extraction skips merge commits and amends
- [ ] PR labeler config generated with ecosystem-appropriate file path mappings
- [ ] PR labeler workflow uses SHA-pinned actions
- [ ] `qsdev enable pr-templates` / `qsdev disable pr-templates` cleanly adds/removes template
- [ ] `qsdev enable branch-naming` / `qsdev disable branch-naming` cleanly adds/removes hook
- [ ] `qsdev enable commit-ticket` / `qsdev disable commit-ticket` cleanly adds/removes hook
- [ ] `qsdev enable pr-labels` / `qsdev disable pr-labels` cleanly adds/removes labeler config and workflow

**Research Citations:**
- `research-spikes/gdev-dx-polish/git-workflow-research.md § Gap Analysis` — four features to include (branch naming, PR templates, commit ticket, PR labels), two to exclude (merge queue, release automation)
- `research-spikes/gdev-dx-polish/git-workflow-research.md § 1. Branch Naming Convention Enforcement` — regex pattern, hook type, default pattern
- `research-spikes/gdev-dx-polish/git-workflow-research.md § 2. PR Template Generation` — ecosystem-aware sections
- `research-spikes/gdev-dx-polish/git-workflow-research.md § 3. Commit Message Ticket Extraction` — prepare-commit-msg hook, opt-in
- `research-spikes/gdev-dx-polish/git-workflow-research.md § 4. Automated PR Labels` — `actions/labeler`, `.github/labeler.yml`
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — tool registry, file ownership, enable/disable commands

**Status:** Not Started

---

### Unit 16.7: Shell & Environment Integration

**Description:** Implement shell and environment integrations that make gdev context visible during daily development: Starship prompt segment generation, gdev environment variables in devenv.nix, enterShell notification, and OTEL environment variable configuration. All integrations are generated into devenv.nix (no separate shell scripts or hooks).

**Code-Grounded Implementation Note:** The `enterShell` content is currently hardcoded in `addons/devenv/security_defaults.go` via `buildEnterShellScript()`. It currently outputs: banner, pre-commit hook check, clean env check, and ripsecrets check. Phase 16 needs to make this extensible — convert from the current hardcoded string concatenation to a template or builder pattern that accepts additional notification lines from other tools/features. The `DEVENV_SECURITY_HARDENED = "true"` env var is set at `addons/devenv/devenv_nix_data.go:67` — new gdev env vars (`QSDEV_PROJECT_NAME`, `QSDEV_SECURITY_PROFILE`, etc.) follow the same pattern of being generated into the devenv.nix env block.

**Context:** Developers switching between client projects need environmental awareness — "which project am I in? what security level? what tools are active?" — without running a command. The research concluded that gdev should NOT add its own shell hook (this duplicates devenv's activation lifecycle and creates ordering conflicts). Instead, all shell integration goes through devenv.nix: environment variables are set unconditionally (useful for any tool), Starship config is opt-in, and the enterShell task provides a one-line notification on shell entry.

The OTEL environment variable configuration is profile-driven and opt-in — consulting firms with client billing needs can enable it to track AI agent session costs. This generates only environment variables pointing at the firm's collector, NOT the collector infrastructure itself (that is explicitly out of scope per the what-not-to-include research).

**Desired Outcome:** Entering a gdev-managed devenv shell shows a brief notification with project context. The shell prompt (if using Starship) shows the gdev security level. Environment variables are available to all tools and scripts.

**Steps:**
1. Implement gdev environment variable generation in the devenv.nix template. These are set unconditionally for all gdev projects — they power the Starship segment, enterShell notification, and any external tooling:
   ```nix
   # --- gdev-env ---
   env.QSDEV_PROJECT_NAME = "acme-frontend";
   env.QSDEV_SECURITY_PROFILE = "enhanced";
   env.QSDEV_VERSION = "0.16.2";
   env.QSDEV_ECOSYSTEMS = "typescript,docker";
   # --- end gdev-env ---
   ```
   Values are populated from `.qsdev.yaml` at generation time. The template function reads `ProjectName`, `SecurityProfile`, the gdev binary version, and the comma-joined ecosystem list.
2. Implement enterShell notification as a devenv.nix task — a brief one-line message on shell entry showing project context:
   ```nix
   # --- gdev-entershell ---
   enterShell = ''
     echo "gdev: $QSDEV_ECOSYSTEMS | $QSDEV_SECURITY_PROFILE security | $(gdev info --oneline 2>/dev/null || echo 'run gdev info')"
   '';
   # --- end gdev-entershell ---
   ```
   The notification is deliberately minimal — one line, no color codes (terminals vary), no blocking commands. It uses the env vars set above, so it adds zero latency (no file reads, no subprocesses beyond the optional `qsdev info --oneline` which itself reads only cached state).
3. Register Starship config generation as an opt-in tool in the lifecycle registry:
   ```go
   Tool{
       Name:        "starship-integration",
       DisplayName: "Starship Prompt Integration",
       Category:    "devex",
       Default:     OptIn,
       OwnedFiles: []FileOwnership{
           {Path: "devenv.nix", Ownership: Shared, SectionID: "starship"},
       },
   }
   ```
4. Implement Starship config generation in devenv.nix. When enabled, generate a `starship.toml` custom module via devenv's native Starship support:
   ```nix
   # --- starship ---
   starship.enable = true;
   starship.config.enable = true;
   starship.config.path = ".starship.toml";

   # Generate .starship.toml with gdev custom modules
   # (file managed as exclusive by starship-integration tool)
   # --- end starship ---
   ```
   Generate `.starship.toml` as an exclusive file owned by the `starship-integration` tool:
   ```toml
   # gdev-managed Starship configuration
   # Extend your existing starship.toml or use this standalone

   [custom.gdev]
   command = "echo $QSDEV_PROJECT_NAME"
   when = 'test -n "$QSDEV_PROJECT_NAME"'
   format = "[$output]($style) "
   style = "bold cyan"
   description = "Active gdev project"

   [custom.gdev_security]
   command = '''
     case "$QSDEV_SECURITY_PROFILE" in
       enhanced) echo "enhanced" ;;
       strict)   echo "strict" ;;
       *)        echo "standard" ;;
     esac
   '''
   when = 'test -n "$QSDEV_SECURITY_PROFILE"'
   format = "[$output]($style) "
   style = "green"
   description = "gdev security profile"

   [custom.gdev_tools]
   command = "echo ${QSDEV_TOOL_COUNT:-?}"
   when = 'test -n "$QSDEV_PROJECT_NAME"'
   format = "[$output tools]($style) "
   style = "dimmed white"
   description = "Active tool count"
   ```
5. Add `QSDEV_TOOL_COUNT` to the env var block — computed from the tool registry at generation time:
   ```nix
   env.QSDEV_TOOL_COUNT = "12"; # count of enabled tools
   ```
6. Implement OTEL environment variable configuration as a profile-driven, opt-in tool:
   ```go
   Tool{
       Name:        "otel-config",
       DisplayName: "OTEL Environment Variables",
       Category:    "infrastructure",
       Default:     OptIn, // profile-driven, not default
       OwnedFiles: []FileOwnership{
           {Path: "devenv.nix", Ownership: Shared, SectionID: "otel-config"},
       },
   }
   ```
   When enabled, generate OTEL env vars in devenv.nix sourced from the infrastructure profile:
   ```nix
   # --- otel-config ---
   env.OTEL_EXPORTER_OTLP_ENDPOINT = "https://otel.acme-corp.internal:4317";
   env.OTEL_EXPORTER_OTLP_PROTOCOL = "grpc";
   env.OTEL_SERVICE_NAME = "claude-code-acme-frontend";
   env.CLAUDE_CODE_ENABLE_TELEMETRY = "1";
   # --- end otel-config ---
   ```
   Values come from the infrastructure profile's OTEL configuration. If no profile provides OTEL config, the tool is not available for enable. The OTEL collector endpoint is infrastructure — gdev generates the env vars pointing at it, nothing more.
7. Wire all three tools (gdev-env is always-on and not a lifecycle tool — it is part of the core devenv template) into the devenv addon template generation pipeline. The generation order is:
   1. Core devenv.nix template (packages, languages, services)
   2. gdev-env section (always-on, not lifecycle-managed)
   3. enterShell notification (always-on, not lifecycle-managed)
   4. Starship section (lifecycle-managed, opt-in)
   5. OTEL section (lifecycle-managed, opt-in)
8. Handle the enterShell composition: devenv.nix supports only one `enterShell` attribute. If other tools also contribute enterShell content, concatenate all contributions:
   ```go
   // In devenv.nix template generation
   func renderEnterShell(contributions []string) string {
       return "enterShell = ''\n" +
           strings.Join(contributions, "\n") +
           "\n'';"
   }
   ```
   The gdev notification is always the LAST line in enterShell so it appears after any tool setup messages.

**Acceptance Criteria:**
- [ ] `QSDEV_PROJECT_NAME`, `QSDEV_SECURITY_PROFILE`, `QSDEV_VERSION`, `QSDEV_ECOSYSTEMS`, `QSDEV_TOOL_COUNT` set in devenv.nix for all gdev projects
- [ ] Environment variables available in devenv shell (verify with `echo $QSDEV_PROJECT_NAME`)
- [ ] enterShell prints one-line gdev notification when entering devenv shell
- [ ] enterShell notification is non-blocking and adds minimal latency (<50ms)
- [ ] `qsdev enable starship-integration` generates `.starship.toml` and enables devenv starship module
- [ ] `qsdev disable starship-integration` removes `.starship.toml` and starship devenv.nix section
- [ ] Starship custom modules display project name, security profile, and tool count
- [ ] `qsdev enable otel-config` generates OTEL env vars from infrastructure profile
- [ ] `qsdev disable otel-config` removes OTEL env vars from devenv.nix
- [ ] OTEL config only available when infrastructure profile provides collector endpoint
- [ ] No separate gdev shell hook — all integration goes through devenv.nix
- [ ] Multiple enterShell contributions from different tools compose correctly (concatenated)

**Research Citations:**
- `research-spikes/gdev-dx-polish/shell-integration-research.md § Starship Prompt Integration` — starship.toml config, env var list, devenv starship module
- `research-spikes/gdev-dx-polish/shell-integration-research.md § Shell Hook for Environment Awareness` — "do NOT add a separate gdev shell hook," use devenv enterShell instead
- `research-spikes/gdev-dx-polish/shell-integration-research.md § Summary` — include/exclude matrix for shell features
- `research-spikes/gdev-dx-polish/research.md § Agentic Session Observability` — OTEL as optional profile-driven config, not default
- `research-spikes/gdev-dx-polish/what-not-to-include-research.md § 7. Full OTEL Infrastructure` — generate env vars only, no collector/Grafana
- `phases/03-devenv-addon-core-generation.md` — devenv.nix template generation pipeline

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] `qsdev repair` fixes all issues detectable by `qsdev devenv doctor` without destroying user customizations
- [ ] `qsdev info` responds in under 100ms with correct project metadata
- [ ] `qsdev outdated` runs correct native commands for all detected ecosystems
- [ ] `qsdev update` coordinates self-update, config regeneration, and devenv input update with rollback on failure
- [ ] `qsdev teardown` cleanly removes gdev artifacts in all three profiles without deleting user-modified files
- [ ] Git workflow tools (PR templates, branch naming, commit tickets, PR labels) are individually lifecycle-manageable
- [ ] Shell integration (env vars, enterShell, Starship, OTEL) generates correctly into devenv.nix
- [ ] All new commands appear in `qsdev --help` with descriptive help text
- [ ] All new lifecycle-managed tools appear in `qsdev list` with correct categories
- [ ] Running `qsdev init` on a fresh project generates default git workflow and shell integration
- [ ] `qsdev enable/disable` works for all Phase 16 lifecycle-managed tools (starship-integration, otel-config, pr-templates, branch-naming, commit-ticket, pr-labels)
- [ ] No feature duplicates existing tooling (devenv tasks, Renovate, existing ecosystem commands)
