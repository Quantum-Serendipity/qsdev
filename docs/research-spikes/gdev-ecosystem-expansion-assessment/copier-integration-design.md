# Copier Template Integration — Implementation Units for Phase 6

These units amend Phase 6 (Wizard & Orchestration) of the qsdev implementation plan. They add Copier-based project scaffolding support to `qsdev init`, enabling firm-wide project templates with lifecycle update capability.

**Rationale:** Code scaffolding was originally rejected (Feature 5) because every ecosystem has its own scaffolding tool and maintaining 27+ templates would drift. Copier was reconsidered because it uniquely supports `copier update` — when the firm's base template evolves (CI pipelines, security configs, compliance templates), existing projects can be brought forward. This is the consulting-critical lifecycle that no ecosystem-specific scaffolding tool provides.

**Research basis:** `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5 Reconsideration`

---

### Unit 5.7: Template Registry & Resolution

**Description:** Implement the template registry that maps short names to git repository URLs, with resolution logic for `--from` arguments that may be short names, full URLs, or local paths.

**Context:** Templates are git repos containing a `copier.yaml` questionnaire. Engineers should not need to remember full URLs. The registry lives in `~/.qsdev/templates.yaml` (user-level) and can also be supplied by organization config (`.qsdev.yaml` in a shared config repo, propagated via the three-layer config from Phase 13). Resolution order: exact URL/path passthrough → user registry lookup → org registry lookup → error.

**Desired Outcome:** `qsdev init --from ts-api` resolves `ts-api` to a full git URL. `qsdev init --from https://github.com/highspring/template-ts-api.git` passes through unchanged. `qsdev init --from ./local-template` resolves to an absolute path.

**Steps:**
1. Define `TemplateRegistry` struct with `Resolve(nameOrURL string) (string, error)`, `List() []TemplateSummary`, `Add(name, url string) error`, `Remove(name string) error`.
2. Implement YAML-based registry file at `~/.qsdev/templates.yaml`:
   ```yaml
   templates:
     ts-api: https://github.com/highspring/template-ts-api.git
     go-service: https://github.com/highspring/template-go-service.git
     python-ml: https://github.com/highspring/template-python-ml.git
   ```
3. Implement resolution logic: if input contains `://` or starts with `.` or `/`, treat as URL/path; otherwise look up in registry.
4. Support org-level registry via `.qsdev.yaml` `templates:` key (merged with user registry, user overrides org for name collisions).
5. Implement `qsdev template list` command showing name, URL, and source (user/org) for each registered template.
6. Implement `qsdev template add <name> <url>` to register a template in the user registry.
7. Implement `qsdev template remove <name>` to unregister a template from the user registry.
8. Validate that URL points to a git-cloneable location (deferred — actual validation happens at `copier copy` time, but basic URL format checking here).

**Acceptance Criteria:**
- [ ] Short name resolves to URL from user registry
- [ ] Short name resolves to URL from org registry when not in user registry
- [ ] User registry overrides org registry for same name
- [ ] Full URL passes through without registry lookup
- [ ] Local path (relative and absolute) resolves correctly
- [ ] Unrecognized short name returns clear error listing available templates
- [ ] `qsdev template list` shows all templates with source labels
- [ ] `qsdev template add/remove` modifies `~/.qsdev/templates.yaml`
- [ ] Registry file is created on first `qsdev template add` if absent

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — Copier integration rationale, template lifecycle
- `research-spikes/gdev-team-config-onboarding/research.md § Three-Layer Config` — org config propagation pattern

**Status:** Not Started

---

### Unit 5.8: Copier Availability Gate & Invocation

**Description:** Implement the Copier tool detection, availability gating, and subprocess invocation layer that wraps `copier copy` and `copier update` commands.

**Context:** Copier is a Python tool available in nixpkgs as `copier`. gdev does not bundle Python — it invokes Copier as an external subprocess. If Copier is not available, gdev must detect this early and provide actionable installation guidance. The invocation layer handles argument construction, output capture, and error translation for both `copier copy` (new project) and `copier update` (existing project).

**Desired Outcome:** A `CopierRunner` that reliably detects Copier availability, invokes it with correct arguments, and translates its exit codes and output into gdev-native errors.

**Steps:**
1. Implement `CopierRunner` with `Available() (version string, error)` — runs `copier --version`, parses output.
2. When Copier is unavailable, return an error with installation instructions:
   - NixOS/devenv: `nix profile install nixpkgs#copier` or add `copier` to devenv packages
   - Generic: `pip install copier` or `pipx install copier`
