# Phase 13: Project Configuration & Team Standards

## Goal

Implement a project-level configuration file (`.qsdev.yaml`) that captures project settings, team standards, and client-specific profiles, enabling a three-layer configuration resolution system (binary defaults -> project config -> local overrides). Add four onboarding modes to `qsdev init` (Create/Join/Update/Repair) so returning engineers get a working environment in under 2 minutes. Implement `qsdev check` as a CI enforcement command.

## Dependencies

Phase 1 complete (shared types, ecosystem module interface, detection engine). Phase 6 complete (wizard orchestration, profile system, huh forms). Phase 8 complete (migration infrastructure, update command, merge strategies). Phase 12 complete (tool lifecycle management, file ownership registry).

## Phase Outputs

- `.qsdev.yaml` project configuration file format and parser
- `.qsdev.local.yaml` gitignored developer overrides
- Three-layer config resolution engine with security floor enforcement
- Config schema versioning (`version: 1`) with migration chain
- `gdev_version` semver constraint (Terraform pattern)
- Four onboarding modes in `qsdev init` (Create/Join/Update/Repair)
- `qsdev check` CI enforcement command with 5 check categories
- Client-specific profiles with compliance levels (baseline/enhanced/strict)

---

### Unit 13.1: .qsdev.yaml Schema & Parser

**Description:** Define the YAML schema for `.qsdev.yaml`, the project-level configuration file that captures all project settings, team standards, and client-specific profiles. Implement Go struct types with YAML tags, a schema validator, and versioned parsing.

**Context:** The three-layer configuration hierarchy (org defaults -> `.qsdev.yaml` -> `.qsdev.local.yaml`) was designed in the team-config-sharing research. `.qsdev.yaml` is the middle layer: checked into git, shared across the team, and the primary source of truth for project configuration. It follows the file-in-repo pattern used by mise (`.mise.toml`), proto (`.prototools`), and EditorConfig. The `version` field (integer) tracks config schema versions separately from the gdev binary version, following JSON Schema versioning best practices. The `gdev_version` field (semver constraint string) follows the Terraform `required_version` pattern.

**Code-Grounded Note:** The current codebase already saves wizard answers at `.devinit/.qsdev-init-answers.yaml` (see `addons/devinit/answers.go:24-25`) and state at `.devinit/.qsdev-init-state.yaml`. The new `.qsdev.yaml` is a PUBLIC-FACING project config that coexists alongside these internal files. `.qsdev.yaml` captures project intent (profile, ecosystems, tools, compliance level); `.devinit/` answers capture full wizard details (every question answered during `qsdev init`). The config resolution engine (Unit 13.2) reads `.qsdev.yaml` first and uses it to seed `WizardAnswers`, which then flows through the existing generation pipeline unchanged.

The schema must be forward-compatible: older binaries reading a v1 config with unknown fields should ignore them gracefully. Required fields are minimal (`version` only); everything else has compiled-in defaults.

**Desired Outcome:** A complete `.qsdev.yaml` schema definition with Go types, a parser that loads and validates the file, and clear error messages for schema violations. The parser handles missing optional fields by returning compiled defaults, and rejects unknown `version` values with actionable upgrade instructions.

**Steps:**
1. Define the root `GdevConfig` struct in `pkg/types/config.go`:
   ```go
   // GdevConfig is the schema for .qsdev.yaml project configuration.
   type GdevConfig struct {
       // Schema version (integer, required). Determines which parser/migrator to use.
       Version int `yaml:"version" validate:"required,min=1"`

       // Semver constraint on compatible gdev binary version.
       // Examples: ">= 0.15.0", "~> 0.16", "^0.15.0"
       GdevVersion string `yaml:"gdev_version,omitempty"`

       // Named profile to use as base configuration (from compiled profiles).
       // Overrides org defaults; overridden by explicit fields below.
       Profile string `yaml:"profile,omitempty"`

       // Language/runtime configuration.
       Languages []LanguageConfig `yaml:"languages,omitempty"`

       // Service dependencies (postgres, redis, etc.).
       Services []ServiceConfig `yaml:"services,omitempty"`

       // Security settings.
       Security SecurityConfig `yaml:"security,omitempty"`

       // Tool enablement overrides (complement to qsdev enable/disable).
       Tools ToolsConfig `yaml:"tools,omitempty"`

       // Claude Code settings.
       ClaudeCode ClaudeCodeConfig `yaml:"claude_code,omitempty"`

       // Infrastructure settings (registry proxy, caches, etc.).
       Infrastructure InfraConfig `yaml:"infrastructure,omitempty"`

       // Client-specific profile (consulting lifecycle).
       Client *ClientConfig `yaml:"client,omitempty"`
   }
   ```
2. Define nested config structs:
   ```go
   type LanguageConfig struct {
       Name    string `yaml:"name" validate:"required"`
       Version string `yaml:"version,omitempty"`
   }

   type ServiceConfig struct {
       Name    string            `yaml:"name" validate:"required"`
       Version string            `yaml:"version,omitempty"`
       Options map[string]string `yaml:"options,omitempty"`
   }

   type SecurityConfig struct {
       // Compliance level: "baseline", "enhanced", "strict"
       Level          string `yaml:"level,omitempty"`
       AgeGating      *bool  `yaml:"age_gating,omitempty"`
       ScriptBlocking *bool  `yaml:"script_blocking,omitempty"`
       LockEnforce    *bool  `yaml:"lock_enforcement,omitempty"`
       VulnScanning   *bool  `yaml:"vuln_scanning,omitempty"`
   }

   type ToolsConfig struct {
       // Explicitly enabled tools (additive to profile defaults).
       Enabled []string `yaml:"enabled,omitempty"`
       // Explicitly disabled tools (overrides profile defaults).
       Disabled []string `yaml:"disabled,omitempty"`
       // Per-tool configuration overrides.
       Config map[string]map[string]any `yaml:"config,omitempty"`
   }

   type ClaudeCodeConfig struct {
       Enabled         *bool    `yaml:"enabled,omitempty"`
       PermissionLevel string   `yaml:"permission_level,omitempty"` // "standard", "restricted", "permissive"
       Skills          []string `yaml:"skills,omitempty"`
       MCPServers      []string `yaml:"mcp_servers,omitempty"`
   }

   type InfraConfig struct {
       RegistryProxy string `yaml:"registry_proxy,omitempty"`
       NixCache      string `yaml:"nix_cache,omitempty"`
       BuildCache    string `yaml:"build_cache,omitempty"`
   }

   type ClientConfig struct {
       Name             string   `yaml:"name" validate:"required"`
       Compliance       []string `yaml:"compliance,omitempty"`       // ["soc2", "hipaa"]
       SecurityLevel    string   `yaml:"security_level,omitempty"`   // overrides Security.Level
       RegistryProxy    string   `yaml:"registry_proxy,omitempty"`   // client-specific override
       NixCache         string   `yaml:"nix_cache,omitempty"`
       AllowedMCP       []string `yaml:"allowed_mcp_servers,omitempty"`
       BlockedMCP       []string `yaml:"blocked_mcp_servers,omitempty"`
       DataClass        string   `yaml:"data_classification,omitempty"` // "public", "internal", "confidential"
   }
   ```
3. Implement `ParseGdevConfig(path string) (*GdevConfig, error)`:
   - Read file, unmarshal YAML.
   - Check `version` field first. If missing: error with "`.qsdev.yaml` must include a `version` field (e.g., `version: 1`)".
   - If `version` > `MaxSupportedVersion`: error with "This `.qsdev.yaml` uses config version N, but your gdev only supports up to version M. Run `qsdev self-update`."
   - If `version` < `MinSupportedVersion`: error with "Config version N is no longer supported. Minimum supported version is M. Run `qsdev config migrate` with gdev >= X.Y.Z."
   - Run struct validation (go-playground/validator or equivalent).
   - Return parsed config.
4. Implement `ValidateGdevConfig(cfg *GdevConfig) []ValidationError`:
   - Validate profile name exists in compiled profile registry.
   - Validate language names against known ecosystem modules.
   - Validate service names against known service templates.
   - Validate security level is one of "baseline", "enhanced", "strict".
   - Validate tool names in `tools.enabled`/`tools.disabled` against tool registry.
   - Validate `gdev_version` is a parseable semver constraint (syntax check only; actual version comparison is in Unit 13.5).
   - Return structured errors with field paths and suggestions.
5. Implement `DefaultGdevConfig() *GdevConfig`:
   - Returns the compiled org defaults as a `GdevConfig` with `Version: 1`.
   - Security defaults: `level: "enhanced"`, all hardening features enabled.
   - Infrastructure defaults from compiled org profile.
   - Tools: profile-default set enabled.
6. Define schema version constants:
   ```go
   const (
       ConfigVersionMin     = 1
       ConfigVersionMax     = 1
       ConfigVersionCurrent = 1
   )
   ```
7. Write unit tests:
   - Valid config parses without error.
   - Missing `version` produces clear error.
   - Unknown version produces upgrade error.
   - Unknown fields are silently ignored (forward compatibility).
   - Invalid enum values (e.g., `security.level: "mega"`) produce clear error.
   - Minimal config (`version: 1` only) parses with all defaults.

