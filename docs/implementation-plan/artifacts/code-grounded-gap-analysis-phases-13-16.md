# Code-Grounded Gap Analysis: Phases 13-16

## 1. Executive Summary

Six analysis passes across the entire qsdev codebase (82 first-party Go source files, ~40K lines) identified **10 findings that validate plan assumptions**, **13 findings that require adjustments**, and **0 findings that fully invalidate assumptions**. The codebase is well-structured with clean extension points; most gaps are additions rather than conflicts.

The most impactful adjustments are: (1) the section marker system supports only one pair per file, not named multi-section markers; (2) the EcosystemModule interface lacks `VerificationCommands()` assumed by Phase 14; (3) the state directory is `.devinit/` not `.gdev/`; and (4) no semver library exists in go.mod.

---

## 2. What the Code Validates

### 2.1 SHA-256 Hash-Based State Tracking
- **File:** `internal/state/state.go:25-42`
- **Function:** `RecordFiles()` computes content hashes via `ComputeHash(f.Content)` for every generated file
- **Function:** `CheckModified()` at lines 46-94 compares stored hashes against on-disk state
- **Impact:** Phase 15 drift detection builds directly on `CheckModified()` -- the foundation exists

### 2.2 Three-Way Merge for settings.json and .mcp.json
- **File:** `addons/devinit/update.go:376-396`
- **Function:** `dispatchMerge()` routes to `merge.MergeSettings()` for settings.json and `merge.MergeMcpJson()` for .mcp.json
- **Impact:** Phase 13 config resolution can reuse these merge strategies for shared files

### 2.3 Section Marker Merge for CLAUDE.md
- **File:** `internal/merge/section.go:24-76`
- **Function:** `SectionMarkers()` splices new generated content between `<!-- BEGIN GENERATED SECTION -->` and `<!-- END GENERATED SECTION -->`
- **Impact:** Phase 14 CLAUDE.md enhancement builds on this (with adjustments -- see Finding 3.2)

### 2.4 Atomic File Writes
- **File:** `internal/fileutil/atomic.go:13-50`
- **Function:** `WriteFileAtomic()` uses temp-file + chmod + rename pattern
- **Impact:** All phases (13-16) should use this for any file write

### 2.5 Cobra Addon Pattern Is Clean and Extensible
- **File:** `cmd/gdev-bootstrap/main.go:1-38`
- **Pattern:** `bootstrap.Configure()` + per-addon `Configure()` + `cmd.Main()`
- Each addon (`devenv`, `claudecode`, `devinit`) registers its own commands
- **Impact:** New commands (check, status, info, repair, outdated, update, teardown, evidence) follow the existing pattern

### 2.6 EcosystemModule Registry
- **File:** `internal/ecosystem/registry.go:1-138`
- **Interface:** `internal/ecosystem/module.go:8-46` -- 12 methods on the `EcosystemModule` interface
- Registry supports `All()`, `ByTier()`, `ByName()`, `DetectAll()`, `Names()`, `Count()`
- 8 ecosystem modules visible (go, javascript, python, rust, java, dotnet, docker, terraform) based on `languageToRules` map and `DetectedProject` fields
- **Impact:** Phase 14 devenv tasks can iterate registered modules via `registry.All()`

### 2.7 `toolcheck.Detect()` Is Generic
- **File:** `internal/toolcheck/toolcheck.go:20-40`
- **Function:** `Detect(ctx, name, versionArg)` uses `exec.LookPath` + version arg
- **Impact:** Phase 15 can reuse for tool availability checks; Phase 16 prerequisite detection

### 2.8 Profile System Exists (InfraProfile + ProjectProfile Registries)
- **File:** `internal/profile/types.go:94-210` -- `InfraProfile` with Registry, NixCache, BuildCache, Scanning, Updates, SBOM configs
- **File:** `addons/devinit/profile_registry.go:1-77` -- `ProjectProfileRegistry` with `Register()`, `Get()`, `List()`, `Names()`
- **File:** `addons/devinit/profile_builtins.go:1-77` -- 4 built-in profiles: `go-web`, `ts-fullstack`, `python-data`, `rust-cli`
- **Impact:** Phase 13 extends this with compliance-level profiles (consulting-default, startup-fast, enterprise)

