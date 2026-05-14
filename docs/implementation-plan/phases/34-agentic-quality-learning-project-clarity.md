# Phase 34: Agentic Quality, Learning & Project Clarity

## Goal

Apply research-backed agentic quality patterns to gdev's generated configuration. The central finding from agentic workflow SOTA research: scaffold architecture produces a 27× larger performance swing than model changes. External verification doubles task success rates. Aggressive directives degrade Claude 4.6 performance. This phase translates those findings into six concrete deliverables: two opt-in learning skills from the DrCatHicks ecosystem, a project clarity template for CLAUDE.md, a tree-sitter repo map skill, a pre-edit validation hook, an audit pass replacing aggressive language in all gdev-generated templates, and time-to-first-environment benchmarking.

## Dependencies

Phase 13 complete (project configuration — `.gdev.yaml` project name, Join mode, onboarding modes). Phase 14 complete (Claude Code integration and agentic skills — skill deployment infrastructure, `gdev claude init`, existing skill manifest and `deploySkills()` function).

## Phase Outputs

- `learning-opportunities` skill deployed via `gdev enable learning-opportunities`
- `orient` skill deployed via `gdev enable orient`
- Project clarity CLAUDE.md template section (with consulting and Copier integration)
- `repo-map` skill deployed via `gdev enable repo-map`
- Pre-edit validation hook for Edit/Write tools
- Calm directive audit pass across all gdev-generated templates and rule files
- Time-to-first-environment benchmarking at `~/.qsdev/benchmarks/`

---

### Unit 34.1: learning-opportunities Skill Integration

**Description:** Integrate the `learning-opportunities` skill from DrCatHicks/learning-opportunities (1,530 stars, CC-BY-4.0) as an opt-in skill deployed via `gdev enable learning-opportunities`. The skill offers research-grounded learning exercises after architectural work to counter the fluency illusion — the consulting-specific risk that AI-accelerated output generates code without building genuine developer understanding.

**Context:** The learning opportunities evaluation established that the skill is a strong fit for consulting: it directly addresses the "fluency illusion" problem (engineers producing AI-accelerated output without understanding unfamiliar client codebases), is properly licensed CC-BY-4.0, has zero runtime dependencies, and is deployed as a standard SKILL.md file compatible with gdev's existing skill infrastructure. The skill has no `disable-model-invocation` flag, which means Claude can autonomously invoke it — intentionally, so it proactively offers exercises after architectural work without requiring explicit invocation.

The 6 exercise types are grounded in published learning science: generation effect, fluency illusion counter, spacing effect, metacognition, testing effect, and interleaving. The research recommendation is to enforce a maximum of 2 exercises per session and stop after the first decline, keeping the learning interruptions below the threshold where they become friction.

gdev deploys the skill by embedding the SKILL.md and `resources/` files in its Go binary via the existing `internal/claudecode/skills/` embedded filesystem, following the identical pattern used for Phase 14's existing skill library. The `deploySkills()` function at `addons/claudecode/generate_skills.go:42-78` already handles recursive directory copying.

**Desired Outcome:** `gdev enable learning-opportunities` copies the SKILL.md and resources/ into `.claude/skills/learning-opportunities/` with correct attribution. Claude autonomously offers exercises after architectural work. The skill respects the 2-exercise-per-session limit and stops after the first decline.

**Steps:**
1. Obtain and embed the skill files:
   - Source: `https://github.com/DrCatHicks/learning-opportunities` (CC-BY-4.0)
   - Files to embed in `internal/claudecode/skills/learning-opportunities/`:
     - `SKILL.md` (from `learning-opportunities/skills/learning-opportunities/SKILL.md`)
     - `resources/PRINCIPLES.md` (from `learning-opportunities/skills/learning-opportunities/resources/PRINCIPLES.md`)
   - Add attribution comment to the embedded SKILL.md front matter header:
     ```yaml
     # Learning Opportunities skill by Dr. Cat Hicks (CC-BY-4.0)
     # Source: https://github.com/DrCatHicks/learning-opportunities
     # Deployed by gdev — do not edit. Run: gdev enable learning-opportunities
     ```
2. Add `learning-opportunities` to the skill manifest at `templates/skills/manifest.yaml`:
   ```yaml
   - name: learning-opportunities
     description: "Offers deliberate practice exercises after architectural work (Dr. Cat Hicks, CC-BY-4.0)"
     opt_in: true
     disable_model_invocation: false
     author: "DrCatHicks"
     license: "CC-BY-4.0"
   ```
3. Implement `gdev enable learning-opportunities`:
   - Register the skill in the tool registry (Phase 12 lifecycle system).
   - Call `deploySkill("learning-opportunities")` which copies the embedded SKILL.md and resources/ into `.claude/skills/learning-opportunities/` using the existing `deploySkills()` infrastructure.
   - Print confirmation: `learning-opportunities skill deployed to .claude/skills/learning-opportunities/`.
4. Add the session limit enforcer as a note in the deployed SKILL.md header (not a hook):
   - The attribution header includes: `Maximum 2 exercises per session. Stop after the first decline.`
   - This is a CLAUDE.md-style instruction prepended to the embedded SKILL.md.
   - This customization must survive skill updates (keep as a separate prepend, not inline edit).
5. Wire into Join mode suggestion (Phase 13 Unit 13.4):
   - After Join mode completes successfully, if learning-opportunities is enabled in `.gdev.yaml`, print:
     ```
     After exploring the codebase, run /orient to generate orientation.md,
     then /learning-opportunities orient for a guided onboarding tour.
     ```
6. Implement `gdev disable learning-opportunities`:
   - Remove `.claude/skills/learning-opportunities/` directory.
   - Remove entry from skill manifest state tracking.
7. Write unit tests:
   - `gdev enable learning-opportunities` creates `.claude/skills/learning-opportunities/SKILL.md`.
   - Attribution header present in deployed SKILL.md.
   - `resources/PRINCIPLES.md` deployed alongside SKILL.md.
   - `gdev disable learning-opportunities` removes the directory.
   - Re-enabling after disable recreates files correctly.

