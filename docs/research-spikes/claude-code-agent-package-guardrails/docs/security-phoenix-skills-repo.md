---
source: https://github.com/Security-Phoenix-demo/security-skills-claude-code
retrieved: 2026-05-12
---

# Security Skills for Claude Code (Phoenix Security)

Open-source MIT-licensed toolkit by Phoenix Security that extends Claude Code with security-focused capabilities.

## Core Skills

- CTI Domain Research (595+ curated security domains, four authority tiers)
- NotebookLM Connector
- Global Research Pipeline
- Secure PRD Generator (STRIDE threat models, RFC 2119 compliance)
- OpenGrep Rule Generator (SAST rules for 30+ languages)
- Security Assessment Suite (4 AppSec skills with active hooks)

## Package Installation Security (Pre-merge Gating)

The Security Assessment Suite includes a PreToolUse hook that gates package installation commands:

Protected commands: npm/yarn/pnpm/pip/uv/poetry/cargo/go get/gem/bundle/composer/dotnet add

Hook behavior:
- Blocks known-malicious packages
- Flags typosquats and brand-new packages with user confirmation
- Zero configuration required post-install

## Directory Structure

```
security-skills-claude-code/
├── skills/
│   ├── cti-search-skill/
│   ├── secure-prd-skill/
│   ├── opengrep-rule-generator/
│   ├── opengrep-rule-generator-research/
│   ├── notebooklm/
│   ├── global-research-notebook-lm/
│   └── project Documentation skill/
├── plugins/
│   ├── cti-search-plugin/ (MCP server + CLI)
│   └── secure-prd/
└── feature-descriptor/ (12 role skill files)
```

## Active Hooks (Security Assessment Suite)

- SessionStart: fingerprints project, runs dependency audits
- PreToolUse: gates package installation
- PostToolUse: pattern scans on file writes (SQL injection, innerHTML, hardcoded secrets)
- SessionEnd: reminder to run /security-0day before PR

## Installation

```bash
bash "skills/Security Assessment/install/install.sh" --full
```

Options: --lite (hooks off), --full (all hooks + subagent), --dry-run, --uninstall