### 2.9 WizardAnswers Flows Cleanly Through the System
- **File:** `pkg/types/types.go:8-32` -- `WizardAnswers` struct with 19 fields
- **Flow:** `commands.go:56` -> `AnswersFromFlags` -> `RunWizard` -> `gen.Generate(answers)` -> `state.RecordFiles`
- **Impact:** Phase 13 layers `.qsdev.yaml` resolution on top; the resolved config maps into WizardAnswers for generation

### 2.10 Embedded Skills + Rules Library
- **File:** `addons/claudecode/templates.go:1-4` -- `//go:embed all:templates` directive
- **File:** `addons/claudecode/generate_skills.go:42-78` -- `deploySkills()` reads from `templateFS`
- **File:** `addons/claudecode/generate_skills.go:108-137` -- `deployRules()` with per-language selection + always-included `security-rules.md`
- **File:** `addons/claudecode/templates/skills/manifest.yaml` -- 6 skills defined (deploy, review-pr, security-review, generate-tests, refactor, db-migration)
- **Impact:** Phase 14 extends the library from 6 to 10+ gdev operation skills + 8 consulting workflow skills. Existing `deploySkills()` pattern applies.

---

## 3. Critical Adjustments Required

### 3.1 State Directory Naming (Phase 13, 15, 16)

**Plan assumes:** `.gdev/` state directory (e.g., `.gdev/state.yaml`, `.gdev/cache/`)

**Code uses:**
- `addons/devinit/commands.go:23`: `statePath = ".devinit/.qsdev-init-state.yaml"`
- `addons/devinit/commands.go:24-25`: `answersDir = ".devinit"`, `answersFileName = ".qsdev-init-answers.yaml"`
- Per-addon state: `devenv.SaveAnswers()` writes to `.devenv/`, `claudecode.SaveAnswers()` writes to `.claude/`

**Recommendation:** Phase 13 introduces `.qsdev.yaml` as the NEW public-facing project config (this is correct and distinct from the internal state). For internal state, either:
- (a) Migrate state to `.gdev/state.yaml` in Phase 13 as a breaking change (cleaner long-term), OR
- (b) Keep `.devinit/` for backward compat and have Phase 15 `qsdev status` aggregate from all three locations

Option (a) is strongly recommended. Add a migration step in Phase 13: detect `.devinit/.qsdev-init-state.yaml` and move to `.gdev/state.yaml`.

---

### 3.2 Single vs Multi-Section CLAUDE.md Markers (Phase 14)

**Plan assumes:** Multiple named section markers: `<!-- gdev:skills -->`, `<!-- gdev:agents -->`, `<!-- gdev:tasks -->`, `<!-- gdev:commands -->`

**Code has:**
- `internal/merge/section.go:14-15`: ONE pair: `BeginMarkerPrefix = "<!-- BEGIN GENERATED SECTION"` and `EndMarker = "<!-- END GENERATED SECTION -->"`
- `addons/claudecode/templates/claude-md.tmpl:3,73`: Uses `<!-- BEGIN GENERATED SECTION -- do not edit between markers -->` / `<!-- END GENERATED SECTION -->`
- `SectionMarkers()` function handles exactly one section per file (finds first begin/end pair only)

**Recommendation:** Two options:
1. **Extend `SectionMarkers()` to support named sections** -- add a `SectionMarkersNamed(existing, newGenerated, sectionID string)` variant that finds `<!-- gdev:X -->` / `<!-- /gdev:X -->` pairs. This enables Phase 14's multi-section CLAUDE.md.
2. **Keep single generated section with subsections inside it** (simpler, lower risk) -- all gdev content stays between the existing `BEGIN/END GENERATED SECTION` markers, with subsection headers (not markers) for skills/agents/tasks/commands inside the generated block.

Option 1 is the correct path given Phase 12's planned `qsdev enable/disable` needs to surgically update individual sections.

---

### 3.3 EcosystemModule Missing `VerificationCommands()` Method (Phase 14)

**Plan assumes:** `EcosystemModule.VerificationCommands()` returns per-ecosystem build/test/lint/format commands

