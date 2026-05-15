<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: WebFetch returned a summary rather than raw content. See prempti-claude-md.md for full architecture details. -->

# Prempti: Policy and Visibility for AI Coding Agents

## Overview

Prempti integrates Falco security rules with AI coding agents to provide guardrails and audit trails. It "gives you guardrails that can deny or ask for confirmation on unwanted behaviors, plus real-time visibility into every tool call your coding agent makes."

## Key Features

The system operates through two modes:

1. **Guardrails mode** (default): Rules enforce verdicts -- deny blocks actions, ask prompts for confirmation, allow permits execution.
2. **Monitor mode**: Tool calls proceed while rules still evaluate and log activity for observation-only scenarios.

The project provides "real-time tool-call interception" for shell commands, file operations, web requests, and MCP calls evaluated before execution.

## Architecture

The system uses a three-layer design: the coding agent's hook triggers the interceptor, which sends events to Falco via Unix socket. Falco's rule engine processes events and returns verdicts (allow/deny/ask).

## Supported Platforms

Prempti works on Linux (x86_64, aarch64), macOS (Apple Silicon, Intel), and Windows (x86_64, ARM64). Currently, it supports Claude Code with Codex integration planned.

## Important Limitation

The documentation emphasizes that "It is not a sandbox, OS-level security, or a substitute for least-privilege environments or system hardening." Hook-level interception sees declared commands but not their runtime side effects -- a critical distinction for security posture.

## Customization

Users can author custom Falco rules in YAML or use an included Claude Code skill for interactive rule generation, enabling workflow-specific policies beyond the default ruleset.
