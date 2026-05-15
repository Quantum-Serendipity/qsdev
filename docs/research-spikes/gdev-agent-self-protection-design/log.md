# Research Log: gdev Agent Self-Protection Design

## 2026-05-15 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. 9 tasks across 2 phases completed. 8 research reports produced (7 topic reports + 1 synthesis). 32+ web sources saved to docs/. All 7 topic reports pass all 6 depth checklist items. Key conclusion: gdev's self-protection is a 32-rule enforcement layer using exit code 2 for all security denials (avoiding Claude Code's buggy permissionDecision machinery), organized as 3 consolidated hook scripts with three-tier bypass policy (absolute deny / interactive bypass / magic comments), severity-tiered fail policy (fail-closed for security, fail-open for advisory), per-rule monitor mode with enforce_always for nuclear rules, and two-tier path canonicalization. Recommended as Phases 33-35 of the gdev implementation plan. No follow-on spike candidates identified — remaining open questions are implementation details captured in the phase recommendations.

## 2026-05-15 — Phase 2 Synthesis Complete — Spike Ready for Closure
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Cross-report synthesis resolved 4 contradictions between the 7 research reports and produced a consolidated 32-rule catalog across 4 categories (self-protection, configuration guard, MCP poisoning, integrity checks). Key resolution: "ask" verdict is dead — all security rules use exit code 2 due to Claude Code bugs #39344/#52822; the 10 rules originally assigned "ask" become Tier 2 interactive-bypass denials. Depth checklist review confirmed all 7 reports pass all 6 criteria with no gaps. 7 of 9 open questions resolved; 2 remaining are out-of-scope implementation details. Three implementation phases recommended (33: core rules + fail-closed, 34: bypass + monitor + audit, 35: Go binary + AST + hardening).
- **Next**: Spike is complete. Ready for `/complete-spike` or handoff to implementation plan creation.

## 2026-05-15 — Phase 1 Complete, Moving to Synthesis
- **Type**: decision
- **Status**: success
- **Depth**: moderate
- **Summary**: All 7 Phase 1 tasks completed successfully. Key cross-cutting findings: (1) Claude Code hooks are inherently fail-open — only exit code 2 blocks, all errors allow; (2) bug #39344 means ask verdict silently overrides deny rules — all security-critical rules must use deny/exit-2 only; (3) Bash tool is the primary bypass vector — every Write/Edit rule needs a companion Bash regex guard; (4) extracting write targets from arbitrary Bash is undecidable — three-strategy defense (regex + AST + blocklist); (5) chained protection pattern solves the self-protection bypass paradox. Phase 2 synthesis tasks created.
- **Next**: Cross-report synthesis to resolve contradictions, consolidate the 23-rule set with verdicts/tiers/bypass-policies, and draft architectural recommendations for the implementation plan.

