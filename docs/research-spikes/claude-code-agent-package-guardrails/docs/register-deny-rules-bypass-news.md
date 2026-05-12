<!-- Source: https://www.theregister.com/2026/04/01/claude_code_rule_cap_raises/ -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Bypasses Safety Rule If Given Too Many Commands (The Register)

## The Core Issue

Claude Code's deny rules can be circumvented by chaining together a sufficiently long sequence of subcommands. Security firm Adversa discovered that "a hard cap of 50 on security subcommands" exists in the code, after which enforcement switches from automatic denial to user permission requests.

## Technical Mechanism

The vulnerability stems from a hardcoded limit in `bashPermissions.ts`. The variable `MAX_SUBCOMMANDS_FOR_SECURITY_CHECK = 50` was intended as "a generous allowance for legitimate usage." However, this assumption failed to account for AI-generated command chains.

Adversa's proof-of-concept: 50 harmless "true" (no-op) subcommands followed by a denied curl command. Instead of blocking curl outright, Claude requested user authorization.

## Attack Vector

Particularly dangerous through prompt injection. A malicious `CLAUDE.md` file can instruct the AI to generate a 50+ subcommand pipeline "that looks like a legitimate build process." Dangerous in:

- `--dangerously-skip-permissions` mode
- CI/CD pipelines running Claude Code non-interactively
- Long development sessions where users reflexively approve prompts

## Fix

Patched in Claude Code v2.1.90. Anthropic had already developed a "tree-sitter" parser that could properly analyze command structures. A one-line code change switching the behavior setting from "ask" to "deny" resolved the issue.
