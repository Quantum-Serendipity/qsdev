# Codex Sandboxing Implementation

- **Source**: https://deepwiki.com/openai/codex/5.6-sandboxing-implementation
- **Retrieved**: 2026-05-12

## Architecture Overview

Codex translates high-level `SandboxPolicy` requirements into platform-specific OS primitives. The system selects appropriate sandbox types based on requested policies and the host operating system.

## Platform-Specific Implementations

### macOS: Seatbelt

On macOS, Codex uses the system's `sandbox-exec` utility. The implementation dynamically generates Sandbox Profile Language (SBPL) scripts based on requested permissions. Key features include:

- Construction of SBPL arguments from the `SandboxPolicy`
- Protection of `.git`, `.agents`, and `.codex` directories as read-only, "even in `WorkspaceWrite` mode"
- Network access control via the resolved policy

### Linux: Bubblewrap and Landlock

The Linux approach uses **Bubblewrap** (the default) with **Landlock** as supplementary restriction or fallback.

**Bubblewrap Features:**
- "Explicitly isolates user, PID, and network namespaces via `--unshare-user`, `--unshare-pid`, and `--unshare-net`"
- Read-only-by-default filesystem with layered writable roots using `--bind`
- Re-applies read-only mounts for sensitive subpaths within writable roots
- Supports three network modes: `Isolated`, `ProxyOnly`, and `FullAccess`
- Fallback to bundled version if system `bwrap` is unavailable

**Landlock and Seccomp:**
- Applies `PR_SET_NO_NEW_PRIVS` and seccomp network filtering in-process
- Restricts syscalls including `ptrace`, `io_uring_*`, and network operations based on `NetworkSeccompMode`
- Landlock serves as legacy fallback, restricting write access while permitting system-wide read access

### Windows: Restricted Tokens and ACLs

Windows sandboxing employs access control mechanisms:

- **Preflight Audit**: Scans for world-writable directories that might circumvent restrictions
- **ACL Management**: Functions manipulate directory permissions for sandbox identities
- **Token-Based Execution**: Commands launch using `create_process_as_user` with restricted tokens

## Denial Detection and Retry Logic

The system detects sandbox-specific failures through:
- Specific exit codes and error signals (like `LandlockRestrict`)
- Stderr pattern matching (e.g., mount failures)
- Automated escalation prompting when sandbox denials occur
