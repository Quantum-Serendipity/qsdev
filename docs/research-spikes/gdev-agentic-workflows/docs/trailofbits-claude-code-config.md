<!-- Source: https://github.com/trailofbits/claude-code-config -->
<!-- Retrieved: 2026-05-12 -->

# Trail of Bits Claude Code Config

Opinionated defaults, documentation, and workflows for Claude Code at Trail of Bits.

## Key Configuration Patterns

### Settings
- DISABLE_TELEMETRY, DISABLE_ERROR_REPORTING
- enableAllProjectMcpServers: false (prevent malicious repo MCP)
- alwaysThinkingEnabled: true
- cleanupPeriodDays: 365
- Deny rules for SSH keys, cloud credentials, shell config

### Hooks
- PreToolUse: block rm -rf, block direct pushes to main/master
- Stop hook: Anti-rationalization gate using prompt-type hooks with fast model

### Sandboxing
- Deny read: ~/.ssh/**, ~/.gnupg/**, ~/.aws/**, ~/.azure/**, ~/.kube/**, ~/.docker/config.json
- Deny read: ~/.npmrc, ~/.pypirc, ~/.gem/credentials
- Deny edit: ~/.bashrc, ~/.zshrc

### Custom Commands
- /review-pr [number]: Multi-agent PR review with parallel evaluation and auto-fixes
- /fix-issue [number]: Full autonomy - research, plan, implement, test, create PR, self-review
- /merge-dependabot [repo]: Audit, build transitive maps, batch overlapping PRs, parallel merge

### MCP Servers
- Global: Context7 (library docs), Exa (web/code search)
- Project: repo-specific tools

### Key Principles
1. Scope work to single sessions
2. Pair permissions bypass with sandboxing
3. Encode expertise in agents, procedures in commands
4. Use hooks for structured decision intervention
5. Prefer fresh sessions over compaction

### Recommended Skills
- ask-questions-if-underspecified (Trail of Bits)
- modern-python (Trail of Bits)
- differential-review (Trail of Bits)
- /superpowers:brainstorm (Superpowers)
- frontend-design (Anthropic Official)

### Global CLAUDE.md Practices
- No speculation, no premature abstraction
- Code quality limits (function length, complexity, line width)
- Language-specific toolchains for Python, Node/TS, Rust, Bash, GitHub Actions
- Testing methodology and code review order