**Acceptance Criteria:**
- [ ] `GdevConfig` struct with YAML tags covers all fields from the research design
- [ ] `ParseGdevConfig` loads valid `.qsdev.yaml` files and returns structured config
- [ ] Missing `version` field produces a clear, actionable error message
- [ ] Unknown config version produces an error with `qsdev self-update` instructions
- [ ] Unknown YAML fields are silently ignored (forward compatibility)
- [ ] `ValidateGdevConfig` reports all validation errors with field paths
- [ ] `DefaultGdevConfig` returns compiled org defaults
- [ ] Minimal config (just `version: 1`) parses successfully using defaults for all other fields
- [ ] Unit tests cover valid config, missing version, unknown version, invalid enum, unknown fields, and minimal config

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/team-config-sharing-research.md` -- three-layer hierarchy design, `.qsdev.yaml` field definitions
- `research-spikes/gdev-team-config-onboarding/config-versioning-drift-research.md` -- `version` (integer) and `gdev_version` (semver constraint) field design
- `research-spikes/gdev-team-config-onboarding/consulting-lifecycle-research.md` -- `client` block schema, compliance level mapping
- `research-spikes/gdev-extension-design/addon-architecture-design.md` -- profile system, config key namespacing

**Status:** Not Started

---

### Unit 13.2: Configuration Resolution Engine

**Description:** Implement the three-layer deep merge engine that resolves final configuration from binary compiled defaults, `.qsdev.yaml` project config, and `.qsdev.local.yaml` local overrides. Enforce security floors so local overrides cannot weaken project-level security settings.

**Context:** The three-layer resolution order is: org defaults (compiled into binary) -> profile (compiled, selected by `.qsdev.yaml` profile field) -> `.qsdev.yaml` overrides -> `.qsdev.local.yaml` overrides. Each layer deep-merges with the previous. The research established clear merge semantics: lists use union for additive fields (permissions, extra packages) and replacement for selective fields (languages, services); maps merge recursively; scalars use last-wins. The critical constraint is the security floor: the `security.level` acts as a minimum that cannot be lowered by lower-priority layers, and `client.blocked_mcp_servers` cannot be unblocked.

`.qsdev.local.yaml` uses the same schema as `.qsdev.yaml` minus the `version`, `gdev_version`, and `client` fields (those are project-level concerns, not developer-level). It is automatically added to `.gitignore` on first `qsdev init`.

**Code-Grounded Note:** The existing `MergeProfileWithFlags()` at `addons/devinit/profile_convert.go:72-121` already handles profile-to-answers merging and should be the model for `.qsdev.yaml`-to-answers merging. Its semantics: languages REPLACE entirely when specified; services APPEND. The resolution engine extends this same pattern to three layers (org -> project -> local) rather than inventing new merge logic. The output of resolution should produce a `WizardAnswers` that is indistinguishable from one produced by the interactive wizard, ensuring the downstream generation pipeline needs no changes.

**Desired Outcome:** `ResolveConfig()` produces a fully resolved `GdevConfig` that correctly merges all layers, enforces security floors, and provides verbose logging of which layer set each value (for debugging via `--verbose`).

**Steps:**
1. Define `LocalConfig` struct (subset of `GdevConfig` without project-level fields):
   ```go
   // LocalConfig is the schema for .qsdev.local.yaml (developer overrides).
   type LocalConfig struct {
       // Language overrides (e.g., different Go version for testing).
       Languages []LanguageConfig `yaml:"languages,omitempty"`
       // Additional services for local development.
       Services []ServiceConfig `yaml:"services,omitempty"`
       // Security overrides (floor-enforced: cannot lower below project level).
       Security SecurityConfig `yaml:"security,omitempty"`
       // Tool overrides.
       Tools ToolsConfig `yaml:"tools,omitempty"`
       // Claude Code overrides (e.g., permission_level for local work).
       ClaudeCode ClaudeCodeConfig `yaml:"claude_code,omitempty"`
       // Extra packages for personal tooling.
       ExtraPackages []string `yaml:"extra_packages,omitempty"`
   }
   ```
2. Implement `ParseLocalConfig(path string) (*LocalConfig, error)`:
   - Same YAML parsing as `ParseGdevConfig`, but no `version`/`gdev_version`/`client` fields.
   - If file does not exist, return nil (no local overrides).
3. Implement the resolution engine in `internal/config/resolve.go`:
   ```go
   type ResolutionTrace struct {
       Field  string // dot-path, e.g., "security.level"
       Value  any
       Source string // "org-default", "profile:go-web", "project:.qsdev.yaml", "local:.qsdev.local.yaml"
   }

   type ResolvedConfig struct {
       Config *GdevConfig
       Traces []ResolutionTrace // populated when verbose=true
   }

   func ResolveConfig(
       orgDefaults *GdevConfig,
       profile *GdevConfig,       // nil if no profile selected
       project *GdevConfig,       // nil if no .qsdev.yaml
       local *LocalConfig,        // nil if no .qsdev.local.yaml
       verbose bool,
   ) (*ResolvedConfig, error)
   ```
4. Implement deep merge with per-field semantics:
   - **Scalars** (strings, bools, ints): last non-zero value wins.
   - **Lists with union semantics** (`tools.enabled`, `claude_code.skills`, `claude_code.mcp_servers`, `extra_packages`): union of all layers, deduplicated.
   - **Lists with replacement semantics** (`languages`, `services`): later layer replaces the entire list if non-empty.
   - **Maps** (`tools.config`, `service.options`): recursive merge, later keys override earlier.
   - **Pointer fields** (`*bool`): nil means "use default from previous layer"; non-nil overrides.
5. Implement security floor enforcement:
   ```go
   func enforceSecurityFloor(resolved *GdevConfig, project *GdevConfig) {
       projectLevel := parseSecurityLevel(project.Security.Level)
       resolvedLevel := parseSecurityLevel(resolved.Security.Level)

       // Security level can only go up, never down
       if resolvedLevel < projectLevel {
           resolved.Security.Level = project.Security.Level
       }

       // Client blocked MCP servers cannot be unblocked
       if project.Client != nil {
           resolved.Client.BlockedMCP = union(
               project.Client.BlockedMCP,
               resolved.Client.BlockedMCP,
           )
       }

       // Security features cannot be disabled if project enables them
       if project.Security.AgeGating != nil && *project.Security.AgeGating {
           resolved.Security.AgeGating = project.Security.AgeGating
       }
       if project.Security.ScriptBlocking != nil && *project.Security.ScriptBlocking {
           resolved.Security.ScriptBlocking = project.Security.ScriptBlocking
       }
   }
   ```
6. Implement `.qsdev.local.yaml` auto-gitignore:
   - On `qsdev init`, check if `.gitignore` exists and contains `.qsdev.local.yaml`.
   - If not present, append `.qsdev.local.yaml` entry (use section markers from Phase 12 shared-file surgery).
7. Implement resolution logging for `--verbose`:
   - Track which layer provided each resolved value.
   - Print trace when `--verbose` flag is set: `security.level = "enhanced" (from project:.qsdev.yaml, overrode org-default:"baseline")`.
   - Include floor enforcement notes: `security.level = "enhanced" (floor enforced: local tried to set "baseline", project requires "enhanced")`.
8. Write `.qsdev.local.yaml` template generator:
   ```yaml
   # .qsdev.local.yaml — Local developer overrides (gitignored)
   # Uncomment and modify lines below to customize your local environment.
   # These settings override .qsdev.yaml but cannot lower security settings.
   #
   # extra_packages:
   #   - neovim
   #   - lazygit
   #   - ripgrep
   #
   # claude_code:
   #   permission_level: permissive  # your local preference
   #
   # tools:
   #   enabled:
   #     - changelog  # opt-in tools for your workflow
   ```
9. Write unit tests:
   - Scalar override: local `permission_level` overrides project.
   - Union list: local `extra_packages` merged with project packages.
   - Replacement list: local `languages` replaces project `languages`.
   - Security floor: local cannot lower `security.level` below project.
   - Security floor: local cannot disable `age_gating` when project enables it.
   - Blocked MCP: client blocked servers persist through all layers.
   - Missing layers: works with only org defaults (no project, no local).
   - Verbose mode: traces populated correctly.

**Acceptance Criteria:**
- [ ] Three-layer resolution: org defaults -> profile -> `.qsdev.yaml` -> `.qsdev.local.yaml`
- [ ] Deep merge with correct per-field semantics (union vs replacement vs recursive)
- [ ] Security floor enforcement: `.qsdev.local.yaml` cannot lower `security.level` below project setting
- [ ] Security floor enforcement: enabled security features (age_gating, script_blocking) cannot be disabled by local overrides
- [ ] Client blocked MCP servers cannot be unblocked by local or project overrides
- [ ] `.qsdev.local.yaml` auto-added to `.gitignore` on `qsdev init`
- [ ] `--verbose` shows which layer set each resolved value, including floor enforcement
- [ ] Template `.qsdev.local.yaml` generated with commented-out examples
- [ ] Resolution works with any combination of missing layers (no project, no local, no profile)
- [ ] Pointer fields (`*bool`) correctly distinguish "not set" from "explicitly false"

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/team-config-sharing-research.md` -- resolution order, merge semantics, security floor
- `research-spikes/gdev-team-config-onboarding/consulting-lifecycle-research.md` -- security level as floor, blocked MCP enforcement
- `research-spikes/gdev-extension-design/migration-strategy-design.md` -- `.qsdev.local.yaml` gitignore pattern