**Code interface at `internal/ecosystem/module.go:8-46`** has 12 methods:
- `Name()`, `DisplayName()`, `Tier()`, `Detect()`, `DevenvNixFragment()`, `DevenvYamlInputs()`, `SecurityConfigs()`, `PreCommitHooks()`, `DenyRules()`, `CICommands()`, `PackageManagers()`, `WizardFields()`
- **No** `VerificationCommands()` method

**Closest analog:** `CICommands(config ModuleConfig) []CICommand` returns CI pipeline commands. Also, `addons/claudecode/generate_claude_md.go:27-39` has a hardcoded `languageCommands` map with `build`, `test`, `lint` per ecosystem.

**Recommendation:** Add `VerificationCommands()` as a supplementary interface:
```go
type VerifiableModule interface {
    VerificationCommands(config ModuleConfig) VerificationCmdSet
}
```
This avoids breaking the existing 8 modules. Modules that implement it get task generation; others fall back to the `languageCommands` map already in `generate_claude_md.go`. Migrate `languageCommands` into each module's `VerificationCommands()` implementation.

---

### 3.4 FileState Lacks Owner Field (Phase 12 prerequisite for 13-16)

**Plan assumes:** File ownership tracking for tool lifecycle (Phase 12), per-tool status reporting (Phase 15)

**Code at `pkg/types/types.go:120-125`:**
```go
type FileState struct {
    Hash        string        `yaml:"hash"         json:"hash"`
    Strategy    MergeStrategy `yaml:"strategy"      json:"strategy"`
    Mode        os.FileMode   `yaml:"mode"          json:"mode"`
    BaseContent []byte        `yaml:"base_content,omitempty" json:"base_content,omitempty"`
}
```
- **No** `Owner` field (which tool generated/owns this file)
- **No** `SectionID` field (which sections within a shared file are owned by which tool)
- **No** `Category` field ("machine-owned" / "human-edited" / "exclusive")

**Recommendation:** Add fields to `FileState`:
```go
Owner    string `yaml:"owner,omitempty" json:"owner,omitempty"`         // e.g., "semgrep", "gdev-core"
Category string `yaml:"category,omitempty" json:"category,omitempty"` // "machine-owned", "human-edited", "exclusive"
Sections []string `yaml:"sections,omitempty" json:"sections,omitempty"` // marker IDs for shared files
```
This is a Phase 12 prerequisite. Phase 15 drift detection uses `Category` to decide severity (machine-owned modification = warning, human-edited = info). Phase 16 teardown uses `Owner` to classify files for removal.

---

### 3.5 Only Two Onboarding Modes (Phase 13)

**Plan assumes:** 4 modes: Create / Join / Update / Repair

**Code has 2:**
- **Create:** `addons/devinit/commands.go:56-200` -- `runInit()` is the full wizard flow
- **Update:** `addons/devinit/update.go:60-178` -- `runUpdate()` loads saved answers and regenerates
- `addons/devinit/merge_mode.go:19-21`: `DetectExistingConfig()` returns `ExistingConfig` with `NeedsMergeMode()` -- but this only BLOCKS init (returns error), does not route to different modes

**What's missing:**
- Join mode (config exists, no local state -- skip wizard, generate from config)
- Repair mode (state exists, files drifted -- show drift report, offer auto-fix)
- Mode detection/routing logic (the `switch result.Mode` dispatch)

**Recommendation:** Extend `runInit()` to add mode detection before the wizard. `DetectExistingConfig` should be expanded to return a `ModeRecommendation` enum and the `runInit` function should dispatch on it. The Update path already exists; Join and Repair are new execution branches.

---

### 3.6 No Semver Parsing Library (Phase 13)

**Plan assumes:** `gdev_version` constraint parsing with operators: `>=`, `~>`, `^`

**Code at `go.mod:1-48`** has NO semver library. Dependencies are:
- `github.com/charmbracelet/huh` (forms)
- `github.com/spf13/cobra` (CLI)
- `github.com/spf13/pflag` (flags)
- `gopkg.in/yaml.v3` (YAML)

**Recommendation:** Add `github.com/Masterminds/semver/v3` -- it supports all required operators (`>=`, `<=`, `>`, `<`, `=`, `!=`, `~`, `^`) and is the de facto Go standard (used by Helm, Hugo, etc.). Alternatively, implement minimal constraint matching for v1 (support only `>= X.Y.Z`) and defer advanced operators.

