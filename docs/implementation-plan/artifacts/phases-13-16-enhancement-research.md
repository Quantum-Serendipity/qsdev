# Phases 13-16 Enhancement Research Synthesis

Research synthesis from 5 parallel investigation spikes into implementation-ready recommendations for 4 new gdev phases. This artifact is the primary research reference for anyone implementing Phases 13-16 of the qsdev plan.

---

## 1. Executive Summary

Five focused research spikes investigated the design space for extending gdev beyond its core 12-phase secure development environment bootstrap. Each spike conducted deep investigations with multiple sub-agent research threads, prior art surveys, and detailed design specifications. The spikes collectively produced 30+ detailed research reports, surveyed 40+ prior art tools and projects, and generated complete data models, CLI output mockups, and ready-to-embed skill/agent definitions.

The research yields four new phases:

- **Phase 13: Project Configuration & Team Standards** -- Three-layer configuration hierarchy (binary defaults, project config, local overrides), four onboarding modes, Terraform-style version constraints, CI enforcement via `qsdev check`, and client-specific compliance profiles.

- **Phase 14: Claude Code Integration & Agentic Skills** -- 10 gdev operation skills (6 user-only, 4 Claude-invocable), 7 consulting agents, 8+ consulting skills, five-layer context architecture, precision-scoped guardrail integration, and a five-layer safety model.

- **Phase 15: Health, Status & Compliance** -- Scorecard-inspired three-layer posture scoring (0-100, A-F grades), conformance labels (baseline/enhanced PASS/FAIL), six-category drift detection completing in under 100ms, machine-readable output (JSON versioned, SARIF, shields.io badge), and team aggregation via CI artifacts.

- **Phase 16: Developer Experience Polish** -- `qsdev repair` (conservative self-healing), `qsdev info` (lightweight status), `qsdev outdated` (cross-ecosystem freshness), `qsdev update` (coordinated infrastructure updates), `qsdev teardown` (clean project exit with 3 profiles), git workflow automation (PR templates, branch naming, ticket extraction, PR labels), and shell integration (starship config, gdev env vars, enterShell notification).

---

## 2. Research Methodology

### Investigation Structure

Five parallel research spikes were launched, each with a focused scope and dedicated sub-agent investigations:

| Spike | Focus Area | Sub-Topics | Reports Produced | Key Sources |
|-------|-----------|------------|-----------------|-------------|
| `gdev-team-config-onboarding` | Team config sharing, onboarding, versioning, CI enforcement, consulting lifecycle | 6 | 6 detailed reports | EditorConfig, Biome, mise, proto, Renovate, Copier, Projen, Nx, Dev Containers, Terraform, Cargo MSRV |
| `gdev-claude-code-integration` | Skill file format, operation mapping, CLI wrapper patterns, CLAUDE.md integration, safety | 6 | 1 comprehensive report (9 sections) | Anthropic docs, Trail of Bits skills, DevOps skill patterns, GitHub PR skill |
| `gdev-agentic-workflows` | Pre-built skills/agents, agent files format, task templates, context management, guardrails, prior art, consulting differentiation | 7 | 7 detailed reports | Trail of Bits skills + config, Security Phoenix, awesome-claude-code-toolkit, Cursor rules, Anthropic agent docs |
| `gdev-health-reporting` | Prior art, posture model, status UX, machine-readable output, drift detection, team reporting, badges | 7 | 7 detailed reports | OpenSSF Scorecard, npm audit, cargo audit, govulncheck, Safety CLI, OWASP Dependency-Check, SARIF spec, ASVS v5.0 |
| `gdev-dx-polish` | Observability, task runner, git workflows, environment switching, error recovery, shell integration, dependency freshness, what NOT to include | 8 | 8 detailed reports | devenv 2.0 tasks, Starship, flutter doctor, brew doctor, act, claude-code-otel |

### Research Approach

Each spike followed the repository's research methodology: sub-agent delegation for all investigation work, full source material saved to `docs/` directories, detailed per-topic analysis reports, real-time task tracking, and revision cycles before marking tasks complete. Every finding was checked against the depth checklist (mechanism explained, tradeoffs identified, alternatives compared, failure modes described, examples found, standalone-readable).

---

## 3. Phase 13: Project Configuration & Team Standards -- Research Findings

**Source spike**: `gdev-team-config-onboarding`

### 3.1 Three-Layer Configuration Hierarchy

The central design decision for Phase 13. Six configuration sharing patterns were surveyed across 10+ tools (EditorConfig, Biome, mise, proto, Renovate, Copier, Projen, Nx, Dev Containers, ESLint). The recommended model combines the strongest patterns:

**Layer 1: Org Defaults (Compiled into binary)**
- Non-negotiable security baselines, default profiles, infrastructure settings
- Works offline, cannot be bypassed by deleting a file, version-locked to the binary
- Updated quarterly via binary releases
- Closest prior art: Nx workspace presets

**Layer 2: Project Config (`.qsdev.yaml` in repo)**
- Profile selection, language/service overrides, client-specific settings, gdev version constraint
- Travels with the repo via git, provides the team standard
- Key fields: `version` (config schema integer), `gdev_version` (semver constraint), `profile` (named profile), `overrides` (per-field), `client` (client-specific settings)
- Closest prior art: mise `.mise.toml`, proto `.prototools`

**Layer 3: Local Overrides (`.qsdev.local.yaml`, gitignored)**
- Developer-specific preferences: extra packages, editor tools, permission levels
- Never committed to git
- Closest prior art: mise `.mise.local.toml`

