# gdev Value Proposition Inventory (Through Phase 16.2)

## The One-Liner

**One command to a fully configured, security-hardened development environment with AI-assisted workflows.**

`gdev init` detects your project, generates devenv.nix + Claude Code config + security hardening + pre-commit hooks + CI workflows, and gets you to a working `devenv shell` in under 60 seconds.

---

## Core Value Propositions (by audience)

### For Individual Developers

| Value Proposition | What It Replaces | Concrete Proof |
|---|---|---|
| **Zero-to-productive in one command** | 30-90 minutes of manual devenv.nix authoring, .envrc setup, Claude Code configuration, pre-commit hooks, security configs | `gdev init` → working environment in <60 seconds; `gdev init --profile go-web --yes` → zero questions |
| **27 ecosystem detection** | Manually figuring out which Nix packages, formatters, linters, and security tools each language needs | Detects go.mod, package.json, Cargo.toml, etc. and generates correct configs for each |
| **AI agent that follows the rules** | Unconstrained Claude Code making risky package installs, ignoring project conventions | 48+ deny rules, PreToolUse hooks checking package age + CVEs, skill library with per-ecosystem verification |
| **Reversible tool adoption** | Committing to a tool means manual cleanup if you don't like it | `gdev enable semgrep` / `gdev disable semgrep` — surgically adds/removes from all config files |
| **Self-healing environment** | Diagnosing and manually fixing config drift, broken hooks, missing tools | `gdev doctor` diagnoses, `gdev repair` fixes; `gdev update` brings everything current in one command |
| **Clean project exit** | Manually removing generated files when leaving a project | `gdev teardown` with 3 profiles (quick/default/compliance); compliance mode creates audit trail |

### For Engineering Leadership (CTOs, VPs Eng, Staff+)

| Value Proposition | Business Impact | Concrete Proof |
|---|---|---|
| **Security by default, not by opt-in** | Reduces supply chain attack surface across all projects without developer friction | 6 defense layers work independently; age-gating catches 92% of PyPI malware; install script blocking prevents arbitrary code execution |
| **Consistent security posture** | Every project meets minimum security standards regardless of developer discipline | Three compliance levels (baseline/enhanced/strict) with security floors that local overrides can't weaken |
| **Measurable compliance** | Auditable evidence for SOC2, HIPAA, OWASP ASVS without manual documentation | `gdev evidence` maps defense layers to specific control IDs with SHA256-hashed artifacts; `gdev status` gives 0-100 posture score |
| **Team-wide visibility** | Engineering lead sees all projects' security posture in one place | `gdev team-report` aggregates posture across 10-50 projects; auto-generated GitHub issues for degradation; 90-day trend tracking |
| **2-minute onboarding** | New developer productive from git clone to devenv shell in under 2 minutes | Join mode reads .gdev.yaml, verifies prerequisites, generates local files — no tribal knowledge needed |
| **$0/month infrastructure stack** | Enterprise-grade security tooling without SaaS licensing costs | Nexus Community + Socket Firewall Free, Cachix/Attic, sccache, OSV Scanner, Renovate, Harden-Runner Community — all free tier |
| **Consulting lifecycle management** | Clean client engagement start/end with compliance evidence | Client-specific profiles with compliance levels; `gdev teardown --compliance` creates evidence archive; team aggregation dashboards |

### For Security Teams

| Value Proposition | What It Provides |
|---|---|
| **Defense-in-depth across 6 layers** | Age-gating, install script blocking, lock file enforcement, vulnerability scanning, PreToolUse hook enforcement, hardened Nix evaluation |
| **Every defense is provably working** | Safe test fixtures that trigger each layer (Verdaccio for age-gating, @lavamoat canary for script blocking, known-CVE manifests, AWS example keys for secrets scanning) |
| **CI security pipeline** | SHA-pinned GitHub Actions, Harden-Runner egress monitoring, SARIF reports to GitHub Security tab, least-privilege permissions |
| **No compromised tools in the stack** | Trivy and KICS explicitly replaced (March/April 2026 supply chain compromises); Grype + Checkov as alternatives |

---

## Feature Map by Phase (MVP scope through 16.2)

### Foundation (Phases 1-2)
- Go module with 3-addon architecture (devenv, claudecode, devinit)
- Detection engine (confidence-scored project scanning in <100ms)
- Template engine with Nix-safe generation
- Atomic write pipeline with SHA256 hash tracking
- 8 Tier 1 ecosystem modules (JS/TS, Python, Go, Rust, Java/Kotlin, .NET, Docker, Terraform)

