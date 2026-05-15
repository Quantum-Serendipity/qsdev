<!-- Source: https://adversa.ai/blog/claude-code-security-bypass-deny-rules-disabled/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content returned via WebFetch AI summary -->

# Claude Code Deny Rule Bypass Vulnerability: Technical Analysis

## Core Vulnerability

Claude Code silently disables security deny rules when commands exceed 50 subcommands. A developer with a configured rule blocking `curl` will see it enforced for isolated commands but bypassed when `curl` appears after 50 other commands chained with `&&`, `||`, or `;`.

## Technical Mechanism

**Location:** bashPermissions.ts, lines 2162-2178

**How it works:** The legacy regex parser in Claude Code implements an analysis cap. When a compound command splits into more than 50 subcommands, the system abandons per-subcommand security validation entirely and falls back to a generic permission prompt.

The problematic code comment states: "Fifty is generous: legitimate user commands don't split that wide. Above the cap we fall back to 'ask' (safe default, we can't prove safety, so we prompt)."

This assumption fails for AI-generated commands, where malicious instructions can create realistic-looking build pipelines with harmless commands padded to position 51.

## The Paradox

Anthropic's codebase contains a newer tree-sitter parser that handles this correctly -- checking deny rules before applying complexity caps. This secure implementation exists in the repository but wasn't deployed to customer-facing builds. The secure pattern was written and tested but never shipped.

## Attack Path

An attacker creates a seemingly legitimate open-source project with a poisoned `CLAUDE.md` containing 50+ legitimate build steps. Hidden at position 51: a credential-exfiltration command. When developers ask Claude Code to build the project, the deny rules never fire, and SSH keys, AWS credentials, and API tokens are silently harvested.

## Security Impact

- **Silent failure:** No warnings or audit trails when deny rules are bypassed
- **False confidence:** Security-conscious developers who configured rules are given unwarranted protection assurance
- **Supply chain risk:** Stolen tokens enable publishing malicious package versions to downstream consumers
- **Infrastructure access:** Compromised cloud credentials provide direct production access

## Structural Implications

This vulnerability reveals a fundamental conflict in agentic AI: security validation consumes tokens that compete with user functionality for the same resource budget. As token economics tighten, the incentive to skip security checks will intensify industry-wide.