**Resolution**: Deep merge at each layer; later layers override earlier ones. Arrays use union semantics for additive fields (permissions, packages) and replacement semantics for selective fields (languages, services). **Security level acts as a floor that cannot be lowered** by project or local overrides.

### 3.2 Four Onboarding Modes

A detection engine determines the appropriate mode when `qsdev init` runs:

| Mode | Trigger | Behavior |
|------|---------|----------|
| **Create** | No `.qsdev.yaml` found | Full wizard (quick path or customize) |
| **Join** | `.qsdev.yaml` exists, fresh clone | Minimal prompts, verify state, local setup only |
| **Update** | `.qsdev.yaml` exists, newer gdev version | Show changes since last version, offer `--update` |
| **Repair** | `.qsdev.yaml` exists, generated files drifted | Show drifted files, offer to fix |

**Target**: 3 commands (`git clone`, `cd`, `qsdev init`), under 2 minutes for returning engineers. Machine-specific setup (`qsdev devenv setup`) runs once per machine, not per project. The detection engine distinguishes machine-specific setup (devenv, direnv, claude CLI installation) from project-specific setup (already in git).

**Edge cases addressed**: Nix download failures (offline), trust prompt fatigue (pre-trust company directories), version skew in team (actionable errors), partial state from interrupted init (atomic write pipeline), conflicting existing config (plan preview before writing).

### 3.3 Terraform-Style Version Constraints

Three independent version axes that can drift:

1. **Binary version** -- the gdev binary itself (e.g., v0.15.0)
2. **Config schema version** -- the `.qsdev.yaml` format version (integer: 1, 2, 3)
3. **Template version** -- templates used to generate output files (tied to binary version)

`.qsdev.yaml` includes `gdev_version: ">= 0.15.0"` checked before any operation. Config schema migrations chain incrementally (v1 -> v2 -> v3, never v1 -> v3 directly). Support window: current version + 2 previous versions.

**Version ratchet strategy**: Newer gdev versions can update generated files; older versions refuse to downgrade files produced by newer versions. This prevents an engineer with an older binary from overwriting improvements made by a teammate with a newer binary. `qsdev init --update --bump-version` updates the constraint to signal the team.

### 3.4 `qsdev check` CI Enforcement

A read-only validation command for CI pipelines that verifies project compliance against org policy. Five check categories:

1. **Binary compatibility** -- gdev version meets `.qsdev.yaml` constraint
2. **Config integrity** -- `.qsdev.yaml` parses correctly, profile exists, schema version supported
3. **Required tools** -- Org policy mandates certain tools are enabled
4. **Generated file state** -- Machine-owned files match expected output; human-edited files checked for required content (deny rules, section markers)
5. **Security hardening** -- Per-ecosystem security configs present and correct

**Output formats**: Human-readable (default), JSON, SARIF 2.1.0 (for GitHub Security tab), JUnit XML (for Jenkins/GitLab CI). Exit codes: 0 (pass), 1 (fail), 2 (gdev error).

**Auto-fix mode** (`qsdev check --fix`): Applies deterministic fixes for safe issues (missing deny rules, missing pre-commit hooks, missing .gitignore entries). Does NOT auto-fix config structure changes, explicitly disabled security settings, or CI workflow changes.

**Distinguished from `qsdev devenv doctor`**: `qsdev check` validates project config compliance (runs in CI); `qsdev devenv doctor` validates machine/system state (runs locally). They complement each other.

### 3.5 Client-Specific Profiles with Compliance Levels

The `.qsdev.yaml` `client` block encodes client-specific configuration with compliance level mapping:

| Setting | Baseline | Enhanced (SOC2) | Strict (HIPAA/FedRAMP) |
|---------|----------|-----------------|----------------------|
| Age-gating threshold | 72 hours | 168 hours (1 week) | 336 hours (2 weeks) |
| Install script blocking | Enabled | Enabled | Enabled + audit log |
| Vulnerability scanning | osv-scanner | osv-scanner + Semgrep | osv-scanner + Semgrep + daily |
| MCP servers | allow-list | allow-list | explicit allow-list only |
| Pre-commit hooks | ripsecrets, gitleaks | + semgrep, conventional commits | + license scanning |
| Claude Code permissions | standard | restricted | restricted + audit log |
| SBOM generation | off | on release | every build |

Client profiles compose with language/project profiles via the resolution order: Org Defaults -> Client Profile -> Project Profile -> Project Overrides -> Local Overrides. **The client security level is a floor** -- no lower layer can reduce it.

**Teardown profiles**: `--quick` (POC/spike, no archive), default (normal engagement end, archive + token revocation), `--compliance` (regulated client, archive + evidence report required). Archives preserve `.qsdev.yaml`, state, last compliance report, tool versions, and devenv lock for re-engagement months later.

**Compliance evidence**: `qsdev evidence` generates machine-readable reports mapping gdev's security controls to compliance frameworks (SOC2 controls, HIPAA safeguards). Evidence is integrity-verified via gdev binary version hash, not cryptographically signed.

### 3.6 Prior Art Analysis

Six configuration sharing patterns analyzed:

| Pattern | Exemplars | Applicable to gdev |
|---------|-----------|-------------------|
| File-in-repo | EditorConfig, mise, proto | Yes -- Layer 2 (`.qsdev.yaml`) |
| Shareable config packages | ESLint, Prettier | No -- requires package registry, gdev is a compiled binary |
| Preset repository | Renovate | Possible future extension via `extends` field |
| Template + update | Copier, Projen | Partially -- gdev's migration strategy draws from Copier's three-way merge |
| Workspace generators | Nx | Yes -- Layer 1 (compiled profiles are workspace generators) |
| Feature composition | Dev Containers | No -- requires container runtime |

---