---

### 3.7 Skills Are Static Markdown -- This Is Correct (Phase 14)

**Plan references:** `!`command`` dynamic context injection in skills

**Code at `addons/claudecode/generate_skills.go:60-77`:** Skills are deployed as plain `.md` files from `embed.FS` with `Strategy: types.LibraryManaged`

**Validation:** This is correct. The `` !`command` `` syntax is a Claude Code runtime feature -- Claude Code preprocesses skill files at invocation time. Skills deployed as static markdown with `!` syntax will work natively without any Go-side template processing. New gdev operation skills with `!`qsdev devenv doctor --json`` will work as-is.

**No adjustment needed.**

---

### 3.8 No --json Output Pattern Exists (Phase 15-16)

**Plan assumes:** `qsdev status --json`, `qsdev status --sarif`, `qsdev check --format json`

**Code:** No existing command has `--json` or `--format` support. All output goes through `fmt.Fprintf` to `cmd.OutOrStdout()`.

**Recommendation:** Establish the pattern in Phase 15 Unit 15.1:
```go
type OutputFormat string
const (
    FormatText  OutputFormat = "text"
    FormatJSON  OutputFormat = "json"
    FormatSARIF OutputFormat = "sarif"
    FormatBadge OutputFormat = "badge"
)
```
Add a `--format` string flag (default "text") and `--json` as shorthand for `--format json`. Use `cmd.OutOrStdout()` for output. Add a shared `internal/output/` package for format rendering so Phase 16 commands can reuse it.

---

### 3.9 devenv.nix Uses Informal Comment Markers (Phase 15)

**Plan assumes:** Formal section markers in devenv.nix for drift detection

**Code at `addons/devenv/security_defaults.go:101-150`:** Custom hooks use comments like:
- `${pkgs.writeShellScript "lock-audit" '' ... ''}`
- The devenv.nix template generates sections with comments like `# Base packages`, `# Go`, `# Services`, `# Git hooks`

These are NOT paired `# --- section ---` / `# --- end section ---` markers. The `internal/merge/section.go` `SectionMarkers()` handles HTML comment markers only (CLAUDE.md pattern).

**Recommendation:** For Phase 15 drift detection on devenv.nix:
- Use **hash-only detection** (already works via `state.CheckModified()`) -- detect that devenv.nix has been modified
- Do NOT rely on section markers in Nix files for Phase 15
- Phase 12's tool lifecycle should introduce formal Nix section markers (`# --- <tool> ---` / `# --- end <tool> ---`) if per-tool surgery in devenv.nix is required
- Add a `NixSectionMarkers()` variant to the merge package when Phase 12 needs it

---

### 3.10 Profiles Lack Compliance Levels (Phase 13)

**Plan assumes:** `baseline` / `enhanced` / `strict` compliance levels on profiles

**Code at `addons/devinit/config.go:11-24`:**
```go
type Profile struct {
    Description     string
    Languages       []LanguageSpec
    Services        []string
    Direnv          bool
    ClaudeCode      bool
    PermissionLevel string
    Skills          []string
    Hooks           []string
    GitHooks        []string
    ExtraPackages   []string
    MCPServers      []string
    InfraProfile    string
}
```
- No `ComplianceLevel` or `SecurityLevel` field
- `InfraProfile` at `internal/profile/types.go:94-105` also has no compliance level (it has Scanning, SBOM, Updates -- the building blocks, but no single "level" field)

**Recommendation:** Phase 13 adds a new config layer (`.qsdev.yaml` `GdevConfig` struct) ABOVE the existing Profile system. The compliance level maps to concrete settings via `complianceLevelToConfig()` (as the plan describes). The existing `Profile` struct does not need a compliance level field -- that goes on `GdevConfig.Security.Level` and `ClientConfig.SecurityLevel`. Resolution merges the compliance level's concrete settings into the resolved config.

---

### 3.11 enterShell Is Hardcoded String (Phase 16)

**Plan assumes:** Extensible enterShell accepting additional notification lines

**Code at `addons/devenv/security_defaults.go:156-184`:** `buildEnterShellScript()` returns a hardcoded multi-line string with no composition mechanism. It's not a template -- it's a Go function returning a string literal.

