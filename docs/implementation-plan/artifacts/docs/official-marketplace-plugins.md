# Anthropic Official Claude Code Plugin Marketplace
> Source: https://github.com/anthropics/claude-plugins-official/blob/main/.claude-plugin/marketplace.json
> Retrieved: 2026-05-12

## Security-Relevant Plugins in Official Marketplace

### 42Crunch API Security Testing
- Repo: 42Crunch-AI/claude-plugins
- Automate API security audits and detect OWASP vulnerabilities

### Aikido Security
- Repo: AikidoSec/aikido-claude-plugin
- SAST, secrets, and IaC vulnerability detection
- Scans files in real-time, guides Claude to fix issues before shipping
- Auto-identifies files changed during session, runs full security scan (up to 50 files/request)
- Remediation loop: applies fixes and re-scans up to 3 times until clean
- Requires Node.js 18+, Aikido API key
- Setup: `/aikido:setup <api-key>`, then `/aikido:scan`

### Endor Labs AI Plugins
- Repo: endorlabs/ai-plugins
- Software supply chain security scanning
- Automates installation and authentication of endorctl CLI
- Bridges AI coding assistants to Endor Labs backend security platform
- Setup via natural language: "set up endorctl"

### SonarQube
- Repo: SonarSource/sonarqube-agent-plugins
- 7,000+ rules, secrets scanning, agentic analysis, quality gates across 40+ languages
- Blocks 450+ secret patterns before content enters LLM context
- PostToolUse hooks run SonarQube analysis after every file edit (Agentic Analysis)
- Slash commands for quality gate status, dependency risks, coverage review

### CrowdStrike Falcon Foundry
- Repo: CrowdStrike/foundry-skills
- Cybersecurity app development on Falcon platform

### Security-Guidance (Official Anthropic)
- Repo: anthropics/claude-code (plugins/security-guidance)
- Warns about command injection, XSS, and unsafe code patterns when editing files
- Uses hooks directory for enforcement
- Install: `/plugin install security-guidance@claude-plugins-official`

### CodeRabbit
- Repo: coderabbitai/skills
- External code review with 40+ static analyzers

### Qodo Skills
- Repo: qodo-ai/qodo-skills
- Shift-left code review: quality, testing, security, compliance checks
- Fetches org- and repo-level coding rules including security requirements
- Identifies bugs, security risks, and quality issues in local diffs or GitHub PRs

### Hookify (Official Anthropic)
- Custom hook creation for behavior prevention

## Other Notable Official Plugins (Non-Security)
- GitHub, GitLab, Firebase, AWS (multiple), Azure (multiple), Cloudflare
- Chrome DevTools MCP, Figma, various LSP servers
- Code Review, Code Simplifier, Feature Dev (Anthropic internal tools)

## Total Count
~130+ plugins in official marketplace as of May 2026