## 4. Phase 14: Claude Code Integration & Agentic Skills -- Research Findings

**Source spikes**: `gdev-claude-code-integration` + `gdev-agentic-workflows`

### 4.1 Skill File Format

Skills are the recommended integration mechanism (replacing legacy `.claude/commands/`). A skill is a directory containing `SKILL.md` with YAML frontmatter and markdown body:

```
.claude/skills/<skill-name>/
+-- SKILL.md           # Main instructions (required)
+-- references/        # Reference material Claude reads on demand
+-- scripts/           # Scripts Claude can execute
+-- examples/          # Example outputs
```

**Key frontmatter fields**: `name`, `description` (triggers auto-invocation), `disable-model-invocation` (user-only), `allowed-tools` (pre-approved tools during skill execution), `model`, `effort`, `context` (fork for subagent isolation), `agent` (subagent type), `arguments`.

**Invocation control**: `disable-model-invocation: true` removes the skill from Claude's context entirely until the user types the command -- zero context cost and no autonomous triggering. This is the correct choice for gdev operations with side effects.

### 4.2 Dynamic Context Injection

The `` !`command` `` syntax is a preprocessor that runs shell commands before skill content reaches Claude. Output replaces the placeholder:

```markdown
## Current system state
!`qsdev devenv doctor --json 2>/dev/null || echo '{"error": "gdev not installed"}'`
```

This is the strongest CLI wrapper pattern identified in the ecosystem (from the GitHub PR summary skill pattern). It gives Claude actual project state before reasoning begins, which is dramatically more efficient than having Claude run discovery commands one by one.

