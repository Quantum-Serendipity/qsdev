# Awesome Claude Code Security — Curated List
> Source: https://github.com/efij/awesome-claude-code-security
> Retrieved: 2026-05-12

330+ curated resources across 18 categories. This is the most comprehensive security-focused resource list for Claude Code.

## Key Categories and Counts

### Hardening and Permissions
- Trail of Bits claude-code-config — Production defaults: sandboxing, permissions, hooks
- marc-shade/claude-code-security — Progressive hardening: config, hooks, runtime, injection prevention
- affaan-m/everything-claude-code — Performance with security components
- FlorianBruniaux/claude-code-ultimate-guide — Production-ready security templates

### Sandboxing and Isolation (10 entries)
- Arrakis — MicroVM sandbox with backtracking
- microsandbox — Sub-200ms MicroVMs using libkrun
- agent-infra/sandbox — Docker sandbox with browser, shell, MCP
- rivet-dev/sandbox-agent — HTTP-controlled sandboxes
- kubernetes-sigs/agent-sandbox — K8s CRD for sandbox management
- SWE-agent/SWE-ReX — Sandboxed shell for AI agents

### Hooks and Guardrails (7 entries)
- kenryu42/claude-code-safety-net — Destructive git/filesystem command interception
- lasso-security/claude-hooks — Prompt injection defense (50+ patterns)
- disler/claude-code-hooks-mastery — Advanced hook patterns
- carlrannaberg/claudekit — Custom commands, hooks, security utilities
- NVIDIA NeMo Guardrails — Programmable LLM guardrails with Colang

### MCP Security Scanners and Auditors (8 entries)
- Snyk agent-scan — Professional scanner covering 15+ risks
- invariantlabs-ai/mcp-scan — MCP scanner with real-time proxy mode
- cisco-ai-defense/mcp-scanner — Enterprise threat detection
- MCP Security Scanner (SARIF) — GitHub code scanning integration
- AWS MCP Security Scanner — Checkov + Semgrep + Bandit integration
- SecureMCP — OAuth leak and prompt injection audit
- mcpserver-audit — Pre-use safety examination
- MCP Security Audit (npm) — npm dependency auditing

### MCP Gateways and Proxies (6 entries)
- Microsoft MCP Gateway — OAuth 2.0, RBAC, session routing
- Hypr MCP Gateway — OAuth proxy with MCP firewall
- Enkrypt Secure MCP Gateway — Admin-level injection/exfiltration blocking
- Lasso MCP Gateway — Plugin-based sensitive info sanitization
- IBM ContextForge — Registry/proxy for MCP/A2A/REST APIs

### Prompt Injection and Agent Threats
- Check Point CVE-2025-59536 / CVE-2026-21852 — RCE and API key harvesting
- promptfoo — CLI for LLM red-teaming
- Garak (NVIDIA) — 37+ probe modules
- PyRIT (Microsoft) — Enterprise red-teaming
- Rebuff (Protect AI) — Multi-layered injection detection

### Secrets and Data Leakage
- TruffleHog — 800+ secret types with live verification
- Gitleaks — Fast regex/entropy secrets detection
- ggshield (GitGuardian) — 500+ patterns with pre-commit integration
- LLM Guard (Protect AI) — PII detection, toxicity, secrets scanning

### Enterprise Governance
- Microsoft Agent Governance Toolkit — Zero-trust for agents
- GitHub Enterprise AI Controls — Agent control plane with MCP allowlists
- OWASP Top 10 for Agentic Applications (2026)

### Standards and Checklists
- MCP Server Security Standard (MSSS)
- SlowMist MCP Security Checklist
- OWASP AI Security Verification Standard (AISVS)
- NIST AI Risk Management Framework
- MITRE ATLAS
