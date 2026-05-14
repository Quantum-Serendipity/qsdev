# Claude Code Sandboxing: Making Claude Code More Secure and Autonomous

- **Source URL**: https://www.anthropic.com/engineering/claude-code-sandboxing
- **Retrieved**: 2026-05-14

## Overview and Impact

Anthropic introduced sandboxing for Claude Code to enhance security while reducing friction. Internal testing demonstrated that "sandboxing safely reduces permission prompts by 84%," enabling more autonomous operation without sacrificing safety.

## Sandboxing Architecture

The system operates on a dual-boundary approach built on operating system-level features:

### Filesystem Isolation
Claude gains read and write capabilities exclusively within the current working directory. The implementation "ensures that Claude can only access or modify specific directories. This is particularly important in preventing a prompt-injected Claude from modifying sensitive system files."

### Network Isolation
Outbound connections route through a Unix domain socket connected to an external proxy server that enforces domain restrictions. The proxy validates requested connections and requires user confirmation for new domains. Users can customize proxy rules for additional traffic control.

## Technical Implementation

**Sandboxed Bash Tool**: The runtime leverages OS-level primitives including Linux bubblewrap and macOS seatbelt to enforce restrictions at the kernel level. This approach covers not just direct Claude interactions but also spawned subprocesses and scripts.

**Git Security**: Claude Code on the web uses a custom proxy handling all Git interactions. Rather than storing credentials in the sandbox, "the git client authenticates to this service with a custom-built scoped credential. The proxy verifies this credential and the contents of the git interaction...then attaches the right authentication token."

## Permission Model

By default, Claude Code operates read-only, requesting approval before modifications or command execution. Sandboxing enables autonomous operation within defined boundaries, dramatically reducing approval fatigue while maintaining defensive posture against prompt injection attacks.
