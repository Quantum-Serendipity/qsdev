# Claude Code Security Ecosystem Research

Comprehensive survey of published skills, plugins, hooks, MCP servers, managed settings configurations, and community tools for Claude Code security — focused on package guardrails, supply chain security, code quality enforcement, and security hardening.

Research date: 2026-05-12. Draws from prior spike research (`claude-code-agent-package-guardrails/`, `package-supply-chain-security/`) plus fresh web research.

---

## 1. Official Anthropic Security Tools

### 1.1 Security-Guidance Plugin
- **Repo**: `anthropics/claude-code` → `plugins/security-guidance`
- **What**: Warns about command injection, XSS, and unsafe code patterns when editing files. Uses hooks/ directory for enforcement.
- **Install**: `/plugin install security-guidance@claude-plugins-official`
- **Relevance**: Lightweight, official — good baseline for code quality. Does not cover package installation.

### 1.2 Claude Code Security Review (GitHub Action)
- **Repo**: `anthropics/claude-code-security-review`
- **What**: AI-powered PR security review. Deep semantic analysis: broken access control, business-logic flaws, insecure deserialization, auth bypass, DNS rebinding. Language-agnostic. Advanced false positive filtering.
- **Cost**: ~$0.90-$1.80 per 500-line PR scan (billed to API key).
- **Track record**: Caught vulnerabilities in Anthropic's own codebase including Claude Code itself (RCE via DNS rebinding in a local HTTP server feature).
- **Relevance**: High. This is the CI/CD security review tool. Could be mandated in gdev's GitHub Action templates.

### 1.3 Claude Code Security (Enterprise Feature)
- **Status**: Limited research preview for Enterprise and Team customers (as of Feb 2026).
- **What**: First-party vulnerability scanning feature. Multi-stage verification. Separate from the GitHub Action.
- **Relevance**: Monitor for GA release. May become the primary enterprise scanning solution.

### 1.4 Built-in Security Architecture
The hooks/permissions/sandbox/managed-settings system is the foundation. Key enforcement primitives:

| Mechanism | Nature | Bypass Resistance | Package Security Role |
|-----------|--------|-------------------|-----------------------|
| PreToolUse hooks | Deterministic, pre-execution | High (fires before permission checks) | Primary enforcement: parse commands, query vuln APIs, block/rewrite |
| Permission deny rules | Deterministic, glob matching | Medium (shell wrappers bypass) | Fast catch of obvious install patterns |
| Sandbox (bubblewrap/Seatbelt) | OS-level isolation | Very high | Network domain allowlists, filesystem restrictions |
| Managed settings | Non-overridable enterprise config | Maximum | Deploy hooks/rules/MCP that users cannot disable |
| CLAUDE.md instructions | Advisory, probabilistic | Low | Route installs through safe paths voluntarily |
| Custom skills | Advisory, user-facing workflow | Low | Structured check-decide-install-audit pipeline |

**Managed settings delivery** (Linux): `/etc/claude-code/managed-settings.json` with `managed-settings.d/` drop-in directory. Also via MDM (macOS: Jamf/Kandji, Windows: GPO/Intune) or server-managed via Claude.ai admin console.

### 1.5 Official Skills Repository
- `anthropics/skills` — document-focused skills only (docx, pdf, pptx, xlsx). No security skills. Security is addressed through hooks and permissions, not skills.

### 1.6 Official Plugin Marketplace
~130+ plugins in `anthropics/claude-plugins-official`. Security-relevant entries:

| Plugin | Vendor | What It Does |
|--------|--------|--------------|
| **security-guidance** | Anthropic | XSS/injection warnings on file edits |
| **aikido** | AikidoSec | SAST, secrets, IaC scanning with remediation loop |
| **sonarqube** | SonarSource | 7,000+ rules, 450+ secret patterns, PostToolUse agentic analysis |
| **ai-plugins** | Endor Labs | Supply chain security via endorctl CLI setup |
| **42crunch-api-security-testing** | 42Crunch | API security audits, OWASP vulnerability detection |
| **coderabbit** | CodeRabbit | Code review with 40+ static analyzers |
| **qodo-skills** | Qodo | Shift-left code review: quality, testing, security, compliance |
| **crowdstrike-falcon-foundry** | CrowdStrike | Cybersecurity app development on Falcon platform |
| **hookify** | Anthropic | Custom hook creation for behavior prevention |

---

## 2. Package Security Plugins and Hooks

### 2.1 attach-guard (Purpose-Built Package Guardrail)
- **Repo**: `attach-dev/attach-guard`
- **Mechanism**: PreToolUse hook (not skill, not MCP — deterministic enforcement)
- **Coverage**: npm, pip, Go, Cargo
- **Capabilities**:
  - Socket.dev API for supply chain risk scoring
  - Blocks packages scoring below 50/100
  - Flags packages scoring 50-70/100
  - Rewrites commands to safer versions via `updatedInput`
  - Age-gates packages published within 48 hours