**Recommendation:** Convert to a string builder pattern that accepts contributions:
```go
func buildEnterShellScript(contributions []string) string {
    var sb strings.Builder
    // ... security checks ...
    for _, c := range contributions {
        sb.WriteString(c)
        sb.WriteByte('\n')
    }
    return sb.String()
}
```
Phase 16's gdev environment notification and any tool-contributed enterShell content concatenate into this. The gdev notification should be the LAST line.

---

### 3.12 Three Separate State Files (Phase 15)

**Plan assumes:** Single state source for status reporting

**Code writes state to 3 locations:**
- `addons/devinit/commands.go:173-175`: `.devinit/.qsdev-init-state.yaml` (master state)
- `addons/devinit/commands.go:184-186`: `devenv.SaveAnswers()` to `.devenv/.gdev-state.yaml`
- `addons/devinit/commands.go:187-189`: `claudecode.SaveAnswers()` to `.claude/.gdev-claude-state.yaml`

The master state at `.devinit/.qsdev-init-state.yaml` contains ALL generated files from both addons (computed at line 172: `state.RecordFiles(allFiles)`). The per-addon saves are ANSWERS files, not state files.

**Recommendation:** Phase 15 `qsdev status` should load the master state from the single canonical location (`.devinit/.qsdev-init-state.yaml` or migrated `.gdev/state.yaml`). The per-addon answer files are inputs, not state. Add `state.LoadAllStates(projectRoot)` that knows the canonical state path. If Phase 13 migrates to `.gdev/state.yaml`, this is simplified.

---

### 3.13 Command Registration for New Commands (Phases 13-16)

**Current top-level commands:** `qsdev init`, `qsdev devenv`, `qsdev claude` (via `cmd.Main()` from the gdev framework)

**New commands needed:** `qsdev check`, `qsdev status`, `qsdev info`, `qsdev repair`, `qsdev outdated`, `qsdev update`, `qsdev teardown`, `qsdev evidence`, `qsdev team-report`

**Code at `cmd/gdev-bootstrap/main.go`:** Shows `bootstrap.Configure()` + per-addon `Configure()` pattern. Each addon registers its own Cobra commands.

**Recommendation:** Create a new addon package (`addons/lifecycle/`) for Phase 13-16 commands:
- `addons/lifecycle/addon.go` -- registration
- `addons/lifecycle/cmd_check.go` -- qsdev check
- `addons/lifecycle/cmd_status.go` -- qsdev status
- `addons/lifecycle/cmd_info.go` -- gdev info
- `addons/lifecycle/cmd_repair.go` -- qsdev repair
- `addons/lifecycle/cmd_outdated.go` -- gdev outdated
- `addons/lifecycle/cmd_update.go` -- qsdev update
- `addons/lifecycle/cmd_teardown.go` -- gdev teardown
- `addons/lifecycle/cmd_evidence.go` -- gdev evidence

This keeps `devinit` focused on the init/wizard flow. `main.go` adds `lifecycle.Configure()`.

---

## 4. What Remains Accurate (No Changes Needed)

| Plan Assumption | Code Validation |
|----------------|-----------------|
| Cobra command pattern for all new commands | `cmd/gdev-bootstrap/main.go` -- addon pattern proven |
| `embed.FS` for template/skill/agent deployment | `addons/claudecode/templates.go` -- `//go:embed all:templates` |
| SHA-256 hash-based state for drift detection | `internal/state/state.go:26-35` -- `ComputeHash()` per file |
| `LibraryManaged` merge strategy for skills/rules/agents | `addons/claudecode/generate_skills.go:70` -- already used |
| Profile registry pattern for client profiles | `addons/devinit/profile_registry.go` -- threadsafe, ordered |
| Detection engine for onboarding mode | `internal/detect/` package + `DetectedProject` struct |
| WizardAnswers as the canonical input | `pkg/types/types.go:8-32` -- 19-field struct |
| Atomic file writes for all new files | `internal/fileutil/atomic.go` -- proven pattern |
| Template engine (`internal/tmpl`) for Nix and Markdown | `addons/claudecode/generate_claude_md.go:88-106` -- `tmpl.NewMarkdownRenderer()` |
| `!`command`` dynamic context in skills | Claude Code runtime feature, no Go processing needed |
| Per-ecosystem build/test/lint commands exist | `addons/claudecode/generate_claude_md.go:27-39` -- `languageCommands` map |

