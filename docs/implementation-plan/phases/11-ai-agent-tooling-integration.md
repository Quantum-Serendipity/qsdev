# Phase 11: AI Agent Tooling Integration

## Goal

Integrate three AI agent enhancement tools — agent-postmortem-skill (task verification), Version-Sentinel (dependency version guardrails), and semble (semantic code search) — into the gdev addon ecosystem. The Claude Code addon generates configuration that deploys, configures, and optionally customizes these tools per-project. The wizard exposes opt-in/opt-out for each tool. At the end of this phase, `qsdev init` can deploy a complete AI agent toolkit alongside the security hardening from earlier phases.

## Dependencies

Phase 4 complete (Claude Code addon core generation — settings.json, CLAUDE.md, hook deployment, skill library). Phase 6 partial (wizard infrastructure — tool selection UI). Phase 2 desirable (ecosystem modules — per-ecosystem verification commands and manifest coverage).

## Phase Outputs

- Agent-postmortem-skill deployment: templated SKILL.md with per-ecosystem verification commands
- Version-Sentinel deployment: plugin installation instructions or manual hook wiring, configuration generation
- Semble deployment: MCP server configuration and/or sub-agent file generation
- Wizard integration: AI agent tooling form group with per-tool enable/disable and configuration
- Per-ecosystem verification command registry (feeding postmortem skill customization)
- Per-ecosystem manifest coverage map (feeding Version-Sentinel awareness)

---

### Unit 11.1: Agent Postmortem Skill Integration

**Description:** Embed and deploy the agent-postmortem-skill SKILL.md as part of Claude Code addon generation, with project-type-aware verification command customization.

**Context:** The agent-postmortem-skill is a prompt-based verification protocol (3.6KB SKILL.md) that prevents AI agents from claiming "done" without evidence. It's MIT-licensed and integrates by placing `SKILL.md` in `.claude/skills/agent-postmortem/`. The base skill is static, but gdev adds value by injecting project-specific verification commands based on detected ecosystems. A Go project gets `go build ./...` and `go test ./...`; a Node project gets `npm test` and `npm run lint`; a Rust project gets `cargo build` and `cargo test`. This makes the postmortem skill immediately useful without manual configuration.

**Desired Outcome:** `qsdev init` generates a customized agent-postmortem SKILL.md with verification commands matching the detected project type.

**Steps:**
1. Embed the base agent-postmortem SKILL.md in `addons/claudecode/skills/agent-postmortem/SKILL.md` via `embed.FS`.
2. Create `internal/agenttools/postmortem.go` with `GeneratePostmortemSkill(ecosystems []EcosystemModule, config ModuleConfig) GeneratedFile`.
3. Define per-ecosystem verification command registry:
   ```go
   type VerificationCommands struct {
       Build    []string // e.g., "go build ./...", "cargo build"
       Test     []string // e.g., "go test ./...", "npm test"
       Lint     []string // e.g., "golangci-lint run", "eslint ."
       TypeCheck []string // e.g., "tsc --noEmit", "mypy ."
       Format   []string // e.g., "gofmt -l .", "prettier --check ."
   }
   ```
4. Add `VerificationCommands() VerificationCommands` method to the `EcosystemModule` interface (or define a supplementary interface `VerifiableModule`).
5. Implement verification commands for Tier 1 ecosystems:
   - Go: `go build ./...`, `go test ./...`, `go vet ./...`, `golangci-lint run`
   - JS/TS (npm): `npm test`, `npm run lint`, `npx tsc --noEmit` (if tsconfig.json exists)
   - JS/TS (pnpm): `pnpm test`, `pnpm lint`, `pnpm exec tsc --noEmit`
   - Python: `pytest`, `mypy .`, `ruff check .`
   - Rust: `cargo build`, `cargo test`, `cargo clippy`
   - Java (Maven): `mvn compile`, `mvn test`
   - Java (Gradle): `gradle build`, `gradle test`
   - .NET: `dotnet build`, `dotnet test`
   - Docker: `hadolint Dockerfile`, `docker build .`
   - Terraform: `terraform validate`, `terraform plan`
