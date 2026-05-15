# Phase 31: Copier Template Integration

## Goal

Integrate Copier as the standard way to scaffold new projects from organizational templates. gdev orchestrates a Copier-first-then-gdev-init flow: Copier creates the project structure, then gdev overlays devenv, Claude Code, and security configuration. Templates are registered in a local registry and support non-interactive use. This makes gdev the enforcement point that ensures every project — regardless of which template created it — ends up with organizational security standards applied.

## Dependencies

Phase 6 complete (wizard orchestration, profile system, huh forms, non-interactive `--yes` mode, `qsdev init` command structure). Phase 13 complete (`.qsdev.yaml` schema, Join mode detection, config resolution engine — Join mode is the post-Copier landing mode when a template ships `.qsdev.yaml`).

## Phase Outputs

- `~/.qsdev/templates.yaml` registry with `qsdev template add/list/remove` commands
- `CopierRunner` Go struct wrapping Copier subprocess invocation
- `qsdev init --from <template>` two-phase orchestration (Copier + gdev)
- `qsdev update --template` pulling latest template version
- Non-interactive template support with `--data answers.yaml`
- Template authoring specification accessible via `qsdev template --help-authoring`

---

### Unit 31.1: Template Registry

**Description:** Implement `~/.qsdev/templates.yaml` as the local template registry and three registry management commands: `qsdev template add`, `qsdev template list`, and `qsdev template remove`. The registry tracks template metadata and supports initial population from an organizational template catalog URL.

**Context:** The template registry solves the discovery problem: developers need to know which templates exist and where they live without consulting a wiki or asking a colleague. The registry is per-developer (lives in `~/.qsdev/`, not in a project), so each consultant accumulates templates across engagements. The design mirrors the way `mise` manages plugins — a flat file maps names to sources, and the tool validates reachability on add. Organization-wide templates are seeded via `qsdev setup --org-templates <url>`, which fetches a remote `templates.yaml` and merges it into the local registry.

Template sources can be git URLs (any URL Copier accepts: `gh:org/repo`, `https://github.com/org/repo`, `git+ssh://...`) or absolute local paths. The registry intentionally does NOT cache template content — it only records names and sources. Copier handles its own caching (in `~/.cache/copier/`). The `last-used-version` field records the git ref that was last successfully copied, enabling `qsdev template list` to show whether templates have pending updates.

**Desired Outcome:** Developers run `qsdev template list` and see their registered templates with descriptions and last-used versions. `qsdev template add qss/go-service gh:quantumserendipity/go-service-template` registers a template in one command and validates the source is reachable before writing to the registry.

**Steps:**

1. Define the registry types in `internal/templates/registry.go`:
   ```go
   // TemplateRegistry is the in-memory representation of ~/.qsdev/templates.yaml.
   type TemplateRegistry struct {
       // SchemaVersion is the registry file format version (integer).
       SchemaVersion int `yaml:"schema_version"`

       // Templates is the ordered list of registered templates.
       Templates []TemplateEntry `yaml:"templates"`
   }

   // TemplateEntry is a single registered template.
   type TemplateEntry struct {
       // Name is the short identifier used in `qsdev init --from <name>`.
       Name string `yaml:"name" validate:"required,kebab-case"`

       // Source is the Copier-compatible source URL or local path.
       // Examples: "gh:org/repo", "https://github.com/org/repo", "/home/user/templates/go-service"
       Source string `yaml:"source" validate:"required"`

       // Description is a one-line human-readable summary.
       Description string `yaml:"description,omitempty"`

       // LastUsedVersion is the git ref (tag, branch, or commit hash) last used
       // when copying or updating from this template. Empty if never used.
       LastUsedVersion string `yaml:"last_used_version,omitempty"`

       // AddedAt is the RFC3339 timestamp when this entry was registered.
       AddedAt string `yaml:"added_at,omitempty"`

       // OrgManaged indicates this entry was seeded by an org template catalog
       // (via gdev setup --org-templates). Org-managed entries are not removed
       // by `qsdev template remove` without --force.
       OrgManaged bool `yaml:"org_managed,omitempty"`
   }
   ```

2. Implement registry file loading and saving in `internal/templates/registry.go`:
   ```go
   // RegistryPath returns the canonical path to the template registry.
   func RegistryPath() string {
       return filepath.Join(os.UserHomeDir(), ".qsdev", "templates.yaml")
   }

   // LoadRegistry reads ~/.qsdev/templates.yaml. Returns an empty registry if the
   // file does not exist (first run).
   func LoadRegistry() (*TemplateRegistry, error)

   // SaveRegistry atomically writes the registry to ~/.qsdev/templates.yaml,
   // creating the directory if it does not exist.
   func SaveRegistry(r *TemplateRegistry) error
   ```
   - Atomic write: write to a temp file in `~/.qsdev/`, then `os.Rename`.
   - Create `~/.qsdev/` directory with `0700` permissions if it does not exist.
   - Preserve yaml comment at top of file: `# gdev template registry — managed by 'gdev template' commands`.