---

## 5. Impact Matrix

| Unit | Status | Key Adjustment |
|------|--------|---------------|
| **13.1** .qsdev.yaml Schema & Parser | Needs Adjustment | New file/package; state directory migration from `.devinit/` to `.gdev/` |
| **13.2** Configuration Resolution Engine | Needs Adjustment | Profiles lack compliance level; resolution must bridge old Profile to new GdevConfig |
| **13.3** Onboarding Mode Detection | Needs Adjustment | Only 2 of 4 modes exist; DetectExistingConfig must return ModeRecommendation |
| **13.4** Join Mode Implementation | Needs Adjustment | Entirely new execution branch |
| **13.5** gdev_version Constraint | Needs Adjustment | No semver library in go.mod; add `Masterminds/semver/v3` |
| **13.6** qsdev check Command | Validated | Follows existing Cobra pattern; add `internal/check/` package |
| **13.7** Client Profiles & Compliance | Needs Adjustment | Compliance levels are new; existing Profile struct needs no changes (GdevConfig layer) |
| **14.1** gdev Operation Skills | Validated | Extends existing `deploySkills()` with 10 more SKILL.md files |
| **14.2** Consulting Workflow Agents | Validated | New `.claude/agents/` directory via embed.FS; same deploy pattern |
| **14.3** Consulting Workflow Skills | Validated | Same pattern as existing 6 skills in manifest |
| **14.4** Context Budget Management | Validated | Straightforward line-counting + generation control |
| **14.5** Deny Rule Conflict Validation | Validated | New test matrix; no architectural conflict |
| **14.6** devenv Task Definitions | Needs Adjustment | `VerificationCommands()` doesn't exist on interface; use `languageCommands` map as seed |
| **14.7** CLAUDE.md Section Enhancement | Needs Adjustment | Single marker pair -> must add named section support to merge package |
| **15.1** qsdev status Command | Needs Adjustment | No --json pattern exists; establish `internal/output/` format package |
| **15.2** Compliance Posture Scoring | Validated | New `internal/posture/` package; reads existing state/registry |
| **15.3** Drift Detection Engine | Validated | Builds on `state.CheckModified()`; extends with 6 categories |
| **15.4** gdev evidence Command | Validated | New command; reads existing PostureReport |
| **15.5** Machine-Readable Output & Badges | Needs Adjustment | Establish --format flag pattern for reuse across commands |
| **15.6** Team Aggregation Pipeline | Validated | New command; consumes JSON output from 15.5 |
| **16.1** qsdev repair | Needs Adjustment | Needs state directory consistency (Finding 3.1) + FileState.Owner (Finding 3.4) |
| **16.2** gdev info | Validated | Simple YAML reader; no conflicts |
| **16.3** gdev outdated | Validated | Thin wrapper; no conflicts |
| **16.4** qsdev update | Validated | Composes existing `runUpdate()` (Phase 8) + self-update + devenv update |
| **16.5** gdev teardown | Needs Adjustment | Needs FileState.Owner/Category for file classification |
| **16.6** Git Workflow Automation | Validated | New lifecycle-managed tools; follows existing pattern |
| **16.7** Shell & Environment Integration | Needs Adjustment | enterShell is hardcoded (Finding 3.11) |

---

## 6. Recommended Implementation Sequence

Given the code state, the implementation order should be:

### Phase 12 Must Come First
Phase 12 (tool lifecycle) is the critical prerequisite because:
1. `FileState.Owner` and `FileState.Category` (Finding 3.4) are needed by Phase 13 (config resolution knows who owns what), Phase 15 (drift severity depends on category), and Phase 16 (teardown classifies files by owner).
2. Named section markers (Finding 3.2) must be added to the merge package for Phase 14's multi-section CLAUDE.md.
3. The tool registry (enable/disable) is referenced by Phase 14 skills and Phase 15 defense assessment.