6. Template the SKILL.md to inject detected verification commands into the "Required verification commands" section. Keep the base skill structure unchanged — only customize the verification command list.
7. Generate the file at `.claude/skills/agent-postmortem/SKILL.md` during `qsdev init`.
8. Support `MergeStrategy: Overwrite` for machine-managed file (regenerated on update).
9. Write unit tests verifying template output for Go, Node, and multi-ecosystem projects.

**Acceptance Criteria:**
- [ ] Base SKILL.md content preserved (all 4 steps, anti-fake-done guardrails, verdict rules)
- [ ] Go project gets `go build ./...`, `go test ./...`, `go vet ./...` as required verification commands
- [ ] Node project gets `npm test` (or pnpm/yarn equivalent) as required verification commands
- [ ] Multi-ecosystem project (Go + TypeScript) gets combined verification commands
- [ ] File generated at `.claude/skills/agent-postmortem/SKILL.md`
- [ ] Merge strategy is Overwrite (machine-owned, safe to regenerate)
- [ ] Unit tests pass for all Tier 1 ecosystems

**Research Citations:**
- `artifacts/agent-postmortem-skill-SKILL.md` — complete skill content (embed source)
- `artifacts/agent-postmortem-skill-README.md` — integration instructions, installation path
- `artifacts/agent-postmortem-skill-example-postmortem.md` — expected output format

**Status:** Not Started

---

### Unit 11.2: Version-Sentinel Integration

**Description:** Generate configuration to install and configure the Version-Sentinel Claude Code plugin, which hard-blocks dependency additions and version changes until versions are verified against upstream registries.

**Context:** Version-Sentinel is a Claude Code plugin (MIT, v0.2.1) that uses PreToolUse hooks to intercept manifest edits and package install commands, blocking them if no fresh version verification exists. It supports npm, pip, pyproject.toml, Cargo.toml, and .csproj — covering 5 of gdev's 8 Tier 1 ecosystems. The plugin installs via the Claude Code plugin marketplace. gdev's integration generates the install command, configures `window_hours` and the ignore file, and ensures prerequisites (bash, jq, curl, python3 >=3.11) are present.

**Desired Outcome:** `qsdev init` with Version-Sentinel enabled generates plugin installation instructions and per-project configuration.

**Steps:**
1. Create `internal/agenttools/versionsentinel.go` with `GenerateVersionSentinelConfig(ecosystems []EcosystemModule, config VSConfig) []GeneratedFile`.
2. Define `VSConfig` struct:
   ```go
   type VSConfig struct {
       Enabled      bool
       WindowHours  int      // default: 24
       IgnoredPkgs  []string // ecosystem:pkg format for private/forked packages
   }
   ```
3. Generate `.version-sentinel/ignore` file with project-specific ignored packages:
   - Auto-populate with private registry packages detected from ecosystem configs (e.g., `@company/` scoped npm packages)
   - Support user-provided ignore list from wizard or config
4. Generate a setup script or CLAUDE.md section with the plugin install command:
   ```
   claude plugin marketplace add https://github.com/KSEGIT/Version-Sentinel.git
   claude plugin install version-sentinel@version-sentinel-marketplace
   ```
5. Check Version-Sentinel prerequisite availability (jq, curl, python3 >=3.11) during `qsdev devenv doctor` — add these to the prerequisite checks in Phase 9.
6. Generate ecosystem-specific coverage notes in CLAUDE.md:
   - Covered: npm (package.json), pip (requirements.txt), pyproject.toml, Cargo.toml, .csproj/.fsproj/.vbproj
   - Not yet covered by Version-Sentinel: Go (go.mod), Maven (pom.xml), Gradle (build.gradle), Docker, Terraform, PHP (composer.json), Ruby (Gemfile)
   - For uncovered ecosystems: note that manual version verification is needed
7. Support `qsdev init --version-sentinel=false` to skip.
8. Add VS_WINDOW_HOURS to generated environment or plugin config when non-default.
9. Handle interaction with attach-guard and other PreToolUse hooks — Version-Sentinel uses different matchers (Edit|Write|MultiEdit for manifests, Bash for install commands) so hooks don't conflict.
10. Write unit tests verifying generated ignore file and CLAUDE.md sections.

