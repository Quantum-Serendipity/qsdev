# Phase 6: Wizard & Orchestration (devinit)

## Goal

Implement the devinit addon's unified wizard using charmbracelet/huh, the profile system, detection pre-population, plan preview, merge mode for existing projects, and non-interactive/CI flag mapping. At the end of this phase, `qsdev init` provides a complete guided setup experience from detection through generation.

## Dependencies

Phases 2 and 3 complete (devenv and claudecode generation). Phase 4 desirable but not blocking (security configs can be added incrementally).

## Phase Outputs

- Unified `qsdev init` command orchestrating both addons
- huh-based wizard with quick path (1 question) and customize (5 form groups)
- Detection engine pre-populating wizard answers
- Profile system with named presets (go-web, ts-fullstack, python-data, etc.)
- Plan preview before generation
- Merge mode for existing projects
- Complete CLI flag mapping for non-interactive/CI use
- Bootstrap step registration for system tool installation

---

### Unit 5.1: devinit Addon Scaffolding & Command Hierarchy

**Description:** Wire the devinit addon's command hierarchy: `qsdev init` as the primary entry point, orchestrating devenv and claudecode generation.

**Context:** devinit is the orchestration addon. It imports devenv and claudecode, runs the unified wizard, and dispatches generation to each addon. The command hierarchy places `qsdev init` at top level, not under a subcommand.

**Desired Outcome:** `qsdev init` runs the full wizard → generation → post-generation pipeline, calling into devenv and claudecode addons.

**Steps:**
1. Register `qsdev init` command in devinit addon.
2. Implement orchestration flow: detect → wizard → generate (devenv) → generate (claudecode) → generate (security configs) → write all → report.
3. Support `qsdev init --devenv-only` and `qsdev init --claude-only` for partial generation.
4. Support `qsdev init --yes` (accept all defaults), `qsdev init --profile <name>` (use preset).
5. Post-generation: print summary of generated files and next steps.

**Acceptance Criteria:**
- [ ] `qsdev init` runs full pipeline
- [ ] `qsdev init --devenv-only` skips Claude Code generation
- [ ] `qsdev init --claude-only` skips devenv generation
- [ ] `qsdev init --yes` accepts defaults without wizard
- [ ] Post-generation summary lists all files with actions

**Research Citations:**
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Command Hierarchy` — `qsdev init` at top level
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Inter-Addon Communication` — direct function calls

**Status:** Not Started

---

### Unit 5.2: huh Wizard Forms

**Description:** Implement the huh-based wizard with quick path and 5 customize form groups using progressive disclosure.

**Context:** gdev's bootstrap addon already uses huh (validated). The wizard follows "opinionated menu with escape hatches": Group 1 asks accept-defaults/customize, Groups 2-5 are hidden on quick path. `WithHideFunc()` controls conditional display.

**Desired Outcome:** Interactive wizard that collects all needed answers in <30 seconds for customizers, <5 seconds for quick-path users.

**Steps:**
1. Implement `buildInitForm(detected DetectedProject, defaults Profile) *huh.Form` per wizard-flow-integration-design.md.
2. Group 1: Quick Selection — Yes (defaults) / Customize / Show defaults.
3. Group 2: Languages & Runtimes — multi-select with version dropdowns, pre-populated from detection.
4. Group 3: Services — multi-select with conditional service-specific config.
5. Group 4: Dev Environment — direnv toggle, git hooks, extra packages.
6. Group 5: Claude Code — enable/disable, permission level, skills, hooks, MCP.
7. Group 6: Plan Preview & Confirm.
8. Groups 2-5 hidden when quick path selected (`WithHideFunc`).
9. Apply theme from config (default: Dracula or team-configured).
10. Accessibility mode via `ACCESSIBLE` or `NO_COLOR` env var detection.

**Acceptance Criteria:**
- [ ] Quick path shows only Group 1 and Group 6 (2 screens)
- [ ] Customize path shows all 6 groups
- [ ] Detection results pre-populate language selections
- [ ] `WithHideFunc` correctly hides groups based on quick choice
- [ ] Claude Code group hidden when `ClaudeCode = false`
- [ ] Accessibility mode works when `ACCESSIBLE` env var set
- [ ] Form returns populated `WizardAnswers`

**Research Citations:**
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § huh Form Construction` — complete form code
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Progressive Disclosure` — WithHideFunc mechanics
- Validation: gdev bootstrap already uses huh, confirming library choice

**Status:** Not Started

---

### Unit 5.3: Detection Pre-Population

**Description:** Wire the detection engine (Phase 1, Unit 1.3) into the wizard to pre-populate answers from existing project files.

**Context:** When a user runs `qsdev init` in a Go project with a package.json, the wizard should pre-select both Go and TypeScript with detected versions. When existing devenv.nix or CLAUDE.md exists, offer merge mode.

**Desired Outcome:** Detection results automatically populate wizard defaults, reducing questions to confirmation rather than input.

**Steps:**
1. Call `Detect(projectRoot)` before building the wizard form.
2. Map `DetectedProject` to `WizardAnswers` defaults (languages, versions, package managers).
3. When existing config detected, set merge mode flag and adjust wizard messaging.
4. Pre-select detected languages in the multi-select (user can deselect).
5. For quick path: detection result informs the default profile shown in the summary.

**Acceptance Criteria:**
- [ ] Go project pre-selects Go with detected version
- [ ] Node project pre-selects TypeScript/JavaScript with detected package manager
- [ ] Existing devenv.nix triggers merge mode prompt
- [ ] Existing CLAUDE.md triggers merge mode prompt
- [ ] Pre-selected items are default-checked in multi-select (user can change)

