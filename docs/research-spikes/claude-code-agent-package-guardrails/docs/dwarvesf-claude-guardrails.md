# dwarvesf/claude-guardrails

- **Source URL**: https://github.com/dwarvesf/claude-guardrails
- **Retrieved**: 2026-05-12
- **Note**: Hardened security configuration for Claude Code from Dwarves Foundation.

---

## Overview

Two variants: **Lite** (4 hooks, 21 deny rules) and **Full** (6 hooks + prompt injection scanner, 40 deny rules).

## Installation

```bash
npx claude-guardrails install          # Lite variant
npx claude-guardrails install full     # Full variant
```

Installer merges configurations into `~/.claude/settings.json` with automatic backup.

## Security Layers

### 1. Permission Deny Rules
- **Lite:** 21 rules blocking SSH keys, AWS credentials, GPG configs, kubeconfig, Azure, `.env`, `.pem` files, and destructive bash commands
- **Full:** 40 rules adding secrets directories, shell profiles, crypto wallets, and additional sensitive paths

### 2. PreToolUse Hooks
- Destructive delete detection (both variants)
- Direct git push prevention (both variants)
- Pipe-to-shell blocking (both variants)
- Commit-time secret scanning via `scan-commit.sh` (both variants)
- Data exfiltration patterns (full only)
- Permission escalation detection (full only)

### 3. UserPromptSubmit Secret Scanner
`scan-secrets.sh` blocks prompts containing live credentials: AWS keys, GitHub/Anthropic/OpenAI tokens, PEM blocks, BIP39 phrases.

### 4. PostToolUse Prompt Injection Scanner (Full Only)
`prompt-injection-defender.sh` scans Read/WebFetch/Bash outputs for injection patterns like "ignore previous instructions".

### 5. CLAUDE.md Security Rules
Natural-language instructions guiding Claude not to hardcode secrets.

## Critical Limitation

"Deny rules only cover Claude's built-in tools, not bash. `Read ~/.ssh/id_rsa` is denied, but `bash cat ~/.ssh/id_rsa` is not." Actual enforcement boundary is OS-level sandboxing via `/sandbox` command (Seatbelt on macOS, bubblewrap on Linux).

## Key Tradeoffs

- False positives interrupt workflow
- Hook latency: 4 extra processes per bash call (lite), 7 (full)
- No per-file exceptions / allowlist mechanism
- Noisy injection scanner pattern-matches legitimate security documentation