**Acceptance Criteria:**
- [ ] Plugin installation command generated correctly
- [ ] `.version-sentinel/ignore` file generated with private package patterns
- [ ] CLAUDE.md includes Version-Sentinel coverage notes per detected ecosystem
- [ ] Prerequisites (jq, curl, python3) flagged in `qsdev devenv doctor` when Version-Sentinel enabled
- [ ] Non-default `window_hours` reflected in configuration
- [ ] No hook conflicts with attach-guard or other PreToolUse hooks
- [ ] Skip flag (`--version-sentinel=false`) works
- [ ] Unit tests pass

**Research Citations:**
- `artifacts/version-sentinel-plugin-json.json` — plugin manifest with userConfig schema
- `artifacts/version-sentinel-hooks-json.json` — hook definitions (events, matchers, commands)
- `artifacts/version-sentinel-readme.md` — installation instructions, configuration options
- `artifacts/version-sentinel-skill.md` — recovery workflow skill
- `artifacts/version-sentinel-changelog.md` — known limitations, ecosystem coverage gaps

**Status:** Not Started

---

### Unit 11.3: Semble Code Search Integration

**Description:** Generate MCP server configuration and/or Claude Code sub-agent file for semble, enabling AI agents to perform semantic code search with ~98% token savings.

**Context:** Semble is a code search library for AI agents (MIT, v0.1.7, 798 stars) that uses tree-sitter AST chunking + hybrid semantic/BM25 search. It runs as an MCP server or Claude Code sub-agent. Integration is lightweight: either add an MCP server config entry or write a `.claude/agents/semble-search.md` file. Semble requires Python >=3.10 and `uvx` (or pip) for installation. The MCP server auto-indexes the project directory and watches for file changes.

**Desired Outcome:** `qsdev init` with semble enabled configures either MCP server mode or sub-agent mode for semantic code search.

**Steps:**
1. Create `internal/agenttools/semble.go` with `GenerateSembleConfig(mode SembleMode, projectRoot string) []GeneratedFile`.
2. Define `SembleMode` enum: `MCP`, `SubAgent`, `Both`, `Disabled`.
3. **MCP mode**: Generate `.mcp.json` entry (or merge into existing `.mcp.json`):
   ```json
   {
     "mcpServers": {
       "semble": {
         "command": "uvx",
         "args": ["--from", "semble[mcp]", "semble"]
       }
     }
   }
   ```
4. **Sub-agent mode**: Generate `.claude/agents/semble-search.md` — embed the standard agent definition from semble's repo (instructs Claude to use `semble search` and `semble find-related` CLI commands).
5. Add `--include-text-files` flag to MCP args when project contains significant non-code files (infrastructure repos with YAML/Markdown, detected from ecosystem modules: Terraform, Helm, Ansible).
6. Check semble prerequisite: `python3 --version` >= 3.10 and `uvx` available. Add to `qsdev devenv doctor` checks when semble is enabled.
7. Support pre-indexing by passing project path as argument to MCP server:
   ```json
   "args": ["--from", "semble[mcp]", "semble", "/path/to/project"]
   ```
8. Handle `.mcp.json` merge strategy — if file exists, merge the `semble` entry into existing servers rather than overwriting. Use `MergeStrategy: ThreeWayMerge` for .mcp.json.
9. Support `qsdev init --semble=false` and `qsdev init --semble-mode=mcp|subagent|both` flags.
10. Write unit tests verifying .mcp.json generation and merge for both clean and existing configs.

**Acceptance Criteria:**
- [ ] MCP mode generates correct `.mcp.json` entry
- [ ] Sub-agent mode generates `.claude/agents/semble-search.md`
- [ ] `--include-text-files` auto-enabled for infrastructure-heavy projects
- [ ] `.mcp.json` merge preserves existing MCP server entries
- [ ] Python/uvx prerequisite check integrated with `qsdev devenv doctor`
- [ ] Mode selection works via flag and wizard
- [ ] Skip flag (`--semble=false`) works
- [ ] Unit tests pass for clean and merge scenarios

