# Phase 1: Foundation & Shared Infrastructure

## Goal

Establish the Go module structure, three addon scaffolds, shared types, and core infrastructure (detection engine, template engine, generation pipeline, hash tracking) that all subsequent phases build on. At the end of this phase, the addon skeleton compiles, registers with gdev, and can generate a trivial test file through the full pipeline.

## Dependencies

None — this is the entry point.

## Phase Outputs

- Three addon packages (`addons/devenv/`, `addons/claudecode/`, `addons/devinit/`) that register with gdev
- Shared types package with `WizardAnswers`, `GeneratedFile`, `DetectedProject`, `GeneratedState`
- **Ecosystem module interface** (`EcosystemModule`) that all 27 language/platform modules implement
- Detection engine that scans a project directory and returns `DetectedProject` (extensible via ecosystem modules)
- Template engine with Nix-specific `FuncMap` and `embed.FS` support
- Generation pipeline with atomic writes and post-generation validation
- Hash tracking with SHA256-based `GeneratedState` persistence
- **Infrastructure profile** types for registry proxy, cache, and scanning tool configuration

---

### Unit 1.1: Go Module & Addon Scaffolding

**Description:** Create the three addon packages following gdev's `_template` pattern. Each addon registers with `cmd.Main()` and has a placeholder `Configure()` + `initialize()`.

**Context:** Every subsequent unit depends on this scaffolding. The `_template` addon at `~/Repos/gdev/addons/_template/addon.go` is the canonical pattern. gdev's `Addon[T]` generic struct requires a `Config` type, `Definition` with name/description/initialize, and a `Configure()` function with option pattern.

**Desired Outcome:** `go build` succeeds with all three addons registered. A test binary that calls `cmd.Main()` starts without error.

**Steps:**
1. Create `addons/devenv/addon.go` — `Addon[Config]` with name `"devenv"`, empty `Config` struct, `Configure(opts ...option)`, `initialize()` placeholder.
2. Create `addons/claudecode/addon.go` — same pattern, name `"claudecode"`.
3. Create `addons/devinit/addon.go` — same pattern, name `"devinit"`. This addon imports devenv and claudecode packages.
4. Create `addons/devenv/config.go`, `addons/claudecode/config.go`, `addons/devinit/config.go` — Config structs with YAML tags matching the config key namespace (`devenv:`, `claudecode:`, `devinit:`).
5. Create a test `main.go` that configures all three addons and calls `cmd.Main()`.
6. Verify `go build` and `go vet` pass.

**Acceptance Criteria:**
- [ ] Three addon packages exist at `addons/devenv/`, `addons/claudecode/`, `addons/devinit/`
- [ ] Each follows `_template` pattern: unexported `addon` var, `Config` struct, `option` type, `Configure()`, `initialize()`
- [ ] Test binary compiles and starts without panic
- [ ] `go vet ./...` passes

**Research Citations:**
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Addon Composition Model` — three-addon pattern with rationale
- `research-spikes/gdev-extension-design/gdev-architecture-research.md § Addon System` — `Addon[T]` struct, registration, lifecycle
- Validation: gdev architecture confirmed unchanged, `_template` still canonical guide

**Status:** Not Started

---

### Unit 1.2: Shared Types & Interfaces

**Description:** Define the shared data types used across all three addons: `WizardAnswers`, `GeneratedFile`, `DetectedProject`, `GeneratedState`, `LanguageChoice`, `ServiceChoice`, and the `Generator` interface.

**Context:** These types form the contract between detection → wizard → generation → migration. `WizardAnswers` flows from wizard to generators. `GeneratedFile` flows from generators to the write pipeline. `DetectedProject` flows from detection to wizard pre-population. `GeneratedState` persists between runs for migration.

**Desired Outcome:** A shared `types` package (or sub-package of devinit) that all three addons import. Types compile with correct YAML/JSON tags.

**Steps:**
1. Create `addons/devinit/types/` package (shared types live under devinit since it's the orchestrator).
2. Define `WizardAnswers` struct with all fields from wizard-flow-integration-design.md: `ProjectName`, `ProjectRoot`, `Detected`, `Languages []LanguageChoice`, `Services []ServiceChoice`, `Direnv bool`, `GitHooks []string`, `ExtraPackages []string`, `EnvVars map[string]string`, `ClaudeCode bool`, `PermissionLevel string`, `Skills []string`, `Hooks HookChoices`, `MCPServers []string`, `QuickChoice string`, `Confirmed bool`.
3. Define `LanguageChoice` struct: `Name`, `Version`, `PackageManager`, `Extras []string`.
4. Define `ServiceChoice` struct: `Name`, `Version`, `Settings map[string]string`.
5. Define `DetectedProject` struct with all detection fields from wizard-flow-integration-design.md.
6. Define `GeneratedFile` struct: `Path string`, `Content []byte`, `Mode os.FileMode`, `Strategy MergeStrategy`.
7. Define `MergeStrategy` enum: `Overwrite`, `Append`, `Merge`, `Skip`, `SectionMarker`, `ThreeWayMerge`, `LibraryManaged`.
8. Define `Generator` interface: `Generate(answers WizardAnswers) ([]GeneratedFile, error)`.
9. Define `GeneratedState` struct for hash tracking: `LastRun time.Time`, `Files map[string]FileState`. `FileState`: `Hash string`, `Strategy MergeStrategy`.
10. Write unit tests verifying YAML/JSON marshal round-trips.

**Acceptance Criteria:**
- [ ] All types compile with correct struct tags
- [ ] `WizardAnswers` JSON round-trip preserves all fields
- [ ] `GeneratedState` YAML round-trip preserves all fields
- [ ] `MergeStrategy` has String() method for readable output
- [ ] Unit tests pass

**Research Citations:**
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Unified Template Data Model` — WizardAnswers, GeneratedFile, Generator interface
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Detection Engine` — DetectedProject struct
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Tracking Mechanism` — GeneratedState with SHA256