- **Install**: `claude plugin marketplace add attach-dev/attach-guard`
- **Relevance**: **The closest existing implementation to what gdev needs.** Could be forked/extended or used as the reference architecture.

### 2.2 Security Phoenix (Most Comprehensive Suite)
- **Repo**: `Security-Phoenix-demo/security-skills-claude-code`
- **Mechanism**: Skills + hooks combined
- **Coverage**: npm, yarn, pnpm, pip, uv, poetry, cargo, go, gem, bundle, composer, dotnet
- **Capabilities**:
  - PreToolUse: known-malicious blocking, typosquat detection, new package flagging
  - SessionStart: project fingerprinting + dependency audit
  - PostToolUse: pattern scanning for SQL injection, innerHTML, hardcoded secrets
  - SessionEnd: cleanup
  - Four slash commands covering security lifecycle
  - Multi-language pre-merge reviewer with subagent dispatch
- **Install**: `bash "skills/Security Assessment/install/install.sh" --full`
- **Relevance**: **Reference implementation for the full security lifecycle.** More comprehensive than attach-guard but more complex.

### 2.3 harish-garg/security-scanner-plugin
- **Mechanism**: Plugin using GitHub MCP Server
- **What**: Scans code for vulnerabilities using GitHub's advisory database
- **Relevance**: Useful for post-install auditing, not pre-install blocking.

---

## 3. Destructive Command Guards

These prevent `rm -rf`, force pushes, credential exposure — not package-specific but part of the security posture.

| Project | Key Feature | Cross-Agent? |
|---------|------------|--------------|
| **hex/claude-guard** | 3-tier protection (block/redirect/warn), credential detection | No |
| **kenryu42/claude-code-safety-net** | Destructive git/filesystem command interception | Yes (Codex, Gemini CLI, Copilot) |
| **Cocabadger/saferun-guard** | Runtime safety firewall, ~20ms overhead | No |
| **Dicklesworthstone/destructive_command_guard** | High-performance destructive command blocking | Yes (multi-agent) |
| **Boucle bash-guard** | Encoding bypass detection (base64, hex, reversed strings) | No |

---

## 4. Prompt Injection Defense

| Project | Mechanism | Detection |
|---------|-----------|-----------|
| **lasso-security/claude-hooks** | PostToolUse hook scanning tool outputs | 50+ patterns, 5 attack categories, warns rather than blocks |
| **slavaspitsyn/claude-code-security-hooks** | 7-layer defense | Self-protection (blocks settings.json edits), network restrictions, canary files |
| **MG-Cafe/agentic-coder-shield** | 3 defense layers | Prompt injection, data exfiltration, credential leaks |
| **efij/secure-claude-code** | YARA-style guard packs | Secrets, exfiltration, prompt injection, MCP abuse |
| **Boucle enforce-hooks** | Dynamic CLAUDE.md enforcement | Re-reads CLAUDE.md on every tool call, blocks violations |

---

## 5. Code Review and Security Scanning

### 5.1 Trail of Bits Skills (40+ plugins, security-focused)
- **Repo**: `trailofbits/skills`
- **Key security skills**:
  - `supply-chain-risk-auditor` — Dependency threat landscape evaluation
  - `differential-review` — Security-focused code change review with git history
  - `static-analysis` — CodeQL, Semgrep, SARIF parsing
  - `insecure-defaults` — Dangerous configs and hardcoded credentials
  - `variant-analysis` — Find similar vulnerabilities across codebases
  - `sharp-edges` — Risky APIs and security-prone designs
  - `constant-time-analysis` — Timing side-channel detection
  - `zeroize-audit` — Missing secret zeroization in C/C++ and Rust
  - `agentic-actions-auditor` — GitHub Actions AI agent vulnerability detection
- **Relevance**: **Professional-grade security audit skills from a leading security firm.** The `supply-chain-risk-auditor` and `differential-review` are directly relevant to gdev.

### 5.2 Trail of Bits claude-code-config
- **Repo**: `trailofbits/claude-code-config`
- **What**: Production security defaults: deny rules for credentials/secrets paths, hooks for `rm -rf` and force-push blocking, MCP server controls.
- **Philosophy**: "Hooks are not a security boundary" — use OS sandbox as primary boundary, hooks as structured guidance.
- **Relevance**: Reference configuration for the gdev addon's generated settings.json.

### 5.3 Official Marketplace Scanning Plugins
- **Aikido Security**: Real-time SAST + secrets + IaC scanning with auto-remediation loop
- **SonarQube**: 7,000+ rules, PostToolUse agentic analysis, 450+ secret pattern blocking
- **CodeRabbit**: 40+ static analyzers
- **Qodo**: Shift-left code review with org-level coding rules

