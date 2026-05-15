<!-- Source: https://codepointer.substack.com/p/agentfs-how-to-stop-ai-agents-from -->
<!-- Retrieved: 2026-05-15 -->

# AgentFS: Kernel-Enforced File Isolation for AI Agents

## Core Approach

AgentFS abandons traditional Unix permission enforcement (chmod/inode-level checks) for kernel-enforced mount table isolation. The system prevents file tampering by controlling which filesystems are visible and writable, rather than decorating files with permission metadata.

## Linux Implementation

### Namespace Isolation Strategy

The sandbox forks a child process and immediately isolates it using `unshare()`. Two key namespaces enable the design:

- **CLONE_NEWUSER**: Creates a user namespace where the child gains `CAP_SYS_ADMIN` without host-level root privileges, allowing mount manipulation
- **CLONE_NEWNS**: Provides a private mount table, giving each sandboxed process its own filesystem view

### Copy-on-Write Overlay with FUSE

Before sandbox startup, a FUSE server implements a copy-on-write overlay backed by SQLite:

- Reads pass through to original files on disk via a pre-opened file descriptor
- Writes are intercepted and stored in SQLite, never touching the actual filesystem
- The overlay bind-mounts onto the working directory, redirecting all file operations through the FUSE server

### Read-Only Enforcement

The system remounts all other mounts as read-only using `MS_BIND | MS_REMOUNT | MS_RDONLY`. This is the critical security boundary -- not individual file permissions, but mount-table restrictions enforced by the kernel's VFS layer.

Key enforcement mechanism: An agent attempting `echo "pwned" >> /etc/passwd` receives `EROFS` (Read-only file system) error from the kernel, not `EACCES` (Permission denied).

### Circular Reference Prevention

The system opens a file descriptor to the working directory before mounting the FUSE overlay. The `HostFS` base layer then accesses original directory contents through `/proc/self/fd/N`, which resolves to the underlying filesystem rather than the FUSE mount sitting on top. This bypasses the overlay when the server itself needs to read original data.

## POSIX Metadata Handling

AgentFS stores complete POSIX metadata (uid, gid, mode bits) in SQLite but deliberately skips enforcement checks. Security lives in mount-table restrictions, not userspace permission checks.

## Relevance to gdev

AgentFS demonstrates that kernel-level enforcement is fundamentally stronger than path-based hook checks. However, it requires a full sandbox runtime (FUSE + namespaces), which is heavier than gdev's hook-based approach. The key lesson: path-based security is a speed bump that catches common cases, while kernel-level enforcement is a wall.
