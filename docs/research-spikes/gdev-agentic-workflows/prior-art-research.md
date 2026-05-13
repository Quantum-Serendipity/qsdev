# Prior Art: Claude Code Workflow Ecosystems

## Executive Summary

The Claude Code ecosystem has rapidly matured into a rich landscape of skills, agents, commands, hooks, and configuration patterns. Four major open-source projects provide directly applicable prior art for gdev's workflow library: Trail of Bits skills (security-focused, 35+ plugins), Trail of Bits claude-code-config (opinionated team defaults), Security Phoenix skills (AppSec workflow orchestration), and the awesome-claude-code-toolkit (comprehensive categorization of 135+ agents). Cross-pollination from Cursor rules collections reveals transferable patterns that need decomposition into Claude Code's multi-file architecture.

## 1. Trail of Bits Skills Repository

**Source**: [github.com/trailofbits/skills](https://github.com/trailofbits/skills) | 5.1k stars | CC-BY-SA-4.0

The most mature and widely-adopted Claude Code skills collection, focused on security research and auditing. 35+ plugins across 10 categories.

### Key Patterns for gdev

**Plugin structure**: Each skill is a self-contained directory registered as a Claude Code plugin via marketplace. Installation: `/plugin marketplace add trailofbits/skills`.

**Category organization**:
- Code Auditing (14 plugins): differential-review, static-analysis, variant-analysis, supply-chain-risk-auditor
- Verification (5): mutation-testing, property-based-testing, constant-time-analysis
- Development (9): ask-questions-if-underspecified, modern-python, git-cleanup, workflow-skill-design
- Infrastructure (1): debug-buttercup (Kubernetes debugging)

**Standout patterns**:
- `ask-questions-if-underspecified`: Requirement clarification skill that improves output quality by making Claude ask before building. Directly applicable to consulting workflows where requirements are often ambiguous.
- `differential-review`: Security-focused code change review. Uses parallel workers for different review angles. Model for gdev's code review skills.
- `skill-improver`: Meta-skill that iteratively refines other skills. Useful for consulting teams tuning workflows to client projects.
- `second-opinion`: External LLM code reviews. Cross-model verification pattern.
- `workflow-skill-design`: Design patterns for creating effective skills. Meta-knowledge about skill authoring.

**Notable achievement**: The `constant-time-analysis` skill found a real timing side-channel vulnerability in RustCrypto's ML-DSA signing, demonstrating that well-designed skills produce production-quality security findings.

### Transferable to gdev
- Plugin marketplace distribution model
- Category-based skill organization
- Parallel worker pattern for multi-angle review
- Meta-skills for skill improvement and design

## 2. Trail of Bits claude-code-config

**Source**: [github.com/trailofbits/claude-code-config](https://github.com/trailofbits/claude-code-config)

Opinionated defaults for Claude Code team deployment. This is the closest existing analog to what gdev generates.

### Configuration Patterns

**Global CLAUDE.md** (`~/.claude/CLAUDE.md`):
- Development philosophy rules (no speculation, no premature abstraction)
- Code quality limits (function length, complexity, line width)
- Per-language toolchain specifications (Python: uv, ruff, ty; Node: oxlint, vitest; Rust: clippy, cargo deny)
- Testing methodology and code review order

**Settings.json**:
- `enableAllProjectMcpServers: false` -- prevents malicious repo MCP servers from auto-loading
- `alwaysThinkingEnabled: true` -- persistent extended thinking
- `cleanupPeriodDays: 365` -- retain year of history for `/insights`
- Privacy controls: disable telemetry, error reporting, feedback surveys
- Deny rules for SSH keys, cloud credentials, shell config files

**Hooks**:
- PreToolUse: Block `rm -rf`, block direct pushes to main/master
- Stop: Anti-rationalization gate using prompt-type hooks with fast model to catch incomplete work

**Custom Commands** (three standout patterns):
- `/review-pr [number]`: Multi-agent PR review with parallel evaluation and auto-fixes
- `/fix-issue [number]`: Full autonomy loop -- research, plan, implement, test, create PR, self-review, fix findings
- `/merge-dependabot [repo]`: Batch overlapping Dependabot PRs with parallel evaluation and sequential merge

### Key Principles (directly applicable to gdev)
1. Scope work to single sessions (prevents context compaction degradation)
2. Pair permissions bypass with sandboxing (never `--dangerously-skip-permissions` without OS isolation)
3. **Encode expertise in agents, procedures in commands** -- skills teach thinking frameworks; agents run specialized personas
4. Use hooks for structured decision intervention (more powerful than system prompts)
5. Prefer fresh sessions over compaction (lossy compression degrades reasoning)

### Transferable to gdev
- The entire settings.json configuration pattern
- Global CLAUDE.md template with per-language toolchain specs
- Command vs agent distinction ("expertise vs procedure")
- Anti-rationalization stop hook pattern
- `/fix-issue` full-autonomy loop as a template for consulting workflows

## 3. Security Phoenix Skills

**Source**: [github.com/Security-Phoenix-demo/security-skills-claude-code](https://github.com/Security-Phoenix-demo/security-skills-claude-code) | MIT

Security automation toolkit with skills, plugins, and a 12-role pipeline.

### Security Assessment Suite

Four graduated security review skills:
- `/security-0day [base-ref]`: End-of-cycle diff scanning (~$0.05-$0.20) -- lightweight, daily use
- `/security-review [scope]`: Pre-merge endpoint/auth/render checks -- pre-PR use
- `/security-assessment [scope]`: Full OWASP Top 10 + ASVS Level 1 (~$8-$10) -- periodic deep review
- `/threatmodel [scope]`: STRIDE + DREAD threat modeling -- architecture-level

**Hook integration**: The `--full` install preset adds:
- SessionStart: Fingerprints project, runs dependency audit
- PreToolUse on Bash: Gates package manager invocations
- PostToolUse on Edit: Pattern scans for SQLi, XSS, hardcoded secrets
- SessionEnd: Reminds to run `/security-0day` for unscanned changes

### Phoenix Pipeline (12-Role Feature Descriptor)

A 12-role specification system for producing security-aware product requirements. Each role is a dedicated skill file: Context Curator, Scope Cutter, Constraint Distiller, Requirements Engineer, Ambiguity Hunter, Security Engineer, Contract Architect, Verification Matrix, Batch Planner, Final Gate, Pipeline Navigator, Orchestrator.

This is the most sophisticated multi-skill orchestration pattern in the wild. Directly applicable to consulting workflows that need structured, auditable output.

### Transferable to gdev
- Graduated skill tiers (lightweight daily → expensive periodic)
- Hook integration pattern for session lifecycle
- 12-role pipeline as model for multi-step consulting workflows
- Domain tier system for research source prioritization

## 4. awesome-claude-code-toolkit

**Source**: [github.com/rohitg00/awesome-claude-code-toolkit](https://github.com/rohitg00/awesome-claude-code-toolkit)

Comprehensive categorization providing the taxonomy for gdev's library design.

### Agent Categories (most relevant for consulting)
- **Quality Assurance (10)**: Code reviewers, test architects, security auditors, performance engineers, accessibility specialists, compliance auditors
- **Developer Experience (15)**: Refactoring specialists, legacy modernizers, dependency managers, documentation engineers
- **Infrastructure (11)**: Incident responders, SRE engineers, deployment specialists
- **Orchestration (8)**: Task coordinators, workflow directors, multi-agent coordinators

### Transferable to gdev
- Taxonomy for organizing generated agents/skills
- Agent role definitions (narrow specialist > broad generalist)
- Orchestration agent patterns for multi-step workflows

## 5. Cursor Rules Cross-Pollination

**Source**: [github.com/PatrickJS/awesome-cursorrules](https://github.com/PatrickJS/awesome-cursorrules) | 13 categories

### Key Insight
Cursor rules are single-file instructions (`.cursorrules` at project root). Claude Code uses a multi-file system: CLAUDE.md + `.claude/rules/*.md` + skills + agents. The **content patterns transfer** but need decomposition into Claude Code's layered architecture.

### Transferable Patterns
- **DevSecOps rules**: Secure coding, secret handling, dependency hygiene, auth patterns → `.claude/rules/security-conventions.md`
- **Anti-over-engineering rules**: "Keep changes scoped, simple, and directly tied to user requests" → CLAUDE.md
- **Dependency verification**: "Run `npm list <package>` before importing, never assume a package exists" → PreToolUse hook
- **Agent boundaries** (2026 update): "NEVER commit without user review, NEVER delete config files without confirmation" → settings.json deny rules
- **Technology-specific conventions**: React patterns, Python practices, Go idioms → `.claude/rules/` with `paths:` frontmatter

### Decomposition Strategy
| Cursor rule type | Claude Code target |
|---|---|
| Always-on conventions | CLAUDE.md or `.claude/rules/*.md` |
| Workflow procedures | `.claude/skills/*/SKILL.md` |
| Safety boundaries | `settings.json` deny rules + hooks |
| Technology patterns | `.claude/rules/*.md` with `paths:` frontmatter |
| Agent behaviors | `.claude/agents/*.md` |

## 6. Community Best Practices

**Source**: [shanraisshan/claude-code-best-practice](https://github.com/shanraisshan/claude-code-best-practice)

### Critical Context Management Findings
- Context degradation begins ~300-400k tokens on 1M model
- "Dumb zone" kicks in at ~40% utilization
- Experienced users target keeping sessions under 30% context utilization
- Implication for gdev: generated CLAUDE.md + rules + skill descriptions must be token-efficient

### Three-Tier Architecture
- **Commands** trigger workflows via slash commands
- **Subagents** provide isolated context and specialized focus
- **Skills** bundle reusable capabilities with progressive disclosure

### Skill Design Best Practices
- Progressive disclosure via folders: `references/`, `scripts/`, `examples/`
- Descriptions as triggers (not summaries) -- the description determines when Claude auto-invokes
- Focus on what pushes Claude out of default behavior -- omit obvious guidance
- Include a "Gotchas" section for failure modes
- `context: fork` for isolated execution

### CLAUDE.md Guidelines
- Target under 200 lines per file (60 lines is conservative)
- Use `.claude/rules/*.md` with `paths:` frontmatter for lazy-loading by file glob
- Put deterministic requirements in `settings.json`, not CLAUDE.md prose (hooks > instructions)

## Depth Checklist

- [x] Underlying mechanism explained (skill/agent/hook/MCP taxonomy, loading behavior)
- [x] Key tradeoffs identified (context cost vs capability, advisory vs enforcement, narrow vs broad agents)
- [x] Compared to alternatives (5 major projects + cursor rules)
- [x] Failure modes described (context bloat, skill misfiring, over-parallelization)
- [x] Concrete examples found (Trail of Bits /review-pr, Security Phoenix graduated tiers, constant-time-analysis real vuln)
- [x] Standalone-readable (sufficient for decisions without visiting original repos)