---

## 6. MCP Servers for Security

| Server | Auth | Coverage | Pre-Install? | Key Tools |
|--------|------|----------|--------------|-----------|
| **Socket.dev MCP** | Free public / API key | npm, PyPI, Cargo | Scoring only | `depscore` (supply chain risk) |
| **Snyk MCP** | Snyk token | Multi-ecosystem | Project scan | `snyk_sca_scan`, `snyk_code_scan`, 6 more |
| **CVE MCP Server** | None | Multi-ecosystem | Dependency check | `scan_dependencies` (OSV.dev), `scan_github_advisories` |
| **Google MCP Security** | Google auth | SecOps/Threat Intel | No | Security operations tools |
| **Contrast Security MCP** | Contrast token | Application security | No | Vulnerability remediation |
| **Codacy MCP** | Codacy token | Multi-language | Post-install | Trivy scanning |
| **AWS MCP Security Scanner** | AWS creds | IaC/code | No | Checkov + Semgrep + Bandit |

**Gap**: No MCP server wraps `install` commands with pre-flight checks. Socket.dev provides scoring; actual enforcement requires hooks. This is the gap attach-guard fills.

---

## 7. Dependency Management Tools

| Tool | What It Does | Mechanism |
|------|-------------|-----------|
| **softaworks/dependency-updater** | Safe semver updates, prompts for major | Claude Code skill |
| **Universal Dependency Installer** | Detects correct package manager | Claude Code skill |
| **Dependency Audit** | Update, clean up, secure dependencies | Claude Code skill |
| **Endor Labs plugin** | Supply chain security via endorctl | Official marketplace plugin |

---

## 8. Artifact Registries and Caches

### artifact-keeper
- **Repo**: `artifact-keeper/artifact-keeper`
- **What**: Open-source universal artifact registry. Drop-in Artifactory/Nexus alternative.
- **Formats**: 45+ (Maven, NPM, PyPI, NuGet, Cargo, Go, Docker/OCI, Helm, Terraform, RPM, Debian, etc.)
- **Security**: Dual scanning (Trivy + Grype), vulnerability grades A-F, policy engine (block/quarantine), artifact signing
- **Stack**: Rust/Axum, PostgreSQL, S3-compatible storage
- **Auth**: JWT, OIDC, LDAP, SAML 2.0, API tokens
- **Extensible**: WASM plugin system for custom formats
- **Relevance**: **Strong candidate for gdev's team artifact cache.** Built-in vulnerability scanning, policy engine, and broad format support. Could cache Nix binary substitutions via custom WASM plugin. Self-hostable.

---

## 9. Enterprise Governance and Hardening

### Published Hardening Guides
- **Anthropic Claude Hardening Guide** (howtoharden.com) — Settings-by-settings configuration guide
- **Trail of Bits claude-code-config** — Production defaults from leading security firm
- **marc-shade/claude-code-security** — Progressive hardening: config, hooks, runtime, injection prevention
- **FlorianBruniaux/claude-code-ultimate-guide** — Production-ready security templates
- **Backslash Security Best Practices** — Comprehensive blog post

### Enterprise Managed Settings
Key fields for organizational enforcement:
```json
{
  "allowManagedHooksOnly": true,
  "allowManagedPermissionRulesOnly": true,
  "allowManagedMcpServersOnly": true,
  "disableBypassPermissionsMode": "disable",
  "disableAutoMode": "disable",
  "disableSkillShellExecution": true
}
```

### Governance Frameworks
- **Microsoft Agent Governance Toolkit** — Zero-trust agent execution
- **GitHub Enterprise AI Controls** — Agent control plane with MCP allowlists and audit logs
- **OWASP Top 10 for Agentic Applications (2026)** — Peer-reviewed standard
- **MCP Server Security Standard (MSSS)** — Open certification with compliance levels
- **CIS MCP Companion Guide** (April 2026) — Governance framework with per-tool capability grants

---

## 10. Curated Resource Lists

| List | Focus | Size | URL |
|------|-------|------|-----|
| **efij/awesome-claude-code-security** | Claude Code security only | 330+ entries | github.com/efij/awesome-claude-code-security |
| **hesreallyhim/awesome-claude-code** | General Claude Code | Large, hand-curated | github.com/hesreallyhim/awesome-claude-code |
| **rohitg00/awesome-claude-code-toolkit** | Comprehensive toolkit | 135 agents, 35 skills, 176+ plugins | github.com/rohitg00/awesome-claude-code-toolkit |
| **Puliczek/awesome-mcp-security** | MCP security | Vulnerabilities, tools, practices | github.com/Puliczek/awesome-mcp-security |
| **ProjectRecon/awesome-ai-agents-security** | AI agent security broadly | Living map | github.com/ProjectRecon/awesome-ai-agents-security |

