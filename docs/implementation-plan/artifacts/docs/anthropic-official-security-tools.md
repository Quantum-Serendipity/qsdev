# Anthropic Official Security Tools for Claude Code
> Retrieved: 2026-05-12

## 1. Security-Guidance Plugin (Built into claude-code repo)
- Source: https://github.com/anthropics/claude-code/tree/main/plugins/security-guidance
- Warns about command injection, XSS, and unsafe code patterns when editing files
- Uses hooks/ directory for enforcement
- Part of the official claude-code repository
- Install: `/plugin install security-guidance@claude-plugins-official`

## 2. Claude Code Security Review (GitHub Action)
- Source: https://github.com/anthropics/claude-code-security-review
- AI-powered security review GitHub Action for PRs
- Deep semantic analysis beyond pattern matching:
  - Broken access control
  - Business-logic flaws
  - Insecure deserialization
  - Auth bypass through unusual state machines
  - DNS rebinding vulnerabilities
- Diff-aware: analyzes only changed files
- Language-agnostic
- Advanced false positive filtering
- Caught vulnerabilities in Anthropic's own codebase (including Claude Code itself)
- Cost: ~$0.90-$1.80 per 500-line PR scan (billed to your API key)

## 3. Claude Code Security (Enterprise Feature)
- Source: https://thenewstack.io/anthropics-claude-security-beta/, https://thehackernews.com/2026/02/anthropic-launches-claude-code-security.html
- Emerged from closed preview to scan codebases for vulnerabilities
- Multi-stage verification
- Currently available to Enterprise and Team customers (limited research preview)
- Separate from the GitHub Action — this is a first-party scanning feature

## 4. Built-in Security Architecture
- Source: https://code.claude.com/docs/en/security
- Permission system: Allow/Ask/Deny with deny-first evaluation
- Sandboxing: bubblewrap (Linux) / Seatbelt (macOS)
- Hooks: PreToolUse/PostToolUse lifecycle events
- Managed settings: /etc/claude-code/managed-settings.json (Linux)
- Key managed-only settings:
  - `allowManagedHooksOnly` — blocks user/project hooks
  - `allowManagedPermissionRulesOnly` — locks down permission rules
  - `allowManagedMcpServersOnly` — controls MCP server access
  - `disableBypassPermissionsMode` — prevents permission bypasses
  - `disableAutoMode` — prevents auto mode

## 5. Official Skills Repository
- Source: https://github.com/anthropics/skills
- Contains document-focused skills (docx, pdf, pptx, xlsx) and creative/dev skills
- NO security-focused skills in the official repository
- Security use case is addressed through hooks and permissions in Anthropic's docs

## 6. Managed Settings Delivery Methods
- Server-managed via Claude.ai admin console (Team v2.1.38+ / Enterprise v2.1.30+)
- MDM/OS-level: macOS managed preferences (Jamf/Kandji), Windows registry (GPO/Intune)
- File-based: /etc/claude-code/managed-settings.json (Linux)
- Drop-in directory: managed-settings.d/*.json for modular policy fragments

## 7. Known Vulnerabilities (Patched)
- CVE-2025-59536: MCP configuration injection (RCE via .claude/settings.json hooks)
- CVE-2026-21852: API key harvesting from Claude Code sessions
- 50-subcommand deny-rule bypass (patched v2.1.90)
- Complete deny-rule non-enforcement (patched, was in v1.0.93)
