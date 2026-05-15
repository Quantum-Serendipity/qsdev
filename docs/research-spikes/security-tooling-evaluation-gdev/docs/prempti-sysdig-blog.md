<!-- Source: https://webflow.sysdig.com/blog/introducing-prempti-runtime-security-for-ai-coding-agents-powered-by-falco -->
<!-- Retrieved: 2026-05-15 -->

# Introducing Prempti: Runtime Security for AI Coding Agents (Sysdig Blog)

## Overview

Sysdig has announced Prempti, an open source project integrating Falco's runtime security detection engine directly into AI coding agent workflows. The tool addresses visibility gaps when AI agents like Claude Code operate within developer environments with user permissions.

## The Security Challenge

AI coding agents present a novel risk: they operate within user sessions with access to credentials, SSH keys, and cloud configuration files. According to the article, documented cases show agents have "read files well outside the project scope, exfiltrated environment variables, or attempted to make network calls to external hosts." Most developers lack structured visibility into agent activity beyond chat outputs, with no policy enforcement or audit trails.

## How Prempti Works

Prempti intercepts agent tool calls before execution and evaluates them against Falco rules, returning one of three verdicts:

- **Allow**: Action proceeds
- **Deny**: Action blocked with explanation sent to agent
- **Ask**: Interactive approval prompt

The system runs as a lightweight user-space service requiring no root, kernel modules, or containers.

## Default Protections

The default ruleset covers:
- Working-directory boundaries
- Sensitive path protection
- Credential access prevention
- Destructive command blocking
- Pipe-to-shell attack prevention
- Exfiltration attempt detection
- MCP server config poisoning prevention
- Persistence vectors (hook/git injection)

## Operating Modes

**Guardrails mode** actively shapes agent behavior by blocking or flagging tool calls. **Monitor mode** provides visibility without enforcement -- suitable for organizations taking conservative approaches to new tooling.

## Customization

Rules use plain YAML with Falco syntax. A Claude Code skill assists with drafting and validating custom rules interactively within the agent interface.