**Status:** Not Started

---

### Unit 13.3: Onboarding Mode Detection & Routing

**Description:** Implement the detection engine that determines which of four onboarding modes (Create/Join/Update/Repair) to use when `qsdev init` runs, and route to the appropriate wizard/workflow for each mode.

**Context:** The developer onboarding research identified four distinct scenarios when `qsdev init` runs. The detection engine must distinguish them based on: `.qsdev.yaml` presence, `.gdev/` state directory presence, binary version vs `gdev_version` constraint, and hash comparison of generated files. Each mode has radically different UX: Create runs the full wizard, Join skips most questions, Update shows a diff of what changed, and Repair shows a drift report with fix suggestions. Mode detection must be fast (under 500ms) because it runs before any user interaction.

The detection engine builds on Phase 1's `DetectedProject` (language/service detection) and Phase 8's `GeneratedState` (file hash tracking), adding the config-layer awareness needed for multi-developer workflows.

**Code-Grounded Note:** The existing `DetectExistingConfig()` at `addons/devinit/merge_mode.go:19-55` returns an `ExistingConfig` struct with a `NeedsMergeMode()` method, but it only blocks re-initialization (doesn't route to distinct modes). Phase 13 extends this with a new `DetectOnboardingMode()` function that returns a mode enum (Create/Join/Update/Repair). The existing `detect.Detect()` at `internal/detect/detect.go:12-83` already provides all the filesystem signals needed for mode detection (`HasDevenvNix`, `HasClaudeDir`, `HasPreCommitConfig`, etc.) and should be called as a sub-step of mode detection rather than duplicating its logic.

**Desired Outcome:** `qsdev init` automatically detects the correct mode, explains its choice to the user, and routes to the appropriate workflow. The mode selection is deterministic and explainable.

**Steps:**
1. Define the `OnboardingMode` type and detection result:
   ```go
   type OnboardingMode int
   const (
       ModeCreate OnboardingMode = iota // No .qsdev.yaml, fresh project setup
       ModeJoin                         // .qsdev.yaml exists, new developer on this machine
       ModeUpdate                       // .qsdev.yaml exists, newer gdev binary or templates
       ModeRepair                       // .qsdev.yaml exists, generated files have drifted
   )

   type ModeDetectionResult struct {
       Mode         OnboardingMode
       Reason       string         // Human-readable explanation
       ProjectState *ProjectState  // Full detection results
   }

   type ProjectState struct {
       // Config layer state
       HasGdevYaml       bool
       GdevConfig        *GdevConfig   // parsed if present
       GdevVersionCompat VersionCompat // satisfied, too-old, too-new
       ConfigVersion     int           // schema version found

       // Generated file state
       HasStateDir       bool                       // .gdev/ or .devinit/ state directory
       GeneratedFiles    map[string]GeneratedFileStatus
       DriftedFiles      []string                   // files that don't match expected hashes
       MissingFiles      []string                   // expected files that don't exist

       // Machine state
       InstalledTools    map[string]ToolStatus       // devenv, direnv, claude CLI
       DetectedProject   *DetectedProject            // from Phase 1 detection engine

       // Version state
       BinaryVersion     string
       LastRunVersion    string  // gdev version that last generated files
       TemplateVersion   string  // template version from state
   }

   type GeneratedFileStatus int
   const (
       FileMatchesExpected GeneratedFileStatus = iota
       FileUserModified
       FileDrifted        // modified but not by user (unexpected changes)
       FileMissing
       FileNew            // exists but not in state tracking
   )

   type VersionCompat int
   const (
       VersionSatisfied VersionCompat = iota
       VersionTooOld    // binary older than gdev_version constraint
       VersionTooNew    // binary has features not in config (safe, but may want update)
       VersionNoConstraint // no gdev_version in config
   )
   ```
2. Implement `DetectOnboardingMode(projectRoot string) (*ModeDetectionResult, error)`:
   ```go
   func DetectOnboardingMode(projectRoot string) (*ModeDetectionResult, error) {
       state := &ProjectState{}

       // Step 1: Check for .qsdev.yaml
       state.HasGdevYaml = fileExists(filepath.Join(projectRoot, ".qsdev.yaml"))

       if !state.HasGdevYaml {
           // No config file -> Create mode (full wizard)
           return &ModeDetectionResult{
               Mode:   ModeCreate,
               Reason: "No .qsdev.yaml found. Starting fresh project setup.",
               ProjectState: state,
           }, nil
       }

       // Step 2: Parse .qsdev.yaml
       cfg, err := ParseGdevConfig(filepath.Join(projectRoot, ".qsdev.yaml"))
       if err != nil {
           return nil, fmt.Errorf("invalid .qsdev.yaml: %w", err)
       }
       state.GdevConfig = cfg

       // Step 3: Check for state directory (has gdev been run here before?)
       state.HasStateDir = fileExists(filepath.Join(projectRoot, ".devinit"))

       if !state.HasStateDir {
           // Config exists but no state -> Join mode (new developer)
           return &ModeDetectionResult{
               Mode:   ModeJoin,
               Reason: "Found .qsdev.yaml but no local state. Setting up as new team member.",
               ProjectState: state,
           }, nil
       }

       // Step 4: Load existing state and check file hashes
       existingState, err := loadGeneratedState(projectRoot)
       if err != nil {
           return &ModeDetectionResult{
               Mode:   ModeRepair,
               Reason: "State file corrupted or unreadable. Running repair.",
               ProjectState: state,
           }, nil
       }

       // Step 5: Compare generated files against expected state
       state.DriftedFiles, state.MissingFiles = compareFileHashes(projectRoot, existingState, cfg)

       // Step 6: Check version state
       state.BinaryVersion = version.Current()
       state.LastRunVersion = existingState.GdevVersion

       // Step 7: Decide between Update and Repair
       if len(state.DriftedFiles) > 0 {
           return &ModeDetectionResult{
               Mode:   ModeRepair,
               Reason: fmt.Sprintf("Found %d files that have drifted from expected state.", len(state.DriftedFiles)),
               ProjectState: state,
           }, nil
       }

       if state.BinaryVersion != state.LastRunVersion || templatesUpdated(existingState) {
           return &ModeDetectionResult{
               Mode:   ModeUpdate,
               Reason: fmt.Sprintf("qsdev updated from %s to %s. Templates may have changed.", state.LastRunVersion, state.BinaryVersion),
               ProjectState: state,
           }, nil
       }

       // Everything matches -> Join mode (idempotent, just verify)
       return &ModeDetectionResult{
           Mode:   ModeJoin,
           Reason: "Project configuration is up to date. Verifying local state.",
           ProjectState: state,
       }, nil
   }
   ```
3. Implement user-facing mode explanation messages:
   - **Create:** "No .qsdev.yaml found. Let's set up this project."
   - **Join:** "Detected existing qsdev configuration (profile: {profile}). Running in Join mode -- verifying your local setup."
   - **Update:** "gdev has been updated ({old} -> {new}). {N} template updates available. Run with `--update` to apply."
   - **Repair:** "Found {N} files that have drifted from expected state. Review the drift report below."
4. Wire mode detection into the `qsdev init` command flow:
   ```go
   func runInit(cmd *cobra.Command, args []string) error {
       result, err := DetectOnboardingMode(projectRoot)
       if err != nil {
           return err
       }

       // Display mode explanation
       fmt.Println(result.Reason)

       switch result.Mode {
       case ModeCreate:
           return runCreateMode(result.ProjectState)  // Full wizard (Phase 6)
       case ModeJoin:
           return runJoinMode(result.ProjectState)     // Unit 13.4
       case ModeUpdate:
           return runUpdateMode(result.ProjectState)   // Phase 8 update flow
       case ModeRepair:
           return runRepairMode(result.ProjectState)    // Drift report + fix
       }
       return nil
   }
   ```
5. Implement Repair mode drift report:
   - List each drifted file with what changed.
   - Categorize drift: "section markers removed" (CLAUDE.md), "deny rule deleted" (settings.json), "package removed" (devenv.nix), "file deleted".
   - Offer auto-fix for safe issues: `qsdev init --repair` regenerates drifted machine-owned files.
   - For human-edited files with drift: show diff and suggest manual review.
6. Implement `--mode` flag for explicit override:
   - `qsdev init --mode create` forces Create mode even if `.qsdev.yaml` exists.
   - `qsdev init --mode join` forces Join mode.
   - `qsdev init --mode repair` forces Repair mode.
   - Useful for debugging or when auto-detection picks the wrong mode.
7. Write unit tests:
   - No `.qsdev.yaml` -> Create mode.
   - `.qsdev.yaml` present, no state dir -> Join mode.
   - `.qsdev.yaml` present, state dir present, all files match -> Join (idempotent).
   - `.qsdev.yaml` present, state dir present, version mismatch -> Update mode.
   - `.qsdev.yaml` present, state dir present, drifted files -> Repair mode.
   - Corrupt state file -> Repair mode (graceful degradation).
   - `--mode` flag overrides auto-detection.

**Acceptance Criteria:**
- [ ] Four modes correctly detected based on project state
- [ ] Mode detection completes in under 500ms
- [ ] Clear, user-facing explanation for each detected mode
- [ ] Create mode detected when no `.qsdev.yaml` exists
- [ ] Join mode detected when `.qsdev.yaml` exists but no local state directory
- [ ] Update mode detected when gdev binary version is newer than last run
- [ ] Repair mode detected when generated files have drifted from expected state
- [ ] Repair mode shows drift report with categorized changes
- [ ] `--mode` flag allows explicit override of auto-detection
- [ ] Corrupt or missing state file gracefully falls back to Repair mode
- [ ] Mode routing dispatches to correct workflow (Create -> wizard, Join -> Unit 13.4, Update -> Phase 8, Repair -> drift fix)

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/developer-onboarding-research.md` -- four onboarding modes, detection engine design, `ProjectState` struct
- `research-spikes/gdev-team-config-onboarding/config-versioning-drift-research.md` -- template drift detection, version ratchet
- `phases/08-migration-update-polish.md` -- update command and modification detection (Phase 8 flow reused by Update mode)

**Status:** Not Started

---

### Unit 13.4: Join Mode Implementation

**Description:** Implement the Join mode workflow: detect existing `.qsdev.yaml`, read project config, perform machine-specific setup, generate local files, and verify the environment is ready. Target: `git clone` + `cd` + `qsdev init` + `devenv shell` in under 2 minutes.

**Context:** Join mode is the most common onboarding scenario for a consulting firm -- an engineer cloning an existing project that already has `.qsdev.yaml` committed by another team member. The research established that Join mode should be near-silent: read the project config, check machine prerequisites, generate local-only files (`.qsdev.local.yaml` template), and verify everything is consistent. No wizard questions are needed because the project config already captures all decisions. The critical path is: parse config -> check binary version -> verify/install prerequisites -> generate local files -> verify generated file state -> done.

Machine-specific setup (devenv, direnv, claude CLI installation) is delegated to `qsdev devenv setup` (Phase 9 bootstrap steps). Join mode checks prerequisites and offers to run `qsdev devenv setup` if anything is missing, but does not duplicate the installation logic.

**Desired Outcome:** A returning engineer runs `git clone <url> && cd project && qsdev init` and has a working environment after `devenv shell`. Total hands-on time under 2 minutes. Join mode is quiet when everything is already set up (idempotent re-run).

**Steps:**
1. Implement `runJoinMode(state *ProjectState) error`:
   ```go
   func runJoinMode(state *ProjectState) error {
       cfg := state.GdevConfig

       // Step 1: Version compatibility check
       if state.GdevVersionCompat == VersionTooOld {
           return fmt.Errorf(
               "Your gdev version (%s) does not satisfy the project requirement (%s).\n"+
               "Run: gdev self-update",
               state.BinaryVersion, cfg.GdevVersion,
           )
       }

       // Step 2: Resolve full configuration
       resolved, err := ResolveConfig(DefaultGdevConfig(), profileFor(cfg.Profile), cfg, nil, false)
       if err != nil {
           return fmt.Errorf("config resolution failed: %w", err)
       }

       // Step 3: Check machine prerequisites
       missing := checkPrerequisites(resolved.Config)
       if len(missing) > 0 {
           fmt.Println("Missing prerequisites:")
           for _, m := range missing {
               fmt.Printf("  - %s: %s\n", m.Name, m.InstallHint)
           }
           if promptYN("Run qsdev devenv setup to install missing tools?") {
               if err := runSetup(missing); err != nil {
                   return fmt.Errorf("setup failed: %w", err)
               }
           }
       }

       // Step 4: Generate local-only files
       if err := writeLocalConfigTemplate(projectRoot); err != nil {
           return err
       }
       if err := ensureGitignoreEntry(projectRoot, ".qsdev.local.yaml"); err != nil {
           return err
       }

       // Step 5: Generate/verify project files from config
       generated, err := generateFromConfig(resolved.Config)
       if err != nil {
           return err
       }

       // Step 6: Compare against existing files
       report := compareGeneratedFiles(projectRoot, generated)
       if len(report.Drifted) > 0 {
           fmt.Println("\n⚠ Some generated files differ from expected state:")
           for _, d := range report.Drifted {
               fmt.Printf("  %s: %s\n", d.Path, d.Reason)
           }
           fmt.Println("Run `qsdev init --repair` to fix, or review manually.")
       }

       // Step 7: Write state tracking
       if err := writeGeneratedState(projectRoot, generated); err != nil {
           return err
       }

       // Step 8: Summary
       fmt.Println("\n✓ Project ready.")
       fmt.Printf("  Profile: %s\n", cfg.Profile)
       fmt.Printf("  Languages: %s\n", joinNames(resolved.Config.Languages))
       fmt.Printf("  Services: %s\n", joinNames(resolved.Config.Services))
       fmt.Println("\nRun `devenv shell` to enter the development environment.")

       return nil
   }
   ```
2. Implement prerequisite checking:
   ```go
   type Prerequisite struct {
       Name        string
       CheckFunc   func() bool           // returns true if installed
       InstallHint string                 // human-readable install instruction
       SetupStep   string                 // qsdev devenv setup step name, if available
       Required    bool                   // false = optional enhancement
   }

   func checkPrerequisites(cfg *GdevConfig) []Prerequisite {
       var missing []Prerequisite
       checks := []Prerequisite{
           {Name: "devenv", CheckFunc: hasDevenv, InstallHint: "nix profile install nixpkgs#devenv", Required: true},
           {Name: "direnv", CheckFunc: hasDirenv, InstallHint: "nix profile install nixpkgs#direnv", Required: true},
       }
       if cfg.ClaudeCode.Enabled == nil || *cfg.ClaudeCode.Enabled {
           checks = append(checks, Prerequisite{
               Name: "claude", CheckFunc: hasClaude,
               InstallHint: "See https://docs.anthropic.com/claude-code",
               Required: false,
           })
       }
       for _, c := range checks {
           if !c.CheckFunc() {
               missing = append(missing, c)
           }
       }
       return missing
   }
   ```
3. Implement local config template generation:
   - Generate `.qsdev.local.yaml` template only if it does not already exist.
   - Template includes commented-out examples relevant to the project's detected ecosystem.
   - If project uses Go: include Go-specific examples. If TypeScript: include TypeScript examples.
4. Implement generated file verification:
   - For each file that `.qsdev.yaml` would generate, check if it exists and matches.
   - Machine-owned files (settings.json deny rules, .pre-commit-config.yaml): verify required content present.
   - Human-edited files (devenv.nix): verify existence only, skip content check.
   - Report drift but do not auto-fix in Join mode (that's Repair mode).
5. Handle the idempotent re-run case:
   - If `qsdev init` in Join mode finds everything already set up, print a brief confirmation and exit.
   - Do not regenerate files that already match expected state.
   - Update state tracking timestamp only.
6. Wire into `--non-interactive` mode:
   - Join mode with `--yes` skips the prerequisite installation prompt and either auto-installs or fails.
   - `--skip-setup` flag skips prerequisite checks entirely (for CI where tools are pre-installed).
7. Write integration tests:
   - Fresh clone with `.qsdev.yaml` -> Join mode generates local template, reports ready.
   - Re-run on already-set-up project -> idempotent, no changes.
   - Missing devenv -> offers `qsdev devenv setup`.
   - Drifted settings.json -> warns but does not fix.
   - `--yes` mode -> auto-installs prerequisites without prompting.

**Acceptance Criteria:**
- [ ] Join mode reads `.qsdev.yaml` and resolves full config without wizard prompts
- [ ] Binary version checked against `gdev_version` constraint before proceeding
- [ ] Missing prerequisites detected and `qsdev devenv setup` offered
- [ ] `.qsdev.local.yaml` template generated with ecosystem-relevant commented examples
- [ ] `.qsdev.local.yaml` added to `.gitignore` if not already present
- [ ] Generated file state verified against expected output from config
- [ ] Drift reported as warnings (not auto-fixed in Join mode)
- [ ] Idempotent: re-running Join mode on an already-set-up project is a no-op
- [ ] `--yes` mode works without prompts (for scripted onboarding)
- [ ] Total Join mode execution time under 10 seconds (excluding `qsdev devenv setup` and `devenv shell`)
- [ ] Clear summary output showing profile, languages, services, and next steps

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/developer-onboarding-research.md` -- Join mode scenario, 3-commands-2-minutes target, machine vs project setup distinction
- `research-spikes/gdev-team-config-onboarding/team-config-sharing-research.md` -- `.qsdev.local.yaml` template, resolution order
- `phases/06-wizard-orchestration.md` -- bootstrap step registration (qsdev devenv setup integration)
- `phases/09-cross-platform-system-detection.md` -- `qsdev devenv doctor`/`qsdev devenv setup` for prerequisite management

**Status:** Not Started

---

### Unit 13.5: gdev_version Constraint & Schema Migration

**Description:** Implement semver constraint parsing for the `gdev_version` field, binary version checking at `qsdev init` startup, version ratchet enforcement, and the incremental config schema migration chain.

**Context:** The config versioning research established a Terraform-inspired pattern: `.qsdev.yaml` declares `gdev_version: ">= 0.15.0"` as a semver constraint, and qsdev checks this before any operation. Additionally, the `version` field (integer) tracks config schema versions separately. Three version axes can drift independently: binary version, config schema version, and template version. The version ratchet strategy prevents older binaries from downgrading files generated by newer binaries. Config migrations chain incrementally (v1 -> v2 -> v3), never skip versions, and run in-memory only (the file on disk is not rewritten unless the user explicitly runs `qsdev config migrate`).

**Code-Grounded Note:** `go.mod` has NO semver library currently. This phase must add `github.com/Masterminds/semver/v3` as a dependency (well-maintained, Terraform/Helm use it), or implement minimal constraint matching inline. The `WizardAnswers` struct at `pkg/types/types.go:11-32` has no version field -- a `ConfigVersion string` field should be added to track which config schema version produced the answers, enabling the ratchet check.

**Desired Outcome:** Incompatible gdev versions are caught before any destructive operation with clear, actionable error messages. Config schema migrations run automatically in-memory and optionally persist to disk.

**Steps:**
1. Implement semver constraint parsing in `internal/config/version.go`:
   ```go
   // Constraint represents a version requirement from .qsdev.yaml gdev_version field.
   type Constraint struct {
       Raw        string           // original string, e.g., ">= 0.15.0"
       Conditions []ConditionGroup // parsed conditions (AND groups of OR comparisons)
   }

   type ConditionGroup struct {
       Operator string // ">=", "<=", ">", "<", "=", "!=", "~>", "^"
       Version  semver.Version
   }

   // ParseConstraint parses a Terraform-style version constraint string.
   // Supported operators: =, !=, >, >=, <, <=, ~> (pessimistic), ^ (compatible)
   // Comma-separated conditions are AND'd: ">= 0.15.0, < 1.0.0"
   func ParseConstraint(raw string) (*Constraint, error)

   // Check returns true if the given version satisfies the constraint.
   func (c *Constraint) Check(v semver.Version) bool
   ```
2. Implement operator semantics:
   - `>=`, `<=`, `>`, `<`, `=`, `!=`: standard semver comparison.
   - `~>` (pessimistic constraint, Terraform style): `~> 0.15` matches `>= 0.15.0, < 0.16.0`; `~> 0.15.3` matches `>= 0.15.3, < 0.16.0`.
   - `^` (compatible constraint, npm/Cargo style): `^0.15.0` matches `>= 0.15.0, < 0.16.0` (for 0.x); `^1.2.3` matches `>= 1.2.3, < 2.0.0`.
   - Pre-release versions: only match exact pins (e.g., `= 0.16.0-rc1`).
3. Implement binary version check as early gate:
   ```go
   func checkBinaryVersion(cfg *GdevConfig) error {
       if cfg.GdevVersion == "" {
           return nil // no constraint specified
       }

       constraint, err := ParseConstraint(cfg.GdevVersion)
       if err != nil {
           return fmt.Errorf("invalid gdev_version constraint %q: %w", cfg.GdevVersion, err)
       }

       current, err := semver.Parse(version.Current())
       if err != nil {
           return fmt.Errorf("cannot parse binary version %q: %w", version.Current(), err)
       }

       if !constraint.Check(current) {
           return &VersionMismatchError{
               BinaryVersion:    version.Current(),
               Constraint:       cfg.GdevVersion,
               UpgradeCommand:   "gdev self-update",
           }
       }
       return nil
   }
   ```
4. Implement `VersionMismatchError` with actionable output:
   ```go
   type VersionMismatchError struct {
       BinaryVersion  string
       Constraint     string
       UpgradeCommand string
   }

   func (e *VersionMismatchError) Error() string {
       return fmt.Sprintf(
           "gdev version mismatch\n"+
           "  Your version:  %s\n"+
           "  Required:      %s (from .qsdev.yaml gdev_version)\n\n"+
           "  Update with: %s\n"+
           "  Or override with --skip-version-check (not recommended)",
           e.BinaryVersion, e.Constraint, e.UpgradeCommand,
       )
   }
   ```
5. Implement version ratchet in state tracking:
   ```go
   // CheckVersionRatchet prevents an older binary from overwriting files
   // generated by a newer binary.
   func CheckVersionRatchet(state *GeneratedState, currentVersion string) *RatchetWarning {
       current, _ := semver.Parse(currentVersion)
       lastRun, _ := semver.Parse(state.GdevVersion)

       if current.LT(lastRun) {
           // Older binary attempting to update files from newer binary
           return &RatchetWarning{
               CurrentVersion: currentVersion,
               LastRunVersion: state.GdevVersion,
               AffectedFiles:  state.FilesGeneratedAfter(currentVersion),
           }
       }
       return nil
   }
   ```
   - Default behavior: skip files generated by newer binary (safe).
   - `--force`: overwrite anyway with explicit confirmation.
   - `--bump-version` flag on `qsdev init --update`: updates `.qsdev.yaml` `gdev_version` to match current binary.
6. Implement config schema migration chain:
   ```go
   type Migration struct {
       FromVersion int
       ToVersion   int
       Description string
       Migrate     func(old map[string]any) (map[string]any, error)
   }

   // MigrationChain holds all registered migrations.
   var MigrationChain = []Migration{
       // Future: {1, 2, "Restructure languages from strings to objects", migrateV1toV2},
   }

   // MigrateConfig applies the migration chain to bring config to current version.
   // Returns the migrated config (in-memory only; does not write to disk).
   func MigrateConfig(raw map[string]any, fromVersion int) (map[string]any, error) {
       current := fromVersion
       for _, m := range MigrationChain {
           if m.FromVersion == current {
               migrated, err := m.Migrate(raw)
               if err != nil {
                   return nil, fmt.Errorf("migration v%d->v%d failed: %w", m.FromVersion, m.ToVersion, err)
               }
               raw = migrated
               current = m.ToVersion
           }
       }
       if current != ConfigVersionCurrent {
           return nil, fmt.Errorf("migration chain incomplete: reached v%d but current is v%d", current, ConfigVersionCurrent)
       }
       return raw, nil
   }
   ```
7. Implement `qsdev config migrate` command:
   - Reads `.qsdev.yaml`, applies migration chain, writes back with updated `version` field.
   - Shows diff before writing.
   - Requires `--write` flag to actually persist (dry-run by default).
8. Wire version check into all `gdev` subcommands:
   - `qsdev init`, `qsdev check`, `qsdev enable`, `qsdev disable`, `qsdev status` all check `gdev_version` constraint before proceeding.
   - `qsdev config migrate` exempted (must work even when constraint is not met, to upgrade the config).
9. Write comprehensive tests:
   - Constraint parsing: `">= 0.15.0"`, `"~> 0.16"`, `"^0.15.0"`, `">= 0.15.0, < 1.0.0"`.
   - Version satisfaction: edge cases at constraint boundaries.
   - `~>` semantics: `~> 0.15` allows 0.15.9 but not 0.16.0.
   - `^` semantics: `^0.15.0` allows 0.15.5 but not 0.16.0; `^1.2.3` allows 1.9.0 but not 2.0.0.
   - Version ratchet: older binary refuses to overwrite newer-generated files.
   - Migration chain: v1 -> v2 migration applied correctly.
   - Migration chain gap: missing migration detected and reported.
   - `--skip-version-check` flag bypasses constraint (with warning).

**Acceptance Criteria:**
- [ ] Semver constraint parsing supports `>=`, `<=`, `>`, `<`, `=`, `!=`, `~>`, `^` operators
- [ ] Comma-separated constraints AND'd correctly (e.g., `">= 0.15.0, < 1.0.0"`)
- [ ] Binary version checked against constraint before any `qsdev init` operation
- [ ] Version mismatch produces clear error with `qsdev self-update` instruction
- [ ] `--skip-version-check` bypasses constraint with a warning
- [ ] Version ratchet prevents older binary from overwriting newer-generated files
- [ ] `--force` overrides ratchet with explicit confirmation
- [ ] `--bump-version` updates `gdev_version` in `.qsdev.yaml` to match current binary
- [ ] Config migration chain applies incrementally (v1 -> v2 -> v3, not v1 -> v3)
- [ ] Migration runs in-memory by default; `qsdev config migrate --write` persists to disk
- [ ] Missing migration in chain detected and reported with actionable error
- [ ] All gdev subcommands (init, check, enable, disable, status) check version constraint

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/config-versioning-drift-research.md` -- three version axes, Terraform `required_version` pattern, migration chain design, ratchet strategy
- `research-spikes/gdev-team-config-onboarding/developer-onboarding-research.md` -- version mismatch scenario, actionable error messages

**Status:** Not Started

---

### Unit 13.6: qsdev check Command (CI Enforcement)

**Description:** Implement `qsdev check` as a read-only validation command that verifies project compliance against org policy. The command checks 5 categories (binary compatibility, config integrity, required tools, generated file state, security hardening), supports 4 output formats (human, JSON, SARIF, JUnit), and integrates with GitHub Actions annotations.

**Context:** The standards enforcement research established `qsdev check` as the CI enforcement point -- the one place where checks run consistently regardless of individual developer behavior. It complements `qsdev devenv doctor` (machine health) but focuses on project compliance. Generated configuration files can drift through manual edits, partial updates, version skew, or developers disabling pre-commit hooks. CI catches this drift before it reaches main.

The command is strictly read-only: it never modifies files, making it safe for CI and auditing. An `--auto-fix` flag provides a separate mode that fixes safe additive issues (missing gitignore entries, missing section markers). An `--audit-level` flag controls the exit code threshold so teams can gradually tighten enforcement.

**Code-Grounded Note:** No `--json` or machine-readable output exists on ANY current gdev command. This unit establishes the pattern for structured output across the CLI: a `--format` flag accepting `text|json|sarif|junit`, using `cmd.OutOrStdout()` for testability. Future commands (`qsdev status`, `qsdev devenv doctor`) should adopt this same pattern. The `toolcheck.Detect()` function at `internal/toolcheck/toolcheck.go:11-40` already checks tool binary availability and can be reused directly for the "Required Tools" check category.

**Desired Outcome:** `qsdev check` runs in CI (GitHub Actions and GitLab CI), produces structured output for code scanning dashboards, and fails the pipeline when the project does not meet org policy. Local runs show human-readable output with remediation suggestions.

**Steps:**
1. Define the check result types in `internal/check/types.go`:
   ```go
   type CheckCategory string
   const (
       CategoryBinaryCompat    CheckCategory = "binary_compatibility"
       CategoryConfigIntegrity CheckCategory = "config_integrity"
       CategoryRequiredTools   CheckCategory = "required_tools"
       CategoryFileState       CheckCategory = "generated_file_state"
       CategorySecurityHarden  CheckCategory = "security_hardening"
   )

   type CheckSeverity string
   const (
       SeverityCritical CheckSeverity = "critical"
       SeverityHigh     CheckSeverity = "high"
       SeverityMedium   CheckSeverity = "medium"
       SeverityLow      CheckSeverity = "low"
       SeverityInfo     CheckSeverity = "info"
   )

   type CheckResult struct {
       Category    CheckCategory `json:"category"`
       Name        string        `json:"name"`
       Status      string        `json:"status"`      // "pass", "fail", "skip", "warn"
       Severity    CheckSeverity `json:"severity"`
       Message     string        `json:"message"`
       Remediation string        `json:"remediation,omitempty"`
       FilePath    string        `json:"file_path,omitempty"`  // for SARIF location
       AutoFixable bool          `json:"auto_fixable"`
   }

   type CheckReport struct {
       Version   string        `json:"version"`
       Project   string        `json:"project"`
       Timestamp string        `json:"timestamp"`
       Checks    []CheckResult `json:"checks"`
       Summary   CheckSummary  `json:"summary"`
   }

   type CheckSummary struct {
       Total    int `json:"total"`
       Pass     int `json:"pass"`
       Fail     int `json:"fail"`
       Warn     int `json:"warn"`
       Skip     int `json:"skip"`
   }
   ```
2. Implement the 5 check categories:

   **Category 1: Binary Compatibility**
   ```go
   func checkBinaryCompatibility(cfg *GdevConfig) []CheckResult {
       results := []CheckResult{}
       // Check gdev_version constraint
       if cfg.GdevVersion != "" {
           constraint, err := ParseConstraint(cfg.GdevVersion)
           if err != nil {
               results = append(results, CheckResult{
                   Category: CategoryBinaryCompat,
                   Name:     "gdev_version_parseable",
                   Status:   "fail", Severity: SeverityCritical,
                   Message: fmt.Sprintf("Cannot parse gdev_version constraint: %s", cfg.GdevVersion),
               })
           } else if !constraint.Check(currentVersion()) {
               results = append(results, CheckResult{
                   Category: CategoryBinaryCompat,
                   Name:     "gdev_version_satisfied",
                   Status:   "fail", Severity: SeverityCritical,
                   Message: fmt.Sprintf("gdev %s does not satisfy %s", version.Current(), cfg.GdevVersion),
                   Remediation: "Run: gdev self-update",
               })
           } else {
               results = append(results, CheckResult{
                   Category: CategoryBinaryCompat,
                   Name:     "gdev_version_satisfied",
                   Status:   "pass", Severity: SeverityInfo,
                   Message: fmt.Sprintf("gdev %s satisfies %s", version.Current(), cfg.GdevVersion),
               })
           }
       }
       return results
   }
   ```

   **Category 2: Config Integrity**
   - `.qsdev.yaml` exists and parses without error.
   - Schema version is supported.
   - Profile name (if specified) exists in compiled profile registry.
   - All referenced language/service names are valid.
   - No YAML syntax errors (report line number on failure).

   **Category 3: Required Tools**
   - Org policy mandates certain tools. Check that they are enabled in the resolved config.
   - Check against compiled `OrgPolicy.RequiredTools` list.
   - If a required tool is in `tools.disabled`, fail with "tool X is required by org policy but explicitly disabled".
   - Check required pre-commit hooks are configured.

   **Category 4: Generated File State**
   - For each machine-owned file: hash comparison against expected output.
   - For settings.json: verify required deny rules are present (parse JSON, check `permissions.deny` array).
   - For CLAUDE.md: verify section markers are intact.
   - For .pre-commit-config.yaml: verify required hooks are present.
   - For devenv.nix: existence check only (human-edited, no content check).
   - Report files that are missing, extra, or modified.

   **Category 5: Security Hardening**
   - Per-ecosystem security configs present: `.npmrc` with age-gating for JS projects, `pip.conf` with age-gating for Python, etc.
   - Install script blocking configured for ecosystems that support it.
   - Lock file present for each detected ecosystem.
   - Vulnerability scanning configured in CI workflow.
   - SBOM generation configured (if compliance level requires it).

3. Implement output formatters:
   ```go
   type OutputFormat string
   const (
       FormatHuman OutputFormat = "human"
       FormatJSON  OutputFormat = "json"
       FormatSARIF OutputFormat = "sarif"
       FormatJUnit OutputFormat = "junit"
   )

   func FormatReport(report *CheckReport, format OutputFormat, w io.Writer) error
   ```
   - **Human format:** Grouped by category with color-coded status (green pass, red fail, yellow warn). Summary at bottom with critical issues listed first.
   - **JSON format:** Direct marshaling of `CheckReport` struct.
   - **SARIF format:** Static Analysis Results Interchange Format for GitHub Security tab. Each failed check maps to a SARIF result with location, message, and level.
   - **JUnit format:** JUnit XML with each category as a testsuite and each check as a testcase. Failed checks produce `<failure>` elements.
4. Implement `--audit-level` flag:
   ```go
   // AuditLevel controls which severity levels cause non-zero exit code.
   // "none": always exit 0 (reporting only)
   // "critical": exit 1 only for critical findings
   // "high": exit 1 for critical + high
   // "medium": exit 1 for critical + high + medium (default)
   // "low": exit 1 for any finding
   func shouldFail(results []CheckResult, auditLevel string) bool
   ```
5. Implement `--auto-fix` flag:
   - Only fixes issues where `AutoFixable: true`.
   - Safe additive fixes: missing gitignore entries, missing deny rules in settings.json, missing section markers, outdated managed skills/rules.
   - Does NOT fix: config structure changes, explicitly disabled security features, CI workflow changes.
   - Reports what was fixed and what remains.
   - Still outputs the full report after fixing (so CI can verify the fix was sufficient).
6. Implement GitHub Actions annotation integration:
   ```go
   func emitGitHubAnnotations(results []CheckResult) {
       for _, r := range results {
           if r.Status == "fail" {
               level := "error"
               if r.Severity == SeverityLow || r.Severity == SeverityMedium {
                   level = "warning"
               }
               if r.FilePath != "" {
                   fmt.Printf("::%s file=%s::%s: %s\n", level, r.FilePath, r.Name, r.Message)
               } else {
                   fmt.Printf("::%s::%s: %s\n", level, r.Name, r.Message)
               }
           }
       }
   }
   ```
   - Automatically enabled when `GITHUB_ACTIONS=true` environment variable is detected.
   - Annotations appear inline on PR diffs next to the affected file.
7. Wire the `qsdev check` command:
   ```go
   func runCheck(cmd *cobra.Command, args []string) error {
       cfg, err := ParseGdevConfig(filepath.Join(projectRoot, ".qsdev.yaml"))
       if err != nil {
           return fmt.Errorf("cannot read .qsdev.yaml: %w (exit code 2)", err)
       }

       resolved, err := ResolveConfig(DefaultGdevConfig(), profileFor(cfg.Profile), cfg, nil, false)
       if err != nil {
           return err
       }

       var allResults []CheckResult
       allResults = append(allResults, checkBinaryCompatibility(cfg)...)
       allResults = append(allResults, checkConfigIntegrity(cfg)...)
       allResults = append(allResults, checkRequiredTools(resolved.Config)...)
       allResults = append(allResults, checkFileState(projectRoot, resolved.Config)...)
       allResults = append(allResults, checkSecurityHardening(projectRoot, resolved.Config)...)

       if autoFix {
           allResults = applyAutoFixes(projectRoot, allResults)
       }

       report := buildReport(allResults)
       FormatReport(report, outputFormat, os.Stdout)

       if isGitHubActions() {
           emitGitHubAnnotations(allResults)
       }

       if shouldFail(allResults, auditLevel) {
           os.Exit(1)
       }
       return nil
   }
   ```
   - Exit code 0: all checks pass (at the configured audit level).
   - Exit code 1: one or more checks failed above the audit level threshold.
   - Exit code 2: `qsdev check` itself errored (cannot read config, invalid flags).
8. Provide CI workflow snippets in generated docs:

   **GitHub Actions:**
   ```yaml
   - name: Run qsdev check
     run: |
       qsdev check --format sarif --audit-level medium > gdev-check.sarif
   - name: Upload SARIF
     if: always()
     uses: github/codeql-action/upload-sarif@v3
     with:
       sarif_file: gdev-check.sarif
   ```

   **GitLab CI:**
   ```yaml
   gdev-check:
     stage: validate
     script:
       - qsdev check --format junit --audit-level medium > gdev-check.xml
     artifacts:
       reports:
         junit: gdev-check.xml
   ```

9. Write comprehensive tests:
   - All checks pass on a well-configured project.
   - Binary version mismatch detected.
   - Missing deny rule in settings.json detected.
   - Missing pre-commit hook detected.
   - Missing security config detected.
   - SARIF output is valid SARIF 2.1.0.
   - JUnit output is valid JUnit XML.
   - `--audit-level none` always exits 0.
   - `--audit-level critical` only fails on critical findings.
   - `--auto-fix` adds missing gitignore entry and re-checks.
   - GitHub annotations emitted when `GITHUB_ACTIONS=true`.

**Acceptance Criteria:**
- [ ] `qsdev check` is read-only: never modifies files (unless `--auto-fix` is explicitly passed)
- [ ] 5 check categories implemented: binary compatibility, config integrity, required tools, file state, security hardening
- [ ] Human output grouped by category with color-coded status
- [ ] `--format json` produces valid, parseable JSON matching `CheckReport` schema
- [ ] `--format sarif` produces valid SARIF 2.1.0 for GitHub Security tab upload
- [ ] `--format junit` produces valid JUnit XML for CI test reporting
- [ ] `--audit-level` controls exit code threshold (none/low/medium/high/critical)
- [ ] `--auto-fix` fixes safe additive issues (gitignore entries, deny rules, section markers)
- [ ] `--auto-fix` does not fix explicitly disabled security features or config structure
- [ ] GitHub Actions annotations emitted when `GITHUB_ACTIONS=true`
- [ ] Exit code 0 when all checks pass, 1 when checks fail, 2 on internal error
- [ ] Remediation suggestions provided for every failed check
- [ ] devenv.nix existence-only check (no content validation for human-edited files)

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/standards-enforcement-ci-research.md` -- 5 check categories, output formats, auto-fix scope, `qsdev check` vs `qsdev devenv doctor` distinction, CI integration examples
- `research-spikes/gdev-team-config-onboarding/config-versioning-drift-research.md` -- file hash comparison, version skew detection
- `research-spikes/gdev-extension-design/migration-strategy-design.md` -- hash-based modification detection

**Status:** Not Started

---

### Unit 13.7: Client-Specific Profiles & Compliance Levels

**Description:** Implement client-specific profile support with three compliance levels (baseline/enhanced/strict), profile inheritance, security floor enforcement, and built-in profiles for common consulting scenarios.

**Context:** The consulting lifecycle research identified that different clients have different security requirements (SOC2, HIPAA, FedRAMP), which map to different security configurations. Compliance levels act as security floors: a project on a strict client can add more restrictions but never loosen them. The `.qsdev.yaml` `client` block encodes client-specific settings that propagate through the resolution engine. Profile inheritance follows: org base -> client overlay -> project specifics -> local overrides. Built-in profiles encode common consulting patterns: `consulting-default` (enhanced security for typical engagements), `startup-fast` (baseline security for speed-focused POCs), and `enterprise` (strict security for regulated industries).

The compliance level determines concrete settings: age-gating thresholds, vulnerability scanning frequency, MCP server allowlists, pre-commit hook requirements, SBOM generation policy, and Claude Code permission levels. These are not just labels -- they map to specific configuration values that `qsdev check` can verify.

**Code-Grounded Note:** The existing `Profile` struct at `addons/devinit/config.go:11-24` has no `ComplianceLevel` field -- it must be added. Similarly, the `InfraProfile` at `internal/profile/types.go:94-210` also lacks compliance levels. Both structs need a `ComplianceLevel string` field (or equivalent) so that profiles can declare their security floor. The existing compiled profiles (go-web, ts-fullstack, etc.) will gain a compliance level field defaulting to `"enhanced"` for consulting profiles.

**Desired Outcome:** Teams set `profile: consulting-default` or `client.security_level: strict` in `.qsdev.yaml` and get the correct security posture without manually configuring dozens of individual settings. `qsdev check` verifies compliance against the selected level.

**Steps:**
1. Define compliance level configuration mappings:
   ```go
   type ComplianceProfile struct {
       Level                string
       Description          string
       AgeGatingThreshold   int      // hours
       ScriptBlocking       bool
       ScriptBlockAuditLog  bool
       VulnScanning         bool
       VulnScanFrequency    string   // "on-pr", "daily", "every-build"
       RequiredPreCommit    []string
       MCPServerPolicy      string   // "allow-list", "explicit-only"
       ClaudePermLevel      string   // "standard", "restricted"
       ClaudeAuditLog       bool
       SBOMPolicy           string   // "off", "on-release", "every-build"
       LicenseScanning      bool
   }

   var ComplianceLevels = map[string]ComplianceProfile{
       "baseline": {
           Level:              "baseline",
           Description:        "Minimum viable security for POCs and internal projects",
           AgeGatingThreshold: 72,  // 3 days
           ScriptBlocking:     true,
           VulnScanning:       true,
           VulnScanFrequency:  "on-pr",
           RequiredPreCommit:  []string{"ripsecrets", "gitleaks"},
           MCPServerPolicy:    "allow-list",
           ClaudePermLevel:    "standard",
           SBOMPolicy:         "off",
           LicenseScanning:    false,
       },
       "enhanced": {
           Level:              "enhanced",
           Description:        "Recommended for client engagements with SOC2 or equivalent",
           AgeGatingThreshold: 168, // 1 week
           ScriptBlocking:     true,
           VulnScanning:       true,
           VulnScanFrequency:  "on-pr",
           RequiredPreCommit:  []string{"ripsecrets", "gitleaks", "semgrep"},
           MCPServerPolicy:    "allow-list",
           ClaudePermLevel:    "standard",
           SBOMPolicy:         "on-release",
           LicenseScanning:    false,
       },
       "strict": {
           Level:              "strict",
           Description:        "Maximum hardening for HIPAA, FedRAMP, or high-security clients",
           AgeGatingThreshold: 336, // 2 weeks
           ScriptBlocking:     true,
           ScriptBlockAuditLog: true,
           VulnScanning:       true,
           VulnScanFrequency:  "daily",
           RequiredPreCommit:  []string{"ripsecrets", "gitleaks", "semgrep", "license-compliance"},
           MCPServerPolicy:    "explicit-only",
           ClaudePermLevel:    "restricted",
           ClaudeAuditLog:     true,
           SBOMPolicy:         "every-build",
           LicenseScanning:    true,
       },
   }
   ```
2. Define built-in project profiles with compliance levels:
   ```go
   var BuiltInProfiles = map[string]GdevConfig{
       "consulting-default": {
           Version: ConfigVersionCurrent,
           Profile: "consulting-default",
           Security: SecurityConfig{
               Level: "enhanced",
           },
           ClaudeCode: ClaudeCodeConfig{
               Enabled:         boolPtr(true),
               PermissionLevel: "standard",
               Skills:          []string{"security-review", "agent-postmortem"},
               MCPServers:      []string{"context7", "github"},
           },
           Tools: ToolsConfig{
               Enabled: []string{"semgrep", "gitleaks", "secretspec"},
           },
       },
       "startup-fast": {
           Version: ConfigVersionCurrent,
           Profile: "startup-fast",
           Security: SecurityConfig{
               Level: "baseline",
           },
           ClaudeCode: ClaudeCodeConfig{
               Enabled:         boolPtr(true),
               PermissionLevel: "standard",
               MCPServers:      []string{"context7", "github"},
           },
           Tools: ToolsConfig{
               Enabled: []string{"gitleaks"},
           },
       },
       "enterprise": {
           Version: ConfigVersionCurrent,
           Profile: "enterprise",
           Security: SecurityConfig{
               Level: "strict",
           },
           ClaudeCode: ClaudeCodeConfig{
               Enabled:         boolPtr(true),
               PermissionLevel: "restricted",
               Skills:          []string{"security-review", "agent-postmortem", "differential-review"},
               MCPServers:      []string{"context7", "github"},
           },
           Tools: ToolsConfig{
               Enabled: []string{"semgrep", "gitleaks", "secretspec", "container-security", "license-compliance"},
           },
       },
   }
   ```
3. Implement profile inheritance in the resolution engine:
   ```go
   // resolveWithProfiles applies the full inheritance chain:
   // org defaults -> built-in profile -> client overlay -> project overrides -> local overrides
   func resolveWithProfiles(
       orgDefaults *GdevConfig,
       projectCfg *GdevConfig,
       localCfg *LocalConfig,
       verbose bool,
   ) (*ResolvedConfig, error) {
       // Layer 1: Org defaults
       result := deepCopy(orgDefaults)

       // Layer 2: Built-in profile (if specified)
       if projectCfg.Profile != "" {
           profile, ok := BuiltInProfiles[projectCfg.Profile]
           if !ok {
               return nil, fmt.Errorf("unknown profile: %s", projectCfg.Profile)
           }
           result = deepMerge(result, &profile)
       }

       // Layer 3: Client compliance overlay (if present)
       if projectCfg.Client != nil && projectCfg.Client.SecurityLevel != "" {
           complianceCfg := complianceLevelToConfig(projectCfg.Client.SecurityLevel)
           result = deepMerge(result, complianceCfg)
       }

       // Layer 4: Project overrides
       result = deepMerge(result, projectCfg)

       // Layer 5: Local overrides
       if localCfg != nil {
           result = deepMergeLocal(result, localCfg)
       }

       // Enforce security floor
       enforceSecurityFloor(result, projectCfg)

       return &ResolvedConfig{Config: result}, nil
   }
   ```
4. Implement `complianceLevelToConfig` that maps a compliance level to concrete config values:
   ```go
   func complianceLevelToConfig(level string) *GdevConfig {
       cp, ok := ComplianceLevels[level]
       if !ok {
           return nil
       }
       return &GdevConfig{
           Security: SecurityConfig{
               Level:          cp.Level,
               AgeGating:      boolPtr(true),
               ScriptBlocking: boolPtr(cp.ScriptBlocking),
               VulnScanning:   boolPtr(cp.VulnScanning),
               LockEnforce:    boolPtr(true),
           },
           ClaudeCode: ClaudeCodeConfig{
               PermissionLevel: cp.ClaudePermLevel,
           },
           Tools: ToolsConfig{
               Enabled: cp.RequiredPreCommit,
           },
       }
   }
   ```
5. Implement client-specific infrastructure override:
   - `client.registry_proxy` overrides `infrastructure.registry_proxy`.
   - `client.nix_cache` overrides `infrastructure.nix_cache`.
   - `client.allowed_mcp_servers` restricts MCP servers to only those listed.
   - `client.blocked_mcp_servers` with `["*"]` blocks all MCP servers except explicitly allowed (deny-by-default for high-security clients).
6. Wire compliance levels into `qsdev check` (Unit 13.6):
   - Security hardening checks use the resolved compliance level's requirements.
   - Check that age-gating threshold meets the level's minimum.
   - Check that required pre-commit hooks are present per level.
   - Check that SBOM generation is configured per level's policy.
   - Check Claude Code permission level meets the level's requirement.
7. Implement `qsdev init --list-profiles` to show available profiles:
   ```
   $ qsdev init --list-profiles
   Built-in Profiles:

     consulting-default   Enhanced security for typical client engagements (SOC2-ready)
     startup-fast         Baseline security for speed-focused POCs and internal projects
     enterprise           Strict security for HIPAA, FedRAMP, or high-security clients
     go-web               Go web service with PostgreSQL and Redis
     ts-fullstack         TypeScript full-stack with pnpm and PostgreSQL
     python-data          Python data science with uv
     rust-cli             Rust CLI application
   ```
8. Write comprehensive tests:
   - `consulting-default` profile resolves to enhanced security settings.
   - `enterprise` profile resolves to strict security with all tools enabled.
   - `startup-fast` profile resolves to baseline with minimal tools.
   - Client `security_level: strict` overrides profile's `security.level: enhanced`.
   - Client `blocked_mcp_servers: ["*"]` blocks all servers except allowed.
   - Security floor prevents local override of client compliance level.
   - `qsdev check` validates compliance level requirements correctly.
   - Profile inheritance: org -> profile -> client -> project -> local all layer correctly.
   - Unknown profile name produces clear error.
   - Compliance level mapping generates correct concrete values.

**Acceptance Criteria:**
- [ ] Three compliance levels defined: baseline (minimum viable), enhanced (SOC2-ready), strict (HIPAA/FedRAMP)
- [ ] Each compliance level maps to concrete configuration values (age-gating threshold, required hooks, SBOM policy, etc.)
- [ ] Built-in profiles: `consulting-default` (enhanced), `startup-fast` (baseline), `enterprise` (strict)
- [ ] Profile inheritance chain: org defaults -> built-in profile -> client overlay -> project overrides -> local overrides
- [ ] Client `security_level` acts as a floor that cannot be lowered by project or local overrides
- [ ] Client `blocked_mcp_servers: ["*"]` enforces deny-by-default MCP policy
- [ ] Client infrastructure overrides (registry_proxy, nix_cache) propagate through resolution
- [ ] `qsdev check` validates compliance level requirements (age-gating threshold, required hooks, SBOM policy)
- [ ] `qsdev init --list-profiles` shows all available profiles with descriptions
- [ ] Unknown profile name produces a clear error with list of available profiles
- [ ] Compliance level concrete values match the mapping table from consulting lifecycle research

**Research Citations:**
- `research-spikes/gdev-team-config-onboarding/consulting-lifecycle-research.md` -- client profiles, compliance level mapping table, security floor enforcement, MCP deny-by-default
- `research-spikes/gdev-team-config-onboarding/team-config-sharing-research.md` -- profile inheritance, resolution order
- `research-spikes/gdev-team-config-onboarding/standards-enforcement-ci-research.md` -- OrgPolicy struct, required tools enforcement
- `phases/06-wizard-orchestration.md` -- profile system integration, `--list-profiles` flag

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Existing Types to Extend

| Type | Location | Change Needed |
|------|----------|---------------|
| `WizardAnswers` | `pkg/types/types.go:11-32` | Add `ConfigVersion string` field for schema version tracking |
| `Profile` | `addons/devinit/config.go:11-24` | Add `ComplianceLevel string` field |
| `InfraProfile` | `internal/profile/types.go:94-210` | Add `ComplianceLevel string` field |

### Functions to Reuse

| Function | Location | Reuse Context |
|----------|----------|---------------|
| `MergeProfileWithFlags()` | `addons/devinit/profile_convert.go:72-121` | Model for `.qsdev.yaml` -> `WizardAnswers` merging (languages replace, services append) |
| `DetectExistingConfig()` | `addons/devinit/merge_mode.go:19-55` | Extend with mode routing (currently only blocks; Phase 13 adds Create/Join/Update/Repair dispatch) |
| `detect.Detect()` | `internal/detect/detect.go:12-83` | Provides filesystem signals (HasDevenvNix, HasClaudeDir, etc.) for `DetectOnboardingMode()` |
| `toolcheck.Detect()` | `internal/toolcheck/toolcheck.go:11-40` | Reuse for "Required Tools" check category in `qsdev check` |

### Internal File Coexistence

- `.devinit/.qsdev-init-answers.yaml` (`addons/devinit/answers.go:24-25`) -- full wizard answers (internal, gitignored)
- `.devinit/.qsdev-init-state.yaml` -- generation state tracking (internal, gitignored)
- `.qsdev.yaml` -- public project config (committed to git, Phase 13 introduces this)
- `.qsdev.local.yaml` -- developer overrides (gitignored, Phase 13 introduces this)

### New Dependencies

- `github.com/Masterminds/semver/v3` -- required for `gdev_version` constraint parsing (not currently in go.mod)

### Patterns to Establish

- `--format text|json|sarif|junit` flag pattern on `qsdev check` using `cmd.OutOrStdout()` -- first structured-output command; all future commands (`qsdev status`, `qsdev devenv doctor`) should follow this pattern

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] `.qsdev.yaml` round-trip: write config -> parse -> resolve -> generate -> `qsdev check` passes
- [ ] Three-layer resolution produces correct output for all layer combinations (org-only, org+project, org+project+local)
- [ ] Security floor enforcement verified: local overrides cannot weaken project security level
- [ ] Four onboarding modes correctly detected and routed: Create (no config) -> wizard, Join (config exists, no state) -> local setup, Update (version mismatch) -> Phase 8, Repair (drift) -> fix suggestions
- [ ] Join mode: `git clone` + `cd` + `qsdev init` + `devenv shell` completes in under 2 minutes
- [ ] `qsdev check` passes on a well-configured project and fails on a misconfigured one
- [ ] `qsdev check --format sarif` produces valid SARIF 2.1.0 uploadable to GitHub Security tab
- [ ] `qsdev check --format junit` produces valid JUnit XML parseable by CI systems
- [ ] `qsdev check --auto-fix` fixes safe additive issues without touching explicit user configuration
- [ ] Client compliance levels (baseline/enhanced/strict) map to correct concrete settings
- [ ] Version constraint (`gdev_version`) checked on all gdev commands before any operation
- [ ] Version ratchet prevents older binary from overwriting newer-generated files
- [ ] Config migration chain infrastructure in place (v1 only, but chain is extensible)
- [ ] `qsdev init --list-profiles` shows all built-in and team-configured profiles
