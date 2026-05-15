# Phase 4: Claude Code Addon — Core Generation

## Goal

Implement the claudecode addon's file generation: settings.json with permission presets and deny rules sourced from ecosystem modules, CLAUDE.md with section markers, hook deployment integrating existing ecosystem tools (attach-guard, Trail of Bits config), curated skill/rule library, and .mcp.json generation. Rather than building security enforcement from scratch, this phase curates and deploys the best existing tools.

## Dependencies

Phases 1 and 2 complete (shared types, template engine, generation pipeline, ecosystem modules providing deny rules).

## Phase Outputs

- settings.json generation via struct marshaling with 3 permission presets and deny rules aggregated from all detected ecosystem modules (48+ base rules, extensible per ecosystem)
- CLAUDE.md generation via text/template with `<!-- BEGIN/END GENERATED SECTION -->` markers
- Hook deployment: attach-guard plugin configuration + custom PreToolUse hook script (Python, with npm age check and OSV version query fixes)
- Curated skill library: Trail of Bits security skills (`supply-chain-risk-auditor`, `differential-review`, `insecure-defaults`) + custom team skills
- Curated rule library: language-specific convention rules + `security-rules.md` with package installation policies
- .mcp.json generation with Socket.dev MCP, plus optional SonarQube/Aikido marketplace plugins
- Managed settings template for enterprise deployment (`/etc/claude-code/managed-settings.json`)
- Five CLI commands: `qsdev claude init`, `update`, `add-skill`, `add-hook`, `list-skills`
- Embedded skill library with manifest and 6+ standard skills
- Embedded rule library with language-specific convention files
- .mcp.json generation via struct marshaling
- Five CLI commands: `qsdev claude init`, `update`, `add-skill`, `add-hook`, `list-skills`

---

### Unit 3.1: settings.json Generation with Permission Presets & Deny Rules

**Description:** Implement struct-marshaled settings.json generation with three permission presets (minimal, standard, permissive) and the complete deny rule set from the guardrails spike.

**Context:** settings.json is machine-owned-with-additions (three-way merge on update). Uses Go struct marshaling for guaranteed valid JSON. The deny rules cover 15+ package managers with 48 patterns. Permission presets define allow/deny/ask rule sets appropriate for different security postures. Validation confirmed all glob patterns are correct for current CLIs.

**Desired Outcome:** `SettingsGenerator.Generate(answers)` produces valid settings.json with appropriate permission preset and deny rules.

**Steps:**
1. Define `SettingsJSON` struct matching current Claude Code schema: `Permissions`, `Sandbox`, `Hooks` map.
2. Implement three permission presets with concrete rule lists from claude-code-addon-design.md.
3. Integrate deny rules from `reference-deny-rules.md` — use individual entries, not `||`-compound `if` syntax.
4. Support `ExtraAllowPatterns` and `ExtraDenyPatterns` from wizard custom mode.
5. Implement sandbox config when enabled: `WriteDeny`, `WriteAllow`, `ReadDeny`, `NetAllow`.
6. Marshal via `json.MarshalIndent()`, wrap in `GeneratedFile` with `ThreeWayMerge` strategy.
7. Write unit tests for each preset, verify JSON validity, verify deny rules count.

**Acceptance Criteria:**
- [ ] Generated JSON is valid (round-trip test)
- [ ] Minimal preset: read-only + explicit build commands only
- [ ] Standard preset: build/test/lint/git + deny dangerous operations
- [ ] Permissive preset: broad access + deny destructive commands
- [ ] All 48+ deny rules present covering npm, yarn, pnpm, bun, pip, uv, cargo, go, nix, curl|bash
- [ ] Individual hook entries used (no `||` compound `if` syntax)
- [ ] Custom mode unions extra patterns with preset base

**Research Citations:**
- `research-spikes/claude-code-agent-package-guardrails/reference-deny-rules.md` — 48 deny rules covering 15+ package managers
- `research-spikes/claude-code-agent-package-guardrails/unified-architecture.md § Section 3.1` — deny rule architecture
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § Permission Presets` — preset definitions
- Validation: glob patterns confirmed correct; individual entries recommended over `||` syntax

**Status:** Not Started

---

### Unit 3.2: CLAUDE.md Generation with Section Markers

**Description:** Implement CLAUDE.md generation via text/template with `<!-- BEGIN GENERATED SECTION -->` / `<!-- END GENERATED SECTION -->` markers separating generated and user-editable content.

**Context:** CLAUDE.md is human-edited — users add custom instructions below the generated section. Section markers enable safe updates: generated content between markers is replaced, user content outside is preserved. Template is conditional on detected languages, frameworks, and project type.

**Desired Outcome:** Template renders project-appropriate CLAUDE.md with security instructions, build commands, and language conventions.

**Steps:**
1. Create `addons/claudecode/templates/claude-md.tmpl` — main template.
2. Create language-specific convention sub-templates: `conventions/go.md.tmpl`, `typescript.md.tmpl`, `python.md.tmpl`, `rust.md.tmpl`.
3. Include security section in generated content: package installation rules (use devenv, don't install via raw commands), secret handling rules, testing requirements.
4. Include build/test/lint commands from wizard answers.
5. Include architecture notes and project description.
6. Wrap generated content in `<!-- BEGIN GENERATED SECTION — do not edit between markers -->` / `<!-- END GENERATED SECTION -->`.
7. Add empty `## Custom Instructions` section below markers.
8. Implement `ClaudeMdTemplateData` struct as bridge from `WizardAnswers`.