---

## 11. Known Vulnerabilities and Security Research

### Claude Code-Specific CVEs
- **CVE-2025-59536**: MCP configuration injection — RCE via malicious .mcp.json auto-approving servers
- **CVE-2026-21852**: API key harvesting from Claude Code sessions via malicious hooks
- **50-subcommand bypass** (patched v2.1.90): 50+ chained commands silently disabled deny enforcement
- **Deny-rule non-enforcement** (patched v1.0.93): All deny rules were silently ignored

### Agent-Level Security Research
- Check Point: RCE and API token exfiltration through project files
- Prompt injection via GitHub PR comments (affects Claude Code, Gemini CLI, Copilot)
- VentureBeat: "Three AI coding agents leaked secrets through a single prompt injection"
- Trend Micro: Claude Code lures and GitHub release payload weaponization

---

## 12. Recommendations for gdev Addon

### What to Embed (Minimum Viable Security)
1. **PreToolUse hook script** — Based on existing `reference-hook-script.py` from guardrails spike. Query OSV.dev + age-check + safety flag rewriting.
2. **Permission deny rules** — 48 rules from `reference-deny-rules.md` covering all major package managers + bypass mitigations.
3. **Managed settings template** — For enterprise deployment via `/etc/claude-code/managed-settings.json`.
4. **CLAUDE.md security section** — Advisory routing through safe install paths.
5. **OS-level config** — `.npmrc` (ignore-scripts), `pip.conf` (only-binary), etc. generated into devenv.

### What to Integrate (Official Marketplace)
1. **SonarQube plugin** — PostToolUse agentic analysis with 7,000+ rules.
2. **Aikido plugin** — SAST + secrets + IaC with remediation loop.
3. **Socket.dev MCP** — Zero-config supply chain scoring.
4. **Claude Code Security Review GitHub Action** — CI/CD PR scanning.

### What to Reference (Community Best Practices)
1. **attach-guard** — Reference architecture for the package guardrail hook.
2. **Security Phoenix** — Reference for full security lifecycle (SessionStart through SessionEnd).
3. **Trail of Bits claude-code-config** — Production security defaults.
4. **Trail of Bits skills** — `supply-chain-risk-auditor`, `differential-review` for audit workflows.
5. **Lasso claude-hooks** — Prompt injection defense.

### What to Evaluate (Artifact Registry)
1. **artifact-keeper** — Self-hosted artifact registry with built-in vulnerability scanning. 45+ formats. Could serve as team package cache with policy enforcement.

### What to Watch
1. **Claude Code Security** (Anthropic enterprise feature) — May supersede some of the custom tooling.
2. **RFC #45427** — "Deterministic tool gate" proposal for governance enforcement beyond hooks.
3. **OWASP Agentic Top 10 (2026)** — Emerging standard for agent-specific risks.

---

## Sources

### Raw Source Documents (saved to docs/)
- `docs/official-marketplace-plugins.md` — Full official marketplace plugin list
- `docs/awesome-claude-code-security-list.md` — 330+ entry curated security list
- `docs/trail-of-bits-skills-and-config.md` — ToB skills catalog and production config
- `docs/community-security-plugins-and-hooks.md` — All community guardrail projects
- `docs/anthropic-official-security-tools.md` — Official Anthropic security tools
- `docs/artifact-keeper-registry.md` — Artifact keeper registry details

### Prior Spike Research (in research-spikes/)
- `claude-code-agent-package-guardrails/` — 7 detailed reports + unified architecture spec
  - `hooks-research.md` — Hook mechanism deep dive with bypass analysis
  - `permissions-research.md` — Permission system with known vulnerabilities
  - `mcp-server-research.md` — Socket.dev, Snyk, custom MCP feasibility
  - `custom-skills-research.md` — Skills as workflow layer
  - `claude-md-guardrails-research.md` — CLAUDE.md effectiveness and limitations
  - `vulnerability-apis-research.md` — 12 APIs surveyed, OSV.dev recommended
  - `unified-architecture.md` — Five-layer defense specification with copy-pasteable configs
  - `reference-hook-script.py` — Working Python PreToolUse hook
  - `reference-hook-settings.json` — Complete settings.json template
  - `reference-deny-rules.md` — 1,069-line deny rule reference
- `package-supply-chain-security/` — 7 reports on package manager defenses
  - `quarantine-gates-research.md` — Age-gating (92% of PyPI malware caught within 24h)
  - `install-sandboxing-research.md` — Install script blocking (pnpm v10+ default)
  - `lockfile-integrity-research.md` — Lockfile enforcement mechanisms
  - `signature-provenance-research.md` — Sigstore, PEP 740 provenance
  - `private-registries-research.md` — Private registry architecture
  - `org-tooling-research.md` — Organizational scanning tools