**Research Citations:**
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Detection Engine` — detection → wizard mapping

**Status:** Not Started

---

### Unit 5.4: Profile System

**Description:** Implement named profiles that encode complete wizard answers for common project types, enabling `qsdev init --profile go-web --yes` for zero-question setup.

**Context:** Profiles are team-defined presets compiled into the binary. They map a profile name to a complete `WizardAnswers` struct. The wizard can also save the current answers as a new profile.

**Desired Outcome:** Profile system supports built-in profiles, team-configured profiles, and profile-based zero-question initialization.

**Steps:**
1. Define built-in profiles: `go-web` (Go + PostgreSQL + Redis + standard Claude), `ts-fullstack` (TypeScript + pnpm + PostgreSQL + standard Claude), `python-data` (Python + uv + minimal Claude), `rust-cli` (Rust + minimal Claude).
2. Implement `ProfileRegistry` with `Get(name) (Profile, error)` and `List() []ProfileSummary`.
3. Support team-configured profiles via `devinit.Configure(devinit.WithProfile(name, profile))` in main.go.
4. `qsdev init --profile <name>` → load profile → skip wizard → generate.
5. `qsdev init --profile <name>` with additional flags → profile as base, flags as overrides.

**Acceptance Criteria:**
- [ ] `qsdev init --profile go-web --yes` generates complete Go web project config with zero questions
- [ ] `qsdev init --profile go-web --service mongodb` uses profile but adds MongoDB
- [ ] Built-in profiles cover Go, TypeScript, Python, Rust
- [ ] Team-configured profiles register via `Configure()`
- [ ] `qsdev init --list-profiles` shows available profiles with descriptions

**Research Citations:**
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Profile System` — profile configuration
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Non-Interactive Mode` — profile + flags

**Status:** Not Started

---

### Unit 5.5: Non-Interactive / CI Flag Mapping

**Description:** Implement complete CLI flag mapping so every wizard question can be answered via flags for CI/scripting use.

**Context:** Non-interactive mode is critical for CI pipelines and automated setup. Every wizard question maps to a flag. Partial flags trigger a wizard for remaining questions. `--yes` fills remaining with defaults.

**Desired Outcome:** Full flag coverage enabling completely scriptable project initialization.

**Steps:**
1. Map every wizard field to a CLI flag per devenv-addon-design.md and claude-code-addon-design.md.
2. Implement `answersFromFlags(flags *pflag.FlagSet) WizardAnswers` — populate from flags.
3. Implement `answers.IsComplete() bool` — check if all required fields are set.
4. When `--yes` and not complete: fill remaining from detected defaults.
5. When not `--yes` and not complete: run wizard for missing fields only.
6. Document all flags in command help text.

**Acceptance Criteria:**
- [ ] Full non-interactive initialization: `qsdev init --lang go --go-version 1.24 --service postgres --direnv --claude-code --claude-permissions standard --yes`
- [ ] Partial flags + wizard: `qsdev init --lang go` opens wizard with Go pre-selected
- [ ] `--yes` fills remaining from defaults
- [ ] All flags documented in `qsdev init --help`
- [ ] Flag names match design spec

**Research Citations:**
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Non-Interactive / CI Mode` — flag mapping
- `research-spikes/gdev-extension-design/devenv-addon-design.md § Non-Interactive Flags` — devenv flag list
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § Non-Interactive Flags` — claude flag list

**Status:** Not Started

---

### Unit 5.6: Bootstrap Step Registration

**Description:** Register bootstrap steps for system tool installation (devenv, direnv, claude) that teams can include in their `qsdev bootstrap` flow.

**Context:** Bootstrap steps are separate from `qsdev init` — they install system-level tools, not project config. Teams opt-in via `bootstrap.Configure(bootstrap.WithSteps(...))`. Steps use gdev's existing bootstrap infrastructure with huh forms, skip handlers, and headless mode.

**Desired Outcome:** Optional bootstrap steps that install devenv, direnv, and Claude Code when missing.

**Steps:**
1. Implement `devenv.InstallDevenvStep()` — check `devenv --version`, offer to install via `nix profile install nixpkgs#devenv`.
2. Implement `devenv.InstallDirenvStep()` — check `direnv --version`, offer to install via `nix profile install nixpkgs#direnv`.
3. Implement `claudecode.InstallClaudeStep()` — check `claude --version`, offer installation instructions.
4. Use `SkipInContainer()` for steps that don't make sense in CI containers.
5. Use `SkipIfNoGUI()` for steps requiring terminal interaction in headless mode.

**Acceptance Criteria:**
- [ ] Steps skip when tools already installed
- [ ] Steps offer installation when tools missing
- [ ] Container detection skips appropriately
- [ ] Headless mode skips interactive steps
- [ ] Steps integrate with gdev's existing bootstrap plan

**Research Citations:**
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Bootstrap Integration` — step registration
- Validation: SkipInContainer/SkipIfNoGUI confirmed as new gdev features

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] `qsdev init` in a Go project: detects Go, offers quick path, generates all files in <60s
- [ ] `qsdev init --profile go-web --yes` generates complete config with zero questions
- [ ] `qsdev init --lang go --service postgres --claude-code --yes` works fully non-interactive
- [ ] Merge mode works when existing config detected
- [ ] Bootstrap steps install missing tools