3. Implement `Copy(templateURL, destDir string, opts CopyOpts) error`:
   - Constructs `copier copy <url> <dest>` with flags
   - `--defaults` flag when non-interactive (`--yes`)
   - `--data key=value` for pre-answered questions from qsdev config
   - `--vcs-ref <tag>` when a specific template version is requested
   - Captures stdout/stderr, translates exit codes to typed errors
4. Implement `Update(destDir string, opts UpdateOpts) error`:
   - Constructs `copier update` in the project directory
   - `--defaults` flag when non-interactive
   - `--conflict rej` to leave `.rej` files for manual resolution (same pattern as git merge conflicts)
   - Captures output for reporting merge conflicts to the user
5. Detect `.copier-answers.yml` presence as indicator that a project was Copier-templated.
6. All subprocess invocations use `exec.CommandContext` with timeout (default 120s, configurable).

**Acceptance Criteria:**
- [ ] `CopierRunner.Available()` returns version when Copier is installed
- [ ] `CopierRunner.Available()` returns actionable error when Copier is missing
- [ ] `Copy()` invokes `copier copy` with correct argument construction
- [ ] `Copy()` passes `--defaults` in non-interactive mode
- [ ] `Copy()` supports `--data` for pre-populated answers
- [ ] `Update()` invokes `copier update` in the project directory
- [ ] `Update()` reports merge conflicts from `.rej` files
- [ ] Subprocess timeout prevents hangs on unresponsive template repos
- [ ] Exit code translation produces typed errors (template not found, git clone failed, questionnaire aborted, merge conflicts)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — Copier as external tool, not reimplemented
- `implementation-plans/qsdev/plan.md § Design Principles` — "Curate, don't reinvent"

**Status:** Not Started

---

### Unit 5.9: `qsdev init --from` Orchestration Flow

**Description:** Wire the `--from` flag into `qsdev init`, orchestrating the Copier-first-then-gdev-init pipeline: resolve template, run Copier questionnaire, then run normal qsdev init on the generated project.

**Context:** When `--from` is specified, qsdev init gains a pre-phase: template scaffolding. The flow is (1) resolve template name, (2) gate on Copier availability, (3) run `copier copy` to generate project files, (4) run the normal qsdev init pipeline (detection, wizard, generation) on the now-populated directory. The Copier questionnaire and gdev wizard are sequential — Copier asks template-specific questions first, then gdev asks environment configuration questions. The generated project may include a `.qsdev.yaml` with org defaults, which the gdev wizard consumes as pre-populated answers.

**Desired Outcome:** `qsdev init --from ts-api` produces a project with both Copier-templated files (README, CI workflows, security policies, CLAUDE.md) AND gdev-generated config (devenv.nix, devenv.yaml, settings.json).

**Steps:**
1. Add `--from <template>` flag to `qsdev init` command (Unit 5.1 extended).
2. When `--from` is set, insert pre-phase into init orchestration:
   a. Resolve template via `TemplateRegistry.Resolve()` (Unit 5.7).
   b. Check Copier availability via `CopierRunner.Available()` (Unit 5.8).
   c. Run `CopierRunner.Copy()` into the target directory.
   d. If the template generated a `.qsdev.yaml`, load it as project config (feeds into wizard defaults).
   e. If the template generated a `CLAUDE.md`, flag it for merge-mode in the claudecode addon (section markers appended, not overwritten).
3. After Copier phase completes, run normal init flow: detect → wizard → generate → write → report.
4. Detection engine (Unit 5.3) now detects languages/services from Copier-generated files (package.json, go.mod, Dockerfile, etc.).
5. Post-generation summary distinguishes Copier-generated files from gdev-generated files.
6. Handle the edge case where `--from` is used in a non-empty directory: warn user, require `--force` or confirmation.

