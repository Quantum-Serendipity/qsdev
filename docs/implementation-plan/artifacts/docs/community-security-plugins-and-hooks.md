# Community Security Plugins, Hooks, and Guardrails for Claude Code
> Retrieved: 2026-05-12

## Package Security Plugins

### attach-guard (attach-dev/attach-guard)
- Source: https://dev.to/hammadtariq/i-built-a-claude-code-plugin-that-blocks-compromised-packages-before-installation-1o3l
- PreToolUse hook intercepting every package install command
- Socket.dev API for supply chain risk scoring
- Blocks packages scoring below 50/100, flags 50-70
- Rewrites commands to safer versions via `updatedInput`
- Supports npm, pip, Go, Cargo
- Age-gates packages published within 48 hours
- Free Socket.dev tier available
- Install: `claude plugin marketplace add attach-dev/attach-guard`

### Security Phoenix (Security-Phoenix-demo/security-skills-claude-code)
- Source: https://github.com/Security-Phoenix-demo/security-skills-claude-code
- Most comprehensive security skills toolkit found
- PreToolUse hook gates npm/yarn/pnpm/pip/uv/poetry/cargo/go/gem/bundle/composer/dotnet
- Known-malicious package blocking
- Typosquat detection
- New package flagging for user confirmation
- SessionStart audit (fingerprints project, runs dependency audits)
- PostToolUse scanning (SQL injection, innerHTML, hardcoded secrets)
- Four hooks: SessionStart, PreToolUse, PostToolUse, SessionEnd
- Install: `bash "skills/Security Assessment/install/install.sh" --full`

### harish-garg/security-scanner-plugin
- Source: https://github.com/harish-garg/security-scanner-plugin
- Scans code for vulnerabilities using GitHub's official advisory data
- AI-powered plain-English explanations of vulnerabilities and fix suggestions
- Requires GitHub MCP Server configured in Claude Code

## Destructive Command Guards

### hex/claude-guard
- Source: https://github.com/hex/claude-guard
- Three-tier protection:
  - Tier 1: Blocks catastrophic (rm -rf /, DROP DATABASE, kubectl delete namespace)
  - Tier 2: Redirects dangerous to safer alternatives (--force -> --force-with-lease)
  - Tier 3: Credential/secret detection in written files (AWS keys, API tokens, JWTs, private keys)
- Activates automatically, no configuration needed

### kenryu42/claude-code-safety-net
- Source: https://github.com/kenryu42/claude-code-safety-net
- Catches destructive git and filesystem commands before execution
- Cross-agent: supports Codex, Claude Code, OpenCode, Gemini CLI, Copilot CLI

### Cocabadger/saferun-guard
- Source: https://github.com/Cocabadger/saferun-guard
- Runtime safety firewall
- Blocks dangerous commands, protects sensitive files (~20ms overhead)

### Dicklesworthstone/destructive_command_guard
- Source: https://github.com/Dicklesworthstone/destructive_command_guard
- High-performance hook for blocking destructive commands
- Cross-agent support

## Prompt Injection Defense

### lasso-security/claude-hooks
- Source: https://github.com/lasso-security/claude-hooks
- PostToolUse hook scanning all tool outputs for injection attempts
- 50+ regex patterns across 5 attack categories:
  - Instruction Override ("ignore previous", "new system prompt")
  - Role-Playing/DAN manipulation
  - Encoding/Obfuscation (Base64, leetspeak, homoglyphs)
  - Context Manipulation (fake authority, hidden comments)
  - Instruction Smuggling (hidden in HTML/code comments)
- Injects warning into Claude's context rather than blocking (reduces false positives)
- Millisecond overhead
- Enterprise deployable via managed settings

### slavaspitsyn/claude-code-security-hooks
- Source: https://github.com/slavaspitsyn/claude-code-security-hooks
- 7 layers of defense against prompt injection:
  1. Inline Edit Hook: blocks modifications to settings.json and hooks/
  2. Network Restrictions: POST/upload restricted to allowed domains
  3. Canary Files: traps in ~/.ssh/ that use prompt injection against prompt injection
- 47 tests covering all three hooks
- Catches 99% of straightforward prompt injection commands