3. Implement `qsdev template add <name> <source> [--description <text>]`:
   - Parse and validate `<name>` as kebab-case (pattern: `[a-z][a-z0-9-]*`).
   - Validate `<source>` is syntactically a git URL or absolute path (does not need to resolve yet).
   - Check that `<name>` is not already registered; fail with: `Template "X" is already registered. Use 'gdev template remove X' first, or choose a different name.`
   - **Reachability check:** run `copier copy --pretend --quiet <source> /tmp/gdev-validate-XXXXXX` (temp dir). If Copier exits non-zero, report the error and abort. This is the only validation step that requires Copier to be installed.
   - If `--skip-validate` is passed: skip the reachability check (for offline environments or local paths under development).
   - On success: append the entry to the registry, print `✓ Template "X" registered (source: Y)`.

4. Implement `qsdev template list`:
   - Read registry and print a table:
     ```
     NAME                   SOURCE                                         LAST USED   ORG
     qss/go-service         gh:quantumserendipity/go-service-template      v1.3.0      ✓
     qss/ts-fullstack        gh:quantumserendipity/ts-fullstack-template    v2.1.1      ✓
     my-custom-template     /home/user/dev/templates/my-template           —
     ```
   - `--json` flag: output JSON array of `TemplateEntry`.
   - If registry is empty, print: `No templates registered. Run 'gdev template add <name> <source>' to register one.`

5. Implement `qsdev template remove <name>`:
   - Look up `<name>` in registry; fail with clear error if not found.
   - If `OrgManaged: true`, require `--force` flag: `Template "X" is org-managed. Use --force to remove it.`
   - On success: remove entry from registry, save, print `✓ Template "X" removed.`
   - Does NOT delete any template source (local path or remote).

6. Implement `qsdev setup --org-templates <url>` integration in `cmd/setup.go`:
   - Fetch the URL (HTTP GET, accept `text/plain` and `application/yaml`).
   - Parse the fetched content as a `TemplateRegistry`.
   - For each entry in the fetched registry: if name not present locally, add it with `OrgManaged: true`; if name already present and `OrgManaged: true`, update the source (org can rotate template locations); if name already present and `OrgManaged: false` (developer-added), skip with a warning.
   - Print a summary: `Added 3 org templates, updated 1, skipped 2 (already registered by user).`

7. Write unit tests:
   - Add entry, list entries, remove entry.
   - Duplicate name rejected.
   - Org-managed removal requires `--force`.
   - `--org-templates` merge: new entries added, org-managed entries updated, user entries skipped.
   - Empty registry produces friendly message on `list`.
   - Registry file created with correct permissions on first write.

**Acceptance Criteria:**
- [ ] `~/.qsdev/templates.yaml` registry with `TemplateEntry` schema (name, source, description, last-used-version, added-at, org-managed)
- [ ] `qsdev template add <name> <source>` registers a template and validates reachability via Copier `--pretend`
- [ ] Duplicate name produces a clear error with remediation hint
- [ ] `qsdev template list` shows registered templates in a table with `--json` output option
- [ ] `qsdev template remove <name>` removes entry without touching template source
- [ ] Org-managed entries require `--force` to remove
- [ ] `qsdev setup --org-templates <url>` seeds registry from remote URL, merging without overwriting user entries
- [ ] Registry file written atomically; `~/.qsdev/` created with `0700` permissions if absent
- [ ] `--skip-validate` skips reachability check for offline use

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — registry design, Copier source URL formats, org template catalog pattern

**Status:** Not Started

---

### Unit 31.2: CopierRunner Subprocess Wrapper

**Description:** Implement `CopierRunner`, a Go struct that wraps Copier subprocess invocation. The runner detects whether Copier is installed, provides actionable install instructions if missing, and exposes typed methods for `copy`, `update`, and version-pinned invocations. All Copier output is captured and surfaced through structured error types.

**Context:** Copier is a Python tool typically installed via `pipx` or `uv tool`. gdev cannot vendor it, so it runs as an external subprocess. The wrapper must handle the full range of failure modes: Copier not installed, Copier version too old (Copier 9.x added `--data` support), template source unreachable, Copier questions not answered (interactive prompt hanging in CI), and Copier exiting non-zero due to template errors. Copier's stdout is user-visible progress output; its stderr is error messages. gdev should forward both to the terminal unless `--quiet` is passed.

Version pinning is a consulting requirement: if a template specifies a minimum Copier version in its `copier.yaml` (`_min_copier_version: "9.0"`), the runner must check the installed version before proceeding. The runner exposes this as a structured pre-flight check rather than discovering the failure mid-copy.

**Desired Outcome:** Calling `runner.Copy(source, dest, opts)` either produces a populated project directory and returns nil, or returns a typed error explaining exactly what went wrong and how to fix it. The runner never hangs waiting for interactive input when called from gdev.

**Steps:**

1. Define the `CopierRunner` struct in `internal/copier/runner.go`:
   ```go
   // CopierRunner wraps Copier subprocess invocations.
   type CopierRunner struct {
       // BinaryPath is the resolved path to the copier binary.
       // Populated by Detect() or set explicitly for testing.
       BinaryPath string

       // Version is the detected Copier version string (e.g. "9.3.1").
       Version string

       // Quiet suppresses forwarding Copier stdout to the terminal.
       Quiet bool

       // WorkDir is the working directory for subprocess invocations.
       // Defaults to the current directory if empty.
       WorkDir string
   }
   ```

2. Implement `Detect() (*CopierRunner, error)`:
   - Search for `copier` binary: `$PATH` lookup using `exec.LookPath`.
   - Also check `~/.local/bin/copier` (pipx default), `~/.local/share/uv/tools/copier/bin/copier` (uv tool default).
   - Run `copier --version` and parse output to extract version string.
   - If not found, return `&CopierNotInstalledError{}` (typed, not string error).
   - If version is below minimum (9.0.0), return `&CopierVersionTooOldError{}`.