**Acceptance Criteria:**
- [ ] `qsdev init --from ts-api` resolves template and runs Copier
- [ ] Copier questionnaire runs interactively (or with `--defaults` when `--yes`)
- [ ] After Copier, gdev detection finds languages from generated files
- [ ] gdev wizard pre-populates from Copier-generated `.qsdev.yaml` if present
- [ ] Copier-generated `CLAUDE.md` is preserved and extended (not overwritten)
- [ ] Post-generation summary lists both Copier and gdev files
- [ ] `--from` in non-empty directory warns and requires confirmation
- [ ] `qsdev init --from ts-api --yes` runs both Copier and gdev non-interactively
- [ ] `qsdev init` without `--from` is unchanged (no regression)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — integration flow: `copier copy` → detect → wizard → generate
- `implementation-plans/qsdev/phases/06-wizard-orchestration.md § Unit 5.1` — existing init orchestration flow
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Detection Engine` — detection on generated files

**Status:** Not Started

---

### Unit 5.10: `qsdev update --template` Flow

**Description:** Implement the `qsdev update --template` command that runs `copier update` to pull latest template changes into an existing Copier-templated project, then re-runs qsdev init in update mode to reconcile gdev-generated files.

**Context:** This is the critical consulting lifecycle feature. When the firm updates its base template (new CI pipeline stages, updated security policies, revised CLAUDE.md conventions), engineers run `qsdev update --template` to pull those changes forward. Copier's 3-way merge applies template changes while preserving local modifications. After Copier updates the scaffolding, gdev re-runs its own update flow (Phase 8, Unit 6.1) to reconcile any changes to `.qsdev.yaml` or other gdev-managed files that the template update may have affected.

**Desired Outcome:** `qsdev update --template` brings an existing project forward to the latest template version, then reconciles qsdev config, with clear reporting of what changed and any merge conflicts.

**Steps:**
1. Add `--template` flag to `qsdev update` command.
2. Detect whether the project is Copier-templated by checking for `.copier-answers.yml`.
3. If not Copier-templated, return clear error: "This project was not created from a template. Use `qsdev init --from <template>` to start from a template."
4. Read `.copier-answers.yml` to extract template URL and current version for display.
5. Run `CopierRunner.Update()` (Unit 5.8).
6. Report Copier results: files updated, files with conflicts (`.rej` files), template version change.
7. If `.qsdev.yaml` was modified by the template update, re-run qsdev init in update mode (Phase 8, Unit 6.1) to reconcile devenv.nix, settings.json, CLAUDE.md, etc.
8. If `.qsdev.yaml` was not modified, skip gdev re-init but report that qsdev config is unchanged.
9. Final summary: template version (old → new), files changed by template, files changed by gdev re-init, conflicts requiring manual resolution.

**Acceptance Criteria:**
- [ ] `qsdev update --template` runs `copier update` on a Copier-templated project
- [ ] Clear error when run on a non-Copier-templated project
- [ ] Displays template version change (old → new)
- [ ] Reports files modified by template update
- [ ] Reports `.rej` files requiring manual conflict resolution
- [ ] Re-runs qsdev init in update mode when `.qsdev.yaml` changed
- [ ] Skips gdev re-init when `.qsdev.yaml` unchanged
- [ ] `qsdev update --template --yes` runs non-interactively
- [ ] Final summary distinguishes template changes from gdev changes

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — `copier update` for template lifecycle
- `implementation-plans/qsdev/phases/08-migration-update-polish.md § Unit 6.1` — qsdev update flow with hash-based modification detection

**Status:** Not Started

---

### Unit 5.11: Non-Interactive & CI Mode for Template Workflows

**Description:** Ensure all Copier template workflows support fully non-interactive execution via `--yes` and `--data` flags, enabling CI pipelines and scripted project setup.

**Context:** CI systems and scripted onboarding flows need to create projects from templates without interactive questionnaires. Copier supports `--defaults` (use `copier.yaml` default values) and `--data key=value` (override specific answers). gdev must bridge its `--yes` flag to Copier's `--defaults` and support forwarding template-specific data.

**Desired Outcome:** `qsdev init --from ts-api --yes` and `qsdev init --from ts-api --data project_name=acme --data author=colin --yes` both produce a complete project with zero interactive prompts.

**Steps:**
1. Map `qsdev init --yes` to `copier copy --defaults` (use template-defined defaults for all Copier questions).
2. Implement `--data key=value` flag (repeatable) on `qsdev init --from` to pass answers to Copier.
3. Implement `--template-ref <tag-or-branch>` flag to pin a specific template version (maps to `copier copy --vcs-ref`).
4. When `--yes` is set but Copier has required questions without defaults, report which questions need `--data` answers and exit with clear error (do not hang waiting for input).
5. Map `qsdev update --template --yes` to `copier update --defaults`.
6. Document the full non-interactive invocation in `qsdev init --help`.
7. Validate with a test scenario: CI script that creates a project from template, runs qsdev init, and verifies all expected files exist.

**Acceptance Criteria:**
- [ ] `qsdev init --from ts-api --yes` uses Copier defaults, no prompts
- [ ] `qsdev init --from ts-api --data project_name=acme --yes` overrides specific Copier answers
- [ ] `--template-ref v2.0` pins template to a specific git tag
- [ ] Missing required Copier answers with no defaults produce clear error listing needed `--data` keys
- [ ] `qsdev update --template --yes` runs non-interactively
- [ ] All flags documented in `qsdev init --help` and `qsdev update --help`

**Research Citations:**
- `implementation-plans/qsdev/phases/06-wizard-orchestration.md § Unit 5.5` — existing non-interactive/CI flag mapping
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — non-interactive mode requirement

**Status:** Not Started

---

### Unit 5.12: Firm-Wide Template Standards Documentation

**Description:** Define and document the conventions for firm-maintained Copier templates that integrate with gdev, including what files a template should contain, how `.qsdev.yaml` in templates pre-configures gdev, and how template-provided CLAUDE.md interacts with gdev-generated CLAUDE.md.

**Context:** The value of Copier integration depends on well-structured templates. This unit does not implement code — it produces the specification that template authors follow. The spec covers file layout conventions, `.qsdev.yaml` integration points, CLAUDE.md section conventions, and CI pipeline templates. This is embedded documentation generated by `qsdev init --from` when creating a new template repo.

**Desired Outcome:** A clear specification for template authors that ensures Copier templates and gdev's generation pipeline work together without conflicts.

**Steps:**
1. Define recommended template directory structure:
   ```
   template-repo/
   ├── copier.yaml           # Copier questionnaire definition
   ├── {{ project_name }}/   # (or root-level files)
   │   ├── .qsdev.yaml        # Pre-configured gdev org defaults
   │   ├── CLAUDE.md         # Firm-wide Claude Code conventions
   │   ├── README.md.jinja   # README with engagement metadata
   │   ├── .github/
   │   │   └── workflows/    # CI/CD pipeline templates
   │   ├── .security/        # Security policy files
   │   └── ...
   ```
2. Document `.qsdev.yaml` integration: templates can set `compliance_level`, `security_profile`, `default_services`, `org_registry`, and other qsdev config that the wizard consumes as defaults.
3. Document CLAUDE.md conventions: template-provided CLAUDE.md uses gdev's section markers so gdev can append its generated sections without overwriting firm conventions. Template provides firm-level rules; gdev appends project-level rules.
4. Document CI pipeline template conventions: GitHub Actions workflows that reference gdev commands (`qsdev check`, `qsdev status --json`).
5. Document `.copier-answers.yml` — explain that this file is auto-generated by Copier, must be committed to git, and tracks the template version for `copier update`.
6. Embed this spec as a help topic: `qsdev template --help-authoring` or as generated documentation in a new template scaffold.

**Acceptance Criteria:**
- [ ] Template directory structure documented with rationale for each convention
- [ ] `.qsdev.yaml` integration points documented (which keys templates should set)
- [ ] CLAUDE.md section marker conventions documented for template authors
- [ ] CI pipeline template conventions documented
- [ ] `.copier-answers.yml` purpose and commit requirement documented
- [ ] Spec is accessible via `qsdev template --help-authoring` or similar

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md § Feature 5` — firm-wide templates as the primary use case
- `research-spikes/gdev-team-config-onboarding/research.md § Three-Layer Config` — `.qsdev.yaml` as org config carrier
- `implementation-plans/qsdev/phases/04-claude-code-addon-core-generation.md` — CLAUDE.md section marker pattern
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Section Markers` — marker-based merge for CLAUDE.md

**Status:** Not Started

---

## Phase Completion Criteria (Amended)

The original Phase 6 completion criteria remain. The following are added:

- [ ] `qsdev init --from <template-url>` creates a project from a Copier template and runs qsdev init on the result
- [ ] `qsdev init --from <short-name> --yes` resolves name from registry and runs fully non-interactive
- [ ] `qsdev update --template` pulls latest template changes and reconciles qsdev config
- [ ] `qsdev template list/add/remove` manages the template registry
- [ ] Template workflows work without Copier installed (clear error with install instructions)
- [ ] Non-Copier projects are unaffected by any of these changes (no regression)

## Dependency Notes

- **Units 5.7-5.12 depend on Unit 5.1** (command hierarchy — `--from` flag added to `qsdev init`).
- **Unit 5.9 depends on Units 5.7 and 5.8** (needs registry resolution and Copier invocation).
- **Unit 5.10 depends on Unit 5.8** (needs Copier invocation) and **Phase 8, Unit 6.1** (qsdev update flow).
- **Unit 5.11 depends on Units 5.9 and 5.10** (extends their non-interactive behavior).
- **Unit 5.12 has no code dependencies** and can be written in parallel with any other unit.