**Status:** Not Started

---

### Unit 1.3: Detection Engine

**Description:** Implement the project detection engine that scans a directory for language markers, existing config files, and git state, returning a populated `DetectedProject`.

**Context:** Detection feeds the wizard's pre-population. When a user runs `gdev init` in a Go project with an existing devenv.nix, the wizard should pre-select Go and offer merge mode. Detection must be fast (<100ms) since it runs before the wizard appears.

**Desired Outcome:** A `detect.go` in the devinit package that returns `DetectedProject` with all fields populated from filesystem scanning.

**Steps:**
1. Create `addons/devinit/detect.go`.
2. Implement language detection: check for `go.mod` (parse Go version), `package.json` (parse engines.node, detect package manager from lockfiles), `Cargo.toml`, `pyproject.toml` (detect poetry vs uv from lockfiles), `.nvmrc`, `.python-version`, `rust-toolchain.toml`.
3. Implement existing config detection: `devenv.nix`, `devenv.yaml`, `.claude/`, `CLAUDE.md`, `.envrc`, `.mcp.json`, `.claude/settings.json`.
4. Implement git detection: `.git/` exists, `.git/hooks/` has files, parse remote URL from `.git/config`.
5. Return `DetectedProject` with confidence levels (certain/probable/absent) for each detection.
6. Add version parsing: extract Go version from `go.mod` (`go` directive), Node version from `.nvmrc` or `package.json engines.node`, Python version from `.python-version` or `pyproject.toml [tool.poetry.dependencies.python]`.
7. Write unit tests with fixture directories (create temp dirs with marker files).

**Acceptance Criteria:**
- [ ] Detects Go project from `go.mod` with version extraction
- [ ] Detects Node project from `package.json` with package manager detection (npm/pnpm/yarn/bun from lockfile)
- [ ] Detects Python project from `pyproject.toml` with poetry/uv detection
- [ ] Detects Rust project from `Cargo.toml`
- [ ] Detects existing devenv/claude config
- [ ] Detects git state
- [ ] Completes in <100ms on a typical project
- [ ] Unit tests with fixture directories pass