**Research Citations:**
- `artifacts/semble-mcp-configuration-examples.md` — MCP config for Claude Code, Cursor, Codex
- `artifacts/semble-readme-and-metadata.md` — installation, CLI usage, sub-agent init
- `artifacts/semble-mcp-server.md` — MCP server implementation details
- `artifacts/semble-cli.md` — CLI entry point and `init` command

**Status:** Not Started

---

### Unit 11.4: Wizard Integration — AI Agent Tooling Form Group

**Description:** Add an "AI Agent Tools" form group to the huh wizard that lets developers enable/disable and configure agent-postmortem-skill, Version-Sentinel, and semble.

**Context:** The wizard's Group 5 (Claude Code) from Phase 6 should be extended or a new Group inserted for AI agent tooling. These tools are opt-in with sensible defaults: agent-postmortem enabled by default (zero-cost prompt addition), Version-Sentinel enabled by default when supported ecosystems detected, semble enabled by default in MCP mode. The form should explain what each tool does in a sentence and allow granular configuration.

**Desired Outcome:** The wizard includes AI agent tool selection with smart defaults based on detected ecosystems.

**Steps:**
1. Extend `WizardAnswers` struct (Phase 1, Unit 1.2) with:
   ```go
   type AgentToolsAnswers struct {
       PostmortemEnabled     bool
       VersionSentinel       bool
       VersionSentinelHours  int
       SembleEnabled         bool
       SembleMode            string // "mcp", "subagent", "both"
   }
   ```
2. Add wizard form group (inserted as Group 5b or merged into existing Claude Code group):
   - **Agent Postmortem**: toggle (default: on) — "Require evidence-backed verification before claiming tasks done"
   - **Version Sentinel**: toggle (default: on when npm/pip/cargo/dotnet detected) — "Block dependency changes until versions verified against registry"
   - Version Sentinel window hours: number input (default: 24, shown when VS enabled)
   - **Semble Code Search**: toggle (default: on) — "Semantic code search for AI agents (98% fewer tokens)"
   - Semble mode: select MCP/Sub-agent/Both (default: MCP, shown when semble enabled)
3. Smart defaults based on detection:
   - Version-Sentinel defaults to on only when at least one supported ecosystem detected (JS/TS, Python, Rust, .NET)
   - Semble defaults to on only when Python >=3.10 detected on system
   - Agent-postmortem always defaults to on (no dependencies)
4. Quick path: all three use defaults (skip form group).
5. Wire answers into generation: `AgentToolsAnswers` flows to Units 11.1-11.3 generators.
6. Non-interactive flags: `--postmortem`, `--version-sentinel`, `--semble`, `--semble-mode`.

**Acceptance Criteria:**
- [ ] Wizard shows AI agent tools form group in customize path
- [ ] Quick path uses smart defaults based on detection
- [ ] Version-Sentinel defaults to off when no supported ecosystem detected
- [ ] Semble defaults to off when Python <3.10 or missing
- [ ] Agent-postmortem always defaults to on
- [ ] Non-interactive flags override wizard choices
- [ ] Form group hidden when `ClaudeCode = false` in wizard
- [ ] All fields persist through `WizardAnswers` to generators

**Research Citations:**
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § huh Form Construction` — wizard form patterns
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md § Progressive Disclosure` — WithHideFunc mechanics
- `artifacts/agent-postmortem-skill-README.md` — zero-dependency, prompt-only integration
- `artifacts/version-sentinel-plugin-json.json` — userConfig (disable, window_hours)
- `artifacts/semble-readme-and-metadata.md` — Python >=3.10 requirement

**Status:** Not Started

---

### Unit 11.5: Per-Ecosystem Verification & Coverage Registries

**Description:** Create the registries that map each ecosystem module to its verification commands (for postmortem) and manifest file coverage (for Version-Sentinel), enabling accurate cross-tool configuration.