### Core Generation (Phases 3-5)
- devenv.yaml/devenv.nix/.envrc generation with hardened defaults
- Claude Code settings.json with deny rules + 3 permission presets
- CLAUDE.md with section markers for safe updates
- PreToolUse hook (package-guard.py) checking OSV + age
- Per-ecosystem package manager security configs
- Pre-commit hook suite (3 tiers: baseline/enhanced/specialized)
- CI vulnerability scanning workflows

### Orchestration (Phase 6)
- `gdev init` — unified wizard with detection pre-population
- Quick path (2 screens, <5 seconds) or customize (6 screens, <30 seconds)
- Profile system (go-web, ts-fullstack, python-data, rust-cli + custom)
- Complete CLI flag mapping for scriptable/CI usage

### Breadth (Phases 7-8)
- 19 additional ecosystem modules (Tiers 2-4, total 27)
- `gdev init --update` with hash-based modification detection
- Three-way merge for settings.json, section markers for CLAUDE.md
- Team standards versioning (binary update propagates new templates)

### Platform & Distribution (Phases 9-10)
- Cross-platform: macOS (Intel + Apple Silicon), Windows (native + WSL2), 12+ Linux distros
- `gdev doctor` / `gdev setup` — diagnose and fix environment
- Static binary, zero prerequisites, GoReleaser pipeline
- Install scripts (curl | sh, irm | iex), Homebrew/Scoop/APT/RPM
- Self-update with rollback, shell completions for 5 shells

### AI Agent Integration (Phases 11, 14)
- Agent postmortem skill (prevents hallucinated task completion)
- Version-Sentinel (blocks unverified dependency changes)
- Semble semantic code search (98% token savings)
- 10 gdev operation skills (6 user-only, 4 Claude-invocable)
- 7 consulting workflow agents (security-reviewer, onboarding-guide, etc.)
- 8+ workflow skills (/review-pr, /add-tests, /upgrade-dep, etc.)
- Context budget management (model-aware generation)

### Lifecycle & Integrations (Phase 12)
- `gdev enable/disable/status/list` — reversible tool adoption
- Semgrep SAST, Gitleaks secrets scanning, Grype+Syft+Cosign containers
- ScanCode license compliance, SecretSpec dev secrets
- CI workflow generation engine, Context7 MCP, git-cliff changelog

### Configuration & Compliance (Phases 13, 15)
- .gdev.yaml project config with three-layer resolution
- Four onboarding modes (Create/Join/Update/Repair)
- `gdev check` CI enforcement (4 output formats)
- `gdev status` with 0-100 posture scoring (A-F grades)
- 6-category drift detection in <100ms
- `gdev evidence` compliance reports (SOC2, HIPAA, OWASP)
- Team aggregation dashboard with trend tracking

### DX Polish (Phases 16, 16.2)
- `gdev repair` (self-healing), `gdev info` (<100ms project context)
- `gdev outdated` (polyglot dependency freshness), `gdev update` (coordinated 3-stage)
- `gdev teardown` (clean exit with compliance evidence)
- Git workflow automation (PR templates, branch naming, labels)
- Research-backed README as conversion funnel (12 winning patterns)

---

## Key Differentiators vs Alternatives

| Alternative | What gdev adds |
|---|---|
| **Plain devenv.sh** | Detection, security hardening, wizard, migration, 27 ecosystem templates, lifecycle management |
| **mise / asdf** | Security-first (6 defense layers), Claude Code integration, compliance reporting, team configuration |
| **Docker Desktop** | Nix reproducibility without container overhead, AI agent integration, per-ecosystem security configs |
| **Manual Claude Code setup** | 48+ deny rules, PreToolUse hooks, curated skill library, model-aware context budgets, consulting agents |
| **Homebrew / Nix alone** | Project-scoped environments (not system-wide), security hardening, team configuration, compliance evidence |

---

## Quantifiable Claims

- **60 seconds** from clone to working devenv shell (vs 30-90 minutes manual)
- **2 minutes** for returning developer onboarding (Join mode)
- **27 ecosystems** detected and configured
- **6 defense layers** working independently
- **92% of PyPI malware** caught by age-gating alone (published <24h)
- **48+ deny rules** for Claude Code package guardrails
- **0-100 posture score** with A-F grades for instant assessment
- **<100ms** drift detection (all local, no network)
- **$0/month** for the complete infrastructure stack
- **Zero prerequisites** — static binary with embedded templates
- **5 shells** with completions (bash, zsh, fish, PowerShell, nushell)
- **12+ Linux distros** + macOS + Windows/WSL2