### MG-Cafe/agentic-coder-shield
- Source: https://github.com/MG-Cafe/agentic-coder-shield
- 3 defense layers against prompt injection, data exfiltration, credential leaks

## Security Guardrail Frameworks

### efij/secure-claude-code
- Source: https://github.com/efij/secure-claude-code
- YARA-style guard packs for: secrets, exfiltration, prompt injection, MCP abuse, risky agent actions
- Local-first, modular
- Installable with profiles (e.g., "balanced")
- Cross-platform: Codex, Cursor, Windsurf, Claude Desktop

### Boucle Framework (Bande-a-Bonnot/Boucle-framework)
- Source: https://framework.boucle.sh/
- bash-guard + git-safe + file-guard safety hooks
- bash-guard blocks: rm -rf, sudo, curl|bash, chmod 777, Docker destruction, database drops, cloud infra deletion, mass file deletion, data exfiltration, env dumps, sensitive file reads, git push --force
- Detects encoding bypasses: base64/hex decode piped to shell, reversed strings, process substitution
- enforce-hooks: turns CLAUDE.md rules into deterministic enforcement

### dwarvesf/claude-guardrails
- Source: https://github.com/dwarvesf/claude-guardrails
- Settings + hooks for destructive command blocking
- General security (rm -rf, git push, secrets)

### rulebricks/claude-code-guardrails
- Source: https://github.com/rulebricks/claude-code-guardrails
- PreToolUse + external policy API
- Real-time guardrails via HTTP hooks
- Policy changes apply instantly across team without git pull or restart

### mafiaguy/claude-security-guardrails
- Source: https://github.com/mafiaguy/claude-security-guardrails
- PreToolUse + PostToolUse with 60+ patterns
- React dashboard for monitoring
- Checks 16 known vulnerable packages by name

## Dependency Management Skills

### softaworks/dependency-updater
- Source: https://claudeskills.club/skills/softaworks-dependency-updater
- Automates dependency management across multiple languages
- Safe semantic versioning updates (Patch and Minor), prompts for Major
- Detects project types automatically

### Universal Dependency Installer
- Source: https://mcpmarket.com/tools/skills/universal-dependency-installer
- Identifies correct package manager for any project
- Supports Node.js (npm, Yarn, pnpm, Bun), Python (uv), Rust (Cargo), Go

### Dependency Audit Skill
- Source: https://www.makr.io/skills/dependency-audit
- Update, clean up, and secure dependencies

## MCP Servers for Security

### Socket.dev MCP
- Public hosted: https://mcp.socket.dev/ (zero auth, free)
- Supply chain scoring via `depscore` tool
- Supports npm, PyPI, Cargo
- Also available as local stdio server with API key

### Snyk MCP
- Bundled with Snyk CLI v1.1298.0+
- 8 tools: snyk_sca_scan, snyk_code_scan, snyk_iac_scan, snyk_container_scan, etc.
- Requires Snyk account/token

### CVE MCP Server
- Source: https://cybersecuritynews.com/cve-mcp-server-and-claude/
- 27 tools across 21 APIs
- DevSecOps tools: scan_dependencies (OSV.dev), scan_github_advisories, urlscan_check
- Scans requirements.txt with prioritized upgrade recommendations

### Google MCP Security
- Source: https://github.com/google/mcp-security
- Security Operations and Threat Intelligence servers

### Contrast Security MCP
- Source: https://github.com/Contrast-Security-OSS/mcp-contrast
- Application security vulnerability remediation

### Codacy MCP
- Source: https://blog.codacy.com/equipping-claude-code-with-deterministic-security-guardrails
- Trivy scanning integration
- Post-install dependency scanning

## Curated Lists

### hesreallyhim/awesome-claude-code
- Hand-curated list of skills, hooks, slash-commands, plugins, agents
- THE_RESOURCES_TABLE.csv with structured entries
- Sections for security, hooks, guardrails

### efij/awesome-claude-code-security
- 330+ entries across 18 security categories
- Most comprehensive security-specific resource list

### rohitg00/awesome-claude-code-toolkit
- 135 agents, 35 skills, 42 commands, 176+ plugins, 20 hooks

### GetBindu/awesome-claude-code-and-skills
- 8.7k stars, general skills collection

### ccplugins/awesome-claude-code-plugins
- Curated plugin list: slash commands, MCP servers