## 2026-05-15 — Canonical Path Resolution Design Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Gemini CLI Symlink Bypass #1121](https://github.com/google-gemini/gemini-cli/issues/1121) -> `docs/gemini-cli-symlink-bypass.md`
  - [Go filepath.Resolve Proposal #37113](https://github.com/golang/go/issues/37113) -> `docs/go-filepath-resolve-proposal.md`
  - [realpath(1) Man Page](https://man7.org/linux/man-pages/man1/realpath.1.html) -> `docs/realpath-man-page.md`
  - [moai-adk Unicode NFD/NFC Mismatch #342](https://github.com/modu-ai/moai-adk/issues/342) -> `docs/moai-adk-unicode-path-mismatch.md`
  - [AgentFS Kernel Isolation](https://codepointer.substack.com/p/agentfs-how-to-stop-ai-agents-from) -> `docs/agentfs-kernel-isolation-approach.md`
  - [POSIX Hardlink Security](https://michael.orlitzky.com/articles/posix_hardlink_heartache.xhtml) -> `docs/posix-hardlink-security-issues.md`
  - [Consul-Template Symlink Bypass CVE-2026-5061](https://discuss.hashicorp.com/t/hcsec-2026-12) -> `docs/consul-template-symlink-bypass-cve-2026-5061.md`
  - [bashguard AST Command Security](https://github.com/sunir/bashguard) -> `docs/bashguard-ast-command-security.md`
  - [tree-sitter-bash Node Types](https://github.com/tree-sitter/tree-sitter-bash) -> `docs/tree-sitter-bash-redirect-node-types.md`
  - [Claude Code file_redirect Issue #47701](https://github.com/anthropics/claude-code/issues/47701) -> `docs/claude-code-file-redirect-permission-issue.md`
- **Summary**: Comprehensive canonical path resolution design for gdev hook scripts. Cataloged 9 bypass technique categories (symlink traversal, relative path manipulation, hardlink creation, /proc/self/root traversal, /dev/fd tricks, case sensitivity, Unicode normalization, TOCTOU race conditions, bind mount namespace tricks). Designed two-tier canonicalization pipeline (filesystem via realpath -> lexical via realpath -m) with Go equivalent (EvalSymlinks -> parent-walk fallback). Critical finding: Write/Edit paths can be reliably canonicalized, but Bash tool commands cannot be reliably parsed for all write targets -- the problem is equivalent to static analysis of arbitrary shell scripts, which is undecidable. Designed three-strategy defense: regex pattern matching (Phase 32 bash), AST-based analysis (future Go via tree-sitter-bash), and evasion mechanism blocklist (complementary). Path matching uses exact/prefix string comparison on canonicalized paths (no glob/regex needed). Full pseudocode for both Bash and Go implementations. Report written to `canonical-path-research.md`.
- **Next**: Mark task as completed. All Phase 1 tasks now complete.

## 2026-05-15 — Monitor/Shadow Mode Design Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [SELinux Permissive Mode (Gentoo Wiki)](https://wiki.gentoo.org/wiki/SELinux/Tutorials/Permissive_versus_enforcing) -> `docs/selinux-permissive-mode-gentoo-wiki.md`
  - [Microsoft ASR Audit Mode Deployment](https://learn.microsoft.com/en-us/defender-endpoint/attack-surface-reduction-rules-deployment-test) -> `docs/microsoft-asr-audit-mode-deployment.md`
  - [AWS WAF Count Mode Testing](https://docs.aws.amazon.com/waf/latest/developerguide/web-acl-testing.html) -> `docs/aws-waf-count-mode-testing.md`
  - [Kubernetes ValidatingAdmissionPolicy Modes](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) -> `docs/kubernetes-validating-admission-policy-modes.md`
  - [seccomp SECCOMP_RET_LOG Manpage](https://man7.org/linux/man-pages/man2/seccomp.2.html) -> `docs/seccomp-ret-log-manpage.md`
- **Summary**: Comprehensive monitor mode design for gdev self-protection hooks. Surveyed 6 established security systems (SELinux permissive, AppArmor complain, Defender ASR audit, AWS WAF count, seccomp RET_LOG, Kubernetes ValidatingAdmissionPolicy warn/audit). Universal finding: no system auto-expires monitor mode; all use same log stream for monitor and enforce events; all support per-entity granularity. Designed per-rule mode control with category defaults, `enforce_always` flag for nuclear self-protection rules (borrowed from AppArmor's "deny enforces even in complain"), 5-day calibration period with escalating reminders, unified JSONL audit log with `mode` and `effective_verdict` fields, and full CLI workflow (`gdev hook monitor/audit/enforce`). Report written to `monitor-mode-research.md`.
- **Next**: Mark task as completed. Continue with remaining Phase 1 tasks (verdict model, canonical path, escape hatches).

## 2026-05-15 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized from proposed-spikes.md entry. Design spike for implementing agent self-protection rules natively in gdev's Go hook architecture. Covers: preventing agent from disabling its own security tools, MCP config poisoning detection, three-outcome verdict model (allow/deny/ask), canonical path resolution, and monitor mode. Source research from security-tooling-evaluation-gdev spike (Prempti and reasoning-core deep dives).
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-15 — Phase 1 Tasks Defined
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Research question confirmed: "How should gdev implement agent self-protection rules natively in its Go hook architecture to prevent a compromised or manipulated AI agent from dismantling gdev's own security infrastructure?" Seven Phase 1 tasks created covering threat modeling, Prempti pattern extraction, verdict model design, canonical path resolution, monitor mode, escape hatches, and fail-closed/fail-open policy. Source material reviewed from security-tooling-evaluation-gdev spike (prempti-research.md, reasoning-core-research.md, cross-tool-comparison-research.md) and gdev implementation plan phases 5, 11, 32.
- **Next**: Begin research. Start with high-priority tasks: threat model, Prempti patterns, verdict model, fail-closed policy.

## 2026-05-15 — Fail-Closed vs Fail-Open Policy Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [AuthZed fail-open/fail-closed](https://authzed.com/blog/fail-open) → `docs/authzed-fail-open-fail-closed.md`
  - [OWASP Fail Securely](https://owasp.org/www-community/Fail_securely) → `docs/owasp-fail-securely.md`
  - [Microsoft Agent Governance Toolkit](https://opensource.microsoft.com/blog/2026/04/02/introducing-the-agent-governance-toolkit-open-source-runtime-security-for-ai-agents/) → `docs/microsoft-agent-governance-toolkit.md`
  - [Datadog Container Security AppArmor/SELinux](https://securitylabs.datadoghq.com/articles/container-security-fundamentals-part-5/) → `docs/datadog-container-security-apparmor-selinux.md`
  - [gVisor Security Model](https://gvisor.dev/docs/architecture_guide/security/) → `docs/gvisor-security-model.md`
  - [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks) → `docs/claude-code-hooks-reference.md`
  - [Silent Hook Failure Mode (Medium)](https://thinkingthroughcode.medium.com/the-silent-failure-mode-in-claude-code-hook-every-dev-should-know-about-0466f139c19f) → `docs/medium-silent-hook-failure-mode.md`
  - [AWS WAF Fail-Open/Fail-Closed](https://cloudsoft.io/blog/aws-load-balancers-waf-availability-security) → `docs/aws-waf-fail-open-fail-closed.md`
- **Summary**: Comprehensive analysis of fail-closed vs fail-open for gdev's hook system. Key finding: Claude Code hooks are inherently fail-open (only exit code 2 blocks; all other errors including crashes, timeouts, and malformed JSON allow the operation to proceed). This means gdev must implement fail-closed behavior at the hook level for security-critical rules. Recommendation: severity-tiered failure policy — fail-closed for self-protection/destructive-prevention/credential-scanning hooks, fail-open for advisory hooks (cost, audit, test, isolation). Monitor mode as transitional deployment state. Analyzed 7 failure scenarios, surveyed industry practices across seccomp, SELinux, AppArmor, firewalls, AWS WAF, gVisor, Prempti, reasoning-core, and Microsoft Agent Governance Toolkit.
- **Next**: Mark task as completed. Continue with remaining Phase 1 tasks (threat model, Prempti patterns, verdict model).

## 2026-05-15 — Prempti Self-Protection Pattern Extraction Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Prempti rules YAML](https://raw.githubusercontent.com/falcosecurity/prempti/main/rules/default/coding_agents_rules.yaml) -> `docs/prempti-self-protection-rules-source.md`, `docs/prempti-mcp-skill-rules-source.md`, `docs/prempti-persistence-rules-source.md`, `docs/prempti-sandbox-disable-rules-source.md`
  - [Prempti event.rs](https://raw.githubusercontent.com/falcosecurity/prempti/main/plugins/coding-agents-plugin/src/event.rs) -> `docs/prempti-path-canonicalization-source.md`
  - [Prempti interceptor main.rs](https://raw.githubusercontent.com/falcosecurity/prempti/main/hooks/claude-code/src/main.rs) -> `docs/prempti-interceptor-path-handling.md`
- **Summary**: Extracted all 6 Prempti self-protection rules (5 deny, 1 ask) with full conditions from source code. Translated each to gdev's bash hook architecture with implementation sketches. Identified 6 gdev-specific self-protection rules Prempti doesn't need (devenv.nix, .pre-commit-config.yaml, nix.conf, .gdev.yaml, audit trail, CLAUDE.md sections). Designed declarative rule format (compiled Go defaults + YAML user overrides). Documented MCP config poisoning detection with 5 deny + 5 ask rules and gdev implementation strategy. Extracted Prempti's two-tier path canonicalization from plugin source (filesystem canonicalize -> lexical fallback). Key design decision: consolidate all self-protection into 2 hook scripts (Bash matcher, Write/Edit/Read matcher) to avoid per-rule process spawning.
- **Next**: Complete remaining Phase 1 tasks — threat model, verdict model design. The Prempti patterns report provides foundation for both.

## 2026-05-15 — Threat Model: Agent Self-Disabling Vectors Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Microsoft RCE in AI Agent Frameworks](https://www.microsoft.com/en-us/security/blog/2026/05/07/prompts-become-shells-rce-vulnerabilities-ai-agent-frameworks/) -> `docs/microsoft-rce-ai-agent-frameworks.md`
  - [Arxiv: Prompt Injection on Agentic Coding Assistants](https://arxiv.org/html/2601.17548v1) -> `docs/arxiv-prompt-injection-agentic-coding-assistants.md`
  - [OWASP AI Agent Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html) -> `docs/owasp-ai-agent-security-cheatsheet.md`
  - [Botmonster: AI Agents as Insider Threats](https://botmonster.com/posts/ai-coding-agent-insider-threat-prompt-injection-mcp-exploits/) -> `docs/botmonster-ai-agents-insider-threats.md`
  - [HiddenLayer: Guardrails Self-Policing Bypass](https://www.hiddenlayer.com/research/same-model-different-hat) -> `docs/hiddenlayer-guardrails-bypass-self-policing.md`
  - [Comment and Control Attack](https://oddguan.com/blog/comment-and-control-prompt-injection-credential-theft-claude-code-gemini-cli-github-copilot/) -> `docs/comment-and-control-attack.md`
  - [Adversa: Claude Code Deny Rule Bypass](https://adversa.ai/blog/claude-code-security-bypass-deny-rules-disabled/) -> `docs/adversa-claude-code-deny-rule-bypass.md`
  - [Ona: Claude Code Sandbox Escape](https://ona.com/stories/how-claude-code-escapes-its-own-denylist-and-sandbox) -> `docs/ona-claude-code-sandbox-escape.md`
- **Summary**: Comprehensive threat model enumerating 12 attack vector categories an AI agent could use to disable gdev's security infrastructure. Expanded reasoning-core's 6-vector model with 6 additional categories (configuration poisoning, privilege escalation, TOCTOU, deny rule exhaustion, hook registration manipulation, audit trail destruction). Mapped each vector against Phase 32's current hooks: 2 fully defended, 3 partially defended, 7 not defended at all. Rated gaps by likelihood/impact and identified 5 minimum self-protection rule sets (A-E) needed to close critical gaps. Analyzed 5 real-world incidents with gdev-specific lessons. Compared to Prempti (6 rules), reasoning-core (L1-L4), and OWASP (9 pillars). Key finding: the most critical undefended vectors are direct/indirect file mutation of settings.json and hook scripts. Report written to `threat-model-research.md`.
- **Next**: Mark task as completed. Continue with verdict model design and canonical path resolution tasks.

## 2026-05-15 — Three-Outcome Verdict Model Design Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Claude Code Issue #39344: ask overrides deny](https://github.com/anthropics/claude-code/issues/39344) -> `docs/claude-code-issue-39344-ask-overrides-deny.md`
  - [Claude Code Hook Development SKILL.md](https://raw.githubusercontent.com/anthropics/claude-code/main/plugins/plugin-dev/skills/hook-development/SKILL.md) -> `docs/claude-code-hook-development-skill.md`
  - [Claude Code Hooks Lifecycle Guide](https://claudefa.st/blog/tools/hooks/hooks-guide) -> `docs/claudefast-hooks-lifecycle-guide.md`
  - [Microsoft Agent Governance Toolkit](https://github.com/microsoft/agent-governance-toolkit) -> `docs/microsoft-agent-governance-toolkit-policy-model.md`
  - [AWS Cedar/Verified Permissions](https://docs.aws.amazon.com/verifiedpermissions/latest/userguide/terminology.html) -> `docs/aws-cedar-verified-permissions-terminology.md`
  - [Claude Code Hooks Reference (detailed)](https://code.claude.com/docs/en/hooks) -> `docs/claude-code-hooks-reference-detailed.md`
  - XACML combining algorithms (web search synthesis) -> `docs/xacml-combining-algorithms-reference.md`
- **Summary**: Designed complete four-verdict model (allow/deny/ask/warn) for gdev hooks, upgrading from Prempti's three-verdict system. Key findings: (1) deny-overrides is the correct combining algorithm, matching XACML, Cedar, and Claude Code's own implied precedence; (2) critical bug #39344 means ask verdict silently overrides deny rules -- gdev must never use ask for operations covered by deny rules; (3) two deny mechanisms (exit 2 fail-closed vs JSON deny structured) serve different failure tiers; (4) 23 rules assigned verdicts (13 deny, 10 ask, 3 warn checks); (5) three-tier rule precedence (managed-policy > user > project) prevents configuration poisoning. Comprehensive audit logging schema designed with verdict records integrated into SOC 2 audit trail.
- **Next**: Run revision cycle via /complete-task. Then continue with remaining Phase 1 tasks (canonical path resolution, monitor mode, escape hatches).

## 2026-05-15 — Escape Hatch and Bypass Mechanism Design Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 
  - [Claude Code Hooks Response Format](https://code.claude.com/docs/en/hooks) -> `docs/claude-code-hooks-response-format.md`
  - [Claude Code Hook Verdict Bug #52822](https://github.com/anthropics/claude-code/issues/52822) -> `docs/claude-code-hook-verdict-bug-52822.md`
  - [Claude Code Sandbox Escape Hatch Issue #20259](https://github.com/anthropics/claude-code/issues/20259) -> `docs/claude-code-sandbox-escape-hatch-issue-20259.md`
  - [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) -> `docs/owasp-logging-cheat-sheet.md`
  - [Prefactor: Audit Trails for AI Agents](https://prefactor.tech/blog/audit-trails-in-ci-cd-best-practices-for-ai-agents) -> `docs/prefactor-audit-trails-ai-agents.md`
  - [Graphite: git --no-verify](https://graphite.com/guides/git-commit--no-verify) -> `docs/graphite-git-no-verify.md`
- **Summary**: Designed complete bypass system for gdev's hook architecture. Enumerated 6 bypass mechanisms (magic comments, CLI command, env var, interactive prompt, time-limited token, out-of-band channel) with human-accessibility vs agent-exploitability analysis for each. Designed three-tier bypass policy: Tier 1 (absolute deny, never bypassable) for settings.json/hooks/audit protection; Tier 2 (interactive bypass via separate-terminal CLI command with chained protection) for devenv.nix/precommit/mcp config; Tier 3 (magic comment bypass) for destructive prevention/credential scan. Key innovation: "chained protection" pattern where the bypass command itself is blocked by a tier 1 rule, forcing the developer to run it in their terminal (outside Claude Code), creating cryptographic separation between agent and human. Critical finding: Claude Code bugs #39344 and #52822 make the JSON ask verdict unreliable for security enforcement — gdev must use exit code 2 for all denials. Designed mandatory audit logging with JSONL schema (14 fields), dual-destination logging (session + dedicated bypass log), tamper-evident hash chain, and tiered alerting. Surveyed prior art from SELinux, sudo, AppArmor, git --no-verify, firewall TTL exceptions, and Claude Code's sandbox escape. Report written to `escape-hatch-research.md`.
- **Next**: Mark task as completed. Continue with remaining Phase 1 tasks (canonical path resolution, monitor mode).