**Research Citations:**
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Detection Engine` — DetectedProject struct, detection heuristics
- `research-spikes/gdev-extension-design/devenv-addon-design.md § devenv-detect-project step` — language detection with confidence

**Status:** Not Started

---

### Unit 1.4: Template Engine — Nix FuncMap & Embed Infrastructure

**Description:** Implement the template engine infrastructure: custom Nix template functions (`nixFuncs` FuncMap), embed.FS setup for all template files, and the template loading/rendering pipeline.

**Context:** devenv.nix must be generated via `text/template` because no Go Nix AST library exists. The custom FuncMap provides Nix-safe escaping (`nixString` escapes `${}` → `\${}`), list formatting (`nixList` produces `[ pkgs.x pkgs.y ]`), and boolean conversion. Markdown templates (CLAUDE.md) also use text/template. YAML/JSON use struct marshaling (separate unit).

**Desired Outcome:** A template engine that can load embedded templates, apply the Nix FuncMap, and render to `[]byte`. Separate from the generation pipeline (Unit 1.5) — this unit is pure template rendering.

**Steps:**
1. Create `addons/devinit/tmpl/` package for shared template utilities.
2. Implement `nixFuncs` FuncMap: `nixList`, `nixString`, `nixBool`, `nixMultiline`, `indent`, `hasAny` — exact implementations from config-template-engine-design.md.
3. Implement `RenderNix(templateName string, data any) ([]byte, error)` — loads from embed.FS, applies nixFuncs, executes.
4. Implement `RenderMarkdown(templateName string, data any) ([]byte, error)` — loads from embed.FS, executes with standard FuncMap.
5. Create embed.FS declarations in devenv and claudecode addon packages (actual templates are placeholder `.tmpl` files for now — real content comes in Phases 2 and 3).
6. Write unit tests: `nixString` escapes `${HOME}` correctly, `nixList` produces valid Nix syntax, `nixBool` maps Go bool to Nix bool, `nixMultiline` escapes `${}`→`''${}`, `indent` preserves empty lines.
7. Write integration test: render a small test template with all FuncMap functions and verify output is valid Nix (parse with `nix-instantiate --parse` if available).

**Acceptance Criteria:**
- [ ] `nixString("hello ${world}")` → `"hello \${world}"`
- [ ] `nixList(["git", "curl", "jq"])` → `[ pkgs.git pkgs.curl pkgs.jq ]`
- [ ] `nixBool(true)` → `"true"`, `nixBool(false)` → `"false"`
- [ ] `nixMultiline("echo ${var}")` → `"echo ''${var}"`
- [ ] `indent(4, "line1\nline2")` → `"    line1\n    line2"`
- [ ] Templates load from embed.FS
- [ ] Rendered Nix passes `nix-instantiate --parse` (when available)
- [ ] Unit tests pass

**Research Citations:**
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Nix Code via text/template` — nixFuncs FuncMap with implementations
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Template Organization` — embed.FS directory structure

**Status:** Not Started

---

### Unit 1.5: Generation Pipeline — Atomic Writes & Validation

**Description:** Implement the `GeneratedFile` pipeline: collect files from generators, validate each, write atomically (temp-file-then-rename), and report results.

**Context:** Multiple generators (devenv, claudecode) each produce `[]GeneratedFile`. The pipeline orchestrates: collect → validate → preview → confirm → write atomically → report. Atomic writes prevent corruption on crash. Validation catches template bugs before files hit disk.

**Desired Outcome:** A `generate` package that accepts `[]GeneratedFile`, validates, writes atomically, and returns a structured result with success/failure per file.

**Steps:**
1. Create `addons/devinit/generate/` package.
2. Implement `WriteFiles(files []GeneratedFile, projectRoot string) (WriteResult, error)`:
   - For each file: validate → write to temp file in same directory → `os.Rename` to final path.
   - On any failure: clean up all temp files (but don't roll back already-written files — too complex for v1).
3. Implement validation dispatch per file extension:
   - `.nix` → run `nix-instantiate --parse` (skip if not available, warn)
   - `.yaml`/`.yml` → YAML round-trip (`yaml.Unmarshal` then check no error)
   - `.json` → JSON round-trip (`json.Unmarshal` then check no error)
   - `.sh`/`.envrc` → run `bash -n` (syntax check, skip if not available)
   - `.md` → no validation (free-form)
4. Implement `WriteResult` struct: `Files []FileResult` with `Path`, `Action` (created/updated/skipped), `Error`.
5. Implement `PreviewFiles(files []GeneratedFile) string` — format a human-readable plan preview showing what will be created/updated.
6. Implement `writeFileAtomic(path string, content []byte, mode os.FileMode) error` per config-template-engine-design.md.
7. Write unit tests: atomic write survives concurrent read, validation catches bad JSON, validation passes good Nix.

**Acceptance Criteria:**
- [ ] Atomic write: partial failure doesn't leave corrupt files
- [ ] JSON validation catches `{"trailing": "comma",}` (actually valid JSON doesn't have trailing commas — test with malformed JSON)
- [ ] YAML validation catches indentation errors
- [ ] Nix validation delegates to `nix-instantiate --parse` when available
- [ ] Bash validation delegates to `bash -n` when available
- [ ] Missing validators produce warnings, not errors
- [ ] `PreviewFiles` produces readable output listing all files with actions
- [ ] `WriteResult` reports per-file success/failure
- [ ] Unit tests pass

**Research Citations:**
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Generation Pipeline` — GeneratedFile pipeline, atomic writes, validation table
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Atomic Writes` — temp-file-then-rename pattern

**Status:** Not Started

---

### Unit 1.6: Hash Tracking & GeneratedState Persistence

**Description:** Implement SHA256 hash tracking for generated files, persisting `GeneratedState` to gdev's config file, enabling the migration strategy in Phase 6.

**Context:** Every file written by the generation pipeline gets its SHA256 recorded. On subsequent runs, comparing the current file hash against the stored hash tells us whether the user modified it. This drives merge strategy decisions: unchanged files can be safely regenerated, modified files need merge/diff.

**Desired Outcome:** A `state` package that reads/writes `GeneratedState` from gdev's config, computes file hashes, and reports modification status per file.

**Steps:**
1. Create `addons/devinit/state/` package.
2. Implement `ComputeHash(content []byte) string` — returns `"sha256:<hex>"`.
3. Implement `RecordFiles(files []GeneratedFile) GeneratedState` — hash each file's content, record path → hash mapping with timestamp.
4. Implement `LoadState(configPath string) (GeneratedState, error)` — read from gdev config YAML (`~/.config/<appname>.yaml` under the addon's namespace).
5. Implement `SaveState(configPath string, state GeneratedState) error` — write back to config.
6. Implement `CheckModified(state GeneratedState, projectRoot string) map[string]ModificationStatus` — for each tracked file, compare stored hash against current file hash. Returns `Unmodified`, `Modified`, `Deleted`, `New` per file.
7. Wire into generation pipeline: after `WriteFiles` succeeds, call `RecordFiles` and `SaveState`.
8. Write unit tests: hash computation is deterministic, modification detection works for each status.

**Acceptance Criteria:**
- [ ] `ComputeHash` produces consistent SHA256 for same content
- [ ] `RecordFiles` captures hash for every generated file
- [ ] `LoadState`/`SaveState` round-trip through YAML preserves all data
- [ ] `CheckModified` correctly identifies unmodified, modified, deleted, and new files
- [ ] Integration with generation pipeline: after write, state is persisted
- [ ] Unit tests pass

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Core Principle: Track What We Generated` — SHA256 hash tracking
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Tracking Mechanism` — GeneratedState YAML structure

**Status:** Not Started

---

### Unit 1.7: Ecosystem Module Interface

**Description:** Define the `EcosystemModule` interface that all 27 language/platform modules implement, plus a module registry for discovery and composition.

**Context:** Instead of hardcoding language support, each ecosystem (Go, Java, TypeScript, Terraform, etc.) is a self-contained module implementing a common interface. This enables adding new ecosystems without modifying core code and keeps each ecosystem's logic (detection, templates, security configs, hooks) co-located. The module registry supports tier-based discovery — Tier 1 modules ship first, Tiers 2-4 follow.

**Desired Outcome:** A module interface and registry that Phase 2 ecosystem modules implement. The registry supports detection delegation, template composition, and config generation dispatch.

**Steps:**
1. Define `EcosystemModule` interface:
   ```go
   type EcosystemModule interface {
       Name() string                                          // e.g., "go", "java", "terraform"
       DisplayName() string                                   // e.g., "Go", "Java/Kotlin", "Terraform"
       Tier() int                                             // 1-4 priority tier
       Detect(projectRoot string) DetectionResult             // scan for this ecosystem's markers
       DevenvNixFragment(config ModuleConfig) (string, error) // Nix template fragment for devenv.nix
       DevenvYamlInputs(config ModuleConfig) []DevenvInput    // extra inputs needed in devenv.yaml
       SecurityConfigs(config ModuleConfig) []GeneratedFile   // .npmrc, pip.conf, settings.xml, etc.
       PreCommitHooks(config ModuleConfig) []HookConfig       // hooks for this ecosystem
       DenyRules(config ModuleConfig) []string                // Claude Code deny rule patterns
       CICommands(config ModuleConfig) []CICommand            // frozen-install, audit commands for CI
       PackageManagers() []PackageManagerInfo                 // metadata about supported package managers
       WizardFields() []WizardField                           // ecosystem-specific wizard questions
   }
   ```
2. Define `ModuleConfig` struct — the ecosystem-specific portion of `WizardAnswers` (version, package manager choice, service-specific settings).
3. Define `DetectionResult` struct — `Detected bool`, `Confidence` (certain/probable/absent), `Evidence []string`, `SuggestedConfig ModuleConfig`.
4. Define `ModuleRegistry` — `Register(module)`, `All() []EcosystemModule`, `ByTier(tier int) []`, `ByName(name string)`, `DetectAll(root string) []DetectionResult`.
5. Implement registration via `init()` functions in each module package — modules self-register on import.
6. Write unit tests: registry discovers modules, DetectAll returns correct results.

**Acceptance Criteria:**
- [ ] `EcosystemModule` interface defined with all methods
- [ ] `ModuleRegistry` supports registration, lookup by name, lookup by tier
- [ ] `DetectAll` delegates to all registered modules and returns aggregated results
- [ ] Interface is sufficient for the 8 Tier 1 ecosystems (validated against their detection/generation needs)
- [ ] Module self-registration via `init()` works
- [ ] Unit tests pass

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Implementation Priority Matrix` — 4-tier ecosystem classification
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Addon Composition Model` — extensible architecture pattern

**Status:** Not Started

---

### Unit 1.8: Infrastructure Profile Types

**Description:** Define the profile types for organization-wide infrastructure choices: registry proxy, Nix cache, build cache, scanning tools, and dependency update strategy.

**Context:** A consulting firm needs to encode infrastructure decisions once and deploy them across all projects. The profile system captures these choices — which registry proxy (Nexus, Artifactory, none), which Nix cache (Cachix, Attic, none), which scanning tools (OSV, Snyk, Socket), etc. The default consulting profile uses a $0/mo stack (Nexus Community + Cachix free + sccache + OSV Scanner + Renovate).

**Desired Outcome:** Profile types that encode infrastructure choices and generate the corresponding environment variables and config files.

**Steps:**
1. Define `InfraProfile` struct:
   ```go
   type InfraProfile struct {
       Registry    RegistryConfig    `yaml:"registry"`
       NixCache    NixCacheConfig    `yaml:"nix_cache"`
       BuildCache  BuildCacheConfig  `yaml:"build_cache"`
       Scanning    ScanningConfig    `yaml:"scanning"`
       Updates     UpdateConfig      `yaml:"updates"`
       SBOM        SBOMConfig        `yaml:"sbom"`
   }
   ```
2. Define sub-configs: `RegistryConfig` (type: nexus/artifactory/github/gitlab/aws/gcp/azure/verdaccio/artifact-keeper/none, URL, ecosystems, credentials env vars), `NixCacheConfig` (type: cachix/attic/nix-serve/none, URL, signing key ref), `BuildCacheConfig` (type: sccache/ccache/turborepo/nx/bazel-remote/none, backend, credentials), `ScanningConfig` (vulnerability: osv/snyk/grype, behavioral: socket/none, ci-protection: harden-runner/none), `UpdateConfig` (type: renovate/dependabot, age-gating days), `SBOMConfig` (generator: syft/sbomnix/none, signing: cosign/none).
3. Define built-in profiles: `consulting-default` ($0/mo stack), `startup-github` (GitHub-native), `enterprise` (JFrog + Snyk + full stack).
4. Implement `InfraProfile.EnvironmentVars() map[string]string` — generate env vars for devenv.nix.
5. Implement `InfraProfile.ConfigFiles() []GeneratedFile` — generate renovate.json, .github/dependabot.yml, etc.
6. Support profile loading from gdev config YAML and from `Configure()` options.

**Acceptance Criteria:**
- [ ] `InfraProfile` captures all infrastructure choices from artifacts/artifact-stores-caches-research.md
- [ ] `consulting-default` profile generates a working $0/mo stack
- [ ] `EnvironmentVars()` produces correct vars for each registry/cache/scanner combination
- [ ] `ConfigFiles()` generates renovate.json or dependabot.yml based on update strategy
- [ ] Profiles are YAML-serializable for team sharing
- [ ] Unit tests verify env var and config file generation for each built-in profile

**Research Citations:**
- `artifacts/artifact-stores-caches-research.md § Recommended Stack by Organization Profile` — consulting/startup/enterprise profiles
- `artifacts/artifact-stores-caches-research.md § Environment Variables` — complete env var reference
- `artifacts/artifact-stores-caches-research.md § Config File Generation` — per-ecosystem registry configs

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All eight units pass acceptance criteria
- [ ] `go build ./...` and `go vet ./...` pass
- [ ] Test binary registers all three addons and starts without error
- [ ] `EcosystemModule` interface supports all Tier 1 ecosystem needs
- [ ] Module registry discovers and dispatches to registered modules
- [ ] Detection engine returns correct results for test fixture directories
- [ ] Template engine renders valid Nix from a test template
- [ ] Generation pipeline writes files atomically with validation
- [ ] Hash tracking persists and detects modifications
- [ ] Infrastructure profiles generate correct env vars and config files
- [ ] No security-sensitive data (credentials, API keys) in any generated code