3. Define error types in `internal/copier/errors.go`:
   ```go
   // CopierNotInstalledError is returned when Copier is not found on the system.
   type CopierNotInstalledError struct{}

   func (e *CopierNotInstalledError) Error() string {
       return "Copier is not installed.\n\n" +
           "Install options:\n" +
           "  pipx install copier        # recommended (isolated install)\n" +
           "  uv tool install copier     # if you use uv\n" +
           "  nix profile install nixpkgs#copier  # NixOS/nix-env\n\n" +
           "After installing, re-run: qsdev init --from <template>"
   }

   // CopierVersionTooOldError is returned when the installed Copier version
   // is below the minimum required (9.0.0 for --data flag support).
   type CopierVersionTooOldError struct {
       InstalledVersion string
       MinimumVersion   string
   }

   func (e *CopierVersionTooOldError) Error() string {
       return fmt.Sprintf(
           "Copier %s is too old. gdev requires Copier >= %s.\n\n"+
           "Upgrade: pipx upgrade copier",
           e.InstalledVersion, e.MinimumVersion,
       )
   }

   // CopierFailedError wraps a non-zero Copier exit with captured output.
   type CopierFailedError struct {
       Command  string
       ExitCode int
       Stderr   string
   }

   func (e *CopierFailedError) Error() string {
       return fmt.Sprintf(
           "Copier failed (exit %d):\n%s",
           e.ExitCode, e.Stderr,
       )
   }
   ```

4. Implement `CopyOptions` and `runner.Copy()`:
   ```go
   // CopyOptions controls the behavior of a Copier copy invocation.
   type CopyOptions struct {
       // Source is a Copier-compatible template source (git URL or path).
       Source string

       // Dest is the destination directory. Must not exist, or must be empty.
       Dest string

       // Data is answers to pass via --data (YAML-encoded key-value pairs).
       // When non-empty, Copier runs non-interactively.
       Data map[string]any

       // VCSRef pins the template to a git ref (tag, branch, commit hash).
       // Passed as --vcs-ref to Copier.
       VCSRef string

       // Overwrite allows copying into an existing directory.
       Overwrite bool

       // Defaults causes Copier to use default values for all unanswered questions.
       // Passed as --defaults to Copier.
       Defaults bool
   }

   // Copy runs `copier copy` with the given options.
   // Returns nil on success. Returns typed errors for all failure modes.
   func (r *CopierRunner) Copy(opts CopyOptions) error
   ```
   - Construct the `copier copy` command from opts.
   - If `opts.Data` is non-empty: marshal to YAML, write to a temp file, pass `--data <path>` to Copier.
   - If `opts.Defaults`: append `--defaults`.
   - If `opts.VCSRef != ""`: append `--vcs-ref <ref>`.
   - Always append `--overwrite` when `opts.Overwrite` is true.
   - Stream stdout to `os.Stdout` unless `r.Quiet`. Always capture stderr.
   - On non-zero exit: return `&CopierFailedError{...}` with captured stderr.

5. Implement `runner.Update()`:
   ```go
   // UpdateOptions controls the behavior of a Copier update invocation.
   type UpdateOptions struct {
       // ProjectDir is the directory to update (must contain .copier-answers.yml).
       ProjectDir string

       // Data overrides answers for this update cycle.
       Data map[string]any

       // Defaults uses default answers for any new questions.
       Defaults bool

       // Skip skips files that conflict with user modifications.
       Skip bool
   }

   // Update runs `copier update` in the given project directory.
   func (r *CopierRunner) Update(opts UpdateOptions) error
   ```

6. Implement `runner.Version()` helper:
   - Runs `copier --version` and returns parsed `semver.Version`.
   - Used by callers checking minimum version constraints from template `copier.yaml`.

7. Write unit tests (using exec stubbing or a fake `copier` binary in `$PATH`):
   - `Detect()` returns `CopierNotInstalledError` when binary absent.
   - `Detect()` returns `CopierVersionTooOldError` when version < 9.0.0.
   - `Copy()` passes `--data <file>` when `Data` is non-empty.
   - `Copy()` returns `CopierFailedError` on non-zero exit with stderr captured.
   - `Copy()` forwards stdout to terminal when `Quiet: false`.
   - `Update()` passes `--defaults` when `Defaults: true`.

**Acceptance Criteria:**
- [ ] `CopierRunner` detects Copier via `$PATH` and common install locations
- [ ] `CopierNotInstalledError` provides install instructions for pipx, uv, and nix
- [ ] `CopierVersionTooOldError` identifies the installed version and the upgrade command
- [ ] `Copy()` marshals `Data` to a YAML temp file and passes `--data <path>` to Copier
- [ ] `Copy()` returns `CopierFailedError` with captured stderr on non-zero exit
- [ ] `Copy()` streams stdout to terminal (respects `Quiet` flag)
- [ ] `Update()` invokes `copier update` with correct flags
- [ ] All error types implement the `error` interface with actionable messages
- [ ] Runner never hangs waiting for interactive input when `Data` or `Defaults` is set

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — Copier CLI flags, `--data` support in Copier 9.x, install paths, non-interactive patterns