**Acceptance Criteria:**
- [ ] `gdev enable learning-opportunities` deploys SKILL.md and `resources/PRINCIPLES.md` to `.claude/skills/learning-opportunities/`
- [ ] Attribution comment present in deployed SKILL.md: author, license, source URL
- [ ] Session limit (maximum 2 exercises, stop after first decline) present as instruction in deployed SKILL.md header
- [ ] `disable-model-invocation` is NOT set — Claude can autonomously offer exercises
- [ ] Skill appears in `gdev info` when enabled
- [ ] `gdev disable learning-opportunities` removes the skill directory
- [ ] CC-BY-4.0 license attribution present in gdev's own documentation

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/learning-opportunities-research.md § 1-3` — format compatibility, deployment path via `deploySkills()`, session limit recommendation, consulting fit assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/learning-opportunities-research.md § 4` — exclusion of learning-opportunities-auto (hook conflicts with gdev's existing hook architecture)
- `phases/14-claude-code-integration-agentic-skills.md` — skill deployment infrastructure, `deploySkills()` function, embedded filesystem pattern

**Status:** Not Started

---

### Unit 34.2: orient Codebase Exploration Skill

**Description:** Integrate the `orient` skill from DrCatHicks/learning-opportunities (CC-BY-4.0, author Dr. Michael Mullarkey) as an opt-in skill deployed via `gdev enable orient`. Orient generates a `resources/orientation.md` file by performing a structured 6-step codebase exploration, producing a teaching scaffold that accelerates onboarding in unfamiliar client codebases.

**Context:** The learning opportunities evaluation identified orient as "the strongest fit for consulting" among the three plugins in the DrCatHicks ecosystem. The consulting scenario it directly addresses: an engineer clones a client codebase they have never seen before and must be productive within hours, not weeks. Orient generates a structured orientation document — purpose statement, primary languages, pipeline stages, key files, core concepts, gotchas, and suggested exercises — by performing a methodical sampling of the codebase following program comprehension research (Spinellis 2003, Hermans 2021, Storey et al. 2006).

Orient has `disable-model-invocation: true`, meaning Claude will NOT invoke it autonomously — the developer must run `/orient` explicitly. This is correct: generating orientation.md involves significant file reading and should be a deliberate action, not a background task. The orientation.md file it produces becomes a persistent resource used by the learning-opportunities skill's `orient` argument.

gdev adds a post-Join-mode suggestion linking orient to the Join mode workflow: after `gdev init` in Join mode, the summary prints a prompt to run `/orient` as the next step.

**Desired Outcome:** `gdev enable orient` deploys the orient skill. Running `/orient` inside Claude Code generates `resources/orientation.md` with a structured codebase overview. Post-Join-mode output mentions `gdev orient` as the next step when the skill is enabled.

**Steps:**
1. Obtain and embed the skill files:
   - Source: `https://github.com/DrCatHicks/learning-opportunities` (CC-BY-4.0, author Dr. Michael Mullarkey for orient)
   - Files to embed in `internal/claudecode/skills/orient/`:
     - `SKILL.md` (from `orient/skills/orient/SKILL.md`)
     - `resources/orient-bibliography.md` (from `orient/skills/orient/resources/orient-bibliography.md`)
   - Add attribution comment in the embedded SKILL.md front matter header:
     ```yaml
     # Orient skill by Dr. Michael Mullarkey (CC-BY-4.0)
     # Source: https://github.com/DrCatHicks/learning-opportunities
     # Deployed by gdev — do not edit. Run: gdev enable orient
     ```
2. Add `orient` to the skill manifest at `templates/skills/manifest.yaml`:
   ```yaml
   - name: orient
     description: "Generates orientation.md for unfamiliar codebases via 6-step structured exploration (CC-BY-4.0)"
     opt_in: true
     disable_model_invocation: true
     author: "DrCatHicks/Dr. Michael Mullarkey"
     license: "CC-BY-4.0"
     invocation: "/orient"
   ```
3. Implement `gdev enable orient`:
   - Register in tool registry.
   - Deploy SKILL.md and resources/ to `.claude/skills/orient/`.
   - Print: `orient skill deployed. Run /orient inside Claude Code to generate resources/orientation.md`.
4. Implement `gdev orient` as a convenience alias for instructing the developer:
   ```go
   var orientCmd = &cobra.Command{
       Use:   "orient",
       Short: "Generate orientation.md for this codebase (runs inside Claude Code)",
       RunE:  runOrient,
   }

   func runOrient(cmd *cobra.Command, args []string) error {
       if !skillIsDeployed("orient") {
           fmt.Println("orient skill not deployed. Run: gdev enable orient")
           return nil
       }
       fmt.Println("Run /orient inside Claude Code to generate resources/orientation.md")
       fmt.Println("The file will be created at: resources/orientation.md")
       return nil
   }
   ```
   - Note: `gdev orient` itself cannot invoke Claude Code — it only informs the user. The actual invocation is `/orient` inside a Claude Code session.
5. Wire orient suggestion into Phase 13 Join mode completion output (Unit 13.4):
   - If orient skill is deployed, add to the Join mode completion summary:
     ```
     Next steps:
       1. devenv shell          — enter the development environment
       2. /orient               — generate orientation.md (run inside Claude Code)
       3. /learning-opportunities orient — guided onboarding exercises
     ```
6. Implement `gdev disable orient`:
   - Remove `.claude/skills/orient/` directory.
   - Remove entry from skill manifest state tracking.
7. Write unit tests:
   - `gdev enable orient` creates `.claude/skills/orient/SKILL.md`.
   - Attribution comment present in deployed SKILL.md.
   - `resources/orient-bibliography.md` deployed alongside SKILL.md.
   - `disable-model-invocation: true` is present in the deployed SKILL.md frontmatter.
   - Join mode completion output includes orient suggestion when skill is deployed.
   - `gdev disable orient` removes the directory.

**Acceptance Criteria:**
- [ ] `gdev enable orient` deploys SKILL.md and `resources/orient-bibliography.md` to `.claude/skills/orient/`
- [ ] Attribution comment present: author (Dr. Michael Mullarkey), license (CC-BY-4.0), source URL
- [ ] `disable-model-invocation: true` preserved in deployed SKILL.md frontmatter
- [ ] `allowed-tools: Read, Glob, Grep, Bash, Write` preserved (no web access)
- [ ] Join mode completion output mentions `/orient` as next step when skill is deployed
- [ ] `gdev disable orient` removes the skill directory
- [ ] CC-BY-4.0 license attribution present in gdev's own documentation

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/learning-opportunities-research.md § 2` — orient deep analysis, 6-step exploration methodology, program comprehension research citations, consulting fit
- `research-spikes/gdev-ecosystem-expansion-assessment/learning-opportunities-research.md § Integration with gdev Join Mode` — Join mode integration pattern
- `phases/13-project-configuration-team-standards.md § Unit 13.4` — Join mode completion flow, where to insert orient suggestion

**Status:** Not Started

---

### Unit 34.3: Project Clarity Template for CLAUDE.md

**Description:** Add a "Project Context" section to the CLAUDE.md template with fields for purpose, stakeholders, success criteria, exclusions, and a consulting-specific block covering client name, engagement type, compliance level, and handoff date. Wire to Copier (Phase 31) questionnaire so the section is populated during `gdev init`. Wire to client profile (Phase 30) when available.

**Context:** The agentic workflow research established that CLAUDE.md functions as the system prompt for Claude Code, and that system prompt quality is a primary quality lever: a well-structured CLAUDE.md with explicit success criteria and project context produces measurably better output than a sparse one. The "write prompts like contracts" principle from the prompt engineering research directly translates to the Project Context section — explicit purpose, success criteria, and exclusions give Claude a contract to work against rather than an open-ended mandate.

The consulting-specific block captures the information that changes with every engagement: client, type of work (greenfield vs brownfield vs assessment), compliance constraints, and handoff date. These fields prevent Claude from making recommendations appropriate for a greenfield startup project when working on a brownfield HIPAA-constrained client system. The Copier integration (Phase 31) allows `copier.yaml` questionnaire answers to populate these fields during `copier copy`, so they are filled in rather than left as placeholder text for the developer to edit.

**Desired Outcome:** The generated CLAUDE.md contains a Project Context section populated during `gdev init` (wizard) or `copier copy` (Copier flow). The section is in the user-editable area, outside gdev section markers, so it is not overwritten by `gdev init --update`. The consulting block is included only when the project profile is `consulting-default` or `enterprise`.

**Steps:**
1. Define the Project Context template block (user-editable, outside gdev markers):
   ```markdown
   ## Project Context

   **Purpose:** [One sentence: what this project does and why it exists]

   **Stakeholders:** [Who this is for and who needs to approve changes]

   **Success Criteria:**
   - [Measurable outcome 1]
   - [Measurable outcome 2]

   **Exclusions:** [What is explicitly out of scope for this engagement]

   **Technical Constraints:**
   - [Framework version pins, API compatibility requirements, deployment target]
   - [Languages and runtimes that cannot be changed]
   ```
2. Define the consulting-only block (added after Technical Constraints when profile is consulting-adjacent):
   ```markdown
   ## Consulting Context

   **Client:** [Client name or alias]
   **Engagement type:** [greenfield | brownfield | migration | assessment | staff-aug]
   **Compliance level:** [baseline | enhanced | strict]
   **Handoff date:** [YYYY-MM-DD or "ongoing"]
   **Data classification:** [public | internal | confidential]
   ```
3. Implement placement logic in the CLAUDE.md generator:
   - Project Context section is inserted BEFORE the first gdev section marker.
   - It uses a comment marker so it can be detected but not overwritten: `<!-- gdev:project-context (user-editable) -->` on the line before.
   - On `gdev init --update`: presence of the marker means this section is NEVER regenerated (idempotent — only generate on first Create or Join mode).
4. Wire to wizard:
   - Add `ProjectContext` struct to `WizardAnswers`:
     ```go
     type ProjectContextAnswers struct {
         Purpose      string
         Stakeholders string
         Success      []string // bullet points
         Exclusions   string
         Constraints  []string

         // Consulting only (populated when profile is consulting-*):
         ClientName      string
         EngagementType  string // greenfield/brownfield/migration/assessment/staff-aug
         ComplianceLevel string
         HandoffDate     string
         DataClass       string
     }
     ```
   - Add a "Project Context" form group to the wizard (Phase 6 huh form integration).
   - All fields are optional with sensible placeholder text; skipping produces placeholder-filled template.
   - Consulting block fields only shown when profile is `consulting-default` or `enterprise`.
5. Wire to Copier (Phase 31) `copier.yaml` template:
   ```yaml
   # copier.yaml
   project_purpose:
     type: str
     help: "One sentence: what this project does and why it exists"
     default: ""

   project_stakeholders:
     type: str
     help: "Who this is for (e.g., 'payments team, product manager')"
     default: ""

   client_name:
     type: str
     help: "Client name or alias (consulting engagements only)"
     default: ""
     when: "{{ profile in ['consulting-default', 'enterprise'] }}"

   engagement_type:
     type: str
     help: "Engagement type"
     choices: [greenfield, brownfield, migration, assessment, staff-aug]
     default: greenfield
     when: "{{ profile in ['consulting-default', 'enterprise'] }}"
   ```
   - Copier answers are persisted in `.copier-answers.yml` and survive `copier update`.
   - On `copier update`, Project Context section is NOT regenerated (idempotent, per step 3).
6. Wire to client profile (Phase 30) when available:
   - If a client profile exists for the current project, pre-populate `ClientName` and `ComplianceLevel` from the profile.
   - The wizard shows these as pre-filled values the user can confirm or override.
7. Write unit tests:
   - Project Context section generated in CLAUDE.md on first Create mode run.
   - Section uses `<!-- gdev:project-context (user-editable) -->` marker.
   - `gdev init --update` does NOT regenerate the section (idempotent).
   - Consulting block included when profile is `consulting-default`.
   - Consulting block omitted when profile is `startup-fast`.
   - Copier template contains correct questions for consulting fields.

**Acceptance Criteria:**
- [ ] Generated CLAUDE.md contains a Project Context section with Purpose, Stakeholders, Success Criteria, Exclusions, Technical Constraints
- [ ] Consulting Context block included when profile is `consulting-default` or `enterprise`
- [ ] Consulting Context block omitted when profile is `startup-fast` or non-consulting profiles
- [ ] Project Context section uses a user-editable marker that prevents overwrite by `gdev init --update`
- [ ] Wizard populates Project Context fields during Create mode
- [ ] Consulting fields (client name, engagement type, compliance level, handoff date) shown only in wizard when consulting profile is selected
- [ ] Copier template (`copier.yaml`) contains corresponding questions for all Project Context fields
- [ ] Copier answers persist across `copier update` cycles via `.copier-answers.yml`

**Research Citations:**
- `research-spikes/agentic-workflow-state-of-art/prompt-engineering-research.md § 1` — "write prompts like contracts," success criteria and output contract as top practice, CLAUDE.md as system prompt
- `research-spikes/agentic-workflow-state-of-art/research.md § Prompt Engineering` — aggressive language degrades performance; clear context improves output quality
- `research-spikes/gdev-ecosystem-expansion-assessment/copier-integration-design.md` — Copier questionnaire design, `.copier-answers.yml` persistence, when-clauses for conditional questions

**Status:** Not Started

---

### Unit 34.4: Tree-sitter Repo Map Skill

**Description:** Implement a `repo-map` skill that generates a ~1K token structural overview of a codebase via AST-based symbol extraction and PageRank-style reference ranking. Deployed as `.claude/skills/repo-map/SKILL.md` with a shell command preprocessor. The repo map is the single highest-impact addition for codebase understanding, per the agentic SOTA research.

**Context:** The agentic SOTA research identified Aider's tree-sitter repo map as "the single highest-impact addition for codebase understanding." The mechanism: tree-sitter parses source files to extract function, class, and method names, then a PageRank-like algorithm ranks symbols by how many other files reference them. The result is a compact (~1K token) list of the most-referenced symbols with file locations — sufficient to orient Claude to a codebase without consuming the entire context window on file contents. The research notes that this approach outperforms simple directory listings because it surfaces semantically important symbols rather than just file names.

The implementation uses a two-step approach. The shell command preprocessor (`repo-map.sh`) runs tree-sitter queries against source files and produces the ranked symbol list. The SKILL.md then injects this output as context before code exploration. This is the `disable-model-invocation: true` / explicit invocation pattern — `/repo-map` runs the preprocessor and injects its output, which Claude then uses to navigate the codebase. The SKILL.md instructs Claude to use the repo map as an index, not as a substitute for reading key files.

The tree-sitter implementation uses the `tree-sitter` CLI (available in nixpkgs) plus per-language grammar packages. The shell script handles the common case (Go, TypeScript/JavaScript, Python, Rust) and gracefully degrades for unsupported languages (outputs a basic directory tree instead).

**Desired Outcome:** `gdev enable repo-map` deploys the skill and the shell preprocessor. Running `/repo-map` inside Claude Code injects a ranked symbol list of the codebase's most important symbols into the conversation, fitting in approximately 1,000 tokens.

**Steps:**
1. Create `internal/claudecode/skills/repo-map/SKILL.md`:
   ```yaml
   ---
   name: repo-map
   description: Generates a ranked symbol map of the codebase (~1K tokens) via tree-sitter AST analysis
   disable-model-invocation: true
   allowed-tools: Bash, Read
   argument-hint: "[path]"
   ---
   ```
   SKILL.md body instructs Claude to:
   - Run the repo-map preprocessor: `bash .claude/skills/repo-map/repo-map.sh [path]`
   - Read the output as a structural index of the codebase.
   - Use the ranked symbols to identify which files to read first.
   - NOT use the repo map as a substitute for reading key files — use it as an index.
   - If the map shows a function referenced by 20+ other files, read that file before making changes.
2. Create `internal/claudecode/skills/repo-map/repo-map.sh`:
   ```bash
   #!/usr/bin/env bash
   # repo-map.sh — Generate ranked symbol map via tree-sitter
   # Output: ranked list of symbols with file locations, ~1K tokens
   set -euo pipefail

   ROOT="${1:-.}"
   MAX_SYMBOLS=80
   MAX_FILE_SIZE=102400  # 100KB — skip large generated files

   # Detect languages present
   has_go=false; has_ts=false; has_py=false; has_rs=false
   find "$ROOT" -name "*.go" -not -path "*/vendor/*" -maxdepth 5 | head -1 | grep -q . && has_go=true
   find "$ROOT" -name "*.ts" -not -path "*/node_modules/*" -maxdepth 5 | head -1 | grep -q . && has_ts=true
   find "$ROOT" -name "*.py" -not -path "*/.venv/*" -maxdepth 5 | head -1 | grep -q . && has_py=true
   find "$ROOT" -name "*.rs" -not -path "*/target/*" -maxdepth 5 | head -1 | grep -q . && has_rs=true

   # Check tree-sitter availability
   if ! command -v tree-sitter &>/dev/null; then
       echo "# Repo Map (directory fallback — install tree-sitter for symbol analysis)"
       find "$ROOT" -type f \( -name "*.go" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" \) \
           -not -path "*/vendor/*" -not -path "*/node_modules/*" -not -path "*/.venv/*" \
           -not -path "*/target/*" | head -60
       exit 0
   fi

   # Run tree-sitter queries per language and collect symbol definitions
   # ... (Go: func/type definitions; TS: export function/class/interface; Python: def/class; Rust: pub fn/struct/trait)
   # ... Reference counting: count how many files import each symbol
   # ... PageRank-style: score = reference_count / total_files_scanned
   # Output: sorted by score descending, capped at MAX_SYMBOLS
   ```
3. Implement the symbol extraction logic in `repo-map.sh`:
   - For each supported language, run appropriate tree-sitter queries to extract symbol names and locations.
   - For Go: extract `func`, `type`, `interface` at package level.
   - For TypeScript: extract `export function`, `export class`, `export interface`, `export type`.
   - For Python: extract top-level `def`, `class`.
   - For Rust: extract `pub fn`, `pub struct`, `pub trait`, `pub enum` at crate level.
   - Count cross-file references by searching for each symbol name in all other source files (`grep -rl`).
   - Sort by reference count descending, take top `MAX_SYMBOLS`.
   - Output format:
     ```
     # Repo Map — generated by gdev/repo-map
     # 847 tokens | 42 files analyzed | top 80 symbols by reference count

     ## internal/config/resolve.go
     ResolveConfig() [referenced by 14 files]
     GdevConfig{} [referenced by 11 files]
     ParseGdevConfig() [referenced by 8 files]

     ## internal/wizard/wizard.go
     RunWizard() [referenced by 6 files]
     WizardAnswers{} [referenced by 23 files]
     ...
     ```
4. Add tree-sitter to devenv packages when repo-map is enabled:
   - In devenv.nix generated section: `pkgs.tree-sitter` added to packages.
   - The shell script degrades gracefully if tree-sitter is not available (directory fallback).
5. Implement `gdev enable repo-map`:
   - Deploy SKILL.md and `repo-map.sh` to `.claude/skills/repo-map/`.
   - Make `repo-map.sh` executable: `os.Chmod(".claude/skills/repo-map/repo-map.sh", 0o755)`.
   - Add `pkgs.tree-sitter` to devenv.nix packages section.
6. Implement `gdev disable repo-map`:
   - Remove `.claude/skills/repo-map/` directory.
   - Remove `pkgs.tree-sitter` from devenv.nix (Phase 12 section marker editing).
7. Write unit tests:
   - `gdev enable repo-map` creates `.claude/skills/repo-map/SKILL.md` and `repo-map.sh`.
   - `repo-map.sh` is executable after deployment.
   - `disable-model-invocation: true` present in deployed SKILL.md frontmatter.
   - tree-sitter added to devenv.nix packages.
   - Script gracefully falls back to directory listing when tree-sitter binary absent.
   - `gdev disable repo-map` removes both files and the tree-sitter package.

**Acceptance Criteria:**
- [ ] `gdev enable repo-map` deploys SKILL.md and `repo-map.sh` to `.claude/skills/repo-map/`
- [ ] `repo-map.sh` is executable (mode 0755) after deployment
- [ ] `disable-model-invocation: true` present in deployed SKILL.md frontmatter
- [ ] Script supports Go, TypeScript, Python, and Rust symbol extraction
- [ ] Script degrades gracefully to directory listing when tree-sitter is unavailable
- [ ] Symbol output capped at 80 symbols to target ~1K token output
- [ ] Each symbol entry shows file path and reference count
- [ ] `pkgs.tree-sitter` added to devenv.nix when repo-map is enabled
- [ ] `gdev disable repo-map` removes the skill directory and tree-sitter package

**Research Citations:**
- `research-spikes/agentic-workflow-state-of-art/memory-context-management-research.md` — tree-sitter repo map as highest-impact codebase understanding addition, Aider's AST + PageRank approach
- `research-spikes/agentic-workflow-state-of-art/tool-use-patterns-research.md` — structured codebase map as second-highest-impact Claude Code improvement, codebase navigation patterns
- `research-spikes/agentic-workflow-state-of-art/research.md § Memory and Context Management` — repo map summary and implementation note

**Status:** Not Started

---

### Unit 34.5: Pre-Edit Validation Hook

**Description:** Implement a PreToolUse hook on Edit and Write tools that runs an ecosystem-appropriate linter on the target file before the edit is applied. Advisory mode by default — warns but does not block. Promoted to blocking via profile configuration. Must complete in under 200ms using cached linter state.

**Context:** The agentic SOTA research identified pre-edit validation as SWE-agent's core quality pattern and listed it as the highest-impact tool improvement for Claude Code: "validate before writing, not after." The mechanism: intercept Edit/Write tool calls, extract the target file path, determine the file's language, run the appropriate linter on the current file state, and surface any existing lint errors before the edit proceeds. This gives Claude awareness of the current lint state, preventing it from introducing edits that compound pre-existing errors or that break the file's parseable structure.

The performance constraint is strict: 200ms maximum. This is achievable by linting only the single target file (not the whole project) and using cached linter invocations. Most linters support single-file mode: `eslint <file>`, `ruff check <file>`, `golangci-lint run <file>`, `clippy` via `rustc --error-format=short <file>`. The hook runs synchronously in the PreToolUse position, so latency directly affects developer experience.

Advisory mode (default) surfaces lint errors as a warning in Claude's context — Claude sees them and can decide whether to address them, but the edit proceeds. Blocking mode is for teams that want to enforce lint-clean edits: the hook returns a non-zero exit code, blocking the edit and surfacing the lint errors as the reason.

**Desired Outcome:** PreToolUse hook on Edit/Write runs a per-file linter in under 200ms. In advisory mode, lint errors appear as warnings in Claude's context. In blocking mode, edits to files with lint errors are blocked until the errors are resolved.

**Steps:**
1. Create the pre-edit validation hook script at `templates/hooks/pre-edit-validate.sh.tmpl`:
   ```bash
   #!/usr/bin/env bash
   # pre-edit-validate.sh — pre-tool-use lint validation for Edit/Write tools
   # Runs in <200ms by linting only the target file (not the whole project).
   set -euo pipefail

   # Parse tool use payload from stdin
   PAYLOAD=$(cat)
   TOOL_NAME=$(echo "$PAYLOAD" | jq -r '.tool_name // empty')
   FILE_PATH=$(echo "$PAYLOAD" | jq -r '.tool_input.path // .tool_input.file_path // empty')

   # Skip if not an Edit/Write tool use or no file path
   [[ "$TOOL_NAME" == "Edit" || "$TOOL_NAME" == "Write" ]] || exit 0
   [[ -n "$FILE_PATH" ]] || exit 0
   [[ -f "$FILE_PATH" ]] || exit 0  # New files: no pre-existing lint state

   # Determine linter by file extension
   case "${FILE_PATH##*.}" in
     ts|tsx|js|jsx|mjs|cjs)
       [[ -f "eslint.config.js" || -f ".eslintrc*" || -f "eslint.config.ts" ]] || exit 0
       LINTER="eslint --no-eslintrc --quiet"
       ;;
     py)
       command -v ruff &>/dev/null || exit 0
       LINTER="ruff check --quiet --no-fix"
       ;;
     go)
       command -v golangci-lint &>/dev/null || exit 0
       LINTER="golangci-lint run --fast --out-format=line-number"
       ;;
     rs)
       # Rust requires cargo context; skip single-file lint
       exit 0
       ;;
     *)
       exit 0
       ;;
   esac

   # Run linter with 180ms timeout (leave 20ms buffer for process overhead)
   LINT_OUTPUT=$(timeout 0.18 $LINTER "$FILE_PATH" 2>&1) || LINT_FAILED=true

   if [[ "${LINT_FAILED:-false}" == "true" && -n "$LINT_OUTPUT" ]]; then
     MODE="${GDEV_LINT_MODE:-advisory}"  # "advisory" or "blocking"

     if [[ "$MODE" == "blocking" ]]; then
       # Return structured blocking response
       jq -n \
         --arg output "Pre-edit lint check failed. Fix these issues before editing:\n\n$LINT_OUTPUT" \
         '{"decision": "block", "reason": $output}'
     else
       # Advisory: inject lint context as a warning (non-blocking)
       jq -n \
         --arg output "Pre-edit lint check found issues in $FILE_PATH:\n\n$LINT_OUTPUT\n\nThese are pre-existing issues. Address them if the edit touches these lines." \
         '{"hookSpecificOutput": $output}'
     fi
   fi
   ```
2. Define the hook registration in `templates/hooks/hooks.json.tmpl`:
   ```json
   {
     "PreToolUse": [
       {
         "matcher": "Edit|Write",
         "hooks": [
           {
             "type": "command",
             "command": "bash .claude/hooks/pre-edit-validate.sh",
             "timeout": 200
           }
         ]
       }
     ]
   }
   ```
3. Wire `GDEV_LINT_MODE` env var generation into devenv.nix when pre-edit validation is enabled:
   - Default: `GDEV_LINT_MODE=advisory` (do not add to devenv.nix — advisory is the hook default).
   - Blocking mode: add `env.GDEV_LINT_MODE = "blocking";` to devenv.nix gdev-owned section.
4. Implement `gdev enable pre-edit-validation`:
   - Deploy `pre-edit-validate.sh` to `.claude/hooks/pre-edit-validate.sh`.
   - Register in `settings.json` PreToolUse hooks (use Phase 14 shared-file surgery for `settings.json`).
   - Make hook script executable.
   - Print: `Pre-edit validation enabled (advisory mode). To enforce blocking, add pre_edit_validation.mode: blocking to .gdev.yaml`.
5. Implement blocking mode configuration in `.gdev.yaml`:
   ```yaml
   tools:
     config:
       pre-edit-validation:
         mode: blocking  # default: advisory
   ```
   - The config resolution engine (Phase 13) reads this and sets `GDEV_LINT_MODE` accordingly.
6. Implement `gdev disable pre-edit-validation`:
   - Remove `pre-edit-validate.sh`.
   - Remove hook entry from `settings.json`.
   - Remove `GDEV_LINT_MODE` env var from devenv.nix (if blocking mode was set).
7. Write unit tests:
   - Hook script exits 0 (no output) for non-Edit/Write tool names.
   - Hook script exits 0 (no output) for tool uses without a file path.
   - Hook script exits 0 when linter binary not available.
   - Advisory mode: lint failure returns `hookSpecificOutput` (non-blocking).
   - Blocking mode: lint failure returns `{"decision": "block", "reason": ...}`.
   - Non-existent file (new file Write): exits 0 without running linter.
   - Timeout: linter taking >180ms produces no output (exits without blocking).

**Acceptance Criteria:**
- [ ] PreToolUse hook registered for Edit and Write tools in `settings.json`
- [ ] Hook runs per-file linter in under 200ms (180ms timeout enforced by the script)
- [ ] Advisory mode (default): lint errors surfaced as `hookSpecificOutput`, edit proceeds
- [ ] Blocking mode (opt-in via `.gdev.yaml`): lint errors block the edit with structured reason
- [ ] Linter selection by file extension: ESLint (TS/JS), ruff (Python), golangci-lint (Go)
- [ ] Hook exits cleanly (no output, no blocking) when linter binary is not installed
- [ ] Hook exits cleanly for new files (Write to a path that does not yet exist)
- [ ] `gdev enable pre-edit-validation` deploys the hook and registers it in `settings.json`
- [ ] `gdev disable pre-edit-validation` removes the hook from `settings.json`

**Research Citations:**
- `research-spikes/agentic-workflow-state-of-art/tool-use-patterns-research.md § Pre-Edit Validation` — SWE-agent's pre-edit validation as highest-impact tool improvement, validate-before-write pattern
- `research-spikes/agentic-workflow-state-of-art/quality-enhancing-techniques-research.md § External Verification` — external feedback as #1 quality multiplier, linter output as concrete feedback signal
- `research-spikes/agentic-workflow-state-of-art/research.md § Constrained tools outperform unconstrained` — linter-validated edits outperform raw writes

**Status:** Not Started

---

### Unit 34.6: Calm Directive Templates

**Description:** Audit and rewrite all gdev-generated CLAUDE.md templates and `.claude/rules/` files to replace aggressive emphasis markers with calm positive equivalents. The research finding is direct: aggressive and negative instructions measurably degrade Claude 4.6 performance. This unit applies a systematic replacement pass across all affected templates.

**Context:** The prompt engineering research established that "CRITICAL!", "MUST", "NEVER", and ALL-CAPS emphasis markers actively hurt performance on Claude 4.6. The mechanism is the Pink Elephant Problem: negative instructions architecturally prime the exact behavior they prohibit. The finding extends to aggressive positive framing ("YOU MUST ALWAYS"). Calm, direct positive statements outperform both. This is not a style preference — it is a measurable performance difference with direct evidence from Anthropic's own research on Claude 4.6 behavior.

The audit scope is: all `.tmpl` files in `templates/claudecode/` and `templates/rules/` that gdev generates into `.claude/` directories, plus any templates in `templates/CLAUDE.md.tmpl`. The Phase 14 templates for skills and rules, the Phase 4 core CLAUDE.md template, and the new Phase 34 templates all fall in scope. Files in `internal/` (gdev's own CLAUDE.md methodology files) are out of scope — those are not delivered to end developers.

**Desired Outcome:** All gdev-generated Claude Code configuration templates use calm positive directives. A generated CLAUDE.md and generated `.claude/rules/` files produce zero matches for the prohibited patterns: no ALL-CAPS words (except acronyms), no exclamation marks in rules, no `IMPORTANT:` prefixes, no triple-emphasis markers.

**Steps:**
1. Build the audit list — all template files in scope:
   - `templates/claudecode/CLAUDE.md.tmpl` (root CLAUDE.md template, Phase 4)
   - `templates/claudecode/rules/*.tmpl` (per-language and per-ecosystem rules, Phase 4 and Phase 14)
   - `templates/claudecode/skills/*/SKILL.md` (gdev-authored skills, not third-party CC-BY-4.0 skills — those are not modified)
   - `templates/hooks/*.sh.tmpl` (hook scripts with user-facing output strings)
2. Define the prohibited pattern list and calm replacements:
   | Prohibited | Calm replacement |
   |-----------|-----------------|
   | `CRITICAL:` | Remove prefix; state the rule directly |
   | `IMPORTANT:` | Remove prefix; state the rule directly |
   | `MUST` (as emphasis) | `always`, `use`, `ensure` |
   | `NEVER` | Reframe as positive: "Use X instead of Y" |
   | `DO NOT` | Reframe: "Use Y for X" or "Avoid X by doing Y" |
   | `YOU MUST` | Remove; state the directive |
   | `!!!` | Remove exclamation marks from rule files |
   | ALL_CAPS words (not acronyms) | lowercase or title case |
3. Apply the replacement pass:
   - For each file in the audit list, apply the replacement table.
   - Acronyms exempt from ALL-CAPS rule: `URL`, `API`, `SQL`, `JWT`, `SDK`, `CLI`, `CI`, `CD`, `MCP`, `OTel`, `YAML`, `JSON`, `HTTP`, `HTTPS`, `EOF`, `SBOM`.
   - Preserve examples that quote third-party tools using their canonical casing.
4. Implement a lint check that can be run as part of `gdev check`:
   ```go
   func checkCalmDirectives(templateDir string) []CheckResult {
       results := []CheckResult{}
       prohibitedPatterns := []struct {
           Pattern     *regexp.Regexp
           Description string
       }{
           {regexp.MustCompile(`\bCRITICAL[!:]`), "CRITICAL emphasis marker"},
           {regexp.MustCompile(`\bIMPORTANT[!:]`), "IMPORTANT emphasis marker"},
           {regexp.MustCompile(`\bNEVER\b`), "NEVER negative directive"},
           {regexp.MustCompile(`\bMUST\b`), "MUST emphasis marker"},
           {regexp.MustCompile(`\bDO NOT\b`), "DO NOT negative directive"},
           {regexp.MustCompile(`!!!`), "triple exclamation mark"},
           // ALL-CAPS word check (3+ letters, not in exempt acronym list)
           {regexp.MustCompile(`\b[A-Z]{3,}\b`), "ALL-CAPS word (check if acronym)"},
       }

       filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
           if !strings.HasSuffix(path, ".tmpl") && !strings.HasSuffix(path, ".md") {
               return nil
           }
           content, _ := os.ReadFile(path)
           for _, p := range prohibitedPatterns {
               if p.Pattern.Match(content) {
                   results = append(results, CheckResult{
                       Category: "directive_tone",
                       Name:     "calm_directive_check",
                       Status:   "fail",
                       Severity: SeverityLow,
                       Message:  fmt.Sprintf("%s: found %s in %s", path, p.Description, path),
                       Remediation: "Replace with calm positive directive. See phases/34-agentic-quality.md § Unit 34.6.",
                       FilePath: path,
                   })
               }
           }
           return nil
       })
       return results
   }
   ```
5. Add a CI check step that runs this lint against the generated template output:
   - `go test ./internal/templatecheck/... -run TestCalmDirectives`
   - Tests instantiate each template with mock data and runs the calm directive lint against the output.
6. Update Phase 4 CLAUDE.md template as the primary example:
   - Existing: "IMPORTANT: Always use the devenv shell for all commands"
   - Replacement: "Use the devenv shell for all commands — tools outside it may not match project versions"
   - Existing: "NEVER commit secrets to git"
   - Replacement: "Keep secrets out of git. Use `.env.local` (gitignored) or the project's secret manager"
   - Existing: "You MUST run tests before submitting changes"
   - Replacement: "Run tests before submitting changes to confirm the implementation is correct"
7. Write unit tests:
   - Audit on a template containing "CRITICAL:" returns a lint failure.
   - Audit on a template containing "NEVER" returns a lint failure.
   - Audit on a template using "YAML" does not return a lint failure (exempt acronym).
   - Audit on a template containing "URL" does not return a lint failure.
   - After the replacement pass, all production templates pass the calm directive lint.

**Acceptance Criteria:**
- [ ] All gdev-generated CLAUDE.md templates contain zero instances of: `CRITICAL:`, `IMPORTANT:`, `NEVER`, `MUST`, `DO NOT`, `!!!`
- [ ] No ALL-CAPS words in generated templates except the approved acronym list
- [ ] `checkCalmDirectives` lint function correctly identifies prohibited patterns
- [ ] Exempt acronyms (URL, API, SQL, JWT, SDK, CLI, CI, CD, MCP, YAML, JSON, HTTP, HTTPS, EOF, SBOM) are not flagged
- [ ] CI test runs `checkCalmDirectives` against rendered template output and fails on prohibited patterns
- [ ] Third-party skill files (CC-BY-4.0 from DrCatHicks) are NOT modified — they are exempt from this audit
- [ ] Phase 4 CLAUDE.md template updated to use calm positive directives throughout

**Research Citations:**
- `research-spikes/agentic-workflow-state-of-art/prompt-engineering-research.md § Negative Instructions` — Pink Elephant Problem, aggressive language degrading Claude 4.6 performance, reframing as positive directives
- `research-spikes/agentic-workflow-state-of-art/research.md § Aggressive and negative instructions hurt Claude 4.6` — research summary, four examples, calm direct statements outperform

**Status:** Not Started

---

### Unit 34.7: Time-to-First-Environment Benchmarking

**Description:** Instrument `gdev init` to measure and record timing at four checkpoints: detection, wizard, generation, and devenv build. Store results at `~/.qsdev/benchmarks/<project-hash>.jsonl`. Surface timing via `gdev info --timing` and `gdev status --verbose`. The "$8,000-$17,000 per new hire" ROI anchor from consulting ROI research becomes actionable when teams can measure their actual environment setup time before and after gdev adoption.

**Context:** The consulting tooling adoption ROI research established that environment setup time is the most mechanically actionable component of onboarding cost — it is front-loaded, blocking, and fully automatable. The research modeled conservative savings of $12,000-$70,000 per consultant per year, anchored on environment setup dropping from 2-5 days to under an hour. But these numbers are estimates. The only way to make an internal ROI case is with measured before/after data. Time-to-first-environment benchmarking is the instrument for collecting that data.

The four measurement points correspond to the distinct phases of `gdev init`: detection (language/service detection, mode selection), wizard (user interaction time, wizard form navigation), generation (template rendering, file writing), and devenv build (first `devenv shell` including Nix derivation evaluation and package fetching). Each phase has different characteristics: detection and generation are milliseconds, wizard is human-scale (seconds to minutes), devenv build is the longest (seconds to minutes depending on cache state).

Storing benchmarks per project-hash allows tracking improvement over time as templates improve, caches warm, and the devenv.nix becomes more accurate (reducing rebuild time from ecosystem changes).

**Desired Outcome:** After every `gdev init` run, a benchmark JSONL entry is written to `~/.qsdev/benchmarks/`. `gdev info --timing` shows the last init timing breakdown. After 3+ runs, `gdev info --timing` shows a trend line (improving/stable/degrading).

**Steps:**
1. Define the benchmark event type:
   ```go
   type BenchmarkEvent struct {
       SchemaVer   int    `json:"schema_ver"`  // 1
       Timestamp   string `json:"timestamp"`   // RFC3339
       ProjectHash string `json:"project_hash"` // sha256(project_root)[:16]
       GdevVersion string `json:"gdev_version"`
       Mode        string `json:"mode"`        // create/join/update/repair
       Phase       string `json:"phase"`       // detection/wizard/generation/devenv_build/total
       DurationMs  int64  `json:"duration_ms"`
       CacheHit    *bool  `json:"cache_hit,omitempty"` // for devenv_build phase only
       ProfileName string `json:"profile_name,omitempty"`
   }
   ```
2. Implement the timing instrumentation in `gdev init`:
   ```go
   type InitTimer struct {
       phases map[string]time.Time
       events []BenchmarkEvent
   }

   func (t *InitTimer) Start(phase string) {
       t.phases[phase] = time.Now()
   }

   func (t *InitTimer) End(phase string) {
       start, ok := t.phases[phase]
       if !ok {
           return
       }
       t.events = append(t.events, BenchmarkEvent{
           SchemaVer:  1,
           Timestamp:  time.Now().Format(time.RFC3339),
           Phase:      phase,
           DurationMs: time.Since(start).Milliseconds(),
       })
   }
   ```
3. Instrument the four checkpoints in `runInit`:
   ```go
   func runInit(cmd *cobra.Command, args []string) error {
       timer := &InitTimer{}

       timer.Start("detection")
       result, err := DetectOnboardingMode(projectRoot)
       timer.End("detection")
       // ...

       timer.Start("wizard")
       answers, err := RunWizard(result)
       timer.End("wizard")
       // ...

       timer.Start("generation")
       err = GenerateFiles(answers)
       timer.End("generation")
       // ...

       timer.Start("devenv_build")
       err = RunDevenvShellCheck()   // runs `devenv shell --command exit` to verify build
       timer.End("devenv_build")

       WriteBenchmark(projectRoot, timer.events)
       return nil
   }
   ```
4. Implement `WriteBenchmark`:
   - Write each event as a JSONL line to `~/.qsdev/benchmarks/<project-hash>.jsonl`.
   - Also write a `total` event with the sum of all phase durations.
   - Create the `~/.qsdev/benchmarks/` directory if it does not exist.
5. Implement `gdev info --timing`:
   - Read `~/.qsdev/benchmarks/<project-hash>.jsonl`.
   - Show last 3 runs with a phase breakdown table:
     ```
     Init Timing (last 3 runs):
     Phase           Last      Avg (3)   Trend
     Detection       45ms      42ms      stable
     Wizard          2m13s     1m52s     stable
     Generation      312ms     298ms     improving
     devenv build    4m22s     5m01s     improving
     Total           6m52s     7m13s     improving
     ```
   - Trend: improving (>10% faster), degrading (>10% slower), stable.
6. Implement `gdev status --verbose` timing summary:
   - One-line summary in `gdev status --verbose`: `Last init: 6m52s (devenv 4m22s, cached)`.
7. Implement `gdev benchmarks` subcommand:
   - `gdev benchmarks`: shows timing history for the current project (all runs in the JSONL).
   - `gdev benchmarks --all`: shows aggregate timing across all projects.
   - `gdev benchmarks export`: outputs the raw JSONL for sharing with the team.
8. Write unit tests:
   - `InitTimer.Start/End` records duration correctly.
   - `WriteBenchmark` creates JSONL file with correct event structure.
   - Trend calculation: three improving runs returns "improving".
   - Trend calculation: mixed runs returns "stable".
   - `gdev benchmarks export` outputs valid JSONL.

**Acceptance Criteria:**
- [ ] `gdev init` records timing events for detection, wizard, generation, and devenv_build phases
- [ ] Benchmark JSONL written to `~/.qsdev/benchmarks/<project-hash>.jsonl` after every `gdev init`
- [ ] Each event includes: timestamp, project hash, gdev version, mode, phase, duration in milliseconds
- [ ] `gdev info --timing` shows last 3 runs with phase breakdown and trend (improving/stable/degrading)
- [ ] `gdev status --verbose` includes a one-line timing summary
- [ ] Trend calculation: improving = >10% faster than 3-run average, degrading = >10% slower
- [ ] `gdev benchmarks` shows full timing history for current project
- [ ] `gdev benchmarks export` outputs the raw JSONL

**Research Citations:**
- `research-spikes/consulting-tooling-adoption-roi/onboarding-costs-research.md` — "$8,000-$17,000 per new hire" environment setup cost, 2-5 days manual vs under 1 hour automated, Spotify/Shopify case studies
- `research-spikes/consulting-tooling-adoption-roi/research.md § Developer Onboarding Costs` — environment setup as most mechanically actionable component, before/after measurement as ROI evidence
- `research-spikes/consulting-tooling-adoption-roi/roi-framework-synthesis.md` — need for internal measurement to upgrade framework confidence

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Skill Deployment Infrastructure

All four skills in this phase (learning-opportunities, orient, repo-map, and any gdev-authored skills) use the existing `deploySkills()` function at `addons/claudecode/generate_skills.go:42-78`. The function already handles:
- Recursive directory copying from embedded filesystem
- File permission preservation
- Writing to `.claude/skills/<name>/`

New skills are added by placing files in `internal/claudecode/skills/<name>/` and adding entries to `templates/skills/manifest.yaml`.

### Template Audit Scope

Unit 34.6 audit scope:

| Template Location | Status |
|------------------|--------|
| `templates/claudecode/CLAUDE.md.tmpl` | Audit required |
| `templates/claudecode/rules/*.tmpl` | Audit required |
| `internal/claudecode/skills/*.md` (gdev-authored) | Audit required |
| `templates/hooks/*.sh.tmpl` (user-facing strings) | Audit required |
| `internal/claudecode/skills/learning-opportunities/` | Exempt (CC-BY-4.0) |
| `internal/claudecode/skills/orient/` | Exempt (CC-BY-4.0) |

### Benchmarking Data Privacy

The `BenchmarkEvent` schema contains only timing metrics and project hash — no file contents, no wizard answers, no project names. The same 256-character field limit used in Unit 33.4's analytics engine applies here: `ProfileName` and `GdevVersion` are the only string fields, and both are well under 256 characters.

### Hook Registration Pattern

Unit 34.5's PreToolUse hook follows the same registration pattern as Phase 22's compliance validation hooks: deploy the script to `.claude/hooks/`, register in `settings.json` using Phase 14's shared-file surgery for the `hooks` section.

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] `gdev enable learning-opportunities` deploys correctly with CC-BY-4.0 attribution
- [ ] `gdev enable orient` deploys correctly with `disable-model-invocation: true` preserved
- [ ] Project Context section appears in generated CLAUDE.md with consulting block conditionally included
- [ ] `gdev enable repo-map` deploys SKILL.md and executable repo-map.sh
- [ ] Pre-edit validation hook runs in under 200ms on a representative TypeScript file with ESLint available
- [ ] All production templates pass the `checkCalmDirectives` lint with zero prohibited patterns
- [ ] `gdev info --timing` shows a phase breakdown after running `gdev init` in Create mode
- [ ] Generated CLAUDE.md contains zero instances of `CRITICAL:`, `IMPORTANT:`, `NEVER`, `MUST`, `DO NOT`, or `!!!`
- [ ] Third-party CC-BY-4.0 skill files (learning-opportunities, orient) are deployed unmodified except for the attribution header comment