### Then Phase 13 (Project Configuration)
1. **13.1** first (GdevConfig struct + parser) -- everything else depends on it
2. **13.5** next (semver constraint) -- needed by 13.3 mode detection
3. **13.2** (resolution engine) -- uses 13.1's config
4. **13.3** (mode detection) -- uses 13.2's resolved config + 13.5's version check
5. **13.4** (join mode) -- new branch using 13.3's routing
6. **13.7** (compliance levels) -- enriches 13.2's resolution
7. **13.6** (qsdev check) -- uses everything above

State directory migration (`.devinit/` -> `.gdev/`) should happen as the FIRST task in Phase 13.1.

### Then Phase 14 (Claude Code Integration)
1. **14.6** (devenv tasks) early -- needs VerificationCommands interface addition
2. **14.7** (CLAUDE.md sections) next -- needs named marker support from Phase 12
3. **14.1** (gdev operation skills) -- depends on 14.7 for directory section
4. **14.2** (agents) -- same deployment pattern
5. **14.3** (consulting skills) -- same pattern
6. **14.4** (context budget) -- validates output of 14.1-14.3
7. **14.5** (deny rule conflicts) -- validates 14.1-14.3 against settings.json

### Then Phase 15 (Health & Status)
1. **15.1** (qsdev status + --format pattern) -- establishes output infrastructure
2. **15.3** (drift detection) -- feeds into 15.2's scoring
3. **15.2** (scoring engine) -- consumes drift + defense + deps
4. **15.5** (JSON/SARIF/badge rendering) -- serializes 15.2's output
5. **15.4** (evidence command) -- consumes full PostureReport
6. **15.6** (team aggregation) -- consumes 15.5's JSON output

### Finally Phase 16 (DX Polish)
1. **16.2** (gdev info) -- lightest, validates state reading
2. **16.3** (gdev outdated) -- thin wrapper, no dependencies
3. **16.1** (qsdev repair) -- needs drift detection from Phase 15
4. **16.4** (qsdev update) -- coordinates existing update flow
5. **16.5** (gdev teardown) -- needs FileState.Owner from Phase 12
6. **16.6** (git workflow) -- independent lifecycle tools
7. **16.7** (shell/env integration) -- needs enterShell refactor

---

## Appendix: Key File Paths Referenced

| File | Purpose |
|------|---------|
| `pkg/types/types.go` | WizardAnswers, GeneratedFile, GeneratedState, FileState structs |
| `pkg/types/merge_strategy.go` | MergeStrategy enum (8 values) |
| `internal/state/state.go` | RecordFiles(), CheckModified() -- hash-based state tracking |
| `internal/merge/section.go` | SectionMarkers() -- single-pair BEGIN/END merge |
| `internal/fileutil/atomic.go` | WriteFileAtomic() -- atomic writes |
| `internal/ecosystem/module.go` | EcosystemModule interface (12 methods) |
| `internal/ecosystem/registry.go` | Registry with All(), ByName(), DetectAll() |
| `internal/toolcheck/toolcheck.go` | Detect() -- binary presence + version |
| `internal/profile/types.go` | InfraProfile struct (Registry, NixCache, BuildCache, etc.) |
| `addons/devinit/commands.go` | runInit() -- the main init flow |
| `addons/devinit/update.go` | runUpdate() -- the update flow |
| `addons/devinit/config.go` | Profile struct (11 fields, no compliance level) |
| `addons/devinit/merge_mode.go` | ExistingConfig + DetectExistingConfig() + NeedsMergeMode() |
| `addons/devinit/answers.go` | saveAnswers/loadAnswers to `.devinit/` |
| `addons/devinit/profile_registry.go` | ProjectProfileRegistry |
| `addons/devinit/profile_builtins.go` | 4 built-in profiles |
| `addons/claudecode/generate_skills.go` | deploySkills(), deployRules(), languageToRules map |
| `addons/claudecode/generate_claude_md.go` | BuildClaudeMdData(), languageCommands map |
| `addons/claudecode/templates/claude-md.tmpl` | CLAUDE.md template with BEGIN/END markers |
| `addons/devenv/security_defaults.go` | buildEnterShellScript() -- hardcoded string |
| `cmd/gdev-bootstrap/main.go` | Entry point, addon configuration pattern |
| `go.mod` | Dependencies -- no semver library |
