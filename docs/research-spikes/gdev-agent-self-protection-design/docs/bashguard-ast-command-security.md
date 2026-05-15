<!-- Source: https://github.com/sunir/bashguard -->
<!-- Retrieved: 2026-05-15 -->

# bashguard: AST-Based Bash Command Security Interceptor

## Core Architecture

bashguard operates as a pipeline that intercepts bash commands before execution:

**"bash string -> parse (tree-sitter AST) -> audit (security rules -> Findings) -> decide (Findings -> Verdict)"**

The system makes three possible verdicts: ALLOW (executes in sandbox), BLOCK (denied), or CONFIRM (requests user approval).

## Command Analysis

Uses tree-sitter to parse bash into an Abstract Syntax Tree. The system can identify write targets through redirection operators (>, >>, tee, dd), though the detailed extraction logic requires examining the source code.

## Security Rules System

bashguard enforces rules across multiple categories:

- **Destructive operations**: `rm -rf`, `dd`, `mkfs`, `shred`
- **Credential access**: reads from `~/.ssh`, `~/.aws`, `.env`
- **Network operations**: curl/wget to unlisted hosts
- **Git safety**: force push, hard reset, branch deletion
- **Protected paths**: writes to `/etc`, `/usr`, `/sys`, `/boot`
- **Evasion techniques**: 13 rules covering `eval`, shell nesting, base64 pipelines
- **Self-protection**: blocks bashguard modification attempts
- **High-risk activities**: communications, SQL destruction, crypto mining, tunneling

## Verdict Model

On ALLOW, the command runs inside sandbox-exec (macOS kernel sandbox with deny-default access). On BLOCK, Claude Code sees a deny and never executes the command. Supports per-project policy configuration through `.bashguard.yaml` with ratcheting constraints.

## Relevance to gdev

bashguard demonstrates that AST-based bash command analysis is feasible and practical for security enforcement. The tree-sitter approach is fundamentally more accurate than regex-based analysis because it respects shell quoting rules, nesting depth, and operator precedence. However, it requires a compiled tree-sitter-bash parser, adding build complexity compared to regex matching in bash scripts.