**Acceptance Criteria:**
- [ ] Generated CLAUDE.md has correct section markers
- [ ] Language-specific conventions included for detected languages
- [ ] Security instructions present (package installation rules, secret handling)
- [ ] Build/test/lint commands included when provided
- [ ] Content between markers is complete and useful
- [ ] Content outside markers is preserved on re-generation (tested in Phase 6)

**Research Citations:**
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § CLAUDE.md Template` — template structure
- `research-spikes/gdev-extension-design/migration-strategy-design.md § CLAUDE.md Section Markers` — marker-based merge
- `research-spikes/claude-code-agent-package-guardrails/claude-md-guardrails-research.md` — instruction effectiveness

**Status:** Not Started

---

### Unit 3.3: PreToolUse Hook Script Deployment

**Description:** Adapt and deploy the Python PreToolUse hook script from the guardrails spike as a project-level hook for package installation safety.

**Context:** The hook fires before every Bash command, checks package install commands against OSV.dev for vulnerabilities and registry APIs for publication age, and uses `updatedInput` to inject safety flags (e.g., `--ignore-scripts`). Validation confirmed the Python script is correct but needs fixes for npm age checking (use version creation time) and OSV version queries.

**Desired Outcome:** A deployable hook script that integrates with settings.json hooks configuration and provides real-time package security enforcement.

**Steps:**
1. Copy `reference-hook-script.py` to `addons/claudecode/templates/hooks/` as embedded resource.
2. Fix npm age check: use `time[dist-tags.latest]` instead of `time.modified`.
3. Fix OSV queries: when version is available from the command, include it in the query.
4. Make configurable: `FAIL_CLOSED`, `MIN_AGE_DAYS`, `ALLOWLIST` as environment variables or config.
5. Deploy to `.claude/hooks/package-guard.py` during generation.
6. Wire into settings.json hooks config: PreToolUse entry pointing to the script.
7. Test: mock OSV.dev responses, verify allow/deny decisions, verify `updatedInput` flag injection.

**Acceptance Criteria:**
- [ ] Hook script deploys to `.claude/hooks/package-guard.py`
- [ ] settings.json hooks section references the deployed script
- [ ] npm age check uses version creation time, not metadata modification time
- [ ] OSV queries include version when parseable from command
- [ ] `FAIL_CLOSED`, `MIN_AGE_DAYS`, `ALLOWLIST` are configurable via environment
- [ ] Script has no external dependencies (stdlib-only Python 3)
- [ ] Hook correctly injects `--ignore-scripts` via `updatedInput` for npm/yarn/pnpm

**Research Citations:**
- `research-spikes/claude-code-agent-package-guardrails/reference-hook-script.py` — base implementation
- `research-spikes/claude-code-agent-package-guardrails/hooks-research.md` — PreToolUse mechanism, updatedInput
- Validation: npm `modified` field imprecise, OSV versionless queries have false positives — both fixed

**Status:** Not Started

---

### Unit 3.4: Embedded Skill & Rule Library

**Description:** Create the embedded skill and rule library with standard skills and language-specific convention rules, deployed via `embed.FS` copy.

**Context:** Skills and rules are copied verbatim (not templated) from the embedded library. Each skill is a standalone markdown file in `.claude/skills/`. Each rule is a convention file in `.claude/rules/`. The manifest tracks available skills with metadata for the wizard's multi-select.

**Desired Outcome:** A curated library of skills and rules that deploy correctly and provide immediate value.

**Steps:**
1. Create `addons/claudecode/templates/skills/` with standard skills: `deploy.md`, `review-pr.md`, `security-review.md`, `generate-tests.md`, `refactor.md`, `db-migration.md`.
2. Create `addons/claudecode/templates/rules/` with convention files: `go-conventions.md`, `typescript-conventions.md`, `python-conventions.md`, `security-rules.md`.
3. Create `addons/claudecode/templates/skills/manifest.yaml` with skill metadata (name, description, tags, applicable languages).
4. Implement `deploySkills(answers WizardAnswers, projectRoot string) []GeneratedFile` — copy selected skills from embed.FS.
5. Implement `deployRules(answers WizardAnswers, projectRoot string) []GeneratedFile` — copy language-matched rules.
6. Security rules should include: no raw package installation, secret handling, dependency update procedures.

**Acceptance Criteria:**
- [ ] manifest.yaml is valid and lists all embedded skills
- [ ] Skills deploy to `.claude/skills/<name>.md`
- [ ] Rules deploy to `.claude/rules/<name>.md`
- [ ] Only wizard-selected skills are deployed
- [ ] Language-matched rules auto-deploy based on detected languages
- [ ] Security rules are always deployed regardless of language
- [ ] Files are plain copies (no template processing)

**Research Citations:**
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § Skill Library` — manifest format, two-tier system
- `research-spikes/gdev-extension-design/config-template-engine-design.md § embed.FS copy` — verbatim copy strategy