Multi-line variant uses fenced code blocks opened with `` ```! ``. The `|| echo` fallback pattern ensures Claude always gets parseable output even when gdev is not installed.

### 4.3 Ten gdev Operation Skills

| Operation | Skill Name | Side Effects | Invocation | Rationale |
|-----------|-----------|-------------|-----------|-----------|
| Initialize project | `/gdev-init` | Creates files | User-only | Major side effect |
| Onboard existing project | `/gdev-onboard` | Creates/modifies files | User-only | Needs user confirmation |
| Run health check | `/gdev-doctor` | None (read-only) | Both | Safe, autonomous troubleshooting |
| Install prerequisites | `/gdev-setup` | Installs packages | User-only | System-level changes |
| Enable tool | `/gdev-enable` | Modifies configs | User-only | Adds tooling |
| Disable tool | `/gdev-disable` | Modifies configs | User-only | Removes tooling |
| Check tool status | `/gdev-status` | None (read-only) | Both | Safe, autonomous queries |
| List available tools | `/gdev-tools` | None (read-only) | Both | Safe, discovery |
| Generate compliance report | `/gdev-compliance` | Creates report file | User-only | Generates artifacts |
| Update configs | `/gdev-update` | Modifies configs | User-only | Needs confirmation |

Complete SKILL.md implementations are ready in the research report, including dynamic context injection, argument handling, error fallbacks, and step-by-step instructions.

### 4.4 Seven Consulting Agents and 8+ Consulting Skills

**Agents** (isolated context windows, specialized tool restrictions, persistent memory):

| Agent | Model | Tools | Memory | Purpose |
|-------|-------|-------|--------|---------|
| `security-reviewer` | inherit | Read, Grep, Glob, Bash | project | Security-focused code review |
| `test-gap-analyzer` | inherit | Read, Grep, Glob, Bash | -- | Find untested code paths |
| `codebase-explorer` | haiku | Read, Grep, Glob, Bash | project | Rapid codebase understanding |
| `performance-analyzer` | inherit | Read, Grep, Glob, Bash | -- | Performance analysis |
| `accessibility-reviewer` | inherit | Read, Grep, Glob, Bash | -- | WCAG compliance review |
| `documentation-auditor` | haiku | Read, Grep, Glob, Bash | -- | Find undocumented code |
| `incident-investigator` | inherit | Read, Grep, Glob, Bash | project | Production issue investigation |

**Skills** (procedural workflows in main context):

| Skill | Category | Auto-invoke | Purpose |
|-------|----------|------------|---------|
| `/review-pr` | Code Review | No | Comprehensive PR review across security, performance, quality |
| `/review-quick` | Code Review | Yes | Quick change review for obvious issues |
| `/review-accessibility` | Code Review | No | WCAG 2.1 AA compliance review |
| `/refactor-safe` | Refactoring | No | Test-validated refactoring with rollback |
| `/add-tests` | Testing | No | Generate tests following existing codebase patterns |
| `/write-adr` | Documentation | No | Architecture Decision Records |
| `/write-runbook` | Documentation | No | Operational runbooks |
| `/incident-debug` | Incident Response | No | Systematic hypothesis-test-conclude debugging |
| `/onboard` | Onboarding | No | Systematic codebase exploration (forks to `codebase-explorer`) |
| `/upgrade-dep` | Migration | No | Dependency upgrade with changelog research and verification |
| `/migration-plan` | Migration | No | Phased migration planning with rollback strategies |
| `/handoff-doc` | Documentation | No | Client handoff documentation |
| `/compliance-check` | Review | No | Compliance requirements verification |
| `/estimate-effort` | Planning | No | Effort estimation for tasks |
| `/write-api-docs` | Documentation | No | API documentation |

Complete SKILL.md and agent file definitions are ready for embedding in the gdev binary via Go's `embed.FS`.

**Key design principle** (from Trail of Bits): "Encode expertise in agents, procedures in skills." Agents are specialized personas with isolated context. Skills are repeatable procedures that run in the main context.

### 4.5 Five-Layer Context Architecture

Context management is the critical constraint. Performance degrades as context fills -- the "dumb zone" begins at ~40% utilization.

| Layer | What | When Loaded | Token Cost | gdev Control |
|-------|------|------------|-----------|-------------|
| 1. CLAUDE.md | Quick reference, security policy | Every request | 50-100 lines (~2-4KB) | Direct |
| 2. Rules (`.claude/rules/*.md`) | Language conventions with `paths:` frontmatter | On file access match | 30-50 lines each | Direct |
| 3. Skills | Workflow procedures | On invocation only | Descriptions always, body on demand | Direct |
| 4. Agents | Isolated workers | Own context window | Zero main-context cost | Direct |
| 5. Hooks/settings.json | External enforcement | Never loaded to context | Zero | Direct |

**5% context budget target**: Generated config should consume less than 5% of the context window. For Sonnet (200k): ~10,000 tokens. For Opus (1M): ~50,000 tokens.

**Model-aware generation**: Different config for Sonnet vs Opus users:

| Setting | Sonnet (200k) | Opus (1M) |
|---------|--------------|----------|
| CLAUDE.md | 50 lines max | 100 lines max |
| Skills auto-invoke | Top 5 only | All 15 |
| Rules lazy-loading | Aggressive (`paths:` on all) | Moderate (security always-on) |
| Skill descriptions | Name-only for low-priority | Full descriptions |

### 4.6 Deny Rule Conflict Validation

The core tension: deny rules must not block test/build operations that skills need. The solution is precision-scoped deny rules:

**Safe deny rules** (do not conflict with workflows):
```
Bash(npm install *), Bash(npm uninstall *), Bash(npx *)
Bash(pip install *), Bash(pip uninstall *)
Bash(cargo install *), Bash(go install *)
```

**Unsafe deny rules** (would break workflows):
```
Bash(npm *)    -- blocks npm test, npm run, npm audit
Bash(pip *)    -- blocks pip list, pip show
Bash(cargo *)  -- blocks cargo test, cargo build, cargo clippy
```

A conflict detection test validates guardrail-workflow compatibility at `qsdev init` and `qsdev update` time:

```go
func TestGuardrailWorkflowCompatibility(config Config) []Conflict {
    for _, skill := range config.EnabledSkills {
        for _, allowedTool := range skill.AllowedTools {
            for _, denyRule := range config.DenyPatterns {
                if globMatch(denyRule, allowedTool) {
                    conflicts = append(conflicts, Conflict{...})
                }
            }
        }
    }
}
```

**Recommended approach for `/upgrade-dep`**: The skill works *with* guardrail hooks, not around them. The hook validates the package being upgraded (checks for vulnerabilities, age, etc.) and either allows or blocks. If the target version has a known vulnerability, the hook blocks the install and the skill reports the finding.

### 4.7 Safety Model (Five Layers)

1. **Skill-level**: `disable-model-invocation: true` prevents autonomous triggering of side-effect operations
2. **Tool-level**: `allowed-tools: Bash(gdev *)` scopes what commands Claude can run
3. **gdev-level**: `--dry-run`, `--non-interactive` flags provide safety within gdev itself
4. **Permission-level**: Claude Code's deny rules and auto mode classifier provide a backstop
5. **Hooks-level**: PreToolUse hooks intercept and validate commands before execution (enterprise via managed settings)

Read-only operations (doctor, status, list) are autonomous. Side-effect operations (init, enable, disable, update) require explicit user invocation. System-level changes (setup) require additional confirmation even after user invocation.

### 4.8 Prior Art

| Source | Key Takeaway for gdev |
|--------|----------------------|
| **Trail of Bits skills** (35+ plugins, 5.1k stars) | Plugin marketplace distribution, category organization, parallel worker pattern, meta-skills for skill improvement |
| **Trail of Bits claude-code-config** | Opinionated team defaults, anti-rationalization stop hook, "expertise in agents, procedures in skills" principle |
| **Security Phoenix** (AppSec workflows) | Graduated security tiers (4 levels: $0.05 to $10), 12-role pipeline, hook integration for session lifecycle |
| **awesome-claude-code-toolkit** (135+ agents) | Taxonomy of agent categories, role definitions (narrow specialist > broad generalist) |
| **Cursor rules** (13 categories) | Content patterns transfer but need decomposition into Claude Code's multi-file architecture |
| **Anthropic official docs** | PR summary skill as dynamic state injection pattern, codebase-visualizer as script bundling pattern |

---

## 5. Phase 15: Health, Status & Compliance -- Research Findings

**Source spike**: `gdev-health-reporting`

### 5.1 Scorecard-Inspired Posture Scoring

A three-layer assessment model evaluating independent dimensions:

**Layer 1: Defense Coverage (40% weight)** -- What percentage of applicable security layers are enabled and correctly configured? Each of gdev's 10 defense layers has a status (enabled/partial/disabled/not-applicable) and a weight (critical 10x, high 7.5x, medium 5x, low 2.5x), following Scorecard's risk-weighted model.

**Layer 2: Configuration Health (30% weight)** -- Are generated configs current, intact, and matching the latest gdev version? Each config file is tracked as current/modified/outdated/missing/corrupt, with categories (machine-owned, human-edited, exclusive) determining expected behavior.

**Layer 3: Dependency Health (30% weight)** -- Are lock files present and valid? Are there known vulnerabilities? Per-ecosystem tracking of lock file state, vulnerability counts by severity, age-gate status, and last scan time.

**Aggregate score**: 0-100 with letter grades (A: 90-100, B: 75-89, C: 60-74, D: 45-59, F: <45). The three sub-scores are weighted: defense 40%, config 30%, deps 30%.

**Complete Go type definitions** are provided in the research, including `PostureReport`, `AggregateScore`, `ConformanceResult`, `DefenseCoverage`, `ConfigHealth`, `DependencyHealth`, `ToolStatus`, and `EcosystemStatus` structs.

### 5.2 Conformance Labels

In addition to the numeric score, a binary conformance track (following Scorecard v6's evolution):

**Baseline** (minimum acceptable posture):
- Lock files present for all detected ecosystems
- Pre-commit hooks installed
- No critical vulnerabilities
- CLAUDE.md generated sections present
- settings.json deny rules present
- All Tier 1 defense layers enabled

**Enhanced** (consulting firm standard):
- Age-gating configured for all supported ecosystems
- Zero high-severity vulnerabilities
- SAST enabled (Semgrep)
- Secrets scanning enabled (Gitleaks)
- License compliance enabled
- CI workflows generated

**Custom**: Per-project overrides via `.gdev-policy.yaml`.

Conformance provides the binary "does it pass or not?" answer for CI gates and compliance checks. The numeric score provides the nuanced "how good is it?" answer for monitoring and improvement.

### 5.3 Six-Category Drift Detection

All detection is local-only and completes in under 100ms:

| Category | Detection Method | Speed |
|----------|-----------------|-------|
| 1. Unauthorized file modification | SHA256 hash comparison against stored hash | < 10ms/file |
| 2. Version drift | Compare project's `gdev_version` against binary version | < 1ms |
| 3. Tool availability drift | Compare enabled tools against applicable tools for detected ecosystems | < 50ms |
| 4. Section marker integrity | Parse human-edited files for expected marker pairs | < 10ms/file |
| 5. Lock file drift | Compare lock file mtime against source manifest mtime | < 5ms/file |
| 6. Pre-commit hook drift | Verify hook files exist, reference correct runner, match config | < 20ms |

Drift detection builds on gdev's existing SHA256 hash tracking from the migration strategy. State is stored in `.gdev/state.yaml` (committed to git) with per-file hashes, generation timestamps, gdev versions, file categories, and section marker inventories.

**Remediation matrix**: Machine-owned files can be auto-regenerated. Section markers can be auto-repaired. Pre-commit hooks can be auto-reinstalled. Lock files require user action (run package manager). Human-edited file conflicts require user merge.

### 5.4 Machine-Readable Output

**Priority order**:

1. **JSON** (canonical, versioned schema) -- All other formats derive from this. `schemaVersion` field at top level with semantic versioning. Breaking changes increment major version. Consumers should ignore unknown fields. Full PostureReport JSON schema is specified in the research.

2. **Terminal text** (developer primary interface) -- Progressive disclosure: `--quiet` (score only), default (summary with section scores), `--verbose` (per-check detail), `--fix` (remediation commands).

3. **SARIF 2.1.0** (GitHub Code Scanning) -- Maps discrete findings (disabled defenses, missing configs, vulnerabilities) to SARIF results with rule IDs (`gdev/defense-disabled`, `gdev/config-outdated`, `gdev/vuln-high`, etc.). Does NOT carry aggregate scores -- SARIF is for findings, not posture.

4. **Exit codes with `--audit-level`** -- Exit 0 (pass), 1 (findings at/above threshold), 2 (gdev error). Threshold accepts: `none`, `critical`, `high`, `moderate`, `low`, `info`, `any`.

5. **Badge JSON** (shields.io endpoint) -- Minimal JSON with label, message, and color. Color mapping: brightgreen (A), green (B), yellow (C), orange (D), red (F).

6. **JUnit XML** (optional, for Jenkins/GitLab CI) -- Each defense check, config check, and conformance requirement becomes a test case.

**Lessons from prior art**: Version the JSON schema from day one (cargo-audit's unstable JSON broke downstream tools). Never paywall machine-readable output (Safety CLI's JSON-behind-API-key blocks automation). Support `NO_COLOR` standard. Exit codes match severity thresholds (universal CI gate pattern). Include remediation hints. Distinguish "disabled by choice" from "disabled by oversight."

### 5.5 Team Aggregation via CI Artifacts

**Recommended architecture**: CI artifact aggregation (no new infrastructure).

Each project's CI pipeline generates `qsdev status --json > posture.json` as a build artifact. A separate aggregation job collects artifacts across repos and generates the team report. This follows the scorecard-monitor pattern.

**Team report outputs**:
- Markdown summary dashboard with project score table, conformance rates, total vulnerability counts, trend tracking, attention-required alerts
- JSON aggregation format with per-project scores, trends, and alerts
- Auto-generated GitHub issues when scores drop (scorecard-monitor pattern)

**Scaling**: 10 projects (local multi-directory scan works), 50 projects (CI artifact aggregation comfortable), 100+ projects (consider lightweight server). For a consulting firm, 10-50 is the realistic range.

### 5.6 Prior Art Analysis

| Tool | Key Pattern Adopted | Key Anti-Pattern Avoided |
|------|--------------------|-----------------------|
| OpenSSF Scorecard | Risk-weighted scoring, dual-track (numeric + conformance), badge generation, multi-repo aggregation | -- |
| npm audit | `--audit-level` threshold, summary-then-detail UX, severity tiers | -- |
| cargo audit | Advisory-centric display, dependency tree visualization | Unstable JSON schema |
| Safety CLI | Severity model, remediation recommendations | JSON behind API key |
| govulncheck | Call-graph analysis (reachability-aware), VSA mode | -- |
| OWASP Dependency-Check | CVSS scoring, multi-format output | Reports too verbose for interactive use |
| flutter doctor | Check/cross/warn indicators, hierarchical display, `-v` for details | -- |

---

## 6. Phase 16: Developer Experience Polish -- Research Findings

**Source spike**: `gdev-dx-polish`

### 6.1 `qsdev repair` (Conservative Self-Healing)

A companion to `qsdev devenv doctor` (read-only diagnostic):

- `qsdev devenv doctor` diagnoses (never modifies files)
- `qsdev repair` fixes what can be safely fixed automatically

**Auto-fix rules**:
- Machine-owned files with no user edits detected: regenerate
- Machine-owned files WITH user edits: backup + regenerate only with `--force`
- Pre-commit hooks: always safe to reinstall
- Section markers: safe to restore
- Lock files: never auto-update (may change dependency versions)
- devenv.nix: **NEVER auto-modify** (established principle from the implementation plan)

**Design principles**: Doctor is read-only, repair is write. Repair is conservative by default (`--force` for aggressive). Always backup before overwriting. Hash tracking (existing SHA256 system) is the foundation for all corruption detection. Exit codes matter for CI.

Four failure categories mapped with detection and recovery strategies: Nix/devenv failures, generated config corruption, tool/package failures, and environment drift.

### 6.2 `qsdev info` (Lightweight Status)

Subsecond response, no evaluation or checks -- just reads cached state and displays it:

```
$ gdev info
Project: acme-frontend
Ecosystems: typescript (pnpm), docker
Security: consulting-default profile (6 tools active)
devenv: v2.1.0, shell healthy
gdev: v1.2.0 (config current)
```

Useful for "where am I? what's active?" without the overhead of `qsdev status` (which runs health checks).

### 6.3 `qsdev outdated` (Cross-Ecosystem Freshness)

A thin wrapper, not a full aggregator. For each detected ecosystem:
1. Run the native outdated command (`npm outdated`, `pip list --outdated`, `go list -m -u all`, etc.)
2. Print results sequentially with ecosystem headers
3. Exit with non-zero if any ecosystem has outdated deps

Does NOT parse/normalize output formats, provide a unified table, track versions itself, or duplicate Renovate's analysis. This is ~50 lines of Go per ecosystem. The value is "one command to check everything" without the complexity of a "unified dependency analysis platform."

### 6.4 `qsdev update` (Coordinated Updates)

Coordinates gdev's own managed artifacts in 3 steps:
1. Check for gdev binary updates (self-update)
2. Regenerate configs for new version (devenv.nix, settings.json, etc.)
3. Update devenv inputs (`devenv update`)

Step 4 (application dependencies: npm, pip, cargo) is explicitly excluded -- that is Renovate's domain. `qsdev update` is the "keep gdev infrastructure current" command, not a general-purpose dependency updater.

### 6.5 `qsdev teardown` (Clean Exit with 3 Profiles)

| Profile | When | Archive | Nix GC | Token Revocation | Evidence |
|---------|------|---------|--------|-----------------|---------|
| `--quick` | POC/spike end | No | Yes | Yes | No |
| (default) | Normal engagement end | Yes | Optional | Yes | Optional |
| `--compliance` | Regulated client end | Yes | Optional | Yes | Yes (required) |

Teardown actions: archive config, remove devenv environment (Nix store GC), revoke MCP tokens, remove from trusted paths, generate teardown evidence (optional). Does NOT delete the git repository, revoke cloud credentials, or clear browser state.

Archives preserve `.qsdev.yaml`, internal state, last compliance report, tool versions, and devenv.lock for re-engagement. Re-engagement workflow: `git clone`, `cd`, `qsdev init` detects archive and offers restore + migration.

### 6.6 Git Workflow Automation

Four features to include, two to exclude:

**Include**:

| Feature | Value | Complexity | Implementation |
|---------|-------|-----------|---------------|
| Branch naming enforcement | High | Low | Regex hook in devenv.nix git-hooks. Default: `^(feat\|fix\|chore\|docs\|refactor\|test\|ci)/[a-z0-9-]+$` |
| PR template generation | Medium-High | Very Low | Static `.github/pull_request_template.md` with ecosystem-aware sections |
| Commit ticket extraction | Medium | Low | `prepare-commit-msg` hook extracting ticket from branch name (opt-in) |
| Automated PR labels | Medium | Low | Generate `.github/labeler.yml` + workflow for path-based and size-based labels |

**Exclude**:
- Merge queue configuration -- repository settings, not file generation; Terraform has GitHub providers
- Release automation -- git-cliff + commitlint cover the hard parts; full release automation (semantic-release) adds significant complexity and was already rejected in Phase 12 research

### 6.7 Shell Integration

**Include**:
- **Starship config generation** (opt-in) -- Generate `starship.toml` with gdev project context via devenv's native `starship.enable`
- **gdev env vars in devenv.nix** -- `QSDEV_PROJECT_NAME`, `QSDEV_SECURITY_PROFILE`, `QSDEV_VERSION`, `QSDEV_ECOSYSTEMS` (enables any prompt tool, not just starship)
- **devenv enterShell notification** -- One-line "gdev project: acme-frontend" on shell entry via devenv's enterShell task

**Exclude**:
- Shell aliases/abbreviations -- deeply personal, cognitive overhead, gdev commands are already short
- Separate gdev shell hook -- duplicates devenv's hook, creates double-activation and ordering issues

### 6.8 OTEL for Agentic Sessions

Claude Code has native OTEL support (April 2026). Include as optional, profile-driven configuration for consulting firms with client billing needs:
- Generate OTEL environment variables pointing at the firm's collector
- Profile-gated (not default)
- Do NOT ship OTEL infrastructure (collector, Grafana) -- that is infrastructure ops

---

## 7. Rejected Features & Rationale

Ten features explicitly rejected with rationale. A three-test decision framework was established: (1) Is there a purpose-built tool? (2) Is it file generation or runtime behavior? (3) Does it compound with existing features?

| # | Feature | Rejection Rationale | Purpose-Built Alternative |
|---|---------|--------------------|-----------------------|
| 1 | Built-in task runner | devenv 2.0+ has a full-featured task system with parallel execution, dependency ordering, lifecycle hooks, and caching | devenv tasks |
| 2 | Docker/container management | devenv 2.0 has built-in process manager with restart policies, readiness probes, dependency ordering | Docker Compose, Podman, devenv containers |
| 3 | CI/CD pipeline execution | Local CI is a deep, complex problem (secrets, service containers, matrix strategies) | nektos/act (56k+ stars) |
| 4 | Deployment automation | Orthogonal to dev environment, requires infrastructure knowledge that varies enormously | Terraform/OpenTofu, Pulumi, ArgoCD |
| 5 | Project scaffolding / code generation | Every ecosystem has its own scaffolding tool; maintaining 27+ templates would drift from upstream | `cargo init`, `go mod init`, `dotnet new`, etc. |
| 6 | IDE/editor config beyond Claude Code | Deeply personal, highly variable; Claude Code is special because gdev's security model requires specific configuration | Per-editor native settings |
| 7 | Full OTEL infrastructure | Running Prometheus + Loki + Grafana is infrastructure ops, not dev env config | claude-code-otel, Grafana Cloud |
| 8 | Package manager installation | devenv.nix already declares which packages are available; Nix provides them | devenv/Nix |
| 9 | Git server API integration | API integration is a maintenance nightmare (auth, versioning, rate limits); gdev generates files, not API calls | Terraform GitHub/GitLab providers |
| 10 | Vulnerability database | OSV.dev, GitHub Advisory Database, and NVD are maintained by dedicated security organizations | OSV Scanner (already integrated in Phase 5) |

**Additional rejections from specific topics**:
- **MCP server infrastructure** -- gdev is not an MCP server; skills are the right integration pattern for CLI wrapping
- **Custom skill marketplace** -- Premature; embed skills in binary via `embed.FS`, evaluate marketplace distribution later
- **Agent performance monitoring** -- Out of scope; Claude Code's native OTEL covers session-level telemetry
- **Unified dependency analysis** -- Too complex; thin wrappers around ecosystem-native commands are sufficient
- **Breaking change detection** -- Per-ecosystem tools do this; Renovate's major/minor PR splitting is the right approach
- **Shell aliases** -- Deeply personal, cognitive overhead exceeds the 4-character savings
- **Separate gdev shell hook** -- Duplicates devenv hook, creates conflicts

---

## 8. Design Decisions

Key architectural choices made across all five research spikes:

### 8.1 Three-Layer Config with Security Floor

The configuration hierarchy (binary -> project -> local) draws from mise/proto for the layering pattern and Terraform for the version constraint. The critical gdev-specific addition is that **client security levels act as a floor that cannot be lowered by project or local overrides**. This ensures that a HIPAA client's strict security requirements cannot be bypassed by a developer's local config.

### 8.2 Skills Over Commands

Skills (`.claude/skills/*/SKILL.md`) replace the legacy command format (`.claude/commands/*.md`). Skills support directory structure for supporting files, full frontmatter control, auto-invocation by Claude, and live change detection. They also match gdev's `embed.FS` pattern -- skills are embedded in the binary and deployed during `qsdev init`.

### 8.3 Context Budget Management

Generated config stays under 5% of the context window. This is achieved through the five-layer context architecture:
- Always-on CLAUDE.md is 50-100 lines, not a kitchen sink
- Rules use `paths:` frontmatter for lazy loading (zero cost when not working with matching files)
- Skill descriptions are concise (200 chars max) and keyword-rich
- Agents have zero main-context cost (own context window)
- Hooks and settings.json have zero context cost

Model-aware generation adjusts the configuration for Sonnet (200k, tight budget) vs Opus (1M, generous budget).

### 8.4 Conservative Repair

`qsdev repair` follows strict rules: never touch `devenv.nix` (human-edited, too variable), always backup before overwriting, only auto-fix unambiguously safe changes, require `--force` for aggressive repair. This prevents the tool from destroying user customizations, which would erode trust and cause engineers to avoid using it.

### 8.5 Compliance Evidence Without New Infrastructure

Instead of building a compliance platform, gdev generates evidence reports that map its security controls to compliance frameworks (SOC2, HIPAA, ASVS). The reports are machine-readable JSON, integrity-verified via gdev binary version hash. Team aggregation uses CI artifacts (each project produces posture JSON as a build artifact), not a central server. This "no new infrastructure" principle keeps gdev lightweight and deployable.

### 8.6 Profile-Driven OTEL (Not Default)

Claude Code's native OTEL support is exposed as a profile-gated configuration option, not a default. OTEL environment variables are generated when the consulting firm's profile enables it. The OTEL infrastructure (collector, storage, dashboards) is explicitly out of scope -- gdev generates config, not infrastructure.

### 8.7 "Encode Expertise in Agents, Procedures in Skills"

From Trail of Bits, this principle governs the entire skill/agent taxonomy:
- **Agents** = specialized personas that need isolated context (security-reviewer, codebase-explorer)
- **Skills** = repeatable procedures with clear steps (/review-pr, /add-tests, /upgrade-dep)

Agents accumulate persistent memory across sessions (useful for consulting, where codebase knowledge builds over an engagement). Skills run in the main context with structured checklists and verification steps.

### 8.8 Precision Guardrails Over Broad Restrictions

Deny rules target specific dangerous operations (`npm install *`), not broad tool categories (`npm *`). This ensures that workflow skills (`npm test *`, `cargo build *`) are not blocked by security guardrails. A conflict detection test validates compatibility at `qsdev init`/`qsdev update` time.

### 8.9 JSON Schema Versioned From Day One

Learning from cargo-audit's unstable JSON that broke downstream tools: every machine-readable output includes a `schemaVersion` field at the top level. Breaking changes increment major version. New fields are minor version additions. Consumers should ignore unknown fields. This is an immutable design decision -- retroactive schema versioning is much harder than doing it from the start.

### 8.10 Dual-Track Evaluation (Score + Conformance)

Following Scorecard v6's evolution: numeric scores (0-100, A-F) provide nuanced posture assessment for monitoring and improvement. Conformance labels (baseline/enhanced PASS/FAIL) provide binary compliance checks for CI gates and audit evidence. Both are needed because different audiences need different things -- developers want to see progress, compliance teams want pass/fail.

---

## 9. Open Questions

### Phase 13 (Configuration & Team Standards)

- **Remote config extension**: Should `.qsdev.yaml` support an `extends` field referencing a remote config repo (like Renovate's preset repos)? This would allow cross-repo standardization without recompiling the binary, but adds network dependency and complexity.
- **Monorepo support**: How should gdev handle monorepo scenarios where subdirectories need different configurations? The current design operates at the project root only.
- **Cryptographic signing**: Should compliance evidence reports be cryptographically signed for non-repudiation, or is integrity verification (hash-based) sufficient?
- **Profile distribution**: What is the right mechanism for distributing new profiles without requiring a full binary rebuild? Approaches: WASM plugins, HTTPS-fetched profile registry, or accept that binary updates are the distribution channel.

### Phase 14 (Claude Code Integration & Agentic Skills)

- **Plugin marketplace**: Should gdev ship as a Claude Code plugin (installable via `/plugin marketplace add`) in addition to embedding skills during `qsdev init`?
- **Existing addon interaction**: How should gdev skills interact with the existing claudecode addon's hooks and deny rules?
- **Interactive wizard skill**: Should there be a `/gdev-wizard` skill providing an interactive chat-based alternative to the huh form wizard?
- **Skill versioning**: How should gdev version its embedded skill library for updates? (embed.FS copies vs git-based remote library)
- **Auto-invocation false positives**: What is the practical false-positive rate of auto-invoked skills in consulting contexts?
- **Agent memory lifecycle**: How should agent memory be handled across engagement completion (archive vs delete)?
- **Agent teams**: Should gdev generate agent team configurations, or is that too experimental (v2.1.32+, Opus 4.6 required)?
- **Model-based differentiation**: How to handle consulting firms that use Sonnet (200k) vs Opus (1M) differently?

### Phase 15 (Health, Status & Compliance)

- **Scoring weight configurability**: Should the scoring weights (defense 40%, config 30%, deps 30%) be configurable per-profile, or hardcoded with escape hatch?
- **First-run state**: How should `qsdev status` handle the first run before any tools are enabled (all zeros vs "not initialized" state)?
- **Vulnerability scan caching**: Should scan results be cached in `.gdev/cache/` (gitignored) or regenerated on every `qsdev status --scan`?
- **JUnit value**: Is JUnit output worth building, or does SARIF cover the CI integration need sufficiently?
- **Team report command**: Should `qsdev team-report` be a separate binary/command or a subcommand of gdev?

### Phase 16 (DX Polish)

- **Project discovery**: Should `qsdev projects` scan a configurable list of directories or discover projects automatically? (Auto-discovery could be slow on large filesystems.)
- **OTEL collector recommendation**: Should gdev's OTEL profile include a recommended free-tier collector (Grafana Cloud free) or only support self-hosted?
- **Repair and tool lifecycle**: How does `qsdev repair` interact with the tool lifecycle system (`qsdev enable/disable`)? Should repair reinstall disabled tools?

---

## 10. Source Spikes

| # | Spike | Location | Topics | Reports |
|---|-------|----------|--------|---------|
| 1 | gdev Claude Code Integration | `/home/colin/Repos/research/research-spikes/gdev-claude-code-integration/` | Skill format, operation mapping, CLI wrapper patterns, CLAUDE.md integration, safety model, skills vs alternatives | 1 comprehensive report (9 sections) |
| 2 | gdev Health & Compliance Reporting | `/home/colin/Repos/research/research-spikes/gdev-health-reporting/` | Prior art survey, compliance posture model, status command UX, machine-readable output, drift detection, team reporting, badge generation | 7 detailed reports |
| 3 | gdev Agentic Workflow Patterns | `/home/colin/Repos/research/research-spikes/gdev-agentic-workflows/` | Workflow skills catalog, agent files format, task templates, context management, guardrail integration, prior art survey, consulting differentiation | 7 detailed reports |
| 4 | gdev Team Configuration & Onboarding | `/home/colin/Repos/research/research-spikes/gdev-team-config-onboarding/` | Team config sharing models, developer onboarding workflow, config versioning/drift, standards enforcement (CI), prior art survey, consulting lifecycle | 6 detailed reports |
| 5 | gdev DX Polish & Observability | `/home/colin/Repos/research/research-spikes/gdev-dx-polish/` | Observability, task runner, git workflows, environment switching, error recovery, shell integration, dependency freshness, what NOT to include | 8 detailed reports |

All source material (web fetches, documentation snapshots) is preserved in each spike's `docs/` directory. Detailed per-topic analysis reports are in each spike's root directory as `*-research.md` files.