**Status:** Not Started

---

### Unit 31.3: `qsdev init --from <template>` Orchestration

**Description:** Implement the two-phase `qsdev init --from <template>` flow: Phase 1 runs Copier to create the project structure, Phase 2 runs gdev's normal `init` on the result. The handoff between phases uses mode detection from Phase 13 — if the template shipped `.qsdev.yaml`, gdev runs in Join mode; if not, gdev runs full Create mode. Template questions (Copier's domain) and devenv/security questions (gdev's domain) remain separate.

**Context:** The Copier-first flow is the key architectural decision: Copier handles everything about project structure (directory layout, boilerplate files, CI pipelines, language-specific config), while gdev handles everything about development environment and security (devenv.nix, settings.json, CLAUDE.md, pre-commit hooks). This separation of concerns means templates stay lean — they do not need to know about devenv or Claude Code — while gdev ensures security standards are applied regardless of which template was used.

The template can optionally ship a `.qsdev.yaml` that captures project configuration decisions. When it does, gdev's Join mode reads this file and skips most wizard questions (the template author already encoded the right defaults). When it does not, gdev runs full Create mode — ecosystem detection runs on the newly-created project files, and the wizard prompts for the usual questions. This "detect-what-Copier-created" approach ensures gdev works with any template, even those authored before gdev existed.

**Code-Grounded Note:** The existing `runInit()` in `cmd/init.go` dispatches to Create/Join/Update/Repair modes via `DetectOnboardingMode()` from Phase 13. The `--from` flag adds a new pre-phase before mode detection: run Copier in a temp directory, move result to destination, then run the normal mode detection. The Copier phase is transparent to the rest of the init flow.

**Desired Outcome:** Developers run `qsdev init --from qss/go-service my-new-service` and get a complete project directory with both the template's project structure and gdev's security configuration, having answered only the questions relevant to their choices (template questions and any gdev questions not already encoded in `.qsdev.yaml`).

**Steps:**

1. Add `--from <template>` flag to the `qsdev init` command in `cmd/init.go`:
   ```go
   var fromTemplate string
   initCmd.Flags().StringVar(&fromTemplate, "from", "", "Copier template name or source URL to scaffold from")
   ```
   - If `--from` is empty: normal init flow (existing behavior, unchanged).
   - If `--from` is a registered name: look up source in registry, proceed.
   - If `--from` is a URL or path not in the registry: use directly as Copier source (allows one-off use without registering).

2. Implement `runCopierPhase(templateSource, destDir string, opts CopierPhaseOpts) error` in `internal/init/copier_phase.go`:
   ```go
   type CopierPhaseOpts struct {
       // Data provides pre-answers for Copier questions (from --data flag).
       Data map[string]any

       // VCSRef pins the template to a specific git ref.
       VCSRef string

       // Defaults uses Copier defaults for unanswered questions.
       Defaults bool

       // NonInteractive runs Copier with --defaults when true.
       // Set when gdev is invoked with --yes.
       NonInteractive bool
   }

   func runCopierPhase(templateSource, destDir string, opts CopierPhaseOpts) error {
       runner, err := copier.Detect()
       if err != nil {
           return err // CopierNotInstalledError provides install instructions
       }

       fmt.Printf("Scaffolding project from template: %s\n", templateSource)

       return runner.Copy(copier.CopyOptions{
           Source:   templateSource,
           Dest:     destDir,
           Data:     opts.Data,
           VCSRef:   opts.VCSRef,
           Defaults: opts.Defaults || opts.NonInteractive,
       })
   }
   ```

3. Implement the two-phase orchestration in `runInit()`:
   ```go
   func runInit(cmd *cobra.Command, args []string) error {
       // Phase 0: If --from is specified, run Copier first.
       if fromTemplate != "" {
           destDir := projectRoot
           if len(args) > 0 {
               destDir = args[0] // qsdev init --from template <directory>
               if err := os.MkdirAll(destDir, 0755); err != nil {
                   return fmt.Errorf("cannot create destination: %w", err)
               }
               projectRoot = destDir
           }

           source, err := resolveTemplateSource(fromTemplate)
           if err != nil {
               return err
           }

           if err := runCopierPhase(source, destDir, CopierPhaseOpts{
               Data:           parsedDataAnswers, // from --data flag
               NonInteractive: nonInteractive,
           }); err != nil {
               return fmt.Errorf("template scaffolding failed: %w", err)
           }

           fmt.Println("\nTemplate applied. Running qsdev initialization...")
       }

       // Phase 1+: Normal qsdev init (mode detection runs on the now-populated directory).
       result, err := DetectOnboardingMode(projectRoot)
       if err != nil {
           return err
       }
       // ... existing dispatch ...
   }
   ```

4. Implement Join mode detection of template-shipped `.qsdev.yaml`:
   - After Copier phase, `DetectOnboardingMode()` runs on the new directory.
   - If Copier wrote `.qsdev.yaml`: mode is Join. gdev reads it and skips redundant wizard questions.
   - If Copier did NOT write `.qsdev.yaml`: mode is Create. gdev runs full wizard, with ecosystem detection now seeing the template's project files (package.json, go.mod, etc.).
   - Log which mode was chosen and why in the terminal output.

5. Handle the directory argument:
   - `qsdev init --from qss/go-service` — initializes current directory.
   - `qsdev init --from qss/go-service my-new-service` — creates `my-new-service/` and initializes it. After init, prints `cd my-new-service && devenv shell`.
   - If directory already exists and is non-empty: prompt user before overwriting (or fail with `--yes` if non-interactive).

6. Implement `resolveTemplateSource(nameOrURL string) (string, error)`:
   - If `nameOrURL` matches a registry entry name: return `entry.Source`.
   - If `nameOrURL` looks like a URL (`://`, `gh:`, `gl:`, `bb:`): return as-is.
   - If `nameOrURL` starts with `/` or `./`: treat as local path, validate it exists.
   - Otherwise: check registry with fuzzy match; if no match found, error with: `Template "X" not found in registry. Run 'gdev template list' to see available templates, or use a full URL.`

7. Update `~/.qsdev/templates.yaml` with last-used version after a successful Copier copy:
   - Run `git -C <copier-cache-dir> describe --tags HEAD` to get the resolved ref.
   - Update `LastUsedVersion` in the registry entry for this template.

8. Write integration tests:
   - `--from` with registered name resolves source and calls Copier.
   - `--from` with direct URL skips registry lookup.
   - Template with `.qsdev.yaml` → Join mode detected after Copier phase.
   - Template without `.qsdev.yaml` → Create mode with ecosystem detection.
   - `--from` with directory argument creates new directory.
   - Copier failure (non-zero exit) aborts before gdev phase.
   - Non-existing template name produces friendly error with `qsdev template list` suggestion.

**Acceptance Criteria:**
- [ ] `--from <name>` looks up template source in registry; `--from <url>` uses URL directly
- [ ] Copier phase runs before mode detection; qsdev init phase runs on Copier output
- [ ] Template shipping `.qsdev.yaml` triggers Join mode (no redundant wizard questions)
- [ ] Template without `.qsdev.yaml` triggers Create mode with detection on template files
- [ ] `qsdev init --from template my-dir` creates `my-dir/` and initializes it
- [ ] Copier failure produces `CopierFailedError` with stderr and aborts before gdev phase
- [ ] `resolveTemplateSource` handles registered names, URLs, and local paths
- [ ] Registry `LastUsedVersion` updated after successful copy
- [ ] Unknown template name produces error with `qsdev template list` hint

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — two-phase orchestration design, template `.qsdev.yaml` conventions

**Status:** Not Started

---

### Unit 31.4: `qsdev update --template`

**Description:** Implement `qsdev update --template` to pull the latest version of the Copier template that created the current project. If the update changes `.qsdev.yaml`, gdev automatically reconciles by running `qsdev init --update`. If the update touches files tracked by gdev's hash system, users are warned about potential conflicts.

**Context:** Copier stores answers in `.copier-answers.yml` at the project root, which records the template source, the last-used git ref, and all answers given during `copier copy`. The `copier update` command reads this file and re-applies the template at the latest version, merging changes with user modifications. This is Copier's core value proposition for long-lived projects: templates can be updated and changes propagate forward.

The gdev integration adds a reconciliation step: if `copier update` modifies `.qsdev.yaml` (the template author updated their gdev defaults), gdev must re-run its generation pipeline to propagate those changes to devenv.nix, settings.json, and other generated files. If `copier update` modifies files that gdev tracks (e.g., the template directly modifies .pre-commit-config.yaml), gdev's hash-based modification detection will detect the change on the next `qsdev init` or `qsdev check` run and report drift.

**Code-Grounded Note:** The `.copier-answers.yml` file at the project root is Copier's state file. `qsdev update --template` should check for this file's existence before proceeding — it's the signal that the project was created by Copier. Projects not created by Copier get a clear error rather than a confusing Copier error.

**Desired Outcome:** `qsdev update --template` pulls the latest template changes, merges them with local modifications, and keeps gdev's generated configuration consistent with the updated template. The operation is idempotent — running it when already at the latest version is a no-op.

**Steps:**

1. Add `--template` flag to the `qsdev update` command:
   ```go
   var updateTemplate bool
   updateCmd.Flags().BoolVar(&updateTemplate, "template", false, "Pull latest version of the Copier template that created this project")
   ```

2. Implement `runTemplateUpdate(projectRoot string, opts TemplateUpdateOpts) error` in `internal/update/template.go`:
   ```go
   type TemplateUpdateOpts struct {
       Data           map[string]any // additional answer overrides
       Defaults       bool           // use defaults for new questions
       NonInteractive bool           // --yes flag
   }
   ```
   - Check that `.copier-answers.yml` exists in `projectRoot`. If absent: return `"This project was not created with Copier (no .copier-answers.yml found). Cannot run template update."`.
   - Detect Copier (reuse `copier.Detect()`).
   - Read `.copier-answers.yml` to extract current `_src_path` and `_commit` for logging.
   - Snapshot gdev-tracked file hashes before update (use Phase 12 hash registry).
   - Run `runner.Update(UpdateOptions{ProjectDir: projectRoot, Defaults: opts.Defaults || opts.NonInteractive})`.
   - After update: check which gdev-tracked files changed by comparing current hashes to pre-update snapshot.
   - Read updated `.copier-answers.yml` to log the new `_commit`.

3. Implement post-update reconciliation:
   ```go
   func reconcileAfterTemplateUpdate(projectRoot string, changedFiles []string) error {
       // Check if .qsdev.yaml changed.
       gdevYamlChanged := contains(changedFiles, ".qsdev.yaml")
       if gdevYamlChanged {
           fmt.Println("Template updated .qsdev.yaml. Re-running qsdev configuration...")
           return runUpdateMode(projectRoot)  // Phase 8 update flow
       }

       // Warn about gdev-tracked files touched by Copier.
       gdevTrackedChanged := filterGdevTracked(changedFiles)
       if len(gdevTrackedChanged) > 0 {
           fmt.Printf("Warning: Template update modified %d file(s) tracked by gdev:\n", len(gdevTrackedChanged))
           for _, f := range gdevTrackedChanged {
               fmt.Printf("  %s\n", f)
           }
           fmt.Println("Run 'qsdev check' to verify configuration consistency.")
       }

       return nil
   }
   ```

4. Handle conflicts between Copier updates and user modifications:
   - Copier's default behavior on conflict: prompt interactively.
   - With `--yes` (`NonInteractive: true`): pass `--defaults` to Copier, which uses the template's answer as the resolved value.
   - After update: if Copier reports conflicts in its output (detectable by parsing stderr for "conflict"), print a summary and suggest `git diff` for review.

5. Update registry `LastUsedVersion` after successful update (same as in Unit 31.3 step 7).

6. Implement `--template --dry-run`:
   - Run `copier update --pretend` which shows what would change without writing.
   - Print the list of files that would be added/modified/deleted.
   - Do not run reconciliation (nothing changed).

7. Wire into the `qsdev update` command:
   - `qsdev update --template`: template update only.
   - `qsdev update`: existing gdev-config update (Phase 8), unchanged behavior.
   - `qsdev update --template --yes`: non-interactive, uses defaults for new Copier questions.

8. Write integration tests:
   - Project without `.copier-answers.yml` produces clear error.
   - Template update that does NOT change `.qsdev.yaml` → warns about tracked files if any changed.
   - Template update that changes `.qsdev.yaml` → triggers gdev Update mode.
   - `--dry-run` shows what would change without writing.
   - `--yes` runs with `--defaults` for Copier questions.

**Acceptance Criteria:**
- [ ] `qsdev update --template` checks for `.copier-answers.yml` before proceeding
- [ ] Absence of `.copier-answers.yml` produces a clear error (not a Copier error)
- [ ] Copier update output (file additions/modifications/deletions) displayed to user
- [ ] If template update changes `.qsdev.yaml`, gdev Update mode runs automatically
- [ ] If template update touches gdev-tracked files, user is warned with `qsdev check` suggestion
- [ ] `--dry-run` shows changes without writing
- [ ] `--yes` passes `--defaults` to Copier for non-interactive CI use
- [ ] Registry `LastUsedVersion` updated after successful update
- [ ] `qsdev update` (without `--template`) is unchanged

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — `copier update` behavior, `.copier-answers.yml` format, conflict resolution

**Status:** Not Started

---

### Unit 31.5: Non-Interactive Template Support

**Description:** Implement `qsdev init --from <template> --data answers.yaml --yes` for fully non-interactive project creation from CI pipelines or scripted onboarding. The `--data` file supports both Copier answers and gdev answers in separate sections. The command validates that all required answers are present before starting (fail-fast, not mid-wizard).

**Context:** The primary use case is CI-driven project setup: an organization's "new project" pipeline clones a repository, runs `qsdev init --from qss/go-service --data ./project-answers.yaml --yes`, and gets a fully configured project committed. The `--data` file encodes decisions that would otherwise require interactive answers from both Copier (project name, description, Go version) and gdev (profile, services, compliance level).

Two sources of required answers exist: Copier's `copier.yaml` defines its questions (parseable from the template source before copying), and gdev's wizard defines its questions. The `--data` file uses a two-section structure to keep them separate, avoiding namespace collisions (both Copier and gdev might have a `project_name` question, but they mean different things).

**Desired Outcome:** `qsdev init --from template --data answers.yaml --yes` exits 0 with a fully configured project directory, having asked zero interactive questions. It exits non-zero before any writes if any required answer is missing, with a clear list of what's needed.

**Steps:**

1. Define the `--data` file schema:
   ```yaml
   # Project answers file for qsdev init --from <template> --data <this-file>
   # Split into two sections: copier (Copier template answers) and gdev (gdev wizard answers).

   copier:
     project_name: my-service
     description: "A new Go microservice"
     go_version: "1.23"
     author: "Jane Smith"

   gdev:
     profile: consulting-default
     services:
       - postgres
       - redis
     compliance_level: enhanced
   ```

2. Implement `ParseDataFile(path string) (*DataAnswers, error)` in `internal/init/data_file.go`:
   ```go
   type DataAnswers struct {
       Copier map[string]any `yaml:"copier"`
       Gdev   GdevAnswerOverrides `yaml:"gdev"`
   }

   type GdevAnswerOverrides struct {
       Profile         string   `yaml:"profile,omitempty"`
       Services        []string `yaml:"services,omitempty"`
       Languages       []string `yaml:"languages,omitempty"`
       ComplianceLevel string   `yaml:"compliance_level,omitempty"`
       // Add fields for any wizard question that can be pre-answered.
   }
   ```
   - Returns structured error if YAML is malformed.
   - Missing sections (`copier:` or `gdev:`) are allowed — empty map/struct used.

3. Implement pre-flight validation:
   ```go
   // ValidateDataFile checks that all required answers are present before starting.
   // requiredCopierKeys comes from parsing the template's copier.yaml.
   // Returns a list of missing keys with descriptions.
   func ValidateDataFile(answers *DataAnswers, requiredCopierKeys []CopierQuestion) []MissingAnswer
   ```
   - Fetch and parse `copier.yaml` from the template source to discover required questions.
   - For each Copier question with no default and not in `answers.Copier`: add to missing list.
   - For gdev answers: validate `profile` name exists in registry if provided; validate `compliance_level` is one of baseline/enhanced/strict.
   - If missing list is non-empty: print table of missing answers and exit 1 before any writes.

4. Integrate data file into the init flow:
   - Pass `answers.Copier` to `CopierRunner.Copy()` via `CopyOptions.Data`.
   - Pass `answers.Gdev` to the gdev wizard to seed answers (so wizard skips pre-answered questions).
   - With `--yes`: combine `--data` seeding with Copier `--defaults` (use Copier defaults for any remaining unanswered questions).

5. Implement `fetchTemplateCopierYaml(source string) ([]CopierQuestion, error)`:
   - Run `copier copy --pretend --quiet <source> /tmp/XXXXXX` to a temp dir.
   - Read the `copier.yaml` from the temp copy.
   - Parse `CopierQuestion` entries (question name, whether it has a default, its type).
   - Clean up temp dir.
   - Cache result for the session to avoid multiple Copier invocations.

6. Implement a `--validate-data` flag:
   - `qsdev init --from template --data answers.yaml --validate-data`: runs pre-flight validation only, exits 0 if all required answers are present, exits 1 with missing answer list if not.
   - Useful in CI to check a data file before committing it to a pipeline.

7. Write unit tests:
   - Valid data file with all required answers → no missing answers.
   - Data file missing a required Copier key → missing answer reported.
   - Data file with invalid profile name → validation error before any Copier invocation.
   - Empty `copier:` section → valid (uses Copier defaults).
   - `--validate-data` exits 0 on valid file, 1 on missing answers.
   - Both Copier and gdev answers correctly separated when passed to their respective consumers.

**Acceptance Criteria:**
- [ ] `--data answers.yaml` file supports separate `copier:` and `gdev:` sections
- [ ] Pre-flight validation runs before any file writes: missing required Copier answers reported upfront
- [ ] Missing answer report is a table with question name and description
- [ ] `gdev:` section answers seed the gdev wizard (skips pre-answered questions)
- [ ] `copier:` section answers passed as `--data` to Copier invocation
- [ ] `--yes` combines `--data` seeding with Copier `--defaults` for remaining questions
- [ ] `--validate-data` flag validates data file without writing anything (exit 0/1)
- [ ] Invalid profile name in `gdev:` section detected during validation
- [ ] Entire flow (validation + Copier + gdev) is non-interactive when `--data` + `--yes` provided

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — non-interactive patterns, `--data` flag, CI use cases

**Status:** Not Started

---

### Unit 31.6: Template Authoring Specification

**Description:** Implement `qsdev template --help-authoring` that prints the template authoring guide, and implement template structure validation on `qsdev template add`. The guide documents conventions for creating Copier templates that work with gdev, including where to put `.qsdev.yaml`, how to declare ecosystem dependencies, and how to include `.claude/` directory contents.

**Context:** Template authors need a clear contract: if they follow these conventions, their template will integrate cleanly with gdev. The conventions are intentionally minimal — most templates need no gdev-specific content at all, and gdev will work regardless (by running Create mode on the template output). The guide exists to help authors who *want* to provide a better-integrated experience by pre-encoding qsdev configuration in the template.

Template validation on `qsdev template add` catches common authoring mistakes early rather than at `qsdev init` time. The validator checks structural issues (not content issues — those require running the template, which validation intentionally avoids).

**Desired Outcome:** A template author reads `qsdev template --help-authoring`, follows the conventions, and their template produces a project that, after `qsdev init --from template`, has the full gdev security stack applied and the user answered only template-specific questions (not re-asked about Go version, profile, etc.).

**Steps:**

1. Implement the authoring guide as an embedded string in `internal/templates/authoring.go`:

   The guide should cover these sections:
   - **Overview:** how the two-phase flow works; what gdev does vs what the template should do.
   - **Minimal template (no gdev integration):** any Copier template works; gdev runs Create mode on the output.
   - **Enhanced integration:** ship `.qsdev.yaml` in the template to pre-encode qsdev configuration; gdev runs Join mode and skips redundant questions.
   - **`.qsdev.yaml` in templates:** the file is templated by Copier (can use Copier variables like `{{project_name}}`); example for a Go service template.
   - **Ecosystem hints:** how to declare languages and services so gdev skips detection.
   - **`.claude/` directory:** templates can ship `.claude/skills/`, `.claude/rules/`, `.claude/agents/` content; gdev merges (does not overwrite) its own generated content.
   - **Example template structure:** a complete directory tree for a well-integrated template.
   - **Validation checklist:** what `qsdev template add` checks.

2. Implement `qsdev template --help-authoring` by adding a `--help-authoring` flag to `qsdev template`:
   ```go
   var helpAuthoring bool
   templateCmd.Flags().BoolVar(&helpAuthoring, "help-authoring", false, "Show guide for creating Copier templates that integrate with gdev")

   if helpAuthoring {
       fmt.Print(templates.AuthoringGuide)
       return nil
   }
   ```

3. Implement template structure validation in `internal/templates/validate.go`:
   ```go
   type TemplateValidationWarning struct {
       Code    string // e.g., "NO_COPIER_YAML", "GDEV_YAML_IN_WRONG_LOCATION"
       Message string
       Hint    string // how to fix
   }

   // ValidateTemplateStructure inspects a locally-cloned template directory
   // and returns warnings about common structural issues.
   // All findings are warnings, not errors — the template may still work.
   func ValidateTemplateStructure(templateDir string) []TemplateValidationWarning
   ```
   - Check for `copier.yaml` or `copier.yml` at root. If absent: warn `NO_COPIER_YAML` — "Template has no copier.yaml. Is this a Copier template?".
   - Check for `.qsdev.yaml` at root (or inside a template subdirectory if using a `_template/` layout). If present: validate it parses without error.
   - Check for `.qsdev.yaml` inside `{{project_name}}/` or similar variable directory — this is the expected location for templates that use subdirectory layouts.
   - Check `.claude/` directory: if present, verify no files that gdev manages (settings.json, CLAUDE.md root file) would be overwritten. Warn if `settings.json` is in the template (gdev generates its own).

4. Wire validation into `qsdev template add`:
   - After reachability check: clone the template to a temp directory (using `copier copy --pretend`).
   - Run `ValidateTemplateStructure()` on the cloned directory.
   - Print warnings (never errors — template is still registered):
     ```
     Template registered with warnings:
       ⚠ NO_COPIER_YAML: Template has no copier.yaml. Is this a Copier template?
         Hint: Add copier.yaml at the template root to define questions and settings.
     ```
   - `--skip-validate` suppresses validation.

5. Add `qsdev template validate <name>` subcommand:
   - Looks up template source, clones to temp dir, runs `ValidateTemplateStructure()`, prints results.
   - Useful for template authors to check their template before publishing.
   - `--json` output: array of `TemplateValidationWarning`.

6. Write unit tests:
   - Valid template (has copier.yaml, no conflicting files) → no warnings.
   - Missing copier.yaml → `NO_COPIER_YAML` warning.
   - Template with settings.json → warning about gdev-managed file conflict.
   - `.qsdev.yaml` that does not parse → warning with parse error included.
   - `qsdev template validate` exits 0 even with warnings (informational only).

**Acceptance Criteria:**
- [ ] `qsdev template --help-authoring` prints the authoring guide covering overview, minimal integration, enhanced integration, `.qsdev.yaml` conventions, `.claude/` directory conventions, and example structure
- [ ] `qsdev template add` runs `ValidateTemplateStructure()` after reachability check and prints warnings
- [ ] Warnings are non-blocking (template is registered regardless)
- [ ] `--skip-validate` suppresses all validation
- [ ] `qsdev template validate <name>` subcommand available for template authors
- [ ] `ValidateTemplateStructure` detects: missing copier.yaml, conflicting settings.json, unparseable .qsdev.yaml
- [ ] All validation findings are warnings (never errors) — a template may still work even if it violates conventions
- [ ] `--json` output on `qsdev template validate` for tooling integration

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-research.md` — template authoring conventions, `.qsdev.yaml` in Copier templates, `.claude/` directory handling

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### New Commands

| Command | Parent | Notes |
|---------|--------|-------|
| `qsdev template` | root | New command group |
| `qsdev template add` | template | |
| `qsdev template list` | template | |
| `qsdev template remove` | template | |
| `qsdev template validate` | template | |
| `qsdev template --help-authoring` | template | Flag on template root |

### Flags Added to Existing Commands

| Command | Flag | Notes |
|---------|------|-------|
| `qsdev init` | `--from <template>` | Triggers Copier phase before mode detection |
| `qsdev init` | `--data <file>` | Answers file with `copier:` and `gdev:` sections |
| `qsdev update` | `--template` | Runs `copier update` |
| `qsdev update` | `--template --dry-run` | `copier update --pretend` |
| `qsdev setup` | `--org-templates <url>` | Seeds template registry from remote URL |

### New Packages

| Package | Path | Purpose |
|---------|------|---------|
| `copier` | `internal/copier/` | CopierRunner, error types, version detection |
| `templates` | `internal/templates/` | Registry types, registry I/O, validation, authoring guide |

### No New Dependencies Required

- Copier subprocess invocation uses `os/exec` (stdlib).
- Registry I/O uses `gopkg.in/yaml.v3` (already in go.mod).
- Version comparison uses `github.com/Masterminds/semver/v3` (added in Phase 13).

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] `qsdev template add/list/remove` manage `~/.qsdev/templates.yaml`
- [ ] `qsdev init --from <template>` runs Copier then gdev in sequence
- [ ] Template with `.qsdev.yaml` → Join mode (no redundant wizard questions)
- [ ] Template without `.qsdev.yaml` → Create mode with detection on template output
- [ ] `qsdev update --template` pulls latest version and reconciles `.qsdev.yaml` changes
- [ ] `qsdev init --from template --data answers.yaml --yes` is fully non-interactive
- [ ] Pre-flight validation fails fast on missing required answers before any writes
- [ ] `qsdev template --help-authoring` prints actionable authoring guide
- [ ] Template validation warnings displayed on `qsdev template add`
- [ ] `qsdev setup --org-templates <url>` seeds registry without overwriting user entries
