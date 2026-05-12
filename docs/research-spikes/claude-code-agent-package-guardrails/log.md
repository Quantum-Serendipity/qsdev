# Research Log: Claude Code Agent Package Guardrails

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Investigating how to configure Claude Code's tools, skills, hooks, and settings to prevent agents from installing compromised, vulnerable, or supply-chain-attacked packages. Focus is on configure-once, background-enforced guardrails — not manual review. Related active spikes exist for general package supply chain security and devenv.sh security boilerplate; this spike is specifically about Claude Code's own mechanisms.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-12 — Scope Confirmed & Phase 1 Tasks Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: User confirmed 7-area scope: hooks, permissions, MCP servers, CLAUDE.md instructions, custom skills, vulnerability DB/API integration, and cross-reference with sibling spikes. Created 7 Phase 1 tasks covering all areas. Three high-priority tasks (hooks, permissions, MCP servers) form the core — these are the enforcement mechanisms. Four medium-priority tasks cover the softer layers and data sources.
- **Next**: Begin high-priority research tasks. Start with hooks mechanism (most likely primary enforcement layer) and permissions (complementary restriction layer).

## 2026-05-12 — MCP Server Integration Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code MCP docs](https://code.claude.com/docs/en/mcp) → `docs/claude-code-mcp-configuration.md`
  - [Socket.dev MCP guide](https://docs.socket.dev/docs/guide-to-socket-mcp) → `docs/socket-dev-mcp-server.md`
  - [Socket.dev MCP GitHub](https://github.com/SocketDev/socket-mcp) → `docs/socket-dev-mcp-server.md`
  - [Snyk MCP announcement](https://snyk.io/articles/secure-ai-coding-with-snyk-now-supporting-model-context-protocol-mcp/) → `docs/snyk-mcp-server.md`
  - [Snyk MCP cheat sheet](https://snyk.io/articles/snyk-mcp-cheat-sheet/) → `docs/snyk-mcp-cheat-sheet.md`
  - [Snyk Agent Scan](https://github.com/snyk/agent-scan) → `docs/snyk-agent-scan.md`
  - [Claude Code hooks guide](https://code.claude.com/docs/en/hooks-guide) → `docs/claude-code-hooks-guide.md`
  - [Claude Code hooks reference (MCP tool hooks)](https://code.claude.com/docs/en/hooks) → `docs/claude-code-mcp-tool-hooks.md`
  - [Claude Code permissions](https://code.claude.com/docs/en/permissions) → `docs/claude-code-permissions.md`
  - [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk) → `docs/mcp-typescript-sdk.md`
  - [OSV.dev API](https://google.github.io/osv.dev/api/) → `docs/osv-dev-api.md`
  - [MCP local server connection](https://modelcontextprotocol.io/docs/develop/connect-local-servers) → `docs/mcp-local-server-connection.md`
- **Summary**: Completed comprehensive research on MCP servers for package security validation. Key findings: (1) Two production MCP servers exist — Socket.dev (free public server, supply chain scoring) and Snyk (comprehensive scanning, requires account). (2) Building a custom MCP server is straightforward via the TypeScript SDK, using OSV.dev (free, no auth, no rate limits) as the primary vulnerability database. (3) Critical insight: MCP servers alone cannot enforce security — agents can bypass them via raw Bash. Must combine with permission deny rules (blocking raw install commands) and PreToolUse hooks (intercepting/blocking). (4) Claude Code supports `"type": "mcp_tool"` hooks that can delegate PreToolUse validation to an MCP server, but these fail-open on error — dangerous for security. (5) The recommended architecture is a three-layer defense: MCP server (validation logic) + permission deny rules (deterministic blocking) + PreToolUse hook (sophisticated command parsing). Full report at `mcp-server-research.md`.
- **Next**: Complete remaining high-priority tasks (hooks deep dive, permissions deep dive). The MCP research partially covers both — hooks and permissions docs were fetched as part of understanding the MCP integration layer.

## 2026-05-12 — Claude Code Hooks Mechanism Deep Dive Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code Hooks Reference (Official)](https://code.claude.com/docs/en/hooks) → `docs/claude-code-hooks-reference-official.md`
  - [Claude Code Hooks Guide (Official)](https://code.claude.com/docs/en/hooks-guide) → `docs/claude-code-hooks-guide-official.md`
  - [Claude Code Settings Reference (Official)](https://code.claude.com/docs/en/settings) → `docs/claude-code-settings-reference.md`
  - [Anthropic Bash Command Validator Example](https://github.com/anthropics/claude-code/blob/main/examples/hooks/bash_command_validator_example.py) → `docs/anthropic-bash-command-validator-example.md`
  - [attach-guard Plugin (DEV Community)](https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l) → `docs/attach-guard-plugin.md`
  - [dwarvesf/claude-guardrails](https://github.com/dwarvesf/claude-guardrails) → `docs/dwarvesf-claude-guardrails.md`
  - [rulebricks/claude-code-guardrails](https://github.com/rulebricks/claude-code-guardrails) → `docs/rulebricks-claude-code-guardrails.md`
  - [mafiaguy/claude-security-guardrails](https://github.com/mafiaguy/claude-security-guardrails) → `docs/mafiaguy-claude-security-guardrails.md`
  - [Codacy Guardrails](https://blog.codacy.com/equipping-claude-code-with-deterministic-security-guardrails) → `docs/codacy-claude-code-guardrails.md`
  - [paddo.dev Guardrails](https://paddo.dev/blog/claude-code-hooks-guardrails/) → `docs/paddo-dev-hooks-guardrails.md`
  - [disler/claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery) → `docs/disler-hooks-mastery.md`
- **Summary**: Completed comprehensive deep dive on Claude Code hooks mechanism for package install guardrails. Key findings: (1) PreToolUse hooks are the primary enforcement mechanism — they fire before every tool call, receive the full command string as JSON, and can block via exit code 2 or structured JSON `permissionDecision: "deny"`. (2) Hooks fire BEFORE permission-mode checks and cannot be bypassed even by `--dangerously-skip-permissions`. (3) The `updatedInput` field enables rewriting commands to safer versions (e.g., pinning to a non-vulnerable version). (4) Five hook handler types exist: command, http, mcp_tool, prompt, agent. (5) Enterprise managed settings (`allowManagedHooksOnly`) make hooks mandatory and non-overridable. (6) The `if` field (v2.1.85+) enables efficient filtering by permission rule syntax. (7) Eight bypass vectors identified with mitigations: indirect installation via scripts, obfuscated commands, direct manifest editing, disableAllHooks, package aliasing, post-install scripts, subagent tool calls, and bare `npm install`. (8) The attach-guard plugin is the first purpose-built package guardrail plugin, using Socket.dev API for supply chain scoring. (9) Multiple community guardrail projects exist (dwarvesf, rulebricks, mafiaguy) but none are package-install-specific. Full report at `hooks-research.md`.
- **Next**: Complete permissions deep dive (partially covered — settings hierarchy and interaction with hooks documented). Then medium-priority tasks: CLAUDE.md instructions, custom skills, vulnerability API survey, sibling spike cross-reference.

## 2026-05-12 — Permission Settings & Command Allowlists Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code Configure Permissions (Official)](https://code.claude.com/docs/en/permissions) -> `docs/official-configure-permissions.md`
  - [Claude Code Permission Modes (Official)](https://code.claude.com/docs/en/permission-modes) -> `docs/official-permission-modes.md`
  - [Claude Code Settings Reference (Official)](https://code.claude.com/docs/en/settings) -> `docs/official-settings-reference.md`
  - [Claude Code Hooks Guide (Official)](https://code.claude.com/docs/en/hooks-guide) -> `docs/official-hooks-guide.md`
  - [Claude Code Auto Mode Engineering (Anthropic)](https://www.anthropic.com/engineering/claude-code-auto-mode) -> `docs/anthropic-auto-mode-engineering.md`
  - [5 Permission Patterns (DEV Community)](https://dev.to/klement_gunndu/lock-down-claude-code-with-5-permission-patterns-4gcn) -> `docs/five-permission-patterns-lockdown.md`
  - [Deny Rules Bypass Vulnerability (Adversa)](https://adversa.ai/blog/claude-code-security-bypass-deny-rules-disabled/) -> `docs/adversa-deny-rules-bypass-vulnerability.md`
  - [Deny Rules Bypass (The Register)](https://www.theregister.com/2026/04/01/claude_code_rule_cap_raises/) -> `docs/register-deny-rules-bypass-news.md`
  - [Security Best Practices (Backslash)](https://www.backslash.security/blog/claude-code-security-best-practices) -> `docs/backslash-security-best-practices.md`
  - [attach-guard Package Plugin (DEV Community)](https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l) -> `docs/attach-guard-package-plugin.md`
- **Summary**: Comprehensive research on Claude Code permission settings for restricting agent package installation. Key findings: (1) Permission rules use deny > ask > allow evaluation order with glob-pattern matching on Bash command strings. (2) Five-tier settings hierarchy with managed settings at top (non-overridable). (3) Compound commands are decomposed at shell operators and each subcommand matched independently. (4) Process wrappers (timeout, nice, etc.) are stripped before matching, but shell wrappers (bash -c, env, command) are NOT. (5) Two critical historical vulnerabilities: 50-subcommand bypass (patched v2.1.90) where deny rules silently stopped being enforced on long command chains, and complete deny-rule non-enforcement in v1.0.93. (6) Deny rules alone are insufficient due to bypass via subprocess spawning, shell builtins, and variable expansion. (7) The optimal strategy is three-layer defense: deny rules (catch obvious cases) + PreToolUse hooks (programmatic enforcement) + OS sandbox (restricts what processes can actually do). (8) Hooks can block what rules allow, but cannot allow what rules deny -- this asymmetry is by design. (9) `allowManagedPermissionRulesOnly` in managed settings prevents all user/project permission changes. (10) Auto mode drops broad allow rules (including package-manager commands) on entry and reviews installs via classifier.
- **Next**: Remaining medium-priority tasks: CLAUDE.md instructions, custom skills, vulnerability API survey, sibling spike cross-reference.

## 2026-05-12 — CLAUDE.md Instruction-Based Guardrails Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code CLAUDE.md Documentation (Official)](https://code.claude.com/docs/en/claude-md) → `docs/official-claude-md-documentation.md`
  - [Claude Code Subagents Documentation (Official)](https://code.claude.com/docs/en/sub-agents) → `docs/official-subagents-documentation.md`
  - [Subagent CLAUDE.md Regression Issue #40459](https://github.com/anthropics/claude-code/issues/40459) → `docs/subagent-claudemd-regression-issue-40459.md`
  - [Dive into Claude Code (arXiv 2604.14228)](https://arxiv.org/html/2604.14228v1) → `docs/arxiv-dive-into-claude-code-instructions.md`
  - [dwarvesf/claude-guardrails CLAUDE.md Security Section](https://github.com/dwarvesf/claude-guardrails) → `docs/dwarvesf-claude-md-security-section.md`
- **Summary**: Comprehensive research on CLAUDE.md as an advisory layer for package security. Key findings: (1) CLAUDE.md is explicitly advisory — delivered as a user message, not enforced by the client. Official docs state "no guarantee of strict compliance." (2) Effectiveness is high (~90%+) for specific, concise instructions under normal conditions but degrades under context pressure, competing instructions, and long sessions. (3) Critical gap: subagents do NOT inherit CLAUDE.md — custom subagents get their own prompt, and built-in subagents have omitClaudeMd:true since v2.1.84 (open regression). (4) Prompt injection can override CLAUDE.md instructions. (5) Best patterns: route installs through MCP tools/skills, explain hook behavior so agent doesn't work around blocks, establish approval gates. (6) CLAUDE.md's primary value is reducing enforcement friction — it makes most installs follow the safe path voluntarily so hooks only catch edge cases.
- **Next**: Complete remaining medium-priority tasks: custom skills, vulnerability API survey, sibling spike cross-reference.

## 2026-05-12 — Custom Skills for Secure Package Installation Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code Skills Official Documentation](https://code.claude.com/docs/en/skills) → `docs/claude-code-skills-official-docs.md`
  - [Agent Skills Platform Overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) → `docs/agent-skills-platform-overview.md`
  - [Skills vs Slash Commands (MindStudio)](https://www.mindstudio.ai/blog/claude-code-skills-vs-slash-commands) → `docs/skills-vs-slash-commands-mindstudio.md`
  - [Anthropic Skills Repository](https://github.com/anthropics/skills) → `docs/anthropic-skills-repository.md`
  - [Claude Code Skills Customization Guide (alexop.dev)](https://alexop.dev/posts/claude-code-customization-guide-claudemd-skills-subagents/) → `docs/alexop-skills-customization-guide.md`
  - [Security Phoenix Skills for Claude Code](https://github.com/Security-Phoenix-demo/security-skills-claude-code) → `docs/security-phoenix-skills-repo.md`
  - [Hooks in Skills and Agents (Official Reference)](https://code.claude.com/docs/en/hooks) → `docs/hooks-in-skills-official-reference.md`
- **Summary**: Comprehensive research on custom Claude Code skills for secure package installation. Key findings: (1) Skills are prompt-based workflow packages (SKILL.md in .claude/skills/) — advisory, not enforcement. They cannot prevent agents from bypassing them via raw Bash. (2) Skills gain enforcement value only when combined with permission deny rules (block raw install commands) and PreToolUse hooks (catch bypasses). (3) Skills CAN define lifecycle-scoped hooks in their own frontmatter — these activate when the skill is invoked and deactivate when it finishes. (4) Claude can auto-invoke skills based on description matching (probabilistic, not deterministic). (5) Agents cannot be directly forced to use skills, but can be indirectly forced by denying raw install commands so the skill is the only viable path. (6) Security Phoenix is the most comprehensive real-world example — skills + embedded hooks for package install gating. (7) The Anthropic official skills repo has no security skills; community guardrails all use hooks, not skills. (8) Skills' unique value is as the workflow orchestration layer: structured check→decide→install→audit pipeline, user-facing discoverable interface, contextual knowledge injection, and dynamic pre-flight context.
- **Next**: Complete remaining tasks: vulnerability API survey, sibling spike cross-reference. Then Phase 2 synthesis.

## 2026-05-12 — Vulnerability Databases & Provenance APIs Research Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [OSV.dev API Documentation](https://google.github.io/osv.dev/api/) → `docs/osv-dev-api.md`
  - [OSV.dev API Latency/SLOs](https://osv.dev/blog/posts/api-latency-improvements-and-revised-slos/) → `docs/osv-dev-api-latency-slos.md`
  - [OSV Schema Specification](https://ossf.github.io/osv-schema/) → `docs/osv-schema-specification.md`
  - [Socket.dev Package Scores](https://docs.socket.dev/docs/package-scores) → `docs/socket-dev-package-scores.md`
  - [Socket.dev REST API Score Endpoint](https://docs.socket.dev/reference/getscorebynpmpackage) → `docs/socket-dev-rest-api-score-endpoint.md`
  - [GitHub Advisory Database API](https://docs.github.com/en/rest/security-advisories/global-advisories) → `docs/github-advisory-database-api.md`
  - [deps.dev API Documentation](https://docs.deps.dev/api/v3alpha/) → `docs/deps-dev-api.md`
  - [npm audit Internals](https://docs.npmjs.com/cli/v11/commands/npm-audit/) → `docs/npm-audit-internals.md`
  - [npm Provenance/Sigstore](https://docs.npmjs.com/generating-provenance-statements/) → `docs/npm-provenance-sigstore.md`
  - [pip-audit Documentation](https://github.com/pypa/pip-audit) → `docs/pip-audit-documentation.md`
  - [PyPI PEP 740 Attestations](https://docs.pypi.org/attestations/) → `docs/pypi-pep740-attestations.md`
  - [cargo-audit Documentation](https://github.com/rustsec/rustsec/blob/main/cargo-audit/README.md) → `docs/cargo-audit-documentation.md`
  - [vulnix Nix Vulnerability Scanner](https://github.com/nix-community/vulnix) → `docs/vulnix-nix-vulnerability-scanner.md`
  - [Nix Security Tracker](https://github.com/NixOS/nix-security-tracker) → `docs/nix-security-tracker.md`
  - [Nixpkgs Security Tracker Web](https://tracker.security.nixos.org/) → `docs/nixpkgs-security-tracker-web.md`
- **Summary**: Comprehensive survey of 12 vulnerability databases, provenance APIs, and security scoring services evaluated for real-time integration with Claude Code hooks and MCP servers. Key findings: (1) OSV.dev is the clear winner for hook-integrated CVE checking — free, no auth, no rate limits, ~120ms median query latency, 60+ ecosystems. (2) Socket.dev's public MCP server is the best supplementary source for supply chain risk beyond CVEs (typosquatting, install scripts, malware). (3) deps.dev uniquely provides `GetSimilarlyNamedPackages` for typosquatting detection and OpenSSF Scorecards for project health. (4) GHSA is the only source with malware-specific advisory type and EPSS scores, but requires GitHub auth. (5) Ecosystem audit tools (npm audit, pip-audit, cargo-audit) are project-level tools unsuitable for per-package hook calls (1-5s latency, require lockfile context). (6) Provenance verification (npm Sigstore, PyPI PEP 740) is not mature enough for hook enforcement — adoption is growing but absence of provenance is not actionable. (7) Significant gap: no Nix-specific tooling has API surface suitable for hooks (vulnix is CLI-only system scanner, nix-security-tracker has no public API). Full report at `vulnerability-apis-research.md`.
- **Next**: Complete sibling spike cross-reference task. Then Phase 2 synthesis of all research into unified guardrail architecture.

## 2026-05-12 — Sibling Spike Cross-Reference Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Cross-referenced `package-supply-chain-security/` (7 reports, 102 sources) and `devenv-security/` (6 reports, 88 sources). Extracted 7 defense categories mapped to 5 enforcement layers. Highest-impact immediate actions: environment-level `.npmrc`/`pip.conf` defaults, PreToolUse `updatedInput` rewriting to append safety flags, permission deny rules for dangerous Nix commands.
- **Next**: Phase 2 synthesis.

## 2026-05-12 — Phase 1 Complete
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: All 7 Phase 1 tasks completed successfully. Produced 7 detailed research reports (hooks, permissions, MCP servers, CLAUDE.md, skills, vulnerability APIs, sibling cross-reference) totaling ~55+ source documents saved to docs/. Key architectural finding: three-layer defense (PreToolUse hooks + permission deny rules + MCP/OS-level config) is required — no single layer is sufficient. Phase 2 tasks created: unified architecture spec, reference hook implementation, reference deny rules, bypass resistance assessment, deployment guide.
- **Next**: Begin Phase 2 synthesis. Start with unified architecture specification.

## 2026-05-12 — Phase 2 Synthesis Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Completed all Phase 2 tasks. The unified architecture specification (1,727 lines) subsumed reference implementations, deny rules, bypass assessment, and deployment guide into a single actionable document. Also produced a standalone Python hook script (reference-hook-script.py), settings.json configuration (reference-hook-settings.json), and comprehensive deny rules reference (reference-deny-rules.md, 1,069 lines). Conclusions written to research.md covering the five-layer defense model, 6 key technical findings, 4 actionable deliverables, and residual risk assessment.
- **Next**: Spike ready for completion via /complete-spike. All Phase 1 research and Phase 2 synthesis tasks done. Depth checklist satisfied across all reports.

## 2026-05-12 — Spike Summary
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike produced 12 report files, 58 source documents, 2 reference implementations (Python hook script + settings.json), and 1 unified architecture specification. Total research covered 7 mechanism areas across 2 phases. The three-layer enforcement architecture (hooks + deny rules + OS config) with two advisory layers (CLAUDE.md + skills) is the recommended approach. Immediate high-impact actions: (1) add Socket.dev MCP server (one command), (2) configure .npmrc/pip.conf age gates and script blocking, (3) deploy PreToolUse hook with OSV.dev validation.

## 2026-05-12 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. No single Claude Code mechanism is sufficient to prevent agents from installing compromised packages — the required architecture is a five-layer defense: PreToolUse hooks (primary enforcement, queries OSV.dev + checks age + rewrites commands), permission deny rules (fast catch for 15+ package managers), OS/environment config (.npmrc/.pip.conf/.nix.conf failsafes), CLAUDE.md instructions (advisory routing), and custom skills (workflow orchestration). Key technical findings: hooks fire before permission checks and can't be bypassed; OSV.dev is optimal for hooks (120ms, free, no auth); publication age gating catches 92% of PyPI malware; updatedInput enables transparent safety flag injection. Produced: unified architecture spec (1,727 lines), reference hook script (Python, stdlib-only), reference settings.json, deny rules reference (1,069 lines), 7 detailed research reports, 58 source documents. Three follow-on candidates flushed to proposed-spikes.md: Nix vulnerability API gap, field testing, and PostToolUse lockfile change detection.