**Status:** Not Started

---

### Unit 3.5: .mcp.json Generation

**Description:** Implement .mcp.json generation via struct marshaling for MCP server configurations.

**Context:** .mcp.json configures Model Context Protocol servers that extend Claude Code's capabilities. The wizard offers common servers (GitHub, filesystem, database). The security integration (Phase 4) adds Socket.dev MCP for supply chain analysis.

**Desired Outcome:** `McpJsonGenerator.Generate(answers)` produces valid .mcp.json with selected servers.

**Steps:**
1. Define `McpJSON` struct: `MCPServers map[string]MCPServerEntry` with `Command`, `Args`, `Env`.
2. Implement server templates for common integrations: GitHub (`gh`-based), filesystem, PostgreSQL.
3. Marshal via `json.MarshalIndent()`, wrap in `GeneratedFile` with `ThreeWayMerge` strategy.
4. Include placeholder for Socket.dev server (actual integration in Phase 4).

**Acceptance Criteria:**
- [ ] Generated JSON is valid (round-trip test)
- [ ] GitHub MCP server configured correctly when selected
- [ ] Multiple servers compose correctly
- [ ] Merge strategy set to ThreeWayMerge (preserves user-added servers on update)

**Research Citations:**
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § MCP Servers` — server configuration
- `research-spikes/claude-code-agent-package-guardrails/mcp-server-research.md` — Socket.dev/Snyk MCP

**Status:** Not Started

---

### Unit 3.6: Claude Code CLI Commands

**Description:** Register the five claudecode CLI commands: `qsdev claude init`, `update`, `add-skill`, `add-hook`, `list-skills`.

**Context:** Mirrors the devenv command structure for standalone use. The `init` command runs the wizard or accepts flags. `add-skill` and `list-skills` interact with the embedded + remote skill library.

**Desired Outcome:** All five commands registered, callable, producing correct output.

**Steps:**
1. Create `addons/claudecode/commands.go` — register parent `claude` command and five sub-commands.
2. `qsdev claude init` — run wizard or accept flags, generate all files.
3. `qsdev claude update` — load saved config, regenerate.
4. `qsdev claude add-skill <name>` — add skill from library, deploy to `.claude/skills/`.
5. `qsdev claude add-hook <type>` — add hook preset (auto-format, safety-block, pre-commit, audit-log).
6. `qsdev claude list-skills` — list available skills with installed status.
7. All commands: support `--yes`, `--force`, `--dry-run`.

**Acceptance Criteria:**
- [ ] `qsdev claude init --permission-preset standard --yes` generates files without wizard
- [ ] `qsdev claude update` regenerates from saved config
- [ ] `qsdev claude add-skill deploy` deploys skill file
- [ ] `qsdev claude add-hook safety-block` adds hook to settings.json
- [ ] `qsdev claude list-skills` shows available and installed skills
- [ ] All commands support `--dry-run`

**Research Citations:**
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § Commands` — five command definitions
- `research-spikes/gdev-extension-design/addon-architecture-design.md § Command Hierarchy` — `qsdev claude *` structure

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] `qsdev claude init --permission-preset standard --skills deploy,security-review --yes` produces valid settings.json + CLAUDE.md + skills + hook
- [ ] Generated settings.json contains appropriate deny rules for detected package managers
- [ ] Generated CLAUDE.md has section markers and language-specific conventions
- [ ] PreToolUse hook script is deployed and referenced in settings.json
- [ ] Generated .mcp.json is valid when MCP servers selected
- [ ] All generated JSON passes `json.Unmarshal` validation
