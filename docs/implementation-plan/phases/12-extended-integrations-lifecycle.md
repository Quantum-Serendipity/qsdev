# Phase 12: Extended Integrations & Tool Lifecycle Management

## Goal

Implement a tool lifecycle management system (`qsdev enable`, `qsdev disable`, `qsdev status`) that allows developers to cleanly add, remove, and swap individual tools, modules, skills, and plugins. Then extend gdev with high-leverage integrations: SAST (Semgrep), secret scanning (Gitleaks), container security (Grype+Syft+Cosign), license compliance (ScanCode), development secret management (SecretSpec), CI workflow generation, MCP server curation (Context7), changelog automation (git-cliff), and enhanced pre-commit hooks. Every tool is individually toggleable through the lifecycle system.

## Dependencies

Phases 1-8 complete (core generation pipeline, all ecosystem modules, migration infrastructure). Phase 9 desirable (OS detection powers platform-appropriate tool installation). Phase 11 desirable (AI agent tools become lifecycle-managed alongside Phase 12 tools).

## Phase Outputs

- `qsdev enable <tool>` / `qsdev disable <tool>` commands for granular tool management
- `qsdev status` showing all tools with enabled/disabled state and generated files
- `qsdev list` showing available tools, modules, skills, and plugins with descriptions
- File ownership registry mapping each tool to its generated files and shared-file sections
- Clean removal of tool artifacts from both dedicated files and shared files (settings.json, CLAUDE.md, .mcp.json, devenv.nix)
- 9 new tool integrations (Semgrep, Gitleaks, Grype+Syft+Cosign, ScanCode, SecretSpec, CI workflow engine, Context7 MCP, git-cliff, enhanced hooks)
- Wizard updates for all new tools with smart defaults

---

### Unit 12.1: Tool Lifecycle Management System

**Description:** Implement the core tool lifecycle infrastructure: a tool registry that tracks what's available and enabled, file ownership that maps tools to their generated artifacts, and `qsdev enable`/`qsdev disable`/`qsdev status`/`qsdev list` commands that cleanly add and remove tools with surgical shared-file editing.

**Context:** The current architecture only supports `qsdev init` (generate everything at once) and `qsdev init --update` (regenerate). There is no way to toggle individual tools after initial setup. A developer who tries Version-Sentinel and decides they don't want it has to manually hunt down and delete its files, remove its entries from settings.json and CLAUDE.md, and clean up devenv.nix packages. This friction discourages experimentation and makes tool adoption a one-way door.

The lifecycle system turns tool adoption into a reversible, low-risk decision. The saved answers file (`.devinit/.qsdev-init-answers.yaml`) already records choices; the state file (`.devinit/.qsdev-init-state.yaml`) already records generated files with hashes. What's missing is: (a) file ownership linking files to the tool that created them, (b) shared-file surgery to remove a tool's contributions from files that multiple tools write to, and (c) commands that orchestrate the enable/disable flow.

**Desired Outcome:** `qsdev enable semgrep` adds Semgrep to the project. `qsdev disable semgrep` cleanly removes it — deleting `.semgrep.yml`, removing `semgrep` from devenv.nix packages, removing the Semgrep pre-commit hook, and removing the Semgrep CI step from generated workflows. No manual cleanup needed.

**Steps:**
1. Define `Tool` registry struct:
   ```go
   type Tool struct {
       Name         string           // "semgrep", "gitleaks", "version-sentinel", "semble", etc.
       DisplayName  string           // "Semgrep CE (SAST)"
       Category     string           // "security", "ai-agent", "devex", "infrastructure"
       Description  string           // One-line description
       Default      DefaultPolicy    // AlwaysOn, OnWhenDetected, OptIn, AlwaysOff
       DetectFunc   func(*OSInfo, *DetectedProject) bool  // When to auto-enable
       Prerequisites []string        // Required tools (e.g., semble needs python3)
       Conflicts    []string         // Mutually exclusive tools
       OwnedFiles   []FileOwnership  // Files this tool creates/contributes to
   }
   ```