**Context:** The agent-postmortem skill needs to know which build/test/lint commands to include per project type. Version-Sentinel needs to know which manifest files it can guard and which it can't. These registries are maintained alongside ecosystem modules and queried during generation. This unit formalizes the data that Units 11.1 and 11.2 consume.

**Desired Outcome:** Structured registries that ecosystem modules populate, queried at generation time to produce accurate per-project configurations for both tools.

**Steps:**
1. Extend `EcosystemModule` interface with:
   ```go
   VerificationCommands() VerificationCommands
   ManifestFiles() []ManifestFileInfo
   ```
2. Define `ManifestFileInfo`:
   ```go
   type ManifestFileInfo struct {
       Path           string // e.g., "package.json", "go.mod", "Cargo.toml"
       Ecosystem      string // e.g., "npm", "go", "cargo"
       VSSupported    bool   // true if Version-Sentinel can guard this file
       LockFile       string // corresponding lock file (e.g., "package-lock.json")
       LockFilePolicy string // "required", "recommended", "none"
   }
   ```
3. Implement for all Tier 1 ecosystem modules:
   - **JS/TS**: package.json (VS: ✓), package-lock.json/pnpm-lock.yaml/yarn.lock/bun.lockb
   - **Python**: requirements.txt (VS: ✓), pyproject.toml (VS: ✓), poetry.lock/uv.lock
   - **Go**: go.mod (VS: ✗), go.sum
   - **Rust**: Cargo.toml (VS: ✓), Cargo.lock
   - **Java Maven**: pom.xml (VS: ✗)
   - **Java Gradle**: build.gradle/build.gradle.kts (VS: ✗)
   - **.NET**: *.csproj/*.fsproj/*.vbproj (VS: ✓)
   - **Docker**: Dockerfile (VS: N/A — not a dependency manifest)
   - **Terraform**: *.tf (VS: ✗), .terraform.lock.hcl
4. Implement `AggregateVerificationCommands(modules []EcosystemModule) VerificationCommands` — merge commands from all detected ecosystems, dedup, sort by priority.
5. Implement `AggregateManifestCoverage(modules []EcosystemModule) ManifestCoverageReport` — summary of which manifests are VS-covered vs uncovered.
6. Write unit tests verifying aggregation for multi-ecosystem projects.

**Acceptance Criteria:**
- [ ] All Tier 1 modules implement `VerificationCommands()` and `ManifestFiles()`
- [ ] Aggregation correctly merges commands from Go + TypeScript project
- [ ] Manifest coverage report accurately reflects Version-Sentinel's current ecosystem support
- [ ] VS-unsupported manifests (go.mod, pom.xml, build.gradle) flagged clearly
- [ ] Lock file policy documented per ecosystem
- [ ] Unit tests verify single-ecosystem and multi-ecosystem aggregation

**Research Citations:**
- `artifacts/agent-postmortem-skill-SKILL.md § Step 2 - Evidence Collection` — required verification artifacts
- `artifacts/version-sentinel-changelog.md` — ecosystem coverage (npm, pip, pyproject, cargo, csproj)
- `artifacts/language-ecosystem-coverage.md` — all 27 ecosystems with manifest/lock file details
- `research-spikes/package-supply-chain-security/lockfile-integrity-research.md` — lock file enforcement per ecosystem

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] `qsdev init` in a Go project generates agent-postmortem skill with Go verification commands
- [ ] `qsdev init` in a Node project generates Version-Sentinel config with npm support noted
- [ ] `qsdev init` generates semble MCP config that Claude Code can connect to
- [ ] Wizard shows AI agent tools with correct smart defaults per detected project
- [ ] `qsdev init --yes` deploys all three tools with defaults on a supported project
- [ ] `qsdev init --postmortem=false --version-sentinel=false --semble=false` skips all three
- [ ] Multi-ecosystem project (Go + TypeScript + Rust) gets combined verification commands and accurate VS coverage report
- [ ] No hook conflicts between Version-Sentinel, attach-guard, and other PreToolUse hooks
- [ ] Generated CLAUDE.md sections document which AI agent tools are active and their coverage
