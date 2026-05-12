# Agent Sandboxes: Technical Deep Dive

- **Source**: https://pierce.dev/notes/a-deep-dive-on-agent-sandboxes
- **Retrieved**: 2026-05-12

## Core Sandboxing Approaches

Pierce Freeman's article examines how Codex implements sandboxing across platforms. Freeman notes that "the safest way to run any coding agent is within virtualization," yet acknowledges most users skip security measures entirely.

## Platform-Specific Implementations

**macOS (Seatbelt):**
- Uses Apple's native Seatbelt framework via `/usr/bin/sandbox-exec`
- Designates writable roots while protecting `.git` directories as read-only
- Implements all-or-nothing network access (no granular domain restrictions)

**Linux (Landlock + seccomp):**
- Applies filesystem restrictions through Landlock capability-based controls
- Blocks network syscalls (`connect`, `bind`, `listen`, etc.) via seccomp-BPF
- Spawns sandboxed commands as separate processes before `execvp`

## Key Design Patterns

The implementation emphasizes three principles:

1. **Default-sandbox execution** — commands run restricted unless explicitly escalated
2. **Session-based trust lists** — users approve commands once per session, reducing repetitive approvals
3. **Selective escape hatches** — failed sandboxed commands can retry unsandboxed with explicit approval

## Practical Limitations

Both OS-level sandboxes struggle with package managers. Freeman quotes: "if you use a native Mac sandbox then it'd need to ask permission to use homebrew," highlighting why dependency installation remains problematic within traditional OS sandboxes.

Network isolation particularly lacks sophistication — you cannot restrict access to specific domains or protocols, only allow/deny entirely.

## Real-World Relevance

Freeman emphasizes this matters increasingly as agents gain code execution capabilities, positioning proper isolation as essential rather than optional infrastructure.