2. Define `FileOwnership`:
   ```go
   type FileOwnership struct {
       Path       string       // Relative path (e.g., ".semgrep.yml", "devenv.nix")
       Ownership  OwnershipType // Exclusive (tool owns entire file) or Shared (tool contributes a section)
       SectionID  string       // For shared files: section identifier (e.g., "semgrep" in devenv.nix packages)
   }
   ```
3. Define `OwnershipType` enum: `Exclusive` (file created and removed entirely by this tool), `Shared` (tool contributes a section/entry to a file other tools also write to).
4. Implement shared-file surgery for each shared file format:
   - **devenv.nix**: Tools contribute packages and enterShell commands. Use section markers (`# --- semgrep ---` / `# --- end semgrep ---`) around each tool's contributions. Enable/disable adds/removes the marked section.
   - **devenv.yaml**: Tools contribute inputs. Structured YAML — parse, add/remove key, marshal back.
   - **settings.json** (Claude Code): Tools contribute hooks and deny rules. Parse JSON, add/remove entries by tool identifier, marshal back.
   - **CLAUDE.md**: Tools contribute documentation sections. Use section markers (`<!-- gdev:semgrep -->` / `<!-- /gdev:semgrep -->`). Enable/disable adds/removes the marked section.
   - **.mcp.json**: Tools contribute server entries. Parse JSON, add/remove server key, marshal back.
   - **.pre-commit-config.yaml** / **devenv hooks**: Tools contribute hook entries. Parse YAML, add/remove entries, marshal back.
   - **CI workflows** (.github/workflows/*.yml): Tools contribute job steps. Regenerate entire workflow from current enabled tools (don't try to surgically edit — too complex).
5. Extend `GeneratedState` to track file ownership:
   ```go
   type FileState struct {
       Hash     string
       Strategy MergeStrategy
       Mode     os.FileMode
       Owner    string // Tool name, or "core" for framework files
       Section  string // Section ID for shared files
   }
   ```
6. Implement `qsdev enable <tool>`:
   a. Validate tool exists and prerequisites met.
   b. Check for conflicts with currently enabled tools.
   c. Update saved answers (toggle the tool's flag to true).
   d. Generate only the affected files (call tool's generator).
   e. For shared files: parse existing file, insert tool's section, write back.
   f. For exclusive files: write new file.
   g. Update state tracking with new file ownership.
   h. Print summary of changes.
7. Implement `qsdev disable <tool>`:
   a. Validate tool is currently enabled.
   b. Update saved answers (toggle flag to false).
   c. For exclusive files: delete the file, remove from state.
   d. For shared files: parse existing file, remove tool's section, write back.
   e. If shared file becomes empty after removal, delete it.
   f. Update state tracking.
   g. Print summary of removals.
   h. Warn if user has modified any of the files being changed (hash mismatch).
8. Implement `qsdev status`:
   - Table output: Tool name, Category, Enabled/Disabled, Files owned, Notes
   - Color-coded: green for enabled, dim for disabled, yellow for enabled-but-modified
   - Support `qsdev status --json` for machine-readable output.
9. Implement `qsdev list`:
   - Show all available tools grouped by category
   - Mark enabled/disabled, show one-line description
   - Show prerequisites and conflicts
   - Support `qsdev list --category security` filtering.
10. Register tool definitions for all existing Phase 4/11 tools (attach-guard, agent-postmortem, Version-Sentinel, semble) and all Phase 12 tools. Phase 2 ecosystem modules are NOT individually toggleable via this system — they're controlled by language detection/selection in the wizard.
11. Wire enable/disable into the wizard: the customize path shows a tool selection screen powered by the tool registry. Quick path uses defaults.
12. Write comprehensive tests: enable → verify files created → disable → verify files removed → verify shared files cleaned up.

**Acceptance Criteria:**
- [ ] `qsdev enable semgrep` generates `.semgrep.yml`, adds package to devenv.nix, adds hook to pre-commit config, adds CI step
- [ ] `qsdev disable semgrep` removes `.semgrep.yml`, removes package from devenv.nix, removes hook, removes CI step
- [ ] `qsdev disable version-sentinel` removes plugin install instructions, cleans CLAUDE.md section, removes hooks from settings.json
- [ ] `qsdev disable semble` removes MCP entry from .mcp.json, removes `.claude/agents/semble-search.md`
- [ ] `qsdev status` shows all tools with correct enabled/disabled state
- [ ] `qsdev list` shows all available tools grouped by category
- [ ] Shared file surgery preserves user modifications outside of tool sections
- [ ] Warning when disabling a tool whose files have been user-modified
- [ ] Enable/disable is idempotent (enabling an already-enabled tool is a no-op)
- [ ] `qsdev enable <tool>` fails cleanly if prerequisites are missing
- [ ] `qsdev enable <tool>` fails cleanly if conflicting tool is enabled
- [ ] State file correctly tracks file ownership after enable/disable cycles

**Research Citations:**
- `addons/devinit/commands.go` — existing `runInit` pipeline and state management
- `internal/state/state.go` — existing `GeneratedState`, `RecordFiles`, `CheckModified`
- `pkg/types/types.go` — existing `GeneratedFile`, `FileState`, `MergeStrategy`
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Section Markers` — marker-based section management for CLAUDE.md
- `research-spikes/gdev-extension-design/config-template-engine-design.md § Per-Format Generation` — JSON/YAML/Nix generation strategies

**Status:** Not Started

---

### Unit 12.2: SAST Integration — Semgrep Per-Ecosystem Configuration

**Description:** Register Semgrep CE as a lifecycle-managed tool with per-ecosystem rule set selection, generating `.semgrep.yml`, devenv.nix package entry, pre-commit hook, and CI workflow step.

**Context:** Semgrep CE (LGPL-2.1, 11K stars) provides single-file SAST with 3,000+ community rules covering 30+ languages. Each ecosystem module maps to specific Semgrep rule registries. The `.semgrep.yml` aggregates rule sets from all detected ecosystems. Integration uses the lifecycle system from Unit 12.1 — Semgrep is default-on and individually disableable.

**Desired Outcome:** `qsdev init` generates Semgrep config for detected ecosystems. `qsdev disable semgrep` cleanly removes it. `qsdev enable semgrep` adds it back.

**Steps:**
1. Register Semgrep in the tool registry: category `security`, default `AlwaysOn`, no prerequisites (Python-based but we install via devenv.nix).
2. Add `SemgrepRuleSets() []string` as a supplementary interface (`SASTModule`) on ecosystem modules — not a breaking change to the base `EcosystemModule` interface.
3. Implement rule set mappings for Tier 1 ecosystems:
   - Go: `p/golang`, `p/owasp-top-ten`
   - JS/TS: `p/typescript`, `p/javascript`, `p/react`, `p/nextjs`, `p/owasp-top-ten`, `p/xss`
   - Python: `p/python`, `p/django`, `p/flask`, `p/owasp-top-ten`
   - Rust: `p/rust`, `p/owasp-top-ten`
   - Java/Kotlin: `p/java`, `p/kotlin`, `p/spring`, `p/owasp-top-ten`
   - .NET: `p/csharp`, `p/owasp-top-ten`
   - Docker: `p/dockerfile`
   - Terraform: `p/terraform`, `p/terraform-aws`
4. Generate `.semgrep.yml` (exclusive file) aggregating rule sets with path exclusions (`vendor/`, `node_modules/`, `dist/`, `.devenv/`).
5. Contribute `semgrep` package to devenv.nix (shared file, `semgrep` section).
6. Contribute pre-commit hook: `semgrep --config auto --error` (shared file, `semgrep` section).
7. Contribute CI workflow step: `semgrep ci` (shared file, `semgrep` section).
8. Contribute CLAUDE.md section documenting Semgrep usage and custom rule patterns.

**Acceptance Criteria:**
- [ ] `.semgrep.yml` generated with ecosystem-appropriate rule sets
- [ ] Multi-ecosystem project (Go + TypeScript) gets combined rule sets
- [ ] `qsdev enable semgrep` / `qsdev disable semgrep` work cleanly
- [ ] Pre-commit hook runs on staged files
- [ ] CI step included in generated workflow
- [ ] Path exclusions prevent false positives from vendored code

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.1` — Semgrep integration design
- `artifacts/devsecops-ecosystem-research.md § 2. SAST` — Semgrep evaluation
- `artifacts/language-ecosystem-coverage.md` — per-ecosystem tooling

**Status:** Not Started

---

### Unit 12.3: Secret Scanning — Gitleaks Pre-Commit and CI

**Description:** Register Gitleaks as a lifecycle-managed tool replacing ripsecrets as the primary secret scanner, with `.gitleaks.toml` configuration, pre-commit hook, and CI full-history scanning.

**Context:** Gitleaks (MIT, 26K stars) detects 150+ secret patterns. It replaces ripsecrets as the primary scanner (ripsecrets retained as ultra-fast backup). The `.gitleaks.toml` includes team-configurable allowlists for false positive management. Integration profile registry URLs are auto-added to the allowlist.

**Desired Outcome:** Gitleaks is default-on, individually disableable, and generates appropriate config including infrastructure-aware allowlists.

**Steps:**
1. Register Gitleaks in tool registry: category `security`, default `AlwaysOn`, no prerequisites (Go binary via devenv.nix).
2. Generate `.gitleaks.toml` (exclusive file):
   - Standard 150+ pattern rule set (built-in)
   - Allowlist: internal registry URLs from infrastructure profile, test fixture paths
   - Path exclusions: `vendor/`, `node_modules/`, `.devenv/`, `docs/`
3. Contribute `gitleaks` package to devenv.nix (shared, `gitleaks` section).
4. Contribute baseline pre-commit hook: `gitleaks protect --staged` (shared, `gitleaks` section).
5. Retain ripsecrets as secondary hook (already in Phase 5) — belt-and-suspenders.
6. Contribute CI workflow step: `gitleaks detect --source . --report-format sarif --report-path gitleaks-report.sarif`.
7. Contribute CLAUDE.md section: false positive management, allowlist editing.

**Acceptance Criteria:**
- [ ] `.gitleaks.toml` generated with infrastructure-aware allowlists
- [ ] Pre-commit hook blocks commits with detected secrets
- [ ] CI produces SARIF report for GitHub Code Scanning integration
- [ ] `qsdev enable gitleaks` / `qsdev disable gitleaks` work cleanly
- [ ] Ripsecrets hook unaffected by gitleaks enable/disable

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.2` — Gitleaks integration design
- `artifacts/devsecops-ecosystem-research.md § 3. Secret Scanning` — Gitleaks evaluation

**Status:** Not Started

---

### Unit 12.4: Container Security Pipeline — Grype + Syft + Cosign

**Description:** Register the container security pipeline (Grype scanning, Syft SBOM, Cosign signing) as a lifecycle-managed tool set, conditional on Docker ecosystem detection.

**Context:** The March 2026 Trivy supply chain compromise makes Grype the necessary primary scanner. Syft generates SBOMs in CycloneDX format. Cosign provides keyless container signing via OIDC in CI. This replaces the Trivy references in Phase 5. The pipeline is auto-enabled when Docker ecosystem is detected, individually disableable.

**Desired Outcome:** Docker projects get a scan-sign-verify pipeline in CI and a local `qsdev docker scan` command, cleanly removable via `qsdev disable container-security`.

**Steps:**
1. Register `container-security` as a composite tool (bundles Grype, Syft, Cosign): category `security`, default `OnWhenDetected` (Docker ecosystem), prerequisites `docker`.
2. Contribute `grype`, `syft`, `cosign` packages to devenv.nix (shared, `container-security` section).
3. Generate `.grype.yaml` (exclusive file): failure threshold `high`, ignored CVEs list.
4. Generate `.cosign/policy.yaml` (exclusive file): keyless verification settings.
5. Contribute CI workflow jobs (shared, `container-security` section):
   - `syft <image>:<tag> -o cyclonedx-json > sbom.json`
   - `grype sbom:sbom.json --fail-on high`
   - `cosign sign --yes <image>@<digest>` (OIDC permissions in workflow)
6. Add `qsdev docker scan <image>` command running Syft+Grype locally.
7. Contribute CLAUDE.md section: Trivy compromise as trust model lesson, Grype as replacement.

**Acceptance Criteria:**
- [ ] Pipeline auto-enabled when Docker detected, skipped otherwise
- [ ] `qsdev disable container-security` removes all three tools' artifacts
- [ ] Syft produces valid CycloneDX JSON
- [ ] Grype fails CI on high/critical vulns
- [ ] Cosign keyless signing works in GitHub Actions via OIDC
- [ ] Local `qsdev docker scan` works without CI infrastructure

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.3` — Container security pipeline design
- `artifacts/devsecops-ecosystem-research.md § 4. Container Security` — Grype + Dockle evaluation
- `artifacts/artifact-stores-caches-research.md § SBOM and Provenance Tools` — Syft + cosign

**Status:** Not Started

---

### Unit 12.5: License Compliance — ScanCode Toolkit

**Description:** Register ScanCode Toolkit as a lifecycle-managed tool for license policy enforcement, generating `.scancode.yml` with the firm's standard policy and a weekly CI scan.

**Context:** ScanCode (Apache-2.0, 5.8K stars) detects licenses at audit-grade accuracy. It's too slow for pre-commit (<10s budget) but appropriate for weekly CI. The firm's standard policy: MIT/Apache/BSD/ISC allowed, GPL/AGPL blocked, LGPL/MPL require review. Infrastructure profiles can customize the policy per client.

**Desired Outcome:** Weekly CI license scans enforce license policy. `qsdev disable license-compliance` removes it entirely.

**Steps:**
1. Register `license-compliance` in tool registry: category `security`, default `OptIn` (requires explicit enable due to scan duration).
2. Generate `.scancode.yml` (exclusive file) with firm's license policy:
   - Allowed: MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, Unlicense, CC0-1.0, 0BSD
   - Blocked: GPL-2.0, GPL-3.0, AGPL-3.0, SSPL-1.0, EUPL-1.1
   - Review: LGPL-2.1, LGPL-3.0, MPL-2.0, CDDL-1.0, EPL-2.0
   - Policy overridable via infrastructure profile
3. Generate `.license-exceptions.yml` (exclusive file): empty template for justified exceptions.
4. Contribute CI workflow job (shared, `license-compliance` section): weekly schedule, `scancode --license --copyright --json-pp license-report.json .`
5. Add `qsdev license check` command for local scanning.
6. Contribute CLAUDE.md section: license policy, how to handle violations, exception process.

**Acceptance Criteria:**
- [ ] `.scancode.yml` generated with firm's standard policy
- [ ] Weekly CI scan detects blocked licenses
- [ ] Exception file supports justified overrides
- [ ] `qsdev enable license-compliance` / `qsdev disable license-compliance` work cleanly
- [ ] Policy customizable via infrastructure profile

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.4` — License compliance design
- `artifacts/devsecops-ecosystem-research.md § 6. License Compliance` — ScanCode evaluation

**Status:** Not Started

---

### Unit 12.6: Development Secret Management — SecretSpec

**Description:** Register SecretSpec as a lifecycle-managed tool generating `secretspec.toml` declaring development secrets inferred from detected services and ecosystems.

**Context:** SecretSpec ships with devenv 2.0 (Apache-2.0). It declares what secrets a project needs and lets each environment provide them differently (keyring for local dev, env vars for CI, 1Password for teams). gdev infers secret declarations from detected services (PostgreSQL needs `DATABASE_URL`, Terraform needs `AWS_ACCESS_KEY_ID`).

**Desired Outcome:** `qsdev init` with services detected generates SecretSpec config. `qsdev disable secretspec` removes it cleanly.

**Steps:**
1. Register `secretspec` in tool registry: category `devex`, default `OnWhenDetected` (services present), no prerequisites (ships with devenv 2.0).
2. Add `SecretDeclarations() []SecretDecl` as a supplementary interface on service templates and ecosystem modules.
3. Implement declarations: PostgreSQL → `DATABASE_URL`/`POSTGRES_PASSWORD`, Redis → `REDIS_URL`, Terraform → `AWS_*`, Docker → `DOCKER_REGISTRY_TOKEN`.
4. Generate `secretspec.toml` (exclusive file) composing declarations from detected modules.
5. Contribute devenv.nix secretspec integration block (shared, `secretspec` section).
6. Configure default providers: `keyring` for local dev, `env` for CI.
7. Contribute CLAUDE.md section: provider options, SOPS limitation, adding custom secrets.

**Acceptance Criteria:**
- [ ] `secretspec.toml` generated with secrets from detected services
- [ ] Auto-generated secrets work for local DB passwords
- [ ] `qsdev enable secretspec` / `qsdev disable secretspec` work cleanly
- [ ] devenv.nix integration block managed via lifecycle system
- [ ] Secrets never written in plaintext to generated files

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.5` — SecretSpec integration design

**Status:** Not Started

---

### Unit 12.7: CI Workflow Generation Engine

**Description:** Implement a lifecycle-aware CI workflow generation engine that composes steps from all enabled tools into GitHub Actions and GitLab CI workflows with SHA-pinned action references.

**Context:** Unlike other tools that contribute sections to shared files, CI workflows are fully regenerated from the current enabled-tool set on every `qsdev enable`/`qsdev disable` call. This is the one shared file type where surgical editing is too complex — workflow YAML has dependency ordering, matrix strategies, and conditional jobs that interact non-locally. Full regeneration from the tool registry is cleaner and more reliable.

**Desired Outcome:** CI workflows always reflect the exact set of currently enabled tools. `qsdev enable semgrep` regenerates the workflow to include the Semgrep step. `qsdev disable semgrep` regenerates without it.

**Steps:**
1. Create `internal/cigeneration/` package with `Generator` interface supporting GitHub Actions and GitLab CI.
2. Each tool in the registry contributes `CISteps() []CIStep` when enabled.
3. Implement `GitHubActionsGenerator` producing `.github/workflows/security.yml`:
   - Harden-Runner as first step in every job (always-on)
   - Job composition from enabled tools: lint/SAST, secret scan, vuln scan, container security, license compliance, security review
   - All action references SHA-pinned
   - OIDC permissions for cosign
   - `permissions:` block with least-privilege
4. Implement `GitLabCIGenerator` producing `.gitlab-ci.yml` with equivalent stages.
5. CI workflow is treated as an exclusive file owned by a virtual `ci-workflows` tool that's always enabled. It regenerates on any tool enable/disable.
6. Add `--ci-platform github|gitlab|none` flag.
7. Infrastructure profile contributes registry proxy config for CI environment.

**Acceptance Criteria:**
- [ ] GitHub Actions workflow is valid YAML with SHA-pinned actions
- [ ] Workflow regenerates correctly when tools are enabled/disabled
- [ ] Harden-Runner is first step in every job
- [ ] Container security jobs conditional on Docker ecosystem
- [ ] GitLab CI equivalent works
- [ ] Least-privilege permissions configured

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.6` — CI workflow generation design
- `artifacts/devsecops-ecosystem-research.md § 8. CI/CD Pipeline Security` — Harden-Runner, SHA pinning

**Status:** Not Started

---

### Unit 12.8: MCP Server Curation — Context7 and Tiered Defaults

**Description:** Register Context7 as a default-on MCP server and formalize the MCP server curation strategy with tiered defaults managed through the lifecycle system.

**Context:** Context7 (Apache-2.0, 55K stars) provides version-specific library docs for 50K+ libraries, preventing AI agents from hallucinating stale APIs. Research consensus: 3-6 MCP servers is the sweet spot. The `.mcp.json` file is shared — each MCP server is a lifecycle-managed tool with its own enable/disable.

**Desired Outcome:** Default `.mcp.json` has 3-4 servers. Each is individually toggleable via `qsdev enable`/`qsdev disable`.

**Steps:**
1. Register MCP servers as individual tools in the registry:
   - `context7`: category `ai-agent`, default `AlwaysOn` — library documentation
   - `github-mcp`: category `ai-agent`, default `AlwaysOn` — GitHub integration (already in Phase 4)
   - `socket-dev-mcp`: category `ai-agent`, default `OnWhenDetected` (JS/Python/Rust/Go) — supply chain risk
   - `semble`: category `ai-agent`, default `OnWhenDetected` (Python >=3.10) — code search (already in Phase 11)
   - `postgres-mcp`: category `ai-agent`, default `OnWhenDetected` (PostgreSQL service) — DB queries
2. Each MCP tool's enable/disable adds/removes its entry from `.mcp.json` (shared file, keyed by server name).
3. Add Context7 MCP config:
   ```json
   "context7": {
     "command": "npx",
     "args": ["-y", "@upstash/context7-mcp"]
   }
   ```
4. Ensure total default-on servers <=4 (context7, github, plus detected ones).
5. Contribute CLAUDE.md section per MCP server documenting its purpose.
6. Add `qsdev list --category ai-agent` to show all available MCP servers.

**Acceptance Criteria:**
- [ ] Context7 configured by default
- [ ] `qsdev disable context7` removes it from .mcp.json
- [ ] `qsdev enable postgres-mcp` adds it to .mcp.json
- [ ] Total default servers <=4
- [ ] Each server documented in CLAUDE.md
- [ ] `.mcp.json` merge preserves non-gdev entries

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.7` — MCP curation design
- `artifacts/mcp-devex-ecosystem-research.md` — Context7, MCP Toolbox, tiered strategy
- `artifacts/claude-code-ecosystem-expansion-research.md § 3. Documentation` — Context7 evaluation

**Status:** Not Started

---

### Unit 12.9: Enhanced Pre-Commit Hook Suite

**Description:** Extend the pre-commit hook suite with lifecycle-managed Gitleaks and Semgrep hooks, plus optional commitlint for conventional commit enforcement.

**Context:** Phase 5 defined three hook tiers (baseline/enhanced/specialized). This unit adds Phase 12 security tools to the appropriate tiers and introduces commitlint as an opt-in hook for teams using conventional commits. All new hooks are individually toggleable — they're registered as part of their parent tool's file ownership (Gitleaks hook is owned by the `gitleaks` tool, Semgrep hook by `semgrep`).

**Desired Outcome:** Pre-commit hooks stay under 10 seconds total. `qsdev disable gitleaks` also removes its pre-commit hook.

**Steps:**
1. Gitleaks hook (owned by `gitleaks` tool, not separately toggleable): `gitleaks protect --staged` in baseline tier.
2. Semgrep hook (owned by `semgrep` tool): `semgrep --config auto --error` on changed files in enhanced tier.
3. Register `commitlint` as a standalone tool: category `devex`, default `OptIn`.
4. Generate commitlint config (`.commitlintrc.yml`) when enabled.
5. Wire commitlint as `commit-msg` hook: `commitlint --edit $1`.
6. Verify total hook execution <10 seconds: gitleaks <1s, Semgrep <3s on staged files, formatters <2s, ripsecrets <0.1s.
7. Support `--hook-tier baseline|enhanced|specialized|full` flag.

**Acceptance Criteria:**
- [ ] Gitleaks hook added/removed with `qsdev enable/disable gitleaks`
- [ ] Semgrep hook added/removed with `qsdev enable/disable semgrep`
- [ ] Commitlint independently toggleable via `qsdev enable/disable commitlint`
- [ ] Total hook time <10 seconds on typical commits
- [ ] All hooks compatible with prek (devenv 1.11+ runner)

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.8` — Hook suite design
- `research-spikes/devenv-security/precommit-hooks-research.md` — original 3-tier design

**Status:** Not Started

---

### Unit 12.10: Changelog Automation — git-cliff

**Description:** Register git-cliff as a lifecycle-managed tool generating `cliff.toml` configuration and a `qsdev changelog` command.

**Context:** git-cliff (MIT/Apache-2.0, 11K stars) is a Rust single binary for changelog generation from conventional commits. It matches gdev's zero-Node-dependency philosophy. This is opt-in — not all projects use conventional commits.

**Desired Outcome:** `qsdev enable changelog` adds git-cliff config and the `qsdev changelog` command. `qsdev disable changelog` removes it.

**Steps:**
1. Register `changelog` in tool registry: category `devex`, default `OptIn`.
2. Generate `cliff.toml` (exclusive file) with firm's standard format.
3. Contribute `git-cliff` package to devenv.nix (shared, `changelog` section).
4. Add `qsdev changelog` command (thin wrapper around `git-cliff`).
5. Contribute CI workflow step (shared, `changelog` section): on tag push, generate CHANGELOG.md.
6. Suggest enabling `commitlint` (Unit 12.9) when `changelog` is enabled.

**Acceptance Criteria:**
- [ ] `cliff.toml` generated with firm's standard format
- [ ] `qsdev enable changelog` / `qsdev disable changelog` work cleanly
- [ ] `qsdev changelog` produces valid CHANGELOG.md
- [ ] CI step generates changelog on tag push

**Research Citations:**
- `artifacts/phase-12-recommendations.md § Unit 12.9` — git-cliff integration design

**Status:** Not Started

---

### Unit 12.11: Retroactive Lifecycle Registration for Phase 4/11 Tools

**Description:** Register all existing Phase 4 and Phase 11 tools (attach-guard, agent-postmortem, Version-Sentinel, semble) in the lifecycle system so they become individually toggleable.

**Context:** Phase 4 deployed attach-guard and Trail of Bits skills. Phase 11 added agent-postmortem, Version-Sentinel, and semble. These were originally generated as part of `qsdev init` with no individual toggle. This unit retrofits them into the lifecycle system by defining their `Tool` registry entries and `FileOwnership` mappings, enabling `qsdev disable version-sentinel` to work even though the tool was deployed by Phase 4/11 code.

**Desired Outcome:** All previously deployed tools are individually toggleable via `qsdev enable`/`qsdev disable`.

**Steps:**
1. Register `attach-guard` — hooks in settings.json, package-guard.py script.
2. Register `agent-postmortem` — `.claude/skills/agent-postmortem/SKILL.md` (exclusive file).
3. Register `version-sentinel` — plugin install instructions in CLAUDE.md, prerequisites in doctor.
4. Register `semble` — `.mcp.json` entry and/or `.claude/agents/semble-search.md`.
5. Register `trail-of-bits-skills` — skill files in `.claude/skills/`.
6. Define file ownership for each, including shared-file sections in CLAUDE.md and settings.json.
7. Migrate existing generated files to use section markers where they don't already.
8. Test full enable/disable cycle for each tool.

**Acceptance Criteria:**
- [ ] `qsdev disable version-sentinel` cleanly removes all VS artifacts
- [ ] `qsdev disable semble` removes MCP entry and/or agent file
- [ ] `qsdev disable agent-postmortem` removes skill file
- [ ] `qsdev enable <tool>` re-adds previously disabled tools
- [ ] Existing projects can adopt lifecycle management without re-running `qsdev init`
- [ ] Section markers added to existing shared files on first lifecycle operation

**Research Citations:**
- `phases/04-claude-code-addon-core-generation.md` — original tool deployment design
- `phases/11-ai-agent-tooling-integration.md` — Phase 11 tool integration design

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All eleven units pass acceptance criteria
- [ ] `qsdev enable semgrep` → verify files created → `qsdev disable semgrep` → verify files cleaned up (no orphaned artifacts)
- [ ] `qsdev status` accurately reflects all enabled/disabled tools
- [ ] `qsdev list` shows all 15+ available tools with correct categories and descriptions
- [ ] Shared files (devenv.nix, settings.json, CLAUDE.md, .mcp.json) correctly updated on every enable/disable
- [ ] CI workflow regenerates correctly as tools are toggled
- [ ] Full enable → disable → re-enable cycle works for every tool
- [ ] User modifications outside of tool sections preserved during enable/disable operations
- [ ] `qsdev init --yes` on a fresh project generates correct default tool set
- [ ] `qsdev init --update` respects current enable/disable state
- [ ] All Phase 4/11 tools retrofitted and individually toggleable
