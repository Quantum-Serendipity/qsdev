# Trail of Bits Claude Code Skills and Security Configuration
> Sources: https://github.com/trailofbits/skills, https://github.com/trailofbits/claude-code-config
> Retrieved: 2026-05-12

## Trail of Bits Skills Repository (40+ plugins)

### Security Audit Skills (Most Relevant)
- **supply-chain-risk-auditor** — Evaluates dependency threat landscapes
- **audit-context-building** — Deep architectural understanding for audits (line-by-line analysis, invariants, reasoning hazards)
- **differential-review** — Security-focused differential review of code changes with git history analysis
- **static-analysis** — CodeQL, Semgrep, and SARIF parsing toolkit
- **semgrep-rule-creator** — Develops custom detection rules
- **semgrep-rule-variant-creator** — Ports Semgrep rules across programming languages
- **variant-analysis** — Identifies similar vulnerabilities across codebases
- **c-review** — C/C++ security review with clustered parallel workers and SARIF output
- **insecure-defaults** — Locates dangerous configurations and hardcoded credentials
- **sharp-edges** — Flags risky APIs and security-prone designs
- **fp-check** — Systematic false positive verification with mandatory gate reviews
- **constant-time-analysis** — Detects timing side-channels in cryptographic implementations
- **zeroize-audit** — Identifies missing secret zeroization in C/C++ and Rust
- **testing-handbook-skills** — Fuzzers, analysis, sanitizers, coverage tools

### Other Notable Skills
- **agentic-actions-auditor** — Examines GitHub Actions workflows for AI agent vulnerabilities
- **firebase-apk-scanner** — Scans APKs for Firebase misconfigurations
- **yara-authoring** — YARA rule authoring with linting and best practices
- **seatbelt-sandboxer** — macOS sandbox profile configuration
- **building-secure-contracts** — Smart contract security for 6 blockchains

### Trophy Case
Documented finding: timing side-channel discovered in ML-DSA signing via constant-time-analysis skill.

## Trail of Bits claude-code-config (Production Defaults)

### Philosophy
"Hooks are not a security boundary" but rather "structured prompt injection at opportune times."
Use `--dangerously-skip-permissions` with OS-level sandbox as primary boundary.

### Permission Deny Rules
**Credentials & Secrets:**
- `~/.ssh/**`, `~/.gnupg/**`
- `~/.aws/**`, `~/.azure/**`, `~/.kube/**`, `~/.docker/config.json`
- `~/.npmrc`, `~/.pypirc`, `~/.gem/credentials`
- `~/.git-credentials`, `~/.config/gh/**`
- Crypto wallets (metamask, electrum, exodus, phantom, solflare)

**System Integrity:**
- Shell config denied: `~/.bashrc`, `~/.zshrc` (prevents backdoor planting)
- macOS keychain: `~/Library/Keychains/**`

### Recommended Hooks
1. Block `rm -rf` — suggests `trash` command instead
2. Block direct push to main/master — requires feature branches
3. (Additional patterns for audit logging, notifications, package manager enforcement)

### MCP Controls
- `enableAllProjectMcpServers: false` — blocks malicious servers in untrusted repos
